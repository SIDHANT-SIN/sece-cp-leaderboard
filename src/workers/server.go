package workers

import (
	"fmt"
	"github.com/hibiken/asynq"
)

func StartServer(redisOpt asynq.RedisConnOpt) error {
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

	fmt.Println("[asynq] Starting worker server")

	return srv.Start(mux)
}
