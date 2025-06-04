#!/bin/bash

# RTPEngine Health Check Script
# Tests RTPEngine health using the same UDP protocol as Voice Ferry

set -euo pipefail

# Configuration
RTPENGINE_HOST="${1:-192.168.1.208}"  # From your ARM deployment config
RTPENGINE_PORT="${2:-22222}"
TIMEOUT="${3:-5}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Function to create bencode-wrapped ping command
create_ping_command() {
    local json_cmd='{"command":"ping"}'
    local json_length=${#json_cmd}
    echo "d7:command${json_length}:${json_cmd}e"
}

# Function to test RTPEngine health
test_rtpengine_health() {
    local host=$1
    local port=$2
    local timeout=$3
    
    log_info "Testing RTPEngine health at ${host}:${port}"
    
    # Check if host is reachable
    if ! ping -c 1 -W 2 "$host" >/dev/null 2>&1; then
        log_error "Host $host is not reachable via ping"
        return 1
    fi
    
    log_success "Host $host is reachable"
    
    # Create the ping command in bencode format
    local ping_cmd=$(create_ping_command)
    log_info "Sending ping command: $ping_cmd"
    
    # Send UDP packet and capture response
    local response
    if command -v nc >/dev/null 2>&1; then
        # Using netcat
        response=$(echo -n "$ping_cmd" | timeout "$timeout" nc -u -w "$timeout" "$host" "$port" 2>/dev/null || echo "TIMEOUT")
    elif command -v socat >/dev/null 2>&1; then
        # Using socat as fallback
        response=$(echo -n "$ping_cmd" | timeout "$timeout" socat - UDP:"$host":"$port" 2>/dev/null || echo "TIMEOUT")
    else
        log_error "Neither 'nc' (netcat) nor 'socat' found. Please install one of them."
        return 1
    fi
    
    # Analyze response
    if [[ "$response" == "TIMEOUT" ]] || [[ -z "$response" ]]; then
        log_error "No response received (timeout or connection failed)"
        return 1
    fi
    
    log_info "Raw response: $response"
    
    # Extract JSON from bencode response (simplified parsing)
    if echo "$response" | grep -q '"result":"ok"'; then
        log_success "RTPEngine is healthy - returned 'result: ok'"
        return 0
    elif echo "$response" | grep -q '"result"'; then
        local result=$(echo "$response" | sed -n 's/.*"result":"\([^"]*\)".*/\1/p')
        log_warning "RTPEngine responded but not healthy - result: $result"
        return 1
    else
        log_error "Invalid response format or unexpected response"
        log_info "Expected JSON response with 'result' field"
        return 1
    fi
}

# Function to test connectivity without health check
test_port_connectivity() {
    local host=$1
    local port=$2
    local timeout=$3
    
    log_info "Testing basic UDP port connectivity to ${host}:${port}"
    
    if command -v nc >/dev/null 2>&1; then
        if echo "test" | timeout "$timeout" nc -u -w 1 "$host" "$port" >/dev/null 2>&1; then
            log_success "UDP port $port is open and accepting connections"
            return 0
        else
            log_error "UDP port $port is not accessible or not responding"
            return 1
        fi
    else
        log_warning "Cannot test port connectivity - netcat not available"
        return 1
    fi
}

# Main execution
main() {
    echo "=============================================="
    echo "         RTPEngine Health Check Test         "
    echo "=============================================="
    echo "Target: ${RTPENGINE_HOST}:${RTPENGINE_PORT}"
    echo "Timeout: ${TIMEOUT}s"
    echo ""
    
    # Test 1: Basic connectivity
    if test_port_connectivity "$RTPENGINE_HOST" "$RTPENGINE_PORT" "$TIMEOUT"; then
        echo ""
        # Test 2: Health check protocol
        if test_rtpengine_health "$RTPENGINE_HOST" "$RTPENGINE_PORT" "$TIMEOUT"; then
            echo ""
            log_success "✅ RTPEngine health check PASSED"
            exit 0
        else
            echo ""
            log_error "❌ RTPEngine health check FAILED"
            exit 1
        fi
    else
        echo ""
        log_error "❌ Basic connectivity test FAILED"
        exit 1
    fi
}

# Show usage if help requested
if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    echo "Usage: $0 [HOST] [PORT] [TIMEOUT]"
    echo ""
    echo "Parameters:"
    echo "  HOST     RTPEngine host (default: 192.168.1.208)"
    echo "  PORT     RTPEngine ng port (default: 22222)"
    echo "  TIMEOUT  Connection timeout in seconds (default: 5)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Test default ARM deployment"
    echo "  $0 127.0.0.1                        # Test localhost"
    echo "  $0 192.168.1.100 22222 10          # Custom host, port, and timeout"
    exit 0
fi

main "$@"
