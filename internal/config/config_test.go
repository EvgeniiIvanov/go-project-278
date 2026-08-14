package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://shortener:pass@localhost:5432/shortener_dev?sslmode=disable")
	t.Setenv("PORT", "")
	t.Setenv("SHORT_URL", "")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("REQUEST_TIMEOUT", "")
	t.Setenv("REDIRECT_TIMEOUT", "")
	t.Setenv("DB_MAX_CONNS", "20")
	t.Setenv("DB_MIN_CONNS", "2")
	t.Setenv("DB_MAX_CONN_LIFETIME", "45m")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, defaultPort, cfg.Port)
	assert.Equal(t, defaultShortURL, cfg.ShortURL)
	assert.Equal(t, []string{defaultCORSOrigin}, cfg.CORSOrigins)
	assert.Equal(t, defaultRequestTimeout, cfg.RequestTimeout)
	assert.Equal(t, defaultRedirectTimeout, cfg.RedirectTimeout)
	assert.Equal(t, int32(20), cfg.DB.MaxConns)
	assert.Equal(t, int32(2), cfg.DB.MinConns)
	assert.Equal(t, 45*time.Minute, cfg.DB.MaxConnLifetime)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := Load()
	require.Error(t, err)
}
