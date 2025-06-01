package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"log/slog"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name       string                 `json:"name"`
	Status     HealthStatus           `json:"status"`
	Message    string                 `json:"message,omitempty"`
	LastCheck  time.Time              `json:"last_check"`
	CheckCount int64                  `json:"check_count"`
	ErrorCount int64                  `json:"error_count"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Duration   time.Duration          `json:"duration"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     HealthStatus                `json:"status"`
	Version    string                      `json:"version"`
	Timestamp  time.Time                   `json:"timestamp"`
	Uptime     time.Duration               `json:"uptime"`
	Components map[string]*ComponentHealth `json:"components"`
	Summary    map[string]interface{}      `json:"summary"`
}

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
	Timeout() time.Duration
}

// HealthManager manages health checks for all system components
type HealthManager struct {
	mu          sync.RWMutex
	checkers    map[string]HealthChecker
	components  map[string]*ComponentHealth
	startTime   time.Time
	version     string
	logger      *slog.Logger
	checkPeriod time.Duration
	stopCh      chan struct{}
	stopped     bool
}

// NewHealthManager creates a new health manager
func NewHealthManager(version string, logger *slog.Logger) *HealthManager {
	return &HealthManager{
		checkers:    make(map[string]HealthChecker),
		components:  make(map[string]*ComponentHealth),
		startTime:   time.Now(),
		version:     version,
		logger:      logger,
		checkPeriod: 30 * time.Second,
		stopCh:      make(chan struct{}),
	}
}

// RegisterChecker registers a health checker
func (hm *HealthManager) RegisterChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	name := checker.Name()
	hm.checkers[name] = checker
	hm.components[name] = &ComponentHealth{
		Name:      name,
		Status:    HealthStatusUnknown,
		LastCheck: time.Time{},
		Details:   make(map[string]interface{}),
	}

	hm.logger.Info("Registered health checker", "component", name)
}

// UnregisterChecker unregisters a health checker
func (hm *HealthManager) UnregisterChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.checkers, name)
	delete(hm.components, name)

	hm.logger.Info("Unregistered health checker", "component", name)
}

// Start starts the health monitoring
func (hm *HealthManager) Start(ctx context.Context) {
	hm.logger.Info("Starting health manager")

	// Perform initial health checks
	hm.performHealthChecks(ctx)

	// Start periodic health checks
	ticker := time.NewTicker(hm.checkPeriod)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hm.performHealthChecks(ctx)
			case <-hm.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the health monitoring
func (hm *HealthManager) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.stopped {
		close(hm.stopCh)
		hm.stopped = true
		hm.logger.Info("Stopped health manager")
	}
}

// performHealthChecks performs health checks for all registered checkers
func (hm *HealthManager) performHealthChecks(ctx context.Context) {
	hm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()

	var wg sync.WaitGroup
	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker HealthChecker) {
			defer wg.Done()
			hm.performSingleHealthCheck(ctx, name, checker)
		}(name, checker)
	}

	wg.Wait()
}

// performSingleHealthCheck performs a health check for a single component
func (hm *HealthManager) performSingleHealthCheck(ctx context.Context, name string, checker HealthChecker) {
	timeout := checker.Timeout()
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := checker.Check(checkCtx)
	duration := time.Since(start)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	component := hm.components[name]
	if component == nil {
		component = &ComponentHealth{
			Name:    name,
			Details: make(map[string]interface{}),
		}
		hm.components[name] = component
	}

	component.LastCheck = time.Now()
	component.CheckCount++
	component.Duration = duration

	if err != nil {
		component.Status = HealthStatusUnhealthy
		component.Message = err.Error()
		component.ErrorCount++
		hm.logger.Error("Health check failed", "component", name, "error", err, "duration", duration)
	} else {
		component.Status = HealthStatusHealthy
		component.Message = ""
		hm.logger.Debug("Health check passed", "component", name, "duration", duration)
	}

	// Add performance details
	component.Details["response_time_ms"] = duration.Milliseconds()
	component.Details["success_rate"] = float64(component.CheckCount-component.ErrorCount) / float64(component.CheckCount)
}

// GetHealth returns the current system health
func (hm *HealthManager) GetHealth() *SystemHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	components := make(map[string]*ComponentHealth)
	for name, component := range hm.components {
		// Create a copy to avoid data races
		components[name] = &ComponentHealth{
			Name:       component.Name,
			Status:     component.Status,
			Message:    component.Message,
			LastCheck:  component.LastCheck,
			CheckCount: component.CheckCount,
			ErrorCount: component.ErrorCount,
			Duration:   component.Duration,
			Details:    make(map[string]interface{}),
		}
		// Copy details
		for k, v := range component.Details {
			components[name].Details[k] = v
		}
	}

	// Determine overall status
	overallStatus := hm.calculateOverallStatus(components)

	// Calculate summary statistics
	summary := hm.calculateSummary(components)

	return &SystemHealth{
		Status:     overallStatus,
		Version:    hm.version,
		Timestamp:  time.Now(),
		Uptime:     time.Since(hm.startTime),
		Components: components,
		Summary:    summary,
	}
}

// calculateOverallStatus calculates the overall system health status
func (hm *HealthManager) calculateOverallStatus(components map[string]*ComponentHealth) HealthStatus {
	if len(components) == 0 {
		return HealthStatusUnknown
	}

	healthyCount := 0
	unhealthyCount := 0
	degradedCount := 0

	for _, component := range components {
		switch component.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		}
	}

	total := len(components)

	// If any component is unhealthy, system is unhealthy
	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}

	// If any component is degraded, system is degraded
	if degradedCount > 0 {
		return HealthStatusDegraded
	}

	// If all components are healthy, system is healthy
	if healthyCount == total {
		return HealthStatusHealthy
	}

	return HealthStatusUnknown
}

// calculateSummary calculates summary statistics
func (hm *HealthManager) calculateSummary(components map[string]*ComponentHealth) map[string]interface{} {
	summary := map[string]interface{}{
		"total_components":     len(components),
		"healthy_components":   0,
		"unhealthy_components": 0,
		"degraded_components":  0,
		"unknown_components":   0,
		"avg_response_time_ms": 0.0,
		"total_checks":         int64(0),
		"total_errors":         int64(0),
	}

	if len(components) == 0 {
		return summary
	}

	var totalResponseTime time.Duration
	var totalChecks, totalErrors int64

	for _, component := range components {
		switch component.Status {
		case HealthStatusHealthy:
			summary["healthy_components"] = summary["healthy_components"].(int) + 1
		case HealthStatusUnhealthy:
			summary["unhealthy_components"] = summary["unhealthy_components"].(int) + 1
		case HealthStatusDegraded:
			summary["degraded_components"] = summary["degraded_components"].(int) + 1
		case HealthStatusUnknown:
			summary["unknown_components"] = summary["unknown_components"].(int) + 1
		}

		totalResponseTime += component.Duration
		totalChecks += component.CheckCount
		totalErrors += component.ErrorCount
	}

	summary["avg_response_time_ms"] = float64(totalResponseTime.Milliseconds()) / float64(len(components))
	summary["total_checks"] = totalChecks
	summary["total_errors"] = totalErrors

	if totalChecks > 0 {
		summary["overall_success_rate"] = float64(totalChecks-totalErrors) / float64(totalChecks)
	} else {
		summary["overall_success_rate"] = 0.0
	}

	return summary
}

// GetComponentHealth returns the health of a specific component
func (hm *HealthManager) GetComponentHealth(name string) (*ComponentHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	component, exists := hm.components[name]
	if !exists {
		return nil, false
	}

	// Return a copy
	return &ComponentHealth{
		Name:       component.Name,
		Status:     component.Status,
		Message:    component.Message,
		LastCheck:  component.LastCheck,
		CheckCount: component.CheckCount,
		ErrorCount: component.ErrorCount,
		Duration:   component.Duration,
		Details:    component.Details,
	}, true
}

// SetCheckPeriod sets the health check period
func (hm *HealthManager) SetCheckPeriod(period time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkPeriod = period
}

// HealthHandler provides HTTP handlers for health endpoints
type HealthHandler struct {
	manager *HealthManager
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(manager *HealthManager) *HealthHandler {
	return &HealthHandler{manager: manager}
}

// HandleHealth handles the main health endpoint
func (hh *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := hh.manager.GetHealth()

	w.Header().Set("Content-Type", "application/json")

	// Set HTTP status based on health
	switch health.Status {
	case HealthStatusHealthy:
		w.WriteHeader(http.StatusOK)
	case HealthStatusDegraded:
		w.WriteHeader(http.StatusOK) // 200 for degraded but functioning
	case HealthStatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// HandleReadiness handles the readiness probe endpoint
func (hh *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	health := hh.manager.GetHealth()

	if health.Status == HealthStatusHealthy || health.Status == HealthStatusDegraded {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not Ready"))
	}
}

// HandleLiveness handles the liveness probe endpoint
func (hh *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - just return OK if the service is running
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alive"))
}

// HandleComponentHealth handles individual component health endpoints
func (hh *HealthHandler) HandleComponentHealth(w http.ResponseWriter, r *http.Request) {
	componentName := r.URL.Query().Get("component")
	if componentName == "" {
		http.Error(w, "component parameter required", http.StatusBadRequest)
		return
	}

	component, exists := hh.manager.GetComponentHealth(componentName)
	if !exists {
		http.Error(w, fmt.Sprintf("component %s not found", componentName), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch component.Status {
	case HealthStatusHealthy:
		w.WriteHeader(http.StatusOK)
	case HealthStatusDegraded:
		w.WriteHeader(http.StatusOK)
	case HealthStatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(component)
}
