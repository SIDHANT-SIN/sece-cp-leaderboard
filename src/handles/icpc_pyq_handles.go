package handles

import (
	"fmt"
	"net/http"

	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

func ShowProblems(c *gin.Context) {

	rows, err := repository.GetProblems()
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

	var probs []map[string]any

	for rows.Next() {

		var id int64
		var title string

		err := rows.Scan(
			&id,
			&title,
		)
		if err != nil {
			continue
		}

		probs = append(probs, map[string]any{
			"ID":    id,
			"Title": title,
		})
	}

	fmt.Printf("chec")

	c.HTML(
		http.StatusOK,
		"problems_list.tmpl",
		gin.H{
			"Problems": probs,
		},
	)
}


func ShowProblem(c *gin.Context) {

	id := c.Param("id")

	row, err := repository.GetProblemByID(id)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	var p struct {
		ID           int64
		Title        string
		Statement    string
		TimeLimit    int
		MemoryLimit  int
		InputDesc    string
		OutputDesc   string
		Constraints  string
		SampleInput  string
		SampleOutput string
		Explanation  string
	}

	err = row.Scan(
		&p.ID,
		&p.Title,
		&p.Statement,
		&p.TimeLimit,
		&p.MemoryLimit,
		&p.InputDesc,
		&p.OutputDesc,
		&p.Constraints,
		&p.SampleInput,
		&p.SampleOutput,
		&p.Explanation,
	)

	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	c.HTML(
		http.StatusOK,
		"problem.tmpl",
		gin.H{
			"Problem": p,
		},
	)
}

func ShowEditor(c *gin.Context) {

	id := c.Param("id")

	row, err := repository.GetProblemByID(id)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	var p struct {
		ID    int64
		Title string

		Statement string

		TimeLimit   int
		MemoryLimit int

		InputDesc    string
		OutputDesc   string
		Constraints  string
		SampleInput  string
		SampleOutput string
		Explanation  string
	}

	err = row.Scan(
		&p.ID,
		&p.Title,
		&p.Statement,
		&p.TimeLimit,
		&p.MemoryLimit,
		&p.InputDesc,
		&p.OutputDesc,
		&p.Constraints,
		&p.SampleInput,
		&p.SampleOutput,
		&p.Explanation,
	)

	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	c.HTML(
		http.StatusOK,
		"editor.tmpl",
		gin.H{
			"Problem": p,
		},
	)
}