package database

import (
	"context"
	"log"

	"leaderboard/src/configs"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(cfg *configs.Config) {
	if cfg.RedisURL == "" {
		log.Println("WARNING: REDIS_URL is not set.")
		return
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse REDIS_URL: %v", err)
	}

	client := redis.NewClient(opt)

	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	RedisClient = client
	log.Println("Successfully connected to Redis!")
}
