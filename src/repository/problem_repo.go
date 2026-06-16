package repository

import (
	"leaderboard/src/database"
)

// InsertProblem inserts an ICPC PYQ problem and returns the inserted problem's ID
func InsertProblem(title, statement, timeLimit, memoryLimit, inputDesc, outputDesc, constraints, sampleInput, sampleOutput, explanation string) (int64, error) {
	res, err := database.DB.Exec(`
		INSERT INTO icpc_pyq (
			title, statement,
			time_limit, memory_limit,
			input_desc, output_desc,
			constraints,
			sample_input, sample_output,
			explanation
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, title, statement, timeLimit, memoryLimit, inputDesc, outputDesc, constraints, sampleInput, sampleOutput, explanation)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
