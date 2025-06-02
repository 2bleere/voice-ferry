# Manual Docker Image Build Guide

This guide provides comprehensive instructions for manually building Docker images for the Voice Ferry SIP B2BUA project.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Basic Docker Build](#basic-docker-build)
- [Web-UI Docker Build](#web-ui-docker-build)
- [Multi-Architecture Builds](#multi-architecture-builds)
- [Build Arguments and Customization](#build-arguments-and-customization)
- [Local Development Builds](#local-development-builds)
- [Production Builds](#production-builds)
- [Full Stack Builds](#full-stack-builds)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Prerequisites

### Required Software

1. **Docker** (version 20.10+ recommended)
   ```bash
   # Check Docker version
   docker --version
   
   # Verify Docker daemon is running
   docker info
   ```

2. **Git** (for version information)
   ```bash
   git --version
   ```

3. **Make** (optional, for using Makefile targets)
   ```bash
   make --version
   ```

4. **Node.js and npm** (for web-ui builds)
   ```bash
   # Check Node.js version (18+ recommended)
   node --version
   
   # Check npm version
   npm --version
   ```

### System Requirements

- **Minimum**: 4GB RAM, 10GB free disk space
- **Recommended**: 8GB RAM, 20GB free disk space
- **Architecture**: amd64 or arm64

## Quick Start

### 1. Basic Build

```bash
# Clone and navigate to project
cd /path/to/voice-ferry

# Build with default settings
docker build -t voice-ferry:latest .
```

### 2. Run the Built Image

```bash
# Run with basic configuration
docker run -p 5060:5060/udp -p 8080:8080 voice-ferry:latest
```

### 3. Build Web-UI (Optional)

```bash
# Navigate to web-ui directory
cd web-ui

# Build web-ui image
docker build -t voice-ferry-ui:latest .

# Run web-ui
docker run -p 3000:3000 voice-ferry-ui:latest
```

## Basic Docker Build

### Standard Build Command

```bash
# Basic build with latest tag
docker build -t voice-ferry:latest .

# Build with specific tag
docker build -t voice-ferry:v1.0.0 .

# Build with multiple tags
docker build -t voice-ferry:latest -t voice-ferry:v1.0.0 .
```

### Build with Custom Context

```bash
# Build from specific directory
docker build -f Dockerfile -t voice-ferry:latest /path/to/voice-ferry

# Build with specific Dockerfile
docker build -f deployments/docker/Dockerfile.prod -t voice-ferry:prod .
```

## Web-UI Docker Build

The Voice Ferry project includes a Node.js-based web interface for configuration and monitoring. The web-UI has its own Dockerfile and build requirements.

### Web-UI Overview

- **Location**: `web-ui/` directory
- **Technology**: Node.js 18+ with Express
- **Default Port**: 3000 (development), 3001 (production)
- **Build Type**: Multi-stage Docker build with development and production targets

### Basic Web-UI Build

```bash
# Navigate to web-ui directory
cd web-ui

# Build with default production target
docker build -t voice-ferry-ui:latest .

# Build specific stage
docker build --target development -t voice-ferry-ui:dev .
docker build --target production -t voice-ferry-ui:prod .
```

### Web-UI Build Stages

#### 1. Development Stage
```bash
# Build development image (includes nodemon, debug tools)
docker build --target development -t voice-ferry-ui:dev .

# Run development container with hot reload
docker run -it \
  -p 3000:3000 \
  -p 9229:9229 \
  -v $(pwd):/app \
  -v /app/node_modules \
  voice-ferry-ui:dev
```

#### 2. Production Stage
```bash
# Build production image (optimized, non-root user)
docker build --target production -t voice-ferry-ui:prod .

# Run production container
docker run -d \
  -p 3001:3001 \
  --name voice-ferry-ui \
  voice-ferry-ui:prod
```

### Web-UI Build Arguments

The web-UI Dockerfile supports Node.js version customization:

```bash
# Build with specific Node.js version
docker build \
  --build-arg NODE_VERSION=20-alpine \
  -t voice-ferry-ui:node20 \
  web-ui/

# Build with custom user ID (for development)
docker build \
  --build-arg USER_ID=$(id -u) \
  --build-arg GROUP_ID=$(id -g) \
  -t voice-ferry-ui:dev \
  web-ui/
```

### Web-UI Environment Variables

```bash
# Run with custom environment
docker run -d \
  -p 3001:3001 \
  -e NODE_ENV=production \
  -e JWT_SECRET=your-secret-key \
  -e REDIS_URL=redis://localhost:6379 \
  -e GRPC_ENDPOINT=localhost:50051 \
  -e LOG_LEVEL=info \
  voice-ferry-ui:latest
```

### Web-UI Development Workflow

#### Local Development Build
```bash
cd web-ui

# Build development image
docker build --target development -t voice-ferry-ui:dev .

# Run with volume mounts for hot reload
docker run -it \
  -p 3000:3000 \
  -p 9229:9229 \
  -v $(pwd):/app \
  -v /app/node_modules \
  -e NODE_ENV=development \
  -e LOG_LEVEL=debug \
  voice-ferry-ui:dev
```

#### Debug Mode
```bash
# Build and run with debug enabled
docker run -it \
  -p 3000:3000 \
  -p 9229:9229 \
  -v $(pwd):/app \
  -v /app/node_modules \
  voice-ferry-ui:dev \
  npm run dev:debug
```

### Web-UI Production Optimization

#### Minimal Production Build
```bash
cd web-ui

# Build with production optimizations
docker build \
  --target production \
  --build-arg NODE_ENV=production \
  -t voice-ferry-ui:prod .

# Verify image size
docker images voice-ferry-ui:prod
```

#### Security Hardening
```bash
# Build with security scanning
docker build --target production -t voice-ferry-ui:secure .

# Run security scan (if available)
docker scout quickview voice-ferry-ui:secure
```

### Web-UI Multi-Architecture Builds

```bash
cd web-ui

# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t voice-ferry-ui:multiarch \
  --push .

# Architecture-specific builds
docker buildx build \
  --platform linux/amd64 \
  -t voice-ferry-ui:amd64 \
  --load .

docker buildx build \
  --platform linux/arm64 \
  -t voice-ferry-ui:arm64 \
  --load .
```

### Web-UI with External Dependencies

#### Build with Redis Connection
```bash
# Run web-ui with Redis dependency
docker network create voice-ferry-net

# Start Redis first
docker run -d \
  --name redis \
  --network voice-ferry-net \
  redis:alpine

# Start web-ui connected to Redis
docker run -d \
  --name voice-ferry-ui \
  --network voice-ferry-net \
  -p 3001:3001 \
  -e REDIS_URL=redis://redis:6379 \
  voice-ferry-ui:latest
```

#### Build with Complete Stack
```bash
# Use docker-compose for full stack
cd web-ui
docker-compose up --build

# Or build specific services
docker-compose build voice-ferry-ui
docker-compose up voice-ferry-ui redis
```

### Web-UI Package Management

#### Update Dependencies
```bash
# Rebuild with updated packages
cd web-ui

# Clean install
docker build --no-cache -t voice-ferry-ui:latest .

# Update specific package
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  node:18-alpine \
  npm update express

# Rebuild after updates
docker build -t voice-ferry-ui:updated .
```

#### Vulnerability Scanning
```bash
# Scan for vulnerabilities
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  node:18-alpine \
  npm audit

# Fix vulnerabilities
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  node:18-alpine \
  npm audit fix
```

### Web-UI Testing

#### Run Tests in Container
```bash
cd web-ui

# Build test image
docker build --target development -t voice-ferry-ui:test .

# Run tests
docker run --rm \
  -v $(pwd):/app \
  -v /app/node_modules \
  voice-ferry-ui:test \
  npm test

# Run tests with coverage
docker run --rm \
  -v $(pwd):/app \
  -v /app/node_modules \
  voice-ferry-ui:test \
  npm run test:coverage
```

#### Integration Testing
```bash
# Test web-ui with backend
docker-compose -f docker-compose.yml -f docker-compose.test.yml up --build

# Test specific endpoints
docker run --rm \
  --network voice-ferry-net \
  curlimages/curl:latest \
  curl -f http://voice-ferry-ui:3001/api/health
```

### Web-UI Build Scripts

#### Using package.json Scripts
```bash
cd web-ui

# Use built-in Docker commands
npm run docker:build
npm run docker:dev
npm run docker:prod

# Custom build with environment
NODE_ENV=production npm run docker:build
```

#### Automated Build Script
```bash
#!/bin/bash
# web-ui-build.sh

cd web-ui

# Build arguments
VERSION=${VERSION:-latest}
ENV=${NODE_ENV:-production}
REGISTRY=${REGISTRY:-}

echo "Building Voice Ferry Web-UI..."
echo "Version: $VERSION"
echo "Environment: $ENV"

# Build command
if [ "$ENV" = "development" ]; then
    docker build --target development -t voice-ferry-ui:$VERSION .
else
    docker build --target production -t voice-ferry-ui:$VERSION .
fi

# Tag for registry if specified
if [ -n "$REGISTRY" ]; then
    docker tag voice-ferry-ui:$VERSION $REGISTRY/voice-ferry-ui:$VERSION
    echo "Tagged for registry: $REGISTRY/voice-ferry-ui:$VERSION"
fi

echo "Build completed successfully!"
```

## Multi-Architecture Builds

### Enable Docker Buildx

```bash
# Create and use buildx builder
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap
```

### Build for Multiple Architectures

```bash
# Build for amd64 and arm64
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t voice-ferry:latest \
  --push .

# Build for specific architecture only
docker buildx build \
  --platform linux/amd64 \
  -t voice-ferry:amd64 \
  --load .
```

### Architecture-Specific Builds

```bash
# AMD64 only
docker buildx build \
  --platform linux/amd64 \
  -t voice-ferry:amd64 \
  --load .

# ARM64 only
docker buildx build \
  --platform linux/arm64 \
  -t voice-ferry:arm64 \
  --load .
```

## Build Arguments and Customization

### Available Build Arguments

The Dockerfile supports several build arguments for customization:

| Argument | Default | Description |
|----------|---------|-------------|
| `VERSION` | `dev` | Application version |
| `BUILD_TIME` | `auto` | Build timestamp |
| `COMMIT_SHA` | `unknown` | Git commit SHA |

### Using Build Arguments

```bash
# Build with version information
docker build \
  --build-arg VERSION=v1.2.3 \
  --build-arg COMMIT_SHA=$(git rev-parse --short HEAD) \
  -t voice-ferry:v1.2.3 .

# Build with all custom arguments
docker build \
  --build-arg VERSION=v1.2.3 \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) \
  -t voice-ferry:v1.2.3 .
```

### Environment-Specific Builds

```bash
# Development build
docker build \
  --build-arg VERSION=dev \
  --target builder \
  -t voice-ferry:dev .

# Production build with optimizations
docker build \
  --build-arg VERSION=prod \
  --build-arg CGO_ENABLED=0 \
  -t voice-ferry:prod .
```

## Local Development Builds

### Development Build with Hot Reload

```bash
# Build development image
docker build \
  --target builder \
  -t voice-ferry:dev .

# Run with volume mounts for development
docker run -it \
  -p 5060:5060/udp \
  -p 8080:8080 \
  -v $(pwd):/app \
  -v $(pwd)/configs:/app/configs \
  voice-ferry:dev
```

### Debug Build

```bash
# Build with debug symbols
docker build \
  --build-arg CGO_ENABLED=1 \
  --build-arg GOFLAGS="-race" \
  -t voice-ferry:debug .
```

## Production Builds

### Optimized Production Build

```bash
# Production build with minimal size
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) \
  -t voice-ferry:$(git describe --tags --always) \
  .
```

### Registry Push

```bash
# Tag for registry
docker tag voice-ferry:latest ghcr.io/2bleere/voice-ferry:latest

# Push to registry
docker push ghcr.io/2bleere/voice-ferry:latest

# Build and push in one command
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/2bleere/voice-ferry:latest \
  --push .
```

## Full Stack Builds

### Building Both Components

The Voice Ferry project consists of two main components: the SIP B2BUA server and the Web-UI. Here's how to build both together.

#### Sequential Build
```bash
# Build B2BUA server first
docker build -t voice-ferry:latest .

# Build Web-UI
cd web-ui
docker build -t voice-ferry-ui:latest .
cd ..

# Verify both images
docker images | grep voice-ferry
```

#### Parallel Build with Docker Compose
```bash
# Build all services using docker-compose
docker-compose -f docker-compose.prod.yml build

# Build specific services
docker-compose -f docker-compose.prod.yml build voice-ferry
docker-compose -f docker-compose.prod.yml build voice-ferry-ui
```

#### Full Stack Development Build
```bash
# Build development versions of both
docker build --target builder -t voice-ferry:dev .
cd web-ui && docker build --target development -t voice-ferry-ui:dev . && cd ..

# Run full development stack
docker-compose -f docker-compose.dev.yml up --build
```

#### Production Stack Build
```bash
# Build production versions
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  -t voice-ferry:prod .

cd web-ui
docker build --target production -t voice-ferry-ui:prod .
cd ..

# Run production stack
docker-compose -f docker-compose.prod.yml up -d
```

### Multi-Architecture Full Stack

#### Build Both Components for Multiple Architectures
```bash
# Create buildx builder if not exists
docker buildx create --name multiarch-builder --use

# Build B2BUA for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/2bleere/voice-ferry:latest \
  --push .

# Build Web-UI for multiple architectures
cd web-ui
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/2bleere/voice-ferry-ui:latest \
  --push .
cd ..
```

#### Automated Multi-Arch Script
```bash
#!/bin/bash
# build-full-stack.sh

set -euo pipefail

REGISTRY="${REGISTRY:-ghcr.io/2bleere}"
VERSION="${VERSION:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

echo "Building Voice Ferry Full Stack"
echo "Registry: $REGISTRY"
echo "Version: $VERSION"
echo "Platforms: $PLATFORMS"

# Build B2BUA
echo "Building B2BUA server..."
docker buildx build \
  --platform $PLATFORMS \
  --build-arg VERSION=$VERSION \
  -t $REGISTRY/voice-ferry:$VERSION \
  --push .

# Build Web-UI
echo "Building Web-UI..."
cd web-ui
docker buildx build \
  --platform $PLATFORMS \
  -t $REGISTRY/voice-ferry-ui:$VERSION \
  --push .
cd ..

echo "Full stack build completed!"
```

### Registry Management for Full Stack

#### Tagging Strategy
```bash
# Consistent versioning for both components
VERSION=$(git describe --tags --always)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Tag B2BUA
docker tag voice-ferry:latest ghcr.io/2bleere/voice-ferry:$VERSION
docker tag voice-ferry:latest ghcr.io/2bleere/voice-ferry:latest

# Tag Web-UI
docker tag voice-ferry-ui:latest ghcr.io/2bleere/voice-ferry-ui:$VERSION
docker tag voice-ferry-ui:latest ghcr.io/2bleere/voice-ferry-ui:latest
```

#### Push Both Components
```bash
# Push B2BUA
docker push ghcr.io/2bleere/voice-ferry:$VERSION
docker push ghcr.io/2bleere/voice-ferry:latest

# Push Web-UI
docker push ghcr.io/2bleere/voice-ferry-ui:$VERSION
docker push ghcr.io/2bleere/voice-ferry-ui:latest
```

### Integration Testing

#### Test Full Stack
```bash
# Start full stack
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be ready
sleep 30

# Test B2BUA health
curl -f http://localhost:8080/health

# Test Web-UI health
curl -f http://localhost:3001/api/health

# Test integration
curl -f http://localhost:3001/api/status

# Cleanup
docker-compose -f docker-compose.prod.yml down
```

#### Automated Integration Test
```bash
#!/bin/bash
# test-full-stack.sh

set -euo pipefail

echo "Starting full stack integration test..."

# Start services
docker-compose -f docker-compose.prod.yml up -d

# Wait for services
echo "Waiting for services to start..."
sleep 60

# Test B2BUA
echo "Testing B2BUA server..."
if curl -f http://localhost:8080/health; then
    echo "✅ B2BUA health check passed"
else
    echo "❌ B2BUA health check failed"
    exit 1
fi

# Test Web-UI
echo "Testing Web-UI..."
if curl -f http://localhost:3001/api/health; then
    echo "✅ Web-UI health check passed"
else
    echo "❌ Web-UI health check failed"
    exit 1
fi

# Test gRPC endpoint
echo "Testing gRPC connection..."
if curl -f http://localhost:3001/api/status; then
    echo "✅ Integration test passed"
else
    echo "❌ Integration test failed"
    exit 1
fi

# Cleanup
docker-compose -f docker-compose.prod.yml down

echo "All tests passed! 🎉"
```

## Advanced Build Scenarios

### Build with Custom Base Image

```bash
# Create custom Dockerfile
cat > Dockerfile.custom << EOF
FROM golang:1.24.3-alpine AS builder
# ... custom build steps
FROM alpine:latest
# ... custom runtime setup
EOF

# Build with custom Dockerfile
docker build -f Dockerfile.custom -t voice-ferry:custom .
```

### Build with Secrets

```bash
# Build with Docker secrets
echo "secret_value" | docker secret create my_secret -

docker build \
  --secret id=my_secret,src=/path/to/secret \
  -t voice-ferry:latest .
```

### Build with Cache Optimization

```bash
# Build with cache mount
docker build \
  --cache-from voice-ferry:latest \
  -t voice-ferry:new .

# Build with BuildKit cache
DOCKER_BUILDKIT=1 docker build \
  --cache-from type=local,src=/tmp/cache \
  --cache-to type=local,dest=/tmp/cache \
  -t voice-ferry:latest .
```

## Using Automated Scripts

### Built-in Build Script

```bash
# Use the provided build script
./scripts/build-docker-images.sh

# With custom registry
REGISTRY=ghcr.io REPOSITORY=myorg/voice-ferry ./scripts/build-docker-images.sh

# With custom tag
TAG=v1.2.3 ./scripts/build-docker-images.sh
```

### Makefile Targets

```bash
# Build using Makefile
make docker-build

# Build with custom tag
make docker-build TAG=v1.2.3

# Build and push
make docker-push
```

## Verification and Testing

### Verify Build

```bash
# Check image details
docker images voice-ferry

# Inspect image
docker inspect voice-ferry:latest

# Check image size
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" voice-ferry
```

### Test the Built Image

```bash
# Quick test run
docker run --rm -p 5060:5060/udp voice-ferry:latest --help

# Full test with configuration
docker run --rm \
  -p 5060:5060/udp \
  -p 8080:8080 \
  -v $(pwd)/configs/development.yaml:/app/configs/config.yaml:ro \
  voice-ferry:latest
```

### Health Check

```bash
# Check if container is healthy
docker run -d --name test-voice-ferry voice-ferry:latest
docker exec test-voice-ferry curl -f http://localhost:8080/health
docker rm -f test-voice-ferry
```

## Troubleshooting

### Common Build Issues

#### Out of Disk Space
```bash
# Clean up Docker resources
docker system prune -a

# Remove old images
docker image prune -a
```

#### Build Cache Issues
```bash
# Force rebuild without cache
docker build --no-cache -t voice-ferry:latest .

# Clear buildx cache
docker buildx prune -a
```

#### Permission Issues
```bash
# Fix file permissions
chmod +x scripts/*.sh

# Build with user mapping
docker build --build-arg USER_ID=$(id -u) --build-arg GROUP_ID=$(id -g) .
```

### Architecture-Specific Issues

#### Cross-compilation Errors
```bash
# Enable experimental features
export DOCKER_CLI_EXPERIMENTAL=enabled

# Use explicit platform
docker build --platform linux/amd64 -t voice-ferry:amd64 .
```

#### QEMU Issues (for cross-platform builds)
```bash
# Install QEMU binfmt
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# Verify QEMU installation
docker buildx ls
```

### Performance Issues

#### Slow Builds
```bash
# Use BuildKit for faster builds
export DOCKER_BUILDKIT=1

# Parallel builds
docker buildx build --max-parallel 4 .

# Use local cache
docker build --cache-from voice-ferry:latest .
```

### Web-UI Specific Issues

#### Node.js Version Conflicts
```bash
# Specify exact Node.js version
docker build --build-arg NODE_VERSION=18.19.0-alpine -t voice-ferry-ui:latest web-ui/

# Check Node.js version in container
docker run --rm voice-ferry-ui:latest node --version
```

#### npm Install Failures
```bash
# Clean npm cache and rebuild
cd web-ui
docker build --no-cache -t voice-ferry-ui:clean .

# Use npm ci for clean installs
docker build --build-arg NPM_COMMAND="npm ci" -t voice-ferry-ui:ci web-ui/
```

#### Port Conflicts
```bash
# Use different ports for development
docker run -p 3001:3000 voice-ferry-ui:dev

# Check what's using the port
lsof -i :3000
```

#### Permission Issues with Node Modules
```bash
# Build with proper user permissions
docker build \
  --build-arg USER_ID=$(id -u) \
  --build-arg GROUP_ID=$(id -g) \
  -t voice-ferry-ui:dev \
  web-ui/

# Fix volume mount permissions
docker run -it \
  -v $(pwd)/web-ui:/app \
  -v /app/node_modules \
  voice-ferry-ui:dev
```

### Multi-Component Issues

#### Service Discovery Problems
```bash
# Use Docker networks for service communication
docker network create voice-ferry-net

# Start services on same network
docker run --network voice-ferry-net --name b2bua voice-ferry:latest
docker run --network voice-ferry-net --name ui voice-ferry-ui:latest
```

#### Version Mismatch Between Components
```bash
# Build both with same version tag
VERSION=$(git describe --tags --always)
docker build -t voice-ferry:$VERSION .
cd web-ui && docker build -t voice-ferry-ui:$VERSION . && cd ..

# Use consistent environment variables
docker-compose -f docker-compose.prod.yml build
```

## Best Practices

### 1. Version Management

```bash
# Always use semantic versioning
VERSION=$(git describe --tags --always --dirty)
docker build -t voice-ferry:$VERSION .

# Tag both specific version and latest
docker build -t voice-ferry:$VERSION -t voice-ferry:latest .
```

### 2. Security

```bash
# Scan for vulnerabilities
docker scout quickview voice-ferry:latest

# Use multi-stage builds to minimize attack surface
# (already implemented in Dockerfile)
```

### 3. Size Optimization

```bash
# Check image layers
docker history voice-ferry:latest

# Use .dockerignore to exclude unnecessary files
echo "*.log" >> .dockerignore
echo "*.tmp" >> .dockerignore
```

### 4. Registry Management

```bash
# Use consistent tagging strategy
docker build -t ${REGISTRY}/${REPOSITORY}:${VERSION} .
docker build -t ${REGISTRY}/${REPOSITORY}:latest .

# Push with retry logic
for i in {1..3}; do
  docker push ${REGISTRY}/${REPOSITORY}:${VERSION} && break
  sleep 5
done
```

### 5. Web-UI Best Practices

#### Development Workflow
```bash
# Use development target for local work
docker build --target development -t voice-ferry-ui:dev web-ui/

# Volume mount for hot reload
docker run -it \
  -p 3000:3000 \
  -v $(pwd)/web-ui:/app \
  -v /app/node_modules \
  voice-ferry-ui:dev
```

#### Production Optimization
```bash
# Use production target with optimizations
docker build --target production -t voice-ferry-ui:prod web-ui/

# Minimize image size
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}" voice-ferry-ui
```

#### Security
```bash
# Run as non-root user (built into Dockerfile)
docker run --rm voice-ferry-ui:latest id

# Scan for vulnerabilities
docker run --rm \
  -v $(pwd)/web-ui:/app \
  node:18-alpine \
  npm audit
```

### 6. Full Stack Coordination

#### Synchronized Builds
```bash
# Build both components with same version
VERSION=$(git describe --tags --always)
docker build -t voice-ferry:$VERSION .
cd web-ui && docker build -t voice-ferry-ui:$VERSION . && cd ..
```

#### Consistent Environment
```bash
# Use docker-compose for coordinated deployment
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d
```

## Configuration Files

### Example .dockerignore

```bash
# Create optimized .dockerignore
cat > .dockerignore << EOF
.git
.github
*.md
logs/
coverage*.out
test/
testing/
documentation/
*.log
.DS_Store
node_modules
EOF
```

### Web-UI .dockerignore

```bash
# Create web-ui specific .dockerignore
cat > web-ui/.dockerignore << EOF
node_modules
npm-debug.log*
yarn-debug.log*
yarn-error.log*
.npm
.eslintcache
.nyc_output
coverage/
*.log
.DS_Store
.env.local
.env.development.local
.env.test.local
.env.production.local
test/
tests/
*.test.js
*.spec.js
EOF
```

### Build Configuration

```bash
# Save build configuration
cat > build.env << EOF
REGISTRY=ghcr.io
REPOSITORY=2bleere/voice-ferry
VERSION=$(git describe --tags --always)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_SHA=$(git rev-parse HEAD)
EOF

# Source and build
source build.env
docker build \
  --build-arg VERSION=$VERSION \
  --build-arg BUILD_TIME=$BUILD_DATE \
  --build-arg COMMIT_SHA=$COMMIT_SHA \
  -t $REGISTRY/$REPOSITORY:$VERSION .
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
# .github/workflows/docker-build.yml
name: Manual Docker Build
on:
  workflow_dispatch:
    inputs:
      tag:
        description: 'Docker tag'
        required: true
        default: 'latest'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: |
          docker build \
            --build-arg VERSION=${{ github.event.inputs.tag }} \
            --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
            --build-arg COMMIT_SHA=${{ github.sha }} \
            -t voice-ferry:${{ github.event.inputs.tag }} .
```

## Conclusion

This guide covers all aspects of manually building Docker images for the Voice Ferry project. For automated builds, consider using the provided scripts in the `scripts/` directory or setting up CI/CD pipelines.

For additional help:
- Check the project's [README.md](../README.md)
- Review [deployment documentation](deployment.md)
- See [configuration guide](configuration.md)
