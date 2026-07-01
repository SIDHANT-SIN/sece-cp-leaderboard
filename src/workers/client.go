package workers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"leaderboard/src/database"

	"github.com/hibiken/asynq"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

var client *asynq.Client
var redisConnOpt asynq.RedisConnOpt

//  parses a Redis URL 
func ParseRedisOpt(redisURL string) (asynq.RedisConnOpt, error) {
	return asynq.ParseRedisURI(redisURL)
}

//  initializes the asynq client and stores connection options
func InitClient(redisOpt asynq.RedisConnOpt) {
	client = asynq.NewClient(redisOpt)
	redisConnOpt = redisOpt
}

//  returns the singleton asynq client for advanced usage
func GetClient() *asynq.Client {
	return client
}

//  enqueues a task to the critical priority queue
func EnqueueCritical(task *asynq.Task) (string, error) {
	if client == nil {
		return "", fmt.Errorf("asynq client not initialized")
	}
	info, err := client.Enqueue(task, asynq.Queue(QueueCritical), asynq.MaxRetry(0))
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// enqueues a task to the default priority queue
func EnqueueDefault(task *asynq.Task) (string, error) {
	if client == nil {
		return "", fmt.Errorf("asynq client not initialized")
	}
	info, err := client.Enqueue(task, asynq.Queue(QueueDefault), asynq.MaxRetry(0))
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// enqueues a task to the low priority queue
func EnqueueLow(task *asynq.Task) (string, error) {
	if client == nil {
		return "", fmt.Errorf("asynq client not initialized")
	}
	info, err := client.Enqueue(task, asynq.Queue(QueueLow), asynq.MaxRetry(0))
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// terminates a running task or deletes a pending task
func CancelTask(taskID string) error {
	if redisConnOpt == nil {
		return fmt.Errorf("redis connection options not initialized")
	}
	inspector := asynq.NewInspector(redisConnOpt)
	
	err := inspector.CancelProcessing(taskID)
	if err == nil {
		return nil
	}

	for _, q := range []string{QueueCritical, QueueDefault, QueueLow} {
		if delErr := inspector.DeleteTask(q, taskID); delErr == nil {
			return nil
		}
	}
	return err
}


type JobState struct {
	JobID       string    `json:"job_id"`
	TaskID      string    `json:"task_id"`
	Status      string    `json:"status"` 
	Total       int       `json:"total"`
	Current     int       `json:"current"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt string    `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// creates a unique job ID for tracking
func GenerateJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("job_%x", b)
}

//  fetches the current job state from Redis
func GetJobState(jobID string) (*JobState, error) {
	if database.RedisClient == nil {
		return nil, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	val, err := database.RedisClient.Get(ctx, "sync:job_state:"+jobID).Result()
	if err != nil {
		return nil, err
	}
	var state JobState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

//  saves the job state to Redis
func SetJobState(jobID string, state *JobState, ttl time.Duration) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ctx := context.Background()
	
	err = database.RedisClient.Set(ctx, "sync:job_state:"+jobID, string(data), ttl).Err()
	if err != nil {
		return err
	}

	database.RedisClient.Set(ctx, "sync:job_id", jobID, ttl)
	database.RedisClient.Set(ctx, "sync:status", state.Status, ttl)
	database.RedisClient.Set(ctx, "sync:current", state.Current, ttl)
	database.RedisClient.Set(ctx, "sync:total", state.Total, ttl)

	return nil
}

// locks the global active job slot using an atomic single-step verification
func AcquireActiveJobLock(jobID string, ttl time.Duration) (bool, error) {
	if database.RedisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.SetNX(ctx, "sync:active_job_id", jobID, ttl).Result()
}

// releases the lock safely only if it belongs to the executing job
func ReleaseActiveJobLock(jobID string) (bool, error) {
	if database.RedisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}

	ctx := context.Background()

	const releaseLockScript = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := database.RedisClient.Eval(
		ctx,
		releaseLockScript,
		[]string{"sync:active_job_id"},
		jobID,
	).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}
// gets the current active job ID holding the execution lock
func GetActiveJobID() (string, error) {
	if database.RedisClient == nil {
		return "", fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	val, err := database.RedisClient.Get(ctx, "sync:active_job_id").Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// appends a  failing contest to the job's failure list in Redis
func AppendFailedContest(jobID, contestDetails string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.RPush(ctx, "sync:failed_contests:"+jobID, contestDetails).Err()
}

//  retrieves all failed contests associated with the job
func GetFailedContests(jobID string) ([]string, error) {
	if database.RedisClient == nil {
		return nil, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.LRange(ctx, "sync:failed_contests:"+jobID, 0, -1).Result()
}

//  deletes the temporary failed contests list from Redis
func ClearFailedContests(jobID string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.Del(ctx, "sync:failed_contests:"+jobID).Err()
}