package database

import (
	"fmt"
)

func CreateTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT UNIQUE NOT NULL,
			display_name TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS past_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_handle TEXT UNIQUE NOT NULL,
			display_name TEXT,
			batch_year INTEGER NOT NULL,
			current_rating INTEGER DEFAULT 0,
			max_rating INTEGER DEFAULT 0,
			title TEXT DEFAULT '',
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sync_history (
			job_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			successful_contests INTEGER DEFAULT 0,
			total_contests INTEGER DEFAULT 0,
			failed_contest_ids TEXT DEFAULT '',
			started_at TEXT NOT NULL,
			completed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS contests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			codeforces_contest_id INTEGER UNIQUE NOT NULL,
			name TEXT,
			start_time INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS icpc_pyq (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			statement TEXT NOT NULL,
			time_limit INTEGER DEFAULT 1,
			memory_limit INTEGER DEFAULT 256,
			input_desc TEXT NOT NULL,
			output_desc TEXT NOT NULL,
			constraints TEXT,
			sample_input TEXT,
			sample_output TEXT,
			explanation TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS icpc_testcases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			problem_id INTEGER NOT NULL,
			testcase_input TEXT NOT NULL,
			testcase_output TEXT NOT NULL,
			FOREIGN KEY(problem_id) REFERENCES icpc_pyq(id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_contest_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			contest_id INTEGER NOT NULL,
			rank INTEGER,
			points INTEGER,
			last_updated INTEGER,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ,
			FOREIGN KEY(contest_id) REFERENCES contests(id) ON DELETE CASCADE,
			UNIQUE(user_id, contest_id)
		)`,
		`CREATE TABLE IF NOT EXISTS problems (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			contest_name TEXT NOT NULL,
			year INTEGER NOT NULL,
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}
