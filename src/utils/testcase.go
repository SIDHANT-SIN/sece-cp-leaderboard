package utils

import "encoding/json"

type TestCase struct {
	Input string `json:"input"`
}

type Solution struct {
	Output string `json:"output"`
}

func ParseTestCases(data string) ([]TestCase, error) {
	var tc []TestCase
	err := json.Unmarshal([]byte(data), &tc)
	return tc, err
}

func ParseSolutions(data string) ([]Solution, error) {
	var sol []Solution
	err := json.Unmarshal([]byte(data), &sol)
	return sol, err
}