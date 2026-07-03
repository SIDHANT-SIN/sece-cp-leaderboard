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

func setupMaintainerRouter(h *MaintainerUsersHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	tmpl := template.Must(template.New("maintainer_users.tmpl").Parse("OK"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/users", h.ShowPastUsers)
	r.POST("/add", h.AddPastUser)
	r.POST("/delete", h.DeletePastUser)
	r.POST("/refresh", h.RefreshRating)
	r.GET("/icpc", h.CreateICPCProblem)

	return r
}

func TestNewMaintainerUsersHandler(t *testing.T) {
	cfg := &configs.Config{}
	repo := &MockRepository{}
	queue := &MockTaskEnqueuer{}

	h := NewMaintainerUsersHandler(cfg, repo, queue)

	assert.NotNil(t, h)
	assert.Equal(t, cfg, h.cfg)
	assert.Equal(t, repo, h.repo)
	assert.Equal(t, queue, h.taskQueue)
}

func TestShowPastUsersUnauthorized(t *testing.T) {
	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestShowPastUsersSuccess(t *testing.T) {
	repo := &MockRepository{
		PastUsersRows: &MockRows{
			Data: [][]any{
				{
					1,
					"tourist",
					"Gennady",
					2023,
					3900,
					3979,
					"Legendary Grandmaster",
				},
			},
		},
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "OK")
}

func TestShowPastUsersDBError(t *testing.T) {
	repo := &MockRepository{
		PastUsersErr: errors.New("db failed"),
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShowPastUsersScanError(t *testing.T) {
	repo := &MockRepository{
		PastUsersRows: &MockRows{
			Data: [][]any{
				{1, "tourist", "Gennady", 2023, 3900, 3979, "Legendary Grandmaster"},
			},
			Err: errors.New("scan failed"),
		},
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddPastUserSuccess(t *testing.T) {
	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	form := url.Values{}
	form.Add("handle", "tourist")
	form.Add("display_name", "Gennady")
	form.Add("batch", "2023")

	req := httptest.NewRequest(
		http.MethodPost,
		"/add",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestAddPastUserInvalidBatch(t *testing.T) {
	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	form := url.Values{}
	form.Add("handle", "tourist")
	form.Add("display_name", "Gennady")
	form.Add("batch", "abcd")

	req := httptest.NewRequest(
		http.MethodPost,
		"/add",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddPastUserRepositoryError(t *testing.T) {
	repo := &MockRepository{
		AddPastUserErr: errors.New("insert failed"),
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	form := url.Values{}
	form.Add("handle", "tourist")
	form.Add("display_name", "Gennady")
	form.Add("batch", "2023")

	req := httptest.NewRequest(
		http.MethodPost,
		"/add",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeletePastUserSuccess(t *testing.T) {
	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	form := url.Values{}
	form.Add("id", "1")

	req := httptest.NewRequest(
		http.MethodPost,
		"/delete",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestDeletePastUserRepositoryError(t *testing.T) {
	repo := &MockRepository{
		DeletePastUserErr: errors.New("delete failed"),
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	form := url.Values{}
	form.Add("id", "1")

	req := httptest.NewRequest(
		http.MethodPost,
		"/delete",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefreshRatingAlreadyRunning(t *testing.T) {
	repo := &MockRepository{
		SyncStatus: map[string]interface{}{
			"status": "processing",
			"job_id": "abc123",
		},
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRefreshRatingNoUsers(t *testing.T) {
	repo := &MockRepository{
		SyncStatus: map[string]interface{}{
			"status": "idle",
		},
		PastHandles: []string{},
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshRatingSuccess(t *testing.T) {
	repo := &MockRepository{
		SyncStatus: map[string]interface{}{
			"status": "idle",
		},
		PastHandles: []string{
			"tourist",
			"Benq",
		},
	}

	queue := &MockTaskEnqueuer{}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		queue,
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestRefreshRatingQueueError(t *testing.T) {
	repo := &MockRepository{
		SyncStatus: map[string]interface{}{
			"status": "idle",
		},
		PastHandles: []string{
			"tourist",
		},
	}

	queue := &MockTaskEnqueuer{
		RefreshErr: errors.New("queue failed"),
	}

	handler := NewMaintainerUsersHandler(
		&configs.Config{},
		repo,
		queue,
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "true",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateICPCProblemAuthorized(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: "secret",
	}

	handler := NewMaintainerUsersHandler(
		cfg,
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/icpc", nil)
	req.AddCookie(&http.Cookie{
		Name:  "maintainer_logged_in",
		Value: "secret",
	})

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestCreateICPCProblemUnauthorized(t *testing.T) {
	cfg := &configs.Config{
		MaintainerPassword: "secret",
	}

	handler := NewMaintainerUsersHandler(
		cfg,
		&MockRepository{},
		&MockTaskEnqueuer{},
	)

	router := setupMaintainerRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/icpc", nil)

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestDefaultTaskEnqueuer(t *testing.T) {
	
	enqueuer := NewDefaultTaskEnqueuer()

	err := enqueuer.EnqueueRefreshRatingTask("job123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Asynq client")

	err = enqueuer.EnqueueAddContestTask("job123", "cf_id_10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Asynq client")

	err = enqueuer.EnqueueBatchRefreshTask("job123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Asynq client")
}
