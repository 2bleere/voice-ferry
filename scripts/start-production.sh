#!/bin/bash

# Voice Ferry Production Environment Startup Script
# This script starts all components for a local production setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE} Voice Ferry Production Environment${NC}"
echo -e "${BLUE}======================================${NC}"

# Check if binary exists
if [ ! -f "build/b2bua-server" ]; then
    echo -e "${RED}Error: b2bua-server binary not found. Please run 'make build' first.${NC}"
    exit 1
fi

# Create log directory
mkdir -p logs

# Function to check if a service is running on a port
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}Waiting for $service to be ready on port $port...${NC}"
    while [ $attempt -le $max_attempts ]; do
        if check_port $port; then
            echo -e "${GREEN}$service is ready!${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}$service failed to start after $max_attempts seconds${NC}"
    return 1
}

# Start Redis
echo -e "${YELLOW}Starting Redis...${NC}"
if check_port 6379; then
    echo -e "${GREEN}Redis is already running on port 6379${NC}"
else
    redis-server configs/redis/redis.conf --daemonize yes --logfile logs/redis.log
    wait_for_service "Redis" 6379
fi

# Start etcd
echo -e "${YELLOW}Starting etcd...${NC}"
if check_port 2379; then
    echo -e "${GREEN}etcd is already running on port 2379${NC}"
else
    nohup etcd --name voice-ferry-etcd \
               --data-dir ./logs/etcd-data \
               --listen-client-urls http://0.0.0.0:2379 \
               --advertise-client-urls http://0.0.0.0:2379 \
               --listen-peer-urls http://0.0.0.0:2380 \
               --initial-advertise-peer-urls http://0.0.0.0:2380 \
               --initial-cluster voice-ferry-etcd=http://0.0.0.0:2380 \
               --initial-cluster-token voice-ferry-cluster \
               --initial-cluster-state new \
               --log-level info > logs/etcd.log 2>&1 &
    wait_for_service "etcd" 2379
fi

# Set production environment variables
export ENVIRONMENT=production
export LOG_LEVEL=info
export CONFIG_FILE=configs/production.yaml
export REDIS_URL=redis://localhost:6379/0
export ETCD_ENDPOINTS=http://localhost:2379
export JWT_SIGNING_KEY=$(grep JWT_SIGNING_KEY .env | cut -d'=' -f2)

# Start the web UI if available
echo -e "${YELLOW}Starting Web UI...${NC}"
if [ -d "web-ui" ] && [ -f "web-ui/server.js" ]; then
    cd web-ui
    if check_port 3001; then
        echo -e "${GREEN}Web UI is already running on port 3001${NC}"
    else
        nohup node server.js > ../logs/web-ui.log 2>&1 &
        cd ..
        wait_for_service "Web UI" 3001
    fi
    cd ..
else
    echo -e "${YELLOW}Web UI not found, skipping...${NC}"
fi

# Start Voice Ferry B2BUA
echo -e "${YELLOW}Starting Voice Ferry B2BUA...${NC}"
if check_port 5060; then
    echo -e "${RED}Warning: Port 5060 is already in use${NC}"
fi

if check_port 50051; then
    echo -e "${RED}Warning: Port 50051 (gRPC) is already in use${NC}"
fi

echo -e "${GREEN}Starting Voice Ferry B2BUA server...${NC}"
./build/b2bua-server &
B2BUA_PID=$!

# Wait a moment for the server to start
sleep 3

# Check if the server is running
if ps -p $B2BUA_PID > /dev/null; then
    echo -e "${GREEN}Voice Ferry B2BUA started successfully (PID: $B2BUA_PID)${NC}"
else
    echo -e "${RED}Failed to start Voice Ferry B2BUA${NC}"
    exit 1
fi

echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN} Voice Ferry Production Environment Started${NC}"
echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}Services running:${NC}"
echo -e "  • Redis:          localhost:6379"
echo -e "  • etcd:           localhost:2379"
echo -e "  • Web UI:         http://localhost:3001"
echo -e "  • SIP B2BUA:      localhost:5060 (UDP/TCP)"
echo -e "  • gRPC API:       localhost:50051"
echo -e "  • Health Check:   http://localhost:8080/healthz"
echo -e "${BLUE}======================================${NC}"
echo -e "${YELLOW}Logs are available in the 'logs/' directory${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"
    
    # Stop B2BUA
    if ps -p $B2BUA_PID > /dev/null; then
        kill $B2BUA_PID
        echo -e "${GREEN}Voice Ferry B2BUA stopped${NC}"
    fi
    
    # Stop Web UI
    pkill -f "node server.js" || true
    echo -e "${GREEN}Web UI stopped${NC}"
    
    # Stop etcd
    pkill -f "etcd" || true
    echo -e "${GREEN}etcd stopped${NC}"
    
    # Stop Redis
    redis-cli shutdown || true
    echo -e "${GREEN}Redis stopped${NC}"
    
    echo -e "${GREEN}All services stopped${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Wait for the B2BUA process
wait $B2BUA_PID
