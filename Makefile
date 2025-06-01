# Makefile for SIP B2BUA project

# Variables
GO_VERSION := 1.21
PROJECT_NAME := sip-b2bua
BINARY_NAME := b2bua-server
DOCKER_IMAGE := $(PROJECT_NAME):latest

# Directories
SRC_DIR := .
BUILD_DIR := build
PROTO_DIR := proto
PROTO_GEN_DIR := proto/gen

# Build flags
LDFLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Default target
.PHONY: all
all: clean test build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Generate code (protobuf, mocks, etc.)
.PHONY: generate
generate:
	@echo "Generating code..."
	@go generate ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run ./...

# Run tests
.PHONY: test
test:
	@echo "Running unit tests..."
	@go test -race -cover ./pkg/...
	@go test -race -cover ./internal/...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test -race -tags=integration ./test/integration/...

# Run all tests
.PHONY: test-all
test-all: test test-integration

# Benchmark tests
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Build application
.PHONY: build
build:
	@echo "Building application..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Run application
.PHONY: run
run: build
	@echo "Running application..."
	@$(BUILD_DIR)/$(BINARY_NAME) --config=configs/development.yaml

# Run with development config
.PHONY: run-dev
run-dev:
	@echo "Running in development mode..."
	@go run ./cmd/server --config=configs/development.yaml --log-level=debug

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

# Docker run
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run --rm -p 5060:5060/udp -p 8080:8080 -p 9090:9090 $(DOCKER_IMAGE)

# Security scan
.PHONY: security
security:
	@echo "Running security scan..."
	@gosec ./...

# Vulnerability check
.PHONY: vuln-check
vuln-check:
	@echo "Checking for vulnerabilities..."
	@govulncheck ./...

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securecodewarrior/govulncheck@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf files
.PHONY: proto
proto:
	@echo "Generating protobuf files..."
	@mkdir -p $(PROTO_GEN_DIR)
	@protoc --go_out=$(PROTO_GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# Start development environment
.PHONY: dev-env
dev-env:
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.dev.yml up -d

# Stop development environment
.PHONY: dev-env-stop
dev-env-stop:
	@echo "Stopping development environment..."
	@docker-compose -f docker-compose.dev.yml down

# Database migration (if needed)
.PHONY: migrate
migrate:
	@echo "Running database migrations..."
	# Add migration commands here

# Performance testing
.PHONY: perf-test
perf-test:
	@echo "Running performance tests..."
	@go test -bench=. -benchtime=10s -benchmem ./...

# Load testing
.PHONY: load-test
load-test:
	@echo "Running load tests..."
	# Add load testing commands here (e.g., using vegeta, k6, etc.)

# Create release
.PHONY: release
release: clean test build-all
	@echo "Creating release..."
	@mkdir -p $(BUILD_DIR)/release
	@tar -czf $(BUILD_DIR)/release/$(PROJECT_NAME)-$(shell git describe --tags --always).tar.gz -C $(BUILD_DIR) .

# Health check
.PHONY: health
health:
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Health check failed"

# Metrics check
.PHONY: metrics
metrics:
	@echo "Checking metrics endpoint..."
	@curl -f http://localhost:9090/metrics || echo "Metrics check failed"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Clean, test, and build"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  generate      - Generate code"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  test          - Run unit tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  test-all      - Run all tests"
	@echo "  bench         - Run benchmarks"
	@echo "  build         - Build application"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  run           - Run application"
	@echo "  run-dev       - Run in development mode"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  security      - Run security scan"
	@echo "  vuln-check    - Check for vulnerabilities"
	@echo "  install-tools - Install development tools"
	@echo "  proto         - Generate protobuf files"
	@echo "  dev-env       - Start development environment"
	@echo "  dev-env-stop  - Stop development environment"
	@echo "  perf-test     - Run performance tests"
	@echo "  load-test     - Run load tests"
	@echo "  release       - Create release"
	@echo "  health        - Check application health"
	@echo "  metrics       - Check metrics endpoint"
	@echo "  help          - Show this help"

# CI/CD health check
.PHONY: ci-check
ci-check:
	@echo "Running CI/CD health check..."
	@./scripts/ci-health-check.sh

# CI/CD targets
.PHONY: ci
ci: deps fmt lint test-all vuln-check build

.PHONY: cd
cd: ci docker-build
