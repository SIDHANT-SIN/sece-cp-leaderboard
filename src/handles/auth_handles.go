package handles

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg  *configs.Config
	repo Repository
}

func NewAuthHandler(cfg *configs.Config, repo Repository) *AuthHandler {
	return &AuthHandler{
		cfg:  cfg,
		repo: repo,
	}
}

func (h *AuthHandler) AdminLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login.tmpl", nil)
}

func (h *AuthHandler) MaintainerLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "maintainer_login.tmpl", nil)
}

func (h *AuthHandler) AdminLogin(c *gin.Context) {

	name := c.PostForm("username")
	password := c.PostForm("password")

	hashp := sha256.Sum256([]byte(password))

	if h.cfg.AdminPasswordHash == hex.EncodeToString(hashp[:]) &&
		h.cfg.AdminUsername == name {

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

func (h *AuthHandler) AdminPage(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	rows, err := h.repo.GetRecentSyncHistory(10)
	history := []map[string]interface{}{}

	if err == nil {
		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("failed to close response body: %v", err)
			}
		}()

		for rows.Next() {
			var (
				jobID            string
				status           string
				successful       int
				total            int
				failedContestIDs string
				startedAt        string
				completedAt      string
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

	c.HTML(http.StatusOK, "admin.tmpl", gin.H{
		"history": history,
		"error":   nil,
	})
}

func (h *AuthHandler) MaintainerLogin(c *gin.Context) {

	password := c.PostForm("password")

	hashp := sha256.Sum256([]byte(password))

	if hex.EncodeToString(hashp[:]) == h.cfg.MaintainerPassword {

		c.SetCookie(
			"maintainer_logged_in",
			h.cfg.MaintainerPassword,
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

func (h *AuthHandler) MaintainerDashboard(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")

	if err != nil || cookie != h.cfg.MaintainerPassword {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.HTML(http.StatusOK, "maintainer_dashboard.tmpl", nil)
}

func (h *AuthHandler) MaintainerICPCPage(c *gin.Context) {

	cookie, err := c.Cookie("maintainer_logged_in")

	if err != nil || cookie != h.cfg.MaintainerPassword {
		c.Redirect(http.StatusSeeOther, "/maintainer")
		return
	}

	c.HTML(http.StatusOK, "maintainer_icpc.tmpl", nil)
}
