# Voice Ferry Production Deployment Guide

## Prerequisites

- Kubernetes cluster with sufficient resources
- `kubectl` configured for your cluster
- Storage class configured (e.g., `longhorn`, `fast-ssd`)
- Namespace with appropriate RBAC permissions

## Resource Requirements

### Minimum Production Resources

| Component | CPU Request | Memory Request | Storage | Replicas |
|-----------|-------------|----------------|---------|----------|
| **etcd** | 200m × 3 | 256Mi × 3 | 2Gi × 3 | 3 |
| **Redis Cluster** | 100m × 6 | 128Mi × 6 | 1Gi × 6 | 6 |
| **SIP B2BUA** | 100m × 2 | 128Mi × 2 | - | 2 |
| **RTPEngine** | 100m × 1 | 128Mi × 1 | - | 1 |

**Total**: ~2.2 CPU cores, ~2.5GB RAM, ~12GB storage

## Deployment Steps

### Step 1: Create Namespace and Secrets

```bash
# Create namespace
kubectl create namespace voice-ferry

# Create JWT signing key secret
kubectl create secret generic b2bua-secrets \
  --from-literal=jwt-signing-key="$(openssl rand -hex 32)" \
  -n voice-ferry

# Create TLS certificates (if using TLS)
kubectl create secret tls voice-ferry-tls \
  --cert=/path/to/tls.crt \
  --key=/path/to/tls.key \
  -n voice-ferry
```

### Step 2: Deploy Infrastructure Dependencies

```bash
# Deploy etcd cluster and RTPEngine
kubectl apply -f dependencies.yaml

# Wait for etcd to be ready
kubectl wait --for=condition=Ready pod -l app=etcd -n voice-ferry --timeout=300s

# Verify etcd cluster health
kubectl exec -it etcd-0 -n voice-ferry -- etcdctl --endpoints=http://etcd:2379 endpoint health
```

### Step 3: Deploy Redis Cluster

```bash
# Deploy Redis cluster
kubectl apply -f redis-cluster.yaml

# Wait for Redis pods to be ready
kubectl wait --for=condition=Ready pod -l app=redis-cluster -n voice-ferry --timeout=300s

# Wait for cluster initialization to complete
kubectl wait --for=condition=Complete job/redis-cluster-init -n voice-ferry --timeout=600s

# Verify Redis cluster status
kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli cluster info
```

### Step 4: Deploy Main Application

```bash
# Deploy SIP B2BUA application
kubectl apply -f sip-b2bua.yaml

# Wait for deployment to be ready
kubectl wait --for=condition=Available deployment/sip-b2bua -n voice-ferry --timeout=300s

# Deploy production configuration
kubectl apply -f voice-ferry-production.yaml
```

## Verification Steps

### Health Check Commands

```bash
# Check all pods status
kubectl get pods -n voice-ferry

# Check services
kubectl get svc -n voice-ferry

# Check persistent volumes
kubectl get pvc -n voice-ferry

# Application health checks
kubectl exec -it deployment/sip-b2bua -n voice-ferry -- curl -f http://localhost:8080/health/live
kubectl exec -it deployment/sip-b2bua -n voice-ferry -- curl -f http://localhost:8080/health/ready
```

### Dependency Verification

```bash
# Verify etcd cluster
kubectl exec -it etcd-0 -n voice-ferry -- etcdctl --endpoints=http://etcd:2379 member list

# Verify Redis cluster
kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli cluster nodes

# Check RTPEngine
kubectl exec -it deployment/sip-b2bua -n voice-ferry -- nc -z rtpengine 22222
```

## Monitoring Setup

### Enable Prometheus Metrics

```bash
# Apply monitoring configuration
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: voice-ferry-metrics
  namespace: voice-ferry
  labels:
    app: sip-b2bua
    component: metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
spec:
  ports:
  - port: 8080
    name: metrics
  selector:
    app: sip-b2bua
EOF
```

## Scaling Guidelines

### Horizontal Scaling

```bash
# Scale SIP B2BUA replicas
kubectl scale deployment sip-b2bua --replicas=4 -n voice-ferry

# Scale Redis cluster (requires cluster reconfiguration)
# Note: This is a complex operation requiring careful planning

# etcd scaling (StatefulSet)
kubectl patch statefulset etcd -n voice-ferry -p '{"spec":{"replicas":5}}'
```

### Vertical Scaling

```bash
# Update resource limits
kubectl patch deployment sip-b2bua -n voice-ferry -p '{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "sip-b2bua",
          "resources": {
            "requests": {"memory": "256Mi", "cpu": "200m"},
            "limits": {"memory": "1Gi", "cpu": "1000m"}
          }
        }]
      }
    }
  }
}'
```

## Backup Strategy

### etcd Backup

```bash
# Create etcd snapshot
kubectl exec etcd-0 -n voice-ferry -- etcdctl \
  --endpoints=http://etcd:2379 \
  snapshot save /tmp/backup-$(date +%Y%m%d-%H%M%S).db

# Copy backup from pod
kubectl cp voice-ferry/etcd-0:/tmp/backup-$(date +%Y%m%d-%H%M%S).db ./etcd-backup.db
```

### Redis Backup

```bash
# Create Redis backup
kubectl exec redis-cluster-0 -n voice-ferry -- redis-cli BGSAVE

# Copy RDB file
kubectl cp voice-ferry/redis-cluster-0:/data/dump.rdb ./redis-backup.rdb
```

## Troubleshooting

### Common Issues

1. **Init containers failing**: Check dependency services are running
2. **Redis cluster not forming**: Verify all 6 pods are ready and network connectivity
3. **etcd cluster issues**: Check persistent volume permissions and storage class
4. **Application startup failures**: Verify all secrets and config maps are present

### Debug Commands

```bash
# Check application logs
kubectl logs deployment/sip-b2bua -n voice-ferry --follow

# Check init container logs
kubectl logs pod-name -c wait-for-redis -n voice-ferry

# Describe problematic resources
kubectl describe pod pod-name -n voice-ferry
kubectl describe pvc pvc-name -n voice-ferry

# Enter debug pod for network testing
kubectl run debug --image=busybox:1.35 -it --rm --restart=Never -n voice-ferry -- sh
```

## Security Considerations

### Network Policies

```bash
# Apply network policies to restrict traffic
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: voice-ferry-network-policy
  namespace: voice-ferry
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: voice-ferry
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: voice-ferry
  - to: []
    ports:
    - protocol: UDP
      port: 53
EOF
```

### RBAC

```bash
# Create service account with minimal permissions
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sip-b2bua
  namespace: voice-ferry
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sip-b2bua-role
  namespace: voice-ferry
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sip-b2bua-binding
  namespace: voice-ferry
subjects:
- kind: ServiceAccount
  name: sip-b2bua
  namespace: voice-ferry
roleRef:
  kind: Role
  name: sip-b2bua-role
  apiGroup: rbac.authorization.k8s.io
EOF
```

## Performance Tuning

### Resource Optimization

- Monitor CPU and memory usage with metrics
- Adjust resource requests/limits based on actual usage
- Consider node affinity for performance-critical components
- Use local SSDs for etcd data persistence

### Network Optimization

- Enable network policies for security without performance impact
- Use headless services for internal communication
- Consider pod anti-affinity for high availability

## Maintenance

### Rolling Updates

```bash
# Update application image
kubectl set image deployment/sip-b2bua sip-b2bua=ghcr.io/voice-ferry-c4/sip-b2bua:v1.1.0 -n voice-ferry

# Monitor rollout
kubectl rollout status deployment/sip-b2bua -n voice-ferry

# Rollback if needed
kubectl rollout undo deployment/sip-b2bua -n voice-ferry
```

### Configuration Updates

```bash
# Update config map
kubectl patch configmap voice-ferry-config -n voice-ferry --patch '{"data":{"config.yaml":"<new-config>"}}'

# Restart deployment to pick up new config
kubectl rollout restart deployment/sip-b2bua -n voice-ferry
```
