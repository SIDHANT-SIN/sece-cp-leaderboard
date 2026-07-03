package routes

import (
	"net/http"

	"leaderboard/src/configs"
	"leaderboard/src/handles"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(cfg *configs.Config) *gin.Engine {

	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	pastEventsHandler := handles.NewPastEventsHandler(cfg)

	apiHandler := handles.NewAPIHandler(
		&handles.DefaultHTTPClient{},
		cfg,
	)

	repo := &handles.DBRepository{}

	leaderboardHandler := handles.NewLeaderboardHandler(repo, cfg)

	authHandler := handles.NewAuthHandler(cfg, repo)

	adminUsersHandler := handles.NewAdminUsersHandler(
		repo,
		leaderboardHandler,
		cfg,
	)

	adminHandler := handles.NewAdminHandler(
		repo,
		leaderboardHandler,
		cfg,
	)
	maintainerUsersHandler := handles.NewMaintainerUsersHandler(
		cfg,
		repo,
		handles.NewDefaultTaskEnqueuer(),
	)

	icpcpyq := handles.NewIcpc_pyq(
		repo,
	)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/index", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/leaderboard")
	})

	r.GET("/admin", adminHandler.ShowAdminDashboard)

	r.GET("/admin_login", authHandler.AdminLoginPage)

	r.POST("/admin", authHandler.AdminLogin)

	r.GET("/maintainer", authHandler.MaintainerLoginPage)

	r.POST("/maintainer/login", authHandler.MaintainerLogin)

	r.GET("/maintainer/dashboard", authHandler.MaintainerDashboard)

	r.GET("/maintainer/users", maintainerUsersHandler.ShowPastUsers)

	r.POST("/maintainer/users/add", maintainerUsersHandler.AddPastUser)

	r.POST("/maintainer/users/delete", maintainerUsersHandler.DeletePastUser)

	r.POST("/admin/users/delete", adminUsersHandler.DeleteUser)

	r.GET("/admin/users", adminUsersHandler.ShowUsers)

	r.POST("/admin/users/add", adminUsersHandler.AddUser)

	// Contest management routes

	r.GET("/admin/contests", adminHandler.ShowContests)
	r.POST("/admin/contests/add", adminHandler.AddContest)

	r.POST("/admin/contests/delete", adminHandler.DeleteContest)

	// admin refresh logs

	r.GET("/admin/sync_status", adminHandler.GetSyncStatus)

	r.POST("/admin/cancel_sync", adminHandler.CancelSync)

	// Leaderboard routes

	r.GET("/leaderboard", leaderboardHandler.ShowLeaderboard)

	r.GET("/past_events", pastEventsHandler.PastEvents)

	r.GET("/past_leaderboard", leaderboardHandler.ShowPastLeaderboard)
	// Refresh rating route

	r.POST("/maintainer/refresh_rating", maintainerUsersHandler.RefreshRating)

	r.POST("/admin/refresh_results", adminHandler.RefreshResults)
	//health checks and cron jobs

	r.GET("/api/health/ping", handles.SendPing)

	r.POST("/admin/check_cf_api", apiHandler.CheckCFAPI)

	r.POST("/api/maintenance/purge", apiHandler.Purg)

	//icpc routes

	r.GET("/problems", icpcpyq.ShowProblemsNew)

	r.GET("/maintainer/icpc_pyq", authHandler.MaintainerICPCPage)

	r.POST("/maintainer/icpc_pyq", maintainerUsersHandler.CreateICPCProblem)

	return r
}
