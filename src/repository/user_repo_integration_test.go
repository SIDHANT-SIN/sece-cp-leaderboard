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

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT UNIQUE NOT NULL,
			display_name TEXT
		);
		CREATE TABLE user_contest_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestAddAndGetUsers_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := AddUser("tourist", "Gennady Korotkevich")
	assert.NoError(t, err)

	rows, err := GetUsers()
	assert.NoError(t, err)
	defer rows.Close()

	type User struct {
		ID     int
		Handle string
		Name   string
	}

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Handle, &u.Name)
		assert.NoError(t, err)
		users = append(users, u)
	}

	require.Len(t, users, 1)
	assert.Equal(t, "tourist", users[0].Handle)
	assert.Equal(t, "Gennady Korotkevich", users[0].Name)
}

func TestDeleteUser_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := db.Exec(`INSERT INTO users (codeforces_handle, display_name) VALUES ('vjudge', 'Virtual Judge')`)
	require.NoError(t, err)

	lastID, err := result.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO user_contest_results (user_id) VALUES (?)`, lastID)
	require.NoError(t, err)

	userIDStr := strconv.FormatInt(lastID, 10)

	err = DeleteUser(userIDStr)
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", lastID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "User should have been deleted")

	err = db.QueryRow("SELECT COUNT(*) FROM user_contest_results WHERE user_id = ?", lastID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Dependent contest results should have been deleted")
}
