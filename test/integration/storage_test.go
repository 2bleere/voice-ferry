package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/etcd"
	"github.com/2bleere/voice-ferry/pkg/redis"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

type StorageIntegrationTestSuite struct {
	suite.Suite
	etcdClient  *etcd.Client
	redisClient *redis.Client
	ctx         context.Context
	cancel      context.CancelFunc
}

func TestStorageIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(StorageIntegrationTestSuite))
}

func (suite *StorageIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Try to connect to etcd (if available)
	etcdConfig := &config.EtcdConfig{
		Enabled:     true,
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	var err error
	suite.etcdClient, err = etcd.NewClient(etcdConfig)
	if err != nil {
		suite.T.Logf("Etcd not available for integration test: %v", err)
		suite.etcdClient = nil
	}

	// Try to connect to Redis (if available)
	redisConfig := &config.RedisConfig{
		Enabled:      true,
		Host:         "localhost",
		Port:         6379,
		Database:     0,
		PoolSize:     10,
		MinIdleConns: 5,
		Timeout:      5,
	}

	suite.redisClient, err = redis.NewClient(redisConfig)
	if err != nil {
		suite.T.Logf("Redis not available for integration test: %v", err)
		suite.redisClient = nil
	}
}

func (suite *StorageIntegrationTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}

	if suite.etcdClient != nil {
		suite.etcdClient.Close()
	}

	if suite.redisClient != nil {
		suite.redisClient.Close()
	}
}

func (suite *StorageIntegrationTestSuite) TestEtcdIntegration() {
	if suite.etcdClient == nil {
		suite.T.Skip("Etcd not available for integration test")
		return
	}

	// Test health check
	err := suite.etcdClient.HealthCheck(suite.ctx)
	require.NoError(suite.T, err, "Etcd health check should succeed")

	// Test routing rule storage and retrieval
	testRule := &v1.RoutingRule{
		RuleId:      "test-rule-" + time.Now().Format("20060102150405"),
		Name:        "Test Rule",
		Priority:    100,
		Description: "Integration test rule",
		Enabled:     true,
		Conditions: &v1.RoutingConditions{
			RequestUriRegex: "sip:.*@example.com",
			FromUriRegex:    "sip:.*@test.com",
			SourceIps:       []string{"192.168.1.0/24"},
			HeaderConditions: map[string]string{
				"X-Test-Header": "test-value",
			},
		},
		Actions: &v1.RoutingActions{
			NextHopUri: "sip:192.168.1.100:5060",
			AddHeaders: map[string]string{
				"X-Route": "integration-test",
			},
			RemoveHeaders: []string{"X-Remove-Me"},
		},
		CreatedAt: timestamppb.New(time.Now()),
		UpdatedAt: timestamppb.New(time.Now()),
	}

	// Store the rule
	err = suite.etcdClient.StoreRoutingRule(suite.ctx, testRule)
	require.NoError(suite.T, err, "Should store routing rule in etcd")

	// Retrieve the rule
	retrievedRule, err := suite.etcdClient.GetRoutingRule(suite.ctx, testRule.RuleId)
	require.NoError(suite.T, err, "Should retrieve routing rule from etcd")
	assert.Equal(suite.T, testRule.RuleId, retrievedRule.RuleId)
	assert.Equal(suite.T, testRule.Name, retrievedRule.Name)
	assert.Equal(suite.T, testRule.Priority, retrievedRule.Priority)

	// List all rules
	rules, err := suite.etcdClient.ListRoutingRules(suite.ctx)
	require.NoError(suite.T, err, "Should list routing rules from etcd")
	assert.GreaterOrEqual(suite.T, len(rules), 1, "Should have at least one rule")

	// Test configuration storage
	testConfig := map[string]interface{}{
		"test_setting": "test_value",
		"test_number":  42,
		"test_bool":    true,
	}

	err = suite.etcdClient.StoreConfig(suite.ctx, "test_config", testConfig)
	require.NoError(suite.T, err, "Should store configuration in etcd")

	var retrievedConfig map[string]interface{}
	err = suite.etcdClient.GetConfig(suite.ctx, "test_config", &retrievedConfig)
	require.NoError(suite.T, err, "Should retrieve configuration from etcd")
	assert.Equal(suite.T, testConfig["test_setting"], retrievedConfig["test_setting"])

	// Test session storage
	sessionData := map[string]string{
		"call_id":    "test-call-123",
		"session_id": "test-session-456",
		"state":      "active",
	}

	err = suite.etcdClient.StoreSession(suite.ctx, "test-session-456", sessionData)
	require.NoError(suite.T, err, "Should store session in etcd")

	var retrievedSession map[string]string
	err = suite.etcdClient.GetSession(suite.ctx, "test-session-456", &retrievedSession)
	require.NoError(suite.T, err, "Should retrieve session from etcd")
	assert.Equal(suite.T, sessionData["call_id"], retrievedSession["call_id"])

	// Cleanup
	err = suite.etcdClient.DeleteRoutingRule(suite.ctx, testRule.RuleId)
	assert.NoError(suite.T, err, "Should delete routing rule from etcd")

	err = suite.etcdClient.DeleteSession(suite.ctx, "test-session-456")
	assert.NoError(suite.T, err, "Should delete session from etcd")
}

func (suite *StorageIntegrationTestSuite) TestRedisIntegration() {
	if suite.redisClient == nil {
		suite.T.Skip("Redis not available for integration test")
		return
	}

	// Test health check
	err := suite.redisClient.HealthCheck(suite.ctx)
	require.NoError(suite.T, err, "Redis health check should succeed")

	// Test session storage and retrieval
	sessionID := "test-session-" + time.Now().Format("20060102150405")
	sessionData := map[string]interface{}{
		"call_id":     "test-call-123",
		"user_agent":  "Test UA",
		"remote_addr": "192.168.1.100",
		"state":       "active",
		"created_at":  time.Now().Unix(),
	}

	err = suite.redisClient.StoreSession(suite.ctx, sessionID, sessionData, 1*time.Hour)
	require.NoError(suite.T, err, "Should store session in Redis")

	var retrievedSession map[string]interface{}
	err = suite.redisClient.GetSessionData(suite.ctx, sessionID, &retrievedSession)
	require.NoError(suite.T, err, "Should retrieve session from Redis")
	assert.Equal(suite.T, sessionData["call_id"], retrievedSession["call_id"])
	assert.Equal(suite.T, sessionData["user_agent"], retrievedSession["user_agent"])

	// Test session TTL extension
	err = suite.redisClient.ExtendSession(suite.ctx, sessionID, 2*time.Hour)
	require.NoError(suite.T, err, "Should extend session TTL")

	// Test call state storage
	callID := "test-call-" + time.Now().Format("20060102150405")
	callState := map[string]interface{}{
		"session_id":  sessionID,
		"call_id":     callID,
		"state":       "ringing",
		"direction":   "inbound",
		"start_time":  time.Now().Unix(),
		"remote_addr": "192.168.1.200",
	}

	err = suite.redisClient.StoreCallState(suite.ctx, callID, callState)
	require.NoError(suite.T, err, "Should store call state in Redis")

	var retrievedCallState map[string]interface{}
	err = suite.redisClient.GetCallState(suite.ctx, callID, &retrievedCallState)
	require.NoError(suite.T, err, "Should retrieve call state from Redis")
	assert.Equal(suite.T, callState["session_id"], retrievedCallState["session_id"])

	// Test active calls listing
	activeCalls, err := suite.redisClient.ListActiveCalls(suite.ctx)
	require.NoError(suite.T, err, "Should list active calls")
	assert.Contains(suite.T, activeCalls, callID, "Should contain our test call")

	// Test cache operations
	cacheKey := "test-cache-key"
	cacheData := map[string]string{
		"user_id":   "user123",
		"username":  "testuser",
		"last_seen": time.Now().Format(time.RFC3339),
	}

	err = suite.redisClient.StoreCache(suite.ctx, cacheKey, cacheData, 30*time.Minute)
	require.NoError(suite.T, err, "Should store data in cache")

	var retrievedCacheData map[string]string
	err = suite.redisClient.GetCache(suite.ctx, cacheKey, &retrievedCacheData)
	require.NoError(suite.T, err, "Should retrieve data from cache")
	assert.Equal(suite.T, cacheData["user_id"], retrievedCacheData["user_id"])

	// Test metrics operations
	metricName := "test_metric"
	err = suite.redisClient.IncrementMetric(suite.ctx, metricName)
	require.NoError(suite.T, err, "Should increment metric")

	err = suite.redisClient.IncrementMetricBy(suite.ctx, metricName, 5)
	require.NoError(suite.T, err, "Should increment metric by amount")

	metricValue, err := suite.redisClient.GetMetric(suite.ctx, metricName)
	require.NoError(suite.T, err, "Should get metric value")
	assert.Equal(suite.T, int64(6), metricValue, "Metric should be 6 (1 + 5)")

	// Test all metrics retrieval
	allMetrics, err := suite.redisClient.GetAllMetrics(suite.ctx)
	require.NoError(suite.T, err, "Should get all metrics")
	assert.Contains(suite.T, allMetrics, metricName, "Should contain our test metric")

	// Test pub/sub functionality
	testChannel := "test-channel"
	testMessage := map[string]string{
		"event": "test_event",
		"data":  "test data",
	}

	err = suite.redisClient.Publish(suite.ctx, testChannel, testMessage)
	require.NoError(suite.T, err, "Should publish message to channel")

	// Test pipeline operations
	pipeline := suite.redisClient.Pipeline()
	assert.NotNil(suite.T, pipeline, "Should get Redis pipeline")

	// Test active sessions listing
	activeSessions, err := suite.redisClient.GetActiveSessionIDs(suite.ctx)
	require.NoError(suite.T, err, "Should list active sessions")
	assert.Contains(suite.T, activeSessions, sessionID, "Should contain our test session")

	// Cleanup
	err = suite.redisClient.DeleteSession(suite.ctx, sessionID)
	assert.NoError(suite.T, err, "Should delete session from Redis")

	err = suite.redisClient.DeleteCallState(suite.ctx, callID)
	assert.NoError(suite.T, err, "Should delete call state from Redis")

	err = suite.redisClient.DeleteCache(suite.ctx, cacheKey)
	assert.NoError(suite.T, err, "Should delete cache data from Redis")
}

func (suite *StorageIntegrationTestSuite) TestEtcdWatchFunctionality() {
	if suite.etcdClient == nil {
		suite.T.Skip("Etcd not available for integration test")
		return
	}

	// Test watch functionality
	watchCtx, watchCancel := context.WithTimeout(suite.ctx, 5*time.Second)
	defer watchCancel()

	watchChan := suite.etcdClient.WatchRoutingRules(watchCtx)
	assert.NotNil(suite.T, watchChan, "Should get watch channel")

	// This is a basic test to ensure watch doesn't panic
	// In a real scenario, you would test actual watch events
	watchCancel() // Cancel to stop watching
}

func (suite *StorageIntegrationTestSuite) TestStorageErrors() {
	if suite.etcdClient != nil {
		// Test non-existent rule retrieval
		_, err := suite.etcdClient.GetRoutingRule(suite.ctx, "non-existent-rule")
		assert.Error(suite.T, err, "Should return error for non-existent rule")

		// Test non-existent config retrieval
		var config map[string]interface{}
		err = suite.etcdClient.GetConfig(suite.ctx, "non-existent-config", &config)
		assert.Error(suite.T, err, "Should return error for non-existent config")

		// Test non-existent session retrieval
		var session map[string]interface{}
		err = suite.etcdClient.GetSession(suite.ctx, "non-existent-session", &session)
		assert.Error(suite.T, err, "Should return error for non-existent session")
	}

	if suite.redisClient != nil {
		// Test non-existent session retrieval
		_, err := suite.redisClient.GetSession(suite.ctx, "non-existent-session")
		assert.Error(suite.T, err, "Should return error for non-existent session")

		// Test non-existent call state retrieval
		var callState map[string]interface{}
		err = suite.redisClient.GetCallState(suite.ctx, "non-existent-call", &callState)
		assert.Error(suite.T, err, "Should return error for non-existent call state")

		// Test non-existent cache retrieval
		var cacheData map[string]interface{}
		err = suite.redisClient.GetCache(suite.ctx, "non-existent-cache", &cacheData)
		assert.Error(suite.T, err, "Should return error for non-existent cache data")

		// Test non-existent metric retrieval
		_, err = suite.redisClient.GetMetric(suite.ctx, "non-existent-metric")
		assert.Error(suite.T, err, "Should return error for non-existent metric")
	}
}
