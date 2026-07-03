//go:build smoke
// +build smoke

package database

import (
	"testing"

	"leaderboard/src/configs"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCreateTables_Success(t *testing.T) {
	cfg := &configs.Config{
		DBUrl:     "file::memory:",
		AuthToken: "dummy",
	}
	err := Connect(cfg)
	require.NoError(t, err)
	defer DB.Close()

	err = CreateTables()
	require.NoError(t, err)

	var name string
	err = DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&name)
	require.NoError(t, err)
	require.Equal(t, "users", name)
}

func TestCreateTables_ErrorPath(t *testing.T) {
	cfg := &configs.Config{
		DBUrl:     "file::memory:",
		AuthToken: "dummy",
	}
	err := Connect(cfg)
	require.NoError(t, err)

	DB.Close()

	err = CreateTables()
	require.Error(t, err)
}
