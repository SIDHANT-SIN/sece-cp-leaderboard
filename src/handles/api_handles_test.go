//go:build integration
// +build integration
package handles

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"leaderboard/src/database"
)

// setupAPITestRouter creates a fresh Gin engine for API handler tests.
func setupAPITestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.Default()
}

// helper to create mock HTTP responses easily
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestCheckCFAPI_ClientError(t *testing.T) {
	mockClient := &MockHTTPClient{
		Err: errors.New("network timeout"),
	}
	handler := NewAPIHandler(mockClient, &configs.Config{})

	r := setupAPITestRouter()
	r.GET("/api/cf-status", handler.CheckCFAPI)

	req, _ := http.NewRequest(http.MethodGet, "/api/cf-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Codeforces API unreachable")
}

func TestCheckCFAPI_Non200(t *testing.T) {
	mockClient := &MockHTTPClient{
		Response: createMockResponse(http.StatusBadGateway, "502 Bad Gateway"),
	}
	handler := NewAPIHandler(mockClient, &configs.Config{})

	r := setupAPITestRouter()
	r.GET("/api/cf-status", handler.CheckCFAPI)

	req, _ := http.NewRequest(http.MethodGet, "/api/cf-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "CF returned non-200")
}

func TestCheckCFAPI_InvalidJSON(t *testing.T) {
	mockClient := &MockHTTPClient{
		Response: createMockResponse(http.StatusOK, "{ invalid json ]"),
	}
	handler := NewAPIHandler(mockClient, &configs.Config{})

	r := setupAPITestRouter()
	r.GET("/api/cf-status", handler.CheckCFAPI)

	req, _ := http.NewRequest(http.MethodGet, "/api/cf-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid JSON from CF")
}

func TestCheckCFAPI_StatusNotOK(t *testing.T) {
	mockClient := &MockHTTPClient{
		Response: createMockResponse(http.StatusOK, `{"status": "FAILED", "comment": "Codeforces is down"}`),
	}
	handler := NewAPIHandler(mockClient, &configs.Config{})

	r := setupAPITestRouter()
	r.GET("/api/cf-status", handler.CheckCFAPI)

	req, _ := http.NewRequest(http.MethodGet, "/api/cf-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "CF API status not OK")
}

func TestCheckCFAPI_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		Response: createMockResponse(http.StatusOK, `{"status": "OK"}`),
	}
	handler := NewAPIHandler(mockClient, &configs.Config{})

	r := setupAPITestRouter()
	r.GET("/api/cf-status", handler.CheckCFAPI)

	req, _ := http.NewRequest(http.MethodGet, "/api/cf-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Codeforces API is alive")
}

func TestSendPing(t *testing.T) {
	r := setupAPITestRouter()
	r.GET("/ping", SendPing)

	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

func TestPurg_Unauthorized(t *testing.T) {
	cfg := &configs.Config{
		CronSecret: "super-secret-token",
	}
	handler := NewAPIHandler(&MockHTTPClient{}, cfg)

	r := setupAPITestRouter()
	r.POST("/api/purge", handler.Purg)

	req, _ := http.NewRequest(http.MethodPost, "/api/purge", nil)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	req.Header.Set("X-Cron-Token", "wrong-token")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestDefaultHTTPClient_Get(t *testing.T) {
	
	client := &DefaultHTTPClient{}

	
	_, err := client.Get("http://invalid-url-that-doesnt-exist.loc")
	assert.Error(t, err)
}

func TestPurg_FailsSweep(t *testing.T) {
	cfg := &configs.Config{
		CronSecret: "super-secret-token",
	}
	handler := NewAPIHandler(&MockHTTPClient{}, cfg)


	database.RedisClient = redis.NewClient(&redis.Options{})

	r := setupAPITestRouter()
	r.POST("/api/purge", handler.Purg)

	req, _ := http.NewRequest(http.MethodPost, "/api/purge", nil)
	req.Header.Set("X-Cron-Token", "super-secret-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to purge Redis metadata")
}
