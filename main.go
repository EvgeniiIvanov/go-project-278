package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"code/internal/api"
	"code/internal/storage"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	} else {
		fmt.Println(".env file loaded successfully")
	}

	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		fmt.Println("SENTRY_DSN is not set, Sentry disabled")
	} else {
		fmt.Printf("SENTRY_DSN is set (length: %d chars), initializing Sentry...\n", len(dsn))
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
		}); err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		} else {
			fmt.Println("Sentry initialized successfully")
		}
		defer sentry.Flush(2 * time.Second)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	store, err := storage.NewPostgres(ctx, databaseURL)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer store.Close()
	fmt.Println("Database connection established")

	router := api.NewRouter(store, api.Config{
		ShortURL:    os.Getenv("SHORT_URL"),
		CORSOrigins: api.ParseCORSOrigins(os.Getenv("CORS_ORIGINS")),
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
