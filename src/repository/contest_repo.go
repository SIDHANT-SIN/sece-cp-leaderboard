package repository

import (
	"database/sql"
	"leaderboard/src/database"
)

// GetContests returns all contests ordered by start_time DESC
func GetContests() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_contest_id, name, start_time 
		FROM contests 
		ORDER BY start_time DESC
	`)
}

// AddContest inserts a new contest
func AddContest(cfID int, name string, startTime int64) error {
	_, err := database.DB.Exec(`
		INSERT INTO contests (codeforces_contest_id, name, start_time) 
		VALUES (?, ?, ?)
	`, cfID, name, startTime)
	return err
}

// DeleteContest deletes a contest by id
func DeleteContest(id string) error {
	_, err := database.DB.Exec(`
		DELETE FROM contests 
		WHERE id = ?
	`, id)
	return err
}

// DeleteAllContests deletes all contests from the table
func DeleteAllContests() error {
	_, err := database.DB.Exec(`
		DELETE FROM contests
	`)
	return err
}

// InsertContestIgnore inserts a contest if it doesn't already exist
func InsertContestIgnore(cfID int, name string, startTime int64) error {
	_, err := database.DB.Exec(`
		INSERT OR IGNORE INTO contests (codeforces_contest_id, name, start_time) 
		VALUES (?, ?, ?)
	`, cfID, name, startTime)
	return err
}

// GetContestIDs returns all contest IDs and codeforces_contest_ids
func GetContestIDs() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_contest_id 
		FROM contests
	`)
}
