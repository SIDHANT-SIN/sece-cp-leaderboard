package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"leaderboard/src/database"
	"leaderboard/src/repository"

	"github.com/hibiken/asynq"
)

// CFBaseURL is extracted so tests can override it with a mock local server
var CFBaseURL = "https://codeforces.com/api"

const cfRateLimit = 2100 * time.Millisecond
const cfRateLimit2 = 10 * time.Millisecond

type userEntry struct {
	ID     int
	Handle string
}

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

// updateJobError updated to accept context and redis client
func updateJobError(ctx context.Context, jobID, msg string) {
	if jobID == "" {
		return
	}
	state, err := GetJobState(ctx, database.RedisClient, jobID)
	if err == nil && state != nil {
		state.Status = "failed"
		state.Error = msg
		state.CompletedAt = time.Now().Format(time.RFC3339)
		if err := SetJobState(ctx, database.RedisClient, jobID, state, 10*time.Minute); err != nil {
			log.Printf("failed to set job state: %v", err)
		}
	}
}

// httpClient safely defines timeouts for external requests
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// processSingleContestStandings refactored to use CFBaseURL and httpClient
func processSingleContestStandings(cfContestID, contestDBID int, users []userEntry) error {
	url := fmt.Sprintf("%s/contest.ratingChanges?contestId=%d", CFBaseURL, cfContestID)

	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

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

func HandleCFRatingChanges(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit2(start)

	var p CFRatingChangesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Processing single contest CF#%d\n", p.CFContestID)

	if p.JobID != "" {
		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}

		_ = database.RedisClient.Set(ctx, "sync:job_id", p.JobID, 30*time.Minute).Err()
		_ = database.RedisClient.Set(ctx, "sync:status", "processing", 30*time.Minute).Err()
		_ = database.RedisClient.Set(ctx, "sync:total", 1, 30*time.Minute).Err()
		_ = database.RedisClient.Set(ctx, "sync:current", 0, 30*time.Minute).Err()
	}

	userRows, err := repository.GetUsers()
	if err != nil {
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, "failed to load users: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}
	defer func() {
		if err := userRows.Close(); err != nil {
			log.Printf("failed to close userRows: %v", err)
		}
	}()

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
			updateJobError(ctx, p.JobID, err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}

	if p.JobID != "" {
		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")
		_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()
		if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
			log.Printf("failed to release active job lock: %v", err)
		}
	}

	return nil
}

func HandleCFBatchRefresh(ctx context.Context, t *asynq.Task) error {
	var p CFBatchRefreshPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Starting batch refresh for JobID %s\n", p.JobID)

	userRows, err := repository.GetUsers()
	if err != nil {
		updateJobError(ctx, p.JobID, "failed to load users: "+err.Error())
		if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
			log.Printf("failed to release active job lock: %v", err)
		}
		return err
	}
	defer func() {
		if err := userRows.Close(); err != nil {
			log.Printf("failed to close userRows: %v", err)
		}
	}()

	var users []userEntry
	for userRows.Next() {
		var id int
		var handle, display string
		if err := userRows.Scan(&id, &handle, &display); err == nil {
			users = append(users, userEntry{ID: id, Handle: handle})
		}
	}

	contestRows, err := repository.GetContests()
	if err != nil {
		updateJobError(ctx, p.JobID, "failed to load contests: "+err.Error())
		if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
			log.Printf("failed to release active job lock: %v", err)
		}
		return err
	}
	defer func() {
		if err := contestRows.Close(); err != nil {
			log.Printf("failed to close contestRows: %v", err)
		}
	}()

	type contestEntry struct {
		ID   int
		CFID int
	}
	var contests []contestEntry
	for contestRows.Next() {
		var id, cfid int
		var name string
		var startTime int64

		if err := contestRows.Scan(&id, &cfid, &name, &startTime); err == nil {
			contests = append(contests, contestEntry{ID: id, CFID: cfid})
		}
	}

	limitStr, err := database.RedisClient.Get(ctx, fmt.Sprintf("sync_limit:%s", p.JobID)).Result()
	if err == nil && limitStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		if limit > 0 && limit <= len(contests) {
			contests = contests[:limit]
		}
		_ = database.RedisClient.Del(ctx, fmt.Sprintf("sync_limit:%s", p.JobID)).Err()
	}

	total := len(contests)
	successful := 0

	state, err := GetJobState(ctx, database.RedisClient, p.JobID)
	if err == nil && state != nil {
		state.Total = total
		state.Current = 0
		if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
			log.Printf("failed to set job state: %v", err)
		}
	}

	_ = database.RedisClient.Set(ctx, "sync:job_id", p.JobID, 30*time.Minute).Err()
	_ = database.RedisClient.Set(ctx, "sync:status", "processing", 30*time.Minute).Err()
	_ = database.RedisClient.Set(ctx, "sync:total", total, 30*time.Minute).Err()
	_ = database.RedisClient.Set(ctx, "sync:current", 0, 30*time.Minute).Err()

	for idx, contest := range contests {
		select {
		case <-ctx.Done():
			fmt.Printf("[worker] Batch refresh JobID %s cancelled mid-way via context\n", p.JobID)
			if state, errState := GetJobState(ctx, database.RedisClient, p.JobID); errState == nil && state != nil {
				state.Status = "cancelled"
				state.CompletedAt = time.Now().Format(time.RFC3339)
				if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
					log.Printf("failed to set job state: %v", err)
				}
			}

			failedList, _ := GetFailedContests(ctx, database.RedisClient, p.JobID)
			failedJSON := "[]"
			if len(failedList) > 0 {
				bytes, _ := json.Marshal(failedList)
				failedJSON = string(bytes)
			}
			_ = repository.UpdateSyncLog(p.JobID, "cancelled", successful, failedJSON)
			if err := ClearFailedContests(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to clear failed contests: %v", err)
			}

			_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
			return fmt.Errorf("task cancelled via context: %w", asynq.SkipRetry)

		default:
			cancelSignal, _ := database.RedisClient.Get(ctx, "sync:cancel_signal").Result()
			if cancelSignal == "1" {
				fmt.Printf("[worker] Batch refresh JobID %s manually aborted via Redis signal\n", p.JobID)
				_ = database.RedisClient.Del(ctx, "sync:cancel_signal").Err()

				if state, errState := GetJobState(ctx, database.RedisClient, p.JobID); errState == nil && state != nil {
					state.Status = "cancelled"
					state.CompletedAt = time.Now().Format(time.RFC3339)
					if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
						log.Printf("failed to set job state: %v", err)
					}
				}

				failedList, _ := GetFailedContests(ctx, database.RedisClient, p.JobID)
				failedJSON := "[]"
				if len(failedList) > 0 {
					bytes, _ := json.Marshal(failedList)
					failedJSON = string(bytes)
				}
				_ = repository.UpdateSyncLog(p.JobID, "cancelled", successful, failedJSON)
				if err := ClearFailedContests(ctx, database.RedisClient, p.JobID); err != nil {
					log.Printf("failed to clear failed contests: %v", err)
				}

				_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()
				if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
					log.Printf("failed to release active job lock: %v", err)
				}
				return fmt.Errorf("batch sync cancelled by admin request: %w", asynq.SkipRetry)
			}
		}

		start := time.Now()
		fmt.Printf("[worker] JobID %s: Processing contest %d/%d (CF#%d)\n", p.JobID, idx+1, total, contest.CFID)

		err = processSingleContestStandings(contest.CFID, contest.ID, users)
		if err != nil {
			fmt.Printf("[worker] Failed CF#%d: %v\n", contest.CFID, err)
			if err := AppendFailedContest(ctx, database.RedisClient, p.JobID, fmt.Sprintf("CF#%d: %v", contest.CFID, err)); err != nil {
				log.Printf("failed to append failed contest: %v", err)
			}
		} else {
			successful++
		}

		state, errState := GetJobState(ctx, database.RedisClient, p.JobID)
		if errState == nil && state != nil {
			state.Current = idx + 1
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}

		_ = database.RedisClient.Set(ctx, "sync:current", idx+1, 30*time.Minute).Err()

		waitForCFRateLimit2(start)
	}

	fmt.Printf("[worker] Batch refresh JobID %s finished. Successful: %d/%d\n", p.JobID, successful, total)

	failedList, _ := GetFailedContests(ctx, database.RedisClient, p.JobID)
	failedJSON := "[]"
	if len(failedList) > 0 {
		bytes, _ := json.Marshal(failedList)
		failedJSON = string(bytes)
	}

	_ = repository.UpdateSyncLog(p.JobID, "completed", successful, failedJSON)
	if err := ClearFailedContests(ctx, database.RedisClient, p.JobID); err != nil {
		log.Printf("failed to clear failed contests: %v", err)
	}

	state, errState := GetJobState(ctx, database.RedisClient, p.JobID)
	if errState == nil && state != nil {
		state.Status = "completed"
		state.CompletedAt = time.Now().Format(time.RFC3339)
		if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
			log.Printf("failed to set job state: %v", err)
		}
	}

	_ = database.RedisClient.Del(ctx, "sync:job_id", "sync:status", "sync:current", "sync:total").Err()

	if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
		log.Printf("failed to release active job lock: %v", err)
	}
	return nil
}

func HandleCFRefreshRating(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit2(start)

	var p CFRefreshRatingPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Refreshing past user ratings for JobID %s\n", p.JobID)

	if p.JobID != "" {
		if p.JobID == "cron_refresh_rating" {
			_ = repository.CreateSyncLog(p.JobID, 1)
		}

		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}
	}

	handles, err := repository.GetPastUserHandles()
	if err != nil {
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, "DB error: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}

	if len(handles) == 0 {
		fmt.Println("[worker] No past users to refresh")
		if p.JobID != "" {
			state, err := GetJobState(ctx, database.RedisClient, p.JobID)
			if err == nil && state != nil {
				state.Current = 1
				state.Status = "completed"
				state.CompletedAt = time.Now().Format(time.RFC3339)
				if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
					log.Printf("failed to set job state: %v", err)
				}
			}
			_ = repository.UpdateSyncLog(p.JobID, "completed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return nil
	}

	handleStr := strings.Join(handles, ";")
	url := fmt.Sprintf("%s/user.info?handles=%s", CFBaseURL, handleStr)
	fmt.Println("[worker] Calling CF API:", url)

	resp, err := httpClient.Get(url)
	if err != nil {
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, "CF request failed: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("CF API error: HTTP %d", resp.StatusCode)
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return fmt.Errorf("error %v", msg)
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
			updateJobError(ctx, p.JobID, "JSON unmarshal error: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}

	if apiResp.Status != "OK" {
		msg := "CF API returned not OK"
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return fmt.Errorf("error %v", msg)
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
		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")

		if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
			log.Printf("failed to release active job lock: %v", err)
		}
	}

	return nil
}

func HandleCFAddContest(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	defer waitForCFRateLimit(start)

	var p CFAddContestPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Printf("[worker] Adding contest CF#%s for JobID %s\n", p.CFContestID, p.JobID)

	if p.JobID != "" {
		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Total = 1
			state.Current = 0
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}
	}

	url := fmt.Sprintf("%s/contest.standings?contestId=%s", CFBaseURL, p.CFContestID)
	resp, err := httpClient.Get(url)
	if err != nil {
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, "Could not fetch contest info from Codeforces")
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Could not fetch contest info (HTTP %d)", resp.StatusCode)
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return fmt.Errorf("error %v", msg)
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
			updateJobError(ctx, p.JobID, msg)
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return fmt.Errorf("could not parse contest info")
	}

	err = repository.AddContest(apiResp.Result.Contest.Id, apiResp.Result.Contest.Name, apiResp.Result.Contest.StartTime)
	if err != nil {
		if p.JobID != "" {
			updateJobError(ctx, p.JobID, "Could not add contest: "+err.Error())
			_ = repository.UpdateSyncLog(p.JobID, "failed", 0, "[]")
			if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
				log.Printf("failed to release active job lock: %v", err)
			}
		}
		return err
	}

	fmt.Printf("[worker] Contest CF#%s added successfully: %s\n", p.CFContestID, apiResp.Result.Contest.Name)

	if p.JobID != "" {
		state, err := GetJobState(ctx, database.RedisClient, p.JobID)
		if err == nil && state != nil {
			state.Current = 1
			state.Status = "completed"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			if err := SetJobState(ctx, database.RedisClient, p.JobID, state, 10*time.Minute); err != nil {
				log.Printf("failed to set job state: %v", err)
			}
		}
		_ = repository.UpdateSyncLog(p.JobID, "completed", 1, "[]")
		if _, err := ReleaseActiveJobLock(ctx, database.RedisClient, p.JobID); err != nil {
			log.Printf("failed to release active job lock: %v", err)
		}
	}

	return nil
}

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
