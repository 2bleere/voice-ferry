# Voice Ferry Deployment Checklist

**Version**: 1.0  
**Last Updated**: June 3, 2025  
**Status**: ✅ Verified in Production

## 📋 Pre-Deployment Checklist

### Infrastructure Requirements
- [ ] Kubernetes cluster v1.20+ available
- [ ] kubectl configured and authenticated
- [ ] Cluster has ARM64 nodes (for ARM deployment)
- [ ] Network policies supported
- [ ] LoadBalancer or Ingress controller available
- [ ] Persistent volume provisioner configured

### External Dependencies
- [ ] RTPEngine instance deployed and accessible
- [ ] RTPEngine responding on port 22222 (UDP)
- [ ] Direct IP address available for RTPEngine
- [ ] Network connectivity between cluster and RTPEngine verified

### Security Prerequisites
- [ ] JWT signing key generated (256-bit minimum)
- [ ] TLS certificates available (optional but recommended)
- [ ] RBAC policies reviewed
- [ ] Network security policies defined

## 🚀 Deployment Steps

### 1. Apply Kubernetes Manifest
```bash
# Deploy complete platform
kubectl apply -f picluster/kubernetes/arm-production-complete.yaml

# Wait for namespace creation
kubectl wait --for=condition=Established namespace/voice-ferry --timeout=60s
```

### 2. Verify Dependencies First
```bash
# Check etcd
kubectl wait --for=condition=Ready pod/etcd-0 -n voice-ferry --timeout=300s

# Check Redis cluster
kubectl wait --for=condition=Ready pod -l app=redis-cluster -n voice-ferry --timeout=300s

# Check RTPEngine
kubectl wait --for=condition=Ready pod -l app=rtpengine -n voice-ferry --timeout=300s
```

### 3. Verify Voice Ferry Deployment
```bash
# Wait for Voice Ferry pods
kubectl wait --for=condition=Ready pod -l app=voice-ferry -n voice-ferry --timeout=300s

# Check deployment status
kubectl get deployment voice-ferry -n voice-ferry
```

### 4. Health Verification
```bash
# Test health endpoint
POD_NAME=$(kubectl get pod -l app=voice-ferry -n voice-ferry -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n voice-ferry $POD_NAME -- curl -s http://localhost:8080/health

# Check RTPEngine connectivity logs
kubectl logs -n voice-ferry -l app=voice-ferry | grep "RTPEngine ping"
```

## ✅ Post-Deployment Verification

### Service Health Checks
- [ ] All pods in Running state (1/1 Ready)
- [ ] Health endpoint returns `{"status":"healthy"}`
- [ ] RTPEngine logs show `result=ok, healthy=true`
- [ ] No error messages in application logs
- [ ] Services responding on expected ports

### Network Connectivity Tests
```bash
# Test RTPEngine connectivity from Voice Ferry pod
kubectl exec -n voice-ferry $POD_NAME -- nc -u 192.168.1.208 22222

# Test SIP port accessibility
kubectl exec -n voice-ferry $POD_NAME -- nc -u localhost 5060

# Test gRPC API
kubectl exec -n voice-ferry $POD_NAME -- nc localhost 50051
```

### Configuration Verification
- [ ] ConfigMap contains correct RTPEngine IP
- [ ] Network policy allows egress to port 22222
- [ ] TLS certificates properly mounted (if using TLS)
- [ ] Environment variables correctly set
- [ ] Resource limits appropriate for environment

## 🔧 Common Issues & Solutions

### Issue: Pods Not Ready
**Symptoms**: Pods stuck in "0/1 Running" state

**Check**:
```bash
kubectl describe pod $POD_NAME -n voice-ferry
kubectl logs $POD_NAME -n voice-ferry
```

**Common Causes**:
- Wrong health check endpoint (should be `/health`)
- Network policy blocking RTPEngine port
- RTPEngine not accessible

### Issue: RTPEngine Connection Failed
**Symptoms**: Logs show "connection refused" or "no healthy instances"

**Check**:
```bash
# Test direct connectivity
kubectl exec -n voice-ferry $POD_NAME -- nc -u 192.168.1.208 22222

# Check network policy
kubectl describe networkpolicy voice-ferry-network-policy -n voice-ferry
```

**Solution**:
- Verify RTPEngine IP in ConfigMap
- Ensure network policy allows port 22222
- Check RTPEngine service status

### Issue: Health Check Failures
**Symptoms**: Readiness/liveness probe failures

**Check**:
```bash
# Test health endpoint manually
kubectl exec -n voice-ferry $POD_NAME -- curl -v http://localhost:8080/health
```

**Solution**:
- Ensure probes use `/health` endpoint (not `/healthz/*`)
- Check if health server is binding to correct port
- Verify no firewall blocking port 8080

## 📊 Monitoring Setup

### Key Metrics to Monitor
- Pod status and restart counts
- Health endpoint response times
- RTPEngine connectivity success rate
- Memory and CPU utilization
- Active SIP sessions

### Log Monitoring
```bash
# Real-time logs
kubectl logs -f -n voice-ferry -l app=voice-ferry

# Error filtering
kubectl logs -n voice-ferry -l app=voice-ferry | grep -i error

# RTPEngine status
kubectl logs -n voice-ferry -l app=voice-ferry | grep "RTPEngine ping"
```

### Alerting Recommendations
- Pod not ready for > 5 minutes
- Health check failure rate > 5%
- RTPEngine connectivity failure
- Memory usage > 80%
- High restart count

## 🔄 Update Procedures

### Rolling Update
```bash
# Update image
kubectl set image deployment/voice-ferry voice-ferry=2bleere/voice-ferry:arm64-latest -n voice-ferry

# Monitor rollout
kubectl rollout status deployment/voice-ferry -n voice-ferry

# Verify health after update
kubectl wait --for=condition=Ready pod -l app=voice-ferry -n voice-ferry --timeout=300s
```

### Configuration Update
```bash
# Edit configuration
kubectl edit configmap voice-ferry-config -n voice-ferry

# Restart to pick up changes
kubectl rollout restart deployment/voice-ferry -n voice-ferry
```

### Rollback Procedure
```bash
# Check rollout history
kubectl rollout history deployment/voice-ferry -n voice-ferry

# Rollback to previous version
kubectl rollout undo deployment/voice-ferry -n voice-ferry

# Monitor rollback
kubectl rollout status deployment/voice-ferry -n voice-ferry
```

## 🔐 Security Checklist

### Network Security
- [ ] Network policies restrict unnecessary traffic
- [ ] Egress limited to required services only
- [ ] Ingress controlled for SIP and management traffic
- [ ] TLS enabled for sensitive communications

### Pod Security
- [ ] Containers run as non-root user
- [ ] Security contexts properly configured
- [ ] Unnecessary capabilities dropped
- [ ] Read-only root filesystem where possible

### Secrets Management
- [ ] JWT signing key stored as Kubernetes secret
- [ ] TLS certificates properly secured
- [ ] No sensitive data in ConfigMaps
- [ ] RBAC limiting access to secrets

## 📈 Performance Optimization

### Resource Tuning
```yaml
# Recommended production resources
resources:
  requests:
    memory: "256Mi"  # Increase from 128Mi for higher load
    cpu: "200m"      # Increase from 100m for higher load
  limits:
    memory: "1Gi"    # Increase from 512Mi for peak usage
    cpu: "1000m"     # Increase from 500m for peak usage
```

### Scaling Configuration
```yaml
# HPA for automatic scaling
spec:
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

## 🆘 Emergency Procedures

### Emergency Rollback
```bash
# Quick rollback to last known good version
kubectl rollout undo deployment/voice-ferry -n voice-ferry --to-revision=1
```

### Service Recovery
```bash
# Restart all Voice Ferry pods
kubectl delete pods -l app=voice-ferry -n voice-ferry

# Force recreate deployment
kubectl rollout restart deployment/voice-ferry -n voice-ferry
```

### Debug Mode
```bash
# Enable debug logging
kubectl set env deployment/voice-ferry LOG_LEVEL=debug -n voice-ferry

# Access debug endpoints
kubectl port-forward svc/voice-ferry-metrics 9090:9090 -n voice-ferry
# Access http://localhost:9090/debug/pprof/
```

---

## ✅ Sign-off

- [ ] All pre-deployment requirements met
- [ ] Deployment completed successfully
- [ ] Post-deployment verification passed
- [ ] Monitoring and alerting configured
- [ ] Emergency procedures documented and tested
- [ ] Team trained on operational procedures

**Deployment Approved By**: _________________  
**Date**: _________________  
**Environment**: _________________

---

**This checklist ensures consistent, reliable Voice Ferry deployments across all environments.**
