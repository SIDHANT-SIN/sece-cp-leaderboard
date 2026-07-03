//go:build integration
// +build integration

//
package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"leaderboard/src/database"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		codeforces_handle TEXT UNIQUE NOT NULL,
		display_name TEXT
	);
	CREATE TABLE IF NOT EXISTS past_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		codeforces_handle TEXT UNIQUE NOT NULL,
		display_name TEXT,
		batch_year INTEGER NOT NULL,
		current_rating INTEGER DEFAULT 0,
		max_rating INTEGER DEFAULT 0,
		title TEXT DEFAULT '',
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS contests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		codeforces_contest_id INTEGER UNIQUE NOT NULL,
		name TEXT,
		start_time INTEGER
	);
	CREATE TABLE IF NOT EXISTS user_contest_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		contest_id INTEGER NOT NULL,
		rank INTEGER,
		points INTEGER,
		last_updated INTEGER,
		UNIQUE(user_id, contest_id)
	);
	CREATE TABLE IF NOT EXISTS sync_history (
		job_id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		successful_contests INTEGER DEFAULT 0,
		total_contests INTEGER DEFAULT 0,
		failed_contest_ids TEXT DEFAULT '',
		started_at TEXT NOT NULL,
		completed_at TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	database.DB = db
	return db
}

func TestDetectDivision(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Div 2 Contest", "Codeforces Round 800 (Div. 2)", "Div. 2"},
		{"Div 3 Contest", "Codeforces Round 750 (Div. 3)", "Div. 3"},
		{"Div 4 Contest", "Codeforces Round 600 (Div. 4)", "Div. 4"},
		{"Div 1 Contest", "Codeforces Global Round 10", "Div. 1"},
		{"Empty Name", "", "Div. 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := detectDivision(tc.input)
			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestCalculatePoints(t *testing.T) {
	tests := []struct {
		name     string
		rank     int
		total    int
		div      string
		expected int
	}{
		{"Top Rank Div 1", 1, 1000, "Div. 1", 28},
		{"Mid Rank Div 2", 500, 1000, "Div. 2", 5},
		{"Low Rank Div 3", 900, 1000, "Div. 3", 2},
		{"Zero Total", 10, 0, "Div. 2", 0},
		{"Zero Rank", 0, 1000, "Div. 2", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculatePoints(tc.rank, tc.total, tc.div)
			if result != tc.expected {
				t.Errorf("expected %d points, got %d", tc.expected, result)
			}
		})
	}
}

func TestRateLimiters(t *testing.T) {
	t.Run("waitForCFRateLimit", func(t *testing.T) {
		start := time.Now()
		fakeStart := start.Add(-1 * time.Second)
		waitForCFRateLimit(fakeStart)

		elapsed := time.Since(fakeStart)
		if elapsed < cfRateLimit {
			t.Errorf("Rate limiter failed, elapsed time %v is less than required %v", elapsed, cfRateLimit)
		}
	})

	t.Run("waitForCFRateLimit2", func(t *testing.T) {
		start := time.Now()
		waitForCFRateLimit2(start)

		elapsed := time.Since(start)
		if elapsed < cfRateLimit2 {
			t.Errorf("Rate limiter failed, elapsed time %v is less than required %v", elapsed, cfRateLimit2)
		}
	})
}

func TestHandlers_InvalidPayloads(t *testing.T) {
	ctx := context.Background()
	invalidPayload := []byte(`{invalid_json`)

	tests := []struct {
		name    string
		handler func(context.Context, *asynq.Task) error
	}{
		{"HandleCFRatingChanges", HandleCFRatingChanges},
		{"HandleCFBatchRefresh", HandleCFBatchRefresh},
		{"HandleCFRefreshRating", HandleCFRefreshRating},
		{"HandleCFAddContest", HandleCFAddContest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := asynq.NewTask("dummy_type", invalidPayload)
			err := tc.handler(ctx, task)
			if err == nil {
				t.Errorf("Expected JSON unmarshal error for %s, got nil", tc.name)
			}
		})
	}
}

func TestHandleCFRefreshRating_HTTPParsing(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO past_users (codeforces_handle, batch_year) VALUES ('tourist', 2023), ('some_guy', 2023)`)
	if err != nil {
		t.Fatalf("failed to insert dummy users: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status": "OK",
			"result": []map[string]interface{}{
				{"handle": "tourist", "rating": 3900, "maxRating": 4000, "rank": "legendary grandmaster"},
				{"handle": "some_guy", "rating": 1500, "maxRating": 1600, "rank": "specialist"},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	payload, _ := json.Marshal(CFRefreshRatingPayload{
		JobID: "", // Empty JobID bypasses Redis state tracking
	})
	task := asynq.NewTask("refresh_rating", payload)

	err = HandleCFRefreshRating(context.Background(), task)
	if err != nil {
		t.Errorf("Expected HandleCFRefreshRating to succeed, got error: %v", err)
	}

	var rating int
	err = db.QueryRow("SELECT current_rating FROM past_users WHERE codeforces_handle = 'tourist'").Scan(&rating)
	if err != nil {
		t.Fatalf("Failed to query updated user: %v", err)
	}
	if rating != 3900 {
		t.Errorf("Expected rating to be updated to 3900, got %d", rating)
	}
}

func TestHandleCFAddContest_HTTPParsing(t *testing.T) {

	db := setupTestDB(t)
	defer db.Close()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status": "OK",
			"result": map[string]interface{}{
				"contest": map[string]interface{}{
					"id":               9999,
					"name":             "Test Contest API Parsing",
					"startTimeSeconds": 1700000000,
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	payload, _ := json.Marshal(CFAddContestPayload{
		CFContestID: "9999",
		JobID:       "",
	})
	task := asynq.NewTask("add_contest", payload)

	err := HandleCFAddContest(context.Background(), task)
	if err != nil {
		t.Errorf("Expected HandleCFAddContest to succeed, got error: %v", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM contests WHERE codeforces_contest_id = 9999").Scan(&name)
	if err != nil {
		t.Fatalf("Failed to query inserted contest: %v", err)
	}
	if name != "Test Contest API Parsing" {
		t.Errorf("Expected contest name 'Test Contest API Parsing', got '%s'", name)
	}
}

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	database.RedisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr
}

func TestUpdateJobError(t *testing.T) {
	mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	jobID := "test_error_job"

	// Create dummy state
	_ = SetJobState(ctx, database.RedisClient, jobID, &JobState{JobID: jobID, Status: "running"}, 10*time.Minute)

	updateJobError(ctx, jobID, "Something went terribly wrong")

	state, _ := GetJobState(ctx, database.RedisClient, jobID)
	if state.Status != "failed" || state.Error != "Something went terribly wrong" {
		t.Errorf("updateJobError failed to set correct state. Got: %+v", state)
	}
}

func TestProcessSingleContestStandings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create user
	db.Exec(`INSERT INTO users (id, codeforces_handle, display_name) VALUES (1, 'tourist', 'Tourist')`)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "OK",
			"result": []map[string]interface{}{
				{"contestId": 1234, "contestName": "Codeforces Round 800 (Div. 2)", "handle": "tourist", "rank": 1},
			},
		})
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	users := []userEntry{{ID: 1, Handle: "tourist"}}
	err := processSingleContestStandings(1234, 1, users)

	if err != nil {
		t.Errorf("Expected success, got: %v", err)
	}

	var points int
	err = db.QueryRow("SELECT points FROM user_contest_results WHERE user_id = 1 AND contest_id = 1").Scan(&points)
	if err != nil {
		t.Errorf("Failed to retrieve upserted result: %v", err)
	}

	if points == 0 {
		t.Errorf("Expected non-zero points, got %d", points)
	}
}

func TestHandleCFRatingChanges_FullCoverage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	db.Exec(`INSERT INTO users (codeforces_handle) VALUES ('tourist')`)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK", "result": []}`)) // Empty result is fine for coverage
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	jobID := "job_rating_changes"
	_ = SetJobState(context.Background(), database.RedisClient, jobID, &JobState{JobID: jobID}, 10*time.Minute)

	payload, _ := json.Marshal(CFRatingChangesPayload{CFContestID: 1000, ContestDBID: 1, JobID: jobID})
	task := asynq.NewTask("rating_changes", payload)

	err := HandleCFRatingChanges(context.Background(), task)
	if err != nil {
		t.Errorf("HandleCFRatingChanges failed: %v", err)
	}
}

func TestHandleCFBatchRefresh_FullCoverage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	// Seed data
	db.Exec(`INSERT INTO users (codeforces_handle) VALUES ('tourist')`)
	db.Exec(`INSERT INTO contests (id, codeforces_contest_id, name) VALUES (1, 999, 'Test Contest')`)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK", "result": []}`))
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	jobID := "job_batch"
	ctx := context.Background()
	_ = SetJobState(ctx, database.RedisClient, jobID, &JobState{JobID: jobID}, 10*time.Minute)

	payload, _ := json.Marshal(CFBatchRefreshPayload{JobID: jobID})
	task := asynq.NewTask("batch", payload)

	err := HandleCFBatchRefresh(ctx, task)
	if err != nil {
		t.Errorf("HandleCFBatchRefresh failed: %v", err)
	}
}

func TestHandleCFRefreshRating_CronCoverage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	db.Exec(`INSERT INTO past_users (codeforces_handle, batch_year) VALUES ('tourist', 2023)`)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "OK", "result": [{"handle": "tourist", "rating": 3000}]}`))
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	jobID := "cron_refresh_rating"
	_ = SetJobState(context.Background(), database.RedisClient, jobID, &JobState{JobID: jobID}, 10*time.Minute)

	payload, _ := json.Marshal(CFRefreshRatingPayload{JobID: jobID})
	task := asynq.NewTask("refresh", payload)

	err := HandleCFRefreshRating(context.Background(), task)
	if err != nil {
		t.Errorf("HandleCFRefreshRating failed: %v", err)
	}
}

func TestHandleCFBatchRefresh_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	db.Exec(`INSERT INTO contests (id, codeforces_contest_id, name, start_time) VALUES (1, 999, 'Test Contest', 1700000000)`)

	ctx, cancel := context.WithCancel(context.Background())

	payload, _ := json.Marshal(CFBatchRefreshPayload{JobID: "job_batch_cancel"})
	task := asynq.NewTask("batch", payload)

	cancel()

	err := HandleCFBatchRefresh(ctx, task)

	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
}

func TestUpdateJobError_Coverage(t *testing.T) {
	mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	jobID := "fail_job"
	state := &JobState{JobID: jobID, Status: "running"}
	_ = SetJobState(ctx, database.RedisClient, jobID, state, 1*time.Minute)

	updateJobError(ctx, jobID, "fatal_error")

	newState, _ := GetJobState(ctx, database.RedisClient, jobID)
	if newState.Status != "failed" || newState.Error != "fatal_error" {
		t.Errorf("updateJobError failed to update state, got: %s", newState.Status)
	}
}

func TestCalculatePoints_Default(t *testing.T) {

	pts := calculatePoints(10, 100, "Div. Unknown")
	if pts == 0 {
		t.Errorf("Expected points > 0 for default division calculation, got %d", pts)
	}
}

func TestUpdateJobError_EdgeCases(t *testing.T) {

	updateJobError(context.Background(), "", "error message")

	mr := setupTestRedis(t)
	mr.Close() // Force Redis error
	updateJobError(context.Background(), "some_job", "error message")
}

func TestProcessSingleContest_Failures(t *testing.T) {
	users := []userEntry{{ID: 1, Handle: "tourist"}}

	// 1. HTTP Error
	originalURL := CFBaseURL
	CFBaseURL = "http://127.0.0.1:0" // Invalid URL to force dial error
	err := processSingleContestStandings(1, 1, users)
	if err == nil || !strings.Contains(err.Error(), "HTTP request failed") {
		t.Errorf("Expected HTTP request failure, got: %v", err)
	}

	// Setup mock server for subsequent tests
	var mockResp string
	var mockStatus int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResp))
	}))
	defer mockServer.Close()
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	// 2. Non-200 Status
	mockStatus = http.StatusNotFound
	mockResp = "Not Found"
	err = processSingleContestStandings(1, 1, users)
	if err == nil || !strings.Contains(err.Error(), "CF standings returned HTTP 404") {
		t.Errorf("Expected HTTP 404 error, got: %v", err)
	}

	// 3. Bad JSON
	mockStatus = http.StatusOK
	mockResp = "{bad_json]"
	err = processSingleContestStandings(1, 1, users)
	if err == nil || !strings.Contains(err.Error(), "JSON decode failed") {
		t.Errorf("Expected JSON decode error, got: %v", err)
	}

	// 4. API Status Not OK
	mockResp = `{"status": "FAILED"}`
	err = processSingleContestStandings(1, 1, users)
	if err == nil || !strings.Contains(err.Error(), "CF API status not OK") {
		t.Errorf("Expected API not OK error, got: %v", err)
	}
}
func TestHandleCFAddContest_Failures(t *testing.T) {
	ctx := context.Background()
	payload, _ := json.Marshal(CFAddContestPayload{JobID: "job_add", CFContestID: "999"})
	task := asynq.NewTask("add_contest", payload)

	db := setupTestDB(t)
	defer db.Close()

	mr := setupTestRedis(t)
	defer mr.Close()

	var mockResp string
	var mockStatus int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResp))
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	// 1. Non-200 Status
	mockStatus = http.StatusBadGateway
	err := HandleCFAddContest(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Errorf("Expected HTTP 502 error, got: %v", err)
	}

	// 2. Bad JSON
	mockStatus = http.StatusOK
	mockResp = `{"status": "FAILED"}`
	err = HandleCFAddContest(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "could not parse") {
		t.Errorf("Expected JSON/Status parse error, got: %v", err)
	}

	// 3. DB Failure
	db.Exec("DROP TABLE contests")
	mockResp = `{"status": "OK", "result": {"contest": {"id": 999, "name": "Test"}}}`

	err = HandleCFAddContest(ctx, task)
	if err == nil {
		t.Errorf("Expected DB insert error, got nil")
	}
}

func TestHandleCFRefreshRating_Failures(t *testing.T) {
	ctx := context.Background()
	payload, _ := json.Marshal(CFRefreshRatingPayload{JobID: "job_refresh"})
	task := asynq.NewTask("refresh_rating", payload)

	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	// 1. DB Failure (GetPastUserHandles fails)
	db.Exec("DROP TABLE past_users")
	err := HandleCFRefreshRating(ctx, task)
	if err == nil {
		t.Errorf("Expected DB load error, got nil")
	}

	// Reset DB & add user
	db = setupTestDB(t)
	defer db.Close()
	db.Exec(`INSERT INTO past_users (codeforces_handle, batch_year) VALUES ('tourist', 2023)`)

	var mockResp string
	var mockStatus int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mockStatus)
		w.Write([]byte(mockResp))
	}))
	defer mockServer.Close()

	originalURL := CFBaseURL
	CFBaseURL = mockServer.URL
	defer func() { CFBaseURL = originalURL }()

	// 2. HTTP Non-200
	mockStatus = http.StatusInternalServerError
	err = HandleCFRefreshRating(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("Expected HTTP 500 error, got: %v", err)
	}

	// 3. Bad JSON
	mockStatus = http.StatusOK
	mockResp = "{invalid_json"
	err = HandleCFRefreshRating(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Expected JSON error, got: %v", err)
	}

	// 4. API Status Not OK
	mockResp = `{"status": "FAILED"}`
	err = HandleCFRefreshRating(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "not OK") {
		t.Errorf("Expected Not OK error, got: %v", err)
	}
}

func TestHandleCFBatchRefresh_AdminCancelSignal(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()
	ctx := context.Background()

	db.Exec(`INSERT INTO contests (id, codeforces_contest_id, name, start_time) VALUES (1, 999, 'Test Contest', 1700000000)`)

	// Set the manual admin cancellation flag in Redis
	database.RedisClient.Set(ctx, "sync:cancel_signal", "1", 10*time.Minute)

	payload, _ := json.Marshal(CFBatchRefreshPayload{JobID: "job_batch_cancel"})
	task := asynq.NewTask("batch", payload)

	err := HandleCFBatchRefresh(ctx, task)

	if err == nil || !strings.Contains(err.Error(), "cancelled by admin request") {
		t.Errorf("Expected admin cancel error, got: %v", err)
	}
}

func TestHandleCFBatchRefresh_DBFailures(t *testing.T) {
	ctx := context.Background()
	payload, _ := json.Marshal(CFBatchRefreshPayload{JobID: "job_batch"})
	task := asynq.NewTask("batch", payload)

	db := setupTestDB(t)
	mr := setupTestRedis(t)
	db.Exec("DROP TABLE users")
	err := HandleCFBatchRefresh(ctx, task)

	if err == nil || !strings.Contains(err.Error(), "no such table: users") {
		t.Errorf("Expected GetUsers error, got: %v", err)
	}
	db.Close()
	mr.Close()

	db2 := setupTestDB(t)
	mr2 := setupTestRedis(t)
	db2.Exec("DROP TABLE contests")
	err = HandleCFBatchRefresh(ctx, task)

	if err == nil || !strings.Contains(err.Error(), "no such table: contests") {
		t.Errorf("Expected GetContests error, got: %v", err)
	}
	db2.Close()
	mr2.Close()
}

func TestHandleCFRatingChanges_DBFailures(t *testing.T) {
	ctx := context.Background()
	payload, _ := json.Marshal(CFRatingChangesPayload{JobID: "job_rating", CFContestID: 999, ContestDBID: 1})
	task := asynq.NewTask("rating", payload)

	db := setupTestDB(t)
	defer db.Close()
	mr := setupTestRedis(t)
	defer mr.Close()

	db.Exec("DROP TABLE users")

	err := HandleCFRatingChanges(ctx, task)
	if err == nil || !strings.Contains(err.Error(), "no such table") {
		t.Errorf("Expected DB error for missing users table, got: %v", err)
	}
}
