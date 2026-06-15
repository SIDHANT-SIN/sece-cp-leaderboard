package database

import (
	"database/sql"
	"fmt"
	"log"

	"leaderboard/src/configs"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var DB *sql.DB

func Connect(cfg *configs.Config) {

	connStr := fmt.Sprintf(
		"%s?authToken=%s",
		cfg.DBUrl,
		cfg.AuthToken,
	)

	d, err := sql.Open("libsql", connStr)
	if err != nil {
		log.Fatal("Failed to open Turso database:", err)
	}

	DB = d
}