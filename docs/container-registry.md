# Container Registry Management

This document provides comprehensive guidance for building, tagging, and uploading Voice Ferry containers to registries with multi-platform support for `linux/amd64` and `linux/arm64` architectures.

## 📦 Container Overview

Voice Ferry consists of multiple containerized components that need to be built and deployed:

### Core Containers

1. **Voice Ferry B2BUA Server** - Main SIP Class 4 Switch
2. **Voice Ferry Web UI** - Management interface
3. **Dependencies** - Supporting services (Redis, etcd, Prometheus, Grafana)

## 🏗️ Container Specifications

### 1. Voice Ferry B2BUA Server

**Primary Container**: `2bleere/voice-ferry`

**Dockerfiles**:
- Development: `/Dockerfile` 
- Production: `/deployments/docker/Dockerfile`

**Target Platforms**: 
- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, Apple Silicon, ARM servers)

**Base Images**:
- Builder: `golang:1.24.3-alpine`
- Runtime: `alpine:3.18`

**Key Features**:
- Multi-stage build for minimal image size
- UPX compression for binary optimization
- Non-root user execution
- Health check capabilities
- SSL/TLS certificate support

### 2. Voice Ferry Web UI

**Container**: `2bleere/voice-ferry-ui`

**Dockerfile**: `/web-ui/Dockerfile`

**Target Platforms**: 
- `linux/amd64`
- `linux/arm64`

**Base Images**:
- Builder: `node:18-alpine`
- Runtime: `node:18-alpine`

**Key Features**:
- Multi-stage build (development/production)
- Non-root user execution
- Volume mounts for configuration
- Environment-based configuration

## 🚀 Building Multi-Platform Images

### Prerequisites

1. **Docker Buildx** (for multi-platform builds)
2. **Container Registry Access** (Docker Hub, ECR, GCR, etc.)
3. **Platform Emulation** (QEMU for cross-platform builds)

### Setup Docker Buildx

```bash
# Enable Docker BuildKit
export DOCKER_BUILDKIT=1

# Create and use a new builder instance
docker buildx create --name voice-ferry-builder --use
docker buildx inspect --bootstrap

# Verify platform support
docker buildx ls
```

### Building Images

#### 1. Voice Ferry B2BUA Server

##### Development Build
```bash
# Navigate to project root
cd /path/to/voice-ferry

# Build multi-platform development image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  -t 2bleere/voice-ferry:dev \
  -f Dockerfile \
  --push .
```

##### Production Build
```bash
# Build optimized production image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  -t 2bleere/voice-ferry:latest \
  -t 2bleere/voice-ferry:$(git describe --tags --always) \
  -f deployments/docker/Dockerfile \
  --push .
```

#### 2. Voice Ferry Web UI

```bash
# Navigate to web-ui directory
cd web-ui

# Build multi-platform web UI image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --target production \
  -t 2bleere/voice-ferry-ui:latest \
  -t 2bleere/voice-ferry-ui:$(git describe --tags --always) \
  --push .
```

## 🏷️ Tagging Strategy

### Semantic Versioning

Voice Ferry follows semantic versioning (`MAJOR.MINOR.PATCH`):

```bash
# Example version tags
2bleere/voice-ferry:v1.0.0        # Specific version
2bleere/voice-ferry:v1.0          # Minor version
2bleere/voice-ferry:v1             # Major version
2bleere/voice-ferry:latest         # Latest stable
2bleere/voice-ferry:dev            # Development builds
2bleere/voice-ferry:edge           # Bleeding edge
```

### Branch-based Tags

```bash
# Main branch
2bleere/voice-ferry:main-$(git rev-parse --short HEAD)

# Feature branches
2bleere/voice-ferry:feature-auth-$(git rev-parse --short HEAD)

# Release candidates
2bleere/voice-ferry:v1.0.0-rc.1
```

## 📋 Registry Upload Checklist

### Required Images for Production

#### Core Application Images
- [ ] `2bleere/voice-ferry:latest` (B2BUA Server)
- [ ] `2bleere/voice-ferry:v1.x.x` (Versioned B2BUA)
- [ ] `2bleere/voice-ferry-ui:latest` (Web UI)
- [ ] `2bleere/voice-ferry-ui:v1.x.x` (Versioned Web UI)

#### Platform Verification
- [ ] `linux/amd64` manifest present
- [ ] `linux/arm64` manifest present
- [ ] Multi-platform manifest list created

#### Dependencies (Optional - if using custom builds)
- [ ] `2bleere/voice-ferry-redis:latest` (Custom Redis config)
- [ ] `2bleere/voice-ferry-etcd:latest` (Custom etcd config)
- [ ] `2bleere/voice-ferry-prometheus:latest` (Custom Prometheus)
- [ ] `2bleere/voice-ferry-grafana:latest` (Custom Grafana)

## 🔧 Registry Configuration

### Docker Hub

```bash
# Login to Docker Hub
docker login

# Build and push
docker buildx build --platform linux/amd64,linux/arm64 -t 2bleere/voice-ferry:latest --push .
```

### Amazon ECR

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Create repositories
aws ecr create-repository --repository-name voice-ferry
aws ecr create-repository --repository-name voice-ferry-ui

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t 123456789012.dkr.ecr.us-east-1.amazonaws.com/voice-ferry:latest \
  --push .
```

### Google Container Registry

```bash
# Configure gcloud
gcloud auth configure-docker

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t gcr.io/project-id/voice-ferry:latest \
  --push .
```

### GitHub Container Registry

```bash
# Login with GitHub token
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/2bleere/voice-ferry:latest \
  --push .
```

## 🤖 Automated CI/CD Pipeline

### GitHub Actions Example

Create `.github/workflows/build-containers.yml`:

```yaml
name: Build and Push Containers

on:
  push:
    branches: [ main ]
    tags: [ 'v*' ]
  pull_request:
    branches: [ main ]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME_B2BUA: ${{ github.repository }}
  IMAGE_NAME_UI: ${{ github.repository }}-ui

jobs:
  build-b2bua:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Log in to Container Registry
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME_B2BUA }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=semver,pattern={{version}}
          type=semver,pattern={{major}}.{{minor}}
          type=semver,pattern={{major}}

    - name: Build and push B2BUA image
      uses: docker/build-push-action@v5
      with:
        context: .
        file: ./deployments/docker/Dockerfile
        platforms: linux/amd64,linux/arm64
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        build-args: |
          VERSION=${{ steps.meta.outputs.version }}
          BUILD_TIME=${{ steps.meta.outputs.date }}
          GIT_COMMIT=${{ github.sha }}

  build-ui:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Log in to Container Registry
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME_UI }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=semver,pattern={{version}}
          type=semver,pattern={{major}}.{{minor}}

    - name: Build and push UI image
      uses: docker/build-push-action@v5
      with:
        context: ./web-ui
        platforms: linux/amd64,linux/arm64
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        target: production
```

## 📊 Image Size Optimization

### Voice Ferry B2BUA
- **Base Size**: ~15MB (Alpine-based)
- **With UPX**: ~8-10MB
- **Multi-stage**: Removes build dependencies

### Voice Ferry Web UI  
- **Base Size**: ~80MB (Node.js Alpine)
- **Production**: ~60MB (production build only)

## 🔍 Registry Verification

### Verify Multi-Platform Support

```bash
# Check manifest list
docker buildx imagetools inspect 2bleere/voice-ferry:latest

# Expected output shows multiple platforms:
# MediaType: application/vnd.docker.distribution.manifest.list.v2+json
# Platforms:
#   - linux/amd64
#   - linux/arm64
```

### Test Platform-Specific Pulls

```bash
# Pull specific platform
docker pull --platform=linux/amd64 2bleere/voice-ferry:latest
docker pull --platform=linux/arm64 2bleere/voice-ferry:latest

# Verify architecture
docker run --rm 2bleere/voice-ferry:latest uname -m
```

## 📋 Deployment Integration

### Update Kubernetes Manifests

```yaml
# deployments/kubernetes/voice-ferry-production.yaml
spec:
  template:
    spec:
      containers:
      - name: voice-ferry
        image: 2bleere/voice-ferry:v1.0.0  # Use specific version
        imagePullPolicy: Always
```

### Update Docker Compose

```yaml
# docker-compose.prod.yml
services:
  voice-ferry:
    image: 2bleere/voice-ferry:latest
    platform: linux/amd64  # Or linux/arm64
```

## 🛡️ Security Considerations

### Image Scanning

```bash
# Scan for vulnerabilities
docker scout cves 2bleere/voice-ferry:latest
trivy image 2bleere/voice-ferry:latest
```

### Signing Images

```bash
# Sign with Docker Content Trust
export DOCKER_CONTENT_TRUST=1
docker trust key generate voice-ferry
docker trust signer add --key voice-ferry.pub voice-ferry 2bleere/voice-ferry
```

## 🚀 Quick Start Script

Create `scripts/build-and-push.sh`:

```bash
#!/bin/bash
set -e

VERSION=${1:-"dev"}
REGISTRY=${2:-"2bleere"}

echo "Building Voice Ferry containers for version: $VERSION"

# Build B2BUA
echo "Building B2BUA server..."
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=$VERSION \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  -t $REGISTRY/voice-ferry:$VERSION \
  -f deployments/docker/Dockerfile \
  --push .

# Build Web UI
echo "Building Web UI..."
cd web-ui
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --target production \
  -t $REGISTRY/voice-ferry-ui:$VERSION \
  --push .

echo "✅ All containers built and pushed successfully!"
echo "Images available:"
echo "  - $REGISTRY/voice-ferry:$VERSION"
echo "  - $REGISTRY/voice-ferry-ui:$VERSION"
```

Usage:
```bash
# Build development version
./scripts/build-and-push.sh dev

# Build production version
./scripts/build-and-push.sh v1.0.0

# Build with custom registry
./scripts/build-and-push.sh v1.0.0 myregistry.com/voice-ferry
```

## 📚 Additional Resources

- [Docker Buildx Documentation](https://docs.docker.com/buildx/)
- [Multi-platform Images](https://docs.docker.com/desktop/multi-arch/)
- [Container Registry Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Kubernetes Image Pull Policies](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)

---

**Note**: Always test images on target platforms before production deployment. Consider using staging environments that match your production architecture.
