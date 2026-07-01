package handles

import (
	"net/http"

	"leaderboard/src/repository"

	"github.com/gin-gonic/gin"
)

//  lists all users for admin
func ShowUsers(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusUnauthorized, "/admin")
		return
	}

	rows, err := repository.GetUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "DB error")
		return
	}
	defer rows.Close()

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
func AddUser(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusSeeOther, "/admin_login")
		return
	}

	handle := c.PostForm("handle")
	displayName := c.PostForm("display_name")

	err = repository.AddUser(handle, displayName)
	if err != nil {
		c.HTML(http.StatusBadRequest, "admin.tmpl", gin.H{
			"Users": repository.GetUsersList(),
			"error": "Could not add user: " + err.Error(),
		})
		return
	}

	rebuildLeaderboardCache()

	c.Redirect(http.StatusSeeOther, "/admin")
}

//  deletes a user by id
func DeleteUser(c *gin.Context) {
	cookie, err := c.Cookie("admin_logged_in")
	if err != nil || cookie != cfg.AdminPasswordHash {
		c.Redirect(http.StatusUnauthorized, "/admin")
		return
	}

	id := c.PostForm("id")
	err = repository.DeleteUser(id)
	if err != nil {
		c.String(http.StatusBadRequest, "Could not delete user: %v", err)
		return
	}

	rebuildLeaderboardCache()

	c.Redirect(http.StatusSeeOther, "/admin/users")
}