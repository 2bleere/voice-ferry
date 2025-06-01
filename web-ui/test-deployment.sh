#!/bin/bash

# Voice Ferry Web UI Test Script
# This script validates the deployment and tests basic functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WEB_UI_URL="http://localhost:3001"
REDIS_HOST="localhost"
REDIS_PORT="6379"
ETCD_HOST="localhost"
ETCD_PORT="2379"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Function to test HTTP endpoint
test_endpoint() {
    local url="$1"
    local expected_status="$2"
    local description="$3"
    
    print_status "Testing $description"
    
    if response=$(curl -s -w "%{http_code}" -o /tmp/response.txt "$url"); then
        if [ "$response" = "$expected_status" ]; then
            print_success "$description - HTTP $response"
            return 0
        else
            print_error "$description - Expected HTTP $expected_status, got $response"
            return 1
        fi
    else
        print_error "$description - Connection failed"
        return 1
    fi
}

# Function to test service connectivity
test_service() {
    local host="$1"
    local port="$2"
    local service="$3"
    
    print_status "Testing $service connectivity"
    
    if nc -z "$host" "$port" 2>/dev/null; then
        print_success "$service is reachable at $host:$port"
        return 0
    else
        print_error "$service is not reachable at $host:$port"
        return 1
    fi
}

# Function to test Redis
test_redis() {
    print_status "Testing Redis functionality"
    
    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
            print_success "Redis ping successful"
            
            # Test Redis operations
            if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" set test_key "test_value" >/dev/null 2>&1; then
                if [ "$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" get test_key)" = "test_value" ]; then
                    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" del test_key >/dev/null 2>&1
                    print_success "Redis read/write operations working"
                    return 0
                else
                    print_error "Redis read operation failed"
                    return 1
                fi
            else
                print_error "Redis write operation failed"
                return 1
            fi
        else
            print_error "Redis ping failed"
            return 1
        fi
    else
        print_warning "redis-cli not available, skipping Redis functionality test"
        return 0
    fi
}

# Function to test etcd
test_etcd() {
    print_status "Testing etcd functionality"
    
    if command -v etcdctl >/dev/null 2>&1; then
        export ETCDCTL_API=3
        if etcdctl --endpoints="http://$ETCD_HOST:$ETCD_PORT" endpoint health >/dev/null 2>&1; then
            print_success "etcd health check passed"
            
            # Test etcd operations
            if etcdctl --endpoints="http://$ETCD_HOST:$ETCD_PORT" put test_key "test_value" >/dev/null 2>&1; then
                if [ "$(etcdctl --endpoints="http://$ETCD_HOST:$ETCD_PORT" get test_key --print-value-only)" = "test_value" ]; then
                    etcdctl --endpoints="http://$ETCD_HOST:$ETCD_PORT" del test_key >/dev/null 2>&1
                    print_success "etcd read/write operations working"
                    return 0
                else
                    print_error "etcd read operation failed"
                    return 1
                fi
            else
                print_error "etcd write operation failed"
                return 1
            fi
        else
            print_error "etcd health check failed"
            return 1
        fi
    else
        print_warning "etcdctl not available, skipping etcd functionality test"
        return 0
    fi
}

# Function to test Web UI functionality
test_web_ui() {
    print_status "Testing Web UI functionality"
    
    # Test main page
    test_endpoint "$WEB_UI_URL" "200" "Main page"
    
    # Test health endpoint
    test_endpoint "$WEB_UI_URL/api/health" "200" "Health endpoint"
    
    # Test static assets
    test_endpoint "$WEB_UI_URL/css/styles.css" "200" "CSS assets"
    test_endpoint "$WEB_UI_URL/js/app.js" "200" "JavaScript assets"
    
    # Test API endpoints (should return 401 without auth)
    test_endpoint "$WEB_UI_URL/api/dashboard/status" "401" "Dashboard API (unauthorized)"
    test_endpoint "$WEB_UI_URL/api/config" "401" "Config API (unauthorized)"
    
    # Test auth endpoint
    local auth_response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"username":"test","password":"test"}' \
        -w "%{http_code}" -o /tmp/auth_response.txt \
        "$WEB_UI_URL/api/auth/login")
    
    if [ "$auth_response" = "401" ] || [ "$auth_response" = "400" ]; then
        print_success "Auth endpoint responding correctly (rejecting invalid credentials)"
    else
        print_warning "Auth endpoint returned unexpected status: $auth_response"
    fi
}

# Function to test Docker services
test_docker_services() {
    print_status "Testing Docker services"
    
    if command -v docker >/dev/null 2>&1; then
        # Check if containers are running
        local containers=(
            "voice-ferry-web-ui"
            "sip-b2bua-redis" 
            "sip-b2bua-etcd"
        )
        
        for container in "${containers[@]}"; do
            if docker ps --format "table {{.Names}}" | grep -q "$container"; then
                print_success "$container container is running"
                
                # Check container health if health check is configured
                local health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "unknown")
                if [ "$health" = "healthy" ]; then
                    print_success "$container is healthy"
                elif [ "$health" = "starting" ]; then
                    print_warning "$container is still starting"
                elif [ "$health" = "unknown" ]; then
                    print_warning "$container has no health check configured"
                else
                    print_error "$container is unhealthy"
                fi
            else
                print_error "$container container is not running"
            fi
        done
    else
        print_warning "Docker not available, skipping container tests"
    fi
}

# Function to generate test report
generate_report() {
    local total_tests="$1"
    local passed_tests="$2"
    local failed_tests="$3"
    
    echo ""
    echo "==============================================="
    echo "           VOICE FERRY WEB UI TEST REPORT"
    echo "==============================================="
    echo "Total Tests: $total_tests"
    echo "Passed: $passed_tests"
    echo "Failed: $failed_tests"
    echo "Success Rate: $(( passed_tests * 100 / total_tests ))%"
    echo "==============================================="
    
    if [ "$failed_tests" -eq 0 ]; then
        print_success "All tests passed! Voice Ferry Web UI is ready."
        return 0
    else
        print_error "$failed_tests test(s) failed. Please check the issues above."
        return 1
    fi
}

# Main test execution
main() {
    echo "==============================================="
    echo "      VOICE FERRY WEB UI DEPLOYMENT TEST"
    echo "==============================================="
    echo ""
    
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    # Test service connectivity
    services=(
        "$WEB_UI_URL:3000:Web UI"
        "$REDIS_HOST:$REDIS_PORT:Redis"
        "$ETCD_HOST:$ETCD_PORT:etcd"
    )
    
    for service in "${services[@]}"; do
        IFS=':' read -r host port name <<< "$service"
        total_tests=$((total_tests + 1))
        if test_service "$host" "$port" "$name"; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Test Redis functionality
    total_tests=$((total_tests + 1))
    if test_redis; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    # Test etcd functionality
    total_tests=$((total_tests + 1))
    if test_etcd; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    # Test Web UI endpoints
    endpoints=(
        "$WEB_UI_URL:200:Main page"
        "$WEB_UI_URL/api/health:200:Health endpoint"
        "$WEB_UI_URL/css/styles.css:200:CSS assets"
        "$WEB_UI_URL/js/app.js:200:JavaScript assets"
        "$WEB_UI_URL/api/dashboard/status:401:Dashboard API"
        "$WEB_UI_URL/api/config:401:Config API"
    )
    
    for endpoint in "${endpoints[@]}"; do
        IFS=':' read -r url status desc <<< "$endpoint"
        total_tests=$((total_tests + 1))
        if test_endpoint "$url" "$status" "$desc"; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Test Docker services
    total_tests=$((total_tests + 1))
    if test_docker_services; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    # Generate final report
    generate_report "$total_tests" "$passed_tests" "$failed_tests"
}

# Check if script is being run directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
