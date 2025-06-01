package handlers

import (
	"context"
	"log"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/2bleere/voice-ferry/pkg/rtpengine"
	"github.com/2bleere/voice-ferry/pkg/sip"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// StatusHandler implements the StatusService
type StatusHandler struct {
	v1.UnimplementedStatusServiceServer
	sipServer *sip.Server
	rtpEngine *rtpengine.Client
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(sipServer *sip.Server, rtpEngine *rtpengine.Client) *StatusHandler {
	return &StatusHandler{
		sipServer: sipServer,
		rtpEngine: rtpEngine,
	}
}

// GetSystemStatus returns the current system status
func (h *StatusHandler) GetSystemStatus(ctx context.Context, req *emptypb.Empty) (*v1.SystemStatusResponse, error) {
	log.Printf("Getting system status")

	// Get active calls count from SIP server
	activeCalls := 0 // TODO: Get from dialog manager
	totalCalls := 0  // TODO: Get from metrics

	// Check rtpengine instances
	var rtpengineStatus []*v1.RTPEngineStatus
	for _, instance := range h.rtpEngine.GetInstances() {
		healthy := h.rtpEngine.IsInstanceHealthy(ctx, instance.ID)
		status := &v1.RTPEngineStatus{
			InstanceId:     instance.ID,
			Healthy:        healthy,
			ActiveSessions: 0, // TODO: Get from rtpengine query
			Version:        "unknown",
		}
		rtpengineStatus = append(rtpengineStatus, status)
	}

	return &v1.SystemStatusResponse{
		Version:     "1.0.0",
		Uptime:      timestamppb.Now(), // TODO: Track actual uptime
		ActiveCalls: int32(activeCalls),
		TotalCalls:  int32(totalCalls),
		SipStatus: &v1.ComponentStatus{
			Name:      "SIP Server",
			Healthy:   true, // TODO: Check actual SIP server health
			Message:   "Running",
			LastCheck: timestamppb.Now(),
		},
		EtcdStatus: &v1.ComponentStatus{
			Name:      "etcd",
			Healthy:   true, // TODO: Check etcd connectivity
			Message:   "Connected",
			LastCheck: timestamppb.Now(),
		},
		RedisStatus: &v1.ComponentStatus{
			Name:      "Redis",
			Healthy:   true, // TODO: Check Redis connectivity
			Message:   "Connected",
			LastCheck: timestamppb.Now(),
		},
		RtpengineStatus: rtpengineStatus,
	}, nil
}

// GetMetrics returns system metrics
func (h *StatusHandler) GetMetrics(ctx context.Context, req *emptypb.Empty) (*v1.MetricsResponse, error) {
	log.Printf("Getting system metrics")

	metrics := map[string]float64{
		"active_calls":        0.0, // TODO: Get real metrics
		"calls_per_second":    0.0,
		"memory_usage_mb":     0.0,
		"cpu_usage_percent":   0.0,
		"sip_requests_total":  0.0,
		"sip_responses_total": 0.0,
	}

	return &v1.MetricsResponse{
		Metrics: metrics,
	}, nil
}

// HealthCheck performs a health check
func (h *StatusHandler) HealthCheck(ctx context.Context, req *emptypb.Empty) (*v1.HealthCheckResponse, error) {
	// Basic health check - verify all components are responsive
	healthy := true
	message := "All systems operational"

	// TODO: Check SIP server, etcd, Redis, rtpengine connectivity

	return &v1.HealthCheckResponse{
		Healthy: healthy,
		Message: message,
	}, nil
}
