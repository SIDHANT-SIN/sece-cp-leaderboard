package main


func main() {
	r := setupRouter()
	// Start periodic refresh goroutine (every 1 hour)
	// go func() {
	// 	for {
	// 		err := refreshAllUserContestResults()
	// 		if err != nil {
	// 			log.Println("[Auto-Refresh] Error refreshing user contest results:", err)
	// 		}
	// 		time.Sleep(2 * time.Hour)
	// 	}
	// }()
	r.Run(":8080")
}
