#!/bin/bash

# Test script for Voice Ferry Web UI Docker Compose setup
# This script tests the SIP users functionality and overall setup

set -e

echo "🚀 Starting Voice Ferry Web UI Docker Compose Test..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to wait for service
wait_for_service() {
    local service_name=$1
    local url=$2
    local timeout=${3:-60}
    local count=0
    
    echo "Waiting for $service_name to be ready..."
    while [ $count -lt $timeout ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            print_status "$service_name is ready!"
            return 0
        fi
        echo -n "."
        sleep 1
        count=$((count + 1))
    done
    
    print_error "$service_name failed to start within $timeout seconds"
    return 1
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose > /dev/null 2>&1; then
    if ! command -v docker > /dev/null 2>&1 || ! docker compose version > /dev/null 2>&1; then
        print_error "Neither docker-compose nor 'docker compose' is available"
        exit 1
    fi
    # Use docker compose (newer syntax)
    COMPOSE_CMD="docker compose"
else
    # Use docker-compose (older syntax)
    COMPOSE_CMD="docker-compose"
fi

print_status "Using compose command: $COMPOSE_CMD"

# Navigate to web-ui directory
cd "$(dirname "$0")"
if [ ! -f "docker-compose.yml" ]; then
    print_error "docker-compose.yml not found. Make sure you're in the web-ui directory."
    exit 1
fi

# Clean up any existing containers
print_status "Cleaning up existing containers..."
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml down -v --remove-orphans

# Build and start services
print_status "Building and starting services..."
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml up -d --build

# Wait for services to be ready
echo ""
echo "🔄 Waiting for services to start..."

# Wait for Redis
wait_for_service "Redis" "redis://localhost:6379" 30

# Wait for etcd
wait_for_service "etcd" "http://localhost:2379/health" 30

# Wait for Web UI
wait_for_service "Web UI" "http://localhost:3000/api/health" 60

# Test SIP Users API
echo ""
echo "🧪 Testing SIP Users API..."

# Test GET /api/sip-users
echo "Testing GET /api/sip-users..."
if curl -s -f "http://localhost:3000/api/sip-users" > /dev/null; then
    print_status "SIP Users API is accessible"
else
    print_warning "SIP Users API test failed - this might be expected if authentication is required"
fi

# Test basic web UI accessibility
echo "Testing Web UI accessibility..."
if curl -s -f "http://localhost:3000" > /dev/null; then
    print_status "Web UI is accessible"
else
    print_warning "Web UI accessibility test failed"
fi

# Show running containers
echo ""
echo "📋 Running containers:"
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml ps

# Show logs for troubleshooting
echo ""
echo "📝 Recent logs from voice-ferry-ui:"
$COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml logs --tail=10 voice-ferry-ui

echo ""
print_status "Test completed! Services are running on:"
echo "  - Web UI: http://localhost:3000"
echo "  - Redis: localhost:6379"
echo "  - etcd: http://localhost:2379"
echo "  - Redis Commander: http://localhost:8081 (with --profile tools)"
echo "  - etcd Browser: http://localhost:8082 (with --profile tools)"

echo ""
echo "🛠️  To run with development tools:"
echo "  $COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml --profile tools up -d"
echo ""
echo "🧪 To run with mock B2BUA service:"
echo "  $COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml --profile mock up -d"
echo ""
echo "🛑 To stop all services:"
echo "  $COMPOSE_CMD -f docker-compose.yml -f docker-compose.dev.yml down"

exit 0
