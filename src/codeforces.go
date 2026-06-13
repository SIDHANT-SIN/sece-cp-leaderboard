package main

import (
	
	

	"encoding/json"
	"fmt"
    "io"
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	
	"strings"
	"time"
    "bytes"
	"os"

	
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)


type TestCase struct {
	Input string `json:"input"`
}

func ParseTestCases(raw string) ([]TestCase, error) {

	var tc []TestCase

	err := json.Unmarshal([]byte(raw), &tc)
	if err != nil {
		return nil, err
	}

	return tc, nil
}



type JDoodleRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Script       string `json:"script"`
	Stdin        string `json:"stdin"`
	Language     string `json:"language"`
	VersionIndex string `json:"versionIndex"`
	CompileOnly  bool   `json:"compileOnly"`
}

type JDoodleResponse struct {
	Output             string      `json:"output"`
	Error              interface{} `json:"error"`
	StatusCode         int         `json:"statusCode"`
	Memory             string      `json:"memory"`
	CPUTime            interface{} `json:"cpuTime"`
	CompilationStatus  interface{} `json:"compilationStatus"`
	IsExecutionSuccess bool        `json:"isExecutionSuccess"`
	IsCompiled         bool        `json:"isCompiled"`
}

func RunSolution(code string, input string) (string, error) {

	reqBody := JDoodleRequest{
		ClientID:     os.Getenv("JDOODLE_CLIENT_ID"),
		ClientSecret: os.Getenv("JDOODLE_CLIENT_SECRET"),
		Script:       code,
		Stdin:        input,
		Language:     "cpp17",
		VersionIndex: "1",
		CompileOnly:  false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(
		"https://api.jdoodle.com/v1/execute",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var result JDoodleResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if result.StatusCode != 200 {
		return "", fmt.Errorf(
			"jdoodle returned status %d",
			result.StatusCode,
		)
	}

	return strings.TrimSpace(result.Output), nil
}









// Fetch contests from Codeforces group and update DB
func fetchAndStoreContests() error {
	resp, err := http.Get("https://codeforces.com/api/contest.list?groupCode=wontreveal")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Status string `json:"status"`
		Result []struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			StartTime int64  `json:"startTimeSeconds"`
			Phase     string `json:"phase"`
			Type      string `json:"type"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.Status != "OK" {
		return err
	}
	for _, c := range result.Result {
		if c.Phase == "FINISHED" && c.Type == "CF" {
			_, err := db.Exec("INSERT OR IGNORE INTO contests (codeforces_contest_id, name, start_time) VALUES (?, ?, ?)", c.Id, c.Name, c.StartTime)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func waitForCFLimit(start time.Time) {
	elapsed := time.Since(start)

	if elapsed < 2*time.Second {
		time.Sleep(2*time.Second - elapsed)
	}
}

func checkCFAPI(c *gin.Context) {
	start := time.Now()

	url := "https://codeforces.com/api/system.status"

	resp, err := http.Get(url)
	if err != nil {
		waitForCFLimit(start)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Codeforces API unreachable",
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		waitForCFLimit(start)
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": "CF returned non-200",
			"http":    resp.StatusCode,
			"body":    string(body),
		})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		waitForCFLimit(start)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Invalid JSON from CF",
		})
		return
	}

	if result["status"] != "OK" {
		waitForCFLimit(start)
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": "CF API status not OK",
		})
		return
	}

	waitForCFLimit(start)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Codeforces API is alive",
	})
}

// Calculate points for a given rank
func calculatePoints(rank, total int, div string) int {
	if total == 0 || rank == 0 {
		return 0
	}
	var d float64
	switch div {
	case "Div. 2", "Div. 1":
		d = 1.0
	case "Div. 3":
		d = 0.67
	case "Div. 4":
		d = 0.33
	default:
		d = 1.0
	}
	baseParticipation := 2
	score := int(math.Max(10*d*math.Log10(float64(total+1)/float64(rank+1)), 0)) + baseParticipation
	return score
}


// manual checks
func refreshAllUserContestResults() error {

	fmt.Println("\n================ REFRESH STARTED ================")

	// Get all users
	userRows, err := db.Query("SELECT id, codeforces_handle FROM users")
	if err != nil {
		fmt.Println("ERROR loading users:", err)
		return err
	}
	defer userRows.Close()

	var users []struct {
		ID     int
		Handle string
	}

	for userRows.Next() {
		var id int
		var handle string

		if err := userRows.Scan(&id, &handle); err != nil {
			fmt.Println("User scan error:", err)
			continue
		}

		users = append(users, struct {
			ID     int
			Handle string
		}{
			ID:     id,
			Handle: handle,
		})
	}

	fmt.Println("Users loaded:", len(users))

	// Get all contests
	contestRows, err := db.Query(
		"SELECT id, codeforces_contest_id FROM contests",
	)
	if err != nil {
		fmt.Println("ERROR loading contests:", err)
		return err
	}
	defer contestRows.Close()

	var contests []struct {
		ID   int
		CFID int
	}

	for contestRows.Next() {
		var id, cfid int

		if err := contestRows.Scan(&id, &cfid); err != nil {
			fmt.Println("Contest scan error:", err)
			continue
		}

		contests = append(contests, struct {
			ID   int
			CFID int
		}{
			ID:   id,
			CFID: cfid,
		})
	}

	fmt.Println("Contests loaded:", len(contests))

	for _, contest := range contests {

		fmt.Println("\n--------------------------------")
		fmt.Println("Processing contest:", contest.CFID)

		url := "https://codeforces.com/api/contest.standings?contestId=" +
			fmt.Sprint(contest.CFID)

		fmt.Println("API URL:", url)

		time.Sleep(2 * time.Second)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("HTTP ERROR:", err)
			continue
		}

		fmt.Println("HTTP Status:", resp.StatusCode)

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Println("Skipping contest because status != 200")
			continue
		}

		var standings struct {
			Status string `json:"status"`
			Result struct {
				Contest struct {
					Name string `json:"name"`
				} `json:"contest"`

				Rows []struct {
					Party struct {
						Members []struct {
							Handle string `json:"handle"`
						} `json:"members"`
					} `json:"party"`

					Rank int `json:"rank"`
				} `json:"rows"`
			} `json:"result"`
		}

		err = json.NewDecoder(resp.Body).Decode(&standings)
		resp.Body.Close()

		if err != nil {
			fmt.Println("JSON Decode Error:", err)
			continue
		}

		fmt.Println("CF Status:", standings.Status)

		if standings.Status != "OK" {
			fmt.Println("CF returned FAILED")
			continue
		}

		fmt.Println("Contest Name:", standings.Result.Contest.Name)

		total := len(standings.Result.Rows)

		fmt.Println("Rows Returned:", total)

		if total == 0 {
			fmt.Println("No standings rows returned")
			continue
		}

		// Detect division
		div := "Div. 1"

		if strings.Contains(standings.Result.Contest.Name, "Div. 2") {
			div = "Div. 2"
		} else if strings.Contains(standings.Result.Contest.Name, "Div. 3") {
			div = "Div. 3"
		} else if strings.Contains(standings.Result.Contest.Name, "Div. 4") {
			div = "Div. 4"
		}

		fmt.Println("Division:", div)

		// Build handle -> rank map
		rankMap := make(map[string]int)

		for _, row := range standings.Result.Rows {
			for _, member := range row.Party.Members {
				rankMap[member.Handle] = row.Rank
			}
		}

		fmt.Println("RankMap Size:", len(rankMap))

		// Print a few samples
		sample := 0
		for h, r := range rankMap {
			fmt.Printf("Sample => %s rank=%d\n", h, r)
			sample++
			if sample == 3 {
				break
			}
		}

		matchCount := 0

		for _, user := range users {

			userRank := rankMap[user.Handle]

			points := 0
			if userRank > 0 {
				points = calculatePoints(userRank, total, div)
				matchCount++
			}

			_, err = db.Exec(
				`INSERT OR REPLACE INTO user_contest_results
				(user_id, contest_id, rank, points, last_updated)
				VALUES (?, ?, ?, ?, strftime('%s','now'))`,
				user.ID,
				contest.ID,
				userRank,
				points,
			)

			if err != nil {
				fmt.Printf(
					"DB INSERT ERROR user=%s contest=%d err=%v\n",
					user.Handle,
					contest.CFID,
					err,
				)
			}
		}

		fmt.Println(
			"Contest processed successfully. Matches found:",
			matchCount,
		)
	}

	fmt.Printf("================ REFRESH ENDED ================\n")

	return nil
}



// Helper to get users for admin.tmpl
func getUsersList() []map[string]interface{} {
	rows, err := db.Query("SELECT codeforces_handle, display_name FROM users")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var users []map[string]interface{}
	for rows.Next() {
		var handle, displayName string
		rows.Scan(&handle, &displayName)
		users = append(users, map[string]interface{}{"Username": handle, "DisplayName": displayName})
	}
	return users
}