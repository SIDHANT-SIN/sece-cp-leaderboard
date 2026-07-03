//go:build integration
// +build integration
package workers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPurgeAsynqMetadata_Integration(t *testing.T) {
	ctx := context.Background()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() {
		_ = rdb.Close()
	}()

	_ = rdb.Set(ctx, "asynq:scheduler_history:1", "data", 0).Err()
	_ = rdb.Set(ctx, "asynq:scheduler_history:2", "data", 0).Err()
	_ = rdb.Set(ctx, "asynq:servers", "active-server", 0).Err()
	_ = rdb.Set(ctx, "asynq:workers", "active-worker", 0).Err()
	_ = rdb.Set(ctx, "asynq:{default}:processed:2026-07-03", "10", 0).Err() // Set with NO expiration
	_ = rdb.Set(ctx, "safe_user_data:123", "keep-me", 0).Err()              // Should remain untouched

	err = PurgeAsynqMetadata(rdb)
	if err != nil {
		t.Fatalf("PurgeAsynqMetadata failed unexpectedly: %v", err)
	}

	exists, _ := rdb.Exists(ctx, "asynq:scheduler_history:1", "asynq:scheduler_history:2").Result()
	if exists != 0 {
		t.Errorf("Expected scheduler history keys to be vaporized, but they still exist")
	}

	exists, _ = rdb.Exists(ctx, "asynq:servers", "asynq:workers").Result()
	if exists != 0 {
		t.Errorf("Expected active structural tracking keys to be reset, but they still exist")
	}

	ttl, err := rdb.TTL(ctx, "asynq:{default}:processed:2026-07-03").Result()
	if err != nil {
		t.Fatalf("Failed to get TTL for stats key: %v", err)
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		t.Errorf("Expected stats key to have a 24-hour expiration cap, but got TTL: %v", ttl)
	}

	nonAsynqExists, _ := rdb.Exists(ctx, "safe_user_data:123").Result()
	if nonAsynqExists != 1 {
		t.Errorf("System swept too aggressively! Unrelated application data was modified or deleted.")
	}
}
