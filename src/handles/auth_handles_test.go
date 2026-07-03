package handles

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupAuthTestRouter creates a test Gin engine with dummy templates for auth pages.
func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	tmpl := template.Must(template.New("").Parse(""))
	template.Must(tmpl.New("admin_login.tmpl").Parse("Admin Login"))
	template.Must(tmpl.New("maintainer_login.tmpl").Parse("Maintainer Login"))
	template.Must(tmpl.New("admin.tmpl").Parse("Admin Dashboard"))
	template.Must(tmpl.New("maintainer_dashboard.tmpl").Parse("Maintainer Dashboard"))
	template.Must(tmpl.New("maintainer_icpc.tmpl").Parse("Maintainer ICPC Page"))
	r.SetHTMLTemplate(tmpl)

	return r
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func TestAdminLoginPage(t *testing.T) {
	handler := NewAuthHandler(&configs.Config{}, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/admin_login", handler.AdminLoginPage)

	req, _ := http.NewRequest(http.MethodGet, "/admin_login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintainerLoginPage(t *testing.T) {
	handler := NewAuthHandler(&configs.Config{}, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/maintainer_login", handler.MaintainerLoginPage)

	req, _ := http.NewRequest(http.MethodGet, "/maintainer_login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminLogin_Success(t *testing.T) {
	cfg := &configs.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: hashPassword("secret123"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.POST("/admin_login", handler.AdminLogin)

	formData := url.Values{"username": {"admin"}, "password": {"secret123"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin_login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies)
	assert.Equal(t, "admin_logged_in", cookies[0].Name)
	assert.Equal(t, cfg.AdminPasswordHash, cookies[0].Value)
}

func TestAdminLogin_Failure(t *testing.T) {
	cfg := &configs.Config{
		AdminUsername:     "admin",
		AdminPasswordHash: hashPassword("secret123"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.POST("/admin_login", handler.AdminLogin)

	formData := url.Values{"username": {"admin"}, "password": {"wrongpassword"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin_login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminPage_Authenticated(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: hashPassword("secret123"),
	}

	mockRepo := &MockRepository{
		SyncHistoryRows: &MockRows{
			Data: [][]any{
				{"job-1", "completed", 5, 5, "", "2023-01-01", "2023-01-01"},
			},
		},
	}

	handler := NewAuthHandler(cfg, mockRepo)
	r := setupAuthTestRouter()
	r.GET("/admin", handler.AdminPage)

	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)

	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: cfg.AdminPasswordHash})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminPage_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: hashPassword("secret123"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/admin", handler.AdminPage)

	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin_login", w.Header().Get("Location"))
}

func TestMaintainerLogin_Success(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.POST("/maintainer_login", handler.MaintainerLogin)

	formData := url.Values{"password": {"maintpass"}}
	req, _ := http.NewRequest(http.MethodPost, "/maintainer_login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/maintainer/dashboard", w.Header().Get("Location"))

	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies)
	assert.Equal(t, "maintainer_logged_in", cookies[0].Name)
	assert.Equal(t, cfg.MaintainerPassword, cookies[0].Value)
}

func TestMaintainerLogin_Failure(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.POST("/maintainer_login", handler.MaintainerLogin)

	formData := url.Values{"password": {"wrongpass"}}
	req, _ := http.NewRequest(http.MethodPost, "/maintainer_login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMaintainerDashboard_Authenticated(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/maintainer/dashboard", handler.MaintainerDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/maintainer/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "maintainer_logged_in", Value: cfg.MaintainerPassword})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintainerDashboard_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/maintainer/dashboard", handler.MaintainerDashboard)

	req, _ := http.NewRequest(http.MethodGet, "/maintainer/dashboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/maintainer", w.Header().Get("Location"))
}

func TestMaintainerICPCPage_Authenticated(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/maintainer/icpc", handler.MaintainerICPCPage)

	req, _ := http.NewRequest(http.MethodGet, "/maintainer/icpc", nil)
	req.AddCookie(&http.Cookie{Name: "maintainer_logged_in", Value: cfg.MaintainerPassword})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintainerICPCPage_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: hashPassword("maintpass"),
	}
	handler := NewAuthHandler(cfg, &MockRepository{})
	r := setupAuthTestRouter()
	r.GET("/maintainer/icpc", handler.MaintainerICPCPage)

	req, _ := http.NewRequest(http.MethodGet, "/maintainer/icpc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/maintainer", w.Header().Get("Location"))
}
