//go:build integration
// +build integration

package workers

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
)

func TestStartScheduler_Integration(t *testing.T) {
	// Spin up our isolated, in-memory Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start in-memory miniredis: %v", err)
	}
	defer mr.Close()

	redisOpt := asynq.RedisClientOpt{
		Addr: mr.Addr(),
	}

	// Execute the scheduler registration and boot sequence
	err = StartScheduler(redisOpt)
	if err != nil {
		t.Fatalf("Expected StartScheduler to register and boot successfully, but got error: %v", err)
	}

}
