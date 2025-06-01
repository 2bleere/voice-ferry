#!/bin/bash
# Voice Ferry Production Deployment Script
# Deploys the complete Voice Ferry system with etcd monitoring support

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOYMENT_ENV="${DEPLOYMENT_ENV:-production}"
NAMESPACE="${NAMESPACE:-voice-ferry}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-2bleere}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker is not running"
        exit 1
    fi
    
    # Check if Docker Compose is available
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check if required environment files exist
    if [[ ! -f "${PROJECT_ROOT}/configs/.env.production" ]]; then
        log_error "Production environment file not found at configs/.env.production"
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

# Function to build Docker images
build_images() {
    log_info "Building Docker images..."
    
    cd "${PROJECT_ROOT}"
    
    # Build B2BUA image
    log_info "Building B2BUA image..."
    docker build -t "${DOCKER_REGISTRY}/voice-ferry:${IMAGE_TAG}" .
    
    # Build Web UI image
    log_info "Building Web UI image..."
    cd "${PROJECT_ROOT}/web-ui"
    docker build -t "${DOCKER_REGISTRY}/voice-ferry-ui:${IMAGE_TAG}" .
    
    cd "${PROJECT_ROOT}"
    log_success "Docker images built successfully"
}

# Function to create Docker network
create_network() {
    log_info "Creating Docker network..."
    
    if ! docker network ls | grep -q voice-ferry-network; then
        docker network create voice-ferry-network \
            --driver bridge \
            --subnet=172.20.0.0/16 \
            --ip-range=172.20.240.0/20
        log_success "Docker network created"
    else
        log_info "Docker network already exists"
    fi
}

# Function to deploy infrastructure services
deploy_infrastructure() {
    log_info "Deploying infrastructure services (Redis, etcd, RTPEngine)..."
    
    # Create data directories
    mkdir -p "${PROJECT_ROOT}/data/redis"
    mkdir -p "${PROJECT_ROOT}/data/etcd"
    mkdir -p "${PROJECT_ROOT}/logs"
    
    # Deploy infrastructure services first
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" up -d redis etcd rtpengine
    
    # Wait for services to be ready
    log_info "Waiting for infrastructure services to be ready..."
    
    # Wait for Redis
    local redis_ready=false
    for i in {1..30}; do
        if docker exec voice-ferry-redis redis-cli ping &> /dev/null; then
            redis_ready=true
            break
        fi
        sleep 2
    done
    
    if [[ "$redis_ready" != "true" ]]; then
        log_error "Redis failed to start within timeout"
        exit 1
    fi
    
    # Wait for etcd
    local etcd_ready=false
    for i in {1..30}; do
        if docker exec voice-ferry-etcd etcdctl endpoint health --endpoints=http://localhost:2379 &> /dev/null; then
            etcd_ready=true
            break
        fi
        sleep 2
    done
    
    if [[ "$etcd_ready" != "true" ]]; then
        log_error "etcd failed to start within timeout"
        exit 1
    fi
    
    log_success "Infrastructure services deployed and ready"
}

# Function to deploy application services
deploy_application() {
    log_info "Deploying application services (B2BUA, Web UI)..."
    
    # Deploy B2BUA
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" up -d voice-ferry
    
    # Wait for B2BUA to be ready
    log_info "Waiting for B2BUA to be ready..."
    local b2bua_ready=false
    for i in {1..60}; do
        if docker exec voice-ferry-b2bua wget --no-verbose --tries=1 --spider http://localhost:8080/healthz/live &> /dev/null; then
            b2bua_ready=true
            break
        fi
        sleep 3
    done
    
    if [[ "$b2bua_ready" != "true" ]]; then
        log_error "B2BUA failed to start within timeout"
        exit 1
    fi
    
    # Deploy Web UI
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" up -d web-ui
    
    # Wait for Web UI to be ready
    log_info "Waiting for Web UI to be ready..."
    local webui_ready=false
    for i in {1..60}; do
        if docker exec voice-ferry-ui curl -f http://localhost:3001/api/health &> /dev/null; then
            webui_ready=true
            break
        fi
        sleep 3
    done
    
    if [[ "$webui_ready" != "true" ]]; then
        log_error "Web UI failed to start within timeout"
        exit 1
    fi
    
    log_success "Application services deployed and ready"
}

# Function to deploy monitoring services
deploy_monitoring() {
    log_info "Deploying monitoring services (Prometheus, Grafana)..."
    
    # Create monitoring directories
    mkdir -p "${PROJECT_ROOT}/data/prometheus"
    mkdir -p "${PROJECT_ROOT}/data/grafana"
    
    # Deploy monitoring services
    docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" up -d prometheus grafana
    
    log_success "Monitoring services deployed"
}

# Function to verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    # Check all services are running
    local services=("voice-ferry-redis" "voice-ferry-etcd" "voice-ferry-rtpengine" "voice-ferry-b2bua" "voice-ferry-ui")
    
    for service in "${services[@]}"; do
        if docker ps --format "table {{.Names}}" | grep -q "^${service}$"; then
            local status=$(docker inspect --format="{{.State.Health.Status}}" "${service}" 2>/dev/null || echo "unknown")
            if [[ "$status" == "healthy" ]] || [[ "$status" == "unknown" ]]; then
                log_success "Service ${service} is running"
            else
                log_warning "Service ${service} is running but health check failed (status: ${status})"
            fi
        else
            log_error "Service ${service} is not running"
            exit 1
        fi
    done
    
    # Test etcd connectivity
    log_info "Testing etcd connectivity..."
    if docker exec voice-ferry-etcd etcdctl endpoint status --endpoints=http://localhost:2379 &> /dev/null; then
        log_success "etcd is accessible and responding"
    else
        log_error "etcd connectivity test failed"
        exit 1
    fi
    
    # Test Redis connectivity
    log_info "Testing Redis connectivity..."
    if docker exec voice-ferry-redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis is accessible and responding"
    else
        log_error "Redis connectivity test failed"
        exit 1
    fi
    
    # Test Web UI accessibility
    log_info "Testing Web UI accessibility..."
    if docker exec voice-ferry-ui curl -f http://localhost:3001/api/health &> /dev/null; then
        log_success "Web UI is accessible and responding"
    else
        log_error "Web UI accessibility test failed"
        exit 1
    fi
    
    # Test B2BUA accessibility
    log_info "Testing B2BUA accessibility..."
    if docker exec voice-ferry-b2bua wget --no-verbose --tries=1 --spider http://localhost:8080/healthz/live &> /dev/null; then
        log_success "B2BUA is accessible and responding"
    else
        log_error "B2BUA accessibility test failed"
        exit 1
    fi
    
    log_success "Deployment verification completed successfully"
}

# Function to show deployment information
show_deployment_info() {
    log_info "Deployment completed successfully!"
    echo ""
    echo "==================================="
    echo "  Voice Ferry Production Deployment"
    echo "==================================="
    echo ""
    echo "Services:"
    echo "  • Web UI:      http://localhost:3001"
    echo "  • B2BUA API:   http://localhost:8080"
    echo "  • SIP Port:    5060 (UDP/TCP)"
    echo "  • gRPC API:    localhost:50051"
    echo "  • Prometheus:  http://localhost:9090"
    echo "  • Grafana:     http://localhost:3000"
    echo ""
    echo "Infrastructure:"
    echo "  • Redis:       localhost:6379"
    echo "  • etcd:        localhost:2379"
    echo "  • RTPEngine:   localhost:22222"
    echo ""
    echo "Status Monitoring:"
    echo "  • All services include health checks"
    echo "  • etcd status is monitored via Web UI"
    echo "  • Real-time updates via WebSocket"
    echo ""
    echo "Management Commands:"
    echo "  • View logs:   docker-compose -f docker-compose.prod.yml logs -f"
    echo "  • Stop all:    docker-compose -f docker-compose.prod.yml down"
    echo "  • Restart:     docker-compose -f docker-compose.prod.yml restart"
    echo ""
    echo "Next Steps:"
    echo "  1. Access the Web UI at http://localhost:3001"
    echo "  2. Verify etcd status in the dashboard"
    echo "  3. Configure SIP routing rules"
    echo "  4. Test call flow"
    echo ""
}

# Main deployment function
main() {
    log_info "Starting Voice Ferry production deployment..."
    
    check_prerequisites
    build_images
    create_network
    deploy_infrastructure
    deploy_application
    deploy_monitoring
    verify_deployment
    show_deployment_info
    
    log_success "Voice Ferry production deployment completed successfully!"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "stop")
        log_info "Stopping Voice Ferry services..."
        docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" down
        log_success "Services stopped"
        ;;
    "restart")
        log_info "Restarting Voice Ferry services..."
        docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" restart
        log_success "Services restarted"
        ;;
    "status")
        docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" ps
        ;;
    "logs")
        docker-compose -f "${PROJECT_ROOT}/docker-compose.prod.yml" logs -f "${2:-}"
        ;;
    *)
        echo "Usage: $0 {deploy|stop|restart|status|logs [service]}"
        exit 1
        ;;
esac
