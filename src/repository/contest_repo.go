package repository

import (
	"database/sql"
	"leaderboard/src/database"
)

// returns all contests ordered by start_time DESC
func GetContests() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_contest_id, name, start_time 
		FROM contests 
		ORDER BY start_time DESC
	`)
}

//  inserts a new contest
func AddContest(cfID int, name string, startTime int64) error {
	_, err := database.DB.Exec(`
		INSERT INTO contests (codeforces_contest_id, name, start_time) 
		VALUES (?, ?, ?)
	`, cfID, name, startTime)
	return err
}

//  deletes a contest by id
func DeleteContest(id string) error {

	_, err := database.DB.Exec(
		`DELETE FROM user_contest_results WHERE contest_id = ?`,
		id,
	)

	if err != nil {
		return err
	}

	_, err = database.DB.Exec(`
		DELETE FROM contests 
		WHERE id = ?
	`, id)

	return err
}

// returns all contest IDs and codeforces_contest_ids
func GetContestIDs() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_contest_id 
		FROM contests
	`)
}
