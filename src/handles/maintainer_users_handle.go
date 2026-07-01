package handles

import (
	"fmt"
	
	"net/http"
	"strconv"
	
    "time"
	"leaderboard/src/repository"
"leaderboard/src/workers"
	"github.com/hibiken/asynq"
	"github.com/gin-gonic/gin"
)

//  lists all past users
func ShowPastUsers(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	rows, err := repository.GetPastUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

	var pastUsers []map[string]interface{}
	for rows.Next() {
		var id, batch, cur, mx int
		var handle, name, title string
		if err := rows.Scan(&id, &handle, &name, &batch, &cur, &mx, &title); err != nil {
			c.String(http.StatusInternalServerError, "DB scan error: %v", err)
			return
		}
		pastUsers = append(pastUsers, map[string]interface{}{
			"id":             id,
			"handle":         handle,
			"display_name":   name,
			"batch":          batch,
			"current_rating": cur,
			"max_rating":     mx,
			"title":          title,
		})
	}

	c.HTML(http.StatusOK, "maintainer_users.tmpl", gin.H{
		"past_users": pastUsers,
	})
}

//  adds a past user
func AddPastUser(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	handle := c.PostForm("handle")
	display := c.PostForm("display_name")
	batchStr := c.PostForm("batch")

	batch, err := strconv.Atoi(batchStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid batch year: %v", err)
		return
	}

	err = repository.AddPastUser(handle, display, batch)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not add user: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}

// deletes a past user by id
func DeletePastUser(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	id := c.PostForm("id")
	err = repository.DeletePastUser(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Delete failed: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}

// refresh all handles rating

func RefreshRating(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	statusData, err := repository.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		c.String(http.StatusConflict, "Another sync operation is currently running (JobID: %v). Please wait.", statusData["job_id"])
		return
	}

	handles, err := repository.GetPastUserHandles()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}

	if len(handles) == 0 {
		c.String(http.StatusOK, "No users to refresh")
		return
	}

	jobID := fmt.Sprintf("rating_refresh_%d", time.Now().Unix())

	
	_ = repository.CreateSyncLog(jobID, 1)

	task, err := workers.NewCFRefreshRatingTask(jobID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to build background task: %v", err)
		return
	}

	asynqClient := workers.GetClient()
	if asynqClient == nil {
		c.String(http.StatusInternalServerError, "Asynq client instance is not initialized in workers package")
		return
	}

	_, err = asynqClient.Enqueue(task, asynq.Queue(workers.QueueDefault))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to enqueue task: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}


func CreateICPCProblem(c *gin.Context) {
	
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != cfg.MaintainerPassword {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/icpc_pyq")
}