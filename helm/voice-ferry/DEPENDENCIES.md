# Voice Ferry Dependencies Installation Guide

This document provides instructions for installing the external dependencies required by Voice Ferry.

## Overview

Voice Ferry requires several external services for full functionality:
- **Redis**: Session storage and caching
- **etcd**: Configuration management and service discovery
- **Prometheus**: Metrics collection and monitoring
- **Grafana**: Metrics visualization and dashboards

## Quick Installation Commands

### 1. Add Helm Repositories

```bash
# Add required repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```

### 2. Install Redis Cluster

```bash
# Install Redis with high availability
helm install redis bitnami/redis \
  --namespace voice-ferry \
  --create-namespace \
  --set auth.enabled=true \
  --set auth.password="your-redis-password" \
  --set cluster.enabled=true \
  --set cluster.slaveCount=3 \
  --set persistence.enabled=true \
  --set persistence.size=20Gi
```

### 3. Install etcd Cluster

```bash
# Install etcd cluster
helm install etcd bitnami/etcd \
  --namespace voice-ferry \
  --set replicaCount=3 \
  --set auth.rbac.enabled=true \
  --set auth.rbac.rootPassword="your-etcd-password" \
  --set persistence.enabled=true \
  --set persistence.size=10Gi
```

### 4. Install Prometheus Stack

```bash
# Install Prometheus with operator
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set prometheus.prometheusSpec.retention=30d \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi
```

### 5. Install Grafana (if not included with Prometheus stack)

```bash
# Install standalone Grafana (optional if using kube-prometheus-stack)
helm install grafana grafana/grafana \
  --namespace monitoring \
  --set persistence.enabled=true \
  --set persistence.size=10Gi \
  --set adminPassword="your-grafana-password"
```

## Configuration Values

### For Development Environment

Use these minimal configurations for development:

```yaml
# values-dev-deps.yaml
redis:
  enabled: true
  auth:
    enabled: false
  cluster:
    enabled: false
  persistence:
    enabled: false

etcd:
  enabled: true
  auth:
    rbac:
      enabled: false
  persistence:
    enabled: false

monitoring:
  prometheus:
    enabled: true
  grafana:
    enabled: true
```

### For Production Environment

Use these production-ready configurations:

```yaml
# values-prod-deps.yaml
redis:
  enabled: true
  auth:
    enabled: true
    password: "secure-redis-password"
  cluster:
    enabled: true
    slaveCount: 3
  persistence:
    enabled: true
    size: "50Gi"
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "1000m"

etcd:
  enabled: true
  replicaCount: 3
  auth:
    rbac:
      enabled: true
      rootPassword: "secure-etcd-password"
  persistence:
    enabled: true
    size: "20Gi"
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "1Gi"
      cpu: "1000m"
```

## Installation Order

1. **Install dependencies first** (Redis, etcd, Prometheus, Grafana)
2. **Wait for dependencies to be ready**
3. **Install Voice Ferry** with appropriate values file

## Verification Commands

```bash
# Check Redis
kubectl get pods -n voice-ferry -l app.kubernetes.io/name=redis

# Check etcd
kubectl get pods -n voice-ferry -l app.kubernetes.io/name=etcd

# Check Prometheus
kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus

# Check Grafana
kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana
```

## Connecting Voice Ferry to Dependencies

Voice Ferry will automatically connect to these services using the following service names:

- **Redis**: `redis-master.voice-ferry.svc.cluster.local:6379`
- **etcd**: `etcd.voice-ferry.svc.cluster.local:2379`
- **Prometheus**: `prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090`
- **Grafana**: `grafana.monitoring.svc.cluster.local:3000`

## Troubleshooting

### Redis Connection Issues
```bash
# Test Redis connectivity
kubectl run redis-test --rm -it --image=redis:alpine -- redis-cli -h redis-master.voice-ferry.svc.cluster.local ping
```

### etcd Connection Issues
```bash
# Test etcd connectivity
kubectl run etcd-test --rm -it --image=bitnami/etcd:latest -- etcdctl --endpoints=http://etcd.voice-ferry.svc.cluster.local:2379 endpoint health
```

### Monitoring Issues
```bash
# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Open http://localhost:9090/targets

# Check Grafana dashboards
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Open http://localhost:3000 (admin/prom-operator)
```
