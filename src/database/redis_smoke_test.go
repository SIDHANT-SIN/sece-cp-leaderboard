//go:build smoke
// +build smoke

package database

import (
	"context"
	"testing"

	"leaderboard/src/configs"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestConnectRedis_EmptyURL(t *testing.T) {
	RedisClient = nil

	cfg := &configs.Config{
		RedisURL: "",
	}

	err := ConnectRedis(cfg)
	require.NoError(t, err)
	require.Nil(t, RedisClient)
}

func TestConnectRedis_SuccessSmoke(t *testing.T) {
	RedisClient = nil

	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	cfg := &configs.Config{
		RedisURL: "redis://" + s.Addr(),
	}

	err = ConnectRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, RedisClient)

	ctx := context.Background()
	err = RedisClient.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestConnectRedis_ParseURLError(t *testing.T) {
	RedisClient = nil

	cfg := &configs.Config{
		RedisURL: "bad-scheme://localhost:6379",
	}

	err := ConnectRedis(cfg)
	require.Error(t, err)
	require.Nil(t, RedisClient)
}

func TestConnectRedis_PingError(t *testing.T) {
	RedisClient = nil

	cfg := &configs.Config{
		RedisURL: "redis://localhost:54321",
	}

	err := ConnectRedis(cfg)
	require.Error(t, err)
	require.Nil(t, RedisClient)
}
