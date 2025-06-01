package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSIPServerHealthChecker(t *testing.T) {
	// Test with server running
	running := true
	checker := NewSIPServerHealthChecker(&running)

	require.NotNil(t, checker)
	assert.Equal(t, "sip_server", checker.Name())

	ctx := context.Background()
	err := checker.Check(ctx)

	assert.NoError(t, err)

	// Test with server stopped
	running = false
	err = checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestRedisHealthChecker(t *testing.T) {
	// Test with nil client
	checker := NewRedisHealthChecker(nil)

	require.NotNil(t, checker)
	assert.Equal(t, "redis", checker.Name())

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "redis", health.Name)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Equal(t, "Redis client is not configured", health.Message)
}

func TestEtcdHealthChecker(t *testing.T) {
	// Test with nil client
	checker := NewEtcdHealthChecker(nil)

	require.NotNil(t, checker)
	assert.Equal(t, "etcd", checker.Name())

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "etcd", health.Name)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Equal(t, "etcd client is not configured", health.Message)
}

func TestRTPEngineHealthChecker(t *testing.T) {
	// Test with nil client
	checker := NewRTPEngineHealthChecker(nil)

	require.NotNil(t, checker)
	assert.Equal(t, "rtpengine", checker.Name())

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "rtpengine", health.Name)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Equal(t, "RTPEngine client is not configured", health.Message)
}

func TestCustomHealthChecker(t *testing.T) {
	// Test successful check
	successChecker := NewCustomHealthChecker("test-service", func(ctx context.Context) error {
		return nil
	})

	require.NotNil(t, successChecker)
	assert.Equal(t, "test-service", successChecker.Name())

	ctx := context.Background()
	health := successChecker.Check(ctx)

	assert.Equal(t, "test-service", health.Name)
	assert.Equal(t, StatusHealthy, health.Status)
	assert.Equal(t, "Service is healthy", health.Message)

	// Test failed check
	failChecker := NewCustomHealthChecker("fail-service", func(ctx context.Context) error {
		return assert.AnError
	})

	health = failChecker.Check(ctx)

	assert.Equal(t, "fail-service", health.Name)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Contains(t, health.Message, "Service check failed")
}

func TestCustomHealthChecker_ContextTimeout(t *testing.T) {
	// Test with context timeout
	checker := NewCustomHealthChecker("slow-service", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	health := checker.Check(ctx)

	assert.Equal(t, "slow-service", health.Name)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Contains(t, health.Message, "context deadline exceeded")
}

func TestCustomHealthChecker_Panic_Recovery(t *testing.T) {
	// Test panic recovery
	panicChecker := NewCustomHealthChecker("panic-service", func(ctx context.Context) error {
		panic("test panic")
	})

	ctx := context.Background()

	// This should not panic
	assert.NotPanics(t, func() {
		health := panicChecker.Check(ctx)
		assert.Equal(t, "panic-service", health.Name)
		assert.Equal(t, StatusUnhealthy, health.Status)
		assert.Contains(t, health.Message, "Service check panicked")
	})
}

func BenchmarkSIPServerHealthChecker(b *testing.B) {
	running := true
	checker := NewSIPServerHealthChecker(&running)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkCustomHealthChecker(b *testing.B) {
	checker := NewCustomHealthChecker("bench-service", func(ctx context.Context) error {
		return nil
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}
