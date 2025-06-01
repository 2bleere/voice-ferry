#!/bin/bash
# Voice Ferry System Health Check Script
# Comprehensive monitoring of all services including etcd status

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Service endpoints
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
ETCD_HOST="${ETCD_HOST:-localhost}"
ETCD_PORT="${ETCD_PORT:-2379}"
B2BUA_HOST="${B2BUA_HOST:-localhost}"
B2BUA_HTTP_PORT="${B2BUA_HTTP_PORT:-8080}"
B2BUA_GRPC_PORT="${B2BUA_GRPC_PORT:-50051}"
WEB_UI_HOST="${WEB_UI_HOST:-localhost}"
WEB_UI_PORT="${WEB_UI_PORT:-3001}"
RTPENGINE_HOST="${RTPENGINE_HOST:-localhost}"
RTPENGINE_PORT="${RTPENGINE_PORT:-22222}"

# Logging functions
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

log_header() {
    echo -e "${CYAN}[==]${NC} $1"
}

# Function to check if a port is open
check_port() {
    local host=$1
    local port=$2
    local timeout=${3:-5}
    
    if timeout "${timeout}" bash -c "echo >/dev/tcp/${host}/${port}" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to check Docker container status
check_docker_container() {
    local container_name=$1
    
    if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
        local status=$(docker inspect --format="{{.State.Status}}" "${container_name}" 2>/dev/null)
        local health=$(docker inspect --format="{{.State.Health.Status}}" "${container_name}" 2>/dev/null || echo "no-healthcheck")
        
        if [[ "$status" == "running" ]]; then
            if [[ "$health" == "healthy" ]]; then
                log_success "Container ${container_name} is running and healthy"
                return 0
            elif [[ "$health" == "no-healthcheck" ]]; then
                log_warning "Container ${container_name} is running (no health check)"
                return 0
            else
                log_error "Container ${container_name} is running but unhealthy (${health})"
                return 1
            fi
        else
            log_error "Container ${container_name} is not running (status: ${status})"
            return 1
        fi
    else
        log_error "Container ${container_name} not found"
        return 1
    fi
}

# Function to check Redis health
check_redis() {
    log_header "Checking Redis Service"
    
    local redis_ok=true
    
    # Check if Redis container is running
    if ! check_docker_container "voice-ferry-redis"; then
        redis_ok=false
    fi
    
    # Check Redis connectivity
    if check_port "${REDIS_HOST}" "${REDIS_PORT}"; then
        log_success "Redis port ${REDIS_PORT} is accessible"
        
        # Test Redis ping
        if docker exec voice-ferry-redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
            log_success "Redis responds to PING command"
            
            # Check Redis info
            local redis_info=$(docker exec voice-ferry-redis redis-cli info replication 2>/dev/null | head -1 || echo "error")
            if [[ "$redis_info" != "error" ]]; then
                log_success "Redis info command successful"
            else
                log_warning "Redis info command failed"
            fi
        else
            log_error "Redis does not respond to PING"
            redis_ok=false
        fi
    else
        log_error "Redis port ${REDIS_PORT} is not accessible"
        redis_ok=false
    fi
    
    return $([[ "$redis_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to check etcd health
check_etcd() {
    log_header "Checking etcd Service"
    
    local etcd_ok=true
    
    # Check if etcd container is running
    if ! check_docker_container "voice-ferry-etcd"; then
        etcd_ok=false
    fi
    
    # Check etcd connectivity
    if check_port "${ETCD_HOST}" "${ETCD_PORT}"; then
        log_success "etcd port ${ETCD_PORT} is accessible"
        
        # Test etcd health endpoint
        if docker exec voice-ferry-etcd etcdctl endpoint health --endpoints=http://localhost:2379 2>/dev/null | grep -q "healthy"; then
            log_success "etcd health check passed"
            
            # Check etcd cluster status
            local cluster_status=$(docker exec voice-ferry-etcd etcdctl endpoint status --endpoints=http://localhost:2379 --write-out=fields 2>/dev/null | grep "Leader" || echo "error")
            if [[ "$cluster_status" != "error" ]]; then
                log_success "etcd cluster status check successful"
            else
                log_warning "etcd cluster status check failed"
            fi
            
            # Test basic etcd operations
            if docker exec voice-ferry-etcd etcdctl put health-check "$(date)" --endpoints=http://localhost:2379 2>/dev/null; then
                if docker exec voice-ferry-etcd etcdctl get health-check --endpoints=http://localhost:2379 2>/dev/null | tail -1 | grep -q "$(date '+%Y-%m-%d')"; then
                    log_success "etcd read/write operations working"
                    docker exec voice-ferry-etcd etcdctl del health-check --endpoints=http://localhost:2379 2>/dev/null
                else
                    log_warning "etcd read operation failed"
                fi
            else
                log_warning "etcd write operation failed"
            fi
        else
            log_error "etcd health check failed"
            etcd_ok=false
        fi
    else
        log_error "etcd port ${ETCD_PORT} is not accessible"
        etcd_ok=false
    fi
    
    return $([[ "$etcd_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to check B2BUA health
check_b2bua() {
    log_header "Checking B2BUA Service"
    
    local b2bua_ok=true
    
    # Check if B2BUA container is running
    if ! check_docker_container "voice-ferry-b2bua"; then
        b2bua_ok=false
    fi
    
    # Check B2BUA HTTP health endpoint
    if check_port "${B2BUA_HOST}" "${B2BUA_HTTP_PORT}"; then
        log_success "B2BUA HTTP port ${B2BUA_HTTP_PORT} is accessible"
        
        # Test health endpoint
        if curl -f -s "http://${B2BUA_HOST}:${B2BUA_HTTP_PORT}/healthz/live" >/dev/null 2>&1; then
            log_success "B2BUA health endpoint responds"
        else
            log_error "B2BUA health endpoint not responding"
            b2bua_ok=false
        fi
        
        # Test readiness endpoint
        if curl -f -s "http://${B2BUA_HOST}:${B2BUA_HTTP_PORT}/healthz/ready" >/dev/null 2>&1; then
            log_success "B2BUA readiness endpoint responds"
        else
            log_warning "B2BUA readiness endpoint not responding"
        fi
    else
        log_error "B2BUA HTTP port ${B2BUA_HTTP_PORT} is not accessible"
        b2bua_ok=false
    fi
    
    # Check B2BUA gRPC port
    if check_port "${B2BUA_HOST}" "${B2BUA_GRPC_PORT}"; then
        log_success "B2BUA gRPC port ${B2BUA_GRPC_PORT} is accessible"
    else
        log_error "B2BUA gRPC port ${B2BUA_GRPC_PORT} is not accessible"
        b2bua_ok=false
    fi
    
    return $([[ "$b2bua_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to check Web UI health
check_web_ui() {
    log_header "Checking Web UI Service"
    
    local webui_ok=true
    
    # Check if Web UI container is running
    if ! check_docker_container "voice-ferry-ui"; then
        webui_ok=false
    fi
    
    # Check Web UI port
    if check_port "${WEB_UI_HOST}" "${WEB_UI_PORT}"; then
        log_success "Web UI port ${WEB_UI_PORT} is accessible"
        
        # Test health endpoint
        if curl -f -s "http://${WEB_UI_HOST}:${WEB_UI_PORT}/api/health" >/dev/null 2>&1; then
            log_success "Web UI health endpoint responds"
            
            # Test dashboard status endpoint
            if curl -f -s "http://${WEB_UI_HOST}:${WEB_UI_PORT}/api/dashboard/status" >/dev/null 2>&1; then
                log_success "Web UI dashboard status endpoint responds"
            else
                log_warning "Web UI dashboard status endpoint not responding"
            fi
        else
            log_error "Web UI health endpoint not responding"
            webui_ok=false
        fi
    else
        log_error "Web UI port ${WEB_UI_PORT} is not accessible"
        webui_ok=false
    fi
    
    return $([[ "$webui_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to check RTPEngine health
check_rtpengine() {
    log_header "Checking RTPEngine Service"
    
    local rtpengine_ok=true
    
    # Check if RTPEngine container is running
    if ! check_docker_container "voice-ferry-rtpengine"; then
        rtpengine_ok=false
    fi
    
    # Check RTPEngine port
    if check_port "${RTPENGINE_HOST}" "${RTPENGINE_PORT}"; then
        log_success "RTPEngine port ${RTPENGINE_PORT} is accessible"
    else
        log_error "RTPEngine port ${RTPENGINE_PORT} is not accessible"
        rtpengine_ok=false
    fi
    
    return $([[ "$rtpengine_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to test etcd status monitoring integration
check_etcd_monitoring() {
    log_header "Checking etcd Status Monitoring Integration"
    
    local monitoring_ok=true
    
    # Test if Web UI can retrieve system status including etcd
    if curl -f -s "http://${WEB_UI_HOST}:${WEB_UI_PORT}/api/dashboard/status" 2>/dev/null | grep -q "etcd"; then
        log_success "Web UI reports etcd status"
    else
        log_warning "Web UI does not report etcd status"
        monitoring_ok=false
    fi
    
    # Check if Web UI monitoring service can connect to etcd
    local web_ui_logs=$(docker logs voice-ferry-ui --tail 50 2>/dev/null | grep -i etcd || echo "")
    if [[ -n "$web_ui_logs" ]]; then
        if echo "$web_ui_logs" | grep -q -i "error\|fail"; then
            log_warning "Web UI shows etcd connection issues in logs"
            monitoring_ok=false
        else
            log_success "Web UI etcd integration looks healthy in logs"
        fi
    else
        log_info "No etcd-related logs found in Web UI"
    fi
    
    return $([[ "$monitoring_ok" == "true" ]] && echo 0 || echo 1)
}

# Function to display overall system status
display_system_summary() {
    echo ""
    log_header "System Health Summary"
    echo ""
    
    # Get overall status from each service
    local redis_status="❌"
    local etcd_status="❌"
    local b2bua_status="❌"
    local webui_status="❌"
    local rtpengine_status="❌"
    local monitoring_status="❌"
    
    if check_redis >/dev/null 2>&1; then redis_status="✅"; fi
    if check_etcd >/dev/null 2>&1; then etcd_status="✅"; fi
    if check_b2bua >/dev/null 2>&1; then b2bua_status="✅"; fi
    if check_web_ui >/dev/null 2>&1; then webui_status="✅"; fi
    if check_rtpengine >/dev/null 2>&1; then rtpengine_status="✅"; fi
    if check_etcd_monitoring >/dev/null 2>&1; then monitoring_status="✅"; fi
    
    echo "┌─────────────────────┬────────┬─────────────────────────────┐"
    echo "│ Service             │ Status │ Details                     │"
    echo "├─────────────────────┼────────┼─────────────────────────────┤"
    printf "│ %-19s │   %s   │ %-27s │\n" "Redis" "$redis_status" "Port ${REDIS_PORT}"
    printf "│ %-19s │   %s   │ %-27s │\n" "etcd" "$etcd_status" "Port ${ETCD_PORT}"
    printf "│ %-19s │   %s   │ %-27s │\n" "B2BUA" "$b2bua_status" "HTTP:${B2BUA_HTTP_PORT} gRPC:${B2BUA_GRPC_PORT}"
    printf "│ %-19s │   %s   │ %-27s │\n" "Web UI" "$webui_status" "Port ${WEB_UI_PORT}"
    printf "│ %-19s │   %s   │ %-27s │\n" "RTPEngine" "$rtpengine_status" "Port ${RTPENGINE_PORT}"
    printf "│ %-19s │   %s   │ %-27s │\n" "etcd Monitoring" "$monitoring_status" "Web UI Integration"
    echo "└─────────────────────┴────────┴─────────────────────────────┘"
    echo ""
    
    # Overall system status
    if [[ "$redis_status" == "✅" && "$etcd_status" == "✅" && "$b2bua_status" == "✅" && "$webui_status" == "✅" && "$rtpengine_status" == "✅" ]]; then
        log_success "Overall System Status: HEALTHY"
        echo ""
        echo "🎉 All services are running and healthy!"
        echo "   Web UI available at: http://${WEB_UI_HOST}:${WEB_UI_PORT}"
        echo "   etcd status monitoring is working correctly"
        return 0
    else
        log_error "Overall System Status: DEGRADED"
        echo ""
        echo "⚠️  Some services are not healthy. Check the details above."
        return 1
    fi
}

# Main function
main() {
    echo "=================================================="
    echo "  Voice Ferry System Health Check"
    echo "  $(date)"
    echo "=================================================="
    echo ""
    
    local overall_status=0
    
    # Run all health checks
    if ! check_redis; then overall_status=1; fi
    echo ""
    
    if ! check_etcd; then overall_status=1; fi
    echo ""
    
    if ! check_b2bua; then overall_status=1; fi
    echo ""
    
    if ! check_web_ui; then overall_status=1; fi
    echo ""
    
    if ! check_rtpengine; then overall_status=1; fi
    echo ""
    
    if ! check_etcd_monitoring; then overall_status=1; fi
    echo ""
    
    # Display summary
    display_system_summary
    
    exit $overall_status
}

# Handle script arguments
case "${1:-check}" in
    "check")
        main
        ;;
    "redis")
        check_redis
        ;;
    "etcd")
        check_etcd
        ;;
    "b2bua")
        check_b2bua
        ;;
    "webui")
        check_web_ui
        ;;
    "rtpengine")
        check_rtpengine
        ;;
    "monitoring")
        check_etcd_monitoring
        ;;
    "summary")
        display_system_summary
        ;;
    *)
        echo "Usage: $0 {check|redis|etcd|b2bua|webui|rtpengine|monitoring|summary}"
        echo ""
        echo "Commands:"
        echo "  check      - Run complete health check (default)"
        echo "  redis      - Check Redis service only"
        echo "  etcd       - Check etcd service only"
        echo "  b2bua      - Check B2BUA service only"
        echo "  webui      - Check Web UI service only"
        echo "  rtpengine  - Check RTPEngine service only"
        echo "  monitoring - Check etcd monitoring integration"
        echo "  summary    - Display system status summary"
        exit 1
        ;;
esac
