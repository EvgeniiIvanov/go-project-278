package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(sentrygin.New(sentrygin.Options{}))
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	return router
}

func main() {
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
		}); err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		}
	}

	router := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err := router.Run(":" + port)
	if err != nil {
		panic(err)
	}
}
