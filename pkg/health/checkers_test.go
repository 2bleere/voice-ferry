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
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestEtcdHealthChecker(t *testing.T) {
	// Test with nil client
	checker := NewEtcdHealthChecker(nil)

	require.NotNil(t, checker)
	assert.Equal(t, "etcd", checker.Name())

	ctx := context.Background()
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestRTPEngineHealthChecker(t *testing.T) {
	// Test with nil client
	checker := NewRTPEngineHealthChecker(nil)

	require.NotNil(t, checker)
	assert.Equal(t, "rtpengine", checker.Name())

	ctx := context.Background()
	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestCustomHealthChecker(t *testing.T) {
	// Test successful check
	successChecker := NewCustomHealthChecker("test-service", func(ctx context.Context) error {
		return nil
	}, 5*time.Second)

	require.NotNil(t, successChecker)
	assert.Equal(t, "test-service", successChecker.Name())

	ctx := context.Background()
	err := successChecker.Check(ctx)

	assert.NoError(t, err)

	// Test failed check
	failChecker := NewCustomHealthChecker("fail-service", func(ctx context.Context) error {
		return assert.AnError
	}, 5*time.Second)

	err = failChecker.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
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
	}, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := checker.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestCustomHealthChecker_Panic_Recovery(t *testing.T) {
	// Test panic recovery
	panicChecker := NewCustomHealthChecker("panic-service", func(ctx context.Context) error {
		panic("test panic")
	}, 5*time.Second)

	ctx := context.Background()

	// This should not panic but should return an error
	err := panicChecker.Check(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
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
	}, 5*time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}
