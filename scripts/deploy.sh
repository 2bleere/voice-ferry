#!/bin/bash
# Voice Ferry Production Deployment Script
# This script helps deploy Voice Ferry in production environments

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEPLOYMENT_TYPE="docker"  # docker, kubernetes, binary
ENVIRONMENT="production"
VERSION="v1.0.0"
NAMESPACE="voice-ferry"

# Functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
    exit 1
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    case $DEPLOYMENT_TYPE in
        "docker")
            command -v docker >/dev/null 2>&1 || error "Docker is not installed"
            command -v docker-compose >/dev/null 2>&1 || error "Docker Compose is not installed"
            ;;
        "kubernetes")
            command -v kubectl >/dev/null 2>&1 || error "kubectl is not installed"
            kubectl cluster-info >/dev/null 2>&1 || error "kubectl cannot connect to cluster"
            ;;
        "binary")
            command -v systemctl >/dev/null 2>&1 || error "systemctl is not available"
            ;;
    esac
    
    success "Prerequisites check passed"
}

# Generate secure JWT key
generate_jwt_key() {
    if [[ ! -f ".env" ]] || ! grep -q "JWT_SIGNING_KEY=" .env; then
        log "Generating secure JWT signing key..."
        JWT_KEY=$(openssl rand -hex 32)
        echo "JWT_SIGNING_KEY=$JWT_KEY" >> .env
        success "JWT key generated and saved to .env"
    else
        log "JWT key already exists in .env"
    fi
}

# Setup SSL certificates
setup_ssl() {
    log "Setting up SSL certificates..."
    
    if [[ ! -d "ssl" ]]; then
        mkdir -p ssl
    fi
    
    if [[ ! -f "ssl/voice-ferry.crt" ]] || [[ ! -f "ssl/voice-ferry.key" ]]; then
        warn "SSL certificates not found. Generating self-signed certificates..."
        warn "For production, replace these with proper SSL certificates!"
        
        openssl req -x509 -newkey rsa:4096 \
            -keyout ssl/voice-ferry.key \
            -out ssl/voice-ferry.crt \
            -days 365 -nodes \
            -subj "/CN=voice-ferry.local/O=Voice Ferry/C=US" \
            -addext "subjectAltName=DNS:voice-ferry.local,DNS:localhost,IP:127.0.0.1"
        
        chmod 600 ssl/voice-ferry.key
        chmod 644 ssl/voice-ferry.crt
        
        success "Self-signed certificates generated"
    else
        log "SSL certificates already exist"
    fi
}

# Deploy with Docker
deploy_docker() {
    log "Deploying Voice Ferry with Docker Compose..."
    
    if [[ ! -f ".env" ]]; then
        log "Creating .env file from template..."
        cp .env.example .env
        generate_jwt_key
        warn "Please review and customize the .env file before continuing"
        read -p "Press Enter to continue after reviewing .env file..."
    fi
    
    setup_ssl
    
    # Build production images if needed
    if [[ "${BUILD_IMAGES:-false}" == "true" ]]; then
        log "Building production Docker images..."
        docker build -f deployments/docker/Dockerfile \
            --build-arg VERSION=$VERSION \
            --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
            --build-arg GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown") \
            -t voice-ferry:$VERSION .
    fi
    
    # Deploy the stack
    log "Starting Voice Ferry production stack..."
    docker-compose -f docker-compose.prod.yml up -d
    
    # Wait for services to be ready
    log "Waiting for services to start..."
    sleep 30
    
    # Health check
    if curl -f http://localhost:8080/healthz/live >/dev/null 2>&1; then
        success "Voice Ferry is running and healthy!"
        log "Access points:"
        log "  - SIP: udp://localhost:5060"
        log "  - SIP TLS: tcp://localhost:5061"
        log "  - gRPC API: localhost:50051"
        log "  - Web UI: http://localhost:8080"
        log "  - Metrics: http://localhost:8080/metrics"
        if docker-compose -f docker-compose.prod.yml ps | grep -q grafana; then
            log "  - Grafana: http://localhost:3000 (admin/admin)"
        fi
    else
        error "Health check failed. Check logs with: docker-compose -f docker-compose.prod.yml logs"
    fi
}

# Deploy with Kubernetes
deploy_kubernetes() {
    log "Deploying Voice Ferry to Kubernetes..."
    
    # Create namespace
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    # Generate and apply secrets
    log "Creating Kubernetes secrets..."
    
    if [[ ! -f ".env" ]]; then
        cp .env.example .env
        generate_jwt_key
    fi
    
    source .env
    
    kubectl create secret generic voice-ferry-secrets \
        --from-literal=jwt-signing-key="${JWT_SIGNING_KEY}" \
        --namespace=$NAMESPACE \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Setup SSL if available
    if [[ -f "ssl/voice-ferry.crt" ]] && [[ -f "ssl/voice-ferry.key" ]]; then
        kubectl create secret tls voice-ferry-tls \
            --cert=ssl/voice-ferry.crt \
            --key=ssl/voice-ferry.key \
            --namespace=$NAMESPACE \
            --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    # Deploy dependencies first
    log "Deploying dependencies (Redis, etcd, RTPEngine)..."
    kubectl apply -f deployments/kubernetes/dependencies.yaml
    
    # Wait for dependencies
    log "Waiting for dependencies to be ready..."
    kubectl wait --for=condition=available --timeout=300s \
        deployment/redis deployment/etcd -n $NAMESPACE
    
    # Deploy Voice Ferry
    log "Deploying Voice Ferry B2BUA..."
    kubectl apply -f deployments/kubernetes/voice-ferry-production.yaml
    
    # Wait for deployment
    log "Waiting for Voice Ferry to be ready..."
    kubectl wait --for=condition=available --timeout=300s \
        deployment/voice-ferry -n $NAMESPACE
    
    # Get service information
    success "Voice Ferry deployed successfully!"
    log "Service information:"
    kubectl get services -n $NAMESPACE
    
    log "Pod status:"
    kubectl get pods -n $NAMESPACE
    
    # Port forward for testing
    log "To access Voice Ferry locally, use:"
    log "  kubectl port-forward svc/voice-ferry-metrics 8080:8080 -n $NAMESPACE"
    log "  kubectl port-forward svc/voice-ferry-grpc 50051:50051 -n $NAMESPACE"
}

# Deploy binary
deploy_binary() {
    log "Deploying Voice Ferry as systemd service..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        error "Binary deployment requires root privileges"
    fi
    
    # Create user
    useradd -r -s /bin/false voice-ferry || true
    
    # Create directories
    mkdir -p /etc/voice-ferry /var/lib/voice-ferry /var/log/voice-ferry
    chown voice-ferry:voice-ferry /var/lib/voice-ferry /var/log/voice-ferry
    
    # Copy binary
    if [[ ! -f "build/b2bua-server" ]]; then
        error "Binary not found. Run 'make build' first."
    fi
    
    cp build/b2bua-server /usr/local/bin/voice-ferry
    chmod +x /usr/local/bin/voice-ferry
    
    # Copy configuration
    cp configs/production.yaml /etc/voice-ferry/config.yaml
    chown root:voice-ferry /etc/voice-ferry/config.yaml
    chmod 640 /etc/voice-ferry/config.yaml
    
    # Create systemd service
    cat > /etc/systemd/system/voice-ferry.service << EOF
[Unit]
Description=Voice Ferry SIP B2BUA
After=network.target
Wants=network.target

[Service]
Type=simple
User=voice-ferry
Group=voice-ferry
ExecStart=/usr/local/bin/voice-ferry -config /etc/voice-ferry/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
LimitNOFILE=65536
WorkingDirectory=/var/lib/voice-ferry

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/voice-ferry /var/log/voice-ferry
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true

[Install]
WantedBy=multi-user.target
EOF
    
    # Start service
    systemctl daemon-reload
    systemctl enable voice-ferry
    systemctl start voice-ferry
    
    success "Voice Ferry service installed and started"
    log "Service status:"
    systemctl status voice-ferry
}

# Cleanup function
cleanup() {
    case $DEPLOYMENT_TYPE in
        "docker")
            log "Stopping Docker services..."
            docker-compose -f docker-compose.prod.yml down
            ;;
        "kubernetes")
            log "Deleting Kubernetes resources..."
            kubectl delete namespace $NAMESPACE
            ;;
        "binary")
            log "Stopping systemd service..."
            systemctl stop voice-ferry
            systemctl disable voice-ferry
            rm -f /etc/systemd/system/voice-ferry.service
            systemctl daemon-reload
            ;;
    esac
}

# Main script
main() {
    log "Voice Ferry Production Deployment Script"
    log "======================================="
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                DEPLOYMENT_TYPE="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --cleanup)
                cleanup
                exit 0
                ;;
            --build)
                BUILD_IMAGES="true"
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  -t, --type TYPE      Deployment type (docker, kubernetes, binary)"
                echo "  -v, --version VER    Version to deploy (default: v1.0.0)"
                echo "  -n, --namespace NS   Kubernetes namespace (default: voice-ferry)"
                echo "      --build          Build Docker images locally"
                echo "      --cleanup        Cleanup deployment"
                echo "  -h, --help           Show this help"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    log "Deployment type: $DEPLOYMENT_TYPE"
    log "Version: $VERSION"
    log "Environment: $ENVIRONMENT"
    
    check_prerequisites
    
    case $DEPLOYMENT_TYPE in
        "docker")
            deploy_docker
            ;;
        "kubernetes")
            deploy_kubernetes
            ;;
        "binary")
            deploy_binary
            ;;
        *)
            error "Invalid deployment type: $DEPLOYMENT_TYPE"
            ;;
    esac
    
    success "Deployment completed successfully!"
}

# Run main function
main "$@"
