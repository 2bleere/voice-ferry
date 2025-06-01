#!/bin/bash
# Voice Ferry Container Build and Push Script
# Builds multi-platform containers for Voice Ferry components

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERSION=${1:-"dev"}
REGISTRY=${2:-"2bleere"}
PLATFORMS=${3:-"linux/amd64,linux/arm64"}
PUSH=${4:-"true"}

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_usage() {
    echo "Usage: $0 [VERSION] [REGISTRY] [PLATFORMS] [PUSH]"
    echo ""
    echo "Arguments:"
    echo "  VERSION    Image version tag (default: dev)"
    echo "  REGISTRY   Container registry prefix (default: 2bleere)"
    echo "  PLATFORMS  Target platforms (default: linux/amd64,linux/arm64)"
    echo "  PUSH       Push to registry (default: true)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Build dev version"
    echo "  $0 v1.0.0                           # Build specific version"
    echo "  $0 v1.0.0 myregistry.com/voice-ferry # Custom registry"
    echo "  $0 dev 2bleere linux/amd64 false    # Build only amd64, don't push"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check Docker Buildx
    if ! docker buildx version &> /dev/null; then
        log_error "Docker Buildx is not available"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -d "web-ui" ]]; then
        log_error "This script must be run from the Voice Ferry project root directory"
        exit 1
    fi
    
    # Check Git (for build metadata)
    if ! command -v git &> /dev/null; then
        log_warning "Git is not available - build metadata will be limited"
    fi
    
    log_success "Prerequisites check passed"
}

setup_buildx() {
    log_info "Setting up Docker Buildx..."
    
    # Create builder if it doesn't exist
    if ! docker buildx ls | grep -q "voice-ferry-builder"; then
        log_info "Creating voice-ferry-builder instance..."
        docker buildx create --name voice-ferry-builder --use
    else
        log_info "Using existing voice-ferry-builder instance..."
        docker buildx use voice-ferry-builder
    fi
    
    # Bootstrap the builder
    docker buildx inspect --bootstrap
    
    log_success "Buildx setup complete"
}

get_build_metadata() {
    log_info "Collecting build metadata..."
    
    if command -v git &> /dev/null; then
        BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
        GIT_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
        
        # If VERSION is "dev", append commit hash
        if [[ "$VERSION" == "dev" ]]; then
            SHORT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
            VERSION="dev-${SHORT_COMMIT}"
        fi
    else
        BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        GIT_COMMIT="unknown"
        GIT_BRANCH="unknown"
    fi
    
    log_info "Version: $VERSION"
    log_info "Build time: $BUILD_TIME"
    log_info "Git commit: $GIT_COMMIT"
    log_info "Git branch: $GIT_BRANCH"
}

build_b2bua() {
    log_info "Building Voice Ferry B2BUA server..."
    
    local dockerfile="deployments/docker/Dockerfile"
    local image_name="$REGISTRY/voice-ferry"
    
    # Build arguments
    local build_args=(
        --build-arg "VERSION=$VERSION"
        --build-arg "BUILD_TIME=$BUILD_TIME"
        --build-arg "GIT_COMMIT=$GIT_COMMIT"
        --build-arg "GIT_BRANCH=$GIT_BRANCH"
    )
    
    # Build command
    local build_cmd=(
        docker buildx build
        --platform "$PLATFORMS"
        "${build_args[@]}"
        -t "$image_name:$VERSION"
        -f "$dockerfile"
    )
    
    # Add latest tag for non-dev versions
    if [[ "$VERSION" != dev* ]]; then
        build_cmd+=(-t "$image_name:latest")
    fi
    
    # Add push flag if enabled
    if [[ "$PUSH" == "true" ]]; then
        build_cmd+=(--push)
    else
        build_cmd+=(--load)
    fi
    
    # Add context
    build_cmd+=(.)
    
    log_info "Executing: ${build_cmd[*]}"
    "${build_cmd[@]}"
    
    log_success "B2BUA server build complete"
}

build_web_ui() {
    log_info "Building Voice Ferry Web UI..."
    
    local dockerfile="web-ui/Dockerfile"
    local image_name="$REGISTRY/voice-ferry-ui"
    
    # Change to web-ui directory
    pushd web-ui > /dev/null
    
    # Build command
    local build_cmd=(
        docker buildx build
        --platform "$PLATFORMS"
        --target production
        -t "$image_name:$VERSION"
    )
    
    # Add latest tag for non-dev versions
    if [[ "$VERSION" != dev* ]]; then
        build_cmd+=(-t "$image_name:latest")
    fi
    
    # Add push flag if enabled
    if [[ "$PUSH" == "true" ]]; then
        build_cmd+=(--push)
    else
        build_cmd+=(--load)
    fi
    
    # Add context
    build_cmd+=(.)
    
    log_info "Executing: ${build_cmd[*]}"
    "${build_cmd[@]}"
    
    # Return to original directory
    popd > /dev/null
    
    log_success "Web UI build complete"
}

verify_images() {
    if [[ "$PUSH" == "true" ]]; then
        log_info "Verifying pushed images..."
        
        # Verify B2BUA
        log_info "Checking B2BUA manifest:"
        docker buildx imagetools inspect "$REGISTRY/voice-ferry:$VERSION" || log_warning "Failed to inspect B2BUA image"
        
        # Verify Web UI
        log_info "Checking Web UI manifest:"
        docker buildx imagetools inspect "$REGISTRY/voice-ferry-ui:$VERSION" || log_warning "Failed to inspect Web UI image"
    else
        log_info "Images built locally (not pushed to registry)"
    fi
}

cleanup() {
    log_info "Cleaning up build cache..."
    docker buildx prune -f || true
}

main() {
    echo "=================================="
    echo "Voice Ferry Container Build Script"
    echo "=================================="
    echo ""
    
    # Show usage if help requested
    if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
        show_usage
        exit 0
    fi
    
    log_info "Starting build process..."
    log_info "Version: $VERSION"
    log_info "Registry: $REGISTRY"
    log_info "Platforms: $PLATFORMS"
    log_info "Push to registry: $PUSH"
    echo ""
    
    # Execute build steps
    check_prerequisites
    setup_buildx
    get_build_metadata
    echo ""
    
    build_b2bua
    echo ""
    
    build_web_ui
    echo ""
    
    verify_images
    cleanup
    
    echo ""
    log_success "🎉 All containers built successfully!"
    echo ""
    echo "Images created:"
    echo "  📦 $REGISTRY/voice-ferry:$VERSION"
    echo "  🌐 $REGISTRY/voice-ferry-ui:$VERSION"
    
    if [[ "$VERSION" != dev* ]]; then
        echo "  📦 $REGISTRY/voice-ferry:latest"
        echo "  🌐 $REGISTRY/voice-ferry-ui:latest"
    fi
    
    if [[ "$PUSH" == "true" ]]; then
        echo ""
        echo "✅ Images pushed to registry: $REGISTRY"
        echo ""
        echo "To deploy in Kubernetes:"
        echo "  kubectl set image deployment/voice-ferry voice-ferry=$REGISTRY/voice-ferry:$VERSION"
        echo "  kubectl set image deployment/voice-ferry-ui voice-ferry-ui=$REGISTRY/voice-ferry-ui:$VERSION"
        echo ""
        echo "To deploy with Docker Compose:"
        echo "  Edit docker-compose.prod.yml to use the new image tags"
    else
        echo ""
        echo "ℹ️  Images built locally. To push to registry:"
        echo "  $0 $VERSION $REGISTRY $PLATFORMS true"
    fi
}

# Handle script interruption
trap 'log_error "Build interrupted"; exit 1' INT TERM

# Run main function
main "$@"
