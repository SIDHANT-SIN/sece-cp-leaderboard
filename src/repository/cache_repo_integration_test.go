//go:build integration
// +build integration

package repository

import (
	"testing"

	"leaderboard/src/database"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCacheRedis(t *testing.T) *miniredis.Miniredis {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	database.RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr
}

func TestLeaderboardCache_Integration(t *testing.T) {
	mr := setupCacheRedis(t)
	defer mr.Close()

	users := []map[string]interface{}{
		{"handle": "tourist", "id": 1},
	}

	contests := []map[string]interface{}{
		{"name": "Round 1", "id": 100},
	}

	results := map[int]map[int]map[string]interface{}{
		1: {
			100: {"rank": 1, "points": 3000},
		},
	}

	userTotals := map[int]int{
		1: 3000,
	}

	err := SetLeaderboardCache(users, contests, results, userTotals)
	assert.NoError(t, err)

	resUsers, resContests, resResults, resTotals, err := GetLeaderboardCache()
	assert.NoError(t, err)

	assert.Len(t, resUsers, 1)
	assert.Equal(t, "tourist", resUsers[0]["handle"])
	assert.Equal(t, 1, resUsers[0]["id"])

	assert.Len(t, resContests, 1)
	assert.Equal(t, "Round 1", resContests[0]["name"])
	assert.Equal(t, 100, resContests[0]["id"])

	assert.Equal(t, 1, resResults[1][100]["rank"])
	assert.Equal(t, 3000, resResults[1][100]["points"])

	assert.Equal(t, 3000, resTotals[1])
}

func TestCacheNilClient_Integration(t *testing.T) {
	database.RedisClient = nil

	err := SetLeaderboardCache(nil, nil, nil, nil)
	assert.NoError(t, err)

	u, c, r, ut, err := GetLeaderboardCache()
	assert.NoError(t, err)
	assert.Nil(t, u)
	assert.Nil(t, c)
	assert.Nil(t, r)
	assert.Nil(t, ut)
}
