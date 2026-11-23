package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"devops-platform/internal/config"
	"devops-platform/internal/repository"
	"devops-platform/internal/server"
)

var version = "0.1.0-alpha"

func main() {
	envFlag := flag.String("env", config.ActiveEnv(), "Target environment (dev, test, prod)")
	configFlag := flag.String("config", "", "Path to custom YAML configuration file")
	portFlag := flag.String("port", "", "Override HTTP listen port (e.g. 8080 or :8080)")
	dbDriverFlag := flag.String("db-driver", "", "Override database driver (sqlite, mysql, postgres)")
	dbDSNFlag := flag.String("db-dsn", "", "Override database connection DSN")
	k8sModeFlag := flag.String("k8s-mode", "", "Override Kubernetes mode (minikube, kubeconfig, in-cluster, fake)")
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Tekton DevOps Platform v%s\n", version)
		fmt.Printf("Build Environment: %s\n", config.ActiveEnv(*envFlag))
		fmt.Printf("Compile Tag Env:   %s\n", config.CompileEnv)
		if config.GitCommit != "" {
			fmt.Printf("Git Commit:        %s\n", config.GitCommit)
		}
		if config.BuildTime != "" {
			fmt.Printf("Build Time:        %s\n", config.BuildTime)
		}
		return
	}

	// Initialize logger
	var logger *zap.Logger
	var err error
	if *envFlag == "dev" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("starting devops platform",
		zap.String("version", version),
		zap.String("active_env", config.ActiveEnv(*envFlag)),
		zap.String("compile_env", config.CompileEnv),
	)

	// Load configuration
	cfg, err := config.LoadWithFile(*envFlag, *configFlag)
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Apply CLI overrides if present
	if *portFlag != "" {
		port := *portFlag
		if !strings.HasPrefix(port, ":") {
			port = ":" + port
		}
		cfg.App.Addr = port
		cfg.Addr = port
	}
	if *dbDriverFlag != "" {
		cfg.Database.Driver = *dbDriverFlag
	}
	if *dbDSNFlag != "" {
		cfg.Database.DSN = *dbDSNFlag
		cfg.DBPath = *dbDSNFlag
	}
	if *k8sModeFlag != "" {
		cfg.Kubernetes.Mode = *k8sModeFlag
	}

	logger.Info("configuration loaded",
		zap.String("env", cfg.App.Env),
		zap.String("addr", cfg.App.Addr),
		zap.String("db_driver", cfg.Database.Driver),
		zap.String("k8s_mode", cfg.Kubernetes.Mode),
	)

	// Initialize database with dialect-aware migrations and pooling
	db, err := repository.InitDBFromConfig(cfg.ToDBConfig(), logger)
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
