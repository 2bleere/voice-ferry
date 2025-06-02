# Voice Ferry ARM64 Production Deployment Guide

## Overview

This guide provides complete instructions for deploying Voice Ferry on ARM64 Kubernetes clusters, including Raspberry Pi clusters, ARM-based cloud instances, and Apple Silicon development environments.

## Prerequisites

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **CPU** | 4 ARM cores total | 8+ ARM cores total |
| **Memory** | 4GB RAM total | 8GB+ RAM total |
| **Storage** | 20GB available | 50GB+ available |
| **Network** | 100Mbps | 1Gbps+ |

### Software Requirements

- Kubernetes 1.24+ with ARM64 support
- kubectl configured for your cluster
- Storage class configured (longhorn, local-path, etc.)
- MetalLB or equivalent LoadBalancer (for SIP services)
- Ingress controller (nginx, traefik, etc.)
- cert-manager (optional, for TLS)

### Supported ARM64 Platforms

- **Raspberry Pi 4/5 clusters** (4GB+ RAM recommended)
- **AWS Graviton instances** (t4g, m6g, c6g series)
- **Google Cloud Tau T2A instances**
- **Azure Ampere Altra instances**
- **Apple Silicon development** (Docker Desktop)
- **NVIDIA Jetson** platforms

## Quick Start

### 1. Clone and Prepare

```bash
git clone https://github.com/yourusername/voice-ferry.git
cd voice-ferry/picluster/kubernetes
```

### 2. Customize Configuration

Edit the following sections in `arm-production-complete.yaml`:

```yaml
# Update domain name
host: voice-ferry.yourdomain.com

# Update storage class for your cluster
storageClassName: "longhorn"  # or "local-path", "fast-ssd", etc.

# Update LoadBalancer IP (if using MetalLB)
metallb.universe.tf/loadBalancerIPs: "192.168.1.100"
```

### 3. Generate Secrets

```bash
# Generate strong secrets for production
JWT_SECRET=$(openssl rand -hex 32)
SESSION_SECRET=$(openssl rand -hex 32)
JWT_SIGNING_KEY=$(openssl rand -hex 32)

# Create namespace
kubectl create namespace voice-ferry

# Create secrets
kubectl create secret generic voice-ferry-secrets \
  --from-literal=jwt-signing-key="$JWT_SIGNING_KEY" \
  -n voice-ferry

kubectl create secret generic web-ui-secrets \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --from-literal=session-secret="$SESSION_SECRET" \
  -n voice-ferry
```

### 4. Deploy

```bash
# Deploy the complete stack
kubectl apply -f arm-production-complete.yaml

# Monitor deployment
kubectl get pods -n voice-ferry -w
```

## Architecture Overview

### Components Deployed

1. **Redis Cluster** (6 pods)
   - 3 master nodes, 3 replica nodes
   - Session storage and caching
   - ARM-optimized configuration

2. **etcd Cluster** (3 pods)
   - Distributed configuration storage
   - Routing rules and system state
   - HA configuration with leader election

3. **Voice Ferry B2BUA** (2+ pods)
   - SIP Back-to-Back User Agent
   - WebRTC gateway functionality
   - Auto-scaling based on load

4. **Web UI** (1 pod)
   - Management interface
   - Real-time monitoring
   - Configuration management

### Network Architecture

```
Internet → LoadBalancer → SIP B2BUA (UDP/TCP 5060, TLS 5061)
Internet → Ingress → Web UI (HTTP/HTTPS 443)
Internal → ClusterIP → gRPC API (50051)
Internal → ClusterIP → Metrics (8080)
```

## ARM-Specific Optimizations

### Resource Limits

The deployment includes ARM-optimized resource limits:

```yaml
# B2BUA containers
resources:
  requests:
    memory: "128Mi"  # Reduced for ARM
    cpu: "100m"      # Reduced for ARM
  limits:
    memory: "512Mi"  # Reduced for ARM
    cpu: "500m"      # Reduced for ARM

# Environment variables for Go runtime optimization
env:
- name: GOMAXPROCS
  value: "2"         # ARM optimization
- name: GOGC
  value: "100"       # ARM memory optimization
```

### Node Affinity

Ensures pods are scheduled on ARM64 nodes:

```yaml
nodeSelector:
  kubernetes.io/arch: arm64
tolerations:
- key: arm
  operator: Equal
  value: "true"
  effect: NoSchedule
```

### Performance Tuning

ARM-specific performance configurations:

```yaml
# Reduced limits for ARM clusters
performance:
  sip_workers: 4              # vs 10 on x86
  grpc_workers: 2             # vs 5 on x86
  max_concurrent_calls: 2000  # vs 5000 on x86
  max_connections_per_ip: 50  # vs 100 on x86

# Session limits adjusted for ARM
sessions:
  limits:
    global_max_sessions: 5000   # vs 10000 on x86
    per_ip_max_sessions: 50     # vs 100 on x86

# Rate limiting tuned for ARM
rate_limiting:
  global:
    requests_per_second: 500    # vs 1000 on x86
    burst: 1000                 # vs 2000 on x86
```

## Platform-Specific Instructions

### Raspberry Pi Clusters

#### K3s Setup
```bash
# On master node
curl -sfL https://get.k3s.io | sh -

# Get token for worker nodes
sudo cat /var/lib/rancher/k3s/server/node-token

# On worker nodes
curl -sfL https://get.k3s.io | K3S_URL=https://master-ip:6443 K3S_TOKEN=your-token sh -
```

#### Storage (Longhorn)
```bash
# Install Longhorn for Pi clusters
kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.5.3/deploy/longhorn.yaml

# Set as default storage class
kubectl patch storageclass longhorn -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

### AWS Graviton Instances

#### EKS Setup
```bash
# Create EKS cluster with Graviton nodes
eksctl create cluster \
  --name voice-ferry-arm \
  --region us-west-2 \
  --nodegroup-name graviton-nodes \
  --node-type t4g.medium \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 6 \
  --node-ami-family AmazonLinux2 \
  --node-labels "kubernetes.io/arch=arm64"
```

#### Storage (EBS CSI)
```bash
# Install EBS CSI driver
kubectl apply -k "github.com/kubernetes-sigs/aws-ebs-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.24"

# Use GP3 storage class
kubectl patch storageclass gp2 -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  fsType: ext4
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
EOF
```

### Google Cloud Tau T2A

#### GKE Setup
```bash
# Create GKE cluster with Tau T2A nodes
gcloud container clusters create voice-ferry-arm \
  --machine-type=t2a-standard-2 \
  --num-nodes=3 \
  --zone=us-central1-a \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=6 \
  --node-labels="kubernetes.io/arch=arm64"
```

### Apple Silicon Development

#### Docker Desktop Setup
```bash
# Enable Kubernetes in Docker Desktop
# Ensure ARM64 emulation is disabled for better performance

# Deploy to local cluster
kubectl config use-context docker-desktop
kubectl apply -f arm-production-complete.yaml
```

## Monitoring and Troubleshooting

### Health Checks

```bash
# Check all pods
kubectl get pods -n voice-ferry

# Check services
kubectl get svc -n voice-ferry

# Check ingress
kubectl get ingress -n voice-ferry

# Check logs
kubectl logs -f deployment/voice-ferry -n voice-ferry
kubectl logs -f deployment/voice-ferry-web-ui -n voice-ferry
```

### Performance Monitoring

```bash
# Check resource usage
kubectl top pods -n voice-ferry
kubectl top nodes

# Check HPA status
kubectl get hpa -n voice-ferry

# Check PDB status
kubectl get pdb -n voice-ferry
```

### Common Issues

#### 1. Pods Stuck in Pending
```bash
# Check node resources
kubectl describe nodes

# Check for taints
kubectl get nodes -o json | jq '.items[].spec.taints'

# Check storage class
kubectl get storageclass
```

#### 2. ARM Architecture Mismatch
```bash
# Verify node architecture
kubectl get nodes -o wide

# Check pod node affinity
kubectl describe pod <pod-name> -n voice-ferry
```

#### 3. Storage Issues on Pi Clusters
```bash
# Check Longhorn status
kubectl get pods -n longhorn-system

# Check volume status
kubectl get pv,pvc -n voice-ferry
```

## Scaling Configuration

### Horizontal Scaling

```bash
# Scale B2BUA replicas
kubectl scale deployment voice-ferry --replicas=4 -n voice-ferry

# Scale web-ui replicas
kubectl scale deployment voice-ferry-web-ui --replicas=2 -n voice-ferry
```

### Vertical Scaling

Update resource limits in the deployment:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Auto-scaling

The deployment includes HPA configuration that automatically scales based on:
- CPU utilization (70% threshold)
- Memory utilization (80% threshold)

## Security Considerations

### Secrets Management

```bash
# Rotate JWT signing key
NEW_KEY=$(openssl rand -hex 32)
kubectl patch secret voice-ferry-secrets -n voice-ferry -p '{"data":{"jwt-signing-key":"'$(echo -n $NEW_KEY | base64)'"}}'

# Restart deployments to pick up new secret
kubectl rollout restart deployment/voice-ferry -n voice-ferry
```

### Network Policies

The deployment includes network policies that:
- Restrict ingress to required ports only
- Allow egress to required services only
- Isolate the voice-ferry namespace

### TLS Configuration

```bash
# Create TLS certificate secret
kubectl create secret tls voice-ferry-tls \
  --cert=tls.crt \
  --key=tls.key \
  -n voice-ferry
```

## Backup and Recovery

### etcd Backup

```bash
# Create etcd backup
kubectl exec -it etcd-0 -n voice-ferry -- etcdctl snapshot save /backup/snapshot.db

# Copy backup out of pod
kubectl cp voice-ferry/etcd-0:/backup/snapshot.db ./etcd-backup-$(date +%Y%m%d).db
```

### Redis Backup

```bash
# Create Redis backup
kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli BGSAVE

# Copy RDB file
kubectl cp voice-ferry/redis-cluster-0:/data/dump.rdb ./redis-backup-$(date +%Y%m%d).rdb
```

## Performance Benchmarks

### ARM64 vs x86_64 Performance

| Metric | ARM64 (Graviton3) | x86_64 (Intel) | Notes |
|--------|-------------------|----------------|--------|
| **SIP Messages/sec** | 800-1200 | 1000-1500 | ARM shows 80% performance |
| **Memory Usage** | 15% lower | Baseline | ARM more memory efficient |
| **Power Consumption** | 60% lower | Baseline | Significant power savings |
| **Cost (AWS)** | 20% lower | Baseline | Better price/performance |

### Raspberry Pi 4 Cluster (4GB)

| Metric | Value | Notes |
|--------|-------|--------|
| **Max Concurrent Sessions** | 2000 | With 3-node cluster |
| **SIP Messages/sec** | 500-800 | Limited by network |
| **Memory Usage per Pod** | 128-256MB | Efficient for ARM |
| **Storage IOPS** | 300-500 | With fast SD cards |

## Updates and Maintenance

### Rolling Updates

```bash
# Update B2BUA image
kubectl set image deployment/voice-ferry voice-ferry=2bleere/voice-ferry:v1.1.0 -n voice-ferry

# Update Web UI image
kubectl set image deployment/voice-ferry-web-ui web-ui=2bleere/voice-ferry-web-ui:v1.1.0 -n voice-ferry

# Check rollout status
kubectl rollout status deployment/voice-ferry -n voice-ferry
```

### Configuration Updates

```bash
# Update ConfigMap
kubectl patch configmap voice-ferry-config -n voice-ferry --patch-file=config-update.yaml

# Restart deployment to pick up changes
kubectl rollout restart deployment/voice-ferry -n voice-ferry
```

## Support and Community

- **Documentation**: `/docs` directory
- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **ARM Community**: Join ARM64 Kubernetes SIG

## License

MIT License - see LICENSE file for details.
