package workers

import (
	"fmt"

	"github.com/hibiken/asynq"
)

func StartScheduler(redisOpt asynq.RedisConnOpt) error {
	scheduler := asynq.NewScheduler(redisOpt, nil)

	task, err := NewCFRefreshRatingTask("cron_refresh_rating")
	if err != nil {
		return fmt.Errorf("failed to instantiate periodic rating task structure: %w", err)
	}

	entryID, err := scheduler.Register(
		"0 */4 * * *",
		task,
		asynq.Queue(QueueLow),
		asynq.MaxRetry(0),
	)
	if err != nil {
		return fmt.Errorf("failed to register periodic task sequence with Redis broker: %w", err)
	}

	fmt.Printf("[scheduler] Registered automated background sync every 4 hours (entry_id=%s, target_queue=%s)\n", entryID, QueueLow)

	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start periodic scheduler engine: %w", err)
	}

	return nil
}
