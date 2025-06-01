package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes for different data types
	SessionPrefix      = "voice-ferry-c4:session:"
	CallStatePrefix    = "voice-ferry-c4:call:"
	CachePrefix        = "voice-ferry-c4:cache:"
	MetricsPrefix      = "voice-ferry-c4:metrics:"
	UserSessionsPrefix = "voice-ferry-c4:user-sessions:"

	// Default TTLs
	DefaultSessionTTL = 4 * time.Hour
	DefaultCacheTTL   = 30 * time.Minute
	CallStateTTL      = 24 * time.Hour

	// Session limit actions
	SessionLimitActionReject          = "reject"
	SessionLimitActionTerminateOldest = "terminate_oldest"
)

// Client wraps Redis client with domain-specific methods
type Client struct {
	client *redis.Client
	cfg    *config.RedisConfig
}

// NewClient creates a new Redis client
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("redis is disabled")
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,

		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,

		DialTimeout:  time.Duration(cfg.Timeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
	}

	if cfg.TLS.Enabled {
		// TODO: Add TLS configuration
		opts.TLSConfig = nil
	}

	client := redis.NewClient(opts)

	// Initialize user session limits map if needed
	if cfg.UserSessionLimits == nil {
		cfg.UserSessionLimits = make(map[string]int)
	}

	redisClient := &Client{
		client: client,
		cfg:    cfg,
	}

	// Load any existing per-user session limits from Redis
	if cfg.EnableSessionLimits {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := redisClient.LoadAllUserSessionLimits(ctx)
		if err != nil {
			// Log the error but continue - not fatal
			fmt.Printf("Warning: failed to load user session limits: %v\n", err)
		}
	}

	return redisClient, nil
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// HealthCheck performs a health check on Redis
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// StoreSession stores session data in Redis
func (c *Client) StoreSession(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) error {
	key := SessionPrefix + sessionID

	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Store the session data
	err = c.client.Set(ctx, key, jsonData, ttl).Err()
	if err != nil {
		return err
	}

	// If session data contains username, track it for session limits
	if dataMap, ok := data.(map[string]interface{}); ok {
		if username, ok := dataMap["username"].(string); ok && username != "" {
			err = c.trackUserSession(ctx, username, sessionID, ttl)
			if err != nil {
				return fmt.Errorf("failed to track user session: %w", err)
			}
		}
	}

	return nil
}

// GetSession retrieves session data from Redis
func (c *Client) GetSession(ctx context.Context, sessionID string) (string, error) {
	key := SessionPrefix + sessionID

	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("session %s not found", sessionID)
		}
		return "", fmt.Errorf("failed to get session from redis: %w", err)
	}

	return data, nil
}

// GetSessionData retrieves session data from Redis and unmarshals it
func (c *Client) GetSessionData(ctx context.Context, sessionID string, target interface{}) error {
	data, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), target)
}

// DeleteSession deletes session data from Redis
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	key := SessionPrefix + sessionID

	// First, get the session data to find the username
	var sessionData map[string]interface{}
	err := c.GetSessionData(ctx, sessionID, &sessionData)
	if err == nil {
		// If there's a username, remove this session from user tracking
		if username, ok := sessionData["username"].(string); ok && username != "" {
			userKey := UserSessionsPrefix + username
			c.client.SRem(ctx, userKey, sessionID)
			// Continue regardless of error here
		}
	}

	// Delete the session itself
	return c.client.Del(ctx, key).Err()
}

// ExtendSession extends the TTL of a session
func (c *Client) ExtendSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := SessionPrefix + sessionID

	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	return c.client.Expire(ctx, key, ttl).Err()
}

// StoreCallState stores call state information
func (c *Client) StoreCallState(ctx context.Context, callID string, state interface{}) error {
	key := CallStatePrefix + callID

	jsonData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal call state: %w", err)
	}

	return c.client.Set(ctx, key, jsonData, CallStateTTL).Err()
}

// GetCallState retrieves call state information
func (c *Client) GetCallState(ctx context.Context, callID string, target interface{}) error {
	key := CallStatePrefix + callID

	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("call state %s not found", callID)
		}
		return fmt.Errorf("failed to get call state from redis: %w", err)
	}

	return json.Unmarshal([]byte(data), target)
}

// DeleteCallState deletes call state information
func (c *Client) DeleteCallState(ctx context.Context, callID string) error {
	key := CallStatePrefix + callID
	return c.client.Del(ctx, key).Err()
}

// ListActiveCalls returns all active call IDs
func (c *Client) ListActiveCalls(ctx context.Context) ([]string, error) {
	pattern := CallStatePrefix + "*"

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list active calls: %w", err)
	}

	// Extract call IDs from keys
	callIDs := make([]string, len(keys))
	for i, key := range keys {
		callIDs[i] = key[len(CallStatePrefix):]
	}

	return callIDs, nil
}

// StoreCache stores arbitrary data in cache
func (c *Client) StoreCache(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	fullKey := CachePrefix + key

	if ttl == 0 {
		ttl = DefaultCacheTTL
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	return c.client.Set(ctx, fullKey, jsonData, ttl).Err()
}

// GetCache retrieves data from cache
func (c *Client) GetCache(ctx context.Context, key string, target interface{}) error {
	fullKey := CachePrefix + key

	data, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache key %s not found", key)
		}
		return fmt.Errorf("failed to get cache data: %w", err)
	}

	return json.Unmarshal([]byte(data), target)
}

// DeleteCache deletes data from cache
func (c *Client) DeleteCache(ctx context.Context, key string) error {
	fullKey := CachePrefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// IncrementMetric increments a metric counter
func (c *Client) IncrementMetric(ctx context.Context, metric string) error {
	key := MetricsPrefix + metric
	return c.client.Incr(ctx, key).Err()
}

// GetMetric gets a metric value
func (c *Client) GetMetric(ctx context.Context, metric string) (int64, error) {
	key := MetricsPrefix + metric
	return c.client.Get(ctx, key).Int64()
}

// SetMetric sets a metric value
func (c *Client) SetMetric(ctx context.Context, metric string, value int64) error {
	key := MetricsPrefix + metric
	return c.client.Set(ctx, key, value, 0).Err()
}

// IncrementMetricBy increments a metric by a specific amount
func (c *Client) IncrementMetricBy(ctx context.Context, metric string, amount int64) error {
	key := MetricsPrefix + metric
	return c.client.IncrBy(ctx, key, amount).Err()
}

// GetAllMetrics retrieves all metrics
func (c *Client) GetAllMetrics(ctx context.Context) (map[string]string, error) {
	pattern := MetricsPrefix + "*"

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metric values: %w", err)
	}

	metrics := make(map[string]string)
	for i, key := range keys {
		metricName := key[len(MetricsPrefix):]
		if values[i] != nil {
			metrics[metricName] = values[i].(string)
		}
	}

	return metrics, nil
}

// Pipeline returns a Redis pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// Watch watches Redis keys for changes (useful for cache invalidation)
func (c *Client) Watch(ctx context.Context, keys ...string) error {
	return c.client.Watch(ctx, func(tx *redis.Tx) error {
		// This is a placeholder - implement specific watch logic as needed
		return nil
	}, keys...)
}

// Publish publishes a message to a Redis channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.client.Publish(ctx, channel, data).Err()
}

// Subscribe subscribes to Redis channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// FlushDB flushes the current database (use with caution!)
func (c *Client) FlushDB(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// GetActiveSessionIDs returns all active session IDs
func (c *Client) GetActiveSessionIDs(ctx context.Context) ([]string, error) {
	pattern := SessionPrefix + "*"

	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}

	// Extract session IDs from keys
	sessionIDs := make([]string, len(keys))
	for i, key := range keys {
		sessionIDs[i] = key[len(SessionPrefix):]
	}

	return sessionIDs, nil
}

// StoreSessionString stores session data as string in Redis
func (c *Client) StoreSessionString(ctx context.Context, sessionID string, data string, ttl time.Duration) error {
	key := SessionPrefix + sessionID

	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// trackUserSession adds a session ID to a user's active sessions
func (c *Client) trackUserSession(ctx context.Context, username, sessionID string, ttl time.Duration) error {
	key := UserSessionsPrefix + username

	// Add to the set of user's sessions
	err := c.client.SAdd(ctx, key, sessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to add session to user tracking: %w", err)
	}

	// Ensure the set expires (refreshed on activity)
	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	return c.client.Expire(ctx, key, ttl).Err()
}

// GetUserSessionCount returns the number of active sessions for a user
func (c *Client) GetUserSessionCount(ctx context.Context, username string) (int64, error) {
	key := UserSessionsPrefix + username
	return c.client.SCard(ctx, key).Result()
}

// GetUserSessions returns all session IDs for a user
func (c *Client) GetUserSessions(ctx context.Context, username string) ([]string, error) {
	key := UserSessionsPrefix + username
	return c.client.SMembers(ctx, key).Result()
}

// UntrackUserSession removes a session from a user's active sessions
func (c *Client) UntrackUserSession(ctx context.Context, username, sessionID string) error {
	key := UserSessionsPrefix + username
	return c.client.SRem(ctx, key, sessionID).Err()
}

// GetOldestUserSession returns the oldest session ID for a user
func (c *Client) GetOldestUserSession(ctx context.Context, username string) (string, error) {
	key := UserSessionsPrefix + username

	// Get all session IDs for this user
	sessionIDs, err := c.client.SMembers(ctx, key).Result()
	if err != nil {
		return "", err
	}

	if len(sessionIDs) == 0 {
		return "", fmt.Errorf("no sessions found for user %s", username)
	}

	// Track the oldest session
	var oldestSession string
	var oldestTime time.Time

	for _, sessionID := range sessionIDs {
		// Get session data to check creation time
		var sessionData map[string]interface{}
		err = c.GetSessionData(ctx, sessionID, &sessionData)
		if err != nil {
			continue // Skip this session if we can't get data
		}

		// Parse the creation time
		var creationTime time.Time
		if createdAtStr, ok := sessionData["created_at"].(string); ok {
			creationTime, err = time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				continue
			}
		} else if createdAtUnix, ok := sessionData["created_at"].(float64); ok {
			creationTime = time.Unix(int64(createdAtUnix), 0)
		} else {
			continue
		}

		// If this is the first or older than our current oldest, update
		if oldestSession == "" || creationTime.Before(oldestTime) {
			oldestSession = sessionID
			oldestTime = creationTime
		}
	}

	if oldestSession == "" {
		return "", fmt.Errorf("could not determine oldest session for user %s", username)
	}

	return oldestSession, nil
}

// CheckSessionLimit checks if user is below session limit and handles if over limit
func (c *Client) CheckSessionLimit(ctx context.Context, username string) (bool, error) {
	// If session limits not enabled, always allow
	if !c.cfg.EnableSessionLimits {
		return true, nil
	}

	// Count current sessions for this user
	count, err := c.GetUserSessionCount(ctx, username)
	if err != nil {
		return false, fmt.Errorf("failed to count user sessions: %w", err)
	}

	// Get user-specific limit if configured, otherwise use default
	userLimit := c.cfg.MaxSessionsPerUser
	if specificLimit, exists := c.cfg.UserSessionLimits[username]; exists {
		userLimit = specificLimit
	}

	// If limit is 0 or negative, no limit is enforced for this user
	if userLimit <= 0 {
		return true, nil
	}

	// If under limit, allow
	if count < int64(userLimit) {
		return true, nil
	}

	// Over limit - handle based on configured action
	if c.cfg.SessionLimitAction == SessionLimitActionTerminateOldest {
		oldestSession, err := c.GetOldestUserSession(ctx, username)
		if err != nil {
			return false, fmt.Errorf("failed to get oldest session: %w", err)
		}

		// Delete the oldest session
		err = c.DeleteSession(ctx, oldestSession)
		if err != nil {
			return false, fmt.Errorf("failed to terminate oldest session: %w", err)
		}

		// Successfully terminated oldest session, now allow this one
		return true, nil
	}

	// Default action is to reject
	return false, nil
}

// SetUserSessionLimit sets a specific session limit for a user
func (c *Client) SetUserSessionLimit(ctx context.Context, username string, limit int) error {
	// Ensure UserSessionLimits map is initialized
	if c.cfg.UserSessionLimits == nil {
		c.cfg.UserSessionLimits = make(map[string]int)
	}

	// Set the user-specific limit
	c.cfg.UserSessionLimits[username] = limit

	// Store the user-specific limit in Redis for persistence
	key := "voice-ferry-c4:user-limit:" + username
	err := c.client.Set(ctx, key, limit, 0).Err() // No expiration
	if err != nil {
		return fmt.Errorf("failed to store user session limit in Redis: %w", err)
	}

	return nil
}

// GetUserSessionLimit gets the specific session limit for a user
func (c *Client) GetUserSessionLimit(ctx context.Context, username string) (int, error) {
	// Check if there's a specific limit in memory
	if c.cfg.UserSessionLimits != nil {
		if limit, exists := c.cfg.UserSessionLimits[username]; exists {
			return limit, nil
		}
	}

	// Try to get from Redis
	key := "voice-ferry-c4:user-limit:" + username
	limit, err := c.client.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			// No specific limit set, return the default
			return c.cfg.MaxSessionsPerUser, nil
		}
		return 0, fmt.Errorf("failed to get user session limit from Redis: %w", err)
	}

	// Cache the limit in memory
	if c.cfg.UserSessionLimits == nil {
		c.cfg.UserSessionLimits = make(map[string]int)
	}
	c.cfg.UserSessionLimits[username] = limit

	return limit, nil
}

// DeleteUserSessionLimit removes a specific session limit for a user
func (c *Client) DeleteUserSessionLimit(ctx context.Context, username string) error {
	// Remove from memory
	if c.cfg.UserSessionLimits != nil {
		delete(c.cfg.UserSessionLimits, username)
	}

	// Remove from Redis
	key := "voice-ferry-c4:user-limit:" + username
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete user session limit from Redis: %w", err)
	}

	return nil
}

// LoadAllUserSessionLimits loads all user-specific session limits from Redis
func (c *Client) LoadAllUserSessionLimits(ctx context.Context) error {
	// Initialize the map if it doesn't exist
	if c.cfg.UserSessionLimits == nil {
		c.cfg.UserSessionLimits = make(map[string]int)
	} else {
		// Clear existing entries to avoid stale data
		for k := range c.cfg.UserSessionLimits {
			delete(c.cfg.UserSessionLimits, k)
		}
	}

	// Get all user limit keys
	pattern := "voice-ferry-c4:user-limit:*"
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to list user limits: %w", err)
	}

	// No keys found
	if len(keys) == 0 {
		return nil
	}

	// Get all values in a single operation
	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("failed to get user limits: %w", err)
	}

	// Process each key-value pair
	for i, key := range keys {
		if values[i] == nil {
			continue
		}

		// Extract username from key
		username := key[len("voice-ferry-c4:user-limit:"):]

		// Convert value to int
		limitStr, ok := values[i].(string)
		if !ok {
			continue
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			continue
		}

		// Store in memory
		c.cfg.UserSessionLimits[username] = limit
	}

	return nil
}
