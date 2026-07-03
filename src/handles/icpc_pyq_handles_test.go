package handles

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupIcpcRouter(h *Icpc_pyq) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tmpl := template.Must(template.New("problems_list.tmpl").Parse("Problems List Mock"))
	r.SetHTMLTemplate(tmpl)
	r.GET("/icpc", h.ShowProblemsNew)
	return r
}

func TestShowProblemsNew_Success(t *testing.T) {
	mockRepo := &MockRepository{
		GetProblemsNewRows: &MockRows{
			Data: [][]any{
				{int64(1), "Prelims", 2023, "Problem A", "http://link1"},
				{int64(2), "chn", 2023, "Problem B", "http://link2"},
			},
		},
	}
	h := NewIcpc_pyq(mockRepo)
	r := setupIcpcRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/icpc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Problems List Mock")
}

func TestShowProblemsNew_DBError(t *testing.T) {
	mockRepo := &MockRepository{
		GetProblemsNewErr: errors.New("database connection lost"),
	}
	h := NewIcpc_pyq(mockRepo)
	r := setupIcpcRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/icpc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShowProblemsNew_ScanError(t *testing.T) {
	mockRepo := &MockRepository{
		GetProblemsNewRows: &MockRows{
			Data: [][]any{
				{int64(1), "Prelims", 2023, "Problem A", "http://link1"},
			},
			Err: errors.New("scan failure"),
		},
	}
	h := NewIcpc_pyq(mockRepo)
	r := setupIcpcRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/icpc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
