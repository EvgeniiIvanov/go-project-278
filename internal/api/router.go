package api

import (
	"time"

	"code/internal/storage"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// Config holds HTTP-layer settings that come from the environment.
type Config struct {
	ShortURL string
}

// NewRouter builds the application HTTP router.
func NewRouter(store storage.Storage, cfg Config) *gin.Engine {
	server := &Server{
		store: store,
		cfg:   cfg,
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
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

	// Public shortener redirect. Keep after /api/links and /ping.
	router.GET("/:short_name", server.redirectByShortName)

	return router
}
