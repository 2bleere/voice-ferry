# Voice Ferry Web UI - Docker Setup

This document describes the Docker Compose setup for the Voice Ferry Web UI and its dependencies.

## 🏗️ Architecture

The Voice Ferry Web UI consists of several containerized services:

### Core Services
- **voice-ferry-ui**: Main web application (Node.js)
- **redis**: Session storage and caching
- **etcd**: Distributed configuration storage

### Optional Services
- **mock-b2bua**: Mock SIP service for development testing
- **nginx**: Reverse proxy (production profile)
- **redis-commander**: Redis management UI (tools profile)
- **etcd-browser**: etcd management UI (tools profile)

## 🚀 Quick Start

### Development Environment

```bash
# Start core services for development
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Or use the test script
./test-docker-setup.sh
```

### Production Environment

```bash
# Start production services
docker-compose up -d

# With reverse proxy
docker-compose --profile proxy up -d
```

## 📋 Service Profiles

### Default Profile
- voice-ferry-ui
- redis
- etcd

### Tools Profile (`--profile tools`)
Adds development and debugging tools:
- redis-commander (http://localhost:8081)
- etcd-browser (http://localhost:8082)

### Mock Profile (`--profile mock`)
Adds mock services for testing:
- mock-b2bua (SIP service simulator)

### Proxy Profile (`--profile proxy`)
Adds production reverse proxy:
- nginx (ports 80/443)

## 🔧 Configuration

### Environment Variables

#### Development (docker-compose.dev.yml)
```env
NODE_ENV=development
LOG_LEVEL=debug
JWT_SECRET=dev-jwt-secret-not-for-production
REDIS_URL=redis://redis:6379
ETCD_ENDPOINTS=http://etcd:2379
GRPC_ENDPOINT=mock-b2bua:50051
RATE_LIMIT_MAX=1000
RATE_LIMIT_WINDOW=15
```

#### Production (docker-compose.yml)
```env
NODE_ENV=production
JWT_SECRET=${JWT_SECRET:-voice-ferry-secret-key-change-in-production}
REDIS_URL=redis://redis:6379
ETCD_ENDPOINTS=http://etcd:2379
GRPC_ENDPOINT=voice-ferry-b2bua:50051
LOG_LEVEL=info
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=15
```

### Volume Mounts

#### Development
- `.:/app` - Source code hot reloading
- `./logs:/app/logs` - Log files
- `./config:/app/config` - Configuration files
- `./data:/app/data` - Application data

#### Production
- `./config:/app/config` - Configuration files only
- `./logs:/app/logs` - Log files
- `./data:/app/data` - Application data

## 🌐 Service Endpoints

| Service | Development | Production | Description |
|---------|-------------|------------|-------------|
| Web UI | http://localhost:3000 | http://localhost:3000 | Main application |
| Redis | localhost:6379 | localhost:6379 | Redis database |
| etcd | http://localhost:2379 | http://localhost:2379 | etcd API |
| Mock B2BUA | localhost:50051 (gRPC)<br>localhost:5060 (SIP) | N/A | Mock SIP service |
| Redis Commander | http://localhost:8081 | N/A | Redis management |
| etcd Browser | http://localhost:8082 | N/A | etcd management |
| Nginx | N/A | http://localhost:80<br>https://localhost:443 | Reverse proxy |

## 🧪 Testing SIP Users Functionality

### Using Mock Service

1. Start with mock profile:
```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile mock up -d
```

2. Access Web UI: http://localhost:3000

3. Navigate to SIP Users section

4. Test CRUD operations:
   - View existing users
   - Add new users
   - Edit user properties
   - Delete users
   - Toggle enabled/disabled status

### API Testing

```bash
# List SIP users
curl -X GET "http://localhost:3000/api/sip-users" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Create SIP user
curl -X POST "http://localhost:3000/api/sip-users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "username": "testuser",
    "password": "testpass123",
    "realm": "example.com",
    "enabled": true
  }'
```

## 🐛 Troubleshooting

### Common Issues

#### Services not starting
```bash
# Check logs
docker-compose logs voice-ferry-ui
docker-compose logs redis
docker-compose logs etcd

# Check container status
docker-compose ps
```

#### Permission issues (macOS/Linux)
```bash
# Fix log directory permissions
sudo chown -R $(id -u):$(id -g) ./logs

# Fix data directory permissions
sudo chown -R $(id -u):$(id -g) ./data
```

#### Port conflicts
```bash
# Check what's using ports
lsof -i :3000
lsof -i :6379
lsof -i :2379

# Use different ports
WEBUI_PORT=3001 docker-compose up -d
```

### Debug Mode

Enable debug logging:
```bash
# Set debug environment
LOG_LEVEL=debug docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Watch logs in real-time
docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f voice-ferry-ui
```

### Health Checks

```bash
# Check service health
curl http://localhost:3000/api/health
curl http://localhost:2379/health

# Check Redis
redis-cli -h localhost ping
```

## 🧹 Cleanup

### Stop Services
```bash
# Stop all services
docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

# Stop and remove volumes
docker-compose -f docker-compose.yml -f docker-compose.dev.yml down -v

# Remove everything including images
docker-compose -f docker-compose.yml -f docker-compose.dev.yml down -v --rmi all
```

### Clean Development Environment
```bash
# Remove development containers and images
docker-compose -f docker-compose.yml -f docker-compose.dev.yml down -v --rmi all --remove-orphans

# Clean Docker system
docker system prune -f
```

## 📦 Building Custom Images

### Development Build
```bash
# Build development image
docker-compose -f docker-compose.dev.yml build voice-ferry-ui

# Force rebuild
docker-compose -f docker-compose.dev.yml build --no-cache voice-ferry-ui
```

### Production Build
```bash
# Build production image
docker-compose build voice-ferry-ui

# Build with specific tag
docker build -t voice-ferry-ui:v1.0.0 .
```

## 🔐 Security Considerations

### Development
- Uses weak JWT secret (change for production)
- Exposes debugging ports
- Higher rate limits
- Verbose logging

### Production
- Strong JWT secret required
- No debugging ports exposed
- Production rate limits
- Minimal logging

### Recommendations
1. Always use HTTPS in production
2. Set strong JWT_SECRET environment variable
3. Regularly update base images
4. Monitor logs for security events
5. Use secrets management for sensitive data

## 📚 Additional Resources

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Voice Ferry Configuration Guide](../docs/configuration.md)
- [Voice Ferry Deployment Guide](../docs/deployment.md)
- [SIP Users API Documentation](../docs/api.md#sip-users)

## 🆘 Support

If you encounter issues:

1. Check this troubleshooting guide
2. Review logs: `docker-compose logs service-name`
3. Verify your Docker and docker-compose versions
4. Check system resources (CPU, memory, disk)
5. Consult the project documentation

---

**Note**: This setup is optimized for development and testing. For production deployments, review security settings and resource allocations.
