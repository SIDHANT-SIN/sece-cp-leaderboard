package routes

import (
	"net/http"

	"leaderboard/src/handles"
	"leaderboard/src/configs"

	"github.com/gin-gonic/gin"


)

func SetupRoutes(cfg *configs.Config) *gin.Engine {

	r := gin.Default()



	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/index", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/admin", handles.ShowAdminDashboard)

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

	//r.POST("/admin/contests/fetch", handles.FetchContests)

	r.POST("/admin/refresh_results", handles.RefreshResults)
	
	r.GET("/admin/sync_status", handles.GetSyncStatus)
	r.POST("/admin/cancel_sync", handles.CancelSync)

	// Leaderboard routes
	r.GET("/leaderboard", func(c *gin.Context) {
    handles.ShowLeaderboard(c, cfg)
         })

	
	r.GET("/past_events",func(c *gin.Context) {
    handles.PastEvents(c, cfg)
         })

	r.GET("/past_leaderboard", handles.ShowPastLeaderboard)

	// Refresh rating route
	r.POST("/maintainer/refresh_rating", handles.RefreshRating)

	//health checks and cron jobs
	r.GET("/api/health/ping", handles.SendPing)

	r.POST("/api/maintenance/purge", handles.Purg)


	//icpc routes
	r.GET("/problems", handles.ShowProblemsNew)

 

	return r
}