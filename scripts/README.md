# Scripts Directory

This directory contains all deployment, build, and operational scripts for the Voice Ferry project.

## Build Scripts

### `build-and-push.sh`
- Builds Docker images and pushes them to container registry
- Used for CI/CD pipelines and container deployment

### `build-docker-images.sh` 
- Builds all Docker images locally
- Useful for development and testing

## Deployment Scripts

### `deploy-production.sh`
- **PRIMARY PRODUCTION DEPLOYMENT SCRIPT**
- Comprehensive production environment deployment
- Includes health verification and status reporting
- Supports deploy, stop, restart, status, and logs commands

### `deploy.sh`
- General deployment script for various environments
- Flexible deployment configuration

### `deploy-docker-production.sh`
- Docker-specific production deployment
- Simplified Docker Compose deployment

### `simple-docker-deploy.sh`
- Basic Docker deployment for development/testing
- Minimal configuration deployment

### `start-production.sh`
- Production environment startup script
- Service initialization and configuration

## Operational Scripts

### `health-check.sh`
- **COMPREHENSIVE HEALTH MONITORING**
- Validates all service health (Redis, etcd, B2BUA, Web UI, RTPEngine)
- Supports individual service checks and complete system validation
- Includes etcd monitoring integration testing

## Usage Examples

### Production Deployment
```bash
# Full production deployment
./scripts/deploy-production.sh deploy

# Check system health
./scripts/health-check.sh

# View deployment status
./scripts/deploy-production.sh status
```

### Development
```bash
# Build images locally
./scripts/build-docker-images.sh

# Simple deployment for testing
./scripts/simple-docker-deploy.sh
```

### CI/CD
```bash
# Build and push to registry
./scripts/build-and-push.sh

# Deploy to production
./scripts/deploy-production.sh deploy
```

## Script Permissions

All scripts are executable and ready to use. If you encounter permission issues:

```bash
chmod +x scripts/*.sh
```

## Environment Requirements

- **Docker & Docker Compose**: Required for all deployment scripts
- **kubectl**: Required for Kubernetes deployments (deploy-production.sh)
- **curl, nc**: Required for health checks
- **jq**: Optional, for enhanced JSON processing in health checks

## Best Practices

1. **Always run health checks** after deployment
2. **Use deploy-production.sh** for production environments
3. **Test with simple-docker-deploy.sh** in development
4. **Monitor logs** using the deployment script log commands
5. **Validate environment** before running production deployments

## Integration

These scripts integrate with:
- **Kubernetes deployments** in `deployments/kubernetes/`
- **Docker Compose** configurations in project root
- **Environment configurations** in `configs/`
- **Health monitoring** for real-time system validation
