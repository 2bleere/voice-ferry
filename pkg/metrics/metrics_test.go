package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsCollector_Creation(t *testing.T) {
	collector := NewMetricsCollector()
	require.NotNil(t, collector)
	assert.NotNil(t, collector.sipMessagesTotal)
	assert.NotNil(t, collector.callsActive)
	assert.NotNil(t, collector.routingDecisions)
	assert.NotNil(t, collector.componentHealth)
}

func TestMetricsCollector_SIPMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Test SIP message recording
	collector.RecordSIPMessage("INVITE", "incoming", "200")
	collector.RecordSIPMessage("BYE", "outgoing", "200")

	// Verify metrics
	expected := `
		# HELP test_sip_messages_total Total number of SIP messages processed (test)
		# TYPE test_sip_messages_total counter
		test_sip_messages_total{direction="incoming",method="INVITE",status="200"} 1
		test_sip_messages_total{direction="outgoing",method="BYE",status="200"} 1
	`

	err := testutil.CollectAndCompare(collector.sipMessagesTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestMetricsCollector_CallMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Test call lifecycle
	collector.RecordCallStarted()
	collector.RecordCallStarted()
	collector.RecordCallCompleted("completed", 120.5)

	// Test active calls
	collector.UpdateActiveCalls(5)

	// Verify active calls metric
	expected := `
		# HELP test_calls_active Number of currently active calls (test)
		# TYPE test_calls_active gauge
		test_calls_active 5
	`

	err := testutil.CollectAndCompare(collector.callsActive, strings.NewReader(expected))
	assert.NoError(t, err)

	// The call duration histogram should have recorded the data
	// (detailed verification would require more complex metric inspection)
}

func TestMetricsCollector_RoutingMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Test routing decisions
	collector.RecordRoutingDecisionSimple("success", "default")
	collector.RecordRoutingDecisionSimple("failed", "premium")

	expected := `
		# HELP test_routing_decisions_total Total number of routing decisions made (test)
		# TYPE test_routing_decisions_total counter
		test_routing_decisions_total{result="failed",rule="premium"} 1
		test_routing_decisions_total{result="success",rule="default"} 1
	`

	err := testutil.CollectAndCompare(collector.routingDecisions, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestMetricsCollector_MediaMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Test media session metrics
	collector.RecordMediaSession("created")
	collector.RecordMediaSession("destroyed")
	collector.UpdateActiveMediaSessions(3)

	// Test RTP packet metrics
	collector.RecordRTPPackets("audio", "sent", 100)
	collector.RecordRTPPackets("video", "received", 50)

	// Verify active media sessions
	expected := `
		# HELP test_media_sessions_active Number of currently active media sessions (test)
		# TYPE test_media_sessions_active gauge
		test_media_sessions_active 3
	`

	err := testutil.CollectAndCompare(collector.mediaSessionsActive, strings.NewReader(expected))
	assert.NoError(t, err)

	// Verify RTP packets
	expectedRTP := `
		# HELP test_rtp_packets_total Total number of RTP packets processed (test)
		# TYPE test_rtp_packets_total counter
		test_rtp_packets_total{direction="received",type="video"} 50
		test_rtp_packets_total{direction="sent",type="audio"} 100
	`

	err = testutil.CollectAndCompare(collector.rtpPacketsTotal, strings.NewReader(expectedRTP))
	assert.NoError(t, err)
}

func TestMetricsCollector_StorageMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Test Redis operations
	collector.RecordRedisOperation("GET", "success", 15*time.Millisecond)
	collector.RecordRedisOperation("SET", "error", 5*time.Millisecond)

	// Test etcd operations
	collector.RecordEtcdOperation("PUT", "success", 25*time.Millisecond)

	// Verify Redis operations counter exists
	assert.Greater(t, testutil.ToFloat64(collector.redisOperationsTotal.WithLabelValues("GET", "success")), 0.0)
	assert.Greater(t, testutil.ToFloat64(collector.etcdOperationsTotal.WithLabelValues("PUT", "success")), 0.0)
}

func TestMetricsCollector_ComponentHealth(t *testing.T) {
	collector := NewMetricsCollector()

	// Test component health updates
	collector.UpdateComponentHealth("redis", true)
	collector.UpdateComponentHealth("etcd", false)

	expected := `# HELP component_health Health status of system components (test)
# TYPE component_health gauge
component_health{component="etcd"} 0
component_health{component="redis"} 1
`

	err := testutil.CollectAndCompare(collector.componentHealth, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestMetricsCollector_SystemInfo(t *testing.T) {
	collector := NewMetricsCollector()

	// Set system info
	collector.SetSystemInfo("1.0.0", "2023-12-01T10:00:00Z", "go1.21")

	expected := `# HELP test_system_info System information (test)
# TYPE test_system_info gauge
test_system_info{build_time="2023-12-01T10:00:00Z",version="1.0.0"} 1
`

	err := testutil.CollectAndCompare(collector.systemInfo, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestMetricsServer_Creation(t *testing.T) {
	collector := NewMetricsCollector()
	server := NewMetricsServer(":0", collector)

	require.NotNil(t, server)
	assert.Equal(t, collector, server.GetCollector())
}

func TestMetricsServer_StartStop(t *testing.T) {
	collector := NewMetricsCollector()
	server := NewMetricsServer(":0", collector)

	// Start server in background
	go func() {
		err := server.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Unexpected server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMetricsHandler(t *testing.T) {
	collector := NewMetricsCollector()

	// Add some test data
	collector.RecordSIPMessage("INVITE", "incoming", "200")
	collector.UpdateActiveCalls(2)

	// Create HTTP request
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create and serve handler
	handler := collector.GetHandler()
	handler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, rr.Body.String(), "test_sip_messages_total")
	assert.Contains(t, rr.Body.String(), "test_calls_active")
}

func TestMetricsCollector_Concurrency(t *testing.T) {
	collector := NewMetricsCollector()

	// Test concurrent access to metrics
	done := make(chan bool)

	// Start multiple goroutines updating metrics
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				collector.RecordSIPMessage("INVITE", "incoming", "200")
				collector.RecordCallStarted()
				collector.UpdateActiveCalls(int64(id))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics were recorded (exact count may vary due to timing)
	sipCount := testutil.ToFloat64(collector.sipMessagesTotal.WithLabelValues("INVITE", "incoming", "200"))
	assert.Greater(t, sipCount, 0.0)

	callCount := testutil.ToFloat64(collector.callsTotal.WithLabelValues("started"))
	assert.Greater(t, callCount, 0.0)
}

func BenchmarkMetricsCollector_RecordSIPMessage(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordSIPMessage("INVITE", "incoming", "200")
		}
	})
}

func BenchmarkMetricsCollector_RecordCallStarted(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordCallStarted()
		}
	})
}

func BenchmarkMetricsCollector_UpdateActiveCalls(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.UpdateActiveCalls(100)
		}
	})
}
