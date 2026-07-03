package database

import (
	"context"
	"fmt"
	"log"

	"leaderboard/src/configs"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(cfg *configs.Config) error {
	if cfg.RedisURL == "" {
		log.Println("WARNING: REDIS_URL is not set.")
		return nil
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	RedisClient = client
	log.Println("Successfully connected to Redis!")
	return nil
}
