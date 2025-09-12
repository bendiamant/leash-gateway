package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bendiamant/leash-gateway/internal/config"
	"github.com/bendiamant/leash-gateway/internal/health"
	"github.com/bendiamant/leash-gateway/internal/logger"
	"github.com/bendiamant/leash-gateway/internal/metrics"
	modulelogger "github.com/bendiamant/leash-gateway/internal/modules/core/logger"
	"github.com/bendiamant/leash-gateway/internal/modules/core/ratelimiter"
	"github.com/bendiamant/leash-gateway/internal/modules/interface"
	"github.com/bendiamant/leash-gateway/internal/modules/pipeline"
	"github.com/bendiamant/leash-gateway/internal/modules/registry"
	pb "github.com/bendiamant/leash-gateway/proto/module"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.ModuleHost.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.ModuleHost.MaxSendMsgSize),
		grpc.KeepaliveParams(cfg.ModuleHost.KeepaliveParams()),
	)

	// Register services
	pb.RegisterModuleHostServer(grpcServer, moduleHost)
	
	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	
	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ModuleHost.GRPCPort))
	if err != nil {
		logger.Fatalf("Failed to listen on gRPC port %d: %v", cfg.ModuleHost.GRPCPort, err)
	}

	go func() {
		logger.Infof("gRPC server listening on port %d", cfg.ModuleHost.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Errorf("gRPC server failed: %v", err)
			cancel()
		}
	}()

	// Start HTTP server for health checks and metrics
	httpMux := http.NewServeMux()
	httpMux.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	httpMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ModuleHost.HealthPort),
		Handler: httpMux,
	}

	go func() {
		logger.Infof("HTTP server listening on port %d", cfg.ModuleHost.HealthPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server failed: %v", err)
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

	// Set health status to serving
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

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
	
	// Set health status to not serving
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Shutdown HTTP servers
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP server shutdown error: %v", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Metrics server shutdown error: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	logger.Info("Module Host shutdown complete")
}

// ModuleHostServer implements the ModuleHost gRPC service
type ModuleHostServer struct {
	pb.UnimplementedModuleHostServer
	logger   *zap.SugaredLogger
	config   *config.Config
	metrics  *metrics.Registry
	registry *registry.ModuleRegistry
	pipeline *pipeline.Pipeline
}

// ProcessRequest processes incoming requests through the module pipeline
func (s *ModuleHostServer) ProcessRequest(ctx context.Context, req *pb.ProcessRequestRequest) (*pb.ProcessRequestResponse, error) {
	start := time.Now()
	
	s.logger.Debugf("Processing request %s from tenant %s to provider %s", 
		req.RequestId, req.TenantId, req.Provider)

	// Create processing context
	processCtx := &interfaces.ProcessRequestContext{
		RequestID: req.RequestId,
		Timestamp: time.Now(),
		TenantID:  req.TenantId,
		Provider:  req.Provider,
		Method:    "POST", // Default for LLM requests
		Path:      fmt.Sprintf("/v1/%s/chat/completions", req.Provider),
		Headers:   make(map[string]string),
		Body:      []byte{}, // Would be populated from actual request
		Annotations: make(map[string]interface{}),
	}

	// Process through module pipeline
	result, err := s.pipeline.ProcessRequest(ctx, processCtx)
	if err != nil {
		s.logger.Errorf("Pipeline processing failed for request %s: %v", req.RequestId, err)
		s.metrics.RequestsTotal.WithLabelValues(
			req.TenantId, req.Provider, "unknown", "POST", "500",
		).Inc()
		
		return &pb.ProcessRequestResponse{
			Action:           pb.Action_ACTION_BLOCK,
			ProcessingTimeMs: time.Since(start).Milliseconds(),
			Annotations:      map[string]string{"error": err.Error()},
		}, nil
	}

	// Record request metrics
	status := "200"
	if result.Action == interfaces.ActionBlock {
		status = "403"
	}
	
	s.metrics.RequestsTotal.WithLabelValues(
		req.TenantId, req.Provider, "unknown", "POST", status,
	).Inc()

	// Convert result to protobuf response
	response := &pb.ProcessRequestResponse{
		Action:           convertActionToProto(result.Action),
		ProcessingTimeMs: result.ProcessingTime.Milliseconds(),
		Annotations:      convertAnnotationsToStringMap(result.Annotations),
		Metadata:         result.Metadata,
	}

	if result.Action == interfaces.ActionBlock {
		response.Annotations["block_reason"] = result.BlockReason
	}

	s.logger.Debugf("Request %s processed in %dms, action: %s", 
		req.RequestId, response.ProcessingTimeMs, response.Action.String())

	return response, nil
}


// Health returns the health status of the module host
func (s *ModuleHostServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	// Check module health
	moduleHealth := s.registry.HealthCheck(ctx)
	allHealthy := true
	for _, health := range moduleHealth {
		if health.Status != interfaces.HealthStateHealthy {
			allHealthy = false
			break
		}
	}

	status := pb.HealthStatus_HEALTH_STATUS_HEALTHY
	message := "Module Host is healthy"
	if !allHealthy {
		status = pb.HealthStatus_HEALTH_STATUS_DEGRADED
		message = "Some modules are unhealthy"
	}

	return &pb.HealthResponse{
		Status:  status,
		Message: message,
		Details: map[string]string{
			"version":        version,
			"build_time":     buildTime,
			"git_commit":     gitCommit,
			"modules_count":  fmt.Sprintf("%d", len(s.registry.List())),
			"pipeline_status": fmt.Sprintf("%v", s.pipeline.GetPipelineStatus()),
		},
	}, nil
}

// Helper functions for type conversion
func convertActionToProto(action interfaces.Action) pb.Action {
	switch action {
	case interfaces.ActionContinue:
		return pb.Action_ACTION_CONTINUE
	case interfaces.ActionBlock:
		return pb.Action_ACTION_BLOCK
	default:
		return pb.Action_ACTION_CONTINUE
	}
}

func convertAnnotationsToStringMap(annotations map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range annotations {
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}
