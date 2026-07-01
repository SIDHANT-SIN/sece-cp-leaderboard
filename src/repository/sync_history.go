

package repository

import (
	"context"
	"database/sql"
	"leaderboard/src/database"
	"strconv"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

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

func GetCurrentSyncStatus() (map[string]interface{}, error) {
	ctx := context.Background()

	status, _ := database.RedisClient.Get(ctx, "sync:status").Result()
	currentStr, _ := database.RedisClient.Get(ctx, "sync:current").Result()
	totalStr, _ := database.RedisClient.Get(ctx, "sync:total").Result()
	jobID, _ := database.RedisClient.Get(ctx, "sync:job_id").Result()

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

	var dbJobID, dbStatus string
	var dbTotal, dbSuccessful int
	
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

	return map[string]interface{}{
		"job_id":  dbJobID,
		"status":  dbStatus,
		"current": dbSuccessful, 
		"total":   dbTotal,
	}, nil
}

func SetSyncCancelSignal() error {
	ctx := context.Background()
	return database.RedisClient.Set(ctx, "sync:cancel_signal", "1", 15*time.Minute).Err()
}