package workers

import (
	"context"
	"fmt"
	"time"

	//"github.com/redis/go-redis/v9"
	"leaderboard/src/database"
)

// PurgeAsynqMetadata clears or forces low TTLs on internal Asynq tracking keys.
func PurgeAsynqMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("[maintenance] Starting Redis optimization sweep...")

	// 1. 💥 FIXED: Scan and completely NUKE all scheduler history keys (handles UUID suffixes)
	var historyCursor uint64
	for {
		var keys []string
		var err error
		// This wildcard pattern grabs 'asynq:scheduler_history' AND 'asynq:scheduler_history:xxxx-xxxx'
		keys, historyCursor, err = database.RedisClient.Scan(ctx, historyCursor, "asynq:scheduler_history*", 100).Result()
		if err != nil {
			fmt.Printf("[maintenance] Error scanning scheduler history keys: %v\n", err)
			break
		}

		if len(keys) > 0 {
			err = database.RedisClient.Del(ctx, keys...).Err()
			if err != nil {
				fmt.Printf("[maintenance] Warning: Failed to delete scheduler history batch: %v\n", err)
			} else {
				fmt.Printf("[maintenance] Vaporized %d scheduler history tracking keys.\n", len(keys))
			}
		}

		if historyCursor == 0 {
			break
		}
	}

	// 2. Clear out old crashed server/worker/scheduler instance registration lists
	trackingKeys := []string{"asynq:servers", "asynq:workers", "asynq:schedulers"}
	for _, key := range trackingKeys {
		_ = database.RedisClient.Del(ctx, key)
	}
	fmt.Println("[maintenance] Reset active structural instances (servers, workers, schedulers)")

	// 3. Scan and enforce a tight 1-Day TTL on daily stats counters 
	var statsCursor uint64
	for {
		var keys []string
		var err error
		keys, statsCursor, err = database.RedisClient.Scan(ctx, statsCursor, "asynq:{*}:processed:*", 100).Result()
		if err != nil {
			fmt.Printf("[maintenance] Error scanning historical stats keys: %v\n", err)
			break
		}

		for _, key := range keys {
			// Overriding default 90-day hardcoded Asynq TTL down to 24 hours
			database.RedisClient.Expire(ctx, key, 24*time.Hour)
		}

		if statsCursor == 0 {
			break
		}
	}
	fmt.Println("[maintenance] Enforced a strict 24-hour expiration cap on historical processed date counters.")
	fmt.Println("[maintenance] Redis sweep complete. Database state optimized.")
}