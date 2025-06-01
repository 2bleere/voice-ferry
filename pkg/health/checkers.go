package health

import (
	"context"
	"fmt"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// RedisClient interface for health checking
type RedisClient interface {
	HealthCheck(ctx context.Context) error
}

// RedisHealthChecker checks Redis connectivity
type RedisHealthChecker struct {
	client RedisClient
}

// NewRedisHealthChecker creates a new Redis health checker
func NewRedisHealthChecker(client RedisClient) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

// Check performs the Redis health check
func (r *RedisHealthChecker) Check(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	return r.client.HealthCheck(ctx)
}

// Name returns the checker name
func (r *RedisHealthChecker) Name() string {
	return "redis"
}

// Timeout returns the check timeout
func (r *RedisHealthChecker) Timeout() time.Duration {
	return 5 * time.Second
}

// EtcdClient interface for health checking
type EtcdClient interface {
	HealthCheck(ctx context.Context) error
}

// EtcdHealthChecker checks etcd connectivity
type EtcdHealthChecker struct {
	client EtcdClient
}

// NewEtcdHealthChecker creates a new etcd health checker
func NewEtcdHealthChecker(client EtcdClient) *EtcdHealthChecker {
	return &EtcdHealthChecker{client: client}
}

// Check performs the etcd health check
func (e *EtcdHealthChecker) Check(ctx context.Context) error {
	if e.client == nil {
		return fmt.Errorf("etcd client not initialized")
	}
	return e.client.HealthCheck(ctx)
}

// Name returns the checker name
func (e *EtcdHealthChecker) Name() string {
	return "etcd"
}

// Timeout returns the check timeout
func (e *EtcdHealthChecker) Timeout() time.Duration {
	return 5 * time.Second
}

// RTPEngineClient interface for health checking
type RTPEngineClient interface {
	GetInstances() []config.RTPEngineInstance
	IsInstanceHealthy(ctx context.Context, instanceID string) bool
}

// RTPEngineHealthChecker checks RTPEngine connectivity
type RTPEngineHealthChecker struct {
	client RTPEngineClient
}

// NewRTPEngineHealthChecker creates a new RTPEngine health checker
func NewRTPEngineHealthChecker(client RTPEngineClient) *RTPEngineHealthChecker {
	return &RTPEngineHealthChecker{client: client}
}

// Check performs the RTPEngine health check
func (r *RTPEngineHealthChecker) Check(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("RTPEngine client not initialized")
	}

	instances := r.client.GetInstances()
	if len(instances) == 0 {
		return fmt.Errorf("no RTPEngine instances configured")
	}

	healthyCount := 0
	for _, instance := range instances {
		if instance.Enabled && r.client.IsInstanceHealthy(ctx, instance.ID) {
			healthyCount++
		}
	}

	if healthyCount == 0 {
		return fmt.Errorf("no healthy RTPEngine instances available")
	}

	// Consider degraded if less than half are healthy
	if healthyCount < len(instances)/2 {
		return fmt.Errorf("only %d out of %d RTPEngine instances are healthy", healthyCount, len(instances))
	}

	return nil
}

// Name returns the checker name
func (r *RTPEngineHealthChecker) Name() string {
	return "rtpengine"
}

// Timeout returns the check timeout
func (r *RTPEngineHealthChecker) Timeout() time.Duration {
	return 10 * time.Second
}

// SIPServerHealthChecker checks SIP server status
type SIPServerHealthChecker struct {
	serverRunning *bool
}

// NewSIPServerHealthChecker creates a new SIP server health checker
func NewSIPServerHealthChecker(serverRunning *bool) *SIPServerHealthChecker {
	return &SIPServerHealthChecker{serverRunning: serverRunning}
}

// Check performs the SIP server health check
func (s *SIPServerHealthChecker) Check(ctx context.Context) error {
	if s.serverRunning == nil {
		return fmt.Errorf("SIP server status not tracked")
	}

	if !*s.serverRunning {
		return fmt.Errorf("SIP server is not running")
	}

	return nil
}

// Name returns the checker name
func (s *SIPServerHealthChecker) Name() string {
	return "sip_server"
}

// Timeout returns the check timeout
func (s *SIPServerHealthChecker) Timeout() time.Duration {
	return 2 * time.Second
}

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	maxMemoryMB int64
}

// NewMemoryHealthChecker creates a new memory health checker
func NewMemoryHealthChecker(maxMemoryMB int64) *MemoryHealthChecker {
	return &MemoryHealthChecker{maxMemoryMB: maxMemoryMB}
}

// Check performs the memory health check
func (m *MemoryHealthChecker) Check(ctx context.Context) error {
	// This is a simplified memory check
	// In production, you might want to use runtime.MemStats or similar
	return nil // Always pass for now
}

// Name returns the checker name
func (m *MemoryHealthChecker) Name() string {
	return "memory"
}

// Timeout returns the check timeout
func (m *MemoryHealthChecker) Timeout() time.Duration {
	return 1 * time.Second
}

// DiskSpaceHealthChecker checks disk space usage
type DiskSpaceHealthChecker struct {
	path           string
	thresholdBytes int64
}

// NewDiskSpaceHealthChecker creates a new disk space health checker
func NewDiskSpaceHealthChecker(path string, thresholdBytes int64) *DiskSpaceHealthChecker {
	return &DiskSpaceHealthChecker{
		path:           path,
		thresholdBytes: thresholdBytes,
	}
}

// Check performs the disk space health check
func (d *DiskSpaceHealthChecker) Check(ctx context.Context) error {
	// This is a simplified disk space check
	// In production, you might want to use syscall.Statfs or similar
	return nil // Always pass for now
}

// Name returns the checker name
func (d *DiskSpaceHealthChecker) Name() string {
	return "disk_space"
}

// Timeout returns the check timeout
func (d *DiskSpaceHealthChecker) Timeout() time.Duration {
	return 2 * time.Second
}

// CustomHealthChecker allows custom health checks
type CustomHealthChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
	timeout   time.Duration
}

// NewCustomHealthChecker creates a new custom health checker
func NewCustomHealthChecker(name string, checkFunc func(ctx context.Context) error, timeout time.Duration) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:      name,
		checkFunc: checkFunc,
		timeout:   timeout,
	}
}

// Check performs the custom health check
func (c *CustomHealthChecker) Check(ctx context.Context) (err error) {
	if c.checkFunc == nil {
		return fmt.Errorf("no check function defined")
	}

	// Recover from panics and convert them to errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("health check panicked: %v", r)
		}
	}()

	return c.checkFunc(ctx)
}

// Name returns the checker name
func (c *CustomHealthChecker) Name() string {
	return c.name
}

// Timeout returns the check timeout
func (c *CustomHealthChecker) Timeout() time.Duration {
	if c.timeout == 0 {
		return 5 * time.Second
	}
	return c.timeout
}
