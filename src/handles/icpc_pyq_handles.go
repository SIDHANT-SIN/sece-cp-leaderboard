package handles

import (
	"net/http"

	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

func ShowProblemsNew(c *gin.Context) {

	rows, err := repository.GetProblemsNew()
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
	defer rows.Close()


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
			id int64
			contest string
			year int
			title string
			link string
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
				ID: id,
				Title: title,
				Link: link,
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