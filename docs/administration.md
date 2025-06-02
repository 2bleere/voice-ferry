# Voice Ferry - Administration & Troubleshooting Guide

## Table of Contents

1. [Installation & Deployment](#installation--deployment)
2. [Command Line Interface](#command-line-interface)
3. [Configuration Management](#configuration-management)
4. [Monitoring & Health Checks](#monitoring--health-checks)
5. [Troubleshooting](#troubleshooting)
6. [Performance Tuning](#performance-tuning)
7. [Security Management](#security-management)
8. [Backup & Recovery](#backup--recovery)

## Installation & Deployment

### Prerequisites

- Go 1.24.3+ (for source builds)
- Docker 20.10+ (for containerized deployment)
- Kubernetes 1.25+ (for Kubernetes deployment)
- Redis 6.0+ (for session management)
- etcd 3.5+ (for configuration management)
- rtpengine (for media handling)

### Quick Start

#### Docker Deployment (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/2bleere/voice-ferry:latest

# Run with basic configuration
docker run -d \
  --name voice-ferry \
  -p 5060:5060/udp \
  -p 50051:50051 \
  -p 8080:8080 \
  -e SIP_HOST=0.0.0.0 \
  -e SIP_PORT=5060 \
  ghcr.io/2bleere/voice-ferry:latest

# Run with custom configuration
docker run -d \
  --name voice-ferry \
  -p 5060:5060/udp \
  -p 50051:50051 \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/b2bua/config.yaml:ro \
  ghcr.io/2bleere/voice-ferry:latest
```

#### Kubernetes Deployment

```bash
# Deploy with Helm (recommended)
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --create-namespace \
  --values values-production.yaml

# Or apply manifests directly
kubectl apply -f deployments/kubernetes/
```

#### Source Build

```bash
# Clone repository
git clone https://github.com/2bleere/voice-ferry.git
cd voice-ferry

# Build binary
make build

# Install systemd service (Linux)
sudo make install

# Start service
sudo systemctl start voice-ferry
sudo systemctl enable voice-ferry
```

## Command Line Interface

### Binary Usage

```bash
# Basic usage
./b2bua-server [options]

# Available options:
./b2bua-server -h
Usage of ./b2bua-server:
  -config string
        Path to configuration file (default "/etc/voice-ferry/config.yaml")
  -debug
        Enable debug logging
  -grpc-port int
        gRPC API port (default 50051)
  -sip-port int
        SIP listening port (default 5060)
  -version
        Show version information
```

### Configuration Management

#### Validate Configuration

```bash
# Validate configuration file
./b2bua-server -config /etc/voice-ferry/config.yaml -validate

# Test configuration without starting services
./b2bua-server -config /etc/voice-ferry/config.yaml -test
```

#### Environment Variable Override

```bash
# Override specific configuration values
export SIP_HOST="192.168.1.100"
export SIP_PORT="5060"
export DEBUG="true"
export JWT_SIGNING_KEY="your-secret-key"

./b2bua-server -config /etc/voice-ferry/config.yaml
```

### Service Management (systemd)

```bash
# Service control
sudo systemctl start voice-ferry
sudo systemctl stop voice-ferry
sudo systemctl restart voice-ferry
sudo systemctl reload voice-ferry  # Graceful config reload

# Service status
sudo systemctl status voice-ferry
sudo systemctl is-active voice-ferry
sudo systemctl is-enabled voice-ferry

# Enable/disable service
sudo systemctl enable voice-ferry
sudo systemctl disable voice-ferry

# View logs
sudo journalctl -u voice-ferry -f
sudo journalctl -u voice-ferry --since="1 hour ago"
```

### Container Management

```bash
# Container lifecycle
docker start voice-ferry
docker stop voice-ferry
docker restart voice-ferry

# View container logs
docker logs voice-ferry -f
docker logs voice-ferry --since=1h

# Execute commands in container
docker exec -it voice-ferry /bin/sh
docker exec voice-ferry /usr/local/bin/b2bua-server -version

# Container resource usage
docker stats voice-ferry

# Container health check
docker inspect voice-ferry | jq '.[0].State.Health'
```

## Configuration Management

### Configuration File Locations

```bash
# Default locations (in order of precedence)
./config.yaml                    # Current directory
/etc/b2bua/config.yaml           # System config
~/.config/voice-ferry/config.yaml # User config
```

### Dynamic Configuration Updates

#### Using gRPC API

```bash
# Install grpcurl for API testing
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Update SIP configuration
grpcurl -plaintext \
  -d '{"sip_config": {"host": "0.0.0.0", "port": 5060}}' \
  localhost:50051 \
  b2bua.v1.ConfigurationService/UpdateSipConfig

# Update Redis configuration
grpcurl -plaintext \
  -d '{"redis_config": {"addr": "redis:6379", "db": 0}}' \
  localhost:50051 \
  b2bua.v1.ConfigurationService/UpdateRedisConfig

# Reload configuration
grpcurl -plaintext \
  localhost:50051 \
  b2bua.v1.ConfigurationService/ReloadConfig
```

#### Using etcd

```bash
# Install etcdctl
export ETCDCTL_API=3

# Update configuration
etcdctl put /voice-ferry/config/sip/host "0.0.0.0"
etcdctl put /voice-ferry/config/sip/port "5060"

# Get configuration
etcdctl get /voice-ferry/config/sip/host
etcdctl get --prefix /voice-ferry/config/

# Watch for configuration changes
etcdctl watch --prefix /voice-ferry/config/
```

### Configuration Backup & Restore

```bash
# Backup current configuration
curl -s http://localhost:8080/api/config/export > config-backup.yaml

# Restore configuration
curl -X POST \
  -H "Content-Type: application/yaml" \
  --data-binary @config-backup.yaml \
  http://localhost:8080/api/config/import

# Backup etcd configuration
etcdctl get --prefix /voice-ferry/config/ > etcd-backup.txt

# Restore etcd configuration
cat etcd-backup.txt | etcdctl put --prefix /voice-ferry/config/
```

## Monitoring & Health Checks

### Health Endpoints

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health check
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/live

# Component-specific health
curl http://localhost:8080/health/sip
curl http://localhost:8080/health/redis
curl http://localhost:8080/health/etcd
curl http://localhost:8080/health/rtpengine
```

### Metrics Collection

```bash
# Prometheus metrics
curl http://localhost:8080/metrics

# System metrics
curl http://localhost:8080/api/metrics/system

# SIP metrics
curl http://localhost:8080/api/metrics/sip

# Call metrics
curl http://localhost:8080/api/metrics/calls
```

### Log Management

```bash
# Follow logs in real-time
tail -f /var/log/voice-ferry/b2bua.log

# Search logs
grep "ERROR" /var/log/voice-ferry/b2bua.log
grep "call_id" /var/log/voice-ferry/b2bua.log

# Rotate logs
logrotate -f /etc/logrotate.d/voice-ferry

# Docker logs
docker logs voice-ferry 2>&1 | grep ERROR
docker logs voice-ferry --since="2023-01-01T00:00:00Z"

# Kubernetes logs
kubectl logs -n voice-ferry deployment/voice-ferry -f
kubectl logs -n voice-ferry -l app=voice-ferry --since=1h
```

### Performance Monitoring

```bash
# System resource monitoring
top -p $(pgrep b2bua)
htop -p $(pgrep b2bua)

# Memory usage
ps aux | grep b2bua
pmap $(pgrep b2bua)

# Network connections
netstat -tlnp | grep :5060
ss -tlnp | grep :5060

# Call statistics
grpcurl -plaintext localhost:50051 b2bua.v1.StatusService/GetSystemStatus
grpcurl -plaintext localhost:50051 b2bua.v1.StatusService/GetMetrics
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check configuration syntax
./b2bua-server -config /etc/voice-ferry/config.yaml -validate

# Check port availability
netstat -tlnp | grep :5060
lsof -i :5060

# Check file permissions
ls -la /etc/voice-ferry/config.yaml
ls -la /var/log/voice-ferry/

# Check systemd status
sudo systemctl status voice-ferry
sudo journalctl -u voice-ferry --no-pager
```

#### SIP Registration Issues

```bash
# Test SIP connectivity
sipsak -s sip:user@localhost:5060

# Check SIP logs
grep "REGISTER" /var/log/voice-ferry/b2bua.log
grep "401\|403" /var/log/voice-ferry/b2bua.log

# Verify authentication configuration
curl http://localhost:8080/api/config/sip/auth

# Test with sipgrep
sipgrep -i any -p 5060
```

#### Database Connection Issues

```bash
# Test Redis connection
redis-cli -h localhost -p 6379 ping

# Test etcd connection
etcdctl --endpoints=localhost:2379 endpoint health

# Check connection pools
curl http://localhost:8080/api/metrics/system | grep pool

# Reset connections
grpcurl -plaintext localhost:50051 b2bua.v1.StatusService/ResetConnections
```

#### Media Issues

```bash
# Check rtpengine status
rtpengine-ctl ping

# Test RTP connectivity
rtpengine-ctl list
rtpengine-ctl query

# Check media statistics
curl http://localhost:8080/api/metrics/media

# Debug RTP flow
tcpdump -i any -n port 10000-20000
```

### Debugging Tools

#### Enable Debug Logging

```bash
# Temporary debug mode
kill -USR1 $(pgrep b2bua)  # Enable debug
kill -USR2 $(pgrep b2bua)  # Disable debug

# Permanent debug mode
echo "debug: true" >> /etc/b2bua/config.yaml
sudo systemctl reload voice-ferry
```

#### Network Debugging

```bash
# Capture SIP traffic
tcpdump -i any -n -s 0 port 5060 -w sip-capture.pcap

# Analyze with wireshark
wireshark sip-capture.pcap

# Test network connectivity
telnet localhost 5060
nc -u localhost 5060
```

#### Performance Profiling

```bash
# CPU profiling
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profiling
curl http://localhost:8080/debug/pprof/heap > mem.prof
go tool pprof mem.prof

# Goroutine analysis
curl http://localhost:8080/debug/pprof/goroutine > goroutines.prof
go tool pprof goroutines.prof
```

### Log Analysis

#### Common Log Patterns

```bash
# Failed calls
grep "call_failed" /var/log/voice-ferry/b2bua.log

# High error rates
grep -c "ERROR" /var/log/voice-ferry/b2bua.log

# Authentication failures
grep "auth_failed" /var/log/voice-ferry/b2bua.log

# Resource exhaustion
grep "too_many\|limit_exceeded" /var/log/voice-ferry/b2bua.log
```

#### Log Rotation and Cleanup

```bash
# Manual log rotation
sudo logrotate -f /etc/logrotate.d/voice-ferry

# Clean old logs
find /var/log/voice-ferry/ -name "*.log.*" -mtime +7 -delete

# Compress logs
gzip /var/log/voice-ferry/*.log.1
```

## Performance Tuning

### System Optimization

```bash
# Increase file descriptor limits
echo "voice-ferry soft nofile 65536" >> /etc/security/limits.conf
echo "voice-ferry hard nofile 65536" >> /etc/security/limits.conf

# Optimize network settings
echo "net.core.rmem_max = 268435456" >> /etc/sysctl.conf
echo "net.core.wmem_max = 268435456" >> /etc/sysctl.conf
echo "net.ipv4.udp_mem = 102400 873800 16777216" >> /etc/sysctl.conf
sysctl -p
```

### Application Tuning

```bash
# Adjust connection pools
curl -X POST http://localhost:8080/api/config/redis \
  -d '{"pool_size": 20, "max_retries": 3}'

# Tune SIP timeouts
curl -X POST http://localhost:8080/api/config/sip \
  -d '{"timeouts": {"transaction": "32s", "dialog": "1800s"}}'

# Optimize garbage collection
export GOGC=100
export GOMEMLIMIT=1GiB
```

### Load Testing

```bash
# Install SIPp for load testing
apt-get install sipp

# Basic load test
sipp -sn uac -r 10 -l 100 localhost:5060

# Custom scenario test
sipp -sf custom-scenario.xml -r 20 -l 200 localhost:5060

# Monitor during load test
watch 'curl -s http://localhost:8080/api/metrics/system | jq .calls'
```

## Security Management

### TLS Configuration

```bash
# Generate self-signed certificates
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Test TLS connection
openssl s_client -connect localhost:5061 -servername voice-ferry

# Verify certificate
openssl x509 -in cert.pem -text -noout
```

### JWT Token Management

```bash
# Generate signing key
openssl rand -base64 32

# Create admin token
curl -X POST http://localhost:8080/api/auth/login \
  -d '{"username": "admin", "password": "password"}'

# Validate token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/status
```

### Access Control

```bash
# Update IP whitelist
curl -X POST http://localhost:8080/api/config/security/ip-acl \
  -d '{"whitelist": ["192.168.1.0/24", "10.0.0.0/8"]}'

# Block specific IPs
curl -X POST http://localhost:8080/api/config/security/ip-acl \
  -d '{"blacklist": ["192.168.1.100", "10.0.0.50"]}'

# View current ACLs
curl http://localhost:8080/api/config/security/ip-acl
```

## Backup & Recovery

### Configuration Backup

```bash
# Create backup script
cat > backup-config.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/voice-ferry"

mkdir -p $BACKUP_DIR

# Backup configuration files
cp /etc/b2bua/config.yaml $BACKUP_DIR/config_$DATE.yaml

# Backup etcd data
etcdctl snapshot save $BACKUP_DIR/etcd_$DATE.db

# Backup certificates
tar -czf $BACKUP_DIR/certs_$DATE.tar.gz /etc/ssl/voice-ferry/

echo "Backup completed: $BACKUP_DIR"
EOF

chmod +x backup-config.sh
```

### Disaster Recovery

```bash
# Restore configuration
cp /var/backups/voice-ferry/config_20231201_120000.yaml /etc/b2bua/config.yaml

# Restore etcd data
etcdctl snapshot restore /var/backups/voice-ferry/etcd_20231201_120000.db

# Restore certificates
tar -xzf /var/backups/voice-ferry/certs_20231201_120000.tar.gz -C /

# Restart services
sudo systemctl restart voice-ferry
```

### Database Recovery

```bash
# Redis backup
redis-cli --rdb /var/backups/voice-ferry/redis-dump.rdb

# Redis restore
redis-cli shutdown
cp /var/backups/voice-ferry/redis-dump.rdb /var/lib/redis/dump.rdb
sudo systemctl start redis
```

## Automation Scripts

### Health Check Script

```bash
cat > health-check.sh << 'EOF'
#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_service() {
    local service=$1
    local url=$2
    
    if curl -s -f "$url" > /dev/null; then
        echo -e "${GREEN}✓${NC} $service is healthy"
        return 0
    else
        echo -e "${RED}✗${NC} $service is unhealthy"
        return 1
    fi
}

echo "Voice Ferry Health Check"
echo "======================"

# Check main service
check_service "Main Service" "http://localhost:8080/health"

# Check SIP service
check_service "SIP Service" "http://localhost:8080/health/sip"

# Check Redis
check_service "Redis" "http://localhost:8080/health/redis"

# Check etcd
check_service "etcd" "http://localhost:8080/health/etcd"

# Check rtpengine
check_service "RTPEngine" "http://localhost:8080/health/rtpengine"

echo
echo "System Metrics:"
curl -s http://localhost:8080/api/metrics/system | jq -r '
  "CPU Usage: " + (.cpu.usage | tostring) + "%",
  "Memory Usage: " + (.memory.usage | tostring) + "%",
  "Active Calls: " + (.calls.active | tostring)
'
EOF

chmod +x health-check.sh
```

### Maintenance Script

```bash
cat > maintenance.sh << 'EOF'
#!/bin/bash

case "$1" in
    start)
        echo "Starting Voice Ferry..."
        sudo systemctl start voice-ferry
        ;;
    stop)
        echo "Stopping Voice Ferry..."
        sudo systemctl stop voice-ferry
        ;;
    restart)
        echo "Restarting Voice Ferry..."
        sudo systemctl restart voice-ferry
        ;;
    status)
        sudo systemctl status voice-ferry
        ;;
    logs)
        sudo journalctl -u voice-ferry -f
        ;;
    backup)
        ./backup-config.sh
        ;;
    health)
        ./health-check.sh
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|backup|health}"
        exit 1
        ;;
esac
EOF

chmod +x maintenance.sh
```

## Advanced Operations

### Load Balancer Configuration

```nginx
# Nginx configuration for Voice Ferry
upstream voice_ferry_api {
    server 127.0.0.1:50051;
    server 127.0.0.1:50052;
    server 127.0.0.1:50053;
}

upstream voice_ferry_health {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

server {
    listen 80;
    server_name voice-ferry.example.com;

    location /api/ {
        grpc_pass grpc://voice_ferry_api;
    }

    location /health {
        proxy_pass http://voice_ferry_health;
    }
}
```

### Cluster Management

```bash
# Join cluster
grpcurl -plaintext \
  -d '{"node_id": "node-2", "address": "192.168.1.102:50051"}' \
  192.168.1.101:50051 \
  b2bua.v1.ClusterService/JoinCluster

# List cluster members
grpcurl -plaintext \
  192.168.1.101:50051 \
  b2bua.v1.ClusterService/ListMembers

# Remove node from cluster
grpcurl -plaintext \
  -d '{"node_id": "node-2"}' \
  192.168.1.101:50051 \
  b2bua.v1.ClusterService/RemoveNode
```

This comprehensive administration guide provides all the necessary commands and procedures for deploying, configuring, monitoring, and troubleshooting Voice Ferry in production environments.
