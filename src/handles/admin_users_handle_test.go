package handles

import (
	"errors"
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

// setupAdminUsersTestRouter creates a test Gin engine with dummy templates
func setupAdminUsersTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	tmpl := template.Must(template.New("").Parse(""))
	template.Must(tmpl.New("admin_users.tmpl").Parse("Admin Users"))
	template.Must(tmpl.New("admin.tmpl").Parse("Admin Dashboard"))
	r.SetHTMLTemplate(tmpl)

	return r
}

func TestShowUsers_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{
		UsersRows: &MockRows{
			Data: [][]any{
				{1, "coder1", "Coder One"},
				{2, "coder2", "Coder Two"},
			},
		},
	}
	mockCache := &MockCacheBuilder{}

	handler := NewAdminUsersHandler(mockRepo, mockCache, cfg)
	r := setupAdminUsersTestRouter()
	r.GET("/admin/users", handler.ShowUsers)

	req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowUsers_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	handler := NewAdminUsersHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminUsersTestRouter()
	r.GET("/admin/users", handler.ShowUsers)

	req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin", w.Header().Get("Location"))
}

func TestShowUsers_DBError(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{
		UsersErr: errors.New("db connection lost"),
	}
	handler := NewAdminUsersHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminUsersTestRouter()
	r.GET("/admin/users", handler.ShowUsers)

	req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "DB error")
}

func TestAddUser_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{}
	mockCache := &MockCacheBuilder{}

	handler := NewAdminUsersHandler(mockRepo, mockCache, cfg)
	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/add", handler.AddUser)

	formData := url.Values{"handle": {"new_user"}, "display_name": {"New User"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin", w.Header().Get("Location"))
}

func TestAddUser_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	handler := NewAdminUsersHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/add", handler.AddUser)

	req, _ := http.NewRequest(http.MethodPost, "/admin/users/add", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin_login", w.Header().Get("Location"))
}

func TestAddUser_RepoError(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{
		AddUserErr: errors.New("user already exists"),
		UsersList:  []map[string]interface{}{{"id": 1, "handle": "existing"}},
	}
	mockCache := &MockCacheBuilder{}

	handler := NewAdminUsersHandler(mockRepo, mockCache, cfg)
	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/add", handler.AddUser)

	formData := url.Values{"handle": {"duplicate_user"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Admin Dashboard") 
}

func TestDeleteUser_Authenticated_Success(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{}
	mockCache := &MockCacheBuilder{}

	handler := NewAdminUsersHandler(mockRepo, mockCache, cfg)
	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/delete", handler.DeleteUser)

	formData := url.Values{"id": {"123"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/delete", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin/users", w.Header().Get("Location"))
}

func TestDeleteUser_Unauthenticated(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	handler := NewAdminUsersHandler(&MockRepository{}, &MockCacheBuilder{}, cfg)

	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/delete", handler.DeleteUser)

	req, _ := http.NewRequest(http.MethodPost, "/admin/users/delete", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin", w.Header().Get("Location"))
}

func TestDeleteUser_RepoError(t *testing.T) {
	cfg := &configs.Config{
		AdminPasswordHash: "hashed_secret",
	}
	mockRepo := &MockRepository{
		DeleteUserErr: errors.New("user not found"),
	}
	handler := NewAdminUsersHandler(mockRepo, &MockCacheBuilder{}, cfg)

	r := setupAdminUsersTestRouter()
	r.POST("/admin/users/delete", handler.DeleteUser)

	formData := url.Values{"id": {"999"}}
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/delete", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_logged_in", Value: "hashed_secret"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Could not delete user: user not found")
}
