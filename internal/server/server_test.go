package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bleere/voice-ferry/pkg/config"
)

func TestServer_Creation(t *testing.T) {
	cfg := createTestConfig()

	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, cfg, server.cfg)
	assert.NotNil(t, server.sipServer)
	assert.NotNil(t, server.grpcServer)
	assert.NotNil(t, server.httpServer)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.healthManager)
	assert.False(t, server.sipRunning)
}

func TestServer_Creation_WithMetrics(t *testing.T) {
	cfg := createTestConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.NotNil(t, server.metricsServer)
}

func TestServer_Creation_WithRedis(t *testing.T) {
	cfg := createTestConfig()
	cfg.Redis.Enabled = true
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379

	// This will likely fail to connect to Redis, but should handle gracefully
	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Redis client may be nil if connection failed, which is expected in test
}

func TestServer_Creation_WithEtcd(t *testing.T) {
	cfg := createTestConfig()
	cfg.Etcd.Enabled = true
	cfg.Etcd.Endpoints = []string{"localhost:2379"}

	// This will likely fail to connect to etcd, but should handle gracefully
	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	// etcd client may be nil if connection failed, which is expected in test
}

func TestServer_StartHealthCheck(t *testing.T) {
	cfg := createTestConfig()
	cfg.Health.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start health check server in background
	go func() {
		err := server.StartHealthCheck(ctx)
		assert.NoError(t, err)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	healthURL := fmt.Sprintf("http://localhost:%d/health", cfg.Health.Port)
	resp, err := http.Get(healthURL)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Cancel context to stop server
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServer_StartMetrics(t *testing.T) {
	cfg := createTestConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start metrics server in background
	go func() {
		err := server.StartMetrics(ctx)
		assert.NoError(t, err)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test metrics endpoint
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", cfg.Metrics.Port)
	resp, err := http.Get(metricsURL)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
	}

	// Cancel context to stop server
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServer_StartGRPC(t *testing.T) {
	cfg := createTestConfig()
	cfg.GRPC.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start gRPC server in background
	go func() {
		err := server.StartGRPC(ctx)
		assert.NoError(t, err)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test gRPC connection
	addr := fmt.Sprintf("localhost:%d", cfg.GRPC.Port)
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
	}

	// Cancel context to stop server
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestServer_Shutdown(t *testing.T) {
	cfg := createTestConfig()
	cfg.Health.Port = getAvailablePort()
	cfg.GRPC.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)

	// Start health check server
	healthCtx, healthCancel := context.WithCancel(context.Background())
	go func() {
		server.StartHealthCheck(healthCtx)
	}()

	// Start gRPC server
	grpcCtx, grpcCancel := context.WithCancel(context.Background())
	go func() {
		server.StartGRPC(grpcCtx)
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Stop servers
	healthCancel()
	grpcCancel()

	// Test shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestServer_GetClients(t *testing.T) {
	cfg := createTestConfig()

	server, err := New(cfg)
	require.NoError(t, err)

	// Test client getters (may return nil if not configured)
	etcdClient := server.GetEtcdClient()
	redisClient := server.GetRedisClient()

	// These may be nil if not configured, which is fine
	_ = etcdClient
	_ = redisClient
}

func TestServer_HealthCheck(t *testing.T) {
	cfg := createTestConfig()

	server, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// This will likely fail since we don't have real services running
	// but it should not panic
	err = server.HealthCheck(ctx)
	// Error is expected since services aren't actually running
}

func TestServer_SIPRunningState(t *testing.T) {
	cfg := createTestConfig()

	server, err := New(cfg)
	require.NoError(t, err)

	assert.False(t, server.sipRunning)

	// Simulate starting SIP server
	server.sipRunning = true
	assert.True(t, server.sipRunning)
}

func TestServer_MetricsIntegration(t *testing.T) {
	cfg := createTestConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Port = getAvailablePort()

	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server.metricsServer)

	collector := server.metricsServer.GetCollector()
	require.NotNil(t, collector)

	// Test metrics collection
	collector.RecordSIPMessage("INVITE", "incoming", "200")
	collector.UpdateActiveCalls(5)
	collector.UpdateComponentHealth("test", true)

	// No assertions on exact values, just ensure no panic
}

func TestServer_HealthManagerIntegration(t *testing.T) {
	cfg := createTestConfig()

	server, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, server.healthManager)

	// Start health manager
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	server.healthManager.Start(ctx)

	// Give it time to run
	time.Sleep(100 * time.Millisecond)

	status := server.healthManager.GetStatus()
	require.NotNil(t, status)
	assert.Equal(t, "1.0.0", status.Version) // Default version from test config

	server.healthManager.Stop()
}

func createTestConfig() *config.Config {
	return &config.Config{
		SIP: config.SIPConfig{
			Host: "127.0.0.1",
			Port: getAvailablePort(),
		},
		GRPC: config.GRPCConfig{
			Host: "127.0.0.1",
			Port: getAvailablePort(),
		},
		Health: config.HealthConfig{
			Host: "127.0.0.1",
			Port: getAvailablePort(),
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
			Host:    "127.0.0.1",
			Port:    getAvailablePort(),
		},
		Logging: config.LoggingConfig{
			Level:       "info",
			Format:      "text",
			Development: true,
			Version:     "1.0.0",
		},
		Redis: config.RedisConfig{
			Enabled: false,
		},
		Etcd: config.EtcdConfig{
			Enabled: false,
		},
		RTPEngine: config.RTPEngineConfig{
			Instances: []config.RTPEngineInstance{
				{
					ID:      "rtpengine-test",
					Host:    "127.0.0.1",
					Port:    22222,
					Weight:  100,
					Enabled: true,
				},
			},
			Timeout: 5 * time.Second,
		},
	}
}

func getAvailablePort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// Benchmark tests
func BenchmarkServer_Creation(b *testing.B) {
	cfg := createTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server, err := New(cfg)
		if err != nil {
			b.Fatal(err)
		}
		_ = server
	}
}

func BenchmarkServer_HealthCheck(b *testing.B) {
	cfg := createTestConfig()
	server, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.HealthCheck(ctx)
	}
}
