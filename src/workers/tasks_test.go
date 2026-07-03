package workers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestNewCFRatingChangesTask(t *testing.T) {
	tests := []struct {
		name        string
		jobID       string
		contestDBID int
		cfContestID int
	}{
		{
			name:        "success with all fields",
			jobID:       "job_abc123",
			contestDBID: 42,
			cfContestID: 2085,
		},
		{
			name:        "empty jobID still creates task",
			jobID:       "",
			contestDBID: 1,
			cfContestID: 999,
		},
		{
			name:        "zero IDs",
			jobID:       "job_zero",
			contestDBID: 0,
			cfContestID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewCFRatingChangesTask(tt.jobID, tt.contestDBID, tt.cfContestID)

			require.NoError(t, err)
			require.NotNil(t, task)

			assert.Equal(t, TypeCFRatingChanges, task.Type())

			var payload CFRatingChangesPayload
			err = json.Unmarshal(task.Payload(), &payload)
			require.NoError(t, err)

			assert.Equal(t, tt.jobID, payload.JobID)
			assert.Equal(t, tt.contestDBID, payload.ContestDBID)
			assert.Equal(t, tt.cfContestID, payload.CFContestID)
		})
	}
}

func TestNewCFRefreshRatingTask(t *testing.T) {
	tests := []struct {
		name  string
		jobID string
	}{
		{name: "normal jobID", jobID: "rating_refresh_1234567890"},
		{name: "empty jobID", jobID: ""},
		{name: "cron jobID", jobID: "cron_refresh_rating"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewCFRefreshRatingTask(tt.jobID)

			require.NoError(t, err)
			require.NotNil(t, task)

			assert.Equal(t, TypeCFRefreshRating, task.Type())

			var payload CFRefreshRatingPayload
			err = json.Unmarshal(task.Payload(), &payload)
			require.NoError(t, err)

			assert.Equal(t, tt.jobID, payload.JobID)
		})
	}
}


func TestNewCFCheckStatusTask(t *testing.T) {
	tests := []struct {
		name  string
		jobID string
	}{
		{name: "normal jobID", jobID: "check_status_abc"},
		{name: "empty jobID", jobID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewCFCheckStatusTask(tt.jobID)

			require.NoError(t, err)
			require.NotNil(t, task)

			assert.Equal(t, TypeCFCheckStatus, task.Type())

			var payload CFCheckStatusPayload
			err = json.Unmarshal(task.Payload(), &payload)
			require.NoError(t, err)

			assert.Equal(t, tt.jobID, payload.JobID)
		})
	}
}

func TestNewCFAddContestTask(t *testing.T) {
	tests := []struct {
		name        string
		jobID       string
		cfContestID string
	}{
		{
			name:        "success with valid IDs",
			jobID:       "add_contest_2085_1234567890",
			cfContestID: "2085",
		},
		{
			name:        "empty jobID",
			jobID:       "",
			cfContestID: "100",
		},
		{
			name:        "empty cfContestID",
			jobID:       "job_x",
			cfContestID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewCFAddContestTask(tt.jobID, tt.cfContestID)

			require.NoError(t, err)
			require.NotNil(t, task)

			assert.Equal(t, TypeCFAddContest, task.Type())

			var payload CFAddContestPayload
			err = json.Unmarshal(task.Payload(), &payload)
			require.NoError(t, err)

			assert.Equal(t, tt.jobID, payload.JobID)
			assert.Equal(t, tt.cfContestID, payload.CFContestID)
		})
	}
}

func TestNewCFBatchRefreshTask(t *testing.T) {
	tests := []struct {
		name  string
		jobID string
	}{
		{name: "normal batch jobID", jobID: "batch_refresh_1234567890"},
		{name: "empty jobID", jobID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewCFBatchRefreshTask(tt.jobID)

			require.NoError(t, err)
			require.NotNil(t, task)

			assert.Equal(t, TypeCFBatchRefresh, task.Type())

			var payload CFBatchRefreshPayload
			err = json.Unmarshal(task.Payload(), &payload)
			require.NoError(t, err)

			assert.Equal(t, tt.jobID, payload.JobID)
		})
	}
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "cf:rating_changes", TypeCFRatingChanges)
	assert.Equal(t, "cf:refresh_rating", TypeCFRefreshRating)
	assert.Equal(t, "cf:check_status", TypeCFCheckStatus)
	assert.Equal(t, "cf:add_contest", TypeCFAddContest)
	assert.Equal(t, "cf:batch_refresh", TypeCFBatchRefresh)
}
