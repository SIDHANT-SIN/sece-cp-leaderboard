package handles

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestRouter creates a test Gin engine with dummy templates

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	tmpl := template.Must(template.New("").Parse(""))
	template.Must(tmpl.New("leaderboard.tmpl").Parse("Leaderboard Dummy"))
	template.Must(tmpl.New("past_leaderboard.tmpl").Parse("Past Leaderboard Dummy"))
	r.SetHTMLTemplate(tmpl)

	return r
}

func TestShowLeaderboard_CacheHit(t *testing.T) {
	
	mockRepo := &MockRepository{
		LeaderboardCacheUsers: []map[string]interface{}{
			{"id": 1, "handle": "testuser", "display_name": "Test User"},
		},
	}
	cfg := &configs.Config{Logo: "logo.png"}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/leaderboard", handler.ShowLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowLeaderboard_CacheMiss_Success(t *testing.T) {
	
	mockRepo := &MockRepository{
		LeaderboardCacheUsers: nil, 
		UsersRows: &MockRows{
			Data: [][]any{{1, "handle1", "User One"}, {2, "handle2", "User Two"}},
		},
		ContestsRows: &MockRows{
			Data: [][]any{{10, 100, "Contest A", 1600000000}},
		},
		AllResultsRows: &MockRows{
			Data: [][]any{
				{1, 10, 1, 100}, 
				{2, 10, 2, 50},  
			},
		},
	}
	cfg := &configs.Config{Logo: "logo.png"}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/leaderboard", handler.ShowLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowLeaderboard_DBErrorUsers(t *testing.T) {
	
	mockRepo := &MockRepository{
		LeaderboardCacheUsers: nil,
		UsersErr:              errors.New("db connection failed"),
	}
	cfg := &configs.Config{Logo: "logo.png"}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/leaderboard", handler.ShowLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)


	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "DB error")
}

func TestShowPastLeaderboard_Success(t *testing.T) {
	mockRepo := &MockRepository{
		PastUsersByBatchRows: &MockRows{
			
			Data: [][]any{
				{1, "pastuser", "Past User", 1500, 1600, "Expert", 2023},
				{2, "pro_user", "Pro User", 1800, 1900, "Candidate Master", 2023},
			},
		},
	}
	cfg := &configs.Config{}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/past_leaderboard", handler.ShowPastLeaderboard)


	req, _ := http.NewRequest(http.MethodGet, "/past_leaderboard?batch=2023", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRebuildLeaderboardCache_Success(t *testing.T) {
	mockRepo := &MockRepository{
		UsersRows: &MockRows{
			Data: [][]any{{1, "handle1", "User One"}},
		},
		ContestsRows: &MockRows{
			Data: [][]any{{10, 100, "Contest A", 1600000000}},
		},
		AllResultsRows: &MockRows{
			Data: [][]any{{1, 10, 1, 100}},
		},
	}
	cfg := &configs.Config{}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	err := handler.RebuildLeaderboardCache()

	assert.NoError(t, err)
}

func TestRebuildLeaderboardCache_UserDBError(t *testing.T) {
	mockRepo := &MockRepository{
		UsersErr: errors.New("db timeout"),
	}
	cfg := &configs.Config{}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	err := handler.RebuildLeaderboardCache()

	assert.Error(t, err)
	assert.Equal(t, "db timeout", err.Error())
}

func TestShowPastLeaderboard_DBError(t *testing.T) {
	mockRepo := &MockRepository{
		PastUsersByBatchErr: errors.New("db error"),
	}
	cfg := &configs.Config{}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/past_leaderboard", handler.ShowPastLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/past_leaderboard?batch=2023", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShowPastLeaderboard_ScanError(t *testing.T) {
	mockRepo := &MockRepository{
		PastUsersByBatchRows: &MockRows{
			Data: [][]any{
				{1, "pastuser", "Past User", 1500, 1600, "Expert", 2023},
			},
			Err: errors.New("scan error"),
		},
	}
	cfg := &configs.Config{}
	handler := NewLeaderboardHandler(mockRepo, cfg)

	r := setupTestRouter()
	r.GET("/past_leaderboard", handler.ShowPastLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/past_leaderboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Row errors gracefully skip to the next row
}
