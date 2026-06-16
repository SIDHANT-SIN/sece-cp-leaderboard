package handles

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

// ShowContests lists all contests
func ShowContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := repository.GetContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

	var contests []map[string]interface{}
	for rows.Next() {
		var id, cfid, startTime int
		var name string
		if err := rows.Scan(&id, &cfid, &name, &startTime); err != nil {
			c.String(http.StatusInternalServerError, "DB scan error: %v", err)
			return
		}
		contests = append(contests, map[string]interface{}{
			"id":         id,
			"cfid":       cfid,
			"name":       name,
			"start_time": startTime,
		})
	}

	c.HTML(http.StatusOK, "admin_contests.tmpl", gin.H{"contests": contests})
}

// AddContest handles adding a single contest using Codeforces API
func AddContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	cfid := c.PostForm("cfid")

	resp, err := http.Get("https://codeforces.com/api/contest.standings?contestId=" + cfid)
	if err != nil {
		fmt.Println("HTTP ERROR:", err)
		c.String(http.StatusBadRequest, "Could not fetch contest info from Codeforces")
		return
	}
	defer resp.Body.Close()

	fmt.Println("Contest ID:", cfid)
	fmt.Println("Status Code:", resp.StatusCode)

	if resp.StatusCode != 200 {
		if resp.StatusCode >= 500 {
			c.String(
				http.StatusBadGateway,
				"Codeforces API server is currently unavailable (HTTP %d). Try later after a few minutes or hours.",
				resp.StatusCode,
			)
			return
		}

		c.String(
			http.StatusBadRequest,
			"Could not fetch contest info from Codeforces (HTTP %d)",
			resp.StatusCode,
		)
		return
	}

	var apiResp struct {
		Status string `json:"status"`
		Result struct {
			Contest struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				StartTime int64  `json:"startTimeSeconds"`
			} `json:"contest"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || apiResp.Status != "OK" {
		c.String(http.StatusBadRequest, "Could not parse contest info from Codeforces")
		return
	}

	err = repository.AddContest(apiResp.Result.Contest.Id, apiResp.Result.Contest.Name, apiResp.Result.Contest.StartTime)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not add contest: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// DeleteContest deletes a contest and its results
func DeleteContest(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	id := c.PostForm("id")

	// Delete all results associated with this contest first
	err = repository.DeleteResultsByContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest results: %v", err)
		return
	}

	// Delete the contest itself
	err = repository.DeleteContest(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete contest: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// DeleteAllContests deletes all contests and all user results
func DeleteAllContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	err = repository.DeleteAllResults()
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not delete contest results: %v", err)
		return
	}

	err = repository.DeleteAllContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not delete all contests: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// FetchContests fetches all contests from a Codeforces group
func FetchContests(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	err = fetchAndStoreContests()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch contests: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
}

// RefreshResults triggers recalculation of scores/ranks for all contests
func RefreshResults(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	err = refreshAllUserContestResults()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to refresh results: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/leaderboard")
}

// Helper to fetch from CF group API and store in DB
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
		return fmt.Errorf("codeforces status not OK")
	}

	for _, c := range result.Result {
		if c.Phase == "FINISHED" && c.Type == "CF" {
			err := repository.InsertContestIgnore(c.Id, c.Name, c.StartTime)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

// Refresh all user standings and point calculations
func refreshAllUserContestResults() error {
	fmt.Println("\n================ REFRESH STARTED ================")

	userRows, err := repository.GetUsers()
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
		var handle, display string
		if err := userRows.Scan(&id, &handle, &display); err != nil {
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

	contestRows, err := repository.GetContestIDs()
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

		url := "https://codeforces.com/api/contest.standings?contestId=" + fmt.Sprint(contest.CFID)
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

		matchCount := 0
		for _, user := range users {
			userRank := rankMap[user.Handle]
			points := 0
			if userRank > 0 {
				points = calculatePoints(userRank, total, div)
				matchCount++
			}

			err = repository.UpsertResult(user.ID, contest.ID, userRank, points)
			if err != nil {
				fmt.Printf("DB INSERT ERROR user=%s contest=%d err=%v\n", user.Handle, contest.CFID, err)
			}
		}

		fmt.Println("Contest processed successfully. Matches found:", matchCount)
	}

	fmt.Printf("================ REFRESH ENDED ================\n")
	return nil
}