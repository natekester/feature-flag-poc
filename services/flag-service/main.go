package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"flag-service/internal/api"
	"flag-service/internal/cache"
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	port := getEnv("PORT", "8080")
	ddbEndpoint := getEnv("DYNAMODB_ENDPOINT", "http://dynamodb-local:8000")
	redisAddr := getEnv("REDIS_ADDR", "redis-local:6379")
	tableName := getEnv("TABLE_NAME", "FeatureFlags")

	log.Printf("Starting flag-service on :%s (DDB: %s, Redis: %s)...", port, ddbEndpoint, redisAddr)

	// AWS SDK Config for DynamoDB Local
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: ddbEndpoint}, nil
			},
		)),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	rulesetCache := cache.NewRulesetCache(redisAddr)
	handler := api.NewFlagHandler(ddbClient, tableName, rulesetCache)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS Setup
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "If-None-Match"},
		ExposeHeaders:    []string{"ETag"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API Routing
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/admin/flags", handler.ListFlags)
		apiGroup.PUT("/admin/flags/:key/overrides/users/:userId", handler.SetUserOverride)
		apiGroup.GET("/evaluate", handler.EvaluateUser)
		apiGroup.GET("/stream", handler.SSEStream)
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// HTTP Server with Network Hardening
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server listen error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down flag-service gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}

	log.Println("Server exited cleanly.")
}
