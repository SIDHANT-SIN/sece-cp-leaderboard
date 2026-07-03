package handles

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"leaderboard/src/configs"

	"leaderboard/src/workers"
)

type TaskEnqueuer interface {
	EnqueueRefreshRatingTask(jobID string) error
	EnqueueAddContestTask(jobID, cfid string) error
	EnqueueBatchRefreshTask(jobID string) error
}

type defaultTaskEnqueuer struct{}

func NewDefaultTaskEnqueuer() TaskEnqueuer {
	return &defaultTaskEnqueuer{}
}

func (d *defaultTaskEnqueuer) EnqueueRefreshRatingTask(jobID string) error {
	task, err := workers.NewCFRefreshRatingTask(jobID)
	if err != nil {
		return err
	}
	client := workers.GetClient()
	if client == nil {
		return fmt.Errorf("Asynq client instance is not initialized")
	}
	_, err = client.Enqueue(task, asynq.Queue(workers.QueueDefault))
	return err
}

func (d *defaultTaskEnqueuer) EnqueueAddContestTask(jobID, cfid string) error {
	task, err := workers.NewCFAddContestTask(jobID, cfid)
	if err != nil {
		return err
	}
	client := workers.GetClient()
	if client == nil {
		return fmt.Errorf("Asynq client instance is not initialized")
	}
	_, err = client.Enqueue(task, asynq.Queue(workers.QueueCritical))
	return err
}

func (d *defaultTaskEnqueuer) EnqueueBatchRefreshTask(jobID string) error {
	task, err := workers.NewCFBatchRefreshTask(jobID)
	if err != nil {
		return err
	}
	client := workers.GetClient()
	if client == nil {
		return fmt.Errorf("Asynq client instance is not initialized")
	}
	_, err = client.Enqueue(task, asynq.Queue(workers.QueueCritical))
	return err
}

type MaintainerUsersHandler struct {
	cfg       *configs.Config
	repo      Repository
	taskQueue TaskEnqueuer
}

func NewMaintainerUsersHandler(cfg *configs.Config, repo Repository, taskQueue TaskEnqueuer) *MaintainerUsersHandler {
	return &MaintainerUsersHandler{
		cfg:       cfg,
		repo:      repo,
		taskQueue: taskQueue,
	}
}

// lists all past users
func (h *MaintainerUsersHandler) ShowPastUsers(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	rows, err := h.repo.GetPastUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close contestRows: %v", err)
		}
	}()

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

// adds a past user
func (h *MaintainerUsersHandler) AddPastUser(c *gin.Context) {
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

	err = h.repo.AddPastUser(handle, display, batch)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not add user: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}

// deletes a past user by id
func (h *MaintainerUsersHandler) DeletePastUser(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	id := c.PostForm("id")
	err = h.repo.DeletePastUser(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Delete failed: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}

// refresh all handles rating

func (h *MaintainerUsersHandler) RefreshRating(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	statusData, err := h.repo.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		c.String(http.StatusConflict, "Another sync operation is currently running (JobID: %v). Please wait.", statusData["job_id"])
		return
	}

	handles, err := h.repo.GetPastUserHandles()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}

	if len(handles) == 0 {
		c.String(http.StatusOK, "No users to refresh")
		return
	}

	jobID := fmt.Sprintf("rating_refresh_%d", time.Now().Unix())

	_ = h.repo.CreateSyncLog(jobID, 1)

	err = h.taskQueue.EnqueueRefreshRatingTask(jobID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to enqueue task: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}

func (h *MaintainerUsersHandler) CreateICPCProblem(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != h.cfg.MaintainerPassword {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.Redirect(http.StatusSeeOther, "/maintainer/icpc_pyq")
}
