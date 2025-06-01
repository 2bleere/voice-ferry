# Voice Ferry Deployment Files

This directory contains production-ready deployment configurations for Voice Ferry SIP B2BUA.

## Directory Structure

```
deployments/
├── docker/
│   └── Dockerfile                 # Production-optimized Docker image
├── kubernetes/
│   ├── voice-ferry-production.yaml   # Complete K8s deployment
│   └── dependencies.yaml             # Redis, etcd, RTPEngine
└── README.md                      # This file
```

## Quick Start

### Docker Deployment

```bash
# Production deployment with all services
docker-compose -f docker-compose.prod.yml up -d

# Minimal deployment (B2BUA only)
docker run -d --name voice-ferry \
  -p 5060:5060/udp \
  -p 50051:50051 \
  -p 8080:8080 \
  -e JWT_SIGNING_KEY="your-secret-key" \
  2bleere/voice-ferry:latest
```

### Kubernetes Deployment

```bash
# Deploy dependencies first
kubectl apply -f kubernetes/dependencies.yaml

# Deploy Voice Ferry
kubectl apply -f kubernetes/voice-ferry-production.yaml

# Check status
kubectl get pods -n voice-ferry
```

## Configuration

### Required Environment Variables

- `JWT_SIGNING_KEY`: Secure 256-bit key for JWT tokens
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

### Optional Environment Variables

- `CONFIG_FILE`: Path to custom configuration file
- `REDIS_URL`: Redis connection string
- `ETCD_ENDPOINTS`: Comma-separated etcd endpoints

## Production Checklist

### Security
- [ ] Generate strong JWT signing key
- [ ] Configure TLS certificates
- [ ] Set up network policies
- [ ] Enable authentication
- [ ] Review IP access controls

### Scalability
- [ ] Configure horizontal pod autoscaler
- [ ] Set resource limits and requests
- [ ] Plan for load balancer configuration
- [ ] Set up persistent storage

### Monitoring
- [ ] Configure Prometheus metrics
- [ ] Set up Grafana dashboards
- [ ] Configure log aggregation
- [ ] Set up alerting rules

### High Availability
- [ ] Deploy multiple replicas
- [ ] Configure etcd cluster
- [ ] Set up Redis sentinel/cluster
- [ ] Plan disaster recovery

## Images

### Official Images

- **Main**: `2bleere/voice-ferry:latest`
- **Specific Version**: `2bleere/voice-ferry:v1.0.0`
- **Web UI**: `2bleere/voice-ferry-ui:latest`

### Building Custom Images

```bash
# Build production image
docker build -f docker/Dockerfile \
  --build-arg VERSION=v1.0.0 \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  -t voice-ferry:custom .
```

## Ports

| Port | Protocol | Service | Description |
|------|----------|---------|-------------|
| 5060 | UDP/TCP | SIP | Standard SIP signaling |
| 5061 | TCP | SIP-TLS | Secure SIP signaling |
| 50051 | TCP | gRPC | Management API |
| 8080 | TCP | HTTP | Health checks & metrics |

## Storage Requirements

### Persistent Volumes

- **etcd**: 10GB SSD (minimum)
- **Redis**: 5GB SSD (for session data)
- **Logs**: 10GB (optional, for persistent logging)

### Backup Strategy

```bash
# etcd backup
kubectl exec etcd-0 -n voice-ferry -- \
  etcdctl snapshot save /tmp/backup.db

# Configuration backup
kubectl get configmap voice-ferry-config -o yaml > config-backup.yaml
kubectl get secret voice-ferry-secrets -o yaml > secrets-backup.yaml
```

## Networking

### Service Mesh Integration

Works with:
- Istio
- Linkerd
- Consul Connect

### Load Balancer Requirements

- Supports Layer 4 (TCP/UDP) load balancing
- Session affinity recommended for SIP
- Health check endpoints available

## Troubleshooting

### Common Issues

1. **Pod startup failures**
   ```bash
   kubectl describe pod -l app=voice-ferry -n voice-ferry
   kubectl logs -f deployment/voice-ferry -n voice-ferry
   ```

2. **SIP connectivity issues**
   ```bash
   # Test from inside cluster
   kubectl exec -it deployment/voice-ferry -n voice-ferry -- \
     nslookup voice-ferry-sip
   ```

3. **Performance issues**
   ```bash
   # Check resource usage
   kubectl top pods -n voice-ferry
   
   # Review metrics
   curl http://voice-ferry:8080/metrics
   ```

### Debug Mode

```bash
# Enable debug logging
kubectl set env deployment/voice-ferry LOG_LEVEL=debug -n voice-ferry

# Access debug endpoints
kubectl port-forward svc/voice-ferry-metrics 8080:8080 -n voice-ferry
curl localhost:8080/debug/pprof/
```

## Support

- **Documentation**: See [docs/deployment.md](../docs/deployment.md)
- **Issues**: [GitHub Issues](https://github.com/2bleere/voice-ferry/issues)
- **Discussions**: [GitHub Discussions](https://github.com/2bleere/voice-ferry/discussions)

## License

MIT License - see [LICENSE](../LICENSE) file for details.
