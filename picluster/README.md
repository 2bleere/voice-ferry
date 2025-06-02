# Voice Ferry Pi Cluster / ARM64 Deployments

This directory contains Kubernetes deployment configurations optimized for ARM64 architectures, including Raspberry Pi clusters, ARM-based cloud instances, and Apple Silicon development environments.

## 🚀 Quick Start

### Deploy Complete ARM64 Stack

```bash
# Generate secrets
kubectl create namespace voice-ferry
kubectl create secret generic voice-ferry-secrets \
  --from-literal=jwt-signing-key="$(openssl rand -hex 32)" \
  -n voice-ferry

# Deploy complete stack
kubectl apply -f arm-production-complete.yaml

# Validate deployment
./validate-arm-deployment.sh
```

## 📁 Files Overview

### Core Deployment Files

| File | Description | Use Case |
|------|-------------|----------|
| **`arm-production-complete.yaml`** | 🎯 **Complete ARM64 production stack** | **Primary deployment file** |
| **`voice-ferry-production.yaml`** | Standard production deployment | x86_64 reference |
| **`dependencies.yaml`** | etcd and infrastructure components | Standalone services |
| **`redis-cluster.yaml`** | Redis cluster configuration | Session storage |
| **`web-ui.yaml`** | Web UI deployment | Management interface |
| **`sip-b2bua.yaml`** | SIP B2BUA service | Core SIP functionality |

### Scripts and Validation

| File | Description | Use Case |
|------|-------------|----------|
| **`validate-arm-deployment.sh`** | 🔍 **ARM64 deployment validator** | **Deployment verification** |
| **`validate-deployment.sh`** | Standard deployment validator | General validation |

### Documentation

| File | Description | Audience |
|------|-------------|----------|
| **`ARM-DEPLOYMENT-GUIDE.md`** | 📖 **Complete ARM64 deployment guide** | **Operators** |
| **`ARM-vs-x86-COMPARISON.md`** | Architecture comparison | Architects |
| **`PRODUCTION_DEPLOYMENT.md`** | General production guide | DevOps |
| **`DEPLOYMENT_STRATEGY.md`** | Deployment strategies | Platform teams |

## 🏗️ Supported ARM64 Platforms

### Raspberry Pi Clusters
- **Pi 4 (4GB+)** - Recommended for development
- **Pi 4 (8GB)** - Recommended for production
- **Pi 5** - Latest and fastest option
- **Pi CM4** - Compute modules for custom builds

### Cloud ARM64 Instances
- **AWS Graviton** (t4g, m6g, c6g series)
- **Google Cloud Tau T2A** instances
- **Azure Ampere Altra** instances
- **Oracle Cloud Ampere A1** instances

### Development Platforms
- **Apple Silicon** (M1/M2/M3 Macs)
- **NVIDIA Jetson** platforms
- **Rockchip RK3588** boards

## 📊 Resource Requirements

### Minimum Cluster Specifications

| Component | Nodes | CPU | Memory | Storage |
|-----------|-------|-----|--------|---------|
| **Development** | 1 | 4 cores | 4GB | 20GB |
| **Small Production** | 3 | 4 cores each | 4GB each | 50GB total |
| **Medium Production** | 3 | 8 cores each | 8GB each | 100GB total |

### Per-Component Resource Usage

| Component | Replicas | CPU Request | Memory Request | Storage |
|-----------|----------|-------------|----------------|---------|
| **etcd** | 3 | 100m each | 256Mi each | 2Gi each |
| **Redis Cluster** | 6 | 50m each | 128Mi each | 1Gi each |
| **Voice Ferry B2BUA** | 2+ | 100m each | 128Mi each | - |
| **Web UI** | 1+ | 50m each | 128Mi each | - |

## 🎛️ Configuration Options

### ARM64 Optimizations

The ARM64 deployment includes several optimizations:

```yaml
# Go runtime optimization
env:
- name: GOMAXPROCS
  value: "2"              # Match available cores
- name: GOGC
  value: "100"            # Memory optimization

# Node.js optimization (Web UI)
env:
- name: UV_THREADPOOL_SIZE
  value: "4"              # ARM core count
- name: NODE_OPTIONS
  value: "--max-old-space-size=256"  # Memory limit
```

### Platform-Specific Storage Classes

```yaml
# Raspberry Pi with Longhorn
storageClassName: "longhorn"

# AWS Graviton with GP3
storageClassName: "gp3"

# Local development
storageClassName: "local-path"
```

## 🔧 Platform Setup Guides

### Raspberry Pi Cluster

1. **Install K3s**
   ```bash
   # Master node
   curl -sfL https://get.k3s.io | sh -
   
   # Worker nodes
   curl -sfL https://get.k3s.io | K3S_URL=https://master-ip:6443 K3S_TOKEN=your-token sh -
   ```

2. **Install Longhorn Storage**
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.5.3/deploy/longhorn.yaml
   ```

3. **Deploy Voice Ferry**
   ```bash
   kubectl apply -f arm-production-complete.yaml
   ```

### AWS Graviton (EKS)

1. **Create Cluster**
   ```bash
   eksctl create cluster \
     --name voice-ferry-arm \
     --region us-west-2 \
     --nodegroup-name graviton-nodes \
     --node-type t4g.medium \
     --nodes 3
   ```

2. **Install EBS CSI Driver**
   ```bash
   kubectl apply -k "github.com/kubernetes-sigs/aws-ebs-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.24"
   ```

### Google Cloud Tau T2A

1. **Create GKE Cluster**
   ```bash
   gcloud container clusters create voice-ferry-arm \
     --machine-type=t2a-standard-2 \
     --num-nodes=3 \
     --zone=us-central1-a
   ```

## 🚀 Deployment Workflows

### Development Deployment

```bash
# Quick development setup
kubectl create namespace voice-ferry
kubectl apply -f arm-production-complete.yaml

# Port-forward for local access
kubectl port-forward -n voice-ferry service/voice-ferry-web-ui 3000:3000
```

### Production Deployment

```bash
# 1. Generate production secrets
JWT_SIGNING_KEY=$(openssl rand -hex 32)
kubectl create secret generic voice-ferry-secrets \
  --from-literal=jwt-signing-key="$JWT_SIGNING_KEY" \
  -n voice-ferry

# 2. Update domain and TLS
sed -i 's/voice-ferry.yourdomain.com/your-actual-domain.com/g' arm-production-complete.yaml

# 3. Deploy with validation
kubectl apply -f arm-production-complete.yaml
./validate-arm-deployment.sh

# 4. Set up monitoring
kubectl port-forward -n voice-ferry service/voice-ferry-metrics 8080:8080
```

## 📈 Monitoring and Observability

### Health Checks

```bash
# Quick health check
./validate-arm-deployment.sh health

# Manual checks
kubectl get pods -n voice-ferry
kubectl get svc -n voice-ferry
kubectl top pods -n voice-ferry
```

### Metrics and Logging

```bash
# View logs
kubectl logs -f deployment/voice-ferry -n voice-ferry
kubectl logs -f deployment/voice-ferry-web-ui -n voice-ferry

# Access metrics
kubectl port-forward -n voice-ferry service/voice-ferry-metrics 8080:8080
curl http://localhost:8080/metrics
```

### Performance Monitoring

```bash
# Generate deployment report
./validate-arm-deployment.sh report

# Monitor resource usage
watch kubectl top pods -n voice-ferry
```

## 🔒 Security Considerations

### Default Security Features

- ✅ **Non-root containers** - All pods run as user 1001
- ✅ **Read-only root filesystem** - Where applicable
- ✅ **Capability dropping** - All capabilities dropped
- ✅ **Network policies** - Ingress/egress restrictions
- ✅ **Pod security contexts** - Proper user/group settings

### Production Security Checklist

- [ ] Replace default JWT signing keys
- [ ] Configure TLS certificates
- [ ] Set up proper RBAC
- [ ] Enable audit logging
- [ ] Configure network policies
- [ ] Set up secret rotation

## 🛠️ Troubleshooting

### Common Issues

1. **Pods stuck in Pending**
   ```bash
   kubectl describe pods -n voice-ferry
   kubectl get events -n voice-ferry
   ```

2. **Architecture mismatch**
   ```bash
   kubectl get nodes -o wide
   kubectl describe pod <pod-name> -n voice-ferry
   ```

3. **Storage issues**
   ```bash
   kubectl get pvc -n voice-ferry
   kubectl get storageclass
   ```

### Platform-Specific Issues

#### Raspberry Pi
- Check SD card speed and space
- Monitor temperature (`vcgencmd measure_temp`)
- Verify Longhorn installation

#### Cloud ARM64
- Check instance types support ARM64
- Verify storage class configuration
- Check security group/firewall rules

## 📚 Additional Resources

### Documentation
- [ARM Deployment Guide](./ARM-DEPLOYMENT-GUIDE.md) - Complete setup guide
- [ARM vs x86 Comparison](./ARM-vs-x86-COMPARISON.md) - Architecture comparison
- [Production Deployment](./PRODUCTION_DEPLOYMENT.md) - General production guide

### Community
- **GitHub Discussions** - Ask questions and share experiences
- **Issues** - Report bugs or request features
- **ARM64 Kubernetes SIG** - ARM-specific Kubernetes community

### External Resources
- [Kubernetes ARM64 Support](https://kubernetes.io/docs/concepts/cluster-administration/platforms/)
- [Pi Cluster Builds](https://github.com/geerlingguy/pi-cluster)
- [ARM64 Container Images](https://hub.docker.com/search?q=arm64)

## 🤝 Contributing

We welcome contributions to improve ARM64 support:

1. **Performance optimizations** for specific ARM platforms
2. **Platform-specific documentation** and guides
3. **Testing on new ARM64 platforms**
4. **Automation and tooling** improvements

## 📄 License

MIT License - see [LICENSE](../../LICENSE) file for details.

---

**🎯 Ready to deploy?** Start with the [ARM Deployment Guide](./ARM-DEPLOYMENT-GUIDE.md) for complete instructions!
