//go:build smoke
// +build smoke

package database

import (
	"testing"

	"leaderboard/src/configs"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestConnect_Smoke(t *testing.T) {
	cfg := &configs.Config{
		DBUrl:     "file::memory:",
		AuthToken: "smoke-test-dummy-token",
	}

	err := Connect(cfg)
	require.NoError(t, err)

	require.NotNil(t, DB)

	err = DB.Ping()
	require.NoError(t, err)

	err = DB.Close()
	require.NoError(t, err)
}
