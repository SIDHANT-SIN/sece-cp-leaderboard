package database

import "log"

func CreateTables() {

	    // Create users table: stores tracked Codeforces users
    _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        codeforces_handle TEXT UNIQUE NOT NULL,
        display_name TEXT
    )`)
    if err != nil {
        log.Fatal("Failed to create users table:", err)
    }
	// create past user table : store all past users
   _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS past_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    codeforces_handle TEXT UNIQUE NOT NULL,
    display_name TEXT,
    batch_year INTEGER NOT NULL,
    current_rating INTEGER DEFAULT 0,
    max_rating INTEGER DEFAULT 0,
    title TEXT DEFAULT '',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`)
if err != nil {
    log.Fatal("Failed to create past_users table:", err)
}
    // Create contests table: stores relevant Codeforces contests
    _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS contests (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        codeforces_contest_id INTEGER UNIQUE NOT NULL,
        name TEXT,
        start_time INTEGER
    )`)
    if err != nil {
        log.Fatal("Failed to create contests table:", err)
    }

// Create icpc pyq 
	_, err = DB.Exec(`
CREATE TABLE IF NOT EXISTS icpc_pyq (
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
)
`)


if err != nil {
    log.Fatal("Failed to create icpc_pyq table:", err)
}


// create icpc testcase to hold links of azure blob links


_, err = DB.Exec(`
CREATE TABLE IF NOT EXISTS icpc_testcases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    problem_id INTEGER NOT NULL,

    testcase_input TEXT NOT NULL,
    testcase_output TEXT NOT NULL,

    FOREIGN KEY(problem_id) REFERENCES icpc_pyq(id)
)
`)
if err != nil {
    log.Fatal("Failed to create icpc_testcases table:", err)
}


    // Create user_contest_results table: stores each user's result in each contest
    _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS user_contest_results (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        contest_id INTEGER NOT NULL,
        rank INTEGER,
        points INTEGER,
        last_updated INTEGER,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ,
        FOREIGN KEY(contest_id) REFERENCES contests(id) ON DELETE CASCADE,
        UNIQUE(user_id, contest_id)
    )`)
    if err != nil {
        log.Fatal("Failed to create user_contest_results table:", err)
    }


// Create ICPC regional/prelims problem links
_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS problems (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    contest_name TEXT NOT NULL,
    year INTEGER NOT NULL,

    title TEXT NOT NULL,

    link TEXT NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`)
if err != nil {
    log.Fatal("Failed to create problems table:", err)
}


    



	
}