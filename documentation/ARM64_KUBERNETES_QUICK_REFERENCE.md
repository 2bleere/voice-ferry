# Voice Ferry ARM64 Kubernetes Quick Reference

**Last Updated:** June 3, 2025  
**Status:** ✅ Production Ready

## 🚀 Quick Deployment

### One-Command Deployment
```bash
kubectl apply -f picluster/kubernetes/arm-production-complete.yaml
```

### Verification Commands
```bash
# Check all pods
kubectl get pods -n voice-ferry

# Expected healthy output:
# NAME                                 READY   STATUS    RESTARTS   AGE
# voice-ferry-6c586b5c59-4gsfq         1/1     Running   0          2m
# voice-ferry-6c586b5c59-xrps6         1/1     Running   0          2m
# rtpengine-7f89d94875-pqkpc           1/1     Running   0          2m
# redis-cluster-0                      1/1     Running   0          2m
# redis-cluster-1                      1/1     Running   0          2m
# redis-cluster-2                      1/1     Running   0          2m
# redis-cluster-3                      1/1     Running   0          2m
# redis-cluster-4                      1/1     Running   0          2m
# redis-cluster-5                      1/1     Running   0          2m
# etcd-0                               1/1     Running   0          2m
# voice-ferry-web-ui-8996797db-k4dwt   1/1     Running   0          2m
```

## 🔧 Key Configuration Details

### Network Configuration
- **RTPEngine**: Direct IP `192.168.1.208:22222` (UDP)
- **SIP Ports**: 5060 (UDP/TCP), 5061 (TCP TLS)
- **Health Check**: Port 8080, endpoint `/health`
- **gRPC API**: Port 50051
- **Metrics**: Port 9090

### Resource Allocation (ARM64 Optimized)
```yaml
Voice Ferry:
  requests: {memory: "128Mi", cpu: "100m"}
  limits: {memory: "512Mi", cpu: "500m"}

RTPEngine:
  requests: {memory: "128Mi", cpu: "100m"}
  limits: {memory: "512Mi", cpu: "500m"}
```

## 🩺 Health Check Commands

### Application Health
```bash
# Overall system health
kubectl exec -n voice-ferry <voice-ferry-pod> -- curl -s http://localhost:8080/health

# Expected response:
# {"status":"healthy","components":{"rtpengine":{"status":"healthy"},"sip_server":{"status":"healthy"}}}
```

### RTPEngine Connectivity
```bash
# Check RTPEngine logs (should show successful pings)
kubectl logs -n voice-ferry -l app=voice-ferry | grep "RTPEngine ping"

# Expected output:
# DEBUG: RTPEngine ping for instance rtpengine-1: result=ok, healthy=true
```

### Network Connectivity Tests
```bash
# Test RTPEngine connectivity
kubectl exec -n voice-ferry <voice-ferry-pod> -- nc -u 192.168.1.208 22222

# Test DNS resolution
kubectl exec -n voice-ferry <voice-ferry-pod> -- nslookup rtpengine
```

## 🚨 Troubleshooting

### Pod Not Ready
```bash
# Check pod events
kubectl describe pod <pod-name> -n voice-ferry

# Check logs
kubectl logs <pod-name> -n voice-ferry

# Common issues:
# - Network policy blocking port 22222
# - Wrong health check endpoint (/healthz/* vs /health)
# - RTPEngine service not available
```

### RTPEngine Connection Issues
```bash
# Test from within pod
kubectl exec -n voice-ferry <pod> -- nc -u 192.168.1.208 22222

# Check network policy
kubectl describe networkpolicy voice-ferry-network-policy -n voice-ferry

# Verify configuration
kubectl get configmap voice-ferry-config -n voice-ferry -o yaml
```

## 🔄 Update Procedures

### Rolling Update
```bash
# Update image
kubectl set image deployment/voice-ferry voice-ferry=2bleere/voice-ferry:arm64-latest -n voice-ferry

# Watch rollout
kubectl rollout status deployment/voice-ferry -n voice-ferry

# Verify health
kubectl get pods -n voice-ferry -l app=voice-ferry
```

### Configuration Update
```bash
# Edit config
kubectl edit configmap voice-ferry-config -n voice-ferry

# Restart deployment to pick up changes
kubectl rollout restart deployment/voice-ferry -n voice-ferry
```

## 📊 Monitoring

### Key Metrics Endpoints
- **Health**: `http://<pod-ip>:8080/health`
- **Metrics**: `http://<pod-ip>:9090/metrics`
- **Debug**: `http://<pod-ip>:8080/debug/pprof/`

### Log Monitoring
```bash
# Real-time logs
kubectl logs -f -n voice-ferry -l app=voice-ferry

# Filter for errors
kubectl logs -n voice-ferry -l app=voice-ferry | grep -i error

# Filter for RTPEngine status
kubectl logs -n voice-ferry -l app=voice-ferry | grep "RTPEngine ping"
```

## 🔐 Security Notes

### Network Policies
- Ingress: Monitoring namespace + SIP ports
- Egress: DNS, HTTPS, RTPEngine (22222), SIP (5060/5061)

### Pod Security
- Non-privileged containers (except RTPEngine)
- Read-only root filesystem where possible
- Minimal capabilities

## 📞 Service Endpoints

### Internal Services
- `voice-ferry-sip.voice-ferry.svc.cluster.local:5060` (SIP)
- `voice-ferry-grpc.voice-ferry.svc.cluster.local:50051` (gRPC)
- `voice-ferry-metrics.voice-ferry.svc.cluster.local:9090` (Metrics)

### External Access
- SIP traffic load balanced across pods
- Web UI available via Ingress
- gRPC API secured with JWT

## 🔧 Critical Success Factors

### Deployment Requirements
1. ✅ **Network Policy**: Must allow egress to port 22222
2. ✅ **Health Endpoints**: Use `/health` not `/healthz/*`
3. ✅ **UDP Connections**: Fresh connections for health checks
4. ✅ **Direct IP**: Use `192.168.1.208` for RTPEngine host

### Verified Working Components
- ARM64 container images
- Network policies with proper egress rules
- Health check probe configurations
- RTPEngine UDP connectivity
- Redis cluster integration
- Rolling update procedures

---

**Deployment Status**: ✅ **FULLY OPERATIONAL**  
**Next Steps**: End-to-end SIP testing and load validation
