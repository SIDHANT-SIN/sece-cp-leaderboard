package handles

import "net/http"

type RowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}

type Repository interface {
	// Users
	GetUsers() (RowScanner, error)
	AddUser(handle, displayName string) error
	DeleteUser(id string) error
	GetUsersList() []map[string]interface{}

	// Auth
	GetRecentSyncHistory(limit int) (RowScanner, error)

	// Leaderboard
	GetLeaderboardCache() ([]map[string]interface{}, []map[string]interface{}, map[int]map[int]map[string]interface{}, map[int]int, error)
	SetLeaderboardCache([]map[string]interface{}, []map[string]interface{}, map[int]map[int]map[string]interface{}, map[int]int) error

	// Contests
	GetContests() (RowScanner, error)

	// Results
	GetAllResults() (RowScanner, error)

	// Past leaderboard
	GetPastUsersByBatch(batch string) (RowScanner, error)

	// Maintainer users
	GetPastUsers() (RowScanner, error)
	AddPastUser(handle, name string, batch int) error
	DeletePastUser(id string) error

	// Sync
	GetCurrentSyncStatus() (map[string]interface{}, error)
	GetPastUserHandles() ([]string, error)
	CreateSyncLog(jobID string, count int) error

	DeleteContest(id string) error

	DeleteResultsByContest(id string) error

	SetSyncCancelSignal() error

	GetProblemsNew() (RowScanner, error)
}

type CacheBuilder interface {
	RebuildLeaderboardCache() error
}

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}
