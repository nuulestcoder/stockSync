package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adapolytech/stock-manager/internal/config"
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/internal/routes"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/mongoclient"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()
	mongo, err := mongoclient.New(mongoCtx, cfg.MongoDB.URI, cfg.MongoDB.DB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongo.Disconnect(disconnectCtx); err != nil {
			log.Printf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	companyRepository := repository.NewCompanyRepository(mongo.Database())
	userRepository := repository.NewUserRepository(mongo.Database())
	productRepository := repository.NewProductRepository(mongo.Database())
	indexContext, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := repository.EnsureProductIndexes(indexContext, mongo.Database()); err != nil {
		indexCancel()
		log.Fatalf("Failed to create product indexes: %v", err)
	}
	indexCancel()
	authService := service.NewAuthService(companyRepository, userRepository, mongo.Client(), service.JWTConfig{
		AccessSecret: cfg.JWT.AccessSecret, RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL: cfg.JWT.AccessTTL, RefreshTTL: cfg.JWT.RefreshTTL,
	})
	authHandler := handler.NewAuthHandler(authService)
	productService := service.NewProductService(productRepository)
	productHandler := handler.NewProductHandler(productService)

	// Create router
	router := gin.Default()

	// Setup routes
	setupRoutes(router)
	routes.SetupAuthRoutes(router, authHandler)
	routes.SetupProductRoutes(router, productHandler, cfg.JWT.AccessSecret)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s (environment: %s)", srv.Addr, cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRoutes(router *gin.Engine) {
	// Health check (public, no auth required)
	router.GET("/health", handler.HealthCheck)
}
