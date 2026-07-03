//go:build integration
// +build integration

package repository

import (
	"database/sql"
	"strconv"
	"testing"

	"leaderboard/src/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupPastUsersDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE past_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT,
			display_name TEXT,
			batch_year INTEGER,
			current_rating INTEGER DEFAULT 0,
			max_rating INTEGER DEFAULT 0,
			title TEXT DEFAULT 'Newbie'
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestPastUsers_Integration(t *testing.T) {
	db := setupPastUsersDB(t)
	defer db.Close()

	err := AddPastUser("old_tourist", "Gennady", 2020)
	assert.NoError(t, err)

	rows, err := GetPastUsers()
	assert.NoError(t, err)
	defer rows.Close()

	var count int
	for rows.Next() {
		var id, batch, curr, max int
		var handle, name, title string
		err := rows.Scan(&id, &handle, &name, &batch, &curr, &max, &title)
		assert.NoError(t, err)
		assert.Equal(t, "old_tourist", handle)
		count++
	}
	assert.Equal(t, 1, count)

	batchRows, err := GetPastUsersByBatch("2020")
	assert.NoError(t, err)
	defer batchRows.Close()

	count = 0
	for batchRows.Next() {
		count++
	}
	assert.Equal(t, 1, count)
}

func TestDeletePastUser_Integration(t *testing.T) {
	db := setupPastUsersDB(t)
	defer db.Close()

	res, err := db.Exec(`INSERT INTO past_users (codeforces_handle, display_name, batch_year) VALUES ('test', 'test', 2021)`)
	require.NoError(t, err)

	id, _ := res.LastInsertId()
	idStr := strconv.FormatInt(id, 10)

	err = DeletePastUser(idStr)
	assert.NoError(t, err)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM past_users`).Scan(&count)
	assert.Equal(t, 0, count)
}
