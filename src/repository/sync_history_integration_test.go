//go:build integration
// +build integration

package repository

import (
	"database/sql"
	"testing"

	"leaderboard/src/database"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSyncDB(t *testing.T) (*sql.DB, *miniredis.Miniredis) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE sync_history (
			job_id TEXT PRIMARY KEY,
			status TEXT,
			successful_contests INTEGER,
			total_contests INTEGER,
			failed_contest_ids TEXT,
			started_at TEXT,
			completed_at TEXT
		);
	`)
	require.NoError(t, err)

	mr, err := miniredis.Run()
	require.NoError(t, err)

	database.DB = db
	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return db, mr
}

func TestSyncSQLFunctions_Integration(t *testing.T) {
	db, mr := setupSyncDB(t)
	defer db.Close()
	defer mr.Close()

	err := CreateSyncLog("job-123", 50)
	assert.NoError(t, err)

	err = UpdateSyncLog("job-123", "completed", 48, "1,2")
	assert.NoError(t, err)

	rows, err := GetRecentSyncHistory(1)
	assert.NoError(t, err)
	defer rows.Close()

	var count int
	for rows.Next() {
		var jobID, status, failed, started, completed string
		var succ, total int
		err := rows.Scan(&jobID, &status, &succ, &total, &failed, &started, &completed)
		assert.NoError(t, err)
		assert.Equal(t, "job-123", jobID)
		assert.Equal(t, "completed", status)
		assert.Equal(t, 48, succ)
		count++
	}
	assert.Equal(t, 1, count)
}

func TestSyncRedisFunctions_Integration(t *testing.T) {
	db, mr := setupSyncDB(t)
	defer db.Close()
	defer mr.Close()

	err := SetSyncCancelSignal()
	assert.NoError(t, err)

	val, err := mr.Get("sync:cancel_signal")
	assert.NoError(t, err)
	assert.Equal(t, "1", val)

	mr.Set("sync:status", "processing")
	mr.Set("sync:current", "10")
	mr.Set("sync:total", "100")
	mr.Set("sync:job_id", "redis-job")

	status, err := GetCurrentSyncStatus()
	assert.NoError(t, err)
	assert.Equal(t, "processing", status["status"])
	assert.Equal(t, 10, status["current"])
	assert.Equal(t, 100, status["total"])
	assert.Equal(t, "redis-job", status["job_id"])
}

func TestSyncDBFallback_Integration(t *testing.T) {
	db, mr := setupSyncDB(t)
	defer db.Close()
	defer mr.Close()

	_, err := db.Exec(`
		INSERT INTO sync_history (job_id, status, total_contests, successful_contests, started_at) 
		VALUES ('db-job', 'processing', 20, 5, datetime('now', 'localtime'))
	`)
	require.NoError(t, err)

	status, err := GetCurrentSyncStatus()
	assert.NoError(t, err)
	assert.Equal(t, "processing", status["status"])
	assert.Equal(t, 5, status["current"])
	assert.Equal(t, 20, status["total"])
	assert.Equal(t, "db-job", status["job_id"])
}
