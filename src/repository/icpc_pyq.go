package repository

import (
	"database/sql"
	"leaderboard/src/database"
)

func GetProblemsNew() (*sql.Rows, error) {

	rows, err := database.DB.Query(`
		SELECT 
			id,
			contest_name,
			year,
			title,
			link
		FROM problems
		ORDER BY contest_name
	`)

	if err != nil {
		return nil, err
	}

	return rows, nil
}
