package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetupRoutes_AndRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	err := os.MkdirAll("templates", 0755)
	assert.NoError(t, err)
	dummyFile := filepath.Join("templates", "dummy.html")
	err = os.WriteFile(dummyFile, []byte("<html></html>"), 0644)
	assert.NoError(t, err)
	defer os.RemoveAll("templates")

	cfg := &configs.Config{Port: "8080"}
	r := SetupRoutes(cfg)
	assert.NotNil(t, r)

	routesToTest := []struct {
		url          string
		expectedCode int
		expectedLoc  string
	}{
		{"/", http.StatusSeeOther, "/leaderboard"},
		{"/index", http.StatusSeeOther, "/leaderboard"},
	}

	for _, tc := range routesToTest {
		req, _ := http.NewRequest("GET", tc.url, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, tc.expectedCode, w.Code)
		assert.Equal(t, tc.expectedLoc, w.Header().Get("Location"))
	}

	expectedRoutes := map[string]string{
		"/admin":                     "GET",
		"/admin_login":               "GET",
		"/maintainer":                "GET",
		"/maintainer/login":          "POST",
		"/maintainer/dashboard":      "GET",
		"/admin/check_cf_api":        "POST",
		"/maintainer/users":          "GET",
		"/maintainer/users/add":      "POST",
		"/maintainer/users/delete":   "POST",
		"/admin/users/delete":        "POST",
		"/admin/users":               "GET",
		"/admin/users/add":           "POST",
		"/admin/contests":            "GET",
		"/admin/contests/add":        "POST",
		"/admin/contests/delete":     "POST",
		"/admin/sync_status":         "GET",
		"/admin/cancel_sync":         "POST",
		"/leaderboard":               "GET",
		"/past_events":               "GET",
		"/past_leaderboard":          "GET",
		"/maintainer/refresh_rating": "POST",
		"/admin/refresh_results":     "POST",
		"/api/health/ping":           "GET",
		"/api/maintenance/purge":     "POST",
		"/problems":                  "GET",
		"/maintainer/icpc_pyq":       "GET",
	}

	registeredRoutes := r.Routes()

	for _, route := range registeredRoutes {
		if expectedMethod, exists := expectedRoutes[route.Path]; exists {
			assert.Equal(t, expectedMethod, route.Method, "Route %s registered with wrong method", route.Path)

			delete(expectedRoutes, route.Path)
		}
	}

	assert.Empty(t, expectedRoutes, "Some routes were missing from registration: %v", expectedRoutes)
}
