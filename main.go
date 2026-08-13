package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"code/internal/api"
	"code/internal/storage"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	} else {
		slog.Info(".env file loaded successfully")
	}

	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		slog.Info("SENTRY_DSN is not set, Sentry disabled")
	} else {
		slog.Info("initializing Sentry", "dsn_length", len(dsn))
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
		}); err != nil {
			slog.Error("Sentry initialization failed", "err", err)
		} else {
			slog.Info("Sentry initialized successfully")
		}
		defer sentry.Flush(2 * time.Second)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := storage.NewPostgres(ctx, databaseURL)
	if err != nil {
		slog.Error("init db failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("database connection established")

	shortURL := os.Getenv("SHORT_URL")
	if shortURL == "" {
		shortURL = "http://127.0.0.1:8080"
	}

	router := api.NewRouter(store, api.Config{
		ShortURL:    shortURL,
		CORSOrigins: api.ParseCORSOrigins(os.Getenv("CORS_ORIGINS")),
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("starting server", "port", port)
	if err := router.Run(":" + port); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
