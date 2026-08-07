package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic:         true,
		WaitForDelivery: true,
		Timeout:         5 * time.Second,
	}))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	router.GET("/error", func(c *gin.Context) {
		panic("test error for sentry")
	})
	return router
}

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

	router := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	err := router.Run(":" + port)
	if err != nil {
		panic(err)
	}
}
