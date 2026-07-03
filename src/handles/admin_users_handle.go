package handles

import (
	"log"
	"net/http"

	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"
)

type AdminUsersHandler struct {
	repo  Repository
	cache CacheBuilder
	cfg   *configs.Config
}

func NewAdminUsersHandler(repo Repository, cache CacheBuilder, cfg *configs.Config) *AdminUsersHandler {
	return &AdminUsersHandler{
		repo:  repo,
		cache: cache,
		cfg:   cfg,
	}
}

// lists all users for admin
func (h *AdminUsersHandler) ShowUsers(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	rows, err := h.repo.GetUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var handle, displayName string
		if err := rows.Scan(&id, &handle, &displayName); err != nil {
			c.String(http.StatusInternalServerError, "DB scan error")
			return
		}
		users = append(users, map[string]interface{}{
			"id":           id,
			"handle":       handle,
			"display_name": displayName,
		})
	}

	c.HTML(http.StatusOK, "admin_users.tmpl", gin.H{"users": users})
}

// adds a new Codeforces user
func (h *AdminUsersHandler) AddUser(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	handle := c.PostForm("handle")
	displayName := c.PostForm("display_name")

	err = h.repo.AddUser(handle, displayName)
	if err != nil {
		c.HTML(http.StatusBadRequest, "admin.tmpl", gin.H{
			"Users": h.repo.GetUsersList(),
			"error": "Could not add user: " + err.Error(),
		})
		return
	}

	if err := h.cache.RebuildLeaderboardCache(); err != nil {
		log.Printf("failed to rebuild leaderboard cache: %v", err)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// deletes a user by id
func (h *AdminUsersHandler) DeleteUser(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != h.cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}

	id := c.PostForm("id")
	err = h.repo.DeleteUser(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete user: %v", err)
		return
	}

	if err := h.cache.RebuildLeaderboardCache(); err != nil {
		log.Printf("failed to rebuild leaderboard cache: %v", err)
	}

	c.Redirect(http.StatusSeeOther, "/admin/users")
}
