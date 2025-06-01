# Changelog

All notable changes to Voice Ferry will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Redis cluster support with 6-node HA deployment configuration
- Comprehensive dependency health checks using init containers
- Production deployment validation script (`validate-deployment.sh`)
- Redis integration testing framework and documentation
- Deployment strategy guide with Redis configuration options
- Production deployment guide with step-by-step instructions
- Enhanced session management with improved Redis backend
- Automated health monitoring for etcd, Redis, and rtpengine dependencies

### Changed
- Updated deployment files with improved dependency management
- Enhanced Redis configuration with cluster support options
- Improved application startup reliability with dependency health checks
- Updated documentation to reflect Redis cluster integration
- Enhanced monitoring and observability features

### Fixed
- Service name consistency in Redis cluster deployment
- Startup race conditions between dependencies
- Redis connection reliability in cluster mode

### Documentation
- Added comprehensive Redis integration testing guide
- Updated configuration documentation with cluster parameters
- Enhanced deployment guides with production best practices
- Added troubleshooting guides for dependency issues

## [1.0.0] - 2025-05-30

### Initial Release
- Full SIP B2BUA implementation with Class 4 switch functionality
- Multi-transport support (UDP, TCP, TLS, WebSocket)
- rtpengine integration for media relay
- etcd integration for configuration management
- Redis integration for session storage and limits
- JWT authentication and security features
- gRPC API for call management
- Comprehensive health checks and metrics
- Kubernetes deployment support
- Docker Compose development environment
- Session limits and per-user restrictions
- Advanced call routing engine
- Production-ready configuration examples

### Dependencies
- Go 1.24.3+
- Redis 7.x (single instance or cluster)
- etcd 3.5.x (3-replica StatefulSet recommended)
- rtpengine for media handling
- Kubernetes 1.20+ for production deployment

### Security
- TLS encryption for all protocols
- JWT-based API authentication
- IP-based access control lists
- SIP digest authentication support
- Security hardening for production deployments
