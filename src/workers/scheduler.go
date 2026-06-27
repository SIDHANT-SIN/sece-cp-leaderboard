package workers

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// StartScheduler initializes and starts the periodic task cron engine.
// It maps system-driven background iterations cleanly to the low priority lane.
func StartScheduler(redisOpt asynq.RedisConnOpt) {
	scheduler := asynq.NewScheduler(redisOpt, nil)

	// We pass an explicit "cron_" identifier instead of an empty string.
	// This ensures the worker updates a valid Redis state key space and
	// can safely differentiate background cron execution from live admin actions.
	task, err := NewCFRefreshRatingTask("cron_refresh_rating")
	if err != nil {
		log.Fatalf("[scheduler] Failed to instantiate periodic rating task structure: %v", err)
	}

	// Schedule the task to run every 4 hours at the lowest priority lane (low)
	entryID, err := scheduler.Register(
		"0 */4 * * *", // Cron expression: executes at minute 0 of every 4th hour
		task,
		asynq.Queue(QueueLow),
		asynq.MaxRetry(0), // Do not automatically retry on failure; wait for the next 4-hour cycle
	)
	if err != nil {
		log.Fatalf("[scheduler] Failed to register periodic task sequence with Redis broker: %v", err)
	}

	fmt.Printf("[scheduler] Registered automated background sync every 4 hours (entry_id=%s, target_queue=%s)\n", entryID, QueueLow)

	// Start spawns the scheduler engine loop non-blockingly
	if err := scheduler.Start(); err != nil {
		log.Fatalf("[scheduler] Fatal error: failed to start periodic scheduler engine: %v", err)
	}
}