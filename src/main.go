package main

import (
	"log"

	"leaderboard/src/configs"
	"leaderboard/src/database"
	"leaderboard/src/routes"
	"leaderboard/src/storage"
)

func main() {
	cfg := configs.LoadConfig()

	database.Connect(cfg)

	database.CreateTables()

	err := storage.InitAzure(
		cfg.AccountName,
		cfg.AccountKey,
		cfg.ContainerName,
	)
	if err != nil {
		log.Fatal("Failed to initialize Azure Blob Storage:", err)
	}

	log.Println("Successfully connected to Azure Blob Storage!")

	r := routes.SetupRoutes()

	port := cfg.Port

	r.Run(":" + port)
}