package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthManager_Creation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	require.NotNil(t, manager)
	assert.Equal(t, "1.0.0", manager.version)
	assert.Equal(t, 0, len(manager.checkers))
}

func TestHealthManager_RegisterChecker(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	checker := &mockHealthChecker{
		name:   "test-checker",
		status: HealthStatusHealthy,
	}

	manager.RegisterChecker(checker)

	assert.Equal(t, 1, len(manager.checkers))
	assert.Equal(t, checker, manager.checkers["test-checker"])
}

func TestHealthManager_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start manager
	manager.Start(ctx)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop manager
	cancel()
	manager.Stop()

	// Verify it's stopped
	time.Sleep(50 * time.Millisecond)
	// No assertion needed, just ensure no panic or deadlock
}

func TestHealthManager_GetStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	// Initially healthy
	status := manager.GetStatus()
	assert.Equal(t, HealthStatusHealthy, status.Status)
	assert.Equal(t, "1.0.0", status.Version)
	assert.True(t, status.Timestamp.After(time.Time{}))
}

func TestHealthManager_WithCheckers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	// Add healthy checker
	healthyChecker := &mockHealthChecker{
		name:   "healthy-service",
		status: HealthStatusHealthy,
	}
	manager.RegisterChecker(healthyChecker)

	// Add unhealthy checker
	unhealthyChecker := &mockHealthChecker{
		name:   "unhealthy-service",
		status: HealthStatusUnhealthy,
	}
	manager.RegisterChecker(unhealthyChecker)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	manager.Start(ctx)

	// Give checkers time to run
	time.Sleep(200 * time.Millisecond)

	status := manager.GetStatus()

	// Overall status should be unhealthy due to one unhealthy service
	assert.Equal(t, HealthStatusUnhealthy, status.Status)
	assert.Equal(t, 2, len(status.Components))

	// Check individual component statuses
	componentMap := make(map[string]*ComponentHealth)
	for name, comp := range status.Components {
		componentMap[name] = comp
	}

	assert.Equal(t, HealthStatusHealthy, componentMap["healthy-service"].Status)
	assert.Equal(t, HealthStatusUnhealthy, componentMap["unhealthy-service"].Status)
}

func TestHealthHandler_HandleHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.HandleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response *SystemHealth
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, HealthStatusHealthy, response.Status)
	assert.Equal(t, "1.0.0", response.Version)
}

func TestHealthHandler_HandleReadiness(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	// Test healthy state
	req, err := http.NewRequest("GET", "/health/ready", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.HandleReadiness(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ready", rr.Body.String())

	// Add unhealthy checker
	unhealthyChecker := &mockHealthChecker{
		name:   "unhealthy-service",
		status: HealthStatusUnhealthy,
	}
	manager.RegisterChecker(unhealthyChecker)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	manager.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Test unhealthy state
	req2, err := http.NewRequest("GET", "/health/ready", nil)
	require.NoError(t, err)

	rr2 := httptest.NewRecorder()
	handler.HandleReadiness(rr2, req2)

	assert.Equal(t, http.StatusServiceUnavailable, rr2.Code)
	assert.Equal(t, "not ready", rr2.Body.String())
}

func TestHealthHandler_HandleLiveness(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	req, err := http.NewRequest("GET", "/health/live", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.HandleLiveness(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "alive", rr.Body.String())
}

func TestHealthHandler_HandleComponentHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	// Add test checker
	checker := &mockHealthChecker{
		name:   "test-service",
		status: HealthStatusHealthy,
	}
	manager.RegisterChecker(checker)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	manager.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	req, err := http.NewRequest("GET", "/health/component?component=test-service", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.HandleComponentHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response ComponentHealth
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-service", response.Name)
	assert.Equal(t, HealthStatusHealthy, response.Status)
}

func TestHealthHandler_HandleComponentHealth_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	req, err := http.NewRequest("GET", "/health/component?component=nonexistent", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.HandleComponentHealth(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "component nonexistent not found\n", rr.Body.String())
}

func TestHealthManager_ConcurrentAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	// Add multiple checkers
	for i := 0; i < 10; i++ {
		checker := &mockHealthChecker{
			name:   fmt.Sprintf("service-%d", i),
			status: HealthStatusHealthy,
		}
		manager.RegisterChecker(checker)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	manager.Start(ctx)

	// Concurrently access status
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				status := manager.GetStatus()
				assert.NotNil(t, status)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Mock health checker for testing
type mockHealthChecker struct {
	name        string
	status      HealthStatus
	checkCount  int64
	shouldPanic bool
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check(ctx context.Context) error {
	atomic.AddInt64(&m.checkCount, 1)

	if m.shouldPanic {
		panic("mock panic")
	}

	if m.status == HealthStatusUnhealthy {
		return fmt.Errorf("mock checker is unhealthy")
	}

	return nil
}

func (m *mockHealthChecker) Timeout() time.Duration {
	return 5 * time.Second
}

func (m *mockHealthChecker) GetCheckCount() int64 {
	return atomic.LoadInt64(&m.checkCount)
}

func TestHealthChecker_Panic_Recovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	// Add panicking checker
	panicChecker := &mockHealthChecker{
		name:        "panic-service",
		status:      HealthStatusHealthy,
		shouldPanic: true,
	}
	manager.RegisterChecker(panicChecker)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// This should not panic the entire application
	manager.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	status := manager.GetStatus()

	// Should still be able to get status
	assert.NotNil(t, status)

	// The panicking component should be marked as unhealthy
	componentMap := make(map[string]*ComponentHealth)
	for name, comp := range status.Components {
		componentMap[name] = comp
	}

	if comp, exists := componentMap["panic-service"]; exists {
		assert.Equal(t, HealthStatusUnhealthy, comp.Status)
		assert.Contains(t, comp.Message, "check failed")
	}
}

func BenchmarkHealthManager_GetStatus(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)

	// Add multiple checkers
	for i := 0; i < 10; i++ {
		checker := &mockHealthChecker{
			name:   fmt.Sprintf("service-%d", i),
			status: HealthStatusHealthy,
		}
		manager.RegisterChecker(checker)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.GetStatus()
		}
	})
}

func BenchmarkHealthHandler_HandleHealth(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewHealthManager("1.0.0", logger)
	handler := NewHealthHandler(manager)

	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.HandleHealth(rr, req)
	}
}
