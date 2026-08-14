package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"code/internal/api"
	"code/internal/config"
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

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	if cfg.SentryDSN == "" {
		slog.Info("SENTRY_DSN is not set, Sentry disabled")
	} else {
		slog.Info("initializing Sentry", "dsn_length", len(cfg.SentryDSN))
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.SentryDSN,
		}); err != nil {
			slog.Error("Sentry initialization failed", "err", err)
		} else {
			slog.Info("Sentry initialized successfully")
		}
		defer sentry.Flush(2 * time.Second)
	}

	ctx := context.Background()
	store, err := storage.NewPostgres(ctx, cfg.DatabaseURL, cfg.DB)
	if err != nil {
		slog.Error("init db failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("database connection established")

	router := api.NewRouter(store, cfg)

	slog.Info("starting server",
		"port", cfg.Port,
		"request_timeout", cfg.RequestTimeout.String(),
		"redirect_timeout", cfg.RedirectTimeout.String(),
	)
	if err := router.Run(":" + cfg.Port); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
