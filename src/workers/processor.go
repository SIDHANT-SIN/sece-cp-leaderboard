package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"leaderboard/src/repository"

	"leaderboard/src/database"

	"github.com/hibiken/asynq"
)

// cfRateLimit is the minimum time each task must take (2.1s for safety over CF's 2s limit)
const cfRateLimit = 2100 * time.Millisecond
const cfRateLimit2 = 10* time.Millisecond

type userEntry struct {
	ID     int
	Handle string
}

// waitForCFRateLimit sleeps the remaining time to ensure 2.1s total from start.
// Measured from before the API call to after data is received + processed.
func waitForCFRateLimit(start time.Time) {
	elapsed := time.Since(start)
	if elapsed < cfRateLimit {
		time.Sleep(cfRateLimit - elapsed)
	}
}
func waitForCFRateLimit2(start time.Time) {
	elapsed := time.Since(start)
	if elapsed < cfRateLimit2 {
		time.Sleep(cfRateLimit2 - elapsed)
	}
}

func updateJobError(jobID, msg string) {
	if jobID == "" {
		return
	}
	state, err := GetJobState(jobID)
	if err == nil && state != nil {
		state.Status = "failed"
		state.Error = msg
		state.CompletedAt = time.Now().Format(time.RFC3339)
		SetJobState(jobID, state, 10*time.Minute)
	}
}

// processSingleContestStandings is a shared helper to call CF API and store results
func processSingleContestStandings(cfContestID, contestDBID int, users []userEntry) error {
	url := "https://codeforces.com/api/contest.ratingChanges?contestId=" + fmt.Sprint(cfContestID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CF standings returned HTTP %d", resp.StatusCode)
	}

	var ratingChanges struct {
		Status string `json:"status"`
		Result []struct {
			ContestId   int    `json:"contestId"`
			ContestName string `json:"contestName"`
			Handle      string `json:"handle"`
			Rank        int    `json:"rank"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ratingChanges); err != nil {
		return fmt.Errorf("JSON decode failed: %w", err)
	}

	if ratingChanges.Status != "OK" {
		return fmt.Errorf("CF API status not OK")
	}

	total := len(ratingChanges.Result)
	if total == 0 {
		return nil
	}

	contestName := ratingChanges.Result[0].ContestName
	div := detectDivision(contestName)

	rankMap := make(map[string]int)
	for _, row := range ratingChanges.Result {
		rankMap[row.Handle] = row.Rank
	}

	for _, user := range users {
		userRank := rankMap[user.Handle]
		points := 0
		if userRank > 0 {
			points = calculatePoints(userRank, total, div)
		}
		if err := repository.UpsertResult(user.ID, contestDBID, userRank, points); err != nil {
			fmt.Printf("[worker] DB INSERT ERROR user=%s contest=%d err=%v\n", user.Handle, cfContestID, err)
		}
	}

	return nil
}

// HandleCFRatingChanges processes rating changes for a single contest.
func HandleCFRatingChanges(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit2(start)

	var p CFRatingChangesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Processing single contest CF#%d\n", p.CFContestID)

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			SetJobState(p.JobID, state, 10*time.Minute)
		}

		// 👉 ADDED: Initialize global tracking keys for polling
		database.RedisClient.Set(ctx, "sync:job_id", p.JobID, 30*time.Minute)
		database.RedisClient.Set(ctx, "sync:status", "processing", 30*time.Minute)
		database.RedisClient.Set(ctx, "sync:total", 1, 30*time.Minute)
		database.RedisClient.Set(ctx, "sync:current", 0, 30*time.Minute)
	}

	userRows, err := repository.GetUsers()
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "failed to load users: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			
			// 👉 ADDED: Clean up global tracking keys on error
			database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")
			
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}
	defer userRows.Close()

	var users []userEntry
	for userRows.Next() {
		var id int
		var handle, display string
		if err := userRows.Scan(&id, &handle, &display); err == nil {
			users = append(users, userEntry{ID: id, Handle: handle})
		}
	}

	err = processSingleContestStandings(p.CFContestID, p.ContestDBID, users)
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			
			// 👉 ADDED: Clean up global tracking keys on execution failure
			database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")
			
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			SetJobState(p.JobID, state, 10*time.Minute)
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")
		
		// 👉 ADDED: Clean up global tracking keys on success completion
		database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")
		
		ReleaseActiveJobLock(p.JobID)
	}

	return nil
}

// HandleCFBatchRefresh processes rating changes for all contests sequentially.
func HandleCFBatchRefresh(ctx context.Context, t *asynq.Task) error {
	var p CFBatchRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Starting batch refresh for JobID %s\n", p.JobID)

	userRows, err := repository.GetUsers()
	if err != nil {
		updateJobError(p.JobID, "failed to load users: "+err.Error())
		ReleaseActiveJobLock(p.JobID)
		return err
	}
	defer userRows.Close()

	var users []userEntry
	for userRows.Next() {
		var id int
		var handle, display string
		if err := userRows.Scan(&id, &handle, &display); err == nil {
			users = append(users, userEntry{ID: id, Handle: handle})
		}
	}
//    //teststart
// 	contestRows, err := repository.GetContestIDs()
// 	if err != nil {
// 		updateJobError(p.JobID, "failed to load contests: "+err.Error())
// 		ReleaseActiveJobLock(p.JobID)
// 		return err
// 	}
// 	defer contestRows.Close()

// 	type contestEntry struct {
// 		ID   int
// 		CFID int
// 	}
// 	var contests []contestEntry
// 	for contestRows.Next() {
// 		var id, cfid int
// 		if err := contestRows.Scan(&id, &cfid); err == nil {
// 			contests = append(contests, contestEntry{ID: id, CFID: cfid})
// 		}
// 	}

// 	total := len(contests)
// 	successful := 0
//    //testclose


// Switch to GetContests() to guarantee they are ordered by start_time DESC
	contestRows, err := repository.GetContests()
	if err != nil {
		updateJobError(p.JobID, "failed to load contests: "+err.Error())
		ReleaseActiveJobLock(p.JobID)
		return err
	}
	defer contestRows.Close()

	type contestEntry struct {
		ID   int
		CFID int
	}
	var contests []contestEntry
	for contestRows.Next() {
		var id, cfid int
		var name string
		var startTime int64
		// GetContests yields 4 columns, so scan all 4 to avoid errors
		if err := contestRows.Scan(&id, &cfid, &name, &startTime); err == nil {
			contests = append(contests, contestEntry{ID: id, CFID: cfid})
		}
	}

	// NEW: Tap into Redis to slice the array if a limit was requested
	limitStr, err := database.RedisClient.Get(ctx, fmt.Sprintf("sync_limit:%s", p.JobID)).Result()
	if err == nil && limitStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		if limit > 0 && limit <= len(contests) {
			contests = contests[:limit] // Slice array to only keep the 'X' newest
		}
		// Wipe the key to keep Redis tidy
		database.RedisClient.Del(ctx, fmt.Sprintf("sync_limit:%s", p.JobID))
	}

	total := len(contests)
	successful := 0

	state, err := GetJobState(p.JobID)
	if err == nil && state != nil {
		state.Total = total
		state.Current = 0
		SetJobState(p.JobID, state, 10*time.Minute)
	}

	database.RedisClient.Set(ctx, "sync:job_id", p.JobID, 30*time.Minute)
	database.RedisClient.Set(ctx, "sync:status", "processing", 30*time.Minute)
	database.RedisClient.Set(ctx, "sync:total", total, 30*time.Minute)
	database.RedisClient.Set(ctx, "sync:current", 0, 30*time.Minute)

	for idx, contest := range contests {
		select {
		case <-ctx.Done():
			fmt.Printf("[worker] Batch refresh JobID %s cancelled mid-way via context\n", p.JobID)
			
			if state, errState := GetJobState(p.JobID); errState == nil && state != nil {
				state.Status = "cancelled"
				state.CompletedAt = time.Now().Format(time.RFC3339)
				SetJobState(p.JobID, state, 10*time.Minute)
			}

			failedList, _ := GetFailedContests(p.JobID)
			failedJSON := "[]"
			if len(failedList) > 0 {
				bytes, _ := json.Marshal(failedList)
				failedJSON = string(bytes)
			}
			_ = repository.UpdateSyncLog(p.JobID, "cancelled", successful, failedJSON)
			ClearFailedContests(p.JobID)
			
			database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")
			ReleaseActiveJobLock(p.JobID)
            
			// 🚀 FIX: Wrap with asynq.SkipRetry to block resurrection attempts
			return fmt.Errorf("task cancelled via context: %w", asynq.SkipRetry)
			
		default:
			// 🚀 CHECKPOINT: Actively check for manual cancellation signal from HTTP route
			cancelSignal, _ := database.RedisClient.Get(ctx, "sync:cancel_signal").Result()
			if cancelSignal == "1" {
				fmt.Printf("[worker] Batch refresh JobID %s manually aborted via Redis signal\n", p.JobID)
				database.RedisClient.Del(ctx, "sync:cancel_signal") // Clear the signal key

				if state, errState := GetJobState(p.JobID); errState == nil && state != nil {
					state.Status = "cancelled"
					state.CompletedAt = time.Now().Format(time.RFC3339)
					SetJobState(p.JobID, state, 10*time.Minute)
				}

				failedList, _ := GetFailedContests(p.JobID)
				failedJSON := "[]"
				if len(failedList) > 0 {
					bytes, _ := json.Marshal(failedList)
					failedJSON = string(bytes)
				}
				_ = repository.UpdateSyncLog(p.JobID, "cancelled", successful, failedJSON)
				ClearFailedContests(p.JobID)

				database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")
				ReleaseActiveJobLock(p.JobID)
                
				// 🚀 FIX: Return asynq.SkipRetry here too so it terminates cleanly and permanently
				return fmt.Errorf("batch sync cancelled by admin request: %w", asynq.SkipRetry)
			}
		}

		start := time.Now()
		fmt.Printf("[worker] JobID %s: Processing contest %d/%d (CF#%d)\n", p.JobID, idx+1, total, contest.CFID)

		err = processSingleContestStandings(contest.CFID, contest.ID, users)
		if err != nil {
			fmt.Printf("[worker] Failed CF#%d: %v\n", contest.CFID, err)
			AppendFailedContest(p.JobID, fmt.Sprintf("CF#%d: %v", contest.CFID, err))
		} else {
			successful++
		}

		state, errState := GetJobState(p.JobID)
		if errState == nil && state != nil {
			state.Current = idx + 1
			SetJobState(p.JobID, state, 10*time.Minute)
		}

		database.RedisClient.Set(ctx, "sync:current", idx+1, 30*time.Minute)

		waitForCFRateLimit2(start)
	}

	fmt.Printf("[worker] Batch refresh JobID %s finished. Successful: %d/%d\n", p.JobID, successful, total)

	failedList, _ := GetFailedContests(p.JobID)
	failedJSON := "[]"
	if len(failedList) > 0 {
		bytes, _ := json.Marshal(failedList)
		failedJSON = string(bytes)
	}

	_ = repository.UpdateSyncLog(p.JobID, "completed", successful, failedJSON)
	ClearFailedContests(p.JobID)

	state, errState := GetJobState(p.JobID)
	if errState == nil && state != nil {
		state.Status = "completed"
		state.CompletedAt = time.Now().Format(time.RFC3339)
		SetJobState(p.JobID, state, 10*time.Minute)
	}

	database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total")

	ReleaseActiveJobLock(p.JobID)
	return nil
}

// HandleCFRefreshRating fetches user.info for all past users and updates ratings in DB.
func HandleCFRefreshRating(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit2(start)

	var p CFRefreshRatingPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Refreshing past user ratings for JobID %s\n", p.JobID)

	if p.JobID != "" {
		// 1. PLACE THE FIX HERE: Creates the SQLite row if triggered by the 4-hour cron
		if p.JobID == "cron_refresh_rating" {
			_ = repository.CreateSyncLog(p.JobID, 1)
		}

		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			SetJobState(p.JobID, state, 10*time.Minute)
		}
	}

	handles, err := repository.GetPastUserHandles()
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "DB error: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID) // Releases lock on error exit
		}
		return err
	}

	if len(handles) == 0 {
		fmt.Println("[worker] No past users to refresh")
		if p.JobID != "" {
			state, err := GetJobState(p.JobID)
			if err == nil && state != nil {
				state.Current = 1
				state.Status = "completed"
				state.CompletedAt = time.Now().Format(time.RFC3339)
				SetJobState(p.JobID, state, 10*time.Minute)
			}
			_ = repository.UpdateSyncLog(p.JobID, "completed", 0, "[]")
			ReleaseActiveJobLock(p.JobID) // Releases lock if no users found
		}
		return nil
	}

	handleStr := strings.Join(handles, ";")
	url := "https://codeforces.com/api/user.info?handles=" + handleStr
	fmt.Println("[worker] Calling CF API:", url)

	resp, err := http.Get(url)
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "CF request failed: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("CF API error: HTTP %d", resp.StatusCode)
		if p.JobID != "" {
			updateJobError(p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return fmt.Errorf(msg)
	}

	var apiResp struct {
		Status string `json:"status"`
		Result []struct {
			Handle    string `json:"handle"`
			Rating    int    `json:"rating"`
			MaxRating int    `json:"maxRating"`
			Rank      string `json:"rank"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "JSON unmarshal error: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}

	if apiResp.Status != "OK" {
		msg := "CF API returned not OK"
		if p.JobID != "" {
			updateJobError(p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return fmt.Errorf(msg)
	}

	for _, u := range apiResp.Result {
		if err := repository.UpdatePastUserRating(u.Rating, u.MaxRating, u.Rank, u.Handle); err != nil {
			fmt.Printf("[worker] DB UPDATE FAILED: %s %v\n", u.Handle, err)
		} else {
			fmt.Printf("[worker] UPDATED: %s\n", u.Handle)
		}
	}

	fmt.Println("[worker] Rating refresh done")

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			SetJobState(p.JobID, state, 10*time.Minute)
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")
		
		// 2. PLACE THE LOCK RELEASE HERE: Standard cleanup before returning nil
		ReleaseActiveJobLock(p.JobID) 
	}

	return nil
}


// HandleCFCheckStatus checks if Codeforces API is alive (system.status).
func HandleCFCheckStatus(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit(start)

	var p CFCheckStatusPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Checking CF API status for JobID %s\n", p.JobID)

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			SetJobState(p.JobID, state, 10*time.Minute)
		}
	}

	resp, err := http.Get("https://codeforces.com/api/system.status")
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "Codeforces API unreachable")
			ReleaseActiveJobLock(p.JobID)
		}
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		if p.JobID != "" {
			updateJobError(p.JobID, fmt.Sprintf("CF returned non-200: HTTP %d", resp.StatusCode))
			ReleaseActiveJobLock(p.JobID)
		}
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "Invalid JSON from CF")
			ReleaseActiveJobLock(p.JobID)
		}
		return nil
	}

	if result["status"] != "OK" {
		if p.JobID != "" {
			updateJobError(p.JobID, "CF API status not OK")
			ReleaseActiveJobLock(p.JobID)
		}
		return nil
	}

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			SetJobState(p.JobID, state, 10*time.Minute)
		}
		ReleaseActiveJobLock(p.JobID)
	}

	return nil
}


// HandleCFAddContest fetches a single contest from CF standings API and adds it to DB.
func HandleCFAddContest(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit(start)

	var p CFAddContestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Adding contest CF#%s for JobID %s\n", p.CFContestID, p.JobID)

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			SetJobState(p.JobID, state, 10*time.Minute)
		}
	}

	resp, err := http.Get("https://codeforces.com/api/contest.standings?contestId=" + p.CFContestID)
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "Could not fetch contest info from Codeforces")
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Could not fetch contest info (HTTP %d)", resp.StatusCode)
		if p.JobID != "" {
			updateJobError(p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return fmt.Errorf(msg)
	}

	var apiResp struct {
		Status string `json:"status"`
		Result struct {
			Contest struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				StartTime int64  `json:"startTimeSeconds"`
			} `json:"contest"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || apiResp.Status != "OK" {
		msg := "Could not parse contest info from Codeforces"
		if p.JobID != "" {
			updateJobError(p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return fmt.Errorf("could not parse contest info")
	}

	err = repository.AddContest(apiResp.Result.Contest.Id, apiResp.Result.Contest.Name, apiResp.Result.Contest.StartTime)
	if err != nil {
		if p.JobID != "" {
			updateJobError(p.JobID, "Could not add contest: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			ReleaseActiveJobLock(p.JobID)
		}
		return err
	}

	fmt.Printf("[worker] Contest CF#%s added successfully: %s\n", p.CFContestID, apiResp.Result.Contest.Name)

	if p.JobID != "" {
		state, err := GetJobState(p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			SetJobState(p.JobID, state, 10*time.Minute)
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")
		ReleaseActiveJobLock(p.JobID)
	}

	return nil
}

// --- Helpers ---

func detectDivision(contestName string) string {
	if strings.Contains(contestName, "Div. 2") {
		return "Div. 2"
	} else if strings.Contains(contestName, "Div. 3") {
		return "Div. 3"
	} else if strings.Contains(contestName, "Div. 4") {
		return "Div. 4"
	}
	return "Div. 1"
}

// calculatePoints computes leaderboard points for a given rank in a contest
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