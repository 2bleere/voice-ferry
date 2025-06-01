# Voice Ferry Deployment Guide

This guide provides comprehensive instructions for deploying Voice Ferry SIP B2BUA in various environments, including the latest Redis cluster integration and automated dependency management.

## Table of Contents

1. [Deployment Options](#deployment-options)
2. [Docker Deployment](#docker-deployment)
3. [Kubernetes Deployment](#kubernetes-deployment)
4. [Redis Configuration](#redis-configuration)
5. [Dependency Management](#dependency-management)
6. [Configuration Management](#configuration-management)
7. [SSL/TLS Setup](#ssltls-setup)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Scaling and High Availability](#scaling-and-high-availability)
10. [Security Considerations](#security-considerations)
11. [Troubleshooting](#troubleshooting)

## Deployment Options

Voice Ferry can be deployed in several ways with enhanced dependency management:

- **Docker Compose** - Simple single-node deployment
- **Kubernetes with Redis Cluster** - Production-ready cluster deployment with HA Redis
- **Kubernetes with Single Redis** - Simpler deployment for development/testing
- **Standalone Binary** - Direct deployment on Linux/macOS
- **Cloud Platforms** - AWS EKS, GCP GKE, Azure AKS with comprehensive health checks

## Docker Deployment

### Production Docker Compose

For a complete production deployment with all dependencies:

```bash
# Clone the repository
git clone https://github.com/2bleere/voice-ferry.git
cd voice-ferry

# Set environment variables
export JWT_SIGNING_KEY="your-secure-jwt-key-here"
export GRAFANA_PASSWORD="your-grafana-password"

# Deploy the stack
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Variables

Required environment variables for production:

```bash
# JWT Configuration
JWT_SIGNING_KEY=your-256-bit-secret-key-here

# Optional: Grafana password
GRAFANA_PASSWORD=secure-password

# Optional: Custom configuration
CONFIG_FILE=/path/to/custom/config.yaml

# Optional: Log level
LOG_LEVEL=info
```

### SSL Certificate Setup

Create SSL certificates for secure SIP and gRPC:

```bash
# Create SSL directory
mkdir -p ssl

# Generate self-signed certificate (for testing)
openssl req -x509 -newkey rsa:4096 -keyout ssl/voice-ferry.key \
    -out ssl/voice-ferry.crt -days 365 -nodes \
    -subj "/CN=voice-ferry.local"

# Or use Let's Encrypt for production
certbot certonly --standalone -d your-domain.com
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem ssl/voice-ferry.crt
cp /etc/letsencrypt/live/your-domain.com/privkey.pem ssl/voice-ferry.key
```

### Docker Build

To build your own Docker image:

```bash
# Build production image
docker build -f deployments/docker/Dockerfile \
    --build-arg VERSION=v1.0.0 \
    --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
    -t voice-ferry:v1.0.0 .

# Tag and push to registry
docker tag voice-ferry:v1.0.0 your-registry/voice-ferry:v1.0.0
docker push your-registry/voice-ferry:v1.0.0
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured
- Helm (optional)
- MetalLB or cloud LoadBalancer

### Production Deployment

```bash
# Deploy dependencies first
kubectl apply -f deployments/kubernetes/dependencies.yaml

# Wait for dependencies to be ready
kubectl wait --for=condition=available --timeout=300s \
    deployment/etcd deployment/redis -n voice-ferry

# Deploy Voice Ferry
kubectl apply -f deployments/kubernetes/voice-ferry-production.yaml

# Verify deployment
kubectl get pods -n voice-ferry
kubectl get services -n voice-ferry
```

### Enhanced Kubernetes Deployment with Redis Cluster

Voice Ferry now supports both single Redis instance and Redis cluster deployments with comprehensive health checking:

```bash
# Option 1: Deploy with single Redis instance (development/testing)
kubectl apply -f deployments/kubernetes/dependencies.yaml

# Option 2: Deploy with Redis cluster (production)
kubectl apply -f deployments/kubernetes/redis-cluster.yaml

# Deploy the main application with dependency health checks
kubectl apply -f deployments/kubernetes/sip-b2bua.yaml

# For production configuration
kubectl apply -f deployments/kubernetes/voice-ferry-production.yaml

# Validate entire deployment
./deployments/kubernetes/validate-deployment.sh
```

## Redis Configuration

Voice Ferry supports flexible Redis deployment options:

### Single Redis Instance

Suitable for development and light production workloads:

- **Resource Requirements**: 512Mi memory, 0.5 CPU
- **Storage**: Persistent volume for data durability
- **High Availability**: No automatic failover

```yaml
# Deployed via dependencies.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
```

### Redis Cluster

Recommended for production environments requiring high availability:

- **Topology**: 6 nodes (3 masters + 3 replicas)
- **Automatic Failover**: Built-in cluster failover
- **Data Sharding**: Automatic key distribution
- **Resource Requirements**: 1Gi memory per node, 0.5 CPU per node

```yaml
# Deployed via redis-cluster.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-cluster
spec:
  replicas: 6
  serviceName: redis
  template:
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command:
          - redis-server
          - /etc/redis/redis.conf
          - --cluster-enabled
          - "yes"
```

## Dependency Management

Voice Ferry implements comprehensive dependency health checking to ensure reliable startup order:

### Init Container Strategy

The main application pod includes init containers that verify dependencies:

```yaml
initContainers:
- name: wait-for-redis
  image: redis:7-alpine
  command: ['sh', '-c', 'until redis-cli -h redis -p 6379 ping; do sleep 2; done']
  
- name: wait-for-etcd
  image: quay.io/coreos/etcd:v3.5.0
  command: ['sh', '-c', 'until etcdctl --endpoints=http://etcd:2379 endpoint health; do sleep 2; done']
  
- name: wait-for-rtpengine
  image: busybox:1.35
  command: ['sh', '-c', 'until nc -z rtpengine 2223; do sleep 2; done']
```

### Health Check Benefits

- **Startup Reliability**: Application starts only when all dependencies are healthy
- **Reduced Downtime**: Prevents connection errors during deployment
- **Production Ready**: Handles race conditions in cluster deployments
- **Clear Failure Points**: Easy identification of failing dependencies

### Deployment Validation

Use the provided validation script to verify your deployment:

```bash
# Make script executable
chmod +x deployments/kubernetes/validate-deployment.sh

# Run validation
./deployments/kubernetes/validate-deployment.sh

# Expected output:
# ✓ etcd StatefulSet is ready (3/3 replicas)
# ✓ Redis is ready and responding to ping
# ✓ SIP B2BUA deployment is ready (2/2 replicas)
# ✓ All Voice Ferry services are running properly
```

### Configuration Secrets

Create Kubernetes secrets for sensitive data:

```bash
# Create JWT signing key secret
kubectl create secret generic voice-ferry-secrets \
    --from-literal=jwt-signing-key="your-256-bit-secret-key" \
    -n voice-ferry

# Create TLS secret (if using TLS)
kubectl create secret tls voice-ferry-tls \
    --cert=ssl/voice-ferry.crt \
    --key=ssl/voice-ferry.key \
    -n voice-ferry
```

### Persistent Storage

Configure persistent volumes for data:

```yaml
# etcd-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: etcd-data
  namespace: voice-ferry
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
```

Apply the PVC:

```bash
kubectl apply -f etcd-pvc.yaml
```

### Service Mesh Integration (Istio)

For service mesh deployment:

```yaml
# istio-gateway.yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: voice-ferry-gateway
  namespace: voice-ferry
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 5060
      name: sip-udp
      protocol: UDP
    hosts:
    - sip.your-domain.com
  - port:
      number: 5061
      name: sip-tls
      protocol: TLS
    tls:
      mode: PASSTHROUGH
    hosts:
    - sip.your-domain.com
```

## Configuration Management

### Configuration Hierarchy

Voice Ferry loads configuration in this order:

1. Default values
2. Configuration file (`config.yaml`)
3. Environment variables
4. Command line flags

### Environment Variable Override

Any configuration value can be overridden with environment variables:

```bash
# Override SIP port
export SIP_PORT=5070

# Override Redis host
export REDIS_HOST=redis.example.com

# Override log level
export LOG_LEVEL=debug
```

### Dynamic Configuration

Use etcd for dynamic configuration updates:

```bash
# Update SIP timeouts
etcdctl put /voice-ferry/config/sip/timeouts/dialog "3600s"

# Update session limits
etcdctl put /voice-ferry/config/redis/max_sessions_per_user "15"

# Configuration changes are applied automatically
```

## SSL/TLS Setup

### Certificate Requirements

For production deployment, you need:

- **SIP TLS**: Certificate for secure SIP communication
- **gRPC TLS**: Certificate for secure API access
- **Web UI TLS**: Certificate for HTTPS web interface

### Certificate Generation

#### Using Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate
certbot certonly --standalone \
    -d sip.your-domain.com \
    -d api.your-domain.com \
    -d ui.your-domain.com

# Create combined certificate
cat /etc/letsencrypt/live/sip.your-domain.com/fullchain.pem > voice-ferry.crt
cat /etc/letsencrypt/live/sip.your-domain.com/privkey.pem > voice-ferry.key
```

#### Using Self-Signed Certificates

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -days 365 -key ca.key -out ca.crt \
    -subj "/CN=Voice Ferry CA"

# Generate server private key
openssl genrsa -out voice-ferry.key 4096

# Generate certificate signing request
openssl req -new -key voice-ferry.key -out voice-ferry.csr \
    -subj "/CN=voice-ferry.local"

# Generate server certificate
openssl x509 -req -days 365 -in voice-ferry.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out voice-ferry.crt
```

## Monitoring and Observability

### Prometheus Metrics

Voice Ferry exposes Prometheus metrics at `/metrics`:

```bash
# Key metrics to monitor
voice_ferry_concurrent_calls_total
voice_ferry_sip_requests_total
voice_ferry_call_duration_seconds
voice_ferry_memory_usage_bytes
voice_ferry_cpu_usage_percent
```

### Grafana Dashboards

Import the pre-built Grafana dashboard:

```bash
# Copy dashboard configuration
cp configs/grafana/dashboards/voice-ferry.json /var/lib/grafana/dashboards/

# Or import via API
curl -X POST \
    -H "Content-Type: application/json" \
    -d @configs/grafana/dashboards/voice-ferry.json \
    http://admin:password@grafana:3000/api/dashboards/db
```

### Log Aggregation

Configure centralized logging:

```yaml
# fluentd-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/voice-ferry*.log
      pos_file /var/log/fluentd-voice-ferry.log.pos
      tag voice-ferry.*
      format json
    </source>
    
    <match voice-ferry.**>
      @type elasticsearch
      host elasticsearch
      port 9200
      index_name voice-ferry
    </match>
```

### Health Checks

Monitor service health:

```bash
# Check liveness
curl http://voice-ferry:8080/healthz/live

# Check readiness
curl http://voice-ferry:8080/healthz/ready

# Check startup
curl http://voice-ferry:8080/healthz/startup

# Get detailed status
curl http://voice-ferry:8080/status
```

## Scaling and High Availability

### Horizontal Pod Autoscaler

The HPA automatically scales based on CPU and memory:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: voice-ferry-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: voice-ferry
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Load Balancing

#### Layer 4 Load Balancing

```bash
# Using MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.0/config/manifests/metallb-native.yaml

# Configure IP pool
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: voice-ferry-pool
  namespace: metallb-system
spec:
  addresses:
  - 192.168.1.240-192.168.1.250
EOF
```

#### DNS-based Load Balancing

```bash
# Configure DNS SRV records
_sip._udp.voice-ferry.com. 300 IN SRV 10 5 5060 sip1.voice-ferry.com.
_sip._udp.voice-ferry.com. 300 IN SRV 10 5 5060 sip2.voice-ferry.com.
_sip._tcp.voice-ferry.com. 300 IN SRV 10 5 5060 sip1.voice-ferry.com.
_sip._tcp.voice-ferry.com. 300 IN SRV 10 5 5060 sip2.voice-ferry.com.
```

### Database High Availability

#### etcd Cluster

```yaml
# etcd StatefulSet for HA
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: etcd
spec:
  serviceName: etcd
  replicas: 3
  selector:
    matchLabels:
      app: etcd
  template:
    spec:
      containers:
      - name: etcd
        image: quay.io/coreos/etcd:v3.5.9
        command:
        - etcd
        - --initial-cluster=etcd-0=http://etcd-0.etcd:2380,etcd-1=http://etcd-1.etcd:2380,etcd-2=http://etcd-2.etcd:2380
```

#### Redis Sentinel

```yaml
# Redis with Sentinel for HA
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
spec:
  serviceName: redis
  replicas: 3
  template:
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command:
        - redis-server
        - --slaveof redis-0.redis 6379
```

## Security Considerations

### Network Security

```yaml
# NetworkPolicy for Voice Ferry
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: voice-ferry-netpol
spec:
  podSelector:
    matchLabels:
      app: voice-ferry
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
```

### Pod Security Standards

```yaml
# Pod Security Policy
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: voice-ferry-psp
spec:
  privileged: false
  runAsUser:
    rule: MustRunAsNonRoot
  seLinux:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - configMap
  - secret
  - emptyDir
  - persistentVolumeClaim
```

### Secrets Management

#### Using External Secrets Operator

```yaml
# External Secret for JWT key
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: voice-ferry-jwt
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: voice-ferry-secrets
  data:
  - secretKey: jwt-signing-key
    remoteRef:
      key: voice-ferry/jwt
      property: signing-key
```

## Troubleshooting

### Common Issues

#### SIP Registration Failures

```bash
# Check SIP configuration
kubectl logs -f deployment/voice-ferry -n voice-ferry | grep -i sip

# Test SIP connectivity
sipsak -s sip:voice-ferry.com -v

# Check network policies
kubectl describe networkpolicy voice-ferry-netpol -n voice-ferry
```

#### High Memory Usage

```bash
# Check memory metrics
kubectl top pods -n voice-ferry

# Review session limits
kubectl exec -it deployment/voice-ferry -n voice-ferry -- \
    redis-cli -h redis keys "session:*" | wc -l

# Adjust memory limits
kubectl patch deployment voice-ferry -n voice-ferry -p \
    '{"spec":{"template":{"spec":{"containers":[{"name":"voice-ferry","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

#### gRPC Connection Issues

```bash
# Test gRPC endpoint
grpcurl -plaintext voice-ferry:50051 list

# Check TLS configuration
openssl s_client -connect voice-ferry:50051 -servername voice-ferry

# Verify certificates
kubectl get secret voice-ferry-tls -n voice-ferry -o yaml
```

### Debug Mode

Enable debug logging:

```bash
# Temporary debug mode
kubectl set env deployment/voice-ferry LOG_LEVEL=debug -n voice-ferry

# Or edit configuration
kubectl edit configmap voice-ferry-config -n voice-ferry
# Change: log_level: "debug"
```

### Performance Tuning

#### CPU Optimization

```bash
# Check CPU usage
kubectl top pods -n voice-ferry

# Adjust worker pools
kubectl patch configmap voice-ferry-config -n voice-ferry --patch \
    '{"data":{"config.yaml":"...performance:\n  sip_workers: 20\n  grpc_workers: 10..."}}'
```

#### Memory Optimization

```bash
# Monitor memory patterns
kubectl exec -it deployment/voice-ferry -n voice-ferry -- \
    curl localhost:8080/debug/pprof/heap > heap.prof

# Analyze with go tool
go tool pprof heap.prof
```

### Backup and Recovery

#### Configuration Backup

```bash
# Backup etcd data
kubectl exec -it etcd-0 -n voice-ferry -- \
    etcdctl snapshot save /tmp/backup.db

# Copy backup
kubectl cp voice-ferry/etcd-0:/tmp/backup.db ./etcd-backup-$(date +%Y%m%d).db
```

#### Disaster Recovery

```bash
# Restore etcd from backup
kubectl exec -it etcd-0 -n voice-ferry -- \
    etcdctl snapshot restore /tmp/backup.db

# Recreate deployment
kubectl delete deployment voice-ferry -n voice-ferry
kubectl apply -f deployments/kubernetes/voice-ferry-production.yaml
```

### Support and Resources

- **Documentation**: [Voice Ferry Docs](https://github.com/2bleere/voice-ferry/docs)
- **Issues**: [GitHub Issues](https://github.com/2bleere/voice-ferry/issues)
- **Community**: [Discussions](https://github.com/2bleere/voice-ferry/discussions)
- **Commercial Support**: Available upon request

---

For additional help, please refer to the [Administration Guide](administration.md) or open an issue on GitHub.
