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

// Queue priority names matched to your architecture
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

var client *asynq.Client
var redisConnOpt asynq.RedisConnOpt

// ParseRedisOpt parses a Redis URL (redis:// or rediss://) into asynq connection options
func ParseRedisOpt(redisURL string) (asynq.RedisConnOpt, error) {
	return asynq.ParseRedisURI(redisURL)
}

// InitClient initializes the asynq client and stores connection options
func InitClient(redisOpt asynq.RedisConnOpt) {
	client = asynq.NewClient(redisOpt)
	redisConnOpt = redisOpt
}

// GetClient returns the singleton asynq client for advanced usage
func GetClient() *asynq.Client {
	return client
}

// --- Enqueue helpers (so handles don't import asynq directly) ---
// They return the unique task ID assigned by Asynq to enable explicit cancellation tracking.

// EnqueueCritical enqueues a task to the critical (highest) priority queue
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

// EnqueueDefault enqueues a task to the default (medium) priority queue
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

// EnqueueLow enqueues a task to the low priority queue
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

// CancelTask terminates a running task or deletes a pending task
func CancelTask(taskID string) error {
	if redisConnOpt == nil {
		return fmt.Errorf("redis connection options not initialized")
	}
	inspector := asynq.NewInspector(redisConnOpt)
	
	// Try to cancel processing first if the task is actively running
	err := inspector.CancelProcessing(taskID)
	if err == nil {
		return nil
	}
	
	// If it was not currently running, iterate through queues to purge it from pending states
	for _, q := range []string{QueueCritical, QueueDefault, QueueLow} {
		if delErr := inspector.DeleteTask(q, taskID); delErr == nil {
			return nil
		}
	}
	return err
}

// --- Job State & Atomic Locking Logic ---

type JobState struct {
	JobID       string    `json:"job_id"`
	TaskID      string    `json:"task_id"`
	Status      string    `json:"status"` // "processing", "completed", "failed", "cancelled"
	Total       int       `json:"total"`
	Current     int       `json:"current"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt string    `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// GenerateJobID creates a unique job ID for tracking
func GenerateJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("job_%x", b)
}

// GetJobState fetches the current job state from Redis
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

// SetJobState saves the job state to Redis with a sliding TTL and exposes flat progress metrics
func SetJobState(jobID string, state *JobState, ttl time.Duration) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ctx := context.Background()
	
	// Write full JSON tracking blob
	err = database.RedisClient.Set(ctx, "sync:job_state:"+jobID, string(data), ttl).Err()
	if err != nil {
		return err
	}

	// UPDATED: Sync flat keys to ensure repository.GetCurrentSyncStatus reads real-time changes
	database.RedisClient.Set(ctx, "sync:job_id", jobID, ttl)
	database.RedisClient.Set(ctx, "sync:status", state.Status, ttl)
	database.RedisClient.Set(ctx, "sync:current", state.Current, ttl)
	database.RedisClient.Set(ctx, "sync:total", state.Total, ttl)

	return nil
}

// AcquireActiveJobLock locks the global active job slot using an atomic single-step verification
func AcquireActiveJobLock(jobID string, ttl time.Duration) (bool, error) {
	if database.RedisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.SetNX(ctx, "sync:active_job_id", jobID, ttl).Result()
}

// ReleaseActiveJobLock releases the lock safely only if it belongs to the executing job
func ReleaseActiveJobLock(jobID string) (bool, error) {
	if database.RedisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	
	currentID, err := database.RedisClient.Get(ctx, "sync:active_job_id").Result()
	if err != nil {
		return false, err
	}
	
	if currentID == jobID {
		err = database.RedisClient.Del(ctx, "sync:active_job_id").Err()
		return err == nil, err
	}
	
	return false, nil
}

// GetActiveJobID gets the current active job ID holding the execution lock
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

// AppendFailedContest appends a flaky or failing contest to the job's failure list in Redis
func AppendFailedContest(jobID, contestDetails string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.RPush(ctx, "sync:failed_contests:"+jobID, contestDetails).Err()
}

// GetFailedContests retrieves all failed contests associated with the job
func GetFailedContests(jobID string) ([]string, error) {
	if database.RedisClient == nil {
		return nil, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.LRange(ctx, "sync:failed_contests:"+jobID, 0, -1).Result()
}

// ClearFailedContests deletes the temporary failed contests list from Redis
func ClearFailedContests(jobID string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	return database.RedisClient.Del(ctx, "sync:failed_contests:"+jobID).Err()
}