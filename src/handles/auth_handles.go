package handles

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"leaderboard/src/configs"
	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

var cfg = configs.LoadConfig()

func AdminLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login.tmpl", nil)
}

func MaintainerLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "maintainer_login.tmpl", nil)
}

func AdminLogin(c *gin.Context) {

	name := c.PostForm("username")
	password := c.PostForm("password")

	hashp := sha256.Sum256([]byte(password))

	if cfg.AdminPasswordHash == hex.EncodeToString(hashp[:]) &&
		cfg.AdminUsername == name {

		c.SetCookie(
			"admin_logged_in",
			hex.EncodeToString(hashp[:]),
			3600*24*2,
			"/",
			"",
			false,
			true,
		)

		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	c.HTML(
		http.StatusUnauthorized,
		"admin_login.tmpl",
		gin.H{"error": "Invalid credentials"},
	)
}

// func AdminPage(c *gin.Context) {

// 	cookie, err := c.Cookie("admin_logged_in")

// 	if err != nil || cookie != cfg.AdminPasswordHash {
// 		c.Redirect(http.StatusSeeOther, "/admin_login")
// 		return
// 	}

// 	c.HTML(http.StatusOK, "admin.tmpl", nil)
// }
func AdminPage(c *gin.Context) {
	// 1. Auth Guard: Verify the admin session cookie
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	// 2. Fetch the latest 10 sync runs from Turso
	rows, err := repository.GetRecentSyncHistory(10)
	history := []map[string]interface{}{}

	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var (
				jobID            string
				status           string
				successful       int
				total            int
				failedContestIDs  string
				startedAt        string
				completedAt       string
			)

			err := rows.Scan(
				&jobID,
				&status,
				&successful,
				&total,
				&failedContestIDs,
				&startedAt,
				&completedAt,
			)
			if err != nil {
				// Skips row if scanning fails
				continue 
			}

			history = append(history, map[string]interface{}{
				"job_id":              jobID,
				"status":              status,
				"successful_contests": successful,
				"total_contests":      total,
				"failed_contest_ids":  failedContestIDs,
				"started_at":          startedAt,
				"completed_at":        completedAt,
			})
		}
	}

	// 3. Inject the real data into the template instead of 'nil'
	c.HTML(http.StatusOK, "admin.tmpl", gin.H{
		"history": history,
		"error":   nil,
	})
}
func AdminLogout(c *gin.Context) {

	c.SetCookie(
		"admin_logged_in",
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	c.Redirect(http.StatusSeeOther, "/admin")
}

func MaintainerLogin(c *gin.Context) {

	password := c.PostForm("password")

	hashp := sha256.Sum256([]byte(password))

	if hex.EncodeToString(hashp[:]) == cfg.MaintainerPassword {

		c.SetCookie(
			"maintainer_logged_in",
			"true",
			3600*24*2,
			"/",
			"",
			false,
			true,
		)

		c.Redirect(http.StatusSeeOther, "/maintainer/dashboard")
		return
	}

	c.HTML(
		http.StatusUnauthorized,
		"maintainer_login.tmpl",
		gin.H{
			"error": "Invalid password",
		},
	)
}

func MaintainerDashboard(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")

	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.HTML(http.StatusOK, "maintainer_dashboard.tmpl", nil)
}

func MaintainerICPCPage(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")

	if err != nil || cookie != "true" {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.HTML(http.StatusOK, "maintainer_icpc.tmpl", nil)
}