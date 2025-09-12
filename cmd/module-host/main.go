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

	// Create module host server
	moduleHost := &ModuleHostServer{
		logger:  logger,
		config:  cfg,
		metrics: metricsRegistry,
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
	logger  *zap.SugaredLogger
	config  *config.Config
	metrics *metrics.Registry
}

// ProcessRequest processes incoming requests through the module pipeline
func (s *ModuleHostServer) ProcessRequest(ctx context.Context, req *pb.ProcessRequestRequest) (*pb.ProcessRequestResponse, error) {
	start := time.Now()
	
	s.logger.Debugf("Processing request %s from tenant %s to provider %s", 
		req.RequestId, req.TenantId, req.Provider)

	// Record request metrics
	s.metrics.RequestsTotal.WithLabelValues(
		req.TenantId,
		req.Provider,
		"unknown", // model - would be extracted from request body
		"POST",
		"200", // status - would be determined by processing
	).Inc()

	// For now, just allow all requests through (basic implementation)
	response := &pb.ProcessRequestResponse{
		Action:             pb.Action_ACTION_CONTINUE,
		ProcessingTimeMs:   time.Since(start).Milliseconds(),
		Annotations:        make(map[string]string),
		Metadata:          make(map[string]string),
	}

	// Add processing metadata
	response.Annotations["processed_by"] = "leash-module-host"
	response.Annotations["processing_time_ms"] = fmt.Sprintf("%d", response.ProcessingTimeMs)
	response.Metadata["tenant_id"] = req.TenantId
	response.Metadata["provider"] = req.Provider

	s.logger.Debugf("Request %s processed in %dms, action: %s", 
		req.RequestId, response.ProcessingTimeMs, response.Action.String())

	return response, nil
}


// Health returns the health status of the module host
func (s *ModuleHostServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:  pb.HealthStatus_HEALTH_STATUS_HEALTHY,
		Message: "Module Host is healthy",
		Details: map[string]string{
			"version":    version,
			"build_time": buildTime,
			"git_commit": gitCommit,
			"uptime":     time.Since(time.Now()).String(), // This would be tracked properly in real implementation
		},
	}, nil
}
