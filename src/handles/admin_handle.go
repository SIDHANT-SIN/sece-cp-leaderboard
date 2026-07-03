package handles

import (
	"context"
	"fmt"
	"leaderboard/src/database"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"leaderboard/src/configs"
	"leaderboard/src/workers"

	"github.com/hibiken/asynq"

	"github.com/gin-gonic/gin"
)

type AdminHandle struct {
	cfg   *configs.Config
	repo  Repository
	cache CacheBuilder
}

func NewAdminHandler(repo Repository, cache CacheBuilder, cfg *configs.Config) *AdminHandle {
	return &AdminHandle{
		cfg:   cfg,
		repo:  repo,
		cache: cache,
	}
}

// lists all contests
func (h *AdminHandle) ShowContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := h.repo.GetContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

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

// Adds a contest if the codeforces ID is correct
func (h *AdminHandle) AddContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	cfid := c.PostForm("cfid")
	if strings.TrimSpace(cfid) == "" {
		c.String(http.StatusBadRequest, "Contest ID cannot be empty")
		return
	}

	statusData, err := h.repo.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		c.String(http.StatusConflict, "Another sync operation is currently running (JobID: %v). Please wait.", statusData["job_id"])
		return
	}

	jobID := fmt.Sprintf("add_contest_%s_%d", cfid, time.Now().Unix())

	_ = h.repo.CreateSyncLog(jobID, 1)

	task, err := workers.NewCFAddContestTask(jobID, cfid)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to build background task: %v", err)
		return
	}

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

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// deletes a contest and its results
func (h *AdminHandle) DeleteContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	id := c.PostForm("id")

	err = h.repo.DeleteResultsByContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest results: %v", err)
		return
	}

	err = h.repo.DeleteContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest: %v", err)
		return
	}

	if err := h.cache.RebuildLeaderboardCache(); err != nil {
		log.Printf("failed to rebuild leaderboard cache: %v", err)
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// triggers recalculation of ranks for contests count defined by user
func (h *AdminHandle) RefreshResults(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
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

	err = h.refreshAllUserContestResults(limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// refresh all user result of limit count

func (h *AdminHandle) refreshAllUserContestResults(limit int) error {

	statusData, err := h.repo.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		return fmt.Errorf("another sync job (%v) is currently active", statusData["job_id"])
	}

	jobID := fmt.Sprintf("batch_refresh_%d", time.Now().Unix())

	totalContests := 0
	contestRows, err := h.repo.GetContests()
	if err == nil {
		for contestRows.Next() {
			totalContests++
		}
		defer func() {
			if err := contestRows.Close(); err != nil {
				log.Printf("failed to close Contest rows: %v", err)
			}
		}()
	}
	if totalContests == 0 {
		totalContests = 1
	}

	if limit > 0 {
		if limit > totalContests {
			return fmt.Errorf("requested limit (%d) exceeds total contests (%d)", limit, totalContests)
		}
		totalContests = limit
	}

	_ = h.repo.CreateSyncLog(jobID, totalContests)

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
	return nil
}

// admin dashboard route handler
func (h *AdminHandle) ShowAdminDashboard(c *gin.Context) {

	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := h.repo.GetRecentSyncHistory(10)

	history := []map[string]interface{}{}

	if err == nil {
		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("failed to close response body: %v", err)
			}
		}()

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

			switch status {
			case "cancelled":
				failedContestIDs = "Idk you cancelled mid way"
			case "completed":
				failedContestIDs = "[None]"
			}

			if len(jobID) > 0 {
				firstChar := jobID[0]
				switch firstChar {
				case 'a':
					if len(jobID) > 11 {
						jobID = jobID[:len(jobID)-11]
					} else {
						jobID = ""
					}
					words := strings.Split(jobID, "_")
					for i, word := range words {
						if len(word) > 0 && word[0] >= 'a' && word[0] <= 'z' {
							words[i] = strings.ToUpper(string(word[0])) + word[1:]
						}
					}
					jobID = strings.Join(words, " ")
				case 'c':
					jobID = "Cron Handles Rating"
					durationSeconds = 16

				case 'b':
					jobID = "All User Results"
				case 'r':
					jobID = "Handles Rating"
				}
			}

			history = append(history, map[string]interface{}{
				"job_id":              jobID,
				"status":              status,
				"successful_contests": successful,
				"total_contests":      total,
				"failed_contest_ids":  failedContestIDs,
				"started_at":          durationSeconds,
				"completed_at":        completedAt,
			})
		}
	}

	c.HTML(http.StatusOK, "admin.tmpl", gin.H{
		"history": history,
		"error":   nil,
	})
}

// returns JSON progress data for the running background sync task
func (h *AdminHandle) GetSyncStatus(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}

	statusData, err := h.repo.GetCurrentSyncStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sync status"})
		return
	}

	status, _ := statusData["status"].(string)
	ctx := context.Background()

	if status == "processing" {
		database.RedisClient.Set(ctx, "sync:was_processing", "1", 30*time.Minute)
	} else {
		wasProcessing, _ := database.RedisClient.Get(ctx, "sync:was_processing").Result()
		if wasProcessing == "1" {
			database.RedisClient.Del(ctx, "sync:was_processing")
			if err := h.cache.RebuildLeaderboardCache(); err != nil {
				log.Printf("failed to rebuild leaderboard cache: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, statusData)
}

// CancelSync sets a termination flag in Redis to halt the current sync loop
func (h *AdminHandle) CancelSync(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}

	err = h.repo.SetSyncCancelSignal()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set cancellation signal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancel_signal_sent"})
}
