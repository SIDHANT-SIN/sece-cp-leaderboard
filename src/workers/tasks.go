package workers

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Task type constants — one per CF API endpoint
const (
	TypeCFRatingChanges = "cf:rating_changes" // contest.ratingChanges (per contest)
	TypeCFRefreshRating = "cf:refresh_rating" // user.info (batch all past user handles)
	TypeCFCheckStatus   = "cf:check_status"   // system.status
	
	TypeCFAddContest    = "cf:add_contest"    // contest.standings (single)

	TypeCFBatchRefresh  = "cf:batch_refresh"  // batch refresh all contests (loops through all contests)
)

// Note: Queue names (QueueCritical, QueueDefault, QueueLow) are defined in client.go

// --- Payloads ---

type CFRatingChangesPayload struct {
	JobID       string `json:"job_id"`
	ContestDBID int    `json:"contest_db_id"`
	CFContestID int    `json:"cf_contest_id"`
}

type CFRefreshRatingPayload struct {
	JobID string `json:"job_id"`
}

type CFCheckStatusPayload struct {
	JobID string `json:"job_id"`
}

type CFAddContestPayload struct {
	JobID       string `json:"job_id"`
	CFContestID string `json:"cf_contest_id"`
}

type CFBatchRefreshPayload struct {
	JobID string `json:"job_id"`
}

// --- Task constructors ---

func NewCFRatingChangesTask(jobID string, contestDBID, cfContestID int) (*asynq.Task, error) {
	payload, err := json.Marshal(CFRatingChangesPayload{
		JobID:       jobID,
		ContestDBID: contestDBID,
		CFContestID: cfContestID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCFRatingChanges, payload), nil
}

func NewCFRefreshRatingTask(jobID string) (*asynq.Task, error) {
	payload, err := json.Marshal(CFRefreshRatingPayload{
		JobID: jobID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCFRefreshRating, payload), nil
}

func NewCFCheckStatusTask(jobID string) (*asynq.Task, error) {
	payload, err := json.Marshal(CFCheckStatusPayload{
		JobID: jobID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCFCheckStatus, payload), nil
}

func NewCFAddContestTask(jobID, cfContestID string) (*asynq.Task, error) {
	payload, err := json.Marshal(CFAddContestPayload{
		JobID:       jobID,
		CFContestID: cfContestID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCFAddContest, payload), nil
}


func NewCFBatchRefreshTask(jobID string) (*asynq.Task, error) {
	payload, err := json.Marshal(CFBatchRefreshPayload{
		JobID: jobID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCFBatchRefresh, payload), nil
}