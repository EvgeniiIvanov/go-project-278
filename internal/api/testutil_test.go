package api

import (
	"time"

	"code/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		Port:            "8080",
		ShortURL:        "http://localhost:8080",
		DatabaseURL:     "postgres://unused",
		CORSOrigins:     []string{"http://localhost:5173"},
		RequestTimeout:  3 * time.Second,
		RedirectTimeout: 2 * time.Second,
	}
}
