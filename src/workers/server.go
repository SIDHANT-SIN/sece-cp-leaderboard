package workers

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

func StartServer(redisOpt asynq.RedisConnOpt) {
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 1, 
		Queues: map[string]int{
			QueueCritical: 6, 
			QueueDefault:  3, 
			QueueLow:      1, 
		},
		StrictPriority: true, 
	})

	mux := asynq.NewServeMux()

	mux.HandleFunc(TypeCFRatingChanges, HandleCFRatingChanges)
	mux.HandleFunc(TypeCFRefreshRating, HandleCFRefreshRating)
	
	mux.HandleFunc(TypeCFAddContest, HandleCFAddContest)
	
	mux.HandleFunc(TypeCFBatchRefresh, HandleCFBatchRefresh)

	fmt.Println("[asynq] Starting worker server ")

	if err := srv.Start(mux); err != nil {
		log.Fatalf("[asynq] Failed to start server: %v", err)
	}
}
