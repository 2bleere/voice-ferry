# Voice Ferry Web UI

A comprehensive web-based management interface for the Voice Ferry Class 4 Switch system. This containerized application provides real-time monitoring, configuration management, and administrative controls for your Voice Ferry deployment.

## Features

### 🎛️ **Dashboard & Monitoring**
- Real-time system status and performance metrics
- Live call statistics and session monitoring
- Interactive charts and visualizations
- WebSocket-powered live updates
- System health monitoring

### ⚙️ **Configuration Management**
- Complete SIP configuration interface
- SIP Users management (CRUD operations)
- Redis and etcd settings management
- Route management and policy configuration
- Backup and restore functionality
- Configuration validation and testing

### 🔐 **Security & Authentication**
- JWT-based authentication system
- Role-based access control
- Session management and limits
- Rate limiting and security headers
- Secure password handling

### 📊 **Real-time Features**
- Live call monitoring
- Real-time metrics updates
- WebSocket communication
- Push notifications
- Event streaming

### 🐳 **Containerized Deployment**
- Docker and Docker Compose support
- Nginx reverse proxy configuration
- Health checks and auto-restart
- Production-ready deployment
- Scalable architecture

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Voice Ferry B2BUA service
- 2GB+ RAM recommended
- Network access to SIP infrastructure

### 1. Clone and Setup
```bash
git clone <repository>
cd voice-ferry-web-ui
cp .env.example .env
```

### 2. Configure Environment
Edit the `.env` file with your settings:
```bash
# Security (CHANGE IN PRODUCTION!)
JWT_SECRET=your-secure-jwt-secret
SESSION_SECRET=your-secure-session-secret

# Service Endpoints
REDIS_URL=redis://redis:6379
ETCD_ENDPOINTS=http://etcd:2379
GRPC_ENDPOINT=voice-ferry-b2bua:50051

# Logging
LOG_LEVEL=info
```

### 3. Deploy
```bash
# Full deployment with all services
./deploy.sh deploy

# Or manually with docker-compose
docker-compose up -d
```

### 4. Access
- Web Interface: http://localhost:3000
- Default credentials: admin/admin (change immediately!)

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Nginx Proxy   │    │   Web UI App    │    │  Voice Ferry    │
│   (Optional)    │◄──►│   (Node.js)     │◄──►│    B2BUA        │
│   Port 80/443   │    │   Port 3000     │    │   Port 50051    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │                         │
                              ▼                         ▼
                    ┌─────────────────┐    ┌─────────────────┐
                    │     Redis       │    │      etcd       │
                    │  (Sessions)     │    │  (Config)       │
                    │   Port 6379     │    │   Port 2379     │
                    └─────────────────┘    └─────────────────┘
```

## Services

### Web UI Application
- **Port**: 3000
- **Technology**: Node.js, Express, WebSocket
- **Purpose**: Main web interface and API

### Redis
- **Port**: 6379
- **Purpose**: Session storage, caching, real-time data
- **Persistence**: Enabled with AOF

### etcd
- **Port**: 2379, 2380
- **Purpose**: Distributed configuration storage
- **Clustering**: Single node (expandable)

### Nginx (Optional)
- **Port**: 80, 443
- **Purpose**: Reverse proxy, SSL termination, rate limiting
- **Profile**: `proxy` (enable with `--profile proxy`)

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NODE_ENV` | Environment mode | `production` |
| `PORT` | Application port | `3000` |
| `JWT_SECRET` | JWT signing secret | Required |
| `REDIS_URL` | Redis connection URL | `redis://redis:6379` |
| `ETCD_ENDPOINTS` | etcd endpoints | `http://etcd:2379` |
| `GRPC_ENDPOINT` | B2BUA gRPC endpoint | `voice-ferry-b2bua:50051` |
| `LOG_LEVEL` | Logging level | `info` |

### Volume Mounts

| Container Path | Host Path | Purpose |
|----------------|-----------|---------|
| `/app/config` | `./config` | Configuration files |
| `/app/logs` | `./logs` | Application logs |
| `/app/data` | `./data` | Application data |

## Management Commands

### Deployment Script
```bash
# Full deployment
./deploy.sh deploy

# Start services
./deploy.sh start

# Stop services  
./deploy.sh stop

# View logs
./deploy.sh logs

# Check status
./deploy.sh status

# Backup configuration
./deploy.sh backup

# Update application
./deploy.sh update
```

### Docker Compose Commands
```bash
# Start all services
docker-compose up -d

# Start with Nginx proxy
docker-compose --profile proxy up -d

# View logs
docker-compose logs -f voice-ferry-ui

# Scale web UI (if needed)
docker-compose up -d --scale voice-ferry-ui=2

# Stop services
docker-compose down
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `GET /api/auth/me` - Current user info

### Dashboard
- `GET /api/dashboard/status` - System status
- `GET /api/dashboard/stats` - Call statistics
- `GET /api/dashboard/metrics` - Performance metrics

### Configuration
- `GET /api/config` - Get configuration
- `PUT /api/config` - Update configuration
- `POST /api/config/backup` - Create backup
- `POST /api/config/restore` - Restore backup

### Sessions
- `GET /api/sessions` - List active sessions
- `DELETE /api/sessions/:id` - Terminate session
- `GET /api/sessions/limits` - Session limits

### Metrics
- `GET /api/metrics` - System metrics
- `GET /api/metrics/alerts` - Active alerts
- `POST /api/metrics/alerts` - Create alert

## Security

### Authentication
- JWT tokens with configurable expiration
- Secure password hashing (bcrypt)
- Session management with Redis
- Rate limiting on authentication endpoints

### Network Security
- CORS protection
- Security headers (helmet.js)
- Request rate limiting
- Input validation and sanitization

### Container Security
- Non-root user execution
- Minimal base image (Alpine Linux)
- Health checks
- Resource limits

## Monitoring

### Health Checks
- Application health endpoint: `/api/health`
- Container health checks enabled
- Service dependency monitoring

### Logging
- Structured JSON logging
- Configurable log levels
- Container log aggregation
- Log rotation support

### Metrics
- Real-time performance metrics
- WebSocket event streaming
- Custom alerting rules
- Dashboard visualizations

## Development

### Local Development
```bash
# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Start development server
npm run dev

# Run tests
npm test

# Lint code
npm run lint
```

### Project Structure
```
voice-ferry-web-ui/
├── public/                 # Static files
│   ├── css/               # Stylesheets
│   ├── js/                # Client-side JavaScript
│   └── index.html         # Main HTML
├── routes/                # API routes
│   ├── auth.js           # Authentication
│   ├── config.js         # Configuration
│   ├── dashboard.js      # Dashboard
│   └── ...
├── services/              # Business logic
│   ├── auth.js           # Auth service
│   ├── config.js         # Config manager
│   ├── monitoring.js     # Monitoring
│   └── websocket.js      # WebSocket manager
├── server.js             # Main application
├── package.json          # Dependencies
├── Dockerfile            # Container definition
└── docker-compose.yml    # Service orchestration
```

## Troubleshooting

### Common Issues

**1. Container won't start**
```bash
# Check logs
docker-compose logs voice-ferry-ui

# Check environment variables
docker-compose config
```

**2. Can't connect to B2BUA**
```bash
# Check gRPC endpoint
docker-compose exec voice-ferry-ui nc -zv voice-ferry-b2bua 50051

# Check network connectivity
docker network ls
docker network inspect voice-ferry_voice-ferry-network
```

**3. Redis connection issues**
```bash
# Test Redis connection
docker-compose exec redis redis-cli ping

# Check Redis logs
docker-compose logs redis
```

**4. Configuration not persisting**
```bash
# Check volume mounts
docker-compose ps
docker volume ls

# Verify permissions
ls -la config/ data/
```

### Debug Mode
```bash
# Enable debug logging
echo "LOG_LEVEL=debug" >> .env
docker-compose restart voice-ferry-ui

# View detailed logs
docker-compose logs -f voice-ferry-ui
```

## Production Deployment

### Pre-deployment Checklist
- [ ] Change default JWT and session secrets
- [ ] Configure proper domain and SSL certificates
- [ ] Set up backup strategy for configuration
- [ ] Configure monitoring and alerting
- [ ] Review security settings
- [ ] Test disaster recovery procedures

### SSL/TLS Setup
```bash
# Generate self-signed certificate (development only)
mkdir -p ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/private.key -out ssl/certificate.crt

# Enable Nginx proxy with SSL
docker-compose --profile proxy up -d
```

### Backup Strategy
```bash
# Automated backup script
cat > backup-cron.sh << 'EOF'
#!/bin/bash
cd /path/to/voice-ferry-web-ui
./deploy.sh backup
find backups/ -name "*.tar.gz" -mtime +7 -delete
EOF

# Add to crontab (daily backup at 2 AM)
echo "0 2 * * * /path/to/backup-cron.sh" | crontab -
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license information here]

## Support

For issues and support:
- Check the troubleshooting section
- Review container logs
- Create an issue with detailed information

## Version History

- **v1.0.0** - Initial release with full web interface
- **v1.1.0** - Added Docker containerization
- **v1.2.0** - Enhanced monitoring and metrics

---

**Note**: This is a management interface for the Voice Ferry Class 4 Switch. Ensure you have the switch service running and properly configured before deploying this web interface.
