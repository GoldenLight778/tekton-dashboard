package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devops-platform/internal/api"
	"devops-platform/internal/config"
	"devops-platform/internal/executor"
	"devops-platform/internal/k8s"
	"devops-platform/internal/repository"
	"devops-platform/internal/service"
	pkgrouter "devops-platform/pkg/router"
)

// Server encapsulates the HTTP server, routing, and dependencies
type Server struct {
	cfg        *config.Config
	logger     *zap.Logger
	db         *sql.DB
	router     *gin.Engine
	httpServer *http.Server

	k8sBundle  *k8s.ClientBundle
	k8sProxy   http.Handler

	auditRepo       repository.AuditRepository
	projectRepo     repository.ProjectRepository
	pipelineRepo    repository.PipelineRepository
	runRepo         repository.PipelineRunRepository
	credRepo        repository.CredentialRepository
	metricsRepo     repository.MetricsRepository
	authService     *service.AuthService
	projectService  *service.ProjectService
	pipelineService *service.PipelineService
	runService      *service.RunService
	credService     *service.CredentialService
	metricsService  *service.MetricsService
}

// New creates a new Server instance with initialized dependencies
func New(cfg *config.Config, logger *zap.Logger, db *sql.DB) (*Server, error) {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = "0123456789abcdef0123456789abcdef"
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Initialize repositories with dialect driver
	dbDriver := cfg.Database.Driver
	userRepo := repository.NewUserRepository(db, dbDriver)
	auditRepo := repository.NewAuditRepository(db, dbDriver)
	projectRepo := repository.NewProjectRepository(db, dbDriver)
	pipelineRepo := repository.NewPipelineRepository(db, dbDriver)
	runRepo := repository.NewPipelineRunRepository(db, dbDriver)
	credRepo := repository.NewCredentialRepository(db, dbDriver)
	metricsRepo := repository.NewMetricsRepository(db, dbDriver)

	// Status update callback for executor
	updateStatus := func(id, status string, startedAt, finishedAt *time.Time) {
		current, err := runRepo.GetByID(context.Background(), id)
		if err == nil && current.Status == "Cancelled" {
			return
		}
		_ = runRepo.UpdateStatus(context.Background(), id, status, startedAt, finishedAt)
	}

	// Determine Kubernetes execution strategy and API proxy
	var pipelineExecutor executor.Executor
	var k8sBundle *k8s.ClientBundle
	var k8sProxyHandler http.Handler

	bundle, err := k8s.BuildClientBundle(cfg.Kubernetes, logger)
	if err == nil && bundle != nil {
		k8sBundle = bundle
		pipelineExecutor = executor.NewK8sExecutor(bundle.Dynamic, bundle.Clientset, bundle.Namespace, updateStatus, logger)
		proxy, pErr := pkgrouter.NewProxyHandler(bundle.RestConfig, 30*time.Second)
		if pErr != nil {
			logger.Warn("failed to initialize K8s API proxy handler, falling back to mock APIs", zap.Error(pErr))
		} else {
			k8sProxyHandler = proxy
			logger.Info("K8s API reverse proxy initialized successfully", zap.String("mode", bundle.Mode), zap.String("namespace", bundle.Namespace))
		}
	} else {
		logger.Info("Using FakeExecutor for pipeline runs", zap.String("mode", cfg.Kubernetes.Mode))
		pipelineExecutor = executor.NewFakeExecutor(updateStatus, logger)
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, logger)
	projectService := service.NewProjectService(projectRepo, auditRepo, logger)
	pipelineService := service.NewPipelineService(pipelineRepo, projectRepo, auditRepo, logger)
	runService := service.NewRunService(runRepo, pipelineRepo, auditRepo, pipelineExecutor, logger)
	credService := service.NewCredentialService(credRepo, projectRepo, auditRepo, cfg.SecretKey, logger)
	metricsService := service.NewMetricsService(metricsRepo, logger)

	// Ensure default admin user exists
	if err := authService.EnsureDefaultAdmin(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize default admin: %w", err)
	}

	s := &Server{
		cfg:             cfg,
		logger:          logger,
		db:              db,
		router:          r,
		k8sBundle:       k8sBundle,
		k8sProxy:        k8sProxyHandler,
		auditRepo:       auditRepo,
		projectRepo:     projectRepo,
		pipelineRepo:    pipelineRepo,
		runRepo:         runRepo,
		credRepo:        credRepo,
		metricsRepo:     metricsRepo,
		authService:     authService,
		projectService:  projectService,
		pipelineService: pipelineService,
		runService:      runService,
		credService:     credService,
		metricsService:  metricsService,
	}

	s.setupRoutes()
	s.setupStaticAndSPA()
	return s, nil
}

func (s *Server) setupRoutes() {
	// 1. Health checks & Tekton compatibility properties
	s.router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	s.router.GET("/readiness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Tekton Dashboard properties endpoint to prevent frontend config loading errors
	s.router.GET("/v1/properties", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"DashboardNamespace": "default",
			"DashboardVersion":   "0.1.0-alpha",
			"DefaultNamespace":   "default",
			"PipelineNamespace":  "default",
			"PipelineVersion":    "v0.62.0",
			"TriggersNamespace":  "default",
			"TriggersVersion":    "v0.28.0",
			"ReadOnly":           false,
			"LogoutURL":          "",
			"TenantNamespaces":   []string{"default"},
			"StreamLogs":         true,
		})
	})

	// 2. Kubernetes and Tekton Proxy or Fallback Mocks
	if s.k8sProxy != nil {
		proxyHandler := gin.WrapH(s.k8sProxy)
		s.router.Any("/api/v1/*filepath", proxyHandler)
		s.router.Any("/apis/*filepath", proxyHandler)
	} else {
		// Mock responses when running in fake mode or when cluster is offline
		s.router.GET("/api/v1/namespaces", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "v1",
				"kind":       "NamespaceList",
				"items": []gin.H{
					{
						"metadata": gin.H{"name": "default"},
					},
					{
						"metadata": gin.H{"name": "tekton-pipelines"},
					},
				},
			})
		})

		s.router.GET("/apis/*filepath", func(c *gin.Context) {
			path := c.Param("filepath")
			if strings.Contains(path, "extensions") {
				c.JSON(http.StatusOK, gin.H{
					"apiVersion": "dashboard.tekton.dev/v1alpha1",
					"kind":       "ExtensionList",
					"items":      []gin.H{},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "List",
				"items":      []gin.H{},
			})
		})
	}

	// 2. Platform API v1
	v1 := s.router.Group("/api/platform/v1")
	{
		// Public: Version
		v1.GET("/version", func(c *gin.Context) {
			api.Success(c, gin.H{
				"version":   "0.1.0-alpha",
				"framework": "gin",
				"go":        "1.27.x",
			})
		})

		// Public: Auth Login
		authHandler := api.NewAuthHandler(s.authService)
		v1.POST("/auth/login", authHandler.Login)

		// Public: Webhook Trigger
		webhookHandler := api.NewWebhookHandler(s.pipelineRepo, s.runService)
		v1.POST("/webhooks/:id", webhookHandler.Handle)

		// Protected endpoints
		protected := v1.Group("")
		protected.Use(api.AuthMiddleware(s.cfg.JWTSecret))
		{
			// Auth Profile
			protected.GET("/auth/me", authHandler.Me)

			// Projects
			projectHandler := api.NewProjectHandler(s.projectService)
			protected.POST("/projects", projectHandler.Create)
			protected.GET("/projects", projectHandler.List)
			protected.GET("/projects/:id", projectHandler.Get)
			protected.PUT("/projects/:id", projectHandler.Update)
			protected.DELETE("/projects/:id", projectHandler.Delete)

			// Pipelines
			pipelineHandler := api.NewPipelineHandler(s.pipelineService)
			protected.POST("/projects/:id/pipelines", pipelineHandler.Create)
			protected.GET("/projects/:id/pipelines", pipelineHandler.ListByProject)
			protected.GET("/pipelines/:id", pipelineHandler.Get)
			protected.PUT("/pipelines/:id", pipelineHandler.Update)
			protected.DELETE("/pipelines/:id", pipelineHandler.Delete)

			// Pipeline Runs
			runHandler := api.NewRunHandler(s.runService)
			protected.POST("/pipelines/:id/runs", runHandler.Trigger)
			protected.GET("/pipelines/:id/runs", runHandler.ListByPipeline)
			protected.GET("/projects/:id/runs", runHandler.ListByProject)
			protected.GET("/runs/:id", runHandler.Get)
			protected.POST("/runs/:id/cancel", runHandler.Cancel)
			protected.GET("/runs/:id/logs/sse", runHandler.LogsSSE)

			// Audit logs
			auditHandler := api.NewAuditHandler(s.auditRepo)
			protected.GET("/audit-logs", auditHandler.List)

			// Credentials
			credHandler := api.NewCredentialHandler(s.credService)
			protected.POST("/projects/:id/credentials", credHandler.Create)
			protected.GET("/projects/:id/credentials", credHandler.ListByProject)
			protected.DELETE("/credentials/:id", credHandler.Delete)

			// Metrics & Dashboard
			metricsHandler := api.NewMetricsHandler(s.metricsService)
			protected.GET("/metrics/dashboard", metricsHandler.Dashboard)
		}
	}
}

func (s *Server) setupStaticAndSPA() {
	candidatePaths := []string{
		"./dist",
		"../frontend/dist",
		"./frontend/dist",
	}

	var distDir string
	for _, p := range candidatePaths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			distDir = p
			break
		}
	}

	if distDir != "" {
		s.logger.Info("serving frontend static assets from directory", zap.String("dir", distDir))

		// Serve static assets directory
		s.router.Static("/assets", filepath.Join(distDir, "assets"))

		// SPA fallback: any non-API route returns index.html
		s.router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/apis/") || strings.HasPrefix(path, "/v1/") {
				api.Error(c, http.StatusNotFound, 40400, "endpoint not found")
				return
			}

			// Try serving the exact file requested if it exists
			filePath := filepath.Join(distDir, filepath.Clean(path))
			if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
				c.File(filePath)
				return
			}

			// Otherwise, return index.html for client-side routing
			c.File(filepath.Join(distDir, "index.html"))
		})
	} else {
		s.logger.Warn("frontend dist directory not found, running in API-only mode")
		s.router.NoRoute(func(c *gin.Context) {
			api.Error(c, http.StatusNotFound, 40400, "endpoint not found")
		})
	}
}

// Start runs the HTTP server non-blocking
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("platform server starting", zap.String("addr", s.cfg.Addr))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("http server listen failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("shutting down platform server...")
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Router returns the underlying Gin engine
func (s *Server) Router() *gin.Engine {
	return s.router
}
