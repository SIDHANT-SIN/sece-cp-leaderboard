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

// ShowPastUsers lists all past users
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

// AddPastUser adds a past user
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

// DeletePastUser deletes a past user by id
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

// // RefreshRating pulls stats for all past users from Codeforces and updates local database ratings
// func RefreshRating(c *gin.Context) {
// 	fmt.Println("STEP 0: endpoint hit")

// 	cookie, err := c.Cookie("maintainer_logged_in")
// 	if err != nil || cookie != "true" {
// 		fmt.Println("STEP 1 FAILED: cookie issue:", err, cookie)
// 		c.Redirect(http.StatusSeeOther, "/maintainer")
// 		return
// 	}
// 	fmt.Println("STEP 1 OK: cookie validated")

// 	handles, err := repository.GetPastUserHandles()
// 	if err != nil {
// 		fmt.Println("STEP 2 FAILED: DB query error:", err)
// 		c.String(http.StatusInternalServerError, "DB error")
// 		return
// 	}

// 	fmt.Println("STEP 2 OK: handles fetched =", len(handles), handles)

// 	if len(handles) == 0 {
// 		fmt.Println("STEP 2 EXIT: no users")
// 		c.String(http.StatusOK, "No users to refresh")
// 		return
// 	}

// 	handleStr := strings.Join(handles, ";")
// 	url := "https://codeforces.com/api/user.info?handles=" + handleStr

// 	fmt.Println("STEP 3: calling CF API:", url)

// 	resp, err := http.Get(url)
// 	if err != nil {
// 		fmt.Println("STEP 3 FAILED: CF request error:", err)
// 		c.String(http.StatusInternalServerError, "CF request failed")
// 		return
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)
// 	fmt.Println("STEP 3 RESPONSE STATUS:", resp.StatusCode)

// 	if resp.StatusCode != 200 {
// 		fmt.Println("STEP 3 FAILED: bad status")
// 		c.String(http.StatusBadGateway, "CF API error: %s", string(body))
// 		return
// 	}

// 	var apiResp struct {
// 		Status string `json:"status"`
// 		Result []struct {
// 			Handle    string `json:"handle"`
// 			Rating    int    `json:"rating"`
// 			MaxRating int    `json:"maxRating"`
// 			Rank      string `json:"rank"`
// 		} `json:"result"`
// 	}

// 	err = json.Unmarshal(body, &apiResp)
// 	if err != nil {
// 		fmt.Println("STEP 4 FAILED: JSON unmarshal error:", err)
// 		return
// 	}

// 	fmt.Println("STEP 4 OK: CF status =", apiResp.Status)
// 	if apiResp.Status != "OK" {
// 		fmt.Println("STEP 4 FAILED: CF API returned not OK")
// 		return
// 	}

// 	for _, u := range apiResp.Result {
// 		err := repository.UpdatePastUserRating(u.Rating, u.MaxRating, u.Rank, u.Handle)
// 		if err != nil {
// 			fmt.Println("DB UPDATE FAILED:", u.Handle, err)
// 		} else {
// 			fmt.Println("UPDATED:", u.Handle)
// 		}
// 	}

// 	fmt.Println("STEP 5 DONE")
// 	c.Redirect(http.StatusSeeOther, "/maintainer/users")
// }


// RefreshRating triggers a background task to pull stats for all past users from Codeforces
func RefreshRating(c *gin.Context) {
	fmt.Println("STEP 0: endpoint hit")

	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		fmt.Println("STEP 1 FAILED: cookie issue:", err, cookie)
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}
	fmt.Println("STEP 1 OK: cookie validated")

	// 1. Pre-Flight Check: Ensure no active task collision
	statusData, err := repository.GetCurrentSyncStatus()
	if err == nil && statusData["status"] == "processing" {
		c.String(http.StatusConflict, "Another sync operation is currently running (JobID: %v). Please wait.", statusData["job_id"])
		return
	}

	// 2. Fetch handles just to make sure we have targets to process
	handles, err := repository.GetPastUserHandles()
	if err != nil {
		fmt.Println("STEP 2 FAILED: DB query error:", err)
		c.String(http.StatusInternalServerError, "DB error")
		return
	}

	if len(handles) == 0 {
		fmt.Println("STEP 2 EXIT: no users")
		c.String(http.StatusOK, "No users to refresh")
		return
	}

	// 3. Generate a unique Job identity
	jobID := fmt.Sprintf("rating_refresh_%d", time.Now().Unix())

	// 4. Initialize the SQLite Sync History log 
	// Codeforces user.info handles multiple records in a single network request batch, so total expected operations = 1
	_ = repository.CreateSyncLog(jobID, 1)

	// 5. Build the Asynq task payload using your workers package constructor
	task, err := workers.NewCFRefreshRatingTask(jobID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to build background task: %v", err)
		return
	}

	// 6. Fire: Fetch the client using your getter function
	asynqClient := workers.GetClient()
	if asynqClient == nil {
		c.String(http.StatusInternalServerError, "Asynq client instance is not initialized in workers package")
		return
	}

	// Enqueue into the standard default queue
	_, err = asynqClient.Enqueue(task, asynq.Queue(workers.QueueDefault))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to enqueue task: %v", err)
		return
	}

	// 7. Forget: Redirect straight back to the maintainer panel
	fmt.Println("STEP 5 DISPATCHED ASYNC SUCCESSFULLY")
	c.Redirect(http.StatusSeeOther, "/maintainer/users")
}