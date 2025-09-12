# Technical Context: Leash Security Gateway

## Technology Stack

### Core Infrastructure
- **Proxy Layer**: Envoy Proxy v1.28+ (HTTP/2, gRPC, ext_proc filter)
- **Module Runtime**: Go 1.21+ (gRPC server, plugin system)
- **Configuration**: YAML-based with hot-reload capability
- **Communication**: gRPC with Protocol Buffers

### Data Storage
- **Primary Database**: PostgreSQL 15+ (tenant config, usage tracking)
- **Cache Layer**: Redis 7+ (rate limiting, response caching)
- **Time Series**: Prometheus (metrics collection)
- **Analytics**: ClickHouse (optional, for advanced analytics)

### Observability Stack
- **Metrics**: Prometheus + Grafana
- **Logging**: Structured JSON with correlation IDs
- **Tracing**: OpenTelemetry with Jaeger
- **Alerting**: Prometheus AlertManager

### Development Tools
- **Build System**: Go modules, Makefile, Docker multi-stage builds
- **Testing**: Go testing, integration tests, K6 load testing
- **CI/CD**: GitHub Actions, automated security scanning
- **Documentation**: Markdown with MkDocs

## Key Technical Constraints

### Performance Requirements
- **Gateway Overhead**: <4ms P50, <10ms P95
- **Throughput**: >1000 RPS sustained, 5000 RPS peak
- **Module Processing**: <2ms P95 per module
- **Memory Usage**: <512MB base, <256MB per module

### Reliability Requirements
- **Uptime**: >99.9% availability
- **Error Rate**: <0.1% under normal load
- **Recovery Time**: <5s for provider failover
- **Data Consistency**: 100% tenant isolation

### Security Requirements
- **Encryption**: TLS 1.3 in transit, AES-256 at rest
- **Authentication**: API key validation, optional mTLS
- **Isolation**: Process-level module sandboxing
- **Audit**: Complete request/response logging

## Architecture Decisions

### ADR-001: Envoy as Data Plane
**Status**: Accepted  
**Context**: Need production-proven proxy with rich HTTP features  
**Decision**: Use Envoy with ext_proc filter for request interception  
**Consequences**: 
- ✅ Battle-tested at scale
- ✅ Rich observability features
- ✅ Native gRPC integration
- ❌ Additional operational complexity

### ADR-002: gRPC Module Communication
**Status**: Accepted  
**Context**: Need efficient, type-safe communication between proxy and modules  
**Decision**: Use gRPC with protobuf schemas  
**Consequences**:
- ✅ Type safety and code generation
- ✅ Streaming support for real-time processing
- ✅ Language-agnostic module development
- ❌ Additional complexity vs HTTP REST

### ADR-003: Go for Module Runtime
**Status**: Accepted  
**Context**: Need high-performance, low-latency module execution  
**Decision**: Go-based gRPC server with plugin system  
**Consequences**:
- ✅ Excellent performance characteristics
- ✅ Strong concurrency primitives
- ✅ Rich ecosystem for infrastructure tools
- ❌ Module developers need Go knowledge

### ADR-004: Configuration-Based Integration
**Status**: Accepted  
**Context**: Minimize application changes for adoption  
**Decision**: Path-based routing with URL rewriting  
**Consequences**:
- ✅ Minimal integration effort (URL change only)
- ✅ Works with any HTTP client
- ✅ No SDK requirement
- ❌ Less flexibility than SDK-based approach

## Development Environment

### Local Development Setup
```bash
# Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Protocol Buffers compiler
- Make

# Quick start
git clone https://github.com/leash-security/gateway
cd gateway
make dev-setup
make dev-up
```

### Development Dependencies
```go
// Core dependencies
- github.com/envoyproxy/go-control-plane
- google.golang.org/grpc
- google.golang.org/protobuf
- github.com/prometheus/client_golang
- github.com/redis/go-redis/v9
- github.com/lib/pq (PostgreSQL driver)

// Development dependencies  
- github.com/stretchr/testify (testing)
- github.com/golangci/golangci-lint (linting)
- github.com/air-verse/air (hot reload)
```

### Testing Strategy
```bash
# Unit tests
make test

# Integration tests
make test-integration

# End-to-end tests
make test-e2e

# Load tests
make load-test

# Security tests
make security-scan
```

## Deployment Architecture

### Container Images
- **Gateway**: `leash-security/gateway:latest` (Envoy + configs)
- **Module Host**: `leash-security/module-host:latest` (Go gRPC server)
- **Dependencies**: PostgreSQL, Redis, Prometheus, Grafana

### Kubernetes Resources
```yaml
# Core components
- Deployment: gateway pods (3 replicas)
- Service: LoadBalancer for external traffic
- ConfigMap: Envoy and module configurations
- Secret: API keys and database credentials
- PersistentVolume: Database and cache storage
```

### Self-Hosted Deployment
```bash
# Docker Compose (development)
docker-compose -f docker/docker-compose.dev.yaml up

# Kubernetes (production)
helm install leash-gateway deployments/helm/leash-gateway/
```

## Integration Patterns

### SDK Architecture (Optional Enhancement)
```typescript
// TypeScript SDK structure
@leash-security/sdk/
├── core/           # Core client implementation
├── providers/      # Provider-specific adapters  
├── middleware/     # Retry, caching, fallback logic
├── integrations/   # Framework integrations
└── types/          # TypeScript definitions
```

### Framework Integrations
- **LangChain**: Custom LLM class with gateway routing
- **CrewAI**: Agent configuration with gateway endpoints
- **Vercel AI SDK**: Provider configuration override
- **OpenAI SDK**: Base URL configuration change

## Monitoring & Operations

### Health Check Endpoints
```bash
# Gateway health
curl http://localhost:8080/health

# Module Host health  
curl http://localhost:8081/health

# Envoy admin interface
curl http://localhost:9901/stats
```

### Metrics Collection
```yaml
# Key metrics
- leash_gateway_requests_total{tenant,provider,status}
- leash_gateway_request_duration_seconds{tenant,provider}
- leash_module_processing_duration_seconds{module,tenant}
- leash_provider_latency_seconds{provider}
- leash_cost_usd_total{tenant,provider,model}
```

### Log Format
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info", 
  "request_id": "req_abc123",
  "tenant_id": "acme-corp",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "latency_ms": 1247,
  "tokens_used": 225,
  "cost_usd": 0.004500,
  "message": "Request completed successfully"
}
```

## Security Implementation

### Authentication Flow
1. **API Key Extraction**: From Authorization header or X-API-Key
2. **Key Validation**: Against tenant database
3. **Tenant Resolution**: Map key to tenant ID
4. **Context Injection**: Add tenant context to request

### Module Sandboxing
```go
// Resource limits per module
type ResourceLimits struct {
    MaxMemoryMB     int           `yaml:"max_memory_mb"`
    MaxCPUPercent   int           `yaml:"max_cpu_percent"`
    MaxExecutionTime time.Duration `yaml:"max_execution_time"`
}
```

### Data Protection
- **PII Redaction**: Configurable field masking
- **Encryption**: Database-level encryption for sensitive fields
- **Audit Logging**: Complete request/response trails
- **TTL Policies**: Automatic data expiration
