//go:build integration
// +build integration

package repository

import (
	"database/sql"
	"testing"

	"leaderboard/src/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupCodeforcesDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE past_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT,
			current_rating INTEGER DEFAULT 0,
			max_rating INTEGER DEFAULT 0,
			title TEXT,
			last_updated TEXT
		);
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT,
			display_name TEXT
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestCodeforcesRepo_Integration(t *testing.T) {
	db := setupCodeforcesDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO past_users (codeforces_handle) VALUES ('past_tourist'), ('past_benq')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (codeforces_handle, display_name) VALUES ('tourist', 'Gennady'), ('benq', 'Benjamin')`)
	require.NoError(t, err)

	handles, err := GetPastUserHandles()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"past_tourist", "past_benq"}, handles)

	err = UpdatePastUserRating(3500, 3800, "Legendary Grandmaster", "past_tourist")
	assert.NoError(t, err)

	var rating, maxRating int
	var title string
	err = db.QueryRow(`SELECT current_rating, max_rating, title FROM past_users WHERE codeforces_handle = 'past_tourist'`).Scan(&rating, &maxRating, &title)
	assert.NoError(t, err)
	assert.Equal(t, 3500, rating)
	assert.Equal(t, 3800, maxRating)
	assert.Equal(t, "Legendary Grandmaster", title)

	usersList := GetUsersList()
	assert.Len(t, usersList, 2)

	handlesMap := make(map[string]string)
	for _, u := range usersList {
		handlesMap[u["Username"].(string)] = u["DisplayName"].(string)
	}

	assert.Equal(t, "Gennady", handlesMap["tourist"])
	assert.Equal(t, "Benjamin", handlesMap["benq"])
}
