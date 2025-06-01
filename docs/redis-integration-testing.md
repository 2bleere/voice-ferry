# Redis Integration Testing Guide

This guide provides comprehensive instructions for testing Redis integration in Voice Ferry, including both single instance and cluster configurations.

## Overview

Voice Ferry uses Redis for:
- Session state management
- Concurrent session limit tracking
- Caching routing decisions
- Storing temporary call data

## Testing Prerequisites

- Python 3.7+ with redis library
- Access to Redis instance or cluster
- Voice Ferry deployment with Redis connectivity

## Installation

Install the required Python dependencies:

```bash
pip install redis
```

## Redis Integration Test Script

Use the provided test script to verify Redis connectivity and functionality:

```bash
# Run the Redis integration test
python test_redis_integration.py
```

The test script performs the following checks:

### 1. Connection Test
- Verifies basic Redis connectivity
- Tests authentication if configured
- Validates network connectivity

### 2. Basic Operations Test
- SET/GET operations
- Key expiration testing
- Data type operations (strings, hashes, lists)

### 3. Session Management Test
- User session tracking
- Concurrent session limits
- Session cleanup

### 4. Cluster-Specific Tests (if applicable)
- Cross-node key distribution
- Failover behavior
- Cluster health monitoring

## Manual Testing Procedures

### Single Redis Instance Testing

1. **Connection Verification**
   ```bash
   kubectl exec -it deployment/redis -n voice-ferry -- redis-cli ping
   ```

2. **Set Test Data**
   ```bash
   kubectl exec -it deployment/redis -n voice-ferry -- redis-cli set test_key "test_value"
   kubectl exec -it deployment/redis -n voice-ferry -- redis-cli get test_key
   ```

3. **Session Limit Testing**
   ```bash
   # Simulate user session
   kubectl exec -it deployment/redis -n voice-ferry -- redis-cli \
     hset "user:alice@example.com:sessions" "call1" "active"
   
   # Check session count
   kubectl exec -it deployment/redis -n voice-ferry -- redis-cli \
     hlen "user:alice@example.com:sessions"
   ```

### Redis Cluster Testing

1. **Cluster Status Check**
   ```bash
   kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli cluster info
   kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli cluster nodes
   ```

2. **Cross-Node Data Distribution**
   ```bash
   # Set keys on different slots
   for i in {1..10}; do
     kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli \
       set "test_key_$i" "value_$i"
   done
   
   # Verify distribution
   kubectl exec -it redis-cluster-0 -n voice-ferry -- redis-cli \
     cluster keyslot "test_key_1"
   ```

3. **Failover Testing**
   ```bash
   # Simulate node failure
   kubectl delete pod redis-cluster-0 -n voice-ferry
   
   # Check cluster reorganization
   kubectl exec -it redis-cluster-1 -n voice-ferry -- redis-cli cluster info
   ```

## Performance Testing

### Load Testing with redis-benchmark

Test Redis performance under load:

```bash
# Basic performance test
kubectl exec -it deployment/redis -n voice-ferry -- \
  redis-benchmark -h localhost -p 6379 -n 10000 -c 50

# Session-specific performance test
kubectl exec -it deployment/redis -n voice-ferry -- \
  redis-benchmark -h localhost -p 6379 -n 10000 -c 50 -t set,get
```

### Cluster Performance Testing

For Redis cluster deployments:

```bash
# Cluster-aware benchmark
kubectl exec -it redis-cluster-0 -n voice-ferry -- \
  redis-benchmark -h redis-cluster-0 -p 6379 -n 10000 -c 50 --cluster
```

## Integration Test Results Analysis

### Expected Outcomes

1. **Connection Test**: Should establish connection without errors
2. **Basic Operations**: All SET/GET operations should succeed
3. **Session Management**: User sessions should be tracked correctly
4. **Performance**: Latency should be < 1ms for local operations

### Common Issues and Solutions

#### Connection Failures
```
Error: Could not connect to Redis at redis:6379
```
**Solution**: Check service name resolution and network policies

#### Authentication Errors
```
Error: NOAUTH Authentication required
```
**Solution**: Configure Redis password in Voice Ferry config

#### Cluster Split-Brain
```
Error: CLUSTERDOWN The cluster is down
```
**Solution**: Check cluster quorum and node connectivity

## Monitoring Redis Integration

### Key Metrics to Monitor

1. **Connection Health**
   - Active connections
   - Connection errors
   - Reconnection attempts

2. **Session Metrics**
   - Active sessions per user
   - Session creation/deletion rate
   - Session limit violations

3. **Performance Metrics**
   - Command latency
   - Memory usage
   - Key expiration rate

### Grafana Dashboard Queries

Monitor Redis integration with these Prometheus queries:

```promql
# Redis connection status
redis_connected_clients{instance="redis:6379"}

# Session limit violations
increase(session_limit_rejections_total[5m])

# Active sessions per user
redis_hash_entries{key=~"user:.*:sessions"}
```

## Automated Testing in CI/CD

Include Redis integration testing in your deployment pipeline:

```yaml
# .github/workflows/test-redis.yml
name: Redis Integration Test
on: [push, pull_request]

jobs:
  test-redis:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
    
    steps:
    - uses: actions/checkout@v3
    - name: Setup Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.9'
    
    - name: Install dependencies
      run: pip install redis
    
    - name: Run Redis integration tests
      run: python test_redis_integration.py
      env:
        REDIS_HOST: localhost
        REDIS_PORT: 6379
```

## Production Validation Checklist

Before deploying to production, verify:

- [ ] Redis connectivity from Voice Ferry pods
- [ ] Session creation and cleanup working
- [ ] Concurrent session limits enforced
- [ ] Redis cluster failover tested (if using cluster)
- [ ] Performance meets requirements
- [ ] Monitoring and alerting configured
- [ ] Backup and recovery procedures tested

## Troubleshooting

### Debug Redis Connectivity

```bash
# Check Redis logs
kubectl logs deployment/redis -n voice-ferry

# Check Voice Ferry logs for Redis errors
kubectl logs deployment/sip-b2bua -n voice-ferry | grep -i redis

# Test connectivity from Voice Ferry pod
kubectl exec -it deployment/sip-b2bua -n voice-ferry -- \
  nc -zv redis 6379
```

### Redis Cluster Debugging

```bash
# Check cluster status
kubectl exec -it redis-cluster-0 -n voice-ferry -- \
  redis-cli --cluster check redis-cluster-0:6379

# View cluster configuration
kubectl exec -it redis-cluster-0 -n voice-ferry -- \
  redis-cli cluster slots
```

## Performance Tuning

### Redis Configuration Optimization

For production workloads, consider these Redis optimizations:

```conf
# redis.conf optimizations
maxmemory 1gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
tcp-keepalive 300
timeout 0
```

### Voice Ferry Redis Client Tuning

Configure Redis client settings in Voice Ferry:

```yaml
redis:
  pool_size: 10
  connection_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  idle_timeout: 300s
  max_retries: 3
```

## Security Considerations

### Redis Security Best Practices

1. **Authentication**: Always use Redis AUTH
2. **Network Security**: Restrict Redis access to Voice Ferry pods only
3. **TLS Encryption**: Enable TLS for Redis connections in production
4. **Regular Updates**: Keep Redis version updated

### Kubernetes Security

```yaml
# Network policy to restrict Redis access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: redis-access-policy
spec:
  podSelector:
    matchLabels:
      app: redis
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: sip-b2bua
    ports:
    - protocol: TCP
      port: 6379
```
