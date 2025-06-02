# Multi-stage Dockerfile for SIP B2BUA

# Build stage
FROM golang:1.24.3-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT_SHA
RUN BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    COMMIT_SHA=${COMMIT_SHA:-unknown} && \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commitSHA=${COMMIT_SHA}" \
    -o b2bua-server \
    ./cmd/b2bua

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    curl \
    procps \
    sngrep\
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/b2bua-server .

# Create necessary directories
RUN mkdir -p /app/logs /app/data /app/configs

# Make binary executable
RUN chmod +x /app/b2bua-server

# Expose ports (using high ports to avoid privilege issues)
EXPOSE 5060/udp 8080 9090 50051

# Set default environment variables
ENV LOG_LEVEL=info

# Run as root for now to avoid all permission issues
# Command to run with production config path by default
ENTRYPOINT ["./b2bua-server", "--config=/etc/voice-ferry/config.yaml"]