# Voice Ferry Deployment Strategy

## Redis Deployment Decision

### Production Recommendation: Use Redis Cluster

**Rationale:**
- **High Availability**: Redis cluster provides automatic failover
- **Scalability**: Can handle higher concurrent session loads  
- **Data Distribution**: Spreads load across multiple nodes
- **Fault Tolerance**: Can survive node failures without data loss

### Deployment Options Comparison

| Feature | Single Redis (dependencies.yaml) | Redis Cluster (redis-cluster.yaml) |
|---------|-----------------------------------|-----------------------------------|
| **Replicas** | 1 | 6 (3 masters + 3 replicas) |
| **Storage** | 1GB | 1GB per node (6GB total) |
| **Availability** | Single point of failure | High availability |
| **Performance** | Limited by single node | Distributed performance |
| **Use Case** | Development/Small scale | Production/High scale |

## Service Naming Consistency Fix

**Issue**: Redis cluster service is named "redis-cluster" but applications expect "redis"

**Solution**: Update redis-cluster.yaml service name to maintain compatibility.

## Deployment Order Strategy

### Recommended Deployment Sequence:

1. **Infrastructure Layer** (dependencies)
   ```bash
   kubectl apply -f dependencies.yaml  # etcd StatefulSet + RTPEngine
   ```

2. **Data Layer** (Redis cluster)
   ```bash
   kubectl apply -f redis-cluster.yaml  # Redis cluster
   ```

3. **Application Layer** 
   ```bash
   kubectl apply -f sip-b2bua.yaml           # Main application
   kubectl apply -f voice-ferry-production.yaml  # Production config
   ```

### Dependency Management

- Add init containers for proper startup ordering
- Implement dependency health checks
- Use readiness probes to prevent premature traffic routing

## Production Optimizations

### Resource Allocation
- **etcd**: 3 replicas × 2GB storage = 6GB total
- **Redis Cluster**: 6 nodes × 1GB storage = 6GB total  
- **Total Storage**: ~12GB for data persistence

### Network Policies
- Restrict inter-pod communication to required services only
- Implement ingress/egress policies for security

### Monitoring Integration
- Enable Prometheus metrics collection
- Configure alerts for dependency health
- Set up grafana dashboards for operational visibility
