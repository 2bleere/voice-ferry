# Voice Ferry Helm Chart

A comprehensive Helm chart for deploying the Voice Ferry VoIP switching platform on Kubernetes with Redis cluster support, monitoring, and high availability.

## Overview

This Helm chart deploys Voice Ferry, a high-performance VoIP switching platform, along with its dependencies including Redis cluster, etcd, and RTPEngine. The chart is designed for production use with security, monitoring, and scalability built-in.

## Features

- **Multi-platform container support** (linux/amd64, linux/arm64)
- **Redis cluster deployment** with automatic initialization
- **High availability** with horizontal pod autoscaling
- **Security hardening** with RBAC, network policies, and security contexts
- **Observability** with Prometheus metrics and health checks
- **Multi-environment support** (development, staging, production)
- **TLS/SSL termination** with cert-manager integration
- **Load balancing** with ingress controller support

## Prerequisites

- Kubernetes 1.20+
- Helm 3.8+
- Ingress controller (nginx recommended)
- Persistent volume provisioner
- cert-manager (for TLS certificates)
- Prometheus operator (for monitoring)

**External Dependencies (install separately):**
- Redis cluster for session storage
- etcd cluster for configuration management  
- Prometheus for metrics collection
- Grafana for metrics visualization

> 📖 **See [DEPENDENCIES.md](DEPENDENCIES.md) for detailed installation instructions for external dependencies.**

## Quick Start

### 1. Install Dependencies

First, install the required external dependencies. See [DEPENDENCIES.md](DEPENDENCIES.md) for detailed instructions.

```bash
# Quick dependency installation
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Redis
helm install redis bitnami/redis --namespace voice-ferry --create-namespace

# Install etcd  
helm install etcd bitnami/etcd --namespace voice-ferry

# Install Prometheus stack
helm install prometheus prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace
```

### 2. Install Voice Ferry

```bash
```bash
# Create namespace
kubectl create namespace voice-ferry

# Install with development configuration
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --values helm/voice-ferry/values-dev.yaml
```

### 3. Install for Production

```bash
# Install with production configuration
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry-prod \
  --values helm/voice-ferry/values-prod.yaml
```

## Configuration

### Values Files

The chart includes several pre-configured values files:

- `values.yaml` - Default values with sensible defaults
- `values-dev.yaml` - Development environment (minimal resources, debug enabled)
- `values-staging.yaml` - Staging environment (production-like with reduced resources)
- `values-prod.yaml` - Production environment (high availability, security hardened)

### Key Configuration Options

#### Application Configuration

```yaml
app:
  name: "voice-ferry"
  image:
    repository: "voice-ferry/voice-ferry"
    tag: "v1.0.0"
    pullPolicy: "IfNotPresent"
  replicas: 3
  resources:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "2000m"
      memory: "2Gi"
```

#### Redis Cluster Configuration

```yaml
redis:
  enabled: true
  cluster:
    enabled: true
    nodes: 6
    replicas: 1
  auth:
    enabled: true
    password: "your-redis-password"
  persistence:
    enabled: true
    size: "20Gi"
```

#### Ingress Configuration

```yaml
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: "voice-ferry.example.com"
      paths:
        - path: "/"
          pathType: "Prefix"
          port: 8080
  tls:
    - secretName: "voice-ferry-tls"
      hosts:
        - "voice-ferry.example.com"
```

#### Monitoring Configuration

```yaml
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    namespace: "monitoring"
    interval: "30s"
```

## Advanced Configuration

### Custom Configuration File

Create a custom values file for your environment:

```yaml
# custom-values.yaml
app:
  replicas: 5
  resources:
    requests:
      cpu: "1000m"
      memory: "1Gi"

config:
  sip:
    listen_address: "0.0.0.0:5060"
    session_expires: 3600
  
  redis:
    endpoints:
      - "my-redis-cluster:6379"
    password: "my-secure-password"

ingress:
  hosts:
    - host: "my-voice-ferry.company.com"
```

Install with custom values:

```bash
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --values custom-values.yaml
```

### Environment-Specific Overrides

You can combine multiple values files:

```bash
# Production with custom overrides
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry-prod \
  --values helm/voice-ferry/values-prod.yaml \
  --values custom-prod-overrides.yaml
```

## Operations

### Upgrade

```bash
# Upgrade to new version
helm upgrade voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --values helm/voice-ferry/values-prod.yaml
```

### Rollback

```bash
# View release history
helm history voice-ferry --namespace voice-ferry

# Rollback to previous version
helm rollback voice-ferry 1 --namespace voice-ferry
```

### Uninstall

```bash
# Uninstall release (keeps PVCs)
helm uninstall voice-ferry --namespace voice-ferry

# Clean up persistent volumes if needed
kubectl delete pvc -l app.kubernetes.io/instance=voice-ferry -n voice-ferry
```

### Scaling

```bash
# Scale application manually
kubectl scale deployment voice-ferry \
  --replicas=5 \
  --namespace voice-ferry

# Or update values and upgrade
helm upgrade voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --set app.replicas=5
```

## Monitoring and Observability

### Health Checks

The application provides several health check endpoints:

- `/health` - General health check
- `/ready` - Readiness probe
- `/live` - Liveness probe

### Metrics

Prometheus metrics are exposed on port 9090 at `/metrics`:

- `voice_ferry_calls_total` - Total number of calls
- `voice_ferry_active_calls` - Current active calls
- `voice_ferry_call_duration_seconds` - Call duration histogram
- `voice_ferry_sip_requests_total` - SIP request counters
- `voice_ferry_redis_operations_total` - Redis operation counters

### Grafana Dashboard

Import the provided Grafana dashboard from `monitoring/grafana-dashboard.json` to visualize metrics.

### Logs

View application logs:

```bash
# View all pod logs
kubectl logs -l app.kubernetes.io/name=voice-ferry -n voice-ferry

# Follow logs from specific pod
kubectl logs -f voice-ferry-deployment-xxx -n voice-ferry

# View logs from all containers in deployment
kubectl logs -f deployment/voice-ferry -n voice-ferry --all-containers
```

## Testing

### Helm Tests

Run built-in Helm tests:

```bash
# Run connection tests
helm test voice-ferry --namespace voice-ferry

# View test results
kubectl logs voice-ferry-test-connection -n voice-ferry
```

### Manual Testing

Test SIP connectivity:

```bash
# Port forward SIP port
kubectl port-forward svc/voice-ferry 5060:5060 -n voice-ferry

# Test with SIP client (e.g., SIPp)
sipp -sn uac localhost:5060
```

Test API endpoints:

```bash
# Port forward API port
kubectl port-forward svc/voice-ferry 8080:8080 -n voice-ferry

# Test health endpoint
curl http://localhost:8080/health

# Test metrics endpoint
curl http://localhost:9090/metrics
```

## Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n voice-ferry

# Describe problematic pod
kubectl describe pod voice-ferry-xxx -n voice-ferry

# Check events
kubectl get events -n voice-ferry --sort-by='.lastTimestamp'
```

#### Redis Cluster Issues

```bash
# Check Redis cluster status
kubectl exec -it voice-ferry-redis-0 -n voice-ferry -- redis-cli cluster nodes

# Check Redis logs
kubectl logs voice-ferry-redis-0 -n voice-ferry
```

#### Ingress Issues

```bash
# Check ingress status
kubectl get ingress -n voice-ferry

# Check ingress controller logs
kubectl logs -l app.kubernetes.io/name=ingress-nginx -n ingress-nginx
```

#### Certificate Issues

```bash
# Check certificate status
kubectl get certificates -n voice-ferry

# Check cert-manager logs
kubectl logs -l app=cert-manager -n cert-manager
```

### Debug Mode

Enable debug mode for troubleshooting:

```bash
helm upgrade voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --set global.debug=true \
  --set app.env.LOG_LEVEL=DEBUG
```

## Security

### RBAC

The chart creates a service account with minimal required permissions:

- Get/list/watch pods and services
- Create/update/patch configmaps
- Access to metrics endpoints

### Network Policies

Network policies restrict traffic between pods:

- Only allow ingress traffic from ingress controller
- Only allow monitoring traffic from monitoring namespace
- Allow inter-pod communication within the application

### Security Context

Pods run with security hardening:

- Non-root user (UID 1000)
- Read-only root filesystem
- Dropped capabilities
- No privilege escalation

### TLS/SSL

Configure TLS certificates:

```yaml
tls:
  enabled: true
  secretName: "voice-ferry-tls-certs"
  # Provide certificate and key data
```

## Performance Tuning

### Resource Optimization

Adjust resources based on load:

```yaml
app:
  resources:
    requests:
      cpu: "1000m"
      memory: "1Gi"
    limits:
      cpu: "4000m"
      memory: "4Gi"
```

### Redis Tuning

Optimize Redis cluster:

```yaml
redis:
  cluster:
    nodes: 6  # 3 masters + 3 replicas
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
```

### JVM Tuning (if applicable)

Add JVM parameters:

```yaml
app:
  env:
    JAVA_OPTS: "-Xms1g -Xmx2g -XX:+UseG1GC"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes to the chart
4. Test with different values files
5. Update documentation
6. Submit a pull request

## License

This Helm chart is licensed under the MIT License. See [LICENSE](../../LICENSE) for details.

## Support

For support and questions:

- GitHub Issues: https://github.com/voice-ferry/voice-ferry/issues
- Documentation: https://docs.voice-ferry.example.com
- Community: https://community.voice-ferry.example.com
