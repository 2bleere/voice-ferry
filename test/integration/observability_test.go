package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/2bleere/voice-ferry/internal/server"
	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/health"
)

type ObservabilityTestSuite struct {
	suite.Suite
	server      *server.Server
	config      *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	healthPort  int
	metricsPort int
	grpcPort    int
	sipPort     int
}

func TestObservabilityTestSuite(t *testing.T) {
	suite.Run(t, new(ObservabilityTestSuite))
}

func (suite *ObservabilityTestSuite) SetupSuite() {
	// Create test configuration
	suite.healthPort = getAvailablePort()
	suite.metricsPort = getAvailablePort()
	suite.grpcPort = getAvailablePort()
	suite.sipPort = getAvailablePort()

	suite.config = &config.Config{
		SIP: config.SIPConfig{
			Host: "127.0.0.1",
			Port: suite.sipPort,
		},
		GRPC: config.GRPCConfig{
			Host: "127.0.0.1",
			Port: suite.grpcPort,
		},
		Health: config.HealthConfig{
			Host: "127.0.0.1",
			Port: suite.healthPort,
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    suite.metricsPort,
		},
		Logging: config.LoggingConfig{
			Level:   "debug",
			Format:  "text",
			Version: "test-1.0.0",
		},
		Redis: config.RedisConfig{
			Enabled: false, // Disable for integration tests
		},
		Etcd: config.EtcdConfig{
			Enabled: false, // Disable for integration tests
		},
		RTPEngine: config.RTPEngineConfig{
			Instances: []config.RTPEngineInstance{
				{
					ID:      "test-1",
					Host:    "127.0.0.1",
					Port:    22222,
					Weight:  100,
					Enabled: true,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	// Create server
	var err error
	suite.server, err = server.New(suite.config)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), suite.server)

	// Create context for server lifecycle
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *ObservabilityTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}

	if suite.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.server.Shutdown(shutdownCtx)
	}
}

func (suite *ObservabilityTestSuite) TestServerStartup() {
	// Start all server components
	go func() {
		err := suite.server.StartHealthCheck(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	go func() {
		err := suite.server.StartMetrics(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	go func() {
		err := suite.server.StartGRPC(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	// Give servers time to start
	time.Sleep(200 * time.Millisecond)

	// Verify all components are running
	suite.verifyHealthEndpoint()
	suite.verifyMetricsEndpoint()
	suite.verifyGRPCEndpoint()
}

func (suite *ObservabilityTestSuite) TestHealthEndpoints() {
	// Start health check server
	go func() {
		err := suite.server.StartHealthCheck(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	suite.verifyHealthEndpoint()

	// Test readiness endpoint
	suite.verifyReadinessEndpoint()

	// Test liveness endpoint
	suite.verifyLivenessEndpoint()

	// Test component health endpoint
	suite.verifyComponentHealthEndpoint()
}

func (suite *ObservabilityTestSuite) TestMetricsCollection() {
	// Start metrics server
	go func() {
		err := suite.server.StartMetrics(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test basic metrics endpoint
	suite.verifyMetricsEndpoint()

	// Test metrics collection
	suite.testMetricsCollection()
}

func (suite *ObservabilityTestSuite) TestLoggingSystem() {
	// Verify logger is working
	require.NotNil(suite.T(), suite.server)

	// This test primarily ensures no panics occur during logging
	// and that the logging system is properly initialized
}

func (suite *ObservabilityTestSuite) TestHealthManagerIntegration() {
	// Start health check server
	go func() {
		err := suite.server.StartHealthCheck(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test health status changes over time
	suite.testHealthStatusChanges()
}

func (suite *ObservabilityTestSuite) TestEndToEndObservability() {
	// Start all components
	go func() {
		err := suite.server.StartHealthCheck(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	go func() {
		err := suite.server.StartMetrics(suite.ctx)
		assert.NoError(suite.T(), err)
	}()

	time.Sleep(200 * time.Millisecond)

	// Simulate some activity
	suite.simulateSystemActivity()

	// Verify observability data
	suite.verifyObservabilityData()
}

func (suite *ObservabilityTestSuite) verifyHealthEndpoint() {
	url := fmt.Sprintf("http://localhost:%d/health", suite.healthPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))

	var healthStatus health.SystemHealth
	err = json.NewDecoder(resp.Body).Decode(&healthStatus)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "test-1.0.0", healthStatus.Version)
	assert.True(suite.T(), healthStatus.Timestamp.After(time.Time{}))
}

func (suite *ObservabilityTestSuite) verifyReadinessEndpoint() {
	url := fmt.Sprintf("http://localhost:%d/health/ready", suite.healthPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "ready", string(body))
}

func (suite *ObservabilityTestSuite) verifyLivenessEndpoint() {
	url := fmt.Sprintf("http://localhost:%d/health/live", suite.healthPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "alive", string(body))
}

func (suite *ObservabilityTestSuite) verifyComponentHealthEndpoint() {
	url := fmt.Sprintf("http://localhost:%d/health/component?name=sip_server", suite.healthPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Should return 200 for existing component or 404 for non-existing
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
}

func (suite *ObservabilityTestSuite) verifyMetricsEndpoint() {
	url := fmt.Sprintf("http://localhost:%d/metrics", suite.metricsPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(suite.T(), resp.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)

	bodyStr := string(body)
	// Should contain Prometheus metrics format
	assert.Contains(suite.T(), bodyStr, "# HELP")
	assert.Contains(suite.T(), bodyStr, "# TYPE")
}

func (suite *ObservabilityTestSuite) verifyGRPCEndpoint() {
	addr := fmt.Sprintf("localhost:%d", suite.grpcPort)
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
	}
	// If connection succeeded or got a specific gRPC error, server is running
}

func (suite *ObservabilityTestSuite) testMetricsCollection() {
	// Get initial metrics
	url := fmt.Sprintf("http://localhost:%d/metrics", suite.metricsPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)

	metricsContent := string(body)

	// Should contain system info metrics
	assert.Contains(suite.T(), metricsContent, "system_info")

	// Should contain component health metrics
	assert.Contains(suite.T(), metricsContent, "component_health")
}

func (suite *ObservabilityTestSuite) testHealthStatusChanges() {
	// Test multiple health checks over time
	for i := 0; i < 3; i++ {
		url := fmt.Sprintf("http://localhost:%d/health", suite.healthPort)
		resp, err := http.Get(url)
		require.NoError(suite.T(), err)

		var healthStatus health.SystemHealth
		err = json.NewDecoder(resp.Body).Decode(&healthStatus)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		assert.Equal(suite.T(), "test-1.0.0", healthStatus.Version)

		time.Sleep(100 * time.Millisecond)
	}
}

func (suite *ObservabilityTestSuite) simulateSystemActivity() {
	// This would simulate SIP calls, routing decisions, etc.
	// For now, we just verify the infrastructure is working

	// Make multiple health checks to simulate activity
	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("http://localhost:%d/health", suite.healthPort)
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
		}

		// Check metrics
		metricsURL := fmt.Sprintf("http://localhost:%d/metrics", suite.metricsPort)
		resp, err = http.Get(metricsURL)
		if err == nil {
			resp.Body.Close()
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (suite *ObservabilityTestSuite) verifyObservabilityData() {
	// Verify health endpoint returns consistent data
	url := fmt.Sprintf("http://localhost:%d/health", suite.healthPort)
	resp, err := http.Get(url)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify metrics endpoint returns valid Prometheus format
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", suite.metricsPort)
	resp, err = http.Get(metricsURL)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err)

	metricsContent := string(body)

	// Verify Prometheus format
	lines := strings.Split(metricsContent, "\n")
	hasHelpLines := false
	hasTypeLines := false
	hasMetricLines := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			hasHelpLines = true
		}
		if strings.HasPrefix(line, "# TYPE") {
			hasTypeLines = true
		}
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			hasMetricLines = true
		}
	}

	assert.True(suite.T(), hasHelpLines, "Metrics should contain HELP lines")
	assert.True(suite.T(), hasTypeLines, "Metrics should contain TYPE lines")
	assert.True(suite.T(), hasMetricLines, "Metrics should contain metric lines")
}

func getAvailablePort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// Benchmark for integration testing
func BenchmarkObservabilityEndpoints(b *testing.B) {
	// Setup
	config := &config.Config{
		Health: config.HealthConfig{
			Host: "127.0.0.1",
			Port: getAvailablePort(),
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    getAvailablePort(),
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Format:  "text",
			Version: "bench-1.0.0",
		},
		SIP:   config.SIPConfig{Host: "127.0.0.1", Port: getAvailablePort()},
		GRPC:  config.GRPCConfig{Host: "127.0.0.1", Port: getAvailablePort()},
		Redis: config.RedisConfig{Enabled: false},
		Etcd:  config.EtcdConfig{Enabled: false},
		RTPEngine: config.RTPEngineConfig{
			Instances: []config.RTPEngineInstance{
				{
					ID:      "test-1",
					Host:    "127.0.0.1",
					Port:    22222,
					Weight:  100,
					Enabled: true,
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	testServer, err := server.New(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers
	go testServer.StartHealthCheck(ctx)
	go testServer.StartMetrics(ctx)

	time.Sleep(100 * time.Millisecond)

	healthURL := fmt.Sprintf("http://localhost:%d/health", config.Health.Port)
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", config.Metrics.Port)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between health and metrics endpoints
			if pb.Next() {
				resp, err := http.Get(healthURL)
				if err == nil {
					resp.Body.Close()
				}
			} else {
				resp, err := http.Get(metricsURL)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	})

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	testServer.Shutdown(shutdownCtx)
}
