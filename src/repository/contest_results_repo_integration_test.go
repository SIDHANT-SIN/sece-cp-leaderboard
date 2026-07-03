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

func setupResultsDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE user_contest_results (
			user_id INTEGER,
			contest_id INTEGER,
			rank INTEGER,
			points INTEGER,
			last_updated TEXT,
			PRIMARY KEY (user_id, contest_id)
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestContestResults_Integration(t *testing.T) {
	db := setupResultsDB(t)
	defer db.Close()

	err := UpsertResult(1, 100, 5, 1500)
	assert.NoError(t, err)
	err = UpsertResult(1, 100, 2, 2000)
	assert.NoError(t, err)
	err = UpsertResult(2, 100, 10, 500)
	assert.NoError(t, err)

	rows, err := GetAllResults()
	assert.NoError(t, err)
	defer rows.Close()

	var count int
	for rows.Next() {
		var uid, cid, rank, points int
		err := rows.Scan(&uid, &cid, &rank, &points)
		assert.NoError(t, err)
		if uid == 1 {
			assert.Equal(t, 2, rank)
			assert.Equal(t, 2000, points)
		}
		count++
	}
	assert.Equal(t, 2, count)

	err = DeleteResultsByContest("100")
	assert.NoError(t, err)

	db.QueryRow(`SELECT COUNT(*) FROM user_contest_results`).Scan(&count)
	assert.Equal(t, 0, count)

	UpsertResult(3, 101, 1, 3000)
	err = DeleteAllResults()
	assert.NoError(t, err)

	db.QueryRow(`SELECT COUNT(*) FROM user_contest_results`).Scan(&count)
	assert.Equal(t, 0, count)
}
