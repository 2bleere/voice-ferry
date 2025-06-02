#!/bin/bash
# Voice Ferry ARM64 Production Deployment Validator
# Validates ARM-specific deployment configurations and health

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="voice-ferry"
TIMEOUT=300
HEALTH_CHECK_RETRIES=5
DEPLOYMENT_FILE="arm-production-complete.yaml"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if cluster has ARM64 nodes
    ARM_NODES=$(kubectl get nodes -o jsonpath='{.items[*].status.nodeInfo.architecture}' | grep -c "arm64" || echo "0")
    if [ "$ARM_NODES" -eq 0 ]; then
        log_error "No ARM64 nodes found in cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed ($ARM_NODES ARM64 nodes found)"
}

# Validate ARM64 node configuration
validate_arm_nodes() {
    log_info "Validating ARM64 node configuration..."
    
    # Check node labels
    UNLABELED_NODES=$(kubectl get nodes -l "!kubernetes.io/arch" --no-headers | wc -l)
    if [ "$UNLABELED_NODES" -gt 0 ]; then
        log_warning "$UNLABELED_NODES nodes missing architecture label"
    fi
    
    # Check node resources
    kubectl get nodes -o custom-columns="NAME:.metadata.name,ARCH:.status.nodeInfo.architecture,CPU:.status.capacity.cpu,MEMORY:.status.capacity.memory" | while read -r line; do
        if [[ $line == *"arm64"* ]]; then
            log_info "ARM64 Node: $line"
        fi
    done
    
    log_success "ARM64 node validation completed"
}

# Check deployment file
check_deployment_file() {
    log_info "Checking deployment file..."
    
    if [ ! -f "$DEPLOYMENT_FILE" ]; then
        log_error "Deployment file $DEPLOYMENT_FILE not found"
        exit 1
    fi
    
    # Validate YAML syntax
    if ! kubectl apply --dry-run=client -f "$DEPLOYMENT_FILE" &> /dev/null; then
        log_error "Invalid YAML syntax in $DEPLOYMENT_FILE"
        exit 1
    fi
    
    # Check for ARM-specific configurations
    ARM_CONFIGS=$(grep -c "kubernetes.io/arch: arm64" "$DEPLOYMENT_FILE" || echo "0")
    if [ "$ARM_CONFIGS" -eq 0 ]; then
        log_warning "No ARM64 node selectors found in deployment"
    else
        log_success "Found $ARM_CONFIGS ARM64 node selector configurations"
    fi
    
    log_success "Deployment file validation passed"
}

# Deploy and monitor
deploy_and_monitor() {
    log_info "Deploying Voice Ferry ARM64 production stack..."
    
    # Apply deployment
    kubectl apply -f "$DEPLOYMENT_FILE"
    
    # Wait for namespace
    kubectl wait --for=condition=Ready namespace/"$NAMESPACE" --timeout=60s
    
    log_info "Waiting for deployments to be ready..."
    
    # Monitor deployment progress
    DEPLOYMENTS=(
        "etcd"
        "redis-cluster" 
        "voice-ferry"
        "voice-ferry-web-ui"
    )
    
    for deployment in "${DEPLOYMENTS[@]}"; do
        log_info "Waiting for $deployment to be ready..."
        
        if [[ $deployment == "etcd" || $deployment == "redis-cluster" ]]; then
            # For StatefulSets
            kubectl wait --for=condition=Ready statefulset/"$deployment" -n "$NAMESPACE" --timeout="${TIMEOUT}s" || {
                log_error "StatefulSet $deployment failed to become ready"
                kubectl describe statefulset/"$deployment" -n "$NAMESPACE"
                exit 1
            }
        else
            # For Deployments
            kubectl wait --for=condition=Available deployment/"$deployment" -n "$NAMESPACE" --timeout="${TIMEOUT}s" || {
                log_error "Deployment $deployment failed to become ready"
                kubectl describe deployment/"$deployment" -n "$NAMESPACE"
                exit 1
            }
        fi
        
        log_success "$deployment is ready"
    done
}

# Validate ARM pod scheduling
validate_arm_scheduling() {
    log_info "Validating ARM64 pod scheduling..."
    
    # Check that pods are scheduled on ARM64 nodes
    PODS=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    
    for pod in $PODS; do
        NODE=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.nodeName}')
        ARCH=$(kubectl get node "$NODE" -o jsonpath='{.status.nodeInfo.architecture}')
        
        if [ "$ARCH" = "arm64" ]; then
            log_success "Pod $pod scheduled on ARM64 node $NODE"
        else
            log_error "Pod $pod scheduled on non-ARM64 node $NODE ($ARCH)"
        fi
    done
}

# Health checks
perform_health_checks() {
    log_info "Performing health checks..."
    
    # Check pod status
    FAILED_PODS=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running --no-headers | wc -l)
    if [ "$FAILED_PODS" -gt 0 ]; then
        log_error "$FAILED_PODS pods are not running"
        kubectl get pods -n "$NAMESPACE"
        exit 1
    fi
    
    # Check service endpoints
    SERVICES=(
        "voice-ferry-sip"
        "voice-ferry-grpc" 
        "voice-ferry-metrics"
        "voice-ferry-web-ui"
        "redis"
        "etcd"
    )
    
    for service in "${SERVICES[@]}"; do
        ENDPOINTS=$(kubectl get endpoints "$service" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
        if [ "$ENDPOINTS" -eq 0 ]; then
            log_error "Service $service has no endpoints"
        else
            log_success "Service $service has $ENDPOINTS endpoint(s)"
        fi
    done
    
    # Test HTTP health endpoints
    test_health_endpoints
}

# Test HTTP health endpoints
test_health_endpoints() {
    log_info "Testing HTTP health endpoints..."
    
    # Port-forward and test B2BUA health
    kubectl port-forward -n "$NAMESPACE" service/voice-ferry-metrics 8080:8080 &
    PF_PID=$!
    sleep 5
    
    for i in $(seq 1 $HEALTH_CHECK_RETRIES); do
        if curl -s http://localhost:8080/healthz/live > /dev/null; then
            log_success "B2BUA health endpoint responding"
            break
        else
            log_warning "B2BUA health check attempt $i/$HEALTH_CHECK_RETRIES failed"
            sleep 5
        fi
    done
    
    kill $PF_PID 2>/dev/null || true
    
    # Port-forward and test Web UI health
    kubectl port-forward -n "$NAMESPACE" service/voice-ferry-web-ui 3000:3000 &
    PF_PID=$!
    sleep 5
    
    for i in $(seq 1 $HEALTH_CHECK_RETRIES); do
        if curl -s http://localhost:3000/health > /dev/null; then
            log_success "Web UI health endpoint responding"
            break
        else
            log_warning "Web UI health check attempt $i/$HEALTH_CHECK_RETRIES failed"
            sleep 5
        fi
    done
    
    kill $PF_PID 2>/dev/null || true
}

# Performance validation
validate_performance() {
    log_info "Validating ARM64 performance characteristics..."
    
    # Check resource usage
    log_info "Current resource usage:"
    kubectl top pods -n "$NAMESPACE" 2>/dev/null || log_warning "Metrics server not available"
    
    # Check HPA status
    HPA_COUNT=$(kubectl get hpa -n "$NAMESPACE" --no-headers | wc -l)
    if [ "$HPA_COUNT" -gt 0 ]; then
        log_success "Found $HPA_COUNT HPA configurations"
        kubectl get hpa -n "$NAMESPACE"
    else
        log_warning "No HPA configurations found"
    fi
    
    # Check PDB status
    PDB_COUNT=$(kubectl get pdb -n "$NAMESPACE" --no-headers | wc -l)
    if [ "$PDB_COUNT" -gt 0 ]; then
        log_success "Found $PDB_COUNT PDB configurations"
        kubectl get pdb -n "$NAMESPACE"
    else
        log_warning "No PDB configurations found"
    fi
}

# Network connectivity tests
test_network_connectivity() {
    log_info "Testing network connectivity..."
    
    # Test Redis connectivity
    REDIS_POD=$(kubectl get pods -n "$NAMESPACE" -l app=redis-cluster -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$REDIS_POD" -- redis-cli ping | grep -q PONG; then
        log_success "Redis connectivity test passed"
    else
        log_error "Redis connectivity test failed"
    fi
    
    # Test etcd connectivity
    ETCD_POD=$(kubectl get pods -n "$NAMESPACE" -l app=etcd -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$ETCD_POD" -- etcdctl endpoint health | grep -q "is healthy"; then
        log_success "etcd connectivity test passed"
    else
        log_error "etcd connectivity test failed"
    fi
    
    # Test internal service resolution
    B2BUA_POD=$(kubectl get pods -n "$NAMESPACE" -l app=voice-ferry -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$B2BUA_POD" -- nslookup redis.voice-ferry.svc.cluster.local > /dev/null 2>&1; then
        log_success "Internal DNS resolution test passed"
    else
        log_warning "Internal DNS resolution test failed (may be normal if nslookup not available)"
    fi
}

# Security validation
validate_security() {
    log_info "Validating security configuration..."
    
    # Check for non-root containers
    NONROOT_COUNT=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].spec.securityContext.runAsNonRoot}' | grep -c true || echo "0")
    TOTAL_PODS=$(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
    
    if [ "$NONROOT_COUNT" -eq "$TOTAL_PODS" ]; then
        log_success "All pods configured to run as non-root"
    else
        log_warning "Some pods may be running as root"
    fi
    
    # Check for read-only root filesystems
    READONLY_COUNT=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].spec.containers[*].securityContext.readOnlyRootFilesystem}' | grep -c true || echo "0")
    if [ "$READONLY_COUNT" -gt 0 ]; then
        log_success "$READONLY_COUNT containers have read-only root filesystem"
    fi
    
    # Check network policies
    NP_COUNT=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers | wc -l)
    if [ "$NP_COUNT" -gt 0 ]; then
        log_success "Found $NP_COUNT network policies"
    else
        log_warning "No network policies configured"
    fi
}

# Generate deployment report
generate_report() {
    log_info "Generating deployment report..."
    
    REPORT_FILE="voice-ferry-arm64-deployment-report-$(date +%Y%m%d-%H%M%S).txt"
    
    {
        echo "Voice Ferry ARM64 Deployment Report"
        echo "Generated: $(date)"
        echo "Cluster: $(kubectl config current-context)"
        echo "Namespace: $NAMESPACE"
        echo ""
        
        echo "=== CLUSTER INFO ==="
        kubectl cluster-info
        echo ""
        
        echo "=== ARM64 NODES ==="
        kubectl get nodes -o wide | grep arm64
        echo ""
        
        echo "=== PODS STATUS ==="
        kubectl get pods -n "$NAMESPACE" -o wide
        echo ""
        
        echo "=== SERVICES ==="
        kubectl get svc -n "$NAMESPACE"
        echo ""
        
        echo "=== INGRESS ==="
        kubectl get ingress -n "$NAMESPACE" 2>/dev/null || echo "No ingress found"
        echo ""
        
        echo "=== STORAGE ==="
        kubectl get pvc -n "$NAMESPACE"
        echo ""
        
        echo "=== RESOURCE USAGE ==="
        kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Metrics server not available"
        echo ""
        
        echo "=== HPA STATUS ==="
        kubectl get hpa -n "$NAMESPACE" 2>/dev/null || echo "No HPA found"
        echo ""
        
        echo "=== EVENTS ==="
        kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' | tail -20
        
    } > "$REPORT_FILE"
    
    log_success "Deployment report saved to $REPORT_FILE"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    # Kill any remaining port-forward processes
    pkill -f "kubectl port-forward" 2>/dev/null || true
}

# Main execution
main() {
    log_info "Starting Voice Ferry ARM64 deployment validation..."
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Run validation steps
    check_prerequisites
    validate_arm_nodes
    check_deployment_file
    deploy_and_monitor
    validate_arm_scheduling
    perform_health_checks
    validate_performance
    test_network_connectivity
    validate_security
    generate_report
    
    log_success "Voice Ferry ARM64 deployment validation completed successfully!"
    log_info "Access the Web UI by running: kubectl port-forward -n $NAMESPACE service/voice-ferry-web-ui 3000:3000"
    log_info "Then visit: http://localhost:3000"
}

# Script options
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "health")
        perform_health_checks
        ;;
    "report")
        generate_report
        ;;
    "cleanup")
        log_info "Removing Voice Ferry deployment..."
        kubectl delete -f "$DEPLOYMENT_FILE" --ignore-not-found=true
        log_success "Cleanup completed"
        ;;
    *)
        echo "Usage: $0 [deploy|health|report|cleanup]"
        echo "  deploy  - Full deployment and validation (default)"
        echo "  health  - Run health checks only"
        echo "  report  - Generate deployment report only"
        echo "  cleanup - Remove deployment"
        exit 1
        ;;
esac
