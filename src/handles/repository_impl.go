package handles

import "leaderboard/src/repository"

type DBRepository struct{}

func (*DBRepository) GetUsers() (RowScanner, error) {
	return repository.GetUsers()
}

func (*DBRepository) AddUser(h, d string) error {
	return repository.AddUser(h, d)
}

func (*DBRepository) DeleteUser(id string) error {
	return repository.DeleteUser(id)
}

func (*DBRepository) GetUsersList() []map[string]interface{} {
	return repository.GetUsersList()
}

func (*DBRepository) GetRecentSyncHistory(limit int) (RowScanner, error) {
	return repository.GetRecentSyncHistory(limit)
}

func (*DBRepository) GetLeaderboardCache() ([]map[string]interface{}, []map[string]interface{}, map[int]map[int]map[string]interface{}, map[int]int, error) {
	return repository.GetLeaderboardCache()
}

func (*DBRepository) SetLeaderboardCache(
	users []map[string]interface{},
	contests []map[string]interface{},
	results map[int]map[int]map[string]interface{},
	totals map[int]int,
) error {
	return repository.SetLeaderboardCache(users, contests, results, totals)
}

func (*DBRepository) GetContests() (RowScanner, error) {
	return repository.GetContests()
}

func (*DBRepository) GetAllResults() (RowScanner, error) {
	return repository.GetAllResults()
}

func (*DBRepository) GetPastUsersByBatch(batch string) (RowScanner, error) {
	return repository.GetPastUsersByBatch(batch)
}

func (*DBRepository) GetPastUsers() (RowScanner, error) {
	return repository.GetPastUsers()
}

func (*DBRepository) AddPastUser(handle, name string, batch int) error {
	return repository.AddPastUser(handle, name, batch)
}

func (*DBRepository) DeletePastUser(id string) error {
	return repository.DeletePastUser(id)
}

func (*DBRepository) GetCurrentSyncStatus() (map[string]interface{}, error) {
	return repository.GetCurrentSyncStatus()
}

func (*DBRepository) GetPastUserHandles() ([]string, error) {
	return repository.GetPastUserHandles()
}

func (*DBRepository) CreateSyncLog(jobID string, count int) error {
	return repository.CreateSyncLog(jobID, count)
}

func (*DBRepository) DeleteContest(id string) error {
	return repository.DeleteContest(id)
}

func (*DBRepository) DeleteResultsByContest(id string) error {
	return repository.DeleteResultsByContest(id)
}

func (*DBRepository) SetSyncCancelSignal() error {
	return repository.SetSyncCancelSignal()
}

func (*DBRepository) GetProblemsNew() (RowScanner, error) {
	return repository.GetProblemsNew()
}
