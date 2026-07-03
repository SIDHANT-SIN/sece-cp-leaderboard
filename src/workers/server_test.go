//go:build integration
// +build integration

//
package workers

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
)

func TestStartServer_Integration(t *testing.T) {

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start in-memory miniredis: %v", err)
	}
	defer mr.Close()

	redisOpt := asynq.RedisClientOpt{
		Addr: mr.Addr(),
	}

	err = StartServer(redisOpt)
	if err != nil {
		t.Fatalf("Expected StartServer to boot successfully, but got error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	client := asynq.NewClient(redisOpt)
	defer client.Close()

	err = client.Ping()
	if err != nil {
		t.Errorf("Worker server started but Redis connection pool is unhealthy: %v", err)
	}
}
