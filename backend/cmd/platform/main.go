package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"devops-platform/internal/config"
	"devops-platform/internal/repository"
	"devops-platform/internal/server"
)

var version = "0.1.0-alpha"

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("starting devops platform", zap.String("version", version))

	// Load configuration
	cfg := config.Load()

	// Initialize SQLite database and run auto-migrations
	db, err := repository.InitDB(cfg.DBPath, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Gin server
	srv, err := server.New(cfg, logger, db)
	if err != nil {
		logger.Fatal("failed to initialize server", zap.Error(err))
	}

	// Start HTTP server
	if err := srv.Start(); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down platform...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("server forced shutdown error", zap.Error(err))
	}

	logger.Info("platform stopped cleanly")
}
