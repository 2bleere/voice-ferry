
<div align="center">
  <img src="assets/Logo.png" alt="Voice Ferry Logo" width="200" height="auto" style="max-width: 100%; height: auto;">
  
  # Voice Ferry a Cloud-Native Class 4 Switch
  
  [![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/doc/devel/release.html)
  [![Kubernetes](https://img.shields.io/badge/Kubernetes-1.25+-green.svg)](https://kubernetes.io/)
  [![Helm](https://img.shields.io/badge/Helm-3.0+-blue.svg)](https://helm.sh/)
</div>

---

A high-performance, cloud-native Class 4 Switch with SIP B2BUA functionality built in Go, designed for Kubernetes environments with comprehensive call routing, media handling via rtpengine, and modern observability features.

## 🚀 Features

### Core Class 4 Switch Functionality
- **Full SIP B2BUA Implementation**: Complete SIP protocol support with independent call legs
- **Media Relay Integration**: rtpengine integration for RTP/RTCP media handling
- **Call State Management**: Advanced dialog and session state management
- **Session Limits**: Configurable per-user concurrent session limits with Redis-based tracking
- **Multi-transport Support**: UDP, TCP, TLS, WebSocket (WS/WSS)

### Cloud-Native Architecture
- **Kubernetes Native**: Purpose-built for Kubernetes deployment
- **Horizontal Scaling**: Stateless design for easy horizontal scaling
- **Service Mesh Ready**: Compatible with Istio and other service meshes
- **Health Checks**: Comprehensive liveness, readiness, and startup probes

### Configuration Management
- **etcd Integration**: Distributed configuration management
- **Dynamic Reconfiguration**: Live configuration updates without restart
- **Environment-based Config**: Support for environment variables and secrets

### API & Management
- **gRPC API**: Modern gRPC API for call management and configuration
- **RESTful Health Endpoints**: Standard health check endpoints
- **Real-time Call Control**: Live call manipulation and monitoring

### Security & Access Control
- **JWT Authentication**: Secure API access with JWT tokens
- **IP-based ACLs**: Flexible IP access control lists
- **TLS Support**: End-to-end TLS encryption for all protocols
- **SIP Digest Authentication**: Optional SIP digest authentication

### Observability & Monitoring
- **Structured Logging**: JSON-formatted logs for easy parsing
- **Metrics Export**: Prometheus-compatible metrics
- **Distributed Tracing**: OpenTelemetry integration (planned)
- **Call Detail Records**: Comprehensive call logging and analytics

## 🏗 Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SIP Client A  │────│   SIP B2BUA     │────│   SIP Client B  │
└─────────────────┘    │                 │    └─────────────────┘
                       │  ┌───────────┐  │
                       │  │Dialog Mgr │  │
                       │  └───────────┘  │
                       │  ┌───────────┐  │
                       │  │Routing Eng│  │
                       │  └───────────┘  │
                       │  ┌───────────┐  │
                       │  │gRPC API   │  │
                       │  └───────────┘  │
                       └─────────────────┘
                              │
                       ┌─────────────────┐
                       │   rtpengine     │
                       │  (Media Relay)  │
                       └─────────────────┘
                              │
                    ┌─────────────────────────┐
                    │    External Network     │
                    └─────────────────────────┘
```

## 📦 Quick Start

### Prerequisites
- Go 1.24.3+ 
- Docker & Kubernetes cluster
- rtpengine instance
- etcd cluster (3-replica StatefulSet recommended for production)
- Redis instance (single instance or 6-node cluster for HA)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/2bleere/voice-ferry.git
   cd sip-b2bua
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Generate protobuf code**
   ```bash
   # Install protoc and plugins
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   
   # Generate code
   export PATH="$PATH:$(go env GOPATH)/bin"
   protoc --proto_path=proto --go_out=proto/gen --go_opt=paths=source_relative \
          --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
          proto/b2bua/v1/b2bua.proto
   ```

4. **Run dependencies with Docker Compose**
   ```bash
   docker compose -f deployments/docker/docker-compose.yml up -d
   ```

5. **Start the B2BUA**
   ```bash
   go run cmd/b2bua/main.go -config configs/config.yaml.example
   ```

### Kubernetes Deployment

#### Option 1: Helm Chart (Recommended)

Deploy using the comprehensive Helm chart with Redis cluster support:

```bash
# Deploy to development
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry \
  --create-namespace \
  --values helm/voice-ferry/values-dev.yaml

# Deploy to staging
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry-staging \
  --create-namespace \
  --values helm/voice-ferry/values-staging.yaml

# Deploy to production
helm install voice-ferry ./helm/voice-ferry \
  --namespace voice-ferry-prod \
  --create-namespace \
  --values helm/voice-ferry/values-prod.yaml
```

The Helm chart includes:
- Redis cluster (6-node with automatic initialization)
- High availability configuration
- Horizontal pod autoscaling
- Monitoring and observability
- Security hardening (RBAC, network policies)
- Multi-environment support

See [Helm Chart Documentation](helm/voice-ferry/README.md) for detailed configuration options.

#### Option 2: Raw Kubernetes Manifests

For production deployments with Redis cluster support:

```bash
# Deploy infrastructure dependencies (etcd + Redis single instance)
kubectl apply -f deployments/kubernetes/dependencies.yaml

# OR deploy Redis cluster for high availability
kubectl apply -f deployments/kubernetes/redis-cluster.yaml

# Deploy the main application with dependency health checks
kubectl apply -f deployments/kubernetes/sip-b2bua.yaml

# For production configuration
kubectl apply -f deployments/kubernetes/voice-ferry-production.yaml

# Validate deployment health
./deployments/kubernetes/validate-deployment.sh
```

**Deployment Features:**
- **Dependency Health Checks**: Init containers ensure Redis, etcd, and rtpengine are ready before startup
- **Redis Options**: Choose between single Redis instance or 6-node Redis cluster for HA
- **High Availability**: 3-replica etcd StatefulSet with proper cluster configuration
- **Production Ready**: Resource limits, health probes, and startup ordering

See [Deployment Strategy Guide](deployments/kubernetes/DEPLOYMENT_STRATEGY.md) and [Production Deployment Guide](deployments/kubernetes/PRODUCTION_DEPLOYMENT.md) for detailed instructions.

## 🔧 Configuration

The B2BUA uses YAML configuration with support for environment variable substitution. See `configs/config.yaml.example` for a complete configuration reference.

### Key Configuration Sections

- **SIP**: Protocol settings, transport options, timeouts
- **gRPC**: API server configuration and TLS settings
- **etcd**: Distributed configuration store settings
- **Redis**: Session storage, caching, and session limits configuration
- **rtpengine**: Media relay instance configuration
- **Security**: Authentication, ACLs, and encryption settings
- **Session Management**: Per-user concurrent session limits and enforcement policies

### Documentation

- [Complete Configuration Guide](docs/configuration.md) - Comprehensive configuration reference
- [Session Limits Guide](docs/session-limits.md) - Detailed session limits configuration and management
- [Routing System](docs/routing-system.md) - Call routing configuration
- [Routing Examples](docs/routing-examples.md) - Common routing scenarios

### Environment Variables

All configuration values can be overridden using environment variables:

```bash
export SIP_PORT=5061
export GRPC_PORT=50052
export DEBUG=true
```

## 📡 API Reference

### gRPC Services

#### B2BUACallService
- `InitiateCall`: Start a new SIP call
- `TerminateCall`: End an active call
- `GetActiveCalls`: Stream active call information
- `GetCallDetails`: Get detailed call information

#### RoutingRuleService  
- `AddRoutingRule`: Create new routing rules
- `UpdateRoutingRule`: Modify existing rules
- `DeleteRoutingRule`: Remove routing rules
- `ListRoutingRules`: Get all routing rules

#### SIPHeaderService
- `AddSipHeader`: Add custom SIP headers
- `RemoveSipHeader`: Remove SIP headers
- `ReplaceSipHeader`: Replace SIP header values
- `GetSipHeaders`: Retrieve current headers

### Health Check Endpoints

- `GET /healthz/live`: Liveness probe
- `GET /healthz/ready`: Readiness probe  
- `GET /healthz/startup`: Startup probe
- `GET /metrics`: Prometheus metrics

## 🔍 Monitoring & Observability

### Metrics

The B2BUA exports Prometheus-compatible metrics:

- `sip_requests_total`: Total SIP requests by method
- `sip_responses_total`: Total SIP responses by code
- `active_calls`: Current number of active calls
- `active_sessions_per_user`: Active sessions count per user (when session limits enabled)
- `session_limit_rejections_total`: Total number of calls rejected due to session limits
- `call_duration_seconds`: Call duration histogram
- `rtpengine_sessions`: Active rtpengine sessions

### Logging

Structured JSON logging with configurable levels:

```json
{
  "timestamp": "2025-05-28T10:30:00Z",
  "level": "info",
  "message": "Call established",
  "call_id": "abc123",
  "from_uri": "sip:alice@example.com",
  "to_uri": "sip:bob@example.com"
}
```

## 🛡 Security

### Authentication
- JWT-based API authentication
- Optional SIP digest authentication
- TLS encryption for all protocols

### Access Control
- IP-based access control lists
- Network policy integration
- Role-based access control (planned)

### Best Practices
- Run as non-root user
- Read-only filesystem
- Security contexts and capabilities
- Regular security updates

## 🚢 Deployment Scenarios

### Production Kubernetes
- Multi-replica deployment for high availability
- Horizontal Pod Autoscaler (HPA) for traffic-based scaling
- Network policies for security isolation
- Persistent storage for configuration and logs
- Redis cluster support for session data resilience
- Automated dependency health checks with init containers
- Comprehensive deployment validation tools

### Docker Compose
- Simple single-node deployment
- Suitable for development and testing
- Includes all dependencies

### Bare Metal
- Systemd service configuration
- Direct deployment on Linux servers
- Manual dependency management

## 🛠 Development

### Project Structure
```
├── cmd/b2bua/          # Main application entry point
├── pkg/                # Public packages
│   ├── sip/           # SIP protocol implementation
│   ├── rtpengine/     # rtpengine client
│   ├── config/        # Configuration management
│   ├── auth/          # Authentication & authorization
│   └── routing/       # Call routing engine
├── internal/           # Private packages
│   ├── server/        # Main server implementation
│   └── handlers/      # gRPC service handlers
├── proto/              # Protocol buffer definitions
├── deployments/        # Deployment configurations
│   ├── kubernetes/    # K8s manifests
│   └── docker/        # Docker configurations
└── docs/              # Documentation
```

### Building

```bash
# Build binary
go build -o bin/b2bua cmd/b2bua/main.go

# Build Docker image
docker build -f deployments/docker/Dockerfile -t sip-b2bua:latest .

# Multi-platform build and push to registry
./scripts/build-and-push.sh

# Run tests
go test ./...
```

For comprehensive container registry management, multi-platform builds, and deployment strategies, see the [Container Registry Documentation](docs/container-registry.md).
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📋 Requirements

### Runtime Dependencies
- rtpengine (for media relay) - automatically health-checked during startup
- etcd (for configuration) - 3-replica StatefulSet with cluster health validation
- Redis (for caching and session limits) - single instance or Redis cluster with health monitoring

### Development Dependencies
- Go 1.24.3+
- Protocol Buffers compiler
- Docker (for containerization)
- Kubernetes (for deployment)

## 📖 Documentation

- [Configuration Reference](docs/configuration.md)
- [API Documentation](docs/api.md) 
- [Deployment Guide](docs/deployment.md)
- [Deployment Strategy Guide](deployments/kubernetes/DEPLOYMENT_STRATEGY.md)
- [Production Deployment Guide](deployments/kubernetes/PRODUCTION_DEPLOYMENT.md)
- [Redis Integration Testing](docs/redis-integration-testing.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Changelog](CHANGELOG.md)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Support

- GitHub Issues: Report bugs and request features
- Documentation: Comprehensive guides and API reference
- Community: Join our community discussions

## 🎯 Roadmap

### Recently Completed ✅
- [x] **Redis Cluster Support**: Full Redis cluster integration with 6-node HA deployment
- [x] **Dependency Health Checks**: Init containers for reliable startup ordering
- [x] **Production Deployment Tools**: Comprehensive validation scripts and deployment guides
- [x] **Enhanced Session Management**: Improved per-user session limits with Redis backend
- [x] **Automated Testing**: Redis integration testing and CI/CD pipeline improvements

### In Progress 🚧
- [ ] Advanced load balancing algorithms
- [ ] Call recording integration

### Planned 📋
- [ ] Kubernetes operator
- [ ] OpenTelemetry tracing
- [ ] GraphQL API
- [ ] Web-based management UI
- [ ] Enhanced monitoring dashboards
- [ ] Multi-region deployment support

---

Built with ❤️ for the telecommunications community
