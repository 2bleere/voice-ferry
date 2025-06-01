#!/bin/bash

# Voice Ferry Web UI Deployment Script
# This script handles the deployment of the Voice Ferry Web UI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="voice-ferry-ui"
COMPOSE_FILE="docker-compose.yml"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    print_success "Docker is running"
}

# Function to check if docker-compose is available
check_docker_compose() {
    if ! command -v docker-compose >/dev/null 2>&1; then
        print_error "docker-compose is not installed. Please install docker-compose and try again."
        exit 1
    fi
    print_success "docker-compose is available"
}

# Function to create necessary directories
create_directories() {
    print_status "Creating necessary directories..."
    mkdir -p config data logs backups ssl
    print_success "Directories created"
}

# Function to check environment configuration
check_environment() {
    print_status "Checking environment configuration..."
    
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            print_warning ".env file not found. Copying from .env.example"
            cp .env.example .env
            print_warning "Please edit .env file with your configuration before proceeding"
            read -p "Press Enter to continue after editing .env file..."
        else
            print_error ".env file not found and no .env.example available"
            exit 1
        fi
    fi
    
    print_success "Environment configuration checked"
}

# Function to build the application
build_app() {
    print_status "Building Voice Ferry Web UI..."
    docker-compose build
    print_success "Application built successfully"
}

# Function to start services
start_services() {
    print_status "Starting Voice Ferry Web UI services..."
    docker-compose up -d
    print_success "Services started successfully"
}

# Function to stop services
stop_services() {
    print_status "Stopping Voice Ferry Web UI services..."
    docker-compose down
    print_success "Services stopped successfully"
}

# Function to restart services
restart_services() {
    print_status "Restarting Voice Ferry Web UI services..."
    docker-compose restart
    print_success "Services restarted successfully"
}

# Function to view logs
view_logs() {
    print_status "Viewing logs for Voice Ferry Web UI..."
    docker-compose logs -f voice-ferry-ui
}

# Function to check service status
check_status() {
    print_status "Checking service status..."
    docker-compose ps
    
    print_status "Checking health status..."
    # Wait for services to start
    sleep 10
    
    # Check web UI health
    if curl -f http://localhost:3000/api/health >/dev/null 2>&1; then
        print_success "Web UI is healthy"
    else
        print_warning "Web UI health check failed"
    fi
    
    # Check Redis
    if docker-compose exec redis redis-cli ping >/dev/null 2>&1; then
        print_success "Redis is healthy"
    else
        print_warning "Redis health check failed"
    fi
    
    # Check etcd
    if docker-compose exec etcd etcdctl endpoint health >/dev/null 2>&1; then
        print_success "etcd is healthy"
    else
        print_warning "etcd health check failed"
    fi
}

# Function to backup configuration
backup_config() {
    print_status "Creating configuration backup..."
    BACKUP_FILE="backups/config-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
    tar -czf "$BACKUP_FILE" config/ data/
    print_success "Configuration backed up to $BACKUP_FILE"
}

# Function to restore configuration
restore_config() {
    if [ -z "$1" ]; then
        print_error "Please specify backup file to restore"
        print_status "Available backups:"
        ls -la backups/
        exit 1
    fi
    
    if [ ! -f "$1" ]; then
        print_error "Backup file $1 not found"
        exit 1
    fi
    
    print_status "Restoring configuration from $1..."
    tar -xzf "$1"
    print_success "Configuration restored from $1"
}

# Function to update application
update_app() {
    print_status "Updating Voice Ferry Web UI..."
    docker-compose pull
    docker-compose up -d --force-recreate
    print_success "Application updated successfully"
}

# Function to clean up
cleanup() {
    print_status "Cleaning up..."
    docker-compose down -v --remove-orphans
    docker system prune -f
    print_success "Cleanup completed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 {deploy|start|stop|restart|status|logs|backup|restore|update|cleanup}"
    echo ""
    echo "Commands:"
    echo "  deploy   - Full deployment (build and start)"
    echo "  start    - Start services"
    echo "  stop     - Stop services" 
    echo "  restart  - Restart services"
    echo "  status   - Check service status"
    echo "  logs     - View application logs"
    echo "  backup   - Backup configuration"
    echo "  restore  - Restore configuration from backup"
    echo "  update   - Update application"
    echo "  cleanup  - Clean up containers and images"
}

# Main execution
case "$1" in
    deploy)
        check_docker
        check_docker_compose
        create_directories
        check_environment
        build_app
        start_services
        sleep 15
        check_status
        print_success "Voice Ferry Web UI deployed successfully!"
        print_status "Access the web interface at: http://localhost:3000"
        ;;
    start)
        check_docker
        check_docker_compose
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    status)
        check_status
        ;;
    logs)
        view_logs
        ;;
    backup)
        backup_config
        ;;
    restore)
        restore_config "$2"
        ;;
    update)
        update_app
        ;;
    cleanup)
        cleanup
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
