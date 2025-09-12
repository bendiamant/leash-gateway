# Makefile for Leash Gateway
.PHONY: all build test clean docker-build docker-push install-tools generate-proto help

# Variables
BINARY_NAME=leash-gateway
MODULE_HOST_BINARY=leash-module-host
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Check if Go is installed
check-go:
	@which go > /dev/null || (echo "Go is not installed. Please install Go 1.21+ from https://golang.org/dl/" && exit 1)

# Install development tools
install-tools: check-go
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-health-probe@latest

# Check if protoc is installed
check-protoc:
	@which protoc > /dev/null || (echo "protoc is not installed. Please install Protocol Buffers compiler" && exit 1)

# Generate protobuf code
generate-proto: check-go check-protoc
	@echo "Generating protobuf code..."
	@mkdir -p proto/module
	@protoc --go_out=. --go-grpc_out=. proto/*.proto

# Download dependencies
deps: check-go
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Build binaries
build: check-go generate-proto deps
	@echo "Building gateway..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/gateway/main.go
	@echo "Building module host..."
	@go build $(LDFLAGS) -o bin/$(MODULE_HOST_BINARY) cmd/module-host/main.go

# Run tests
test: check-go
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-integration: check-go
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

test-e2e: check-go
	@echo "Running end-to-end tests..."
	@go test -v -tags=e2e ./tests/e2e/...

# Linting
lint: check-go
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt: check-go
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w . 2>/dev/null || true

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out
	@rm -rf proto/module/*.pb.go

# Docker builds
docker-build:
	@echo "Building Docker images..."
	@docker build -f docker/Dockerfile.module-host -t leash-security/module-host:$(VERSION) .

docker-push:
	@echo "Pushing Docker images..."
	@docker push leash-security/module-host:$(VERSION)

# Development environment
dev-up:
	@echo "Starting development environment..."
	@docker-compose -f docker/docker-compose.dev.yaml up -d

dev-down:
	@echo "Stopping development environment..."
	@docker-compose -f docker/docker-compose.dev.yaml down

dev-logs:
	@echo "Showing development logs..."
	@docker-compose -f docker/docker-compose.dev.yaml logs -f

# Quick development setup (without Go installed)
dev-setup:
	@echo "Setting up development environment..."
	@echo "Note: This requires Docker and Docker Compose to be installed"
	@docker-compose -f docker/docker-compose.dev.yaml build
	@docker-compose -f docker/docker-compose.dev.yaml up -d postgres redis
	@echo "Development environment setup complete!"
	@echo "Run 'make dev-up' to start all services"

# Test the gateway
test-gateway:
	@echo "Testing gateway connectivity..."
	@curl -f http://localhost:8080/health || echo "Gateway health check failed"
	@curl -f http://localhost:9901/ready || echo "Envoy admin check failed"
	@curl -f http://localhost:8081/health || echo "Module host health check failed"

# Load test (requires k6)
load-test:
	@echo "Running load tests..."
	@k6 run tests/load/basic-load.js 2>/dev/null || echo "k6 not installed, skipping load test"

# Security scanning
security-scan: check-go
	@echo "Running security scan..."
	@gosec ./... 2>/dev/null || echo "gosec not installed, skipping security scan"

# Documentation
docs-serve:
	@echo "Serving documentation..."
	@python3 -m http.server 8000 -d docs 2>/dev/null || echo "Python not available for docs server"

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build all binaries (requires Go)"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  docker-build   - Build Docker images"
	@echo "  dev-setup      - Set up development environment (Docker only)"
	@echo "  dev-up         - Start development environment"
	@echo "  dev-down       - Stop development environment"
	@echo "  test-gateway   - Test gateway connectivity"
	@echo "  load-test      - Run load tests (requires k6)"
	@echo "  security-scan  - Run security scans"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Quick start (without Go installed):"
	@echo "  1. make dev-setup"
	@echo "  2. make dev-up"
	@echo "  3. make test-gateway"
