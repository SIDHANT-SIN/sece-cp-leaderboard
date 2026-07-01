package workers

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

func StartScheduler(redisOpt asynq.RedisConnOpt) {
	scheduler := asynq.NewScheduler(redisOpt, nil)

	
	task, err := NewCFRefreshRatingTask("cron_refresh_rating")
	if err != nil {
		log.Fatalf("[scheduler] Failed to instantiate periodic rating task structure: %v", err)
	}

	entryID, err := scheduler.Register(
		"0 */4 * * *", 
		task,
		asynq.Queue(QueueLow),
		asynq.MaxRetry(0), 
	)
	if err != nil {
		log.Fatalf("[scheduler] Failed to register periodic task sequence with Redis broker: %v", err)
	}

	fmt.Printf("[scheduler] Registered automated background sync every 4 hours (entry_id=%s, target_queue=%s)\n", entryID, QueueLow)

	if err := scheduler.Start(); err != nil {
		log.Fatalf("[scheduler] Fatal error: failed to start periodic scheduler engine: %v", err)
	}
}