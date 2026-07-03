package handles

import (
	"errors"
	"net/http"
)

type MockRows struct {
	Data  [][]any
	Index int
	Err   error
}

func (m *MockRows) Next() bool {
	if m.Index < len(m.Data) {
		m.Index++
		return true
	}
	return false
}

func (m *MockRows) Scan(dest ...any) error {
	if m.Err != nil {
		return m.Err
	}

	row := m.Data[m.Index-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *int:

			switch v := row[i].(type) {
			case int:
				*d = v
			case int64:
				*d = int(v)
			}
		case *int64:
			switch v := row[i].(type) {
			case int:
				*d = int64(v)
			case int64:
				*d = v
			}
		case *string:
			*d = row[i].(string)
		}
	}
	return nil
}

func (m *MockRows) Close() error {
	return nil
}

type MockRepository struct {
	PastUsersRows *MockRows
	PastUsersErr  error

	AddPastUserErr error

	DeletePastUserErr error

	SyncStatus    map[string]interface{}
	SyncStatusErr error

	PastHandles    []string
	PastHandlesErr error

	CreateSyncLogErr error

	LeaderboardCacheUsers      []map[string]interface{}
	LeaderboardCacheContests   []map[string]interface{}
	LeaderboardCacheResults    map[int]map[int]map[string]interface{}
	LeaderboardCacheUserTotals map[int]int
	LeaderboardCacheErr        error

	SetLeaderboardCacheErr error

	UsersRows *MockRows
	UsersErr  error

	ContestsRows *MockRows
	ContestsErr  error

	AllResultsRows *MockRows
	AllResultsErr  error

	PastUsersByBatchRows *MockRows
	PastUsersByBatchErr  error

	SyncHistoryRows *MockRows
	SyncHistoryErr  error

	AddUserErr    error
	DeleteUserErr error
	UsersList     []map[string]interface{}

	DeleteContestErr          error
	DeleteResultsByContestErr error
	SetSyncCancelSignalErr    error

	GetProblemsNewRows *MockRows
	GetProblemsNewErr  error
}

func (m *MockRepository) GetPastUsers() (RowScanner, error) {
	return m.PastUsersRows, m.PastUsersErr
}

func (m *MockRepository) AddPastUser(handle, name string, batch int) error {
	return m.AddPastUserErr
}

func (m *MockRepository) DeletePastUser(id string) error {
	return m.DeletePastUserErr
}

func (m *MockRepository) GetCurrentSyncStatus() (map[string]interface{}, error) {
	return m.SyncStatus, m.SyncStatusErr
}

func (m *MockRepository) GetPastUserHandles() ([]string, error) {
	return m.PastHandles, m.PastHandlesErr
}

func (m *MockRepository) CreateSyncLog(jobID string, count int) error {
	return m.CreateSyncLogErr
}

func (m *MockRepository) AddUser(handle, displayName string) error {
	return m.AddUserErr
}

func (m *MockRepository) DeleteUser(id string) error {
	return m.DeleteUserErr
}

func (m *MockRepository) GetUsersList() []map[string]interface{} {
	return m.UsersList
}

func (m *MockRepository) GetRecentSyncHistory(limit int) (RowScanner, error) {
	return m.SyncHistoryRows, m.SyncHistoryErr
}

func (m *MockRepository) GetUsers() (RowScanner, error) {
	return m.UsersRows, m.UsersErr
}

func (m *MockRepository) GetProblemsNew() (RowScanner, error) {
	return m.GetProblemsNewRows, m.GetProblemsNewErr
}

func (m *MockRepository) GetLeaderboardCache() ([]map[string]interface{}, []map[string]interface{}, map[int]map[int]map[string]interface{}, map[int]int, error) {
	return m.LeaderboardCacheUsers, m.LeaderboardCacheContests, m.LeaderboardCacheResults, m.LeaderboardCacheUserTotals, m.LeaderboardCacheErr
}

func (m *MockRepository) SetLeaderboardCache(users []map[string]interface{}, contests []map[string]interface{}, results map[int]map[int]map[string]interface{}, totals map[int]int) error {
	return m.SetLeaderboardCacheErr
}

func (m *MockRepository) GetContests() (RowScanner, error) {
	return m.ContestsRows, m.ContestsErr
}

func (m *MockRepository) GetAllResults() (RowScanner, error) {
	return m.AllResultsRows, m.AllResultsErr
}

func (m *MockRepository) GetPastUsersByBatch(batch string) (RowScanner, error) {
	return m.PastUsersByBatchRows, m.PastUsersByBatchErr
}

func (m *MockRepository) DeleteContest(id string) error {
	return m.DeleteContestErr
}

func (m *MockRepository) DeleteResultsByContest(id string) error {
	return m.DeleteResultsByContestErr
}

func (m *MockRepository) SetSyncCancelSignal() error {
	return m.SetSyncCancelSignalErr
}

type MockTaskEnqueuer struct {
	RefreshErr error
}

func (m *MockTaskEnqueuer) EnqueueRefreshRatingTask(jobID string) error {
	return m.RefreshErr
}

func (*MockTaskEnqueuer) EnqueueAddContestTask(jobID, cfid string) error {
	return errors.New("not used")
}

func (*MockTaskEnqueuer) EnqueueBatchRefreshTask(jobID string) error {
	return errors.New("not used")
}

type MockCacheBuilder struct {
	Err error
}

func (m *MockCacheBuilder) RebuildLeaderboardCache() error {
	return m.Err
}

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (m *MockHTTPClient) Get(string) (*http.Response, error) {
	return m.Response, m.Err
}
