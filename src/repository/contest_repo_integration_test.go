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

func setupContestsDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE contests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_contest_id INTEGER UNIQUE,
			name TEXT,
			start_time INTEGER
		);
		CREATE TABLE user_contest_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			contest_id INTEGER
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestContests_Integration(t *testing.T) {
	db := setupContestsDB(t)
	defer db.Close()

	err := AddContest(1234, "Div2 Round 1", 1600000000)
	assert.NoError(t, err)

	rows, err := GetContests()
	assert.NoError(t, err)
	defer rows.Close()

	var count int
	for rows.Next() {
		var id, cfID, startTime int
		var name string
		err := rows.Scan(&id, &cfID, &name, &startTime)
		assert.NoError(t, err)
		assert.Equal(t, 1234, cfID)
		assert.Equal(t, "Div2 Round 1", name)
		count++
	}
	assert.Equal(t, 1, count)

	idsRows, err := GetContestIDs()
	assert.NoError(t, err)
	defer idsRows.Close()

	count = 0
	for idsRows.Next() {
		count++
	}
	assert.Equal(t, 1, count)
}

func TestDeleteContest_Integration(t *testing.T) {
	db := setupContestsDB(t)
	defer db.Close()

	res, err := db.Exec(`INSERT INTO contests (codeforces_contest_id, name, start_time) VALUES (999, 'Test', 0)`)
	require.NoError(t, err)

	id, _ := res.LastInsertId()
	idStr := strconv.FormatInt(id, 10)

	_, err = db.Exec(`INSERT INTO user_contest_results (contest_id) VALUES (?)`, idStr)
	require.NoError(t, err)

	err = DeleteContest(idStr)
	assert.NoError(t, err)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM contests`).Scan(&count)
	assert.Equal(t, 0, count)

	db.QueryRow(`SELECT COUNT(*) FROM user_contest_results`).Scan(&count)
	assert.Equal(t, 0, count)
}
