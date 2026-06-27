package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AccountName        string
	AccountKey         string
	ContainerName      string

	AdminUsername      string
	AdminPasswordHash  string
	MaintainerPassword string

	Port string
	Logo string

	DBUrl     string
	AuthToken string

	RedisURL string
	CronSecret string
	SupaBase string
	FolderName string
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, works in deployment.")
	}

	return &Config{
		AccountName:        os.Getenv("AZURE_STORAGE_ACCOUNT"),
		AccountKey:         os.Getenv("AZURE_STORAGE_KEY"),
		ContainerName:      os.Getenv("AZURE_CONTAINER_NAME"),

		AdminUsername:      os.Getenv("ADMIN_USERNAME"),
		AdminPasswordHash:  os.Getenv("ADMIN_PASSWORD"),
		MaintainerPassword: os.Getenv("MAINTAINER_PASSWORD"),

		Port: os.Getenv("PORT"),
		Logo : os.Getenv("LOGO_URL"),

		DBUrl:     os.Getenv("TURSO_DATABASE_URL"),
		AuthToken: os.Getenv("TURSO_AUTH_TOKEN"),

		RedisURL: os.Getenv("REDIS_URL"),
		CronSecret: os.Getenv("CRON_SECRET"),

		SupaBase : os.Getenv("SUPABASE_URL"),
		FolderName: os.Getenv("FOLDER"),
	}
}