#!/bin/bash

# Build Docker Images for Voice Ferry SIP B2BUA
# This script builds production-ready Docker images

set -euo pipefail

# Configuration
REGISTRY="${REGISTRY:-ghcr.io}"
REPOSITORY="${REPOSITORY:-2bleere/voice-ferry}"
TAG="${TAG:-latest}"
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD)
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Build main application image
build_main_image() {
    local image_name="${REGISTRY}/${REPOSITORY}:${TAG}"
    local image_name_latest="${REGISTRY}/${REPOSITORY}:latest"
    
    log_info "Building main application image: ${image_name}"
    
    docker build \
        --file Dockerfile \
        --tag "${image_name}" \
        --tag "${image_name_latest}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        --build-arg VERSION="${VERSION}" \
        --platform linux/amd64 \
        .
    
    log_info "Main application image built successfully"
}

# Build web UI image
build_webui_image() {
    local image_name="${REGISTRY}/${REPOSITORY}-webui:${TAG}"
    local image_name_latest="${REGISTRY}/${REPOSITORY}-webui:latest"
    
    log_info "Building web UI image: ${image_name}"
    
    docker build \
        --file web-ui/Dockerfile \
        --tag "${image_name}" \
        --tag "${image_name_latest}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        --build-arg VERSION="${VERSION}" \
        --platform linux/amd64 \
        web-ui/
    
    log_info "Web UI image built successfully"
}

# Test images
test_images() {
    log_info "Testing built images..."
    
    local main_image="${REGISTRY}/${REPOSITORY}:${TAG}"
    local webui_image="${REGISTRY}/${REPOSITORY}-webui:${TAG}"
    
    # Test main application image
    log_info "Testing main application image..."
    if docker run --rm "${main_image}" --version; then
        log_info "Main application image test passed"
    else
        log_error "Main application image test failed"
        exit 1
    fi
    
    # Test web UI image (just check if it starts)
    log_info "Testing web UI image..."
    if docker run --rm --entrypoint="" "${webui_image}" npm --version; then
        log_info "Web UI image test passed"
    else
        log_error "Web UI image test failed"
        exit 1
    fi
}

# Push images to registry
push_images() {
    if [ "${PUSH_IMAGES:-false}" = "true" ]; then
        log_info "Pushing images to registry..."
        
        local main_image="${REGISTRY}/${REPOSITORY}:${TAG}"
        local main_image_latest="${REGISTRY}/${REPOSITORY}:latest"
        local webui_image="${REGISTRY}/${REPOSITORY}-webui:${TAG}"
        local webui_image_latest="${REGISTRY}/${REPOSITORY}-webui:latest"
        
        docker push "${main_image}"
        docker push "${main_image_latest}"
        docker push "${webui_image}"
        docker push "${webui_image_latest}"
        
        log_info "Images pushed successfully"
    else
        log_info "Skipping image push (set PUSH_IMAGES=true to enable)"
    fi
}

# Display image information
show_image_info() {
    log_info "Built images:"
    echo
    docker images --filter "reference=${REGISTRY}/${REPOSITORY}*" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    echo
    
    log_info "Image details:"
    echo "Registry: ${REGISTRY}"
    echo "Repository: ${REPOSITORY}"
    echo "Tag: ${TAG}"
    echo "Version: ${VERSION}"
    echo "Git Commit: ${GIT_COMMIT}"
    echo "Build Date: ${BUILD_DATE}"
}

# Cleanup old images
cleanup_old_images() {
    if [ "${CLEANUP_OLD:-false}" = "true" ]; then
        log_info "Cleaning up old images..."
        
        # Remove dangling images
        docker image prune -f
        
        # Remove old tagged images (keep last 5)
        docker images "${REGISTRY}/${REPOSITORY}" --format "table {{.Tag}}" | tail -n +6 | xargs -r docker rmi "${REGISTRY}/${REPOSITORY}:" || true
        docker images "${REGISTRY}/${REPOSITORY}-webui" --format "table {{.Tag}}" | tail -n +6 | xargs -r docker rmi "${REGISTRY}/${REPOSITORY}-webui:" || true
        
        log_info "Cleanup completed"
    fi
}

# Main execution
main() {
    log_info "Starting Docker image build process..."
    echo "Registry: ${REGISTRY}"
    echo "Repository: ${REPOSITORY}"
    echo "Tag: ${TAG}"
    echo "Version: ${VERSION}"
    echo "Git Commit: ${GIT_COMMIT}"
    echo
    
    check_prerequisites
    build_main_image
    build_webui_image
    test_images
    push_images
    show_image_info
    cleanup_old_images
    
    log_info "Docker image build process completed successfully!"
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo
        echo "Environment variables:"
        echo "  REGISTRY      - Container registry (default: ghcr.io)"
        echo "  REPOSITORY    - Repository name (default: 2bleere/voice-ferry)"
        echo "  TAG           - Image tag (default: latest)"
        echo "  VERSION       - Application version (default: git describe)"
        echo "  PUSH_IMAGES   - Push images to registry (default: false)"
        echo "  CLEANUP_OLD   - Cleanup old images (default: false)"
        echo
        echo "Examples:"
        echo "  $0                           # Build with defaults"
        echo "  TAG=v1.0.0 $0               # Build with specific tag"
        echo "  PUSH_IMAGES=true $0         # Build and push images"
        echo "  CLEANUP_OLD=true $0         # Build and cleanup old images"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac