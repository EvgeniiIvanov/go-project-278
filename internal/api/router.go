package api

import (
	"strings"
	"time"

	"code/internal/storage"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Config holds HTTP-layer settings that come from the environment.
type Config struct {
	ShortURL    string
	CORSOrigins []string
}

// NewRouter builds the application HTTP router.
func NewRouter(store storage.Storage, cfg Config) *gin.Engine {
	server := &Server{
		store: store,
		cfg:   cfg,
	}

	router := gin.New()
	// Caddy proxies from the same container; trust only local hop for ClientIP().
	_ = router.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	// Render may sit behind Cloudflare. If CF-Connecting-IP is absent,
	// Gin falls back to X-Forwarded-For from trusted proxies.
	router.TrustedPlatform = gin.PlatformCloudflare
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(cors.New(corsConfig(cfg.CORSOrigins)))
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic:         true,
		WaitForDelivery: true,
		Timeout:         5 * time.Second,
	}))

	router.GET("/ping", server.ping)

	links := router.Group("/api/links")
	{
		links.GET("", server.listLinks)
		links.POST("", server.createLink)
		links.GET("/:id", server.getLinkByID)
		links.PUT("/:id", server.updateLink)
		links.DELETE("/:id", server.deleteLink)
	}

	router.GET("/api/link_visits", server.listLinkVisits)

	// Public shortener redirect with visit tracking.
	router.GET("/r/:code", server.redirectByCode)

	return router
}

func corsConfig(origins []string) cors.Config {
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	return cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "Range"},
		// Frontend pagination needs to read this header.
		ExposeHeaders:    []string{"Content-Range", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// ParseCORSOrigins splits a comma-separated CORS_ORIGINS value.
func ParseCORSOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
