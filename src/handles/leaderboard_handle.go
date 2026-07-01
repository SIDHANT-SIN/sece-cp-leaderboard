package handles

import (
	"fmt"
	"net/http"
	"sort"
     
	"leaderboard/src/repository"
	"leaderboard/src/configs"
      
	"github.com/gin-gonic/gin"
)

// renders the current leaderboard 
func ShowLeaderboard(c *gin.Context, cfg *configs.Config) {


	cachedUsers, cachedContests, cachedResults, cachedUserTotals, err := repository.GetLeaderboardCache()
	if err == nil && cachedUsers != nil && len(cachedUsers) > 0 {
		fmt.Printf("cache hit")
		
		c.HTML(http.StatusOK, "leaderboard.tmpl", gin.H{
			"users":      cachedUsers,
			"contests":   cachedContests,
			"results":    cachedResults,
			"userTotals": cachedUserTotals,
			"logo": cfg.Logo,
		})
		return
	}

	
	fmt.Printf("cache miss")
	userRows, err := repository.GetUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer userRows.Close()

	var users []map[string]interface{}
	for userRows.Next() {
		var id int
		var handle, displayName string
		userRows.Scan(&id, &handle, &displayName)
		users = append(users, map[string]interface{}{
			"id":           id,
			"handle":       handle,
			"display_name": displayName,
		})
	}

	contestRows, err := repository.GetContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer contestRows.Close()

	var contests []map[string]interface{}
	for contestRows.Next() {
		var id, cfid, startTime int
		var name string
		contestRows.Scan(&id, &cfid, &name, &startTime)
		contests = append(contests, map[string]interface{}{
			"id":         id,
			"cfid":       cfid,
			"name":       name,
			"start_time": startTime,
		})
	}

	results := make(map[int]map[int]map[string]interface{}) 
	userTotals := make(map[int]int)                         

	rows, err := repository.GetAllResults()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID, contestID, rank, points int
			rows.Scan(&userID, &contestID, &rank, &points)

			contestExists := false
			for _, c := range contests {
				if c["id"].(int) == contestID {
					contestExists = true
					break
				}
			}
			if !contestExists {
				continue
			}

			if results[userID] == nil {
				results[userID] = make(map[int]map[string]interface{})
			}
			results[userID][contestID] = map[string]interface{}{
				"rank":   rank,
				"points": points,
			}
			userTotals[userID] += points
		}
	}

	type userWithTotal struct {
		User  map[string]interface{}
		Total int
	}

	var userList []userWithTotal
	for _, u := range users {
		uid := u["id"].(int)
		total := userTotals[uid]
		userList = append(userList, userWithTotal{User: u, Total: total})
	}

	sort.Slice(userList, func(i, j int) bool {
		return userList[i].Total > userList[j].Total
	})

	rankedUsers := make([]map[string]interface{}, len(userList))
	for i, ut := range userList {
		rankedUsers[i] = ut.User
		rankedUsers[i]["rank"] = i + 1
		rankedUsers[i]["total_points"] = ut.Total
	}

	err = repository.SetLeaderboardCache(rankedUsers, contests, results, userTotals)
	if err != nil {
		fmt.Printf("Warning: failed to save leaderboard cache: %v\n", err)
	}
	
	fmt.Printf("cache saved")
	

	c.HTML(http.StatusOK, "leaderboard.tmpl", gin.H{
		"users":      rankedUsers,
		"contests":   contests,
		"results":    results,
		"userTotals": userTotals,
		"logo" : cfg.Logo,
	})
}

//  renders the past leaderboard filtered by batch year
func ShowPastLeaderboard(c *gin.Context) {
	batch := c.Query("batch")
	if batch == "" {
		batch = "2023" 
	}

	rows, err := repository.GetPastUsersByBatch(batch)
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, cur, mx, by int
		var handle, name, title string
		rows.Scan(&id, &handle, &name, &cur, &mx, &title, &by)
		users = append(users, map[string]interface{}{
			"id":             id,
			"handle":         handle,
			"display_name":   name,
			"current_rating": cur,
			"max_rating":     mx,
			"title":          title,
			"batch":          by,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i]["max_rating"].(int) > users[j]["max_rating"].(int)
	})

	for i := range users {
		users[i]["rank"] = i + 1
	}

	c.HTML(http.StatusOK, "past_leaderboard.tmpl", gin.H{
		"users":         users,
		"selectedBatch": batch,
	})
}

// recalculates the entire leaderboard and stores it in Redis
func rebuildLeaderboardCache() error {
	userRows, err := repository.GetUsers()
	if err != nil {
		return err
	}
	defer userRows.Close()

	var users []map[string]interface{}
	for userRows.Next() {
		var id int
		var handle, displayName string
		userRows.Scan(&id, &handle, &displayName)
		users = append(users, map[string]interface{}{
			"id":           id,
			"handle":       handle,
			"display_name": displayName,
		})
	}

	contestRows, err := repository.GetContests()
	if err != nil {
		return err
	}
	defer contestRows.Close()

	var contests []map[string]interface{}
	for contestRows.Next() {
		var id, cfid, startTime int
		var name string
		contestRows.Scan(&id, &cfid, &name, &startTime)
		contests = append(contests, map[string]interface{}{
			"id":         id,
			"cfid":       cfid,
			"name":       name,
			"start_time": startTime,
		})
	}

	results := make(map[int]map[int]map[string]interface{})
	userTotals := make(map[int]int)

	rows, err := repository.GetAllResults()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID, contestID, rank, points int
			rows.Scan(&userID, &contestID, &rank, &points)

			contestExists := false
			for _, c := range contests {
				if c["id"].(int) == contestID {
					contestExists = true
					break
				}
			}
			if !contestExists {
				continue
			}

			if results[userID] == nil {
				results[userID] = make(map[int]map[string]interface{})
			}
			results[userID][contestID] = map[string]interface{}{
				"rank":   rank,
				"points": points,
			}
			userTotals[userID] += points
		}
	}

	type userWithTotal struct {
		User  map[string]interface{}
		Total int
	}

	var userList []userWithTotal
	for _, u := range users {
		uid := u["id"].(int)
		total := userTotals[uid]
		userList = append(userList, userWithTotal{User: u, Total: total})
	}

	sort.Slice(userList, func(i, j int) bool {
		return userList[i].Total > userList[j].Total
	})

	rankedUsers := make([]map[string]interface{}, len(userList))
	for i, ut := range userList {
		rankedUsers[i] = ut.User
		rankedUsers[i]["rank"] = i + 1
		rankedUsers[i]["total_points"] = ut.Total
	}

	return repository.SetLeaderboardCache(rankedUsers, contests, results, userTotals)
}
