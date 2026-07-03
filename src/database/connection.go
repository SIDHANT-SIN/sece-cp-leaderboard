package database

import (
	"database/sql"
	"fmt"

	"leaderboard/src/configs"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var DB *sql.DB

func Connect(cfg *configs.Config) error {
	connStr := fmt.Sprintf(
		"%s?authToken=%s",
		cfg.DBUrl,
		cfg.AuthToken,
	)

	d, err := sql.Open("libsql", connStr)
	if err != nil {
		return fmt.Errorf("failed to open Turso database: %w", err)
	}

	DB = d
	return nil
}
