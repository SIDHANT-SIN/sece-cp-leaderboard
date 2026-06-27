package workers

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// StartServer initializes and starts the asynq worker server.
// Concurrency is set to 1 so only one CF API call happens at a time.
// StrictPriority ensures critical tasks are always processed before default/low.
func StartServer(redisOpt asynq.RedisConnOpt) {
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 1, // one CF API call at a time — prevents rate limiting
		Queues: map[string]int{
			QueueCritical: 6, // highest — ratingChanges, addContest, fetchContests
			QueueDefault:  3, // medium  — user.info (manual refresh)
			QueueLow:      1, // lowest  — system.status, scheduled tasks
		},
		StrictPriority: true, // always drain higher priority queue first
	})

	mux := asynq.NewServeMux()

	// Register all CF API task handlers
	mux.HandleFunc(TypeCFRatingChanges, HandleCFRatingChanges)
	mux.HandleFunc(TypeCFRefreshRating, HandleCFRefreshRating)
	mux.HandleFunc(TypeCFCheckStatus, HandleCFCheckStatus)
	
	mux.HandleFunc(TypeCFAddContest, HandleCFAddContest)
	
	mux.HandleFunc(TypeCFBatchRefresh, HandleCFBatchRefresh)

	fmt.Println("[asynq] Starting worker server (concurrency=1, strict priority)")

	if err := srv.Start(mux); err != nil {
		log.Fatalf("[asynq] Failed to start server: %v", err)
	}
}
