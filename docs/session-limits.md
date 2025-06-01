# Session Limits Guide

This guide provides comprehensive documentation for configuring and managing per-user session limits in the Voice Ferry SIP B2BUA.

## Overview

Session limits prevent individual users from consuming excessive system resources by limiting the number of concurrent active sessions (calls) each user can maintain. This feature is essential for:

- **Resource Protection**: Preventing resource exhaustion from excessive concurrent calls
- **Fair Usage**: Ensuring equal access to system resources across all users
- **Cost Control**: Limiting usage in environments with per-session costs
- **Quality Assurance**: Maintaining service quality by preventing oversubscription
- **User Management**: Applying different limits to different users based on their requirements

## How Session Limits Work

### Session Tracking

1. **Session Creation**: When a new call is initiated, the system extracts the username from the SIP From header
2. **Redis Storage**: Session information is stored in Redis with the following key patterns:
   - `session:{session_id}`: Individual session data
   - `user_sessions:{username}`: Set of active session IDs for each user
3. **Limit Checking**: Before allowing a new session, the system counts existing sessions for the user
4. **Enforcement**: If the limit would be exceeded, the configured action is taken
5. **Cleanup**: Sessions are automatically removed when calls terminate

### User Identification

Sessions are tracked by username extracted from the SIP From header:
- `sip:alice@example.com` → username: `alice`
- `sip:user123@domain.com` → username: `user123`
- `sip:+14155551234@provider.net` → username: `+14155551234`

## Configuration

### Basic Configuration

Enable session limits in your `config.yaml`:

```yaml
redis:
  enabled: true
  host: "127.0.0.1"
  port: 6379
  # Session limit settings
  enable_session_limits: true
  max_sessions_per_user: 5
  session_limit_action: "reject"
  # Optional: User-specific limits
  user_session_limits:
    alice: 10  # Allow user 'alice' to have 10 concurrent sessions
    bob: 3     # Restrict user 'bob' to only 3 concurrent sessions
    charlie: 0 # No limit for user 'charlie' (0 means unlimited)
```

### Configuration Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enable_session_limits` | boolean | `false` | Enable/disable session limits functionality |
| `max_sessions_per_user` | integer | `5` | Default maximum concurrent sessions per user |
| `user_session_limits` | map | `{}` | User-specific session limits that override the default |
| `session_limit_action` | string | `reject` | Action when limit exceeded: `reject` or `terminate_oldest` |

### Session Limit Actions

#### Reject Action (Default)

When `session_limit_action: "reject"` is configured:
- New calls that would exceed the limit are rejected
- SIP response: `486 Busy Here`
- Existing sessions remain unaffected
- Most predictable behavior for users

```yaml
redis:
  enable_session_limits: true
  max_sessions_per_user: 3
  session_limit_action: "reject"
```

#### Terminate Oldest Action

When `session_limit_action: "terminate_oldest"` is configured:
- The user's oldest active session is terminated
- New call is allowed to proceed
- Provides flexible session management
- May cause unexpected call terminations

```yaml
redis:
  enable_session_limits: true
  max_sessions_per_user: 5
  session_limit_action: "terminate_oldest"
```

## Deployment Scenarios

### Production Environment

Recommended configuration for production deployments:

```yaml
redis:
  enabled: true
  host: "redis-service"
  port: 6379
  password: "${REDIS_PASSWORD}"
  database: 0
  pool_size: 20
  min_idle_conns: 10
  
  # Conservative session limits
  enable_session_limits: true
  max_sessions_per_user: 3
  session_limit_action: "reject"
```

**Rationale:**
- Low session limit (3) prevents resource abuse
- Reject action provides predictable behavior
- High connection pool for performance

### Development Environment

Configuration for development and testing:

```yaml
redis:
  enabled: true
  host: "localhost"
  port: 6379
  database: 1  # Separate database for dev
  
  # Relaxed session limits for testing
  enable_session_limits: true
  max_sessions_per_user: 10
  session_limit_action: "terminate_oldest"
```

**Rationale:**
- Higher session limit allows extensive testing
- Terminate oldest action for flexible testing scenarios
- Separate Redis database for isolation

### High-Volume Environment

Configuration for high-traffic deployments:

```yaml
redis:
  enabled: true
  host: "redis-cluster"
  port: 6379
  password: "${REDIS_PASSWORD}"
  pool_size: 50
  min_idle_conns: 25
  
  # Balanced session limits
  enable_session_limits: true
  max_sessions_per_user: 5
  session_limit_action: "reject"
```

## Monitoring and Observability

### Metrics

Session limits generate several metrics for monitoring:

```prometheus
# Active sessions per user
active_sessions_per_user{username="alice"} 3

# Total session limit rejections
session_limit_rejections_total 42

# Total active sessions
active_sessions_total 156
```

### Log Messages

Session limit events are logged at INFO level:

```json
{
  "timestamp": "2025-05-29T10:15:30Z",
  "level": "info",
  "message": "Session limit exceeded",
  "username": "alice",
  "current_sessions": 5,
  "max_sessions": 5,
  "action": "reject",
  "call_id": "abc123-def456"
}
```

```json
{
  "timestamp": "2025-05-29T10:16:45Z",
  "level": "info", 
  "message": "Terminated oldest session for new call",
  "username": "bob",
  "terminated_session": "old123-session",
  "new_call_id": "new456-call"
}
```

### Health Checks

Monitor session limits health through:

1. **Redis Connectivity**: Ensure Redis is accessible
2. **Session Cleanup**: Verify sessions are removed on call termination
3. **Limit Enforcement**: Test that limits are actually enforced

## Testing and Validation

### Test Scripts

The repository includes comprehensive test scripts:

#### Basic Session Limits Test
```bash
python3 test_simple_session_limits.py
```

#### Same User Limits Test
```bash
python3 test_same_user_limits.py
```

#### Session Limits Verification
```bash
python3 verify_session_limits.py
```

#### Stress Testing
```bash
python3 test_session_limits_stress.py
```

### Manual Testing

#### Using gRPC API

```python
import grpc
from b2bua.v1.b2bua_pb2_grpc import B2BUACallServiceStub
from b2bua.v1 import b2bua_pb2

# Connect to B2BUA
channel = grpc.insecure_channel('localhost:50051')
stub = B2BUACallServiceStub(channel)

# Test session limits
for i in range(6):  # Attempt 6 calls (should hit limit at 5)
    request = b2bua_pb2.InitiateCallRequest(
        from_uri=f"sip:testuser@example.com",
        to_uri="sip:target@example.com",
        sdp="v=0\r\no=- 0 0 IN IP4 127.0.0.1..."
    )
    
    try:
        response = stub.InitiateCall(request)
        print(f"Call {i+1}: Success - {response.call_id}")
    except grpc.RpcError as e:
        print(f"Call {i+1}: Failed - {e.details()}")
```

#### Using SIP Client

Test with a SIP client (like PJSIP) by creating multiple concurrent calls from the same user account.

## Troubleshooting

### Common Issues

#### Session Limits Not Working

**Symptoms:**
- All calls are accepted regardless of configuration
- No rejection logs in server output

**Solutions:**
1. Verify Redis is enabled: `redis.enabled: true`
2. Check session limits are enabled: `redis.enable_session_limits: true`
3. Ensure Redis connectivity
4. Verify username extraction from SIP headers

#### Sessions Not Cleaned Up

**Symptoms:**
- Session count keeps growing
- Legitimate calls are rejected due to phantom sessions

**Solutions:**
1. Check call termination handling
2. Verify Redis TTL settings
3. Monitor for connection issues to Redis
4. Review session cleanup code

#### Redis Connection Issues

**Symptoms:**
- Error logs about Redis connectivity
- Session limits not enforced during Redis outages

**Solutions:**
1. Check Redis server status
2. Verify connection parameters
3. Implement Redis high availability
4. Monitor Redis performance

### Debug Commands

#### Check Active Sessions in Redis
```bash
# Connect to Redis
redis-cli

# List all session keys
KEYS session:*

# List user session tracking
KEYS user_sessions:*

# Check specific user's sessions
SMEMBERS user_sessions:alice

# Get session count for user
SCARD user_sessions:alice
```

#### View Session Data
```bash
# Get session details
GET session:abc123-session-id

# Check session TTL
TTL session:abc123-session-id
```

## Best Practices

### Security Considerations

1. **Redis Security**: Secure Redis with authentication and network isolation
2. **Session Data**: Avoid storing sensitive information in session data
3. **User Privacy**: Log usernames carefully considering privacy requirements

### Performance Optimization

1. **Connection Pooling**: Configure adequate Redis connection pools
2. **TTL Management**: Set appropriate session TTLs to prevent memory leaks
3. **Cleanup Frequency**: Regular cleanup of expired sessions

### Operational Guidelines

1. **Monitoring**: Implement comprehensive monitoring of session metrics
2. **Alerting**: Set up alerts for session limit violations and Redis issues
3. **Capacity Planning**: Monitor session usage patterns for capacity planning
4. **Testing**: Regular testing of session limit functionality

## Advanced Configuration

### Custom Session Identification

For advanced use cases, session tracking can be customized by modifying the user extraction logic in the codebase. The current implementation extracts the user part from the SIP From header URI.

### Integration with External Systems

Session limits can be integrated with external billing or management systems:

1. **Custom Metrics**: Export session data to external monitoring
2. **Dynamic Limits**: Implement per-user dynamic limits based on account type
3. **Notification Systems**: Send alerts when users hit session limits

### Clustering and High Availability

For clustered deployments:

1. **Redis Cluster**: Use Redis cluster for distributed session tracking
2. **Consistency**: Ensure session data consistency across B2BUA instances
3. **Failover**: Implement proper failover handling for Redis outages

## Migration and Upgrades

### Enabling Session Limits on Existing Deployment

When enabling session limits on an existing deployment:

1. **Gradual Rollout**: Start with high limits and gradually reduce
2. **Monitoring**: Monitor session patterns before enforcing strict limits
3. **User Communication**: Inform users about new session limits

### Configuration Changes

Session limit configuration can be updated without restart in most cases:

1. **Redis Configuration**: Requires restart for connection changes
2. **Limit Values**: Can be updated dynamically via configuration management
3. **Action Changes**: May require restart depending on implementation

## Conclusion

Session limits provide essential resource protection and fair usage enforcement for SIP B2BUA deployments. Proper configuration, monitoring, and testing ensure reliable operation while preventing resource abuse.

For additional support or questions, refer to the main project documentation or open an issue in the project repository.

## Per-User Session Limits

Voice Ferry supports different session limits for different users, allowing for fine-grained control over system resources.

### Configuration

You can configure per-user limits in two ways:

1. **Static Configuration**: Define limits in the `config.yaml` file:

```yaml
redis:
  # Enable session limits
  enable_session_limits: true
  # Default limit for users without specific limits
  max_sessions_per_user: 5
  # Per-user limits
  user_session_limits:
    high_volume_user: 20
    standard_user: 5
    restricted_user: 2
    unlimited_user: 0  # 0 means unlimited
```

2. **API Management**: Use the REST API to manage limits at runtime:

```
# Get a user's session limit
GET /api/sessions/limits/username

# Set a user's session limit
PUT /api/sessions/limits/username
{ "limit": 10 }

# Remove a user-specific limit (revert to default)
DELETE /api/sessions/limits/username
```

### Special Values

- **Default Limit**: Users without specific limits use the global `max_sessions_per_user` value
- **Unlimited Sessions**: Setting a user's limit to `0` or a negative number removes any limit for that user
- **Restricted Access**: Setting a user's limit to `1` ensures they can only have one active call at a time

### Persistence

User-specific limits are stored in Redis for persistence across service restarts. The system automatically loads all user-specific limits when starting up.

### Monitoring

You can monitor per-user session usage through the web UI dashboard or the API:

```
# Get current sessions for a specific user
GET /api/sessions/users/username

# Get all users with their current session counts and limits
GET /api/sessions/users
```
