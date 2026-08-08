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

	urls := router.Group("/urls")
	{
		urls.GET("", server.listLinks)
		urls.POST("", server.createLink)
		urls.GET("/:id", server.getLinkByID)
		urls.PUT("/:id", server.updateLink)
		urls.DELETE("/:id", server.deleteLink)
	}

	// Public shortener redirect. Keep after /urls and /ping.
	router.GET("/:short_name", server.redirectByShortName)

	return router
}
