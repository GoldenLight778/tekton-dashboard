package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	tektonFallbackMu     sync.RWMutex
	tektonFallbackActive bool
	standaloneTaskRuns   []gin.H

	auditRepo       repository.AuditRepository
	projectRepo     repository.ProjectRepository
	pipelineRepo    repository.PipelineRepository
	taskRepo        repository.TaskRepository
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
	taskRepo := repository.NewTaskRepository(db, dbDriver)
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
		taskRepo:        taskRepo,
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

	s.ensureDefaultRBACAndNamespaces(context.Background())
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
		k8sMode := s.cfg.Kubernetes.Mode
		if k8sMode == "" {
			k8sMode = "fake"
		}
		c.JSON(http.StatusOK, gin.H{
			"dashboardNamespace": "default",
			"dashboardVersion":   "v0.73.0-devops",
			"defaultNamespace":   "default",
			"pipelinesNamespace": "tekton-pipelines",
			"pipelinesVersion":   "v0.62.0",
			"triggersNamespace":  "tekton-pipelines",
			"triggersVersion":    "v0.28.0",
			"isReadOnly":         false,
			"logoutURL":          "/#/login",
			"tenantNamespaces":   []string{},
			"streamLogs":         true,
			"k8sMode":            k8sMode,
			// Backwards-compatible PascalCase aliases
			"DashboardNamespace": "default",
			"DashboardVersion":   "v0.73.0-devops",
			"DefaultNamespace":   "default",
			"PipelineNamespace":  "tekton-pipelines",
			"PipelineVersion":    "v0.62.0",
			"TriggersNamespace":  "tekton-pipelines",
			"TriggersVersion":    "v0.28.0",
			"ReadOnly":           false,
			"LogoutURL":          "/#/login",
			"TenantNamespaces":   []string{},
			"StreamLogs":         true,
		})
	})

	// 2. Kubernetes and Tekton Proxy or Fallback Mocks
	handleExtensions := func(c *gin.Context) {
		if isUpgradeRequest(c.Request) {
			handleWebSocketWatch(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "dashboard.tekton.dev/v1alpha1",
			"kind":       "ExtensionList",
			"metadata": gin.H{
				"resourceVersion": "1",
			},
			"items": []gin.H{},
		})
	}

	handleNamespaces := func(c *gin.Context) {
		if isUpgradeRequest(c.Request) {
			handleWebSocketWatch(c)
			return
		}
		nsList := s.listNamespacesInternal(c.Request.Context(), "admin", "admin")
		items := make([]gin.H, 0, len(nsList))
		for _, n := range nsList {
			items = append(items, gin.H{
				"metadata": gin.H{
					"name":              n.Name,
					"uid":               "namespace-" + n.Name,
					"creationTimestamp": n.CreatedAt.Format(time.RFC3339),
					"annotations": gin.H{
						"dashboard.tekton.dev/alias":       n.Alias,
						"dashboard.tekton.dev/description": n.Description,
					},
				},
				"status": gin.H{"phase": n.Status},
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "v1",
			"kind":       "NamespaceList",
			"metadata": gin.H{
				"resourceVersion": "1",
			},
			"items": items,
		})
	}

	handleCoreV1Fallback := func(c *gin.Context) {
		if isUpgradeRequest(c.Request) {
			handleWebSocketWatch(c)
			return
		}
		if !s.sanitizeAndPrepareK8sProxy(c) {
			return
		}
		path := strings.TrimRight(c.Request.URL.Path, "/")
		if strings.HasSuffix(path, "/log") && strings.Contains(path, "/pods/") {
			s.handlePodLog(c, path)
			return
		}
		if strings.Contains(path, "/pods/") {
			parts := strings.Split(path, "/")
			podName := parts[len(parts)-1]
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": gin.H{
					"name":      podName,
					"namespace": "default",
					"uid":       podName,
				},
				"status": gin.H{
					"phase": "Succeeded",
				},
			})
			return
		}
		if strings.HasSuffix(path, "/namespaces") || strings.Contains(path, "/api/v1/namespaces") && !strings.Contains(path, "/pods") && !strings.Contains(path, "/events") && !strings.Contains(path, "/serviceaccounts") {
			handleNamespaces(c)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "v1",
			"kind":       "List",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      []gin.H{},
		})
	}

	wrapProxy := func(h http.Handler) gin.HandlerFunc {
		return func(c *gin.Context) {
			path := c.Request.URL.Path

			// Always handle extensions locally without forwarding to K8s
			if strings.HasPrefix(path, "/apis/dashboard.tekton.dev") {
				handleExtensions(c)
				return
			}

			// If it's a websocket upgrade: if Tekton CRDs were already detected missing, handle locally
			if isUpgradeRequest(c.Request) {
				s.tektonFallbackMu.RLock()
				fallbackActive := s.tektonFallbackActive
				s.tektonFallbackMu.RUnlock()
				if fallbackActive && (strings.HasPrefix(path, "/apis/tekton.dev") || strings.HasPrefix(path, "/apis/triggers.tekton.dev")) {
					handleWebSocketWatch(c)
					return
				}
				if !s.sanitizeAndPrepareK8sProxy(c) {
					return
				}
				h.ServeHTTP(c.Writer, c.Request)
				return
			}

			if !s.sanitizeAndPrepareK8sProxy(c) {
				return
			}

			// Capture response
			rec := &statusRecorder{
				ResponseWriter: c.Writer,
				statusCode:     0,
			}
			h.ServeHTTP(rec, c.Request)

			// If cluster returned 404 or 403 (e.g. Tekton CRDs not installed in cluster) and it's a Tekton, Dashboard, or Pod Log route, intercept with fallback
			if rec.statusCode == http.StatusNotFound || rec.statusCode == http.StatusForbidden {
				if strings.HasPrefix(path, "/apis/tekton.dev") ||
					strings.HasPrefix(path, "/apis/triggers.tekton.dev") ||
					strings.HasPrefix(path, "/apis/dashboard.tekton.dev") {
					s.tektonFallbackMu.Lock()
					s.tektonFallbackActive = true
					s.tektonFallbackMu.Unlock()
					s.synthesizeTektonResponse(c, path)
					return
				}
				if strings.HasPrefix(path, "/api/v1/") && (strings.Contains(path, "/pods/") || strings.Contains(path, "/namespaces")) {
					handleCoreV1Fallback(c)
					return
				}
				// Otherwise write the original error response
				c.Writer.WriteHeader(rec.statusCode)
				_, _ = c.Writer.Write(rec.buf.Bytes())
			}
		}
	}

	if s.k8sProxy != nil {
		proxyHandler := wrapProxy(s.k8sProxy)
		s.router.Any("/api/v1/*filepath", proxyHandler)
		s.router.Any("/apis/*filepath", proxyHandler)
	} else {
		// Mock responses when running in fake mode or when cluster is offline
		s.router.Any("/api/v1/*filepath", handleCoreV1Fallback)
		s.router.Any("/apis/*filepath", func(c *gin.Context) {
			if !s.sanitizeAndPrepareK8sProxy(c) {
				return
			}
			s.synthesizeTektonResponse(c, c.Request.URL.Path)
		})
	}

	s.router.Any("/apis", func(c *gin.Context) {
		s.synthesizeTektonResponse(c, "/apis")
	})

	// 2. Platform API v1 (Auth, RBAC, K8s Native Namespaces/Secrets, and Dashboard Metrics)
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

		// Public: Dual-mode Login (Username/Password -> K8s Impersonation OR K8s SA Bearer Token)
		authHandler := api.NewAuthHandler(s.authService)
		v1.POST("/auth/login", s.handleDualModeLogin)

		// Public: Webhook Trigger
		webhookHandler := api.NewWebhookHandler(s.pipelineRepo, s.runService)
		v1.POST("/webhooks/:id", webhookHandler.Handle)

		// Protected endpoints
		protected := v1.Group("")
		protected.Use(api.AuthMiddleware(s.cfg.JWTSecret, s.cfg.App.Env == "dev"))
		{
			// Auth Profile & K8s SelfSubjectAccessReview Permissions
			protected.GET("/auth/me", authHandler.Me)
			protected.GET("/auth/permissions", s.handleAuthPermissions)

			// Read-Only K8s Cluster Namespaces (Namespaces are created by K8s cluster admin)
			protected.GET("/namespaces", s.handleListK8sNamespaces)

			// Native K8s Secrets + ServiceAccount Auto-Binding
			protected.GET("/namespaces/:namespace/secrets", s.handleListK8sSecrets)
			protected.POST("/namespaces/:namespace/secrets", s.handleCreateK8sSecret)
			protected.DELETE("/namespaces/:namespace/secrets/:name", s.handleDeleteK8sSecret)

			// K8s RBAC Users, Labeled ClusterRoles & RoleBindings (Strictly scoped to label `tekton-devops-dashboard=true`)
			protected.GET("/rbac/roles", s.handleRBACRoles)
			protected.POST("/rbac/roles", s.handleSaveRBACRole)
			protected.PUT("/rbac/roles/:name", s.handleSaveRBACRole)
			protected.DELETE("/rbac/roles/:name", s.handleDeleteRBACRole)
			protected.GET("/rbac/users", s.handleListRBACUsers)
			protected.POST("/rbac/users", s.handleCreateRBACUser)
			protected.PUT("/rbac/users/:id", s.handleUpdateRBACUser)
			protected.DELETE("/rbac/users/:id", s.handleDeleteRBACUser)
			protected.GET("/rbac/bindings", s.handleListRBACBindings)
			protected.POST("/rbac/bindings", s.handleCreateRBACBinding)
			protected.DELETE("/rbac/bindings/:id", s.handleDeleteRBACBinding)

			// Legacy Compatibility Routes (Projects / Pipelines / Runs / Credentials / Audit)
			projectHandler := api.NewProjectHandler(s.projectService)
			protected.POST("/projects", projectHandler.Create)
			protected.GET("/projects", projectHandler.List)
			protected.GET("/projects/:id", projectHandler.Get)
			protected.PUT("/projects/:id", projectHandler.Update)
			protected.DELETE("/projects/:id", projectHandler.Delete)

			pipelineHandler := api.NewPipelineHandler(s.pipelineService)
			protected.POST("/projects/:id/pipelines", pipelineHandler.Create)
			protected.GET("/projects/:id/pipelines", pipelineHandler.ListByProject)
			protected.GET("/pipelines/:id", pipelineHandler.Get)
			protected.PUT("/pipelines/:id", pipelineHandler.Update)
			protected.DELETE("/pipelines/:id", pipelineHandler.Delete)

			runHandler := api.NewRunHandler(s.runService)
			protected.POST("/pipelines/:id/runs", runHandler.Trigger)
			protected.GET("/pipelines/:id/runs", runHandler.ListByPipeline)
			protected.GET("/projects/:id/runs", runHandler.ListByProject)
			protected.GET("/runs/:id", runHandler.Get)
			protected.POST("/runs/:id/cancel", runHandler.Cancel)
			protected.GET("/runs/:id/logs/sse", runHandler.LogsSSE)

			auditHandler := api.NewAuditHandler(s.auditRepo)
			protected.GET("/audit-logs", auditHandler.List)

			credHandler := api.NewCredentialHandler(s.credService)
			protected.POST("/projects/:id/credentials", credHandler.Create)
			protected.GET("/projects/:id/credentials", credHandler.ListByProject)
			protected.DELETE("/credentials/:id", credHandler.Delete)

			// Native Tekton Dashboard Metrics
			protected.GET("/metrics/dashboard", s.handleNativeDashboardMetrics)
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

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	buf        bytes.Buffer
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	if code != http.StatusNotFound && code != http.StatusForbidden {
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.statusCode == http.StatusNotFound || r.statusCode == http.StatusForbidden {
		return r.buf.Write(b)
	}
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}

func (r *statusRecorder) Flush() {
	if fl, ok := r.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func isUpgradeRequest(req *http.Request) bool {
	connection := req.Header.Get("Connection")
	upgrade := req.Header.Get("Upgrade")
	return strings.EqualFold(connection, "upgrade") || strings.EqualFold(upgrade, "websocket")
}

func handleWebSocketWatch(c *gin.Context) {
	wsKey := c.GetHeader("Sec-WebSocket-Key")
	if wsKey == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	hj, ok := c.Writer.(http.Hijacker)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()

	h := sha1.New()
	h.Write([]byte(wsKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
	_, _ = bufrw.WriteString(resp)
	_ = bufrw.Flush()

	buf := make([]byte, 512)
	for {
		if _, err := conn.Read(buf); err != nil {
			break
		}
	}
}

func (s *Server) handlePodLog(c *gin.Context, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var podName string
	for i, p := range parts {
		if p == "pods" && i+1 < len(parts) {
			podName = parts[i+1]
			break
		}
	}

	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")

	if podName != "" {
		runs, err := s.runRepo.ListAll(c.Request.Context())
		if err == nil {
			for _, r := range runs {
				if r.K8sPipelineRunName == podName || r.ID == podName || strings.HasPrefix(podName, r.K8sPipelineRunName) || (len(r.ID) >= 8 && strings.HasPrefix(podName, r.ID[:8])) {
					logChan, streamErr := s.runService.StreamLogs(c.Request.Context(), r.ID)
					if streamErr == nil {
						c.Writer.WriteHeader(http.StatusOK)
						for {
							select {
							case <-c.Request.Context().Done():
								return
							case line, ok := <-logChan:
								if !ok {
									return
								}
								_, _ = io.WriteString(c.Writer, line+"\n")
								c.Writer.Flush()
							}
						}
					}
					break
				}
			}
		}
	}

	c.String(http.StatusOK, fmt.Sprintf("[%s] [TASK] Pod '%s' container step executed.\n[%s] [SUCCESS] Completed with exit code 0.\n",
		time.Now().UTC().Format("15:04:05"), podName, time.Now().UTC().Format("15:04:05")))
}

func runStatusToCondition(status string) string {
	switch status {
	case "Succeeded":
		return "True"
	case "Failed", "Cancelled":
		return "False"
	case "Running":
		return "Unknown"
	default:
		return "Unknown"
	}
}

func formatTaskResource(t repository.Task) gin.H {
	var spec gin.H
	if t.SpecJSON != "" && t.SpecJSON != "{}" {
		_ = json.Unmarshal([]byte(t.SpecJSON), &spec)
	}
	if len(spec) == 0 {
		step := gin.H{
			"name":  "execute",
			"image": t.Image,
		}
		if t.Script != "" {
			step["script"] = t.Script
		}
		spec = gin.H{
			"description": t.Description,
			"steps":       []gin.H{step},
		}
	}
	ns := t.Namespace
	if ns == "" {
		ns = "default"
	}
	return gin.H{
		"apiVersion": "tekton.dev/v1",
		"kind":       "Task",
		"metadata": gin.H{
			"name":              t.Name,
			"namespace":         ns,
			"uid":               t.ID,
			"creationTimestamp": t.CreatedAt.Format(time.RFC3339),
			"labels": gin.H{
				"platform.devops.io/managed-by": "devops-platform",
			},
		},
		"spec": spec,
	}
}

func (s *Server) formatPipelineRunAndTaskRun(r repository.PipelineRun, pipelineNames map[string]string) (gin.H, gin.H) {
	runName := r.K8sPipelineRunName
	if runName == "" {
		if len(r.ID) >= 8 {
			runName = r.ID[:8] + "-run"
		} else {
			runName = r.ID + "-run"
		}
	}
	ns := r.Namespace
	if ns == "" {
		ns = "default"
	}
	pipelineName := pipelineNames[r.PipelineID]
	if pipelineName == "" {
		pipelineName = r.PipelineID
	}
	startTime := r.CreatedAt.Format(time.RFC3339)
	if r.StartedAt != nil {
		startTime = r.StartedAt.Format(time.RFC3339)
	}
	taskRunName := runName + "-task"

	stepState := gin.H{
		"name":      "execute",
		"container": "step-execute",
	}
	if r.Status == "Running" || r.Status == "Pending" {
		stepState["running"] = gin.H{"startedAt": startTime}
	} else {
		exitCode := 0
		if r.Status == "Failed" || r.Status == "Cancelled" {
			exitCode = 1
		}
		stepState["terminated"] = gin.H{
			"exitCode":   exitCode,
			"reason":     r.Status,
			"startedAt":  startTime,
			"finishedAt": time.Now().UTC().Format(time.RFC3339),
		}
	}

	prStatus := gin.H{
		"startTime": startTime,
		"conditions": []gin.H{
			{
				"type":    "Succeeded",
				"status":  runStatusToCondition(r.Status),
				"reason":  r.Status,
				"message": fmt.Sprintf("PipelineRun status: %s", r.Status),
			},
		},
		"childReferences": []gin.H{
			{
				"apiVersion":       "tekton.dev/v1",
				"kind":             "TaskRun",
				"name":             taskRunName,
				"pipelineTaskName": "execute",
			},
		},
		"pipelineSpec": gin.H{
			"tasks": []gin.H{
				{
					"name": "execute",
					"taskSpec": gin.H{
						"steps": []gin.H{
							{
								"name":  "execute",
								"image": "alpine:latest",
							},
						},
					},
				},
			},
		},
	}
	if r.FinishedAt != nil {
		prStatus["completionTime"] = r.FinishedAt.Format(time.RFC3339)
	}

	pr := gin.H{
		"apiVersion": "tekton.dev/v1",
		"kind":       "PipelineRun",
		"metadata": gin.H{
			"name":              runName,
			"namespace":         ns,
			"uid":               r.ID,
			"creationTimestamp": r.CreatedAt.Format(time.RFC3339),
			"labels": gin.H{
				"tekton.dev/pipeline":            pipelineName,
				"platform.devops.io/pipeline-id": r.PipelineID,
				"platform.devops.io/run-id":      r.ID,
			},
		},
		"spec": gin.H{
			"pipelineRef": gin.H{"name": pipelineName},
		},
		"status": prStatus,
	}

	trStatus := gin.H{
		"podName":   runName,
		"startTime": startTime,
		"conditions": []gin.H{
			{
				"type":    "Succeeded",
				"status":  runStatusToCondition(r.Status),
				"reason":  r.Status,
				"message": fmt.Sprintf("TaskRun status: %s", r.Status),
			},
		},
		"steps": []gin.H{stepState},
		"taskSpec": gin.H{
			"steps": []gin.H{
				{
					"name":  "execute",
					"image": "alpine:latest",
				},
			},
		},
	}
	if r.FinishedAt != nil {
		trStatus["completionTime"] = r.FinishedAt.Format(time.RFC3339)
	}

	tr := gin.H{
		"apiVersion": "tekton.dev/v1",
		"kind":       "TaskRun",
		"metadata": gin.H{
			"name":              taskRunName,
			"namespace":         ns,
			"uid":               r.ID + "-tr",
			"creationTimestamp": r.CreatedAt.Format(time.RFC3339),
			"labels": gin.H{
				"tekton.dev/pipeline":            pipelineName,
				"tekton.dev/pipelineRun":         runName,
				"tekton.dev/pipelineTask":        "execute",
				"platform.devops.io/pipeline-id": r.PipelineID,
			},
		},
		"spec": gin.H{
			"taskSpec": gin.H{
				"steps": []gin.H{
					{
						"name":  "execute",
						"image": "alpine:latest",
					},
				},
			},
		},
		"status": trStatus,
	}

	return pr, tr
}

func (s *Server) synthesizeTektonResponse(c *gin.Context, path string) {
	if isUpgradeRequest(c.Request) {
		handleWebSocketWatch(c)
		return
	}

	cleanPath := strings.Trim(path, "/")
	parts := strings.Split(cleanPath, "/")

	var resourceType string
	var reqNamespace string = "default"
	hasExplicitNamespace := false
	for i, p := range parts {
		if p == "namespaces" && i+1 < len(parts) && parts[i+1] != "" {
			reqNamespace = parts[i+1]
			hasExplicitNamespace = true
		}
		switch p {
		case "pipelines", "pipelineruns", "tasks", "taskruns", "customruns",
			"eventlisteners", "triggers", "triggertemplates", "triggerbindings",
			"clusterinterceptors", "clustertriggerbindings", "interceptors", "extensions":
			resourceType = p
		}
	}

	switch resourceType {
	case "extensions":
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "dashboard.tekton.dev/v1alpha1",
			"kind":       "ExtensionList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      []gin.H{},
		})
		return

	case "pipelines":
		isDetail := false
		var pipelineName string
		for i, p := range parts {
			if p == "pipelines" && i+1 < len(parts) && parts[i+1] != "" {
				isDetail = true
				pipelineName = parts[i+1]
				break
			}
		}

		if c.Request.Method == http.MethodPost {
			var payload struct {
				Metadata struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				} `json:"metadata"`
				Spec map[string]interface{} `json:"spec"`
			}
			if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Metadata.Name) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid pipeline definition: metadata.name is required"})
				return
			}
			pName := strings.TrimSpace(payload.Metadata.Name)
			ns := payload.Metadata.Namespace
			if ns == "" {
				ns = reqNamespace
			}
			if ns == "" {
				ns = "default"
			}

			// Resolve or auto-create a default workspace record so the Tekton pipeline is persisted in DB
			var projectID string
			if projects, _, pErr := s.projectRepo.List(c.Request.Context(), 1, 10); pErr == nil && len(projects) > 0 {
				projectID = projects[0].ID
			} else {
				defaultProj := &repository.Project{
					ID:          uuid.New().String(),
					Name:        "default-workspace",
					Description: "Default workspace for Tekton resources",
					CreatedBy:   "admin",
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				}
				if allProjs, _, _ := s.projectRepo.List(c.Request.Context(), 1, 1); len(allProjs) > 0 {
					projectID = allProjs[0].ID
				} else {
					var adminID string
					_ = s.db.QueryRowContext(c.Request.Context(), `SELECT id FROM users LIMIT 1`).Scan(&adminID)
					if adminID != "" {
						defaultProj.CreatedBy = adminID
					}
					if cErr := s.projectRepo.Create(c.Request.Context(), defaultProj); cErr == nil {
						projectID = defaultProj.ID
					}
				}
			}

			specOut := gin.H{
				"description": pName,
				"tasks": []gin.H{
					{
						"name": "execute",
						"taskSpec": gin.H{
							"steps": []gin.H{
								{"name": "execute", "image": "alpine:latest"},
							},
						},
					},
				},
			}
			if len(payload.Spec) > 0 {
				for k, v := range payload.Spec {
					specOut[k] = v
				}
			}

			// Store full native Tekton spec + namespace as JSON in DSLContent
			storedBytes, _ := json.Marshal(gin.H{
				"namespace": ns,
				"spec":      specOut,
			})

			if existing, gErr := s.pipelineRepo.GetByProjectAndName(c.Request.Context(), projectID, pName); gErr == nil && existing != nil {
				_ = s.pipelineRepo.Delete(c.Request.Context(), existing.ID)
			}

			newPipe := &repository.Pipeline{
				ID:         uuid.New().String(),
				ProjectID:  projectID,
				Name:       pName,
				DSLContent: string(storedBytes),
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}
			if err := s.pipelineRepo.Create(c.Request.Context(), newPipe); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "Pipeline",
				"metadata": gin.H{
					"name":              newPipe.Name,
					"namespace":         ns,
					"uid":               newPipe.ID,
					"creationTimestamp": newPipe.CreatedAt.Format(time.RFC3339),
					"labels": gin.H{
						"app.kubernetes.io/managed-by": "tekton-dashboard",
					},
				},
				"spec": specOut,
			})
			return
		}

		if c.Request.Method == http.MethodDelete && isDetail {
			pipelines, err := s.pipelineRepo.ListAll(c.Request.Context())
			if err == nil {
				for _, p := range pipelines {
					if p.Name == pipelineName {
						_ = s.pipelineRepo.Delete(c.Request.Context(), p.ID)
						break
					}
				}
			}
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "Status",
				"status":     "Success",
				"message":    "Pipeline deleted successfully",
			})
			return
		}

		if isDetail {
			pipelines, err := s.pipelineRepo.ListAll(c.Request.Context())
			if err == nil {
				for _, p := range pipelines {
					if p.Name == pipelineName || p.ID == pipelineName {
						pNS := extractPipelineNamespace(p)
						c.JSON(http.StatusOK, gin.H{
							"apiVersion": "tekton.dev/v1",
							"kind":       "Pipeline",
							"metadata": gin.H{
								"name":              p.Name,
								"namespace":         pNS,
								"uid":               p.ID,
								"creationTimestamp": p.CreatedAt.Format(time.RFC3339),
								"labels": gin.H{
									"app.kubernetes.io/managed-by": "tekton-dashboard",
								},
							},
							"spec": extractPipelineSpec(p),
						})
						return
					}
				}
			}
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "Pipeline",
				"metadata": gin.H{
					"name":              pipelineName,
					"namespace":         reqNamespace,
					"uid":               pipelineName,
					"creationTimestamp": time.Now().UTC().Format(time.RFC3339),
				},
				"spec": gin.H{
					"tasks": []gin.H{},
				},
			})
			return
		}

		items := []gin.H{}
		pipelines, err := s.pipelineRepo.ListAll(c.Request.Context())
		if err == nil {
			for _, p := range pipelines {
				pNS := extractPipelineNamespace(p)
				if hasExplicitNamespace && pNS != reqNamespace {
					continue
				}
				items = append(items, gin.H{
					"apiVersion": "tekton.dev/v1",
					"kind":       "Pipeline",
					"metadata": gin.H{
						"name":              p.Name,
						"namespace":         pNS,
						"uid":               p.ID,
						"creationTimestamp": p.CreatedAt.Format(time.RFC3339),
						"labels": gin.H{
							"app.kubernetes.io/managed-by": "tekton-dashboard",
						},
					},
					"spec": extractPipelineSpec(p),
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "PipelineList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      items,
		})
		return

	case "pipelineruns":
		pipelineNames := make(map[string]string)
		if pipelines, pErr := s.pipelineRepo.ListAll(c.Request.Context()); pErr == nil {
			for _, p := range pipelines {
				pipelineNames[p.ID] = p.Name
			}
		}

		isDetail := false
		var runIdentifier string
		for i, p := range parts {
			if p == "pipelineruns" && i+1 < len(parts) && parts[i+1] != "" {
				isDetail = true
				runIdentifier = parts[i+1]
				break
			}
		}

		if c.Request.Method == http.MethodPost {
			// Support creating/triggering PipelineRun via Tekton API
			var body map[string]interface{}
			_ = c.ShouldBindJSON(&body)
			var targetPipelineID string
			if spec, ok := body["spec"].(map[string]interface{}); ok {
				if pref, ok := spec["pipelineRef"].(map[string]interface{}); ok {
					if pName, ok := pref["name"].(string); ok {
						for id, name := range pipelineNames {
							if name == pName || id == pName {
								targetPipelineID = id
								break
							}
						}
					}
				}
			}
			// If pipeline wasn't in DB yet, auto-create or use first pipeline
			if targetPipelineID == "" && len(pipelineNames) > 0 {
				for id := range pipelineNames {
					targetPipelineID = id
					break
				}
			}
			if targetPipelineID != "" {
				if triggered, tErr := s.runService.TriggerRun(c.Request.Context(), "admin", c.ClientIP(), targetPipelineID, "tekton-ui"); tErr == nil {
					pr, _ := s.formatPipelineRunAndTaskRun(*triggered, pipelineNames)
					c.JSON(http.StatusCreated, pr)
					return
				}
			}
		}

		if c.Request.Method == http.MethodDelete && isDetail {
			if runs, err := s.runRepo.ListAll(c.Request.Context()); err == nil {
				for _, r := range runs {
					if r.ID == runIdentifier || r.K8sPipelineRunName == runIdentifier || (len(r.ID) >= 8 && r.ID[:8]+"-run" == runIdentifier) {
						_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver, `DELETE FROM pipeline_runs WHERE id = ?`), r.ID)
						break
					}
				}
			}
			c.JSON(http.StatusOK, gin.H{"apiVersion": "tekton.dev/v1", "kind": "Status", "status": "Success"})
			return
		}

		if isDetail {
			runs, err := s.runRepo.ListAll(c.Request.Context())
			if err == nil {
				for _, r := range runs {
					if r.ID == runIdentifier || r.K8sPipelineRunName == runIdentifier {
						pr, _ := s.formatPipelineRunAndTaskRun(r, pipelineNames)
						c.JSON(http.StatusOK, pr)
						return
					}
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"message": "PipelineRun not found"})
			return
		}

		items := []gin.H{}
		runs, err := s.runRepo.ListAll(c.Request.Context())
		if err == nil {
			for _, r := range runs {
				pr, _ := s.formatPipelineRunAndTaskRun(r, pipelineNames)
				items = append(items, pr)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "PipelineRunList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      items,
		})
		return

	case "tasks":
		isDetail := false
		var taskName string
		for i, p := range parts {
			if p == "tasks" && i+1 < len(parts) && parts[i+1] != "" {
				isDetail = true
				taskName = parts[i+1]
				break
			}
		}

		if c.Request.Method == http.MethodPost {
			var payload struct {
				Metadata struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				} `json:"metadata"`
				Spec map[string]interface{} `json:"spec"`
			}
			if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Metadata.Name) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid task definition: metadata.name is required"})
				return
			}
			ns := payload.Metadata.Namespace
			if ns == "" {
				ns = reqNamespace
			}
			if ns == "" {
				ns = "default"
			}
			desc, _ := payload.Spec["description"].(string)
			image := "alpine:latest"
			script := ""
			if steps, ok := payload.Spec["steps"].([]interface{}); ok && len(steps) > 0 {
				if firstStep, ok := steps[0].(map[string]interface{}); ok {
					if img, ok := firstStep["image"].(string); ok && img != "" {
						image = img
					}
					if scr, ok := firstStep["script"].(string); ok {
						script = scr
					}
				}
			}
			specBytes, _ := json.Marshal(payload.Spec)
			newTask := &repository.Task{
				ID:          uuid.New().String(),
				Name:        strings.TrimSpace(payload.Metadata.Name),
				Namespace:   ns,
				Description: desc,
				Image:       image,
				Script:      script,
				SpecJSON:    string(specBytes),
				CreatedAt:   time.Now().UTC(),
			}
			// If task with same name exists in namespace, replace it
			_ = s.taskRepo.DeleteByNamespaceAndName(c.Request.Context(), ns, newTask.Name)
			if err := s.taskRepo.Create(c.Request.Context(), newTask); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, formatTaskResource(*newTask))
			return
		}

		if c.Request.Method == http.MethodDelete && isDetail {
			_ = s.taskRepo.DeleteByNamespaceAndName(c.Request.Context(), reqNamespace, taskName)
			c.JSON(http.StatusOK, gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "Status",
				"status":     "Success",
				"message":    "Task deleted successfully",
			})
			return
		}

		if isDetail {
			if t, err := s.taskRepo.GetByNamespaceAndName(c.Request.Context(), reqNamespace, taskName); err == nil && t != nil {
				c.JSON(http.StatusOK, formatTaskResource(*t))
				return
			}
			// Also check across all namespaces if not found in reqNamespace
			if allTasks, err := s.taskRepo.ListAll(c.Request.Context()); err == nil {
				for _, t := range allTasks {
					if t.Name == taskName {
						c.JSON(http.StatusOK, formatTaskResource(t))
						return
					}
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"message": "Task not found"})
			return
		}

		items := []gin.H{}
		if allTasks, err := s.taskRepo.ListAll(c.Request.Context()); err == nil {
			for _, t := range allTasks {
				items = append(items, formatTaskResource(t))
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "TaskList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      items,
		})
		return

	case "taskruns":
		pipelineNames := make(map[string]string)
		if pipelines, pErr := s.pipelineRepo.ListAll(c.Request.Context()); pErr == nil {
			for _, p := range pipelines {
				pipelineNames[p.ID] = p.Name
			}
		}

		isDetail := false
		var trName string
		for i, p := range parts {
			if p == "taskruns" && i+1 < len(parts) && parts[i+1] != "" {
				isDetail = true
				trName = parts[i+1]
				break
			}
		}

		if c.Request.Method == http.MethodPost {
			var body map[string]interface{}
			_ = c.ShouldBindJSON(&body)
			nowStr := time.Now().UTC().Format(time.RFC3339)
			meta, _ := body["metadata"].(map[string]interface{})
			name, _ := meta["name"].(string)
			if name == "" {
				gen, _ := meta["generateName"].(string)
				if gen == "" {
					gen = "taskrun-"
				}
				name = fmt.Sprintf("%s%d", gen, time.Now().Unix()%100000)
			}
			ns, _ := meta["namespace"].(string)
			if ns == "" {
				ns = reqNamespace
			}
			spec, _ := body["spec"].(map[string]interface{})
			labels, _ := meta["labels"].(map[string]interface{})
			if labels == nil {
				labels = map[string]interface{}{}
			}
			if tref, ok := spec["taskRef"].(map[string]interface{}); ok {
				if tName, ok := tref["name"].(string); ok && tName != "" {
					labels["tekton.dev/task"] = tName
				}
			}
			newTR := gin.H{
				"apiVersion": "tekton.dev/v1",
				"kind":       "TaskRun",
				"metadata": gin.H{
					"name":              name,
					"namespace":         ns,
					"uid":               uuid.New().String(),
					"creationTimestamp": nowStr,
					"labels":            labels,
				},
				"spec": spec,
				"status": gin.H{
					"podName":        name + "-pod",
					"startTime":      nowStr,
					"completionTime": nowStr,
					"conditions": []gin.H{
						{
							"type":    "Succeeded",
							"status":  "True",
							"reason":  "Succeeded",
							"message": "All Steps have completed executing",
						},
					},
					"steps": []gin.H{
						{
							"name":      "execute",
							"container": "step-execute",
							"terminated": gin.H{
								"exitCode":   0,
								"reason":     "Completed",
								"startedAt":  nowStr,
								"finishedAt": nowStr,
							},
						},
					},
					"taskSpec": gin.H{
						"steps": []gin.H{
							{"name": "execute", "image": "alpine:latest"},
						},
					},
				},
			}
			s.tektonFallbackMu.Lock()
			s.standaloneTaskRuns = append([]gin.H{newTR}, s.standaloneTaskRuns...)
			s.tektonFallbackMu.Unlock()
			c.JSON(http.StatusCreated, newTR)
			return
		}

		if c.Request.Method == http.MethodDelete && isDetail {
			s.tektonFallbackMu.Lock()
			filtered := make([]gin.H, 0, len(s.standaloneTaskRuns))
			for _, tr := range s.standaloneTaskRuns {
				meta, _ := tr["metadata"].(gin.H)
				if meta["name"] != trName {
					filtered = append(filtered, tr)
				}
			}
			s.standaloneTaskRuns = filtered
			s.tektonFallbackMu.Unlock()
			c.JSON(http.StatusOK, gin.H{"kind": "Status", "status": "Success"})
			return
		}

		allTRs := []gin.H{}
		s.tektonFallbackMu.RLock()
		allTRs = append(allTRs, s.standaloneTaskRuns...)
		s.tektonFallbackMu.RUnlock()

		if runs, err := s.runRepo.ListAll(c.Request.Context()); err == nil {
			for _, r := range runs {
				_, tr := s.formatPipelineRunAndTaskRun(r, pipelineNames)
				allTRs = append(allTRs, tr)
			}
		}

		if isDetail {
			for _, tr := range allTRs {
				if meta, ok := tr["metadata"].(gin.H); ok && meta["name"] == trName {
					c.JSON(http.StatusOK, tr)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"message": "TaskRun not found"})
			return
		}

		labelSelector := c.Query("labelSelector")
		items := []gin.H{}
		for _, tr := range allTRs {
			if labelSelector != "" && strings.Contains(labelSelector, "tekton.dev/pipelineRun=") {
				targetPR := strings.SplitAfter(labelSelector, "tekton.dev/pipelineRun=")[1]
				targetPR = strings.Split(targetPR, ",")[0]
				meta, _ := tr["metadata"].(gin.H)
				labels, _ := meta["labels"].(gin.H)
				if labels == nil || labels["tekton.dev/pipelineRun"] != targetPR {
					continue
				}
			}
			items = append(items, tr)
		}

		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "TaskRunList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      items,
		})
		return
	case "customruns":
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "CustomRunList",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      []gin.H{},
		})
		return
	case "eventlisteners", "triggers", "triggertemplates", "triggerbindings", "clusterinterceptors", "clustertriggerbindings", "interceptors":
		s.handleTektonTriggersFallback(c, resourceType, reqNamespace, hasExplicitNamespace, parts)
		return
	default:
		c.JSON(http.StatusOK, gin.H{
			"apiVersion": "tekton.dev/v1",
			"kind":       "List",
			"metadata":   gin.H{"resourceVersion": "1"},
			"items":      []gin.H{},
		})
	}
}
