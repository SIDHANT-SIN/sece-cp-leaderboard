package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
	"log"

	"leaderboard/src/database"
)

const LeaderboardCacheKey = "leaderboard:cache"

//  set the redis value of leaderboard

func SetLeaderboardCache(users, contests []map[string]interface{}, results map[int]map[int]map[string]interface{}, userTotals map[int]int) error {
	if database.RedisClient == nil {
		return nil;
	}

	strResults := make(map[string]map[string]map[string]interface{})
	for uID, contestMap := range results {
		strContestMap := make(map[string]map[string]interface{})
		for cID, res := range contestMap {
			strContestMap[strconv.Itoa(cID)] = res
		}
		strResults[strconv.Itoa(uID)] = strContestMap
	}

	strUserTotals := make(map[string]int)
	for uID, total := range userTotals {
		strUserTotals[strconv.Itoa(uID)] = total
	}

	payload := map[string]interface{}{
		"users":       users,
		"contests":    contests,
		"results":     strResults,
		"user_totals": strUserTotals,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return database.RedisClient.Set(ctx, LeaderboardCacheKey, string(jsonData), 7*24*time.Hour).Err()
}

//  fetches and parses the cached leaderboard data from Redis
func GetLeaderboardCache() (
	users []map[string]interface{},
	contests []map[string]interface{},
	results map[int]map[int]map[string]interface{},
	userTotals map[int]int,
	err error,
) {
	if database.RedisClient == nil {
		return nil, nil, nil, nil, nil
	}

	ctx := context.Background()

st := time.Now()

	val, err := database.RedisClient.Get(ctx, LeaderboardCacheKey).Result()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	
log.Printf("Redis GET took %v", time.Since(st))

	var rawData struct {
		Users      []map[string]interface{}                     `json:"users"`
		Contests   []map[string]interface{}                     `json:"contests"`
		Results    map[string]map[string]map[string]interface{} `json:"results"`
		UserTotals map[string]int                               `json:"user_totals"`
	}

	if err := json.Unmarshal([]byte(val), &rawData); err != nil {
		return nil, nil, nil, nil, err
	}

	for _, u := range rawData.Users {
		for k, v := range u {
			if f, ok := v.(float64); ok {
				u[k] = int(f)
			}
		}
	}
	for _, c := range rawData.Contests {
		for k, v := range c {
			if f, ok := v.(float64); ok {
				c[k] = int(f)
			}
		}
	}

	intResults := make(map[int]map[int]map[string]interface{})
	for uIDStr, contestMap := range rawData.Results {
		uID, err := strconv.Atoi(uIDStr)
		if err != nil {
			continue
		}
		intContestMap := make(map[int]map[string]interface{})
		for cIDStr, res := range contestMap {
			cID, err := strconv.Atoi(cIDStr)
			if err != nil {
				continue
			}
			for k, v := range res {
				if f, ok := v.(float64); ok {
					res[k] = int(f)
				}
			}
			intContestMap[cID] = res
		}
		intResults[uID] = intContestMap
	}

	intUserTotals := make(map[int]int)
	for uIDStr, total := range rawData.UserTotals {
		uID, err := strconv.Atoi(uIDStr)
		if err != nil {
			continue
		}
		intUserTotals[uID] = total
	}

	return rawData.Users, rawData.Contests, intResults, intUserTotals, nil
}
