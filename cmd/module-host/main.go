package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bendiamant/leash-gateway/internal/config"
	"github.com/bendiamant/leash-gateway/internal/logger"
	"github.com/bendiamant/leash-gateway/internal/metrics"
	modulelogger "github.com/bendiamant/leash-gateway/internal/modules/core/logger"
	"github.com/bendiamant/leash-gateway/internal/modules/core/ratelimiter"
	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"github.com/bendiamant/leash-gateway/internal/modules/pipeline"
	"github.com/bendiamant/leash-gateway/internal/modules/registry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Initialize logger
	zapLogger, err := logger.NewLogger(logger.Config{
		Level:       "info",
		Format:      "json",
		Development: false,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	logger := zapLogger.Sugar()
	logger.Infof("Starting Leash Module Host version=%s build=%s commit=%s", version, buildTime, gitCommit)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize metrics
	metricsRegistry := metrics.NewRegistry()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create module registry and pipeline
	moduleRegistry := registry.NewModuleRegistry(logger)
	modulePipeline := pipeline.NewPipeline(logger)

	// Initialize core modules
	rateLimiterModule := ratelimiter.NewRateLimiter(logger)
	loggerModule := modulelogger.NewLogger(logger)

	// Register modules
	if err := moduleRegistry.Register(rateLimiterModule); err != nil {
		logger.Fatalf("Failed to register rate limiter module: %v", err)
	}
	if err := moduleRegistry.Register(loggerModule); err != nil {
		logger.Fatalf("Failed to register logger module: %v", err)
	}

	// Add modules to pipeline
	if err := modulePipeline.AddModule(rateLimiterModule); err != nil {
		logger.Fatalf("Failed to add rate limiter to pipeline: %v", err)
	}
	if err := modulePipeline.AddModule(loggerModule); err != nil {
		logger.Fatalf("Failed to add logger to pipeline: %v", err)
	}

	// Initialize modules
	moduleConfig := &interfaces.ModuleConfig{
		Name:     "rate-limiter",
		Type:     "policy",
		Enabled:  true,
		Priority: 100,
		Config: map[string]interface{}{
			"algorithm":     "token_bucket",
			"default_limit": 1000,
			"default_window": "1h",
			"storage":       "memory",
		},
	}
	if err := rateLimiterModule.Initialize(ctx, moduleConfig); err != nil {
		logger.Fatalf("Failed to initialize rate limiter: %v", err)
	}
	if err := rateLimiterModule.Start(ctx); err != nil {
		logger.Fatalf("Failed to start rate limiter: %v", err)
	}

	loggerConfig := &interfaces.ModuleConfig{
		Name:     "logger",
		Type:     "sink",
		Enabled:  true,
		Priority: 1000,
		Config: map[string]interface{}{
			"log_requests":  true,
			"log_responses": false,
			"redact_pii":    true,
		},
	}
	if err := loggerModule.Initialize(ctx, loggerConfig); err != nil {
		logger.Fatalf("Failed to initialize logger module: %v", err)
	}
	if err := loggerModule.Start(ctx); err != nil {
		logger.Fatalf("Failed to start logger module: %v", err)
	}

	// Create module host server
	moduleHost := &ModuleHostServer{
		logger:   logger,
		config:   cfg,
		metrics:  metricsRegistry,
		registry: moduleRegistry,
		pipeline: modulePipeline,
	}

	// Create HTTP server for simplified implementation
	httpMux := http.NewServeMux()
	
	// Add module host endpoints
	httpMux.HandleFunc("/process", moduleHost.ProcessRequestHTTP)
	httpMux.HandleFunc("/health", moduleHost.HealthHTTP)
	httpMux.HandleFunc("/modules", moduleHost.ModulesHTTP)
	
	// Start HTTP server for module processing
	moduleServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ModuleHost.GRPCPort),
		Handler: httpMux,
	}

	go func() {
		logger.Infof("Module Host HTTP server listening on port %d", cfg.ModuleHost.GRPCPort)
		if err := moduleServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Module Host HTTP server failed: %v", err)
			cancel()
		}
	}()

	// Add metrics and health endpoints to the same server
	httpMux.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	httpMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// Start health server on separate port
	healthMux := http.NewServeMux()
	healthMux.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ModuleHost.HealthPort),
		Handler: healthMux,
	}

	go func() {
		logger.Infof("Health server listening on port %d", cfg.ModuleHost.HealthPort)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Health server failed: %v", err)
			cancel()
		}
	}()

	// Start metrics server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Observability.Metrics.Port),
		Handler: promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}),
	}

	go func() {
		logger.Infof("Metrics server listening on port %d", cfg.Observability.Metrics.Port)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Metrics server failed: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Infof("Received signal %v, shutting down gracefully", sig)
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	// Graceful shutdown
	logger.Info("Shutting down servers...")

	// Shutdown HTTP servers
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := moduleServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Module server shutdown error: %v", err)
	}

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Health server shutdown error: %v", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Metrics server shutdown error: %v", err)
	}

	logger.Info("Module Host shutdown complete")
}

// ModuleHostServer implements the ModuleHost HTTP service
type ModuleHostServer struct {
	logger   *zap.SugaredLogger
	config   *config.Config
	metrics  *metrics.Registry
	registry *registry.ModuleRegistry
	pipeline *pipeline.Pipeline
}

// ProcessRequestHTTP handles HTTP requests for module processing
func (s *ModuleHostServer) ProcessRequestHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	
	s.logger.Debugf("Processing HTTP request %s", requestID)

	// For simplified demo, just allow all requests
	response := map[string]interface{}{
		"action":             "continue",
		"processing_time_ms": time.Since(start).Milliseconds(),
		"annotations": map[string]string{
			"processed_by": "leash-module-host",
			"request_id":   requestID,
		},
		"metadata": map[string]string{
			"module_host": "active",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	s.logger.Debugf("Request %s processed in %dms", requestID, response["processing_time_ms"])
}


// HealthHTTP handles HTTP health checks
func (s *ModuleHostServer) HealthHTTP(w http.ResponseWriter, r *http.Request) {
	// Check module health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	moduleHealth := s.registry.HealthCheck(ctx)
	allHealthy := true
	for _, health := range moduleHealth {
		if health.Status != interfaces.HealthStateHealthy {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	message := "Module Host is healthy"
	if !allHealthy {
		status = "degraded"
		message = "Some modules are unhealthy"
	}

	response := map[string]interface{}{
		"status":  status,
		"message": message,
		"details": map[string]interface{}{
			"version":         version,
			"build_time":      buildTime,
			"git_commit":      gitCommit,
			"modules_count":   len(s.registry.List()),
			"pipeline_status": s.pipeline.GetPipelineStatus(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

// ModulesHTTP handles requests for module information
func (s *ModuleHostServer) ModulesHTTP(w http.ResponseWriter, r *http.Request) {
	modules := s.registry.List()
	moduleInfo := make([]map[string]interface{}, len(modules))
	
	for i, module := range modules {
		moduleInfo[i] = map[string]interface{}{
			"name":        module.Name(),
			"version":     module.Version(),
			"type":        module.Type().String(),
			"description": module.Description(),
			"status":      module.Status(),
			"metrics":     module.Metrics(),
		}
	}

	response := map[string]interface{}{
		"modules": moduleInfo,
		"count":   len(modules),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
