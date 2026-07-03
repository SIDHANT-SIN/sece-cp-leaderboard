package handles

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewPastEventsHandler(t *testing.T) {
	cfg := &configs.Config{
		SupaBase:   "test-project",
		FolderName: "gallery",
	}

	h := NewPastEventsHandler(cfg)

	assert.NotNil(t, h)
	assert.Equal(t, cfg, h.cfg)
}

func TestGenerateSlides(t *testing.T) {
	slides := generateSlides("project123", "gallery", 4, 2, 3)

	assert.Len(t, slides, 3)

	assert.Equal(
		t,
		"https://project123.supabase.co/storage/v1/object/public/gallery/4_2_1.jpg",
		slides[0].URL,
	)

	assert.Equal(
		t,
		"Batch 2024 - Event 2 - Photo 1",
		slides[0].Caption,
	)

	assert.Equal(
		t,
		"https://project123.supabase.co/storage/v1/object/public/gallery/4_2_3.jpg",
		slides[2].URL,
	)
}

func TestGenerateSlidesZero(t *testing.T) {
	slides := generateSlides("abc", "folder", 1, 1, 0)
	assert.Empty(t, slides)
}

func TestPastEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	tmpl := template.Must(template.New("events.tmpl").Parse("EVENTS PAGE"))
	router.SetHTMLTemplate(tmpl)

	cfg := &configs.Config{
		SupaBase:   "project123",
		FolderName: "gallery",
	}

	handler := NewPastEventsHandler(cfg)

	router.GET("/past_events", handler.PastEvents)

	req := httptest.NewRequest(http.MethodGet, "/past_events", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "EVENTS PAGE")
}
