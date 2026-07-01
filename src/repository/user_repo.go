package repository

import (
	"leaderboard/src/database"
	"database/sql"
	_ "github.com/tursodatabase/libsql-client-go/libsql"

)

func GetUsers() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_handle, display_name
		FROM users
	`)
}

func AddUser(handle string, displayName string) error {

	_, err := database.DB.Exec(
		`INSERT INTO users
		(codeforces_handle, display_name)
		VALUES (?, ?)`,
		handle,
		displayName,
	)

	return err
}

func DeleteUser(id string) error {

	_, err := database.DB.Exec(
		`DELETE FROM user_contest_results WHERE user_id = ?`,
		id,
	)

	if err != nil {
		return err
	}

	_, err = database.DB.Exec(
		`DELETE FROM users WHERE id = ?`,
		id,
	)

	return err
}