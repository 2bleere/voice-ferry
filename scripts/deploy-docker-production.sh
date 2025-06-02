#!/bin/bash

# Deploy Voice Ferry SIP B2BUA to Production using Docker
# This script handles production deployment with Docker Compose

set -euo pipefail

# Configuration
DEPLOYMENT_ENV="${DEPLOYMENT_ENV:-production}"
DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-docker-compose.production.yml}"
REGISTRY="${REGISTRY:-ghcr.io}"
REPOSITORY="${REPOSITORY:-2bleere/voice-ferry}"
TAG="${TAG:-latest}"
CONFIG_DIR="${CONFIG_DIR:-/opt/voice-ferry/configs}"
DATA_DIR="${DATA_DIR:-/opt/voice-ferry/data}"
LOGS_DIR="${LOGS_DIR:-/opt/voice-ferry/logs}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    if [ ! -f "${DOCKER_COMPOSE_FILE}" ]; then
        log_error "Docker Compose file not found: ${DOCKER_COMPOSE_FILE}"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Setup production directories
setup_directories() {
    log_info "Setting up production directories..."
    
    sudo mkdir -p "${CONFIG_DIR}" "${DATA_DIR}" "${LOGS_DIR}"
    sudo chown -R $USER:$USER "${CONFIG_DIR}" "${DATA_DIR}" "${LOGS_DIR}"
    
    # Copy production configuration if it doesn't exist
    if [ ! -f "${CONFIG_DIR}/production.yaml" ]; then
        log_info "Copying production configuration..."
        cp configs/production.yaml "${CONFIG_DIR}/"
    fi
    
    log_info "Directories setup completed"
}

# Pull latest images
pull_images() {
    log_info "Pulling latest Docker images..."
    
    local main_image="${REGISTRY}/${REPOSITORY}:${TAG}"
    local webui_image="${REGISTRY}/${REPOSITORY}-webui:${TAG}"
    
    docker pull "${main_image}"
    docker pull "${webui_image}"
    
    log_info "Images pulled successfully"
}

# Backup current deployment
backup_deployment() {
    local backup_dir="/opt/voice-ferry/backups/$(date +%Y%m%d_%H%M%S)"
    
    log_info "Creating backup at ${backup_dir}..."
    
    sudo mkdir -p "${backup_dir}"
    
    # Backup configuration
    if [ -d "${CONFIG_DIR}" ]; then
        sudo cp -r "${CONFIG_DIR}" "${backup_dir}/"
    fi
    
    # Backup data
    if [ -d "${DATA_DIR}" ]; then
        sudo cp -r "${DATA_DIR}" "${backup_dir}/"
    fi
    
    # Export current Docker Compose configuration
    if docker-compose -f "${DOCKER_COMPOSE_FILE}" config > /dev/null 2>&1; then
        docker-compose -f "${DOCKER_COMPOSE_FILE}" config > "${backup_dir}/docker-compose-backup.yml"
    fi
    
    log_info "Backup created successfully"
}

# Deploy application
deploy_application() {
    log_info "Deploying application..."
    
    # Set environment variables for Docker Compose
    export VOICE_FERRY_IMAGE="${REGISTRY}/${REPOSITORY}:${TAG}"
    export VOICE_FERRY_WEBUI_IMAGE="${REGISTRY}/${REPOSITORY}-webui:${TAG}"
    export CONFIG_DIR="${CONFIG_DIR}"
    export DATA_DIR="${DATA_DIR}"
    export LOGS_DIR="${LOGS_DIR}"
    
    # Deploy using Docker Compose
    docker-compose -f "${DOCKER_COMPOSE_FILE}" up -d --remove-orphans
    
    log_info "Application deployed successfully"
}

# Health check
health_check() {
    log_info "Performing health checks..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log_debug "Health check attempt ${attempt}/${max_attempts}..."
        
        # Check if containers are running
        if docker-compose -f "${DOCKER_COMPOSE_FILE}" ps | grep -q "Up"; then
            # Check application health endpoint
            if curl -f http://localhost:8080/health > /dev/null 2>&1; then
                log_info "Health check passed"
                return 0
            fi
        fi
        
        sleep 10
        ((attempt++))
    done
    
    log_error "Health check failed after ${max_attempts} attempts"
    return 1
}

# Show deployment status
show_status() {
    log_info "Deployment status:"
    echo
    
    # Show running containers
    docker-compose -f "${DOCKER_COMPOSE_FILE}" ps
    echo
    
    # Show service logs (last 20 lines)
    log_info "Recent logs:"
    docker-compose -f "${DOCKER_COMPOSE_FILE}" logs --tail=20
}

# Rollback deployment
rollback_deployment() {
    log_warn "Rolling back deployment..."
    
    # Stop current deployment
    docker-compose -f "${DOCKER_COMPOSE_FILE}" down
    
    # Find latest backup
    local latest_backup=$(find /opt/voice-ferry/backups -type d -name "20*" | sort | tail -1)
    
    if [ -n "${latest_backup}" ] && [ -d "${latest_backup}" ]; then
        log_info "Restoring from backup: ${latest_backup}"
        
        # Restore configuration
        if [ -d "${latest_backup}/configs" ]; then
            sudo cp -r "${latest_backup}/configs/"* "${CONFIG_DIR}/"
        fi
        
        # Restore data
        if [ -d "${latest_backup}/data" ]; then
            sudo cp -r "${latest_backup}/data/"* "${DATA_DIR}/"
        fi
        
        # Deploy previous version
        if [ -f "${latest_backup}/docker-compose-backup.yml" ]; then
            docker-compose -f "${latest_backup}/docker-compose-backup.yml" up -d
        fi
        
        log_info "Rollback completed"
    else
        log_error "No backup found for rollback"
        exit 1
    fi
}

# Cleanup old backups and images
cleanup() {
    if [ "${CLEANUP_OLD:-false}" = "true" ]; then
        log_info "Cleaning up old backups and images..."
        
        # Keep only last 5 backups
        find /opt/voice-ferry/backups -type d -name "20*" | sort | head -n -5 | xargs rm -rf
        
        # Remove unused Docker images
        docker image prune -a -