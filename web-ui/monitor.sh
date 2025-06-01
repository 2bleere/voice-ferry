#!/bin/bash

# Voice Ferry System Monitor
# Monitors the complete Voice Ferry ecosystem including B2BUA and Web UI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
B2BUA_GRPC_ENDPOINT="localhost:50051"
WEB_UI_ENDPOINT="http://localhost:3000"
REDIS_ENDPOINT="localhost:6379"
ETCD_ENDPOINT="localhost:2379"
MONITORING_INTERVAL=5

# Function to print colored output
print_header() {
    echo -e "${PURPLE}[MONITOR]${NC} $1"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_metric() {
    echo -e "${CYAN}[METRIC]${NC} $1"
}

# Function to check service health
check_service_health() {
    local service="$1"
    local endpoint="$2"
    local method="$3"
    
    case "$method" in
        "http")
            if curl -f -s "$endpoint/api/health" >/dev/null 2>&1; then
                print_success "$service is healthy"
                return 0
            else
                print_error "$service is unhealthy"
                return 1
            fi
            ;;
        "tcp")
            local host=$(echo $endpoint | cut -d: -f1)
            local port=$(echo $endpoint | cut -d: -f2)
            if nc -z "$host" "$port" 2>/dev/null; then
                print_success "$service is reachable"
                return 0
            else
                print_error "$service is unreachable"
                return 1
            fi
            ;;
        "grpc")
            if command -v grpcurl >/dev/null 2>&1; then
                if grpcurl -plaintext "$endpoint" list >/dev/null 2>&1; then
                    print_success "$service gRPC is responding"
                    return 0
                else
                    print_error "$service gRPC is not responding"
                    return 1
                fi
            else
                print_warning "grpcurl not available, skipping gRPC check for $service"
                return 0
            fi
            ;;
        "redis")
            if command -v redis-cli >/dev/null 2>&1; then
                local host=$(echo $endpoint | cut -d: -f1)
                local port=$(echo $endpoint | cut -d: -f2)
                if redis-cli -h "$host" -p "$port" ping >/dev/null 2>&1; then
                    print_success "$service is responding"
                    return 0
                else
                    print_error "$service is not responding"
                    return 1
                fi
            else
                print_warning "redis-cli not available, using TCP check for $service"
                check_service_health "$service" "$endpoint" "tcp"
                return $?
            fi
            ;;
    esac
}

# Function to get system metrics
get_system_metrics() {
    print_header "System Metrics"
    
    # CPU usage
    if command -v top >/dev/null 2>&1; then
        local cpu_usage=$(top -l 1 -s 0 | grep "CPU usage" | awk '{print $3}' | sed 's/%//')
        print_metric "CPU Usage: ${cpu_usage}%"
    fi
    
    # Memory usage
    if command -v vm_stat >/dev/null 2>&1; then
        local memory_info=$(vm_stat | grep -E "(free|inactive|wired|compressed)")
        local free_pages=$(echo "$memory_info" | grep "Pages free" | awk '{print $3}' | sed 's/\.//')
        local inactive_pages=$(echo "$memory_info" | grep "Pages inactive" | awk '{print $3}' | sed 's/\.//')
        local wired_pages=$(echo "$memory_info" | grep "Pages wired down" | awk '{print $4}' | sed 's/\.//')
        
        if [ -n "$free_pages" ] && [ -n "$inactive_pages" ] && [ -n "$wired_pages" ]; then
            local page_size=4096
            local free_mb=$(( (free_pages + inactive_pages) * page_size / 1024 / 1024 ))
            local used_mb=$(( wired_pages * page_size / 1024 / 1024 ))
            local total_mb=$(( free_mb + used_mb ))
            local usage_percent=$(( used_mb * 100 / total_mb ))
            
            print_metric "Memory Usage: ${usage_percent}% (${used_mb}MB used, ${free_mb}MB free)"
        fi
    fi
    
    # Disk usage
    if command -v df >/dev/null 2>&1; then
        local disk_usage=$(df -h / | awk 'NR==2 {print $5}')
        print_metric "Disk Usage: $disk_usage"
    fi
    
    # Load average
    if command -v uptime >/dev/null 2>&1; then
        local load_avg=$(uptime | awk -F'load averages: ' '{print $2}')
        print_metric "Load Average: $load_avg"
    fi
}

# Function to get B2BUA metrics
get_b2bua_metrics() {
    print_header "Voice Ferry B2BUA Metrics"
    
    # Check if B2BUA is running
    if pgrep -f "b2bua" >/dev/null 2>&1; then
        print_success "B2BUA process is running"
        
        # Get process info
        local pid=$(pgrep -f "b2bua" | head -1)
        local cpu_usage=$(ps -p $pid -o %cpu --no-headers 2>/dev/null || echo "N/A")
        local mem_usage=$(ps -p $pid -o %mem --no-headers 2>/dev/null || echo "N/A")
        local rss=$(ps -p $pid -o rss --no-headers 2>/dev/null || echo "N/A")
        
        print_metric "B2BUA CPU: ${cpu_usage}%"
        print_metric "B2BUA Memory: ${mem_usage}% (${rss}KB RSS)"
        
        # Check listening ports
        if command -v lsof >/dev/null 2>&1; then
            local sip_port=$(lsof -Pi :5060 -sTCP:LISTEN -t 2>/dev/null | wc -l | tr -d ' ')
            local grpc_port=$(lsof -Pi :50051 -sTCP:LISTEN -t 2>/dev/null | wc -l | tr -d ' ')
            
            if [ "$sip_port" -gt 0 ]; then
                print_success "SIP port (5060) is listening"
            else
                print_warning "SIP port (5060) is not listening"
            fi
            
            if [ "$grpc_port" -gt 0 ]; then
                print_success "gRPC port (50051) is listening"
            else
                print_warning "gRPC port (50051) is not listening"
            fi
        fi
    else
        print_error "B2BUA process is not running"
    fi
    
    # Try to get metrics via gRPC
    if command -v grpcurl >/dev/null 2>&1; then
        print_status "Attempting to get B2BUA metrics via gRPC..."
        # This would need the actual gRPC service definition
        # grpcurl -plaintext $B2BUA_GRPC_ENDPOINT b2bua.v1.MetricsService/GetMetrics
    fi
}

# Function to get Web UI metrics
get_webui_metrics() {
    print_header "Web UI Metrics"
    
    # Check if Web UI is running
    if pgrep -f "node.*server.js" >/dev/null 2>&1; then
        print_success "Web UI process is running"
        
        # Get process info
        local pid=$(pgrep -f "node.*server.js" | head -1)
        local cpu_usage=$(ps -p $pid -o %cpu --no-headers 2>/dev/null || echo "N/A")
        local mem_usage=$(ps -p $pid -o %mem --no-headers 2>/dev/null || echo "N/A")
        local rss=$(ps -p $pid -o rss --no-headers 2>/dev/null || echo "N/A")
        
        print_metric "Web UI CPU: ${cpu_usage}%"
        print_metric "Web UI Memory: ${mem_usage}% (${rss}KB RSS)"
    else
        print_error "Web UI process is not running"
    fi
    
    # Check Web UI health endpoint
    if curl -f -s "$WEB_UI_ENDPOINT/api/health" >/dev/null 2>&1; then
        local health_data=$(curl -s "$WEB_UI_ENDPOINT/api/health" 2>/dev/null)
        if command -v jq >/dev/null 2>&1; then
            local uptime=$(echo "$health_data" | jq -r '.uptime // "N/A"')
            local memory_used=$(echo "$health_data" | jq -r '.memory.rss // "N/A"')
            local memory_heap=$(echo "$health_data" | jq -r '.memory.heapUsed // "N/A"')
            
            print_metric "Web UI Uptime: ${uptime}s"
            print_metric "Web UI RSS Memory: ${memory_used} bytes"
            print_metric "Web UI Heap Memory: ${memory_heap} bytes"
        else
            print_metric "Web UI Health: OK (jq not available for detailed metrics)"
        fi
    else
        print_warning "Web UI health endpoint not accessible"
    fi
}

# Function to get Redis metrics
get_redis_metrics() {
    print_header "Redis Metrics"
    
    if command -v redis-cli >/dev/null 2>&1; then
        local host=$(echo $REDIS_ENDPOINT | cut -d: -f1)
        local port=$(echo $REDIS_ENDPOINT | cut -d: -f2)
        
        if redis-cli -h "$host" -p "$port" ping >/dev/null 2>&1; then
            print_success "Redis is responding"
            
            # Get Redis info
            local redis_info=$(redis-cli -h "$host" -p "$port" info 2>/dev/null)
            
            if [ -n "$redis_info" ]; then
                local connected_clients=$(echo "$redis_info" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r')
                local used_memory=$(echo "$redis_info" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r')
                local total_commands=$(echo "$redis_info" | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r')
                local keyspace_hits=$(echo "$redis_info" | grep "keyspace_hits:" | cut -d: -f2 | tr -d '\r')
                local keyspace_misses=$(echo "$redis_info" | grep "keyspace_misses:" | cut -d: -f2 | tr -d '\r')
                
                print_metric "Redis Connected Clients: $connected_clients"
                print_metric "Redis Memory Usage: $used_memory"
                print_metric "Redis Total Commands: $total_commands"
                print_metric "Redis Cache Hit Ratio: $(( keyspace_hits * 100 / (keyspace_hits + keyspace_misses) ))%" 2>/dev/null || print_metric "Redis Cache Hit Ratio: N/A"
            fi
        else
            print_error "Redis is not responding"
        fi
    else
        print_warning "redis-cli not available"
        check_service_health "Redis" "$REDIS_ENDPOINT" "tcp"
    fi
}

# Function to get etcd metrics
get_etcd_metrics() {
    print_header "etcd Metrics"
    
    if command -v etcdctl >/dev/null 2>&1; then
        export ETCDCTL_API=3
        
        if etcdctl --endpoints="http://$ETCD_ENDPOINT" endpoint health >/dev/null 2>&1; then
            print_success "etcd is healthy"
            
            # Get etcd status
            local etcd_status=$(etcdctl --endpoints="http://$ETCD_ENDPOINT" endpoint status --write-out=table 2>/dev/null)
            if [ -n "$etcd_status" ]; then
                print_metric "etcd Status:"
                echo "$etcd_status" | sed 's/^/  /'
            fi
            
            # Get member list
            local members=$(etcdctl --endpoints="http://$ETCD_ENDPOINT" member list 2>/dev/null | wc -l)
            print_metric "etcd Members: $members"
        else
            print_error "etcd is not healthy"
        fi
    else
        print_warning "etcdctl not available"
        check_service_health "etcd" "$ETCD_ENDPOINT" "tcp"
    fi
}

# Function to run continuous monitoring
continuous_monitoring() {
    print_header "Starting Continuous Monitoring (Interval: ${MONITORING_INTERVAL}s)"
    print_status "Press Ctrl+C to stop"
    echo ""
    
    while true; do
        clear
        echo "==============================================="
        echo "    VOICE FERRY SYSTEM MONITOR"
        echo "    $(date)"
        echo "==============================================="
        echo ""
        
        # Service health checks
        print_header "Service Health"
        check_service_health "Web UI" "$WEB_UI_ENDPOINT" "http"
        check_service_health "B2BUA gRPC" "$B2BUA_GRPC_ENDPOINT" "grpc"
        check_service_health "Redis" "$REDIS_ENDPOINT" "redis"
        check_service_health "etcd" "$ETCD_ENDPOINT" "tcp"
        echo ""
        
        # Get metrics
        get_system_metrics
        echo ""
        get_b2bua_metrics
        echo ""
        get_webui_metrics
        echo ""
        get_redis_metrics
        echo ""
        get_etcd_metrics
        echo ""
        
        echo "==============================================="
        echo "Next update in ${MONITORING_INTERVAL} seconds..."
        sleep $MONITORING_INTERVAL
    done
}

# Function to run single check
single_check() {
    echo "==============================================="
    echo "    VOICE FERRY SYSTEM STATUS CHECK"
    echo "    $(date)"
    echo "==============================================="
    echo ""
    
    # Service health checks
    print_header "Service Health Check"
    check_service_health "Web UI" "$WEB_UI_ENDPOINT" "http"
    check_service_health "B2BUA gRPC" "$B2BUA_GRPC_ENDPOINT" "grpc"
    check_service_health "Redis" "$REDIS_ENDPOINT" "redis"
    check_service_health "etcd" "$ETCD_ENDPOINT" "tcp"
    echo ""
    
    print_header "Quick Metrics"
    get_system_metrics
    echo ""
    
    echo "==============================================="
    print_status "Status check complete"
}

# Function to show usage
show_usage() {
    echo "Voice Ferry System Monitor"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  monitor     - Start continuous monitoring (default)"
    echo "  check       - Run single status check"
    echo "  system      - Show system metrics only"
    echo "  b2bua       - Show B2BUA metrics only"
    echo "  webui       - Show Web UI metrics only"
    echo "  redis       - Show Redis metrics only"
    echo "  etcd        - Show etcd metrics only"
    echo ""
    echo "Environment Variables:"
    echo "  MONITORING_INTERVAL - Monitoring interval in seconds (default: 5)"
    echo "  B2BUA_GRPC_ENDPOINT - B2BUA gRPC endpoint (default: localhost:50051)"
    echo "  WEB_UI_ENDPOINT     - Web UI endpoint (default: http://localhost:3000)"
    echo "  REDIS_ENDPOINT      - Redis endpoint (default: localhost:6379)"
    echo "  ETCD_ENDPOINT       - etcd endpoint (default: localhost:2379)"
}

# Main execution
case "${1:-monitor}" in
    monitor)
        continuous_monitoring
        ;;
    check)
        single_check
        ;;
    system)
        get_system_metrics
        ;;
    b2bua)
        get_b2bua_metrics
        ;;
    webui)
        get_webui_metrics
        ;;
    redis)
        get_redis_metrics
        ;;
    etcd)
        get_etcd_metrics
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
