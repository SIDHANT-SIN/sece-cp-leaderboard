package repository

import (
	"database/sql"
	"leaderboard/src/database"
)

// queries all user contest results
func GetAllResults() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT user_id, contest_id, rank, points 
		FROM user_contest_results
	`)
}

//  deletes user contest results for a specific contest
func DeleteResultsByContest(contestID string) error {
	_, err := database.DB.Exec(`
		DELETE FROM user_contest_results 
		WHERE contest_id = ?
	`, contestID)
	return err
}

// deletes all user contest results
func DeleteAllResults() error {
	_, err := database.DB.Exec(`
		DELETE FROM user_contest_results
	`)
	return err
}

//  inserts or replaces a user's contest result
func UpsertResult(userID, contestID, rank, points int) error {
	_, err := database.DB.Exec(`
		INSERT OR REPLACE INTO user_contest_results
		(user_id, contest_id, rank, points, last_updated)
		VALUES (?, ?, ?, ?, strftime('%s', 'now'))
	`, userID, contestID, rank, points)
	return err
}
