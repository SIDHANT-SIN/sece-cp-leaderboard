package routes

import (
	"net/http"

	"leaderboard/src/handles"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {

	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/index", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/admin", handles.AdminPage)

	r.GET("/admin_login", handles.AdminLoginPage)

	r.POST("/admin", handles.AdminLogin)

	r.GET("/logout", handles.AdminLogout)

	r.GET("/maintainer", handles.MaintainerLoginPage)

	r.POST("/maintainer/login", handles.MaintainerLogin)

	r.GET("/maintainer/dashboard", handles.MaintainerDashboard)

	r.GET("/maintainer/icpc_pyq", handles.MaintainerICPCPage)

	r.POST("/admin/check_cf_api", handles.CheckCFAPI)

	r.POST("/maintainer/icpc_pyq", handles.CreateICPCProblem)

	r.GET("/maintainer/users", handles.ShowPastUsers)

	r.POST("/maintainer/users/add", handles.AddPastUser)

	r.POST("/maintainer/users/delete", handles.DeletePastUser)

	r.POST("/admin/users/delete", handles.DeleteUser)

	r.GET("/admin/users", handles.ShowUsers)

	r.POST("/admin/users/add", handles.AddUser)

	// Contest management routes
	r.GET("/admin/contests", handles.ShowContests)

	r.POST("/admin/contests/add", handles.AddContest)

	r.POST("/admin/contests/delete", handles.DeleteContest)

	r.POST("/admin/contests/delete_all", handles.DeleteAllContests)

	r.POST("/admin/contests/fetch", handles.FetchContests)

	r.POST("/admin/refresh_results", handles.RefreshResults)

	// Leaderboard routes
	r.GET("/leaderboard", handles.ShowLeaderboard)

	r.GET("/past_leaderboard", handles.ShowPastLeaderboard)

	// Refresh rating route
	r.POST("/maintainer/refresh_rating", handles.RefreshRating)

	//health checks
	r.GET("/health/ping", handles.SendPing)

	//icpc routes
	r.GET("/problems", handles.ShowProblemsNew)

  //  r.GET("/problems/:id", handles.ShowProblem)

  //  r.GET("/problems/:id/solve", handles.ShowEditor)

//	r.POST("/api/judge", handles.Judge)

	return r
}