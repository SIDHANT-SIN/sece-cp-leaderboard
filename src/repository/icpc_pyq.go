package repository

import (
	"leaderboard/src/database"
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
