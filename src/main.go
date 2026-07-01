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

	database.Connect(cfg)

	database.CreateTables()

	database.ConnectRedis(cfg)

	if cfg.RedisURL != "" {
		redisOpt, err := workers.ParseRedisOpt(cfg.RedisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL for Asynq: %v", err)
		}

		workers.PurgeAsynqMetadata()

		workers.InitClient(redisOpt)
		go workers.StartServer(redisOpt)
		go workers.StartScheduler(redisOpt)
	} else {
		log.Println("WARNING: REDIS_URL not set. Asynq worker server not started.")
	}

	r := routes.SetupRoutes(cfg)

	port := cfg.Port

	if err := r.Run(":" + port); 
	err != nil {
		log.Fatal(err)
	}
}