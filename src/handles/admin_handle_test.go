//go:build integration
// +build integration

package handles

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"leaderboard/src/configs"
	"leaderboard/src/database"
	"leaderboard/src/workers"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupAdminTestRouter() *gin.Engine {
	
	database.RedisClient = redis.NewClient(&redis.Options{})

	gin.SetMode(gin.TestMode)
	r := gin.Default()

	tmpl := template.Must(template.New("").Parse(""))
	template.Must(tmpl.New("admin_contests.tmpl").Parse("Admin Contests"))
	template.Must(tmpl.New("admin.tmpl").Parse("Admin Dashboard"))
	r.SetHTMLTemplate(tmpl)

	return r
}
func TestShowContests_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		ContestsRows: &MockRows{
			Data: [][]any{{1, 1001, "Codeforces Round 1", 1600000000}},
		},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/contests", handler.ShowContests)

	req, _ := http.NewRequest(http.MethodGet, "/admin/contests", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Admin Contests")
}

func TestShowContests_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/contests", handler.ShowContests)

	req, _ := http.NewRequest(http.MethodGet, "/admin/contests", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestAddContest_EmptyCFID(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/add", handler.AddContest)

	formData := url.Values{"cfid": {"  "}} // empty/spaces
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddContest_SyncInProgress(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncStatus: map[string]interface{}{"status": "processing", "job_id": "job123"},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/add", handler.AddContest)

	formData := url.Values{"cfid": {"1001"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteContest_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/delete", handler.DeleteContest)

	formData := url.Values{"id": {"1"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/delete", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin/contests", w.Header().Get("Location"))
}

func TestRefreshResults_InvalidLimit(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/refresh", handler.RefreshResults)

	formData := url.Values{"limit": {"-5"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/refresh", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestRefreshResults_ActiveSync(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncStatus: map[string]interface{}{"status": "processing", "job_id": "job123"},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/refresh", handler.RefreshResults)

	formData := url.Values{"limit": {"5"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/refresh", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "another sync job")
}

func TestShowAdminDashboard_Success(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncHistoryRows: &MockRows{
			Data: [][]any{
				{"a_12345678901", "completed", 5, 5, "", "2023-01-01 10:00:00", "2023-01-01 10:05:00"},
				{"c_123", "cancelled", 0, 10, "", "2023-01-02 12:00:00", "2023-01-02 12:01:00"},
			},
		},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/dashboard", handler.ShowAdminDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSyncStatus_RepoError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncStatusErr: errors.New("db error"),
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/sync-status", handler.GetSyncStatus)

	req, _ := http.NewRequest(http.MethodGet, "/admin/sync-status", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSyncStatus_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/sync-status", handler.GetSyncStatus)

	req, _ := http.NewRequest(http.MethodGet, "/admin/sync-status", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCancelSync_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/cancel-sync", handler.CancelSync)

	req, _ := http.NewRequest(http.MethodPost, "/admin/cancel-sync", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cancel_signal_sent")
}

func TestShowContests_DBError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		ContestsErr: errors.New("db error"),
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/contests", handler.ShowContests)

	req, _ := http.NewRequest(http.MethodGet, "/admin/contests", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShowContests_ScanError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		ContestsRows: &MockRows{
			Data: [][]any{{1, 1001, "Codeforces Round 1", 1600000000}},
			Err:  errors.New("scan error"),
		},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/contests", handler.ShowContests)

	req, _ := http.NewRequest(http.MethodGet, "/admin/contests", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddContest_AsynqClientNil(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	handler := NewAdminHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/add", handler.AddContest)

	formData := url.Values{"cfid": {"1001"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Asynq client")
}

func TestDeleteContest_ResultsError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		DeleteResultsByContestErr: errors.New("could not delete results"),
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/delete", handler.DeleteContest)

	formData := url.Values{"id": {"1"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/delete", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteContest_ContestError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		DeleteContestErr: errors.New("could not delete contest"),
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/delete", handler.DeleteContest)

	formData := url.Values{"id": {"1"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/delete", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshResults_SuccessPath(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		ContestsRows: &MockRows{
			Data: [][]any{{1, 1001, "Codeforces Round 1", 1600000000}},
		},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/refresh", handler.RefreshResults)

	formData := url.Values{"limit": {"1"}} // Limit > 0 branch
	req, _ := http.NewRequest(http.MethodPost, "/admin/refresh", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshResults_ExceedsLimit(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		ContestsRows: &MockRows{
			Data: [][]any{{1, 1001, "Contest A", 1600000000}}, 
		},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/refresh", handler.RefreshResults)

	formData := url.Values{"limit": {"999"}} // Exceeds total
	req, _ := http.NewRequest(http.MethodPost, "/admin/refresh", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "exceeds total contests")
}

func TestCancelSync_RepoError(t *testing.T) {
	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SetSyncCancelSignalErr: errors.New("redis failure"),
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/cancel-sync", handler.CancelSync)

	req, _ := http.NewRequest(http.MethodPost, "/admin/cancel-sync", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddContest_SuccessPath(t *testing.T) {
	
	mr, _ := miniredis.Run()
	defer mr.Close()
	workers.InitClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	defer workers.CloseClient()

	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncStatus: map[string]interface{}{"status": "idle"},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.POST("/admin/contests/add", handler.AddContest)

	formData := url.Values{"cfid": {"1001"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/contests/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin/contests", w.Header().Get("Location"))
}


func TestGetSyncStatus_StateTransitions(t *testing.T) {
	
	mr, _ := miniredis.Run()
	defer mr.Close()
	database.RedisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cfg := &configs.Config{AdminPasswordHash: "hashed_secret"}
	mockRepo := &MockRepository{
		SyncStatus: map[string]interface{}{"status": "processing"},
	}
	handler := NewAdminHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminTestRouter()
	r.GET("/admin/sync-status", handler.GetSyncStatus)

	
	req, _ := http.NewRequest(http.MethodGet, "/admin/sync-status", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req)

	assert.Equal(t, http.StatusOK, w1.Code)

	
	mockRepo.SyncStatus = map[string]interface{}{"status": "idle"}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)

	
	val, err := database.RedisClient.Get(context.Background(), "sync:was_processing").Result()
	assert.NotNil(t, err) 
	assert.Equal(t, "", val)
}
