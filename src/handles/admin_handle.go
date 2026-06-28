package handles

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"leaderboard/src/database"
	"leaderboard/src/repository"
	"leaderboard/src/workers"

	"github.com/hibiken/asynq"

	"github.com/gin-gonic/gin"
)

// ShowContests lists all contests
func ShowContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := repository.GetContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

	var contests []map[string]interface{}
	for rows.Next() {
		var id, cfid, startTime int
		var name string
		if err := rows.Scan(&id, &cfid, &name, &startTime); err != nil {
			c.String(http.StatusInternalServerError, "DB scan error: %v", err)
			return
		}
		contests = append(contests, map[string]interface{}{
			"id":         id,
			"cfid":       cfid,
			"name":       name,
			"start_time": startTime,
		})
	}

	c.HTML(http.StatusOK, "admin_contests.tmpl", gin.H{"contests": contests})
}

// // AddContest handles adding a single contest using Codeforces API
// func AddContest(c *gin.Context) {
// 	cookie, err := c.Cookie("admin_logged_in")
// 	if err != nil || cookie != cfg.AdminPasswordHash {
// 		c.Redirect(http.StatusSeeOther, "/admin_login")
// 		return
// 	}

// 	cfid := c.PostForm("cfid")

// 	resp, err := http.Get("https://codeforces.com/api/contest.standings?contestId=" + cfid)
// 	if err != nil {
// 		fmt.Println("HTTP ERROR:", err)
// 		c.String(http.StatusBadRequest, "Could not fetch contest info from Codeforces")
// 		return
// 	}
// 	defer resp.Body.Close()

// 	fmt.Println("Contest ID:", cfid)
// 	fmt.Println("Status Code:", resp.StatusCode)

// 	if resp.StatusCode != 200 {
// 		if resp.StatusCode >= 500 {
// 			c.String(
// 				http.StatusBadGateway,
// 				"Codeforces API server is currently unavailable (HTTP %d). Try later after a few minutes or hours.",
// 				resp.StatusCode,
// 			)
// 			return
// 		}

// 		c.String(
// 			http.StatusBadRequest,
// 			"Could not fetch contest info from Codeforces (HTTP %d)",
// 			resp.StatusCode,
// 		)
// 		return
// 	}

// 	var apiResp struct {
// 		Status string `json:"status"`
// 		Result struct {
// 			Contest struct {
// 				Id        int    `json:"id"`
// 				Name      string `json:"name"`
// 				StartTime int64  `json:"startTimeSeconds"`
// 			} `json:"contest"`
// 		} `json:"result"`
// 	}

// 	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || apiResp.Status != "OK" {
// 		c.String(http.StatusBadRequest, "Could not parse contest info from Codeforces")
// 		return
// 	}

// 	err = repository.AddContest(apiResp.Result.Contest.Id, apiResp.Result.Contest.Name, apiResp.Result.Contest.StartTime)
// 	if err != nil {
// 		c.String(http.StatusBadRequest, "Could not add contest: %v", err)
// 		return
// 	}

// 	rebuildLeaderboardCache()

// 	c.Redirect(http.StatusSeeOther, "/admin/contests")
// }

// AddContest handles adding a single contest using Codeforces API asynchronously
func AddContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	cfid := c.PostForm("cfid")
	if strings.TrimSpace(cfid) == "" {
		c.String(http.StatusBadRequest, "Contest ID cannot be empty")
		return
	}

	// 1. Pre-Flight Check: Ensure no active task collision
	statusData, err := repository.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		c.String(http.StatusConflict, "Another sync operation is currently running (JobID: %v). Please wait.", statusData["job_id"])
		return
	}

	// 2. Generate a unique Job identity
	jobID := fmt.Sprintf("add_contest_%s_%d", cfid, time.Now().Unix())

	// 3. Initialize the SQLite Sync History log (1 expected item)
	_ = repository.CreateSyncLog(jobID, 1)

	// 4. Build the Asynq task payload using your workers package constructor
	task, err := workers.NewCFAddContestTask(jobID, cfid)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to build background task: %v", err)
		return
	}

	// 5. Fire: Fetch the client using your getter function 👈
	asynqClient := workers.GetClient()
	if asynqClient == nil {
		c.String(http.StatusInternalServerError, "Asynq client instance is not initialized in workers package")
		return
	}
	
	_, err = asynqClient.Enqueue(task, asynq.Queue(workers.QueueCritical))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to enqueue task: %v", err)
		return
	}

	// 6. Forget: Redirect straight back to the admin contests page
	c.Redirect(http.StatusSeeOther, "/admin/contests")
}


// DeleteContest deletes a contest and its results
func DeleteContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	id := c.PostForm("id")

	// Delete all results associated with this contest first
	err = repository.DeleteResultsByContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest results: %v", err)
		return
	}

	// Delete the contest itself
	err = repository.DeleteContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest: %v", err)
		return
	}

	rebuildLeaderboardCache()

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// DeleteAllContests deletes all contests and all user results
func DeleteAllContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	err = repository.DeleteAllResults()
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not delete contest results: %v", err)
		return
	}

	err = repository.DeleteAllContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not delete all contests: %v", err)
		return
	}

	rebuildLeaderboardCache()

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}


// RefreshResults triggers recalculation of scores/ranks for all contests
func RefreshResults(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	limitStr := c.PostForm("limit")
	limit := 0
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid limit provided. Must be a positive integer."})
			return
		}
	}

	err = refreshAllUserContestResults(limit) // Passing limit down
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Cache rebuild intentionally NOT called here — the job was just enqueued
	// and hasn't written anything yet. GetSyncStatus rebuilds the cache once
	// it observes the job has actually finished (see below).
	// Return a JSON 200 OK so the JS fetch() initiates polling smoothly
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}


// Calculate points for a given rank
func calculatePoints(rank, total int, div string) int {
	if total == 0 || rank == 0 {
		return 0
	}
	var d float64
	switch div {
	case "Div. 2", "Div. 1":
		d = 1.0
	case "Div. 3":
		d = 0.67
	case "Div. 4":
		d = 0.33
	default:
		d = 1.0
	}
	baseParticipation := 2
	score := int(math.Max(10*d*math.Log10(float64(total+1)/float64(rank+1)), 0)) + baseParticipation
	return score
}

// // Refresh all user standings and point calculations asynchronously via Asynq
// func refreshAllUserContestResults() error {
// 	fmt.Println("\n================ ASYNC REFRESH INITIALIZED ================")

// 	// 1. Pre-Flight Check: Ensure no active collision
// 	statusData, err := repository.GetCurrentSyncStatus()
// 	if err == nil && statusData["status"] == "processing" {
// 		return fmt.Errorf("another sync job (%v) is currently active", statusData["job_id"])
// 	}

// 	// 2. Generate unique Job ID for tracking
// 	jobID := fmt.Sprintf("batch_refresh_%d", time.Now().Unix())

// 	// 3. Calculate total expected items from database for progress tracking
// 	totalContests := 0
// 	contestRows, err := repository.GetContestIDs()
// 	if err == nil {
// 		for contestRows.Next() {
// 			totalContests++
// 		}
// 		contestRows.Close()
// 	}
// 	if totalContests == 0 {
// 		totalContests = 1 // Prevent boundary errors if table is empty
// 	}

// 	// 4. Create the initial sync history tracking row in SQLite
// 	_ = repository.CreateSyncLog(jobID, totalContests)

// 	// 5. Build your background loop task payload
// 	task, err := workers.NewCFBatchRefreshTask(jobID)
// 	if err != nil {
// 		return fmt.Errorf("failed to construct batch task: %v", err)
// 	}

// 	// 6. Enqueue using your getter function 👈
// 	asynqClient := workers.GetClient()
// 	if asynqClient == nil {
// 		return fmt.Errorf("asynq client is uninitialized in workers package")
// 	}

// 	_, err = asynqClient.Enqueue(task, asynq.Queue(workers.QueueCritical))
// 	if err != nil {
// 		return fmt.Errorf("failed to dispatch task to redis: %v", err)
// 	}

// 	fmt.Printf("[handles] Batch refresh task successfully pushed to queue. JobID: %s\n", jobID)
// 	return nil
// }


func refreshAllUserContestResults(limit int) error {
	fmt.Println("\n================ ASYNC REFRESH INITIALIZED ================")

	statusData, err := repository.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		return fmt.Errorf("another sync job (%v) is currently active", statusData["job_id"])
	}

	jobID := fmt.Sprintf("batch_refresh_%d", time.Now().Unix())

	totalContests := 0
	contestRows, err := repository.GetContests() // Using GetContests to count ALL
	if err == nil {
		for contestRows.Next() {
			totalContests++
		}
		contestRows.Close()
	}
	if totalContests == 0 {
		totalContests = 1 
	}

	// Boundary check against DB
	if limit > 0 {
		if limit > totalContests {
			return fmt.Errorf("requested limit (%d) exceeds total contests (%d)", limit, totalContests)
		}
		totalContests = limit // Adjust UI progress bar to the limit
	}

	_ = repository.CreateSyncLog(jobID, totalContests)

	// Temporarily embed the limit in Redis so the worker finds it seamlessly 
	if limit > 0 {
		database.RedisClient.Set(context.Background(), fmt.Sprintf("sync_limit:%s", jobID), limit, 30*time.Minute)
	}

	task, err := workers.NewCFBatchRefreshTask(jobID)
	if err != nil {
		return fmt.Errorf("failed to construct batch task: %v", err)
	}

	asynqClient := workers.GetClient()
	if asynqClient == nil {
		return fmt.Errorf("asynq client is uninitialized in workers package")
	}

	_, err = asynqClient.Enqueue(task, asynq.Queue(workers.QueueCritical))
	if err != nil {
		return fmt.Errorf("failed to dispatch task to redis: %v", err)
	}

	fmt.Printf("[handles] Batch refresh task pushed to queue. JobID: %s, Limit: %d\n", jobID, limit)
	return nil
}


// admin dashboard route handler
func ShowAdminDashboard(c *gin.Context) {


	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := repository.GetRecentSyncHistory(10)

	history := []map[string]interface{}{}

	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var (
				jobID            string
				status           string
				successful       int
				total            int
				failedContestIDs string
				startedAt        string
				completedAt      string
			)

			err := rows.Scan(
				&jobID,
				&status,
				&successful,
				&total,
				&failedContestIDs,
				&startedAt,
				&completedAt,
			)

			if err != nil {
				continue
			}

			// 1. Calculate time difference in seconds for "started_at"
			// Adjust the layout string ("2006-01-02 15:04:05") if your database stores timestamps differently
			timeLayout := "2006-01-02 15:04:05" 
			var durationSeconds int64 = 0

			startT, errStart := time.Parse(timeLayout, startedAt)
			endT, errEnd := time.Parse(timeLayout, completedAt)

			if errStart == nil && errEnd == nil {
				durationSeconds = int64(endT.Sub(startT).Seconds())
			}

			loc, errLoc := time.LoadLocation("Asia/Kolkata")
        if errLoc == nil && errEnd == nil {  
        completedAt = endT.In(loc).Format(timeLayout)
               }

			// 2. Handle failedContestIDs based on status
			switch status {
            case "cancelled":
				failedContestIDs = "Idk you cancelled mid way"
			case "completed":
				failedContestIDs = "[None]"
			}

			// 3. Process job_id based on starting character
			if len(jobID) > 0 {
				firstChar := jobID[0]
				switch firstChar {
             case 'a':
					// Safely remove the last 11 characters if length permits
					if len(jobID) > 11 {
						jobID = jobID[:len(jobID)-11]
					} else {
						jobID = ""
					}
					// Split by underscore and join with space
					words := strings.Split(jobID, "_")
					for i, word := range words {
         if len(word) > 0 && word[0] >= 'a' && word[0] <= 'z' {
            words[i] = strings.ToUpper(string(word[0])) + word[1:]
        }
    }
					jobID = strings.Join(words, " ")
				case 'c':
					jobID = "Cron Handles Rating"
					durationSeconds=16
				
			     case 'b':
				jobID = "All User Results"
				 case 'r':
					jobID = "Handles Rating"
			}
			}

			history = append(history, map[string]interface{}{
				"job_id":               jobID,
				"status":               status,
				"successful_contests": successful,
				"total_contests":      total,
				"failed_contest_ids":  failedContestIDs,
				"started_at":          durationSeconds, // Named exactly as requested, now holds seconds taken
				"completed_at":        completedAt,
			})
		}
	}

	c.HTML(http.StatusOK, "admin.tmpl", gin.H{
		"history": history,
		"error":   nil,
	})
}
// GetSyncStatus returns JSON progress data for the running background sync task
func GetSyncStatus(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}

	// Fetch current progress metrics from the repository layer
	statusData, err := repository.GetCurrentSyncStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sync status"})
		return
	}

	// GetCurrentSyncStatus only ever reports "processing" or "idle" (the
	// worker deletes its Redis keys as soon as it finishes, before any poll
	// can observe "completed"/"cancelled" there, and the matching SQL row
	// has already flipped status away from 'processing' too). admin.tmpl's
	// JS reloads the page on exactly this processing -> idle transition, so
	// catching that same transition here lets us rebuild the cache right
	// before the response that triggers the reload — guaranteeing the
	// reload sees fresh data instead of whatever was cached before refresh.
	status, _ := statusData["status"].(string)
	ctx := context.Background()

	if status == "processing" {
		database.RedisClient.Set(ctx, "sync:was_processing", "1", 30*time.Minute)
	} else {
		wasProcessing, _ := database.RedisClient.Get(ctx, "sync:was_processing").Result()
		if wasProcessing == "1" {
			database.RedisClient.Del(ctx, "sync:was_processing")
			rebuildLeaderboardCache()
		}
	}

	c.JSON(http.StatusOK, statusData)
}

// CancelSync sets a termination flag in Redis to gracefully halt the current sync loop
func CancelSync(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}

	// Write cancellation trigger to Redis/DB state layer
	err = repository.SetSyncCancelSignal()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set cancellation signal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancel_signal_sent"})
}