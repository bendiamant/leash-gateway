package health

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC health checking protocol
type Server struct {
	grpc_health_v1.UnimplementedHealthServer
	
	mu       sync.RWMutex
	statusMap map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

// NewServer creates a new health check server
func NewServer() *Server {
	return &Server{
		statusMap: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
}

// Check implements the health check method
func (s *Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	service := req.Service
	servingStatus, exists := s.statusMap[service]
	
	if !exists {
		return nil, status.Errorf(codes.NotFound, "service %s not found", service)
	}
	
	return &grpc_health_v1.HealthCheckResponse{
		Status: servingStatus,
	}, nil
}

// Watch implements the health check streaming method
func (s *Server) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	service := req.Service
	
	// Send initial status
	s.mu.RLock()
	servingStatus, exists := s.statusMap[service]
	s.mu.RUnlock()
	
	if !exists {
		return status.Errorf(codes.NotFound, "service %s not found", service)
	}
	
	if err := stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: servingStatus,
	}); err != nil {
		return err
	}
	
	// Keep the stream open (simplified implementation)
	// In a real implementation, you would watch for status changes
	<-stream.Context().Done()
	return nil
}

// SetServingStatus sets the serving status for a service
func (s *Server) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.statusMap[service] = status
}

// GetServingStatus gets the serving status for a service
func (s *Server) GetServingStatus(service string) grpc_health_v1.HealthCheckResponse_ServingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	servingStatus, exists := s.statusMap[service]
	if !exists {
		return grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN
	}
	
	return servingStatus
}
