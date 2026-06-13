package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"leaderboard/src/storage"
)

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


	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found (this is fine if you set environment variables manually in production)")
	}

	// 2. Grab your Azure credentials from the .env file
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")
	accountKey := os.Getenv("AZURE_STORAGE_KEY")
	containerName := os.Getenv("AZURE_CONTAINER_NAME")

	// 3. Initialize Azure exactly ONCE
	err = storage.InitAzure(accountName, accountKey, containerName)
	if err != nil {
		log.Fatal("💥 Failed to initialize Azure Blob Storage:", err)
	}
	log.Println("✅ Successfully connected to Azure Blob Storage!")







	r.Run(":8080")
}
