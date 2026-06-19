package repository

import (
	"leaderboard/src/database"
	"database/sql"
)

// InsertTestcase inserts a testcase record linking to the uploaded Azure file URLs
func InsertTestcase(problemID int64, inputURL, outputURL string) error {
	_, err := database.DB.Exec(`
		INSERT INTO icpc_testcases (
			problem_id,
			testcase_input,
			testcase_output
		) VALUES (?, ?, ?)
	`, problemID, inputURL, outputURL)
	return err
}

func GetProblems() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT
			id,
			title
		FROM icpc_pyq
		ORDER BY id DESC
	`)
}

func GetProblemByID(id string) (*sql.Row, error) {

	return database.DB.QueryRow(`
		SELECT
			id,
			title,
			statement,
			time_limit,
			memory_limit,
			input_desc,
			output_desc,
			constraints,
			sample_input,
			sample_output,
			explanation
		FROM icpc_pyq
		WHERE id = ?
	`, id), nil
}