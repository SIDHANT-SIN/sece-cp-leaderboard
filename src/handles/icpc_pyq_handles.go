package handles

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Icpc_pyq struct {
	repo Repository
}

func NewIcpc_pyq(repo Repository) *Icpc_pyq {
	return &Icpc_pyq{
		repo: repo,
	}
}

// Bind to struct and use h.repo
func (h *Icpc_pyq) ShowProblemsNew(c *gin.Context) {

	rows, err := h.repo.GetProblemsNew()
	if err != nil {
		c.HTML(
			http.StatusInternalServerError,
			"problems_list.tmpl",
			gin.H{
				"error": "failed to load problems",
			},
		)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	type Problem struct {
		ID    int64
		Title string
		Link  string
	}

	contests := map[string][]Problem{
		"Prelims": {},
		"chn":     {},
		"k":       {},
		"amr":     {},
	}

	for rows.Next() {
		var (
			id      int64
			contest string
			year    int
			title   string
			link    string
		)

		err := rows.Scan(
			&id,
			&contest,
			&year,
			&title,
			&link,
		)

		if err != nil {
			continue
		}

		contests[contest] = append(
			contests[contest],
			Problem{
				ID:    id,
				Title: title,
				Link:  link,
			},
		)
	}

	c.HTML(
		http.StatusOK,
		"problems_list.tmpl",
		gin.H{
			"Problems": contests,
		},
	)
}
