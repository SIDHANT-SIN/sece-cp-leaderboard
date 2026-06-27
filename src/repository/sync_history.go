

package repository

import (
	"context"
	"database/sql"
	//"fmt"
	"leaderboard/src/database"
	"strconv"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// func GetRecentSyncHistory(limit int) (*sql.Rows, error) {
// 	return database.DB.Query(`
// 		SELECT job_id, status, successful_contests, total_contests, 
// 		       failed_contest_ids, started_at, COALESCE(completed_at, '-')
// 		FROM sync_history
// 		ORDER BY started_at DESC
// 		LIMIT ?
// 	`, limit)
// }

func GetRecentSyncHistory(limit int) (*sql.Rows, error) {
    return database.DB.Query(`
        SELECT 
            job_id, 
            status, 
            COALESCE(successful_contests, 0), 
            total_contests, 
            COALESCE(failed_contest_ids, ''), 
            started_at, 
            COALESCE(completed_at, '-')
        FROM sync_history
        ORDER BY completed_at DESC
        LIMIT ?
    `, limit)
}

func CreateSyncLog(jobID string, totalContests int) error {
	_, err := database.DB.Exec(`
		INSERT INTO sync_history (job_id, status, total_contests, started_at)
		VALUES (?, 'processing', ?, datetime('now', 'localtime'))
	`, jobID, totalContests)
	
	return err
}

func UpdateSyncLog(jobID string, status string, successful int, failedIDs string) error {
	_, err := database.DB.Exec(`
		UPDATE sync_history
		SET status = ?, 
		    successful_contests = ?, 
		    failed_contest_ids = ?, 
		    completed_at = datetime('now', 'localtime')
		WHERE job_id = ?
	`, status, successful, failedIDs, jobID)

	return err
}
// GetCurrentSyncStatus retrieves real-time progress counters from Redis
// and falls back to looking for an active SQL log if Redis is cold.
func GetCurrentSyncStatus() (map[string]interface{}, error) {
	ctx := context.Background()

	// 1. Read all progress metrics from Redis
	status, _ := database.RedisClient.Get(ctx, "sync:status").Result()
	currentStr, _ := database.RedisClient.Get(ctx, "sync:current").Result()
	totalStr, _ := database.RedisClient.Get(ctx, "sync:total").Result()
	jobID, _ := database.RedisClient.Get(ctx, "sync:job_id").Result()

	// 2. CRITICAL FIX: If Redis has active processing signals, trust it!
	if status == "processing" || totalStr != "" {
		current, _ := strconv.Atoi(currentStr)
		total, _ := strconv.Atoi(totalStr)
		return map[string]interface{}{
			"job_id":  jobID,
			"status":  "processing",
			"current": current,
			"total":   total,
		}, nil
	}

	// 3. Fallback: If Redis is cold, check Turso DB for an active processing row
	var dbJobID, dbStatus string
	var dbTotal, dbSuccessful int
	
	// Added successful_contests to the scan path
	err := database.DB.QueryRow(`
		SELECT job_id, status, total_contests, COALESCE(successful_contests, 0)
		FROM sync_history 
		WHERE status = 'processing' AND job_id != 'cron_refresh_rating'
		ORDER BY started_at DESC LIMIT 1
	`).Scan(&dbJobID, &dbStatus, &dbTotal, &dbSuccessful)

	if err == sql.ErrNoRows {
		return map[string]interface{}{
			"status":  "idle",
			"current": 0,
			"total":   0,
			"job_id":  "",
		}, nil
	} else if err != nil {
		return nil, err
	}

	// Return actual live numbers from DB row metrics instead of hardcoded 0
	return map[string]interface{}{
		"job_id":  dbJobID,
		"status":  dbStatus,
		"current": dbSuccessful, 
		"total":   dbTotal,
	}, nil
}


// SetSyncCancelSignal sets an ephemeral abort flag in Redis that the loop worker checks
func SetSyncCancelSignal() error {
	ctx := context.Background()
	// Sets an abort flag valid for 15 minutes to guarantee it drops off automatically
	return database.RedisClient.Set(ctx, "sync:cancel_signal", "1", 15*time.Minute).Err()
}