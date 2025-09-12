# Development Guide

This guide covers setting up and developing the Leash Security Gateway.

## Prerequisites

### Required
- Docker and Docker Compose
- Git

### Optional (for Go development)
- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- Make

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/bendiamant/leash-gateway
cd leash-gateway
```

### 2. Development Setup (Docker Only)

If you don't have Go installed, you can still develop using Docker:

```bash
# Set up development environment
make dev-setup

# Start all services
make dev-up

# Check service health
make test-gateway
```

### 3. Development Setup (With Go)

If you have Go installed:

```bash
# Install development tools
make install-tools

# Generate protobuf code
make generate-proto

# Build binaries
make build

# Run tests
make test
```

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │───▶│   Envoy Proxy   │───▶│  LLM Providers  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Module Host    │
                       │  (gRPC Server)  │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │ Policy Pipeline │
                       │ Inspect→Policy→ │
                       │ Transform→Sink  │
                       └─────────────────┘
```

## Service Components

### Envoy Proxy (Port 8080)
- **Purpose**: HTTP reverse proxy and load balancer
- **Configuration**: `configs/envoy/bootstrap.yaml`
- **Features**: Path-based routing, ext_proc filter integration
- **Admin Interface**: http://localhost:9901

### Module Host (Port 50051)
- **Purpose**: gRPC server for processing requests through policy modules
- **Configuration**: `configs/gateway/config.yaml`
- **Health Check**: http://localhost:8081/health
- **Metrics**: http://localhost:9090/metrics

### Supporting Services
- **PostgreSQL**: Configuration and audit storage
- **Redis**: Caching and rate limiting
- **Prometheus**: Metrics collection
- **Grafana**: Metrics visualization

## Development Workflow

### 1. Making Changes

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes to code
# ...

# Format and lint
make fmt lint

# Run tests
make test

# Build and test locally
make build
make dev-up
make test-gateway
```

### 2. Working with Modules

```bash
# Create new module
mkdir -p internal/modules/my-module

# Implement module interface
# See internal/modules/README.md for details

# Test module
go test ./internal/modules/my-module/...
```

### 3. Configuration Changes

```bash
# Edit configuration
vim configs/gateway/config.yaml

# Restart services to pick up changes
make dev-down
make dev-up
```

### 4. Database Changes

```bash
# Edit init script
vim scripts/init-db.sql

# Recreate database
docker-compose -f docker/docker-compose.dev.yaml down -v
make dev-up
```

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# Run specific package tests
go test ./internal/config/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Start test environment
make dev-up

# Run integration tests
make test-integration
```

### Load Testing

```bash
# Install k6
# macOS: brew install k6
# Linux: See https://k6.io/docs/getting-started/installation/

# Run load tests
make load-test
```

## Debugging

### Logs

```bash
# View all service logs
make dev-logs

# View specific service logs
docker-compose -f docker/docker-compose.dev.yaml logs -f module-host
docker-compose -f docker/docker-compose.dev.yaml logs -f envoy
```

### Metrics and Monitoring

- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3000 (admin/admin)
- **Module Host Metrics**: http://localhost:9090/metrics
- **Envoy Admin**: http://localhost:9901

### Health Checks

```bash
# Gateway health
curl http://localhost:8080/health

# Module Host health  
curl http://localhost:8081/health

# Envoy readiness
curl http://localhost:9901/ready

# Database connection
docker-compose -f docker/docker-compose.dev.yaml exec postgres pg_isready -U leash
```

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_PATH` | `configs/gateway/config.yaml` | Configuration file path |
| `DATABASE_URL` | `postgres://leash:leash@localhost:5432/leash` | Database connection |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `LOG_LEVEL` | `info` | Logging level |

### Key Configuration Sections

#### Providers
```yaml
providers:
  openai:
    endpoint: "https://api.openai.com/v1"
    timeout: "30s"
    models:
      - name: "gpt-4o-mini"
        cost_per_1k_input_tokens: 0.15
```

#### Modules
```yaml
modules:
  rate-limiter:
    enabled: true
    type: "policy"
    config:
      default_limit: 1000
      default_window: "1h"
```

#### Observability
```yaml
observability:
  metrics:
    enabled: true
    port: 9090
  logging:
    level: "info"
    format: "json"
```

## Troubleshooting

### Common Issues

#### Port Conflicts
```bash
# Check what's using ports
lsof -i :8080
lsof -i :9090
lsof -i :50051

# Kill conflicting processes
sudo kill -9 <PID>
```

#### Docker Issues
```bash
# Clean up Docker resources
docker-compose -f docker/docker-compose.dev.yaml down -v
docker system prune -f

# Rebuild containers
make docker-build
make dev-up
```

#### Go Module Issues
```bash
# Clean module cache
go clean -modcache
go mod download
go mod tidy
```

### Getting Help

1. Check the logs: `make dev-logs`
2. Verify health endpoints: `make test-gateway`  
3. Review configuration: `configs/gateway/config.yaml`
4. Check GitHub Issues: https://github.com/bendiamant/leash-gateway/issues

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make lint test`
6. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.
