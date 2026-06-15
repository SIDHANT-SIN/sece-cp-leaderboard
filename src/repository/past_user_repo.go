package repository

import (
	"leaderboard/src/database"
	"database/sql"
	_ "github.com/tursodatabase/libsql-client-go/libsql"

)


func GetPastUsers() (*sql.Rows, error) {
	return database.DB.Query(`
		SELECT id, codeforces_handle, display_name, batch_year, current_rating, max_rating, title
        FROM past_users
	`)
	//rows, err := db.Query(`
        
    // `)
}

func AddPastUser(handle string, display string, batch int) error {

	_, err := database.DB.Exec(`
	INSERT INTO past_users (codeforces_handle, display_name, batch_year)
         VALUES (?, ?, ?)`,
		handle,
		display,
		batch,
	)
	return err
}

func DeletePastUser(id string) error {

	_, err := database.DB.Exec(
		`DELETE FROM past_users WHERE id = ?`,
		id,
	)

	//_, err = db.Exec("DELETE FROM past_users WHERE id = ?", id)

	return err
}