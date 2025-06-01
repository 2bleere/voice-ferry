# etcd Status Monitoring - Quick Reference Guide

## 🚀 Quick Start Commands

### 1. Check Current System Status
```bash
# Complete health check of all services
./scripts/health-check.sh

# Check specific etcd monitoring integration
./scripts/health-check.sh monitoring

# Get system status summary
./scripts/health-check.sh summary
```

### 2. Deploy Production Environment
```bash
# Full production deployment
./scripts/deploy-production.sh deploy

# Check deployment status
./scripts/deploy-production.sh status

# View service logs
./scripts/deploy-production.sh logs web-ui
```

### 3. Kubernetes Validation
```bash
# Validate Kubernetes deployment
./deployments/kubernetes/validate-deployment.sh

# Wait for deployment readiness then validate
./deployments/kubernetes/validate-deployment.sh wait
```

## 🔍 etcd Status Monitoring Features

### Dashboard Interface
- **Location**: Web UI Dashboard (http://localhost:8080)
- **Status Indicator**: Real-time etcd cluster health display
- **Updates**: Live WebSocket-based status updates every 5 seconds

### Monitoring Data
```json
{
  "services": {
    "etcd": {
      "status": "healthy",
      "members": 3,
      "leader": "etcd-0",
      "connectivity": "ok",
      "response_time": "45ms"
    }
  }
}
```

### Status Indicators
- 🟢 **Green**: etcd cluster healthy, all members responding
- 🟡 **Yellow**: etcd cluster degraded, some members slow/unavailable
- 🔴 **Red**: etcd cluster unavailable or connection failed
- ⚪ **Gray**: etcd status unknown or initializing

## 📋 Environment Configuration

### Docker Compose (Production)
```bash
# Required environment variables in docker-compose.prod.yml
ETCD_ENDPOINTS=http://etcd:2379
ETCD_DIAL_TIMEOUT=5000
ETCD_REQUEST_TIMEOUT=10000
MONITORING_ENABLED=true
```

### Kubernetes Deployment
```bash
# ConfigMap includes all necessary etcd configuration
# See: deployments/kubernetes/web-ui.yaml
kubectl apply -f deployments/kubernetes/web-ui.yaml
```

## 🛠️ Troubleshooting

### Common Issues

#### etcd Status Shows "Unknown"
```bash
# Check etcd service connectivity
./scripts/health-check.sh etcd

# Verify environment variables
docker exec web-ui printenv | grep ETCD
```

#### WebSocket Connection Failed
```bash
# Check Web UI health
./scripts/health-check.sh webui

# Verify port accessibility
curl http://localhost:8080/health
```

#### Monitoring API Not Responding
```bash
# Test monitoring endpoint directly
curl http://localhost:8080/api/monitoring

# Check service logs
./scripts/deploy-production.sh logs web-ui
```

### Debug Commands
```bash
# View real-time etcd cluster status
etcdctl --endpoints=http://localhost:2379 endpoint health

# Check etcd cluster members
etcdctl --endpoints=http://localhost:2379 member list

# Monitor Web UI logs for etcd connectivity
docker logs -f voice-ferry-web-ui
```

## 📊 Monitoring Architecture

### Component Flow
```
Browser Dashboard ←→ WebSocket ←→ Web UI ←→ etcd Cluster
                                      ↓
                              Monitoring Service
                                      ↓
                            Health Check Scripts
```

### Health Check Intervals
- **Frontend Updates**: Every 5 seconds via WebSocket
- **Backend Monitoring**: Every 5 seconds from monitoring service
- **Health Scripts**: On-demand or CI/CD integration
- **etcd Timeouts**: 5 second dial, 10 second request timeouts

## 🔐 Security Configuration

### TLS Configuration (Kubernetes)
```yaml
# Automatic TLS termination via Ingress
# See: deployments/kubernetes/web-ui-ingress.yaml
spec:
  tls:
    - hosts:
        - voice-ferry.example.com
      secretName: voice-ferry-tls
```

### Network Policies
```yaml
# Restricts Web UI to only necessary service communication
# Allows: etcd:2379, redis:6379, rtpengine:22222
# See: deployments/kubernetes/web-ui-ingress.yaml (NetworkPolicy)
```

## 🎯 Production Readiness

### Deployment Verification Checklist
- ✅ etcd cluster healthy (3 members)
- ✅ Web UI deployment running
- ✅ WebSocket connectivity working
- ✅ Monitoring API accessible
- ✅ Dashboard showing real-time status
- ✅ Health checks passing
- ✅ Auto-scaling configured
- ✅ TLS termination active

### Performance Metrics
- **etcd Response Time**: < 100ms typical
- **WebSocket Updates**: 5 second intervals
- **Health Check Duration**: < 30 seconds complete validation
- **Resource Usage**: < 100MB RAM per Web UI pod

## 📚 Additional Resources

### Documentation Files
- `ETCD_MONITORING_DEPLOYMENT_COMPLETE.md` - Complete implementation summary
- `deployments/kubernetes/DEPLOYMENT_STRATEGY.md` - Kubernetes deployment guide
- `deployments/kubernetes/PRODUCTION_DEPLOYMENT.md` - Production setup details

### Configuration Files
- `configs/.env.production` - Production environment variables
- `deployments/kubernetes/web-ui.yaml` - Complete Kubernetes deployment
- `docker-compose.prod.yml` - Enhanced production Docker setup

### Automation Scripts
- `scripts/deploy-production.sh` - Automated deployment pipeline
- `scripts/health-check.sh` - Comprehensive health validation
- `deployments/kubernetes/validate-deployment.sh` - Kubernetes validation

---

**🎉 etcd Status Monitoring is now fully operational and production-ready!**

For support or questions, refer to the comprehensive documentation or run the health check scripts for current status.
