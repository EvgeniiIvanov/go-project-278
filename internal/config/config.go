package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort                = "8080"
	defaultShortURL            = "http://127.0.0.1:8080"
	defaultCORSOrigin          = "http://localhost:5173"
	defaultDBMaxConns          = int32(10)
	defaultDBMinConns          = int32(1)
	defaultDBMaxConnLifetime   = 30 * time.Minute
	defaultDBMaxConnIdleTime   = 5 * time.Minute
	defaultDBHealthCheckPeriod = 1 * time.Minute
	defaultDBPingTimeout       = 5 * time.Second
	defaultRequestTimeout      = 3 * time.Second
	defaultRedirectTimeout     = 2 * time.Second
)

// Config is the process configuration loaded once at startup.
type Config struct {
	Port        string
	ShortURL    string
	DatabaseURL string
	CORSOrigins []string
	SentryDSN   string

	DB DBConfig

	// RequestTimeout bounds normal API/storage operations.
	RequestTimeout time.Duration
	// RedirectTimeout bounds redirect + visit write path.
	RedirectTimeout time.Duration
}

// DBConfig holds Postgres pool settings.
type DBConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	PingTimeout       time.Duration
}

// Load reads configuration from environment variables.
// Call godotenv.Load() before Load() if a local .env should be considered.
func Load() (Config, error) {
	cfg := Config{
		Port:        getString("PORT", defaultPort),
		ShortURL:    getString("SHORT_URL", defaultShortURL),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		CORSOrigins: parseCSV(getString("CORS_ORIGINS", defaultCORSOrigin)),
		SentryDSN:   strings.TrimSpace(os.Getenv("SENTRY_DSN")),
		DB: DBConfig{
			MaxConns:          getInt32("DB_MAX_CONNS", defaultDBMaxConns),
			MinConns:          getInt32("DB_MIN_CONNS", defaultDBMinConns),
			MaxConnLifetime:   getDuration("DB_MAX_CONN_LIFETIME", defaultDBMaxConnLifetime),
			MaxConnIdleTime:   getDuration("DB_MAX_CONN_IDLE_TIME", defaultDBMaxConnIdleTime),
			HealthCheckPeriod: getDuration("DB_HEALTHCHECK_PERIOD", defaultDBHealthCheckPeriod),
			PingTimeout:       getDuration("DB_PING_TIMEOUT", defaultDBPingTimeout),
		},
		RequestTimeout:  getDuration("REQUEST_TIMEOUT", defaultRequestTimeout),
		RedirectTimeout: getDuration("REDIRECT_TIMEOUT", defaultRedirectTimeout),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.DB.MinConns > cfg.DB.MaxConns {
		return Config{}, fmt.Errorf("DB_MIN_CONNS (%d) cannot be greater than DB_MAX_CONNS (%d)", cfg.DB.MinConns, cfg.DB.MaxConns)
	}
	if cfg.RequestTimeout <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT must be > 0")
	}
	if cfg.RedirectTimeout <= 0 {
		return Config{}, fmt.Errorf("REDIRECT_TIMEOUT must be > 0")
	}

	return cfg, nil
}

func getString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getInt32(key string, fallback int32) int32 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v < 0 {
		return fallback
	}
	return int32(v)
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return fallback
	}
	return d
}

func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}
