package main

import (
	"crypto/sha256"
    "leaderboard/src/storage"
	"leaderboard/src/utils"

	"encoding/hex"
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func setupRouter() *gin.Engine {
	admin_username := os.Getenv("ADMIN_USERNAME")
	admin_password_hash := os.Getenv("ADMIN_PASSWORD")
	maintainer_password := os.Getenv("MAINTAINER_PASSWORD")
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	//router.LoadHTMLFiles("templates/template1.html", "templates/template2.html")
	r.GET("/index", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})
	r.POST("/maintainer/login", func(c *gin.Context) {
    password := c.PostForm("password")

    hashp := sha256.Sum256([]byte(password))
if hex.EncodeToString(hashp[:]) == maintainer_password  {
        c.SetCookie("maintainer_logged_in", "true", 3600*24*2, "/", "", false, true)
        c.Redirect(http.StatusSeeOther, "/maintainer/dashboard")
        return
    }

    c.HTML(http.StatusUnauthorized, "maintainer_login.tmpl", gin.H{
        "error": "Invalid password",
    })
})
	r.GET("/admin", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")

		if err != nil || cookie != admin_password_hash {
			// Use 303 See Other for redirect, not 401
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		c.HTML(http.StatusOK, "admin.tmpl", nil)
	})
	r.GET("/maintainer/icpc_pyq", func(c *gin.Context) {
	cookie, err := c.Cookie("maintainer_logged_in")

	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.HTML(http.StatusOK, "maintainer_icpc.tmpl", nil)
})
	r.GET("/admin_login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin_login.tmpl", nil)
	})
	r.GET("/maintainer", func(c *gin.Context) {
    c.HTML(http.StatusOK, "maintainer_login.tmpl", nil)
})
r.GET("/maintainer/dashboard", func(c *gin.Context) {
    cookie, err := c.Cookie("maintainer_logged_in")

    if err != nil || cookie != "true" {
        c.Redirect(http.StatusSeeOther, "/maintainer")
        return
    }

    c.HTML(http.StatusOK, "maintainer_dashboard.tmpl", nil)
})
	r.GET("/logout", func(c *gin.Context) {
		c.SetCookie("admin_logged_in", "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/admin")
	})
    r.POST("/admin/check_cf_api", checkCFAPI)
	r.POST("/admin", func(c *gin.Context) {
		name := c.PostForm("username")
		password := c.PostForm("password")
		hashp := sha256.Sum256([]byte(password))
		if admin_password_hash == hex.EncodeToString(hashp[:]) && admin_username == name {
			c.SetCookie("admin_logged_in", hex.EncodeToString(hashp[:]), 3600*24*2, "/", "", false, true)
			c.Redirect(http.StatusSeeOther, "/admin")
		} else {
			c.HTML(http.StatusUnauthorized, "admin_login.tmpl", gin.H{"error": "Invalid credentials"})
		}
	})

	r.GET("/admin/users", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusUnauthorized, "/admin")
			return
		}
		rows, err := db.Query("SELECT id, codeforces_handle, display_name FROM users")
		if err != nil {
			c.String(http.StatusInternalServerError, "DB error")
			return
		}
		defer rows.Close()
		var users []map[string]interface{}
		for rows.Next() {
			var id int
			var handle, displayName string
			rows.Scan(&id, &handle, &displayName)
			users = append(users, map[string]interface{}{"id": id, "handle": handle, "display_name": displayName})
		}
		c.HTML(http.StatusOK, "admin_users.tmpl", gin.H{"users": users})
	})
	r.POST("/admin/users/add", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		handle := c.PostForm("handle")
		displayName := c.PostForm("display_name")
		resp, err := http.Get("https://codeforces.com/api/user.info?handles=" + handle)
		if err != nil || resp.StatusCode != 200 {
			c.HTML(http.StatusBadRequest, "admin.tmpl", gin.H{"Users": getUsersList(), "error": "Invalid Codeforces handle"})
			return
		}
		_, err = db.Exec("INSERT INTO users (codeforces_handle, display_name) VALUES (?, ?)", handle, displayName)
		if err != nil {
			c.HTML(http.StatusBadRequest, "admin.tmpl", gin.H{"Users": getUsersList(), "error": "Could not add user: " + err.Error()})
			return
		}
		c.Redirect(http.StatusSeeOther, "/admin")
	})

// 	


// without jdoodle

r.POST("/maintainer/icpc_pyq", func(c *gin.Context) {
	// --- AUTH CHECK ---
	cookie, err := c.Cookie("maintainer_logged_in")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	// --- 1. GATHER FORM DATA ---
	title := c.PostForm("title")
	statement := c.PostForm("statement")
	inputDesc := c.PostForm("input_desc")
	outputDesc := c.PostForm("output_desc")
	constraints := c.PostForm("constraints")
	sampleInput := c.PostForm("sample_input")
	sampleOutput := c.PostForm("sample_output")
	explanation := c.PostForm("explanation")
	
	timeLimit := c.PostForm("time_limit")
	memoryLimit := c.PostForm("memory_limit")

	if timeLimit == "" {
		timeLimit = "1"
	}
	if memoryLimit == "" {
		memoryLimit = "256"
	}

	// --- 2. PARSE TESTCASES (Inputs) ---
	testcaseJSON := c.PostForm("testcases")
	testcases, err := utils.ParseTestCases(testcaseJSON)
	if err != nil {
		c.String(400, "Invalid testcases JSON")
		return
	}

	// --- 3. PARSE SOLUTIONS (Outputs) ---
	solutionJSON := c.PostForm("solution_code")
	solutions, err := utils.ParseSolutions(solutionJSON)
	if err != nil {
		c.String(400, "Invalid solution JSON")
		return
	}

	// Safety check: Make sure no one messed up the form submission
	if len(testcases) != len(solutions) {
		c.String(400, "Mismatch: Number of inputs does not match number of outputs")
		return
	}

	// --- 4. INSERT PROBLEM INTO DB ---
	res, err := db.Exec(`
		INSERT INTO icpc_pyq (
			title, statement,
			time_limit, memory_limit,
			input_desc, output_desc,
			constraints,
			sample_input, sample_output,
			explanation
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		title, statement,
		timeLimit, memoryLimit,
		inputDesc, outputDesc,
		constraints,
		sampleInput, sampleOutput,
		explanation,
	)

	if err != nil {
		c.String(500, "DB insert failed")
		return
	}

	// Get the ID of the problem we just created so we can use it in our Azure paths
	problemID, _ := res.LastInsertId()
	//problemID := 67
  
	// fmt.Printf("RAW TESTCASE JSON FROM FRONTEND: %s\n", testcaseJSON)
	// fmt.Printf("PARSED GO STRUCT: %+v\n", testcases)
	// fmt.Printf("LENGTH OF FIRST UPLOAD BYTE ARRAY: %d\n", len([]byte(testcases[0].Input)))


	// --- 5. PROCESS TESTCASES ---
	for i := range testcases {

		//	fmt.Printf("UPLOAD BYTE ARRAY: %d\n", []byte(testcases[i].Input))
		
	//	Upload Input File to Azure
		inputPath := fmt.Sprintf("problems/%d/tc_%d/input.txt", problemID, i)
		inputURL, err := storage.UploadFile(inputPath, []byte(testcases[i].Input))
		if err != nil {
			c.String(500, fmt.Sprintf("Azure upload failed for input %d", i))
			return
		}

	//	Upload Output File to Azure
		outputPath := fmt.Sprintf("problems/%d/tc_%d/output.txt", problemID, i)
		outputURL, err := storage.UploadFile(outputPath, []byte(solutions[i].Output))
		if err != nil {
			c.String(500, fmt.Sprintf("Azure upload failed for output %d", i))
			return
		}


	//	fmt.Printf("hehe %s \n", outputURL);
	//	fmt.Printf("hehe %s \n", inputURL);

	//	Save both URLs to the database
		_, err = db.Exec(`
			INSERT INTO icpc_testcases (
				problem_id,
				testcase_input,
				testcase_output
			)
			VALUES (?, ?, ?)
		`,
			problemID,
			inputURL,
			outputURL,
		)

		if err != nil {
			c.String(500, "DB testcase insert failed")
			return
		}
	}

	// --- 6. REDIRECT ON SUCCESS ---
	c.Redirect(http.StatusSeeOther, "/maintainer/icpc_pyq")
})
	r.POST("/admin/users/delete", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusUnauthorized, "/admin")
			return
		}
		id := c.PostForm("id")
		_, err = db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			c.String(http.StatusBadRequest, "Could not delete user: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/admin/users")
	})


r.GET("/maintainer/users", func(c *gin.Context) {
    cookie, err := c.Cookie("maintainer_logged_in")
    if err != nil || cookie != "true" {
        c.Redirect(http.StatusSeeOther, "/maintainer")
        return
    }

    rows, err := db.Query(`
        SELECT id, codeforces_handle, display_name, batch_year, current_rating, max_rating, title
        FROM past_users
    `)
    if err != nil {
        c.String(http.StatusInternalServerError, "DB error")
        return
    }
    defer rows.Close()

    var past_users []map[string]interface{}

    for rows.Next() {
        var id, batch, cur, mx int
        var handle, name, title string

        rows.Scan(&id, &handle, &name, &batch, &cur, &mx, &title)

        past_users = append(past_users, map[string]interface{}{
            "id": id,
            "handle": handle,
            "display_name": name,
            "batch": batch,
            "current_rating": cur,
            "max_rating": mx,
            "title": title,
        })
    }

    c.HTML(http.StatusOK, "maintainer_users.tmpl", gin.H{
        "past_users": past_users,
    })
})


r.POST("/maintainer/users/add", func(c *gin.Context) {
    cookie, err := c.Cookie("maintainer_logged_in")
    if err != nil || cookie != "true" {
        c.Redirect(http.StatusSeeOther, "/maintainer")
        return
    }

    handle := c.PostForm("handle")
    display := c.PostForm("display_name")
    batch := c.PostForm("batch")

    _, err = db.Exec(`
        INSERT INTO past_users (codeforces_handle, display_name, batch_year)
        VALUES (?, ?, ?)
    `, handle, display, batch)

    if err != nil {
        c.String(http.StatusBadRequest, "Could not add user: %v", err)
        return
    }

    c.Redirect(http.StatusSeeOther, "/maintainer/users")
})


r.POST("/maintainer/users/delete", func(c *gin.Context) {
    cookie, err := c.Cookie("maintainer_logged_in")
    if err != nil || cookie != "true" {
        c.Redirect(http.StatusSeeOther, "/maintainer")
        return
    }

    id := c.PostForm("id")

    _, err = db.Exec(`DELETE FROM past_users WHERE id = ?`, id)
    if err != nil {
        c.String(http.StatusBadRequest, "Delete failed: %v", err)
        return
    }

    c.Redirect(http.StatusSeeOther, "/maintainer/users")
})




	// contest part


	r.GET("/admin/contests", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		rows, err := db.Query("SELECT id, codeforces_contest_id, name, start_time FROM contests ORDER BY start_time DESC")
		if err != nil {
			c.String(http.StatusInternalServerError, "DB error")
			return
		}
		defer rows.Close()
		var contests []map[string]interface{}
		for rows.Next() {
			var id, cfid, startTime int
			var name string
			rows.Scan(&id, &cfid, &name, &startTime)
			contests = append(contests, map[string]interface{}{"id": id, "cfid": cfid, "name": name, "start_time": startTime})
		}
		c.HTML(http.StatusOK, "admin_contests.tmpl", gin.H{"contests": contests})
	})
	r.POST("/admin/contests/add", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		cfid := c.PostForm("cfid")
	
resp, err := http.Get(
	"https://codeforces.com/api/contest.standings?contestId=" + cfid,
)

if err != nil {
	fmt.Println("HTTP ERROR:", err)
	c.String(http.StatusBadRequest,
		"Could not fetch contest info from Codeforces")
	return
}

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
		err = json.NewDecoder(resp.Body).Decode(&apiResp)

resp.Body.Close()


		fmt.Printf("%+v\n", apiResp)
		if err != nil || apiResp.Status != "OK" {
			c.String(http.StatusBadRequest, "Could not parse contest info from Codeforces")
			return
		}
		_, err = db.Exec("INSERT INTO contests (codeforces_contest_id, name, start_time) VALUES (?, ?, ?)", apiResp.Result.Contest.Id, apiResp.Result.Contest.Name, apiResp.Result.Contest.StartTime)
		if err != nil {
			c.String(http.StatusBadRequest, "Could not add contest: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/admin/contests")
	})
r.POST("/admin/contests/delete", func(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != admin_password_hash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	id := c.PostForm("id")

	// Delete all results associated with this contest first
	_, err = db.Exec(
		"DELETE FROM user_contest_results WHERE contest_id = ?",
		id,
	)
	if err != nil {
		c.String(http.StatusBadRequest,
			"Could not delete contest results: %v",
			err,
		)
		return
	}

	// Delete the contest itself
	_, err = db.Exec(
		"DELETE FROM contests WHERE id = ?",
		id,
	)
	if err != nil {
		c.String(http.StatusBadRequest,
			"Could not delete contest: %v",
			err,
		)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
})
r.POST("/admin/contests/delete_all", func(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != admin_password_hash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	_, err = db.Exec("DELETE FROM user_contest_results")
	if err != nil {
		c.String(http.StatusInternalServerError,
			"Could not delete contest results: %v",
			err)
		return
	}

	_, err = db.Exec("DELETE FROM contests")
	if err != nil {
		c.String(http.StatusInternalServerError,
			"Could not delete all contests: %v",
			err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/contests")
})



// leaderboards

r.GET("/past_leaderboard", func(c *gin.Context) {

	batch := c.Query("batch")
	if batch == "" {
		batch = "2023" // default
	}

	rows, err := db.Query(`
		SELECT id, codeforces_handle, display_name,
		       current_rating, max_rating, title, batch_year
		FROM past_users
		WHERE batch_year = ?
	`, batch)

	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

	var users []map[string]interface{}

	for rows.Next() {
		var id, cur, mx, by int
		var handle, name, title string

		rows.Scan(&id, &handle, &name, &cur, &mx, &title, &by)

		users = append(users, map[string]interface{}{
			"id": id,
			"handle": handle,
			"display_name": name,
			"current_rating": cur,
			"max_rating": mx,
			"title": title,
			"batch": by,
		})
	}

	// sort by max_rating DESC (IMPORTANT)
	sort.Slice(users, func(i, j int) bool {
		return users[i]["max_rating"].(int) > users[j]["max_rating"].(int)
	})

	// assign rank
	for i := range users {
		users[i]["rank"] = i + 1
	}

	c.HTML(http.StatusOK, "past_leaderboard.tmpl", gin.H{
		"users": users,
		"selectedBatch": batch,
	})
})

	r.GET("/leaderboard", func(c *gin.Context) {
		userRows, err := db.Query("SELECT id, codeforces_handle, display_name FROM users")
		if err != nil {
			c.String(http.StatusInternalServerError, "DB error")
			return
		}
		defer userRows.Close()
		var users []map[string]interface{}
		for userRows.Next() {
			var id int
			var handle, displayName string
			userRows.Scan(&id, &handle, &displayName)
			users = append(users, map[string]interface{}{"id": id, "handle": handle, "display_name": displayName})
		}
		contestRows, err := db.Query("SELECT id, codeforces_contest_id, name, start_time FROM contests ORDER BY start_time DESC")
		if err != nil {
			c.String(http.StatusInternalServerError, "DB error")
			return
		}
		defer contestRows.Close()
		var contests []map[string]interface{}
		for contestRows.Next() {
			var id, cfid, startTime int
			var name string
			contestRows.Scan(&id, &cfid, &name, &startTime)
			contests = append(contests, map[string]interface{}{"id": id, "cfid": cfid, "name": name, "start_time": startTime})
		}
		// Query results for each user in each contest
		results := make(map[int]map[int]map[string]interface{}) // user_id -> contest_id -> result
		userTotals := make(map[int]int)                         // user_id -> total points
		rows, err := db.Query("SELECT user_id, contest_id, rank, points FROM user_contest_results")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var userID, contestID, rank, points int
				rows.Scan(&userID, &contestID, &rank, &points)
				// Only sum points for contests that are currently in the DB
				contestExists := false
				for _, c := range contests {
					if c["id"].(int) == contestID {
						contestExists = true
						break
					}
				}
				if !contestExists {
					continue
				}
				if results[userID] == nil {
					results[userID] = make(map[int]map[string]interface{})
				}
				results[userID][contestID] = map[string]interface{}{"rank": rank, "points": points}
				userTotals[userID] += points
			}
		}
		// Sort users by total points descending
		type userWithTotal struct {
			User  map[string]interface{}
			Total int
		}
		var userList []userWithTotal
		for _, u := range users {
			uid := u["id"].(int)
			total := userTotals[uid]
			userList = append(userList, userWithTotal{User: u, Total: total})
		}
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].Total > userList[j].Total
		})
		// Assign ranks
		rankedUsers := make([]map[string]interface{}, len(userList))
		for i, ut := range userList {
			rankedUsers[i] = ut.User
			rankedUsers[i]["rank"] = i + 1
			rankedUsers[i]["total_points"] = ut.Total
		}
		c.HTML(http.StatusOK, "leaderboard.tmpl", gin.H{
			"users":      rankedUsers,
			"contests":   contests,
			"results":    results,
			"userTotals": userTotals,
		})
	})

	r.POST("/admin/contests/fetch", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		err = fetchAndStoreContests()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch contests: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/admin/contests")
	})





	// refresh 

r.POST("/maintainer/refresh_rating", func(c *gin.Context) {
    fmt.Println("STEP 0: endpoint hit")

    cookie, err := c.Cookie("maintainer_logged_in")
    if err != nil || cookie != "true" {
        fmt.Println("STEP 1 FAILED: cookie issue:", err, cookie)
        c.Redirect(http.StatusSeeOther, "/maintainer")
        return
    }
    fmt.Println("STEP 1 OK: cookie validated")

    // 1. fetch all handles
    rows, err := db.Query("SELECT codeforces_handle FROM past_users")
    if err != nil {
        fmt.Println("STEP 2 FAILED: DB query error:", err)
        c.String(http.StatusInternalServerError, "DB error")
        return
    }
    defer rows.Close()

    handles := []string{}
    for rows.Next() {
        var h string
        if err := rows.Scan(&h); err != nil {
            fmt.Println("ROW SCAN ERROR:", err)
            continue
        }
        handles = append(handles, h)
    }

    fmt.Println("STEP 2 OK: handles fetched =", len(handles), handles)

    if len(handles) == 0 {
        fmt.Println("STEP 2 EXIT: no users")
        c.String(http.StatusOK, "No users to refresh")
        return
    }

    // 2. build API request
    handleStr := strings.Join(handles, ";")
    url := "https://codeforces.com/api/user.info?handles=" + handleStr

    fmt.Println("STEP 3: calling CF API:", url)

    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("STEP 3 FAILED: CF request error:", err)
        c.String(http.StatusInternalServerError, "CF request failed")
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println("STEP 3 RESPONSE STATUS:", resp.StatusCode)
  //  fmt.Println("STEP 3 RESPONSE BODY:", string(body))

    if resp.StatusCode != 200 {
        fmt.Println("STEP 3 FAILED: bad status")
        c.String(http.StatusBadGateway, "CF API error: %s", string(body))
        return
    }

    // 3. parse response
    var apiResp struct {
        Status string `json:"status"`
        Result []struct {
            Handle    string `json:"handle"`
            Rating    int    `json:"rating"`
            MaxRating int    `json:"maxRating"`
            Rank      string `json:"rank"`
        } `json:"result"`
    }

    err = json.Unmarshal(body, &apiResp)
    if err != nil {
        fmt.Println("STEP 4 FAILED: JSON unmarshal error:", err)
        return
    }

    fmt.Println("STEP 4 OK: CF status =", apiResp.Status)

    if apiResp.Status != "OK" {
        fmt.Println("STEP 4 FAILED: CF API returned not OK")
        return
    }

    // 4. update DB
    for _, u := range apiResp.Result {
        _, err := db.Exec(`
            UPDATE past_users
            SET current_rating = ?,
                max_rating = ?,
                title = ?,
                last_updated = CURRENT_TIMESTAMP
            WHERE codeforces_handle = ?
        `, u.Rating, u.MaxRating, u.Rank, u.Handle)

        if err != nil {
            fmt.Println("DB UPDATE FAILED:", u.Handle, err)
        } else {
            fmt.Println("UPDATED:", u.Handle)
        }
    }

    fmt.Println("STEP 5 DONE")

    // 5. done
    c.Redirect(http.StatusSeeOther, "/maintainer/users")
})

	r.POST("/admin/refresh_results", func(c *gin.Context) {
		cookie, err := c.Cookie("admin_logged_in")
		if err != nil || cookie != admin_password_hash {
			c.Redirect(http.StatusSeeOther, "/admin_login")
			return
		}
		err = refreshAllUserContestResults()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to refresh results: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	return r
}

