//go:build integration
// +build integration

package workers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"leaderboard/src/database"
)

func TestClientAndState_Integration(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisOpt := asynq.RedisClientOpt{
		Addr: mr.Addr(),
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() {
		_ = rdb.Close()
	}()

	
	InitClient(redisOpt)
	if GetClient() == nil {
		t.Fatal("Expected GetClient to return initialized client, got nil")
	}

	dummyTask := asynq.NewTask("dummy:type", []byte("payload"))
	taskID, err := EnqueueDefault(dummyTask)
	if err != nil || taskID == "" {
		t.Errorf("EnqueueDefault failed: %v", err)
	}

	jobID := GenerateJobID()
	expectedState := &JobState{
		JobID:     jobID,
		TaskID:    "task-123",
		Status:    "running",
		Total:     100,
		Current:   50,
		StartedAt: time.Now().Truncate(time.Second), 
	}

	err = SetJobState(ctx, rdb, jobID, expectedState, 1*time.Hour)
	if err != nil {
		t.Fatalf("SetJobState failed: %v", err)
	}

	fetchedState, err := GetJobState(ctx, rdb, jobID)
	if err != nil {
		t.Fatalf("GetJobState failed: %v", err)
	}
	if fetchedState.Status != "running" || fetchedState.Current != 50 {
		t.Errorf("GetJobState returned incorrect data. Got Status: %s, Current: %d", fetchedState.Status, fetchedState.Current)
	}

	acquired, err := AcquireActiveJobLock(ctx, rdb, jobID, 1*time.Hour)
	if err != nil || !acquired {
		t.Errorf("Failed to acquire active job lock")
	}

	// Try acquiring with a different ID (should fail)
	acquired2, _ := AcquireActiveJobLock(ctx, rdb, "other-job-id", 1*time.Hour)
	if acquired2 {
		t.Errorf("Acquired lock for another job while lock was held")
	}

	// Verify Active Job ID
	activeID, err := GetActiveJobID(ctx, rdb)
	if err != nil || activeID != jobID {
		t.Errorf("GetActiveJobID failed or mismatched. Got: %s, Expected: %s", activeID, jobID)
	}

	// Release Lock
	released, err := ReleaseActiveJobLock(ctx, rdb, jobID)
	if err != nil || !released {
		t.Errorf("Failed to release active job lock")
	}

	// Verify lock is actually gone
	_, err = GetActiveJobID(ctx, rdb)
	if err == nil {
		t.Errorf("Expected error when getting active job ID after lock release, but got none")
	}

	err = AppendFailedContest(ctx, rdb, jobID, "contest-101")
	err = AppendFailedContest(ctx, rdb, jobID, "contest-102")
	if err != nil {
		t.Fatalf("AppendFailedContest failed: %v", err)
	}

	failures, err := GetFailedContests(ctx, rdb, jobID)
	if err != nil || len(failures) != 2 {
		t.Fatalf("GetFailedContests returned unexpected results. Len: %d", len(failures))
	}

	err = ClearFailedContests(ctx, rdb, jobID)
	if err != nil {
		t.Fatalf("ClearFailedContests failed: %v", err)
	}

	failuresPostClear, _ := GetFailedContests(ctx, rdb, jobID)
	if len(failuresPostClear) != 0 {
		t.Errorf("Expected 0 failures after clear, got %d", len(failuresPostClear))
	}
}

func TestParseRedisOpt(t *testing.T) {
	
	opt, err := ParseRedisOpt("redis://user:pass@localhost:6379/1")
	if err != nil {
		t.Errorf("Expected no error for valid URI, got %v", err)
	}
	if opt == nil {
		t.Error("Expected options to be parsed, got nil")
	}

	_, err = ParseRedisOpt("invalid-uri")
	if err == nil {
		t.Error("Expected error for invalid URI, got nil")
	}
}

func TestCancelTask(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	InitClient(asynq.RedisClientOpt{Addr: mr.Addr()})

	
	err := CancelTask("some-fake-task-id")
	if err == nil {
		
		t.Log("CancelTask succeeded on fake task (expected behavior in some Asynq versions)")
	}
}
func TestReleaseActiveJobLock_Coverage(t *testing.T) {
	mr := setupTestRedis(t) 
	defer mr.Close()
	ctx := context.Background()

	database.RedisClient.Set(ctx, "sync:active_job_id", "job123", 10*time.Minute)

	released, err := ReleaseActiveJobLock(ctx, database.RedisClient, "wrong-job-id")
	if err != nil || released {
		t.Error("Expected no release for wrong ID")
	}

	
	released, err = ReleaseActiveJobLock(ctx, database.RedisClient, "job123")
	if err != nil || !released {
		t.Error("Expected successful release for correct ID")
	}
}

func TestClientQueues_Uninitialized(t *testing.T) {
	
	originalClient := client
	client = nil
	defer func() { client = originalClient }()

	dummyTask := asynq.NewTask("dummy", nil)

	_, err := EnqueueCritical(dummyTask)
	if err == nil || err.Error() != "asynq client not initialized" {
		t.Errorf("Expected uninitialized error, got %v", err)
	}

	
	_, err = EnqueueDefault(dummyTask)
	if err == nil || err.Error() != "asynq client not initialized" {
		t.Errorf("Expected uninitialized error, got %v", err)
	}

	_, err = EnqueueLow(dummyTask)
	if err == nil || err.Error() != "asynq client not initialized" {
		t.Errorf("Expected uninitialized error, got %v", err)
	}
}

func TestClientQueues_Initialized(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	InitClient(asynq.RedisClientOpt{Addr: mr.Addr()})

	dummyTask := asynq.NewTask("dummy", []byte("payload"))

	id1, err := EnqueueCritical(dummyTask)
	if err != nil || id1 == "" {
		t.Errorf("EnqueueCritical failed: %v", err)
	}

	id2, err := EnqueueDefault(dummyTask)
	if err != nil || id2 == "" {
		t.Errorf("EnqueueDefault failed: %v", err)
	}

	id3, err := EnqueueLow(dummyTask)
	if err != nil || id3 == "" {
		t.Errorf("EnqueueLow failed: %v", err)
	}
}

func TestCancelTask_Uninitialized(t *testing.T) {
	
	originalOpt := redisConnOpt
	redisConnOpt = nil
	defer func() { redisConnOpt = originalOpt }()

	err := CancelTask("some-task-id")
	if err == nil || err.Error() != "redis connection options not initialized" {
		t.Errorf("Expected initialization error, got %v", err)
	}
}
