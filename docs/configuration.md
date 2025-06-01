# Voice Ferry B2BUA Configuration Guide

This documentation provides a comprehensive guide to configuring the Voice Ferry SIP B2BUA server. It covers all configuration sections, parameters, and includes examples for common deployment scenarios.

## Table of Contents

1. [Configuration Overview](#configuration-overview)
2. [Configuration File Format](#configuration-file-format)
3. [Core Configuration](#core-configuration)
4. [SIP Server Configuration](#sip-server-configuration)
5. [gRPC API Configuration](#grpc-api-configuration)
6. [WebRTC Gateway Configuration](#webrtc-gateway-configuration)
7. [Health and Metrics Configuration](#health-and-metrics-configuration)
8. [Logging Configuration](#logging-configuration)
9. [External Integrations](#external-integrations)
   - [RTPEngine Configuration](#rtpengine-configuration)
   - [Redis Configuration](#redis-configuration)
   - [Etcd Configuration](#etcd-configuration)
10. [Session Management](#session-management)
    - [Session Limits Configuration](#session-limits-configuration)
11. [Security Configuration](#security-configuration)
    - [TLS Configuration](#tls-configuration)
    - [Authentication](#authentication)
    - [IP Access Control](#ip-access-control)
12. [Deployment Scenarios](#deployment-scenarios)
    - [Basic SIP Server](#basic-sip-server)
    - [Secure Production Deployment](#secure-production-deployment)
    - [High Availability Setup](#high-availability-setup)
    - [WebRTC Gateway](#webrtc-gateway)

## Configuration Overview

The Voice Ferry B2BUA configuration is managed through a YAML file that defines all server behavior, including network endpoints, security settings, external integrations, and more. The configuration file is typically loaded from the `configs/` directory.

### Loading Configuration

By default, the B2BUA will look for a configuration file at one of the following locations:

1. Path specified via command-line argument: `--config /path/to/config.yaml`
2. Environment variable: `B2BUA_CONFIG=/path/to/config.yaml`
3. Default location: `configs/config.yaml`

If no configuration file is found, the server will use default values.

## Configuration File Format

The configuration file uses YAML format, with sections organized by functionality. Here's an overview of the top-level sections:

```yaml
# Global debug flag
debug: false

# SIP Protocol settings
sip:
  # SIP configuration details...

# gRPC API settings
grpc:
  # gRPC configuration details...

# WebRTC gateway settings
webrtc:
  # WebRTC configuration details...

# Health check endpoint
health:
  # Health check configuration details...

# Metrics collection
metrics:
  # Metrics configuration details...

# Logging settings
logging:
  # Logging configuration details...

# External integrations
rtpengine:
  # RTPEngine configuration details...
redis:
  # Redis configuration details...
etcd:
  # Etcd configuration details...

# Security settings
security:
  # Security configuration details...
```

## Core Configuration

These settings control fundamental aspects of the B2BUA server.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `debug` | boolean | `false` | When enabled, increases logging verbosity and enables additional debug endpoints |

## SIP Server Configuration

The `sip` section configures the core SIP protocol behavior.

```yaml
sip:
  host: "0.0.0.0"
  port: 5060
  transport: "UDP"  # UDP, TCP, TLS, WS, WSS
  tls:
    # TLS configuration...
  timeouts:
    transaction: 32s
    dialog: 1800s      # 30 minutes
    registration: 3600s # 1 hour
  auth:
    enabled: false
    realm: "sip.example.com"
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | `0.0.0.0` | IP address to bind the SIP server to |
| `port` | integer | `5060` | Port to listen for SIP messages |
| `transport` | string | `UDP` | Transport protocol (UDP, TCP, TLS, WS, WSS) |

### Timeouts

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `transaction` | duration | `32s` | SIP transaction timeout |
| `dialog` | duration | `1800s` | SIP dialog timeout (30 minutes) |
| `registration` | duration | `3600s` | SIP registration expiry (1 hour) |

### SIP Authentication

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `auth.enabled` | boolean | `false` | Enable SIP authentication |
| `auth.realm` | string | `sip.example.com` | Authentication realm |

## gRPC API Configuration

The `grpc` section configures the gRPC API server that allows programmatic control of the B2BUA.

```yaml
grpc:
  host: "0.0.0.0"
  port: 50051
  tls:
    # TLS configuration...
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | `0.0.0.0` | IP address to bind the gRPC server to |
| `port` | integer | `50051` | Port to listen for gRPC requests |

## WebRTC Gateway Configuration

The WebRTC gateway allows web browsers to connect to the B2BUA using WebRTC.

```yaml
webrtc:
  enabled: false
  host: "0.0.0.0"
  port: 8081
  ws_path: "/ws"
  stun_servers:
    - "stun:stun.l.google.com:19302"
    - "stun:stun1.l.google.com:19302"
  turn_servers:
    - url: "turn:turn.example.com:3478"
      username: "turnuser"
      password: "turnpass"
  tls:
    # TLS configuration...
  auth:
    enabled: false
    jwt_secret: "your-jwt-secret"
    allowed_origins:
      - "*"
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable WebRTC gateway functionality |
| `host` | string | `0.0.0.0` | IP address to bind the WebRTC server to |
| `port` | integer | `8081` | Port to listen for WebRTC connections |
| `ws_path` | string | `/ws` | WebSocket endpoint path |
| `stun_servers` | string[] | Google STUN servers | STUN servers for NAT traversal |

### TURN Servers

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | - | URL of the TURN server |
| `username` | string | - | Authentication username |
| `password` | string | - | Authentication password |

### WebRTC Authentication

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `auth.enabled` | boolean | `false` | Enable WebRTC authentication |
| `auth.jwt_secret` | string | - | Secret key for JWT verification |
| `auth.allowed_origins` | string[] | `["*"]` | List of allowed CORS origins |

## Health and Metrics Configuration

These sections configure monitoring and observability endpoints.

### Health Check

```yaml
health:
  host: "0.0.0.0"
  port: 8080
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `host` | string | `0.0.0.0` | IP address to bind the health check server to |
| `port` | integer | `8080` | Port for the health check endpoint |

### Metrics

```yaml
metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
  update_period: "15s"
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable Prometheus metrics collection |
| `host` | string | `0.0.0.0` | IP address to bind the metrics server to |
| `port` | integer | `9090` | Port for the metrics endpoint |
| `path` | string | `/metrics` | URL path for metrics endpoint |
| `update_period` | duration | `15s` | Interval for metrics updates |

## Logging Configuration

Configure log format, output, and verbosity.

```yaml
logging:
  level: "info"          # debug, info, warn, error
  format: "text"         # text, json
  output: "stdout"       # stdout, stderr, file
  file: "/var/log/b2bua.log"  # Used when output is set to "file"
  include_source: false  # Include source code location in logs
  version: "1.0.0"       # Software version for log context
  instance_id: "b2bua-1" # Instance identifier for clustered deployments
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `level` | string | `info` | Log level (debug, info, warn, error) |
| `format` | string | `text` | Log format (text, json) |
| `output` | string | `stdout` | Log output destination |
| `file` | string | - | Log file path (when output is "file") |
| `include_source` | boolean | `false` | Include source file and line in log entries |
| `version` | string | - | Application version for log context |
| `instance_id` | string | - | Instance identifier for log context |

## External Integrations

### RTPEngine Configuration

Configure connections to RTPEngine media proxy instances.

```yaml
rtpengine:
  instances:
    - id: "rtpengine-1"
      host: "127.0.0.1"
      port: 22222
      weight: 100
      enabled: true
  timeout: 5s
```

#### RTPEngine Instances

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `id` | string | - | Unique identifier for the instance |
| `host` | string | - | RTPEngine host address |
| `port` | integer | - | RTPEngine control protocol port |
| `weight` | integer | `100` | Load balancing weight |
| `enabled` | boolean | `true` | Whether this instance is active |

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `timeout` | duration | `5s` | Timeout for RTPEngine communication |

### Redis Configuration

Configure Redis connection for session state, caching, data storage, and session limits enforcement. Voice Ferry supports both single Redis instance and Redis cluster deployments.

```yaml
redis:
  enabled: false
  host: "127.0.0.1"      # Single instance host or cluster endpoint
  port: 6379
  password: ""
  database: 0
  pool_size: 10
  min_idle_conns: 5
  max_idle_time: 300     # seconds
  conn_max_lifetime: 3600 # seconds
  timeout: 5             # seconds
  
  # Cluster configuration (for Redis cluster mode)
  cluster:
    enabled: false
    nodes:               # List of cluster nodes
      - "redis-cluster-0:6379"
      - "redis-cluster-1:6379" 
      - "redis-cluster-2:6379"
    
  # Session limit settings
  enable_session_limits: true
  max_sessions_per_user: 5          # Maximum concurrent sessions per user
  session_limit_action: "reject"    # Action when limit reached: "reject" or "terminate_oldest"
  
  # Health monitoring
  health_check_interval: 30s        # Health check frequency
  retry_attempts: 3                 # Connection retry attempts
  
  tls:
    # TLS configuration...
```

#### Connection Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable Redis integration |
| `host` | string | `127.0.0.1` | Redis server host (single instance) |
| `port` | integer | `6379` | Redis server port |
| `password` | string | - | Redis server password |
| `database` | integer | `0` | Redis database number (ignored in cluster mode) |
| `pool_size` | integer | `10` | Connection pool size |
| `min_idle_conns` | integer | `5` | Minimum idle connections in pool |
| `max_idle_time` | integer | `300` | Maximum idle time in seconds |
| `conn_max_lifetime` | integer | `3600` | Maximum connection lifetime in seconds |
| `timeout` | integer | `5` | Connection timeout in seconds |

#### Cluster Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cluster.enabled` | boolean | `false` | Enable Redis cluster mode |
| `cluster.nodes` | string[] | - | List of Redis cluster node endpoints |

#### Session Limits Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enable_session_limits` | boolean | `false` | Enable per-user session limits enforcement |
| `max_sessions_per_user` | integer | `5` | Maximum number of concurrent sessions allowed per user |
| `session_limit_action` | string | `reject` | Action when session limit is exceeded. Options: `reject` (reject new calls) or `terminate_oldest` (terminate oldest session) |

#### Health Monitoring Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `health_check_interval` | duration | `30s` | Frequency of Redis health checks |
| `retry_attempts` | integer | `3` | Number of connection retry attempts |

### Etcd Configuration

Configure Etcd connection for distributed configuration and service discovery.

```yaml
etcd:
  enabled: false
  endpoints:
    - "127.0.0.1:2379"
  dial_timeout: 5s
  username: ""
  password: ""
  tls:
    # TLS configuration...
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable Etcd integration |
| `endpoints` | string[] | `["127.0.0.1:2379"]` | List of Etcd endpoints |
| `dial_timeout` | duration | `5s` | Connection timeout |
| `username` | string | - | Authentication username |
| `password` | string | - | Authentication password |

## Session Management

### Session Limits Configuration

The B2BUA provides configurable per-user session limits to prevent resource exhaustion and enforce fair usage policies. Session limits are tracked using Redis and can be configured with different enforcement actions.

#### Overview

Session limits work by:
1. **Tracking Active Sessions**: Each active call session is tracked in Redis with the user's identifier
2. **Counting Per User**: The system maintains a count of concurrent sessions for each user
3. **Enforcing Limits**: When a new session would exceed the configured limit, the specified action is taken
4. **Automatic Cleanup**: Sessions are automatically removed when calls end

#### Configuration

Session limits are configured in the Redis section:

```yaml
redis:
  enable_session_limits: true
  max_sessions_per_user: 5
  session_limit_action: "reject"
```

#### Session Limit Actions

When a user attempts to create more sessions than allowed, the B2BUA can take one of the following actions:

- **`reject`** (default): Reject the new call attempt with a SIP 486 Busy Here response
- **`terminate_oldest`**: Terminate the user's oldest active session to make room for the new one

#### Example Configurations

**Strict Session Limits (Recommended for Production)**
```yaml
redis:
  enable_session_limits: true
  max_sessions_per_user: 3
  session_limit_action: "reject"
```

**Flexible Session Management (for Development)**
```yaml
redis:
  enable_session_limits: true
  max_sessions_per_user: 10
  session_limit_action: "terminate_oldest"
```

**Disabled Session Limits**
```yaml
redis:
  enable_session_limits: false
  # max_sessions_per_user and session_limit_action are ignored when disabled
```

#### User Identification

Sessions are tracked by extracting the username from the SIP From header. The system uses the user part of the SIP URI (e.g., for `sip:user123@example.com`, the username is `user123`).

#### Monitoring and Troubleshooting

- **Active Sessions**: Use the gRPC API `GetActiveCalls` to monitor current sessions
- **Redis Keys**: Session data is stored with keys like `session:*` and user tracking with `user_sessions:*`
- **Logs**: Session limit enforcement events are logged at INFO level
- **Metrics**: Session count metrics are available for monitoring

#### Testing Session Limits

The repository includes several test scripts to verify session limits functionality:

- `test_same_user_limits.py`: Tests limits using the same user
- `test_simple_session_limits.py`: Simple gRPC-based session testing
- `verify_session_limits.py`: Verification script for session limits
- `test_session_limits_stress.py`: Stress testing for session limits

For detailed information about session limits, including deployment scenarios, monitoring, and troubleshooting, see the [Session Limits Guide](session-limits.md).

## Security Configuration

### TLS Configuration

TLS can be configured for SIP, gRPC, WebRTC, Redis, and Etcd connections. Each section follows the same format:

```yaml
tls:
  enabled: false
  cert_file: "/etc/ssl/certs/cert.crt"
  key_file: "/etc/ssl/private/key.key"
  ca_file: "/etc/ssl/certs/ca.crt"
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable TLS for this component |
| `cert_file` | string | - | Path to certificate file |
| `key_file` | string | - | Path to private key file |
| `ca_file` | string | - | Path to CA certificate for client verification |

### Authentication

#### JWT Authentication

```yaml
security:
  jwt:
    signing_key: "your-secret-key-change-this-in-production"
    expiration: 24h
    issuer: "b2bua"

auth:
  enabled: true
  jwt:
    public_key_path: "/etc/ssl/public.key"
    private_key_path: "/etc/ssl/private.key"
    signing_key: "your-signing-key"
    expiration: 24h
    issuer: "b2bua"
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `signing_key` | string | - | Secret key for JWT signing and verification |
| `expiration` | duration | `24h` | JWT token validity period |
| `issuer` | string | `b2bua` | JWT issuer claim value |
| `public_key_path` | string | - | Path to public key for RSA-based JWT validation |
| `private_key_path` | string | - | Path to private key for RSA-based JWT signing |

#### SIP Digest Authentication

```yaml
security:
  sip:
    digest_auth:
      realm: "sip.example.com"
      enabled: false
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `realm` | string | `sip.example.com` | Authentication realm |
| `enabled` | boolean | `false` | Enable SIP digest authentication |

### IP Access Control

Configure IP-based access control for SIP and API endpoints.

```yaml
security:
  sip:
    ip_acls:
      - name: "localhost"
        action: "allow"
        networks:
          - "127.0.0.0/8"
          - "::1/128"
      - name: "private_networks"
        action: "allow"
        networks:
          - "10.0.0.0/8"
          - "172.16.0.0/12"
          - "192.168.0.0/16"
      - name: "blocked_network"
        action: "deny"
        networks:
          - "192.168.100.0/24"

auth:
  ip_whitelist:
    - "127.0.0.1"
    - "::1"
  ip_blacklist:
    - "192.168.1.100"
```

#### IP ACLs

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Descriptive name for the ACL |
| `action` | string | - | Action to take (allow, deny) |
| `networks` | string[] | - | List of IP networks in CIDR notation |

#### IP Whitelist/Blacklist

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ip_whitelist` | string[] | `["127.0.0.1", "::1"]` | IPs that are always allowed |
| `ip_blacklist` | string[] | - | IPs that are always blocked |

## Deployment Scenarios

### Basic SIP Server

This configuration sets up a basic SIP server with minimal features:

```yaml
debug: false

sip:
  host: "0.0.0.0" 
  port: 5060
  transport: "UDP"

rtpengine:
  instances:
    - id: "rtpengine-1"
      host: "127.0.0.1"
      port: 22222
      enabled: true
  timeout: 5s

logging:
  level: "info"
  format: "text"
  output: "stdout"

health:
  host: "0.0.0.0"
  port: 8080
```

### Secure Production Deployment

This configuration demonstrates a secure production deployment:

```yaml
debug: false

sip:
  host: "0.0.0.0"
  port: 5061
  transport: "TLS"
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/sip.example.com.crt"
    key_file: "/etc/ssl/private/sip.example.com.key"
    ca_file: "/etc/ssl/certs/ca.crt"
  auth:
    enabled: true
    realm: "sip.example.com"

grpc:
  host: "0.0.0.0"
  port: 50051
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/api.example.com.crt"
    key_file: "/etc/ssl/private/api.example.com.key"
    ca_file: "/etc/ssl/certs/ca.crt"

rtpengine:
  instances:
    - id: "rtpengine-1"
      host: "10.0.1.10"
      port: 22222
      weight: 100
      enabled: true
    - id: "rtpengine-2"
      host: "10.0.1.11"
      port: 22222
      weight: 100
      enabled: true
  timeout: 5s

logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/var/log/b2bua.log"
  include_source: true
  instance_id: "b2bua-prod-1"

security:
  jwt:
    signing_key: "your-strong-random-key-here"
    expiration: 1h
    issuer: "sip.example.com"
  sip:
    ip_acls:
      - name: "trusted_carriers"
        action: "allow"
        networks:
          - "203.0.113.0/24"
          - "198.51.100.0/24"
      - name: "internal_network"
        action: "allow"
        networks:
          - "10.0.0.0/8"
      - name: "default"
        action: "deny"
        networks:
          - "0.0.0.0/0"
          - "::/0"
    digest_auth:
      realm: "sip.example.com"
      enabled: true

metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"

redis:
  enabled: true
  host: "10.0.2.5"
  port: 6379
  password: "your-strong-redis-password"
  database: 0
  pool_size: 20
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/redis-client.crt"
    key_file: "/etc/ssl/private/redis-client.key"
    ca_file: "/etc/ssl/certs/redis-ca.crt"
```

### High Availability Setup

This configuration shows how to set up a high availability B2BUA cluster:

```yaml
sip:
  host: "0.0.0.0"
  port: 5060
  transport: "UDP"

rtpengine:
  instances:
    - id: "rtpengine-pod1"
      host: "rtpengine-1.example.com"
      port: 22222
      weight: 100
      enabled: true
    - id: "rtpengine-pod2"
      host: "rtpengine-2.example.com"
      port: 22222
      weight: 100
      enabled: true
    - id: "rtpengine-pod3"
      host: "rtpengine-3.example.com"
      port: 22222
      weight: 100
      enabled: true
  timeout: 5s

redis:
  enabled: true
  host: "redis-cluster.example.com"
  port: 6379
  password: "your-strong-redis-password"
  database: 0
  pool_size: 50
  min_idle_conns: 10
  max_idle_time: 300
  conn_max_lifetime: 3600
  timeout: 5

etcd:
  enabled: true
  endpoints:
    - "etcd-1.example.com:2379"
    - "etcd-2.example.com:2379"
    - "etcd-3.example.com:2379"
  dial_timeout: 5s
  username: "b2bua"
  password: "your-strong-etcd-password"
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/etcd-client.crt"
    key_file: "/etc/ssl/private/etcd-client.key"
    ca_file: "/etc/ssl/certs/etcd-ca.crt"

logging:
  level: "info"
  format: "json"
  output: "stdout"  # For container environments
  include_source: false
  instance_id: "${POD_NAME}"  # Environment variable substitution

metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
  update_period: "10s"
```

### WebRTC Gateway

This configuration enables the WebRTC gateway for browser-based SIP clients:

```yaml
sip:
  host: "0.0.0.0"
  port: 5060
  transport: "UDP"

webrtc:
  enabled: true
  host: "0.0.0.0"
  port: 8081
  ws_path: "/ws"
  stun_servers:
    - "stun:stun.example.com:3478"
  turn_servers:
    - url: "turn:turn.example.com:3478?transport=udp"
      username: "turnuser"
      password: "turnpass"
    - url: "turn:turn.example.com:3478?transport=tcp"
      username: "turnuser"
      password: "turnpass"
    - url: "turns:turn.example.com:5349?transport=tcp"
      username: "turnuser"
      password: "turnpass"
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/webrtc.example.com.crt"
    key_file: "/etc/ssl/private/webrtc.example.com.key"
  auth:
    enabled: true
    jwt_secret: "your-strong-webrtc-jwt-secret"
    allowed_origins:
      - "https://app.example.com"
      - "https://web-phone.example.com"

rtpengine:
  instances:
    - id: "rtpengine-webrtc"
      host: "127.0.0.1"
      port: 22222
      enabled: true
  timeout: 5s
```

---

## Environment Variables

Most configuration options can also be set using environment variables. The naming convention follows the pattern:

```
B2BUA_[SECTION]_[PARAMETER]
```

For nested parameters, use underscores to represent the hierarchy.

Examples:
- `B2BUA_SIP_PORT=5080` - Sets the SIP port to 5080
- `B2BUA_LOGGING_LEVEL=debug` - Sets log level to debug
- `B2BUA_SECURITY_JWT_SIGNING_KEY=secretkey` - Sets the JWT signing key

Environment variables take precedence over configuration file values.

## Reloading Configuration

The B2BUA supports runtime configuration reloading for certain parameters. To trigger a reload, send a SIGHUP signal to the process:

```bash
kill -HUP $(pidof b2bua)
```

Parameters that can be reloaded without restart:
- Log level
- IP ACLs
- Metrics settings
- RTPEngine instances

Parameters requiring a restart:
- Network bindings (host, port)
- TLS configuration
- Authentication methods

## Conclusion

This documentation covers the configuration options for the Voice Ferry SIP B2BUA. For additional assistance, please refer to the project wiki or submit an issue on the project repository.

For routing rule configuration details, refer to the [Routing System Documentation](routing-system.md) file.
