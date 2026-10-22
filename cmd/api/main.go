package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adapolytech/stock-manager/internal/config"
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/repository"
	"github.com/adapolytech/stock-manager/internal/routes"
	"github.com/adapolytech/stock-manager/internal/service"
	"github.com/adapolytech/stock-manager/pkg/mongoclient"
	"github.com/adapolytech/stock-manager/pkg/storage"
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
	uploadStore, err := storage.New(context.Background(), storage.Config{
		Mode: cfg.Storage.Mode, UploadDir: cfg.Storage.UploadDir, PublicURL: cfg.Storage.PublicURL,
		R2: storage.R2Config{
			Endpoint: cfg.Storage.R2.Endpoint, Bucket: cfg.Storage.R2.Bucket,
			AccessKey: cfg.Storage.R2.AccessKey, SecretKey: cfg.Storage.R2.SecretKey,
			PublicURL: cfg.Storage.R2.PublicURL,
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize file storage: %v", err)
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
	stockRepository := repository.NewStockRepository(mongo.Database())
	stockMovementRepository := repository.NewStockMovementRepository(mongo.Database())
	stockRequestRepository := repository.NewStockRequestRepository(mongo.Database())
	saleRepository := repository.NewSaleRepository(mongo.Database())
	indexContext, indexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := repository.EnsureProductIndexes(indexContext, mongo.Database()); err != nil {
		indexCancel()
		log.Fatalf("Failed to create product indexes: %v", err)
	}
	indexCancel()
	stockIndexContext, stockIndexCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := repository.EnsureStockIndexes(stockIndexContext, mongo.Database()); err != nil {
		stockIndexCancel()
		saleIndexContext, saleIndexCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := repository.EnsureSaleIndexes(saleIndexContext, mongo.Database()); err != nil {
			saleIndexCancel()
			log.Fatalf("Failed to create sale indexes: %v", err)
		}
		saleIndexCancel()
		log.Fatalf("Failed to create stock indexes: %v", err)
	}
	stockIndexCancel()
	authService := service.NewAuthService(companyRepository, userRepository, mongo.Client(), service.JWTConfig{
		AccessSecret: cfg.JWT.AccessSecret, RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL: cfg.JWT.AccessTTL, RefreshTTL: cfg.JWT.RefreshTTL,
	})
	authHandler := handler.NewAuthHandler(authService)
	productService := service.NewProductService(productRepository)
	productHandler := handler.NewProductHandler(productService)
	stockService := service.NewStockService(productRepository, stockRepository, stockMovementRepository, stockRequestRepository)
	stockHandler := handler.NewStockHandler(stockService)
	saleService := service.NewSaleService(saleRepository, productRepository, companyRepository, stockRepository, stockMovementRepository)
	saleHandler := handler.NewSaleHandler(saleService)
	uploadHandler := handler.NewUploadHandler(uploadStore)

	// Create router
	router := gin.Default()
	router.MaxMultipartMemory = 32 << 20
	if strings.EqualFold(cfg.Storage.Mode, "") || strings.EqualFold(cfg.Storage.Mode, "local") {
		router.Static("/uploads", cfg.Storage.UploadDir)
	}

	// Setup routes
	setupRoutes(router)
	routes.SetupAuthRoutes(router, authHandler)
	routes.SetupProductRoutes(router, productHandler, cfg.JWT.AccessSecret)
	routes.SetupStockRoutes(router, stockHandler, cfg.JWT.AccessSecret)
	routes.SetupSaleRoutes(router, saleHandler, cfg.JWT.AccessSecret)
	routes.SetupUploadRoutes(router, uploadHandler, cfg.JWT.AccessSecret)

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
