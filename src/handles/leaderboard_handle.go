package handles

import (
	"net/http"
	"sort"

	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

// ShowLeaderboard renders the primary leaderboard with ranked active users
func ShowLeaderboard(c *gin.Context) {
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

	// Query results for each user in each contest
	results := make(map[int]map[int]map[string]interface{}) // user_id -> contest_id -> result
	userTotals := make(map[int]int)                         // user_id -> total points

	rows, err := repository.GetAllResults()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID, contestID, rank, points int
			rows.Scan(&userID, &contestID, &rank, &points)

			// Only sum points for contests that are currently in the DB
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

	// Sort users by total points descending
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

	// Assign ranks
	rankedUsers := make([]map[string]interface{}, len(userList))
	for i, ut := range userList {
		rankedUsers[i] = ut.User
		rankedUsers[i]["rank"] = i + 1
		rankedUsers[i]["total_points"] = ut.Total
	}

	c.HTML(http.StatusOK, "leaderboard.tmpl", gin.H{
		"users":      rankedUsers,
		"contests":   contests,
		"results":    results,
		"userTotals": userTotals,
	})
}

// ShowPastLeaderboard renders the past leaderboard filtered by batch_year
func ShowPastLeaderboard(c *gin.Context) {
	batch := c.Query("batch")
	if batch == "" {
		batch = "2023" // default
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

	// Sort by max_rating DESC
	sort.Slice(users, func(i, j int) bool {
		return users[i]["max_rating"].(int) > users[j]["max_rating"].(int)
	})

	// Assign rank
	for i := range users {
		users[i]["rank"] = i + 1
	}

	c.HTML(http.StatusOK, "past_leaderboard.tmpl", gin.H{
		"users":         users,
		"selectedBatch": batch,
	})
}
