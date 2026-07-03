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

func setupProblemsDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE problems (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			contest_name TEXT,
			year INTEGER,
			title TEXT,
			link TEXT
		);
	`)
	require.NoError(t, err)

	database.DB = db
	return db
}

func TestGetProblemsNew_Integration(t *testing.T) {
	db := setupProblemsDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO problems (contest_name, year, title, link) 
		VALUES ('Amritapuri', 2023, 'A. Problem 1', 'link1'), 
		       ('Kanpur', 2022, 'B. Problem 2', 'link2')
	`)
	require.NoError(t, err)

	rows, err := GetProblemsNew()
	assert.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var id, year int
		var cName, title, link string
		err := rows.Scan(&id, &cName, &year, &title, &link)
		assert.NoError(t, err)
		names = append(names, cName)
	}

	require.Len(t, names, 2)
	assert.Equal(t, "Amritapuri", names[0])
	assert.Equal(t, "Kanpur", names[1])
}
