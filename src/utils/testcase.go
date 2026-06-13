package utils

import "encoding/json"

type TestCase struct {
	Input string `json:"input"`
}

func ParseTestCases(data string) ([]TestCase, error) {
	var tc []TestCase
	err := json.Unmarshal([]byte(data), &tc)
	return tc, err
}