package main

import (
	"log"

	"leaderboard/src/configs"
	"leaderboard/src/database"
	"leaderboard/src/routes"
	"leaderboard/src/workers"
)

func main() {
	cfg := configs.LoadConfig()

	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Database connection setup failed: %v", err)
	}

	if err := database.CreateTables(); err != nil {
		log.Fatalf("Database schema setup failed: %v", err)
	}

	if err := database.ConnectRedis(cfg); err != nil {
		log.Fatalf("Redis connection setup failed: %v", err)
	}

	if cfg.RedisURL != "" {
		redisOpt, err := workers.ParseRedisOpt(cfg.RedisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL for Asynq: %v", err)
		}

		if err := workers.PurgeAsynqMetadata(database.RedisClient); err != nil {
			log.Fatalf("Redis optimization sweep failed: %v", err)
		}
		workers.InitClient(redisOpt)

		// Kicks off the worker server loops in the background natively
		if err := workers.StartServer(redisOpt); err != nil {
			log.Fatalf("[asynq] Worker server failed to start: %v", err)
		}

		// Kicks off the cron scheduler loops in the background natively
		if err := workers.StartScheduler(redisOpt); err != nil {
			log.Fatalf("[asynq] Scheduler failed to start: %v", err)
		}
	} else {
		log.Println("WARNING: REDIS_URL not set. Asynq worker server not started.")
	}

	// This is your blocking call that keeps the whole application running
	r := routes.SetupRoutes(cfg)
	port := cfg.Port

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
