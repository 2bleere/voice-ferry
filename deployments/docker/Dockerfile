# Production Dockerfile for Voice Ferry SIP B2BUA
# Multi-stage build optimized for production use

FROM golang:1.24.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    upx

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies with verification
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build info
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static' \
              -X github.com/2bleere/voice-ferry/pkg/version.Version=${VERSION} \
              -X github.com/2bleere/voice-ferry/pkg/version.BuildTime=${BUILD_TIME} \
              -X github.com/2bleere/voice-ferry/pkg/version.GitCommit=${GIT_COMMIT}" \
    -o b2bua ./cmd/b2bua

# Compress binary (optional, saves ~30% space)
RUN upx --lzma b2bua

# Final production stage
FROM scratch

# Import ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary from builder stage
COPY --from=builder /app/b2bua /usr/local/bin/b2bua

# Copy default configuration
COPY --from=builder /app/configs/production.yaml /etc/voice-ferry/config.yaml

# Set metadata
LABEL maintainer="Voice Ferry Team" \
      description="Voice Ferry SIP B2BUA - Cloud-native SIP proxy" \
      version="${VERSION}" \
      org.opencontainers.image.title="Voice Ferry SIP B2BUA" \
      org.opencontainers.image.description="High-performance SIP B2BUA for cloud deployments" \
      org.opencontainers.image.url="https://github.com/2bleere/voice-ferry" \
      org.opencontainers.image.source="https://github.com/2bleere/voice-ferry" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.licenses="MIT"

# Expose ports
EXPOSE 5060/udp 5060/tcp 5061/tcp 50051/tcp 8080/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["/usr/local/bin/b2bua", "-health-check"]

# Use non-root user (defined at runtime)
USER 1001:1001

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/b2bua"]

# Default command
CMD ["-config", "/etc/voice-ferry/config.yaml"]
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz/live || exit 1

# Set entrypoint
ENTRYPOINT ["./b2bua"]

# Default command
CMD ["-config", "/etc/b2bua/config.yaml"]
