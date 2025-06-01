#!/bin/bash
# Voice Ferry Deployment Validation Script
# This script validates that all dependencies are properly deployed and healthy

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="voice-ferry"
TIMEOUT=300

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

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    log_info "kubectl is available"
}

# Check if namespace exists
check_namespace() {
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace '$NAMESPACE' does not exist"
        exit 1
    fi
    log_info "Namespace '$NAMESPACE' exists"
}

# Check if all required pods are running
check_pods() {
    log_info "Checking pod status..."
    
    # Expected pods
    local expected_pods=(
        "etcd-0"
        "etcd-1" 
        "etcd-2"
        "redis-cluster-0"
        "redis-cluster-1"
        "redis-cluster-2"
        "redis-cluster-3"
        "redis-cluster-4"
        "redis-cluster-5"
    )
    
    # Check each expected pod
    for pod in "${expected_pods[@]}"; do
        if kubectl get pod "$pod" -n "$NAMESPACE" &> /dev/null; then
            local status=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
            if [[ "$status" == "Running" ]]; then
                log_info "Pod $pod is running"
            else
                log_warn "Pod $pod is in state: $status"
            fi
        else
            log_warn "Pod $pod does not exist"
        fi
    done
    
    # Check SIP B2BUA deployment
    local b2bua_ready=$(kubectl get deployment sip-b2bua -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local b2bua_desired=$(kubectl get deployment sip-b2bua -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    
    if [[ "$b2bua_ready" -eq "$b2bua_desired" ]] && [[ "$b2bua_ready" -gt 0 ]]; then
        log_info "SIP B2BUA deployment is ready ($b2bua_ready/$b2bua_desired)"
    else
        log_warn "SIP B2BUA deployment is not ready ($b2bua_ready/$b2bua_desired)"
    fi
}

# Check etcd cluster health
check_etcd() {
    log_info "Checking etcd cluster health..."
    
    if kubectl exec etcd-0 -n "$NAMESPACE" -- etcdctl --endpoints=http://etcd:2379 endpoint health &> /dev/null; then
        log_info "etcd cluster is healthy"
        
        # Check cluster members
        local members=$(kubectl exec etcd-0 -n "$NAMESPACE" -- etcdctl --endpoints=http://etcd:2379 member list 2>/dev/null | wc -l)
        log_info "etcd cluster has $members members"
    else
        log_error "etcd cluster health check failed"
        return 1
    fi
}

# Check Redis cluster health  
check_redis() {
    log_info "Checking Redis cluster health..."
    
    if kubectl exec redis-cluster-0 -n "$NAMESPACE" -- redis-cli ping &> /dev/null; then
        log_info "Redis cluster is responding"
        
        # Check cluster status
        local cluster_state=$(kubectl exec redis-cluster-0 -n "$NAMESPACE" -- redis-cli cluster info 2>/dev/null | grep cluster_state | cut -d: -f2 | tr -d '\r')
        if [[ "$cluster_state" == "ok" ]]; then
            log_info "Redis cluster state is OK"
            
            # Count cluster nodes
            local nodes=$(kubectl exec redis-cluster-0 -n "$NAMESPACE" -- redis-cli cluster nodes 2>/dev/null | wc -l)
            log_info "Redis cluster has $nodes nodes"
        else
            log_warn "Redis cluster state: $cluster_state"
        fi
    else
        log_error "Redis cluster health check failed"
        return 1
    fi
}

# Check RTPEngine connectivity
check_rtpengine() {
    log_info "Checking RTPEngine connectivity..."
    
    # Try to connect to RTPEngine port
    if kubectl run rtpengine-test --image=busybox:1.35 --rm -i --restart=Never -n "$NAMESPACE" -- nc -z rtpengine 22222 &> /dev/null; then
        log_info "RTPEngine is reachable on port 22222"
    else
        log_warn "RTPEngine connectivity test failed"
    fi
}

# Check Web UI deployment and etcd monitoring
check_web_ui() {
    log_info "Checking Web UI deployment and etcd monitoring..."
    
    # Check Web UI deployment
    local ui_ready=$(kubectl get deployment voice-ferry-web-ui -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local ui_desired=$(kubectl get deployment voice-ferry-web-ui -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    
    if [[ "$ui_ready" -eq "$ui_desired" ]] && [[ "$ui_ready" -gt 0 ]]; then
        log_info "Web UI deployment is ready ($ui_ready/$ui_desired)"
        
        # Get a Web UI pod for testing
        local ui_pod=$(kubectl get pods -n "$NAMESPACE" -l app=voice-ferry-web-ui --no-headers 2>/dev/null | head -1 | awk '{print $1}')
        
        if [[ -n "$ui_pod" ]]; then
            # Check Web UI health endpoint
            if kubectl exec "$ui_pod" -n "$NAMESPACE" -- curl -sf http://localhost:8080/health &> /dev/null; then
                log_info "Web UI health endpoint is accessible"
            else
                log_warn "Web UI health endpoint check failed"
            fi
            
            # Check etcd monitoring endpoint
            if kubectl exec "$ui_pod" -n "$NAMESPACE" -- curl -sf http://localhost:8080/api/monitoring &> /dev/null; then
                log_info "Web UI monitoring API is accessible"
                
                # Test etcd monitoring data
                local etcd_status=$(kubectl exec "$ui_pod" -n "$NAMESPACE" -- curl -s http://localhost:8080/api/monitoring 2>/dev/null | grep -o '"etcd":[^}]*}' || echo "")
                if [[ -n "$etcd_status" ]]; then
                    log_info "etcd status data is available in monitoring API"
                else
                    log_warn "etcd status data not found in monitoring API"
                fi
            else
                log_warn "Web UI monitoring API check failed"
            fi
            
            # Check WebSocket endpoint
            if kubectl exec "$ui_pod" -n "$NAMESPACE" -- nc -z localhost 8080 &> /dev/null; then
                log_info "Web UI WebSocket port is accessible"
            else
                log_warn "Web UI WebSocket connectivity test failed"
            fi
            
            # Check environment variables for etcd configuration
            local etcd_endpoints=$(kubectl exec "$ui_pod" -n "$NAMESPACE" -- printenv ETCD_ENDPOINTS 2>/dev/null || echo "")
            if [[ -n "$etcd_endpoints" ]]; then
                log_info "etcd endpoints configured: $etcd_endpoints"
            else
                log_warn "ETCD_ENDPOINTS environment variable not set"
            fi
            
        else
            log_warn "No Web UI pods found for detailed checking"
        fi
    else
        log_warn "Web UI deployment is not ready ($ui_ready/$ui_desired)"
    fi
    
    # Check Web UI service
    if kubectl get service voice-ferry-web-ui -n "$NAMESPACE" &> /dev/null; then
        local ui_cluster_ip=$(kubectl get service voice-ferry-web-ui -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
        local ui_ports=$(kubectl get service voice-ferry-web-ui -n "$NAMESPACE" -o jsonpath='{.spec.ports[*].port}')
        log_info "Web UI service exists: $ui_cluster_ip (ports: $ui_ports)"
    else
        log_warn "Web UI service does not exist"
    fi
}

# Check services
check_services() {
    log_info "Checking services..."
    
    local services=("etcd" "redis" "rtpengine" "sip-b2bua" "voice-ferry-web-ui")
    
    for service in "${services[@]}"; do
        if kubectl get service "$service" -n "$NAMESPACE" &> /dev/null; then
            local cluster_ip=$(kubectl get service "$service" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
            local ports=$(kubectl get service "$service" -n "$NAMESPACE" -o jsonpath='{.spec.ports[*].port}')
            log_info "Service $service exists: $cluster_ip (ports: $ports)"
        else
            log_warn "Service $service does not exist"
        fi
    done
}

# Check persistent volumes
check_storage() {
    log_info "Checking persistent storage..."
    
    local pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    log_info "Found $pvcs persistent volume claims"
    
    # Check if any PVCs are pending
    local pending=$(kubectl get pvc -n "$NAMESPACE" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
    if [[ "$pending" -gt 0 ]]; then
        log_warn "$pending PVCs are in Pending state"
        kubectl get pvc -n "$NAMESPACE" --field-selector=status.phase=Pending
    else
        log_info "All PVCs are bound"
    fi
}

# Check application health endpoints
check_application_health() {
    log_info "Checking application health endpoints..."
    
    # Get a B2BUA pod name
    local pod=$(kubectl get pods -n "$NAMESPACE" -l app=sip-b2bua --no-headers 2>/dev/null | head -1 | awk '{print $1}')
    
    if [[ -n "$pod" ]]; then
        # Check liveness endpoint
        if kubectl exec "$pod" -n "$NAMESPACE" -- curl -sf http://localhost:8080/health/live &> /dev/null; then
            log_info "Application liveness check passed"
        else
            log_warn "Application liveness check failed"
        fi
        
        # Check readiness endpoint  
        if kubectl exec "$pod" -n "$NAMESPACE" -- curl -sf http://localhost:8080/health/ready &> /dev/null; then
            log_info "Application readiness check passed"
        else
            log_warn "Application readiness check failed"
        fi
        
        # Check metrics endpoint
        if kubectl exec "$pod" -n "$NAMESPACE" -- curl -sf http://localhost:8080/metrics &> /dev/null; then
            log_info "Metrics endpoint is accessible"
        else
            log_warn "Metrics endpoint check failed"
        fi
    else
        log_warn "No B2BUA pods found for health checking"
    fi
}

# Check resource usage
check_resources() {
    log_info "Checking resource usage..."
    
    # Check if metrics-server is available
    if kubectl top nodes &> /dev/null; then
        log_info "Node resource usage:"
        kubectl top nodes
        
        log_info "Pod resource usage in $NAMESPACE:"
        kubectl top pods -n "$NAMESPACE" 2>/dev/null || log_warn "Could not get pod metrics"
    else
        log_warn "metrics-server not available, skipping resource usage check"
    fi
}

# Check network connectivity between components
check_network() {
    log_info "Checking network connectivity..."
    
    # Get a B2BUA pod for network testing
    local pod=$(kubectl get pods -n "$NAMESPACE" -l app=sip-b2bua --no-headers 2>/dev/null | head -1 | awk '{print $1}')
    
    if [[ -n "$pod" ]]; then
        # Test Redis connectivity
        if kubectl exec "$pod" -n "$NAMESPACE" -- nc -z redis 6379 &> /dev/null; then
            log_info "Network connectivity to Redis: OK"
        else
            log_warn "Network connectivity to Redis: Failed"
        fi
        
        # Test etcd connectivity
        if kubectl exec "$pod" -n "$NAMESPACE" -- nc -z etcd 2379 &> /dev/null; then
            log_info "Network connectivity to etcd: OK"
        else
            log_warn "Network connectivity to etcd: Failed"
        fi
        
        # Test RTPEngine connectivity
        if kubectl exec "$pod" -n "$NAMESPACE" -- nc -z rtpengine 22222 &> /dev/null; then
            log_info "Network connectivity to RTPEngine: OK"
        else
            log_warn "Network connectivity to RTPEngine: Failed"
        fi
        
        # Test Web UI connectivity
        if kubectl exec "$pod" -n "$NAMESPACE" -- nc -z voice-ferry-web-ui 8080 &> /dev/null; then
            log_info "Network connectivity to Web UI: OK"
        else
            log_warn "Network connectivity to Web UI: Failed"
        fi
    else
        log_warn "No B2BUA pods available for network testing"
    fi
}

# Print summary
print_summary() {
    echo
    log_info "=== Deployment Validation Summary ==="
    echo
    
    # Count pods by status
    local total_pods=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    local running_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    local pending_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
    local failed_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Failed --no-headers 2>/dev/null | wc -l)
    
    echo "Pods Status:"
    echo "  Total: $total_pods"
    echo "  Running: $running_pods"
    echo "  Pending: $pending_pods" 
    echo "  Failed: $failed_pods"
    echo
    
    # Services
    local services=$(kubectl get services -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    echo "Services: $services"
    
    # Storage
    local pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    echo "Persistent Volume Claims: $pvcs"
    
    echo
    if [[ "$running_pods" -eq "$total_pods" ]] && [[ "$total_pods" -gt 0 ]] && [[ "$failed_pods" -eq 0 ]]; then
        log_info "✅ Deployment appears healthy!"
    elif [[ "$pending_pods" -gt 0 ]]; then
        log_warn "⚠️  Some pods are still pending"
    else
        log_error "❌ Deployment has issues that need attention"
    fi
}

# Main execution
main() {
    echo "Voice Ferry Deployment Validation"
    echo "================================="
    echo
    
    check_kubectl
    check_namespace
    check_pods
    check_services
    check_storage
    
    # Health checks (with error handling)
    check_etcd || true
    check_redis || true  
    check_rtpengine || true
    check_web_ui || true
    check_application_health || true
    check_network || true
    check_resources || true
    
    print_summary
}

# Handle script arguments
case "${1:-validate}" in
    "validate"|"check"|"")
        main
        ;;
    "wait")
        log_info "Waiting for deployment to be ready..."
        kubectl wait --for=condition=Ready pod -l app=etcd -n "$NAMESPACE" --timeout="${TIMEOUT}s"
        kubectl wait --for=condition=Ready pod -l app=redis-cluster -n "$NAMESPACE" --timeout="${TIMEOUT}s"
        kubectl wait --for=condition=Available deployment/sip-b2bua -n "$NAMESPACE" --timeout="${TIMEOUT}s"
        kubectl wait --for=condition=Available deployment/voice-ferry-web-ui -n "$NAMESPACE" --timeout="${TIMEOUT}s"
        log_info "All components are ready!"
        main
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [validate|wait|help]"
        echo "  validate (default): Run validation checks"
        echo "  wait: Wait for deployment to be ready, then validate"
        echo "  help: Show this help message"
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
