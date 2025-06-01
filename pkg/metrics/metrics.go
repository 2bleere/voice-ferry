package metrics

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Ensure metrics are only registered once
	once           sync.Once
	globalRegistry *prometheus.Registry

	// SIP metrics
	SIPRequestsTotal   *prometheus.CounterVec
	SIPRequestDuration *prometheus.HistogramVec

	// Call metrics
	ActiveCallsGauge *prometheus.GaugeVec
	CallDuration     *prometheus.HistogramVec
	CallsTotal       *prometheus.CounterVec

	// Routing metrics
	RoutingDecisionsTotal *prometheus.CounterVec
	RoutingRulesGauge     prometheus.Gauge
	RoutingLatency        *prometheus.HistogramVec

	// Media metrics
	MediaSessionsGauge  *prometheus.GaugeVec
	RTPEngineOperations *prometheus.CounterVec
	RTPEngineLatency    *prometheus.HistogramVec

	// Storage metrics
	RedisOperations *prometheus.CounterVec
	RedisLatency    *prometheus.HistogramVec
	EtcdOperations  *prometheus.CounterVec
	EtcdLatency     *prometheus.HistogramVec

	// System metrics
	SystemInfo      *prometheus.GaugeVec
	ComponentHealth *prometheus.GaugeVec
)

// initMetrics initializes all metrics (called only once)
func initMetrics() {
	once.Do(func() {
		// Create a new registry to avoid conflicts with default registry
		globalRegistry = prometheus.NewRegistry()

		// SIP metrics
		SIPRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sip_requests_total",
				Help: "Total number of SIP requests processed",
			},
			[]string{"method", "status", "source"},
		)

		SIPRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "sip_request_duration_seconds",
				Help:    "Duration of SIP request processing",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		)

		// Call metrics
		ActiveCallsGauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_calls_total",
				Help: "Number of currently active calls",
			},
			[]string{"state"},
		)

		CallDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "call_duration_seconds",
				Help:    "Duration of completed calls",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200, 3600},
			},
			[]string{"termination_reason"},
		)

		CallsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "calls_total",
				Help: "Total number of calls processed",
			},
			[]string{"result", "source", "destination"},
		)

		// Routing metrics
		RoutingDecisionsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "routing_decisions_total",
				Help: "Total number of routing decisions made",
			},
			[]string{"action", "rule_id"},
		)

		RoutingRulesGauge = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "routing_rules_total",
				Help: "Number of active routing rules",
			},
		)

		RoutingLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "routing_latency_seconds",
				Help:    "Time taken to make routing decisions",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"rule_type"},
		)

		// Media metrics
		MediaSessionsGauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "media_sessions_total",
				Help: "Number of active media sessions",
			},
			[]string{"rtpengine_instance"},
		)

		RTPEngineOperations = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rtpengine_operations_total",
				Help: "Total RTPEngine operations",
			},
			[]string{"operation", "result", "instance"},
		)

		RTPEngineLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rtpengine_operation_duration_seconds",
				Help:    "Duration of RTPEngine operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"operation", "instance"},
		)

		// Storage metrics
		RedisOperations = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_operations_total",
				Help: "Total Redis operations",
			},
			[]string{"operation", "result"},
		)

		RedisLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_operation_duration_seconds",
				Help:    "Duration of Redis operations",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
			},
			[]string{"operation"},
		)

		EtcdOperations = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etcd_operations_total",
				Help: "Total etcd operations",
			},
			[]string{"operation", "result"},
		)

		EtcdLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "etcd_operation_duration_seconds",
				Help:    "Duration of etcd operations",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"operation"},
		)

		// System metrics
		SystemInfo = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sip_b2bua_info",
				Help: "SIP B2BUA system information",
			},
			[]string{"version", "build_time", "go_version"},
		)

		ComponentHealth = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "component_health",
				Help: "Health status of system components (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		)

		// Register all metrics with our custom registry
		globalRegistry.MustRegister(
			SIPRequestsTotal,
			SIPRequestDuration,
			ActiveCallsGauge,
			CallDuration,
			CallsTotal,
			RoutingDecisionsTotal,
			RoutingRulesGauge,
			RoutingLatency,
			MediaSessionsGauge,
			RTPEngineOperations,
			RTPEngineLatency,
			RedisOperations,
			RedisLatency,
			EtcdOperations,
			EtcdLatency,
			SystemInfo,
			ComponentHealth,
		)

		// Add Go runtime metrics but with our custom registry
		globalRegistry.MustRegister(prometheus.NewGoCollector())

		// Add process metrics but with our custom registry
		globalRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

		// Set initial system info
		SystemInfo.WithLabelValues("unknown", "unknown", runtime.Version()).Set(1)
	})
}

// MetricsCollector manages Prometheus metrics collection
type MetricsCollector struct {
	registry *prometheus.Registry
	handler  http.Handler
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	// Initialize metrics first
	initMetrics()

	return &MetricsCollector{
		registry: globalRegistry,
		handler:  promhttp.HandlerFor(globalRegistry, promhttp.HandlerOpts{}),
	}
}

// GetHandler returns the HTTP handler for metrics endpoint
func (mc *MetricsCollector) GetHandler() http.Handler {
	return mc.handler
}

// GetRegistry returns the metrics registry
func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.registry
}

// MetricsServer provides an HTTP server for metrics
type MetricsServer struct {
	server    *http.Server
	collector *MetricsCollector
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(addr string, collector *MetricsCollector) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", collector.GetHandler())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &MetricsServer{
		server:    server,
		collector: collector,
	}
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	return ms.server.ListenAndServe()
}

// Shutdown gracefully shuts down the metrics server
func (ms *MetricsServer) Shutdown(ctx context.Context) error {
	return ms.server.Shutdown(ctx)
}

// GetCollector returns the metrics collector
func (ms *MetricsServer) GetCollector() *MetricsCollector {
	return ms.collector
}

// SetSystemInfo sets system information metrics
func (mc *MetricsCollector) SetSystemInfo(version, buildTime, goVersion string) {
	if SystemInfo != nil {
		SystemInfo.WithLabelValues(version, buildTime, goVersion).Set(1)
	}
}

// UpdateComponentHealth updates component health status
func (mc *MetricsCollector) UpdateComponentHealth(component string, healthy bool) {
	if ComponentHealth != nil {
		value := float64(0)
		if healthy {
			value = 1
		}
		ComponentHealth.WithLabelValues(component).Set(value)
	}
}

// RecordSIPRequest records a SIP request metric
func (mc *MetricsCollector) RecordSIPRequest(method, status, source string, duration time.Duration) {
	if SIPRequestsTotal != nil {
		SIPRequestsTotal.WithLabelValues(method, status, source).Inc()
	}
	if SIPRequestDuration != nil {
		SIPRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
	}
}

// RecordCallMetrics records call-related metrics
func (mc *MetricsCollector) RecordCallMetrics(result, source, destination, terminationReason string, duration time.Duration) {
	if CallsTotal != nil {
		CallsTotal.WithLabelValues(result, source, destination).Inc()
	}
	if duration > 0 && CallDuration != nil {
		CallDuration.WithLabelValues(terminationReason).Observe(duration.Seconds())
	}
}

// UpdateActiveCallsGauge updates the active calls gauge
func (mc *MetricsCollector) UpdateActiveCallsGauge(state string, count float64) {
	if ActiveCallsGauge != nil {
		ActiveCallsGauge.WithLabelValues(state).Set(count)
	}
}

// RecordRoutingDecision records a routing decision
func (mc *MetricsCollector) RecordRoutingDecision(action, ruleID string, latency time.Duration, ruleType string) {
	if RoutingDecisionsTotal != nil {
		RoutingDecisionsTotal.WithLabelValues(action, ruleID).Inc()
	}
	if RoutingLatency != nil {
		RoutingLatency.WithLabelValues(ruleType).Observe(latency.Seconds())
	}
}

// UpdateRoutingRulesCount updates the routing rules count
func (mc *MetricsCollector) UpdateRoutingRulesCount(count float64) {
	if RoutingRulesGauge != nil {
		RoutingRulesGauge.Set(count)
	}
}

// RecordRTPEngineOperation records an RTPEngine operation
func (mc *MetricsCollector) RecordRTPEngineOperation(operation, result, instance string, duration time.Duration) {
	if RTPEngineOperations != nil {
		RTPEngineOperations.WithLabelValues(operation, result, instance).Inc()
	}
	if RTPEngineLatency != nil {
		RTPEngineLatency.WithLabelValues(operation, instance).Observe(duration.Seconds())
	}
}

// UpdateMediaSessionsGauge updates the media sessions gauge
func (mc *MetricsCollector) UpdateMediaSessionsGauge(instance string, count float64) {
	if MediaSessionsGauge != nil {
		MediaSessionsGauge.WithLabelValues(instance).Set(count)
	}
}

// RecordRedisOperation records a Redis operation
func (mc *MetricsCollector) RecordRedisOperation(operation, result string, duration time.Duration) {
	if RedisOperations != nil {
		RedisOperations.WithLabelValues(operation, result).Inc()
	}
	if RedisLatency != nil {
		RedisLatency.WithLabelValues(operation).Observe(duration.Seconds())
	}
}

// RecordEtcdOperation records an etcd operation
func (mc *MetricsCollector) RecordEtcdOperation(operation, result string, duration time.Duration) {
	if EtcdOperations != nil {
		EtcdOperations.WithLabelValues(operation, result).Inc()
	}
	if EtcdLatency != nil {
		EtcdLatency.WithLabelValues(operation).Observe(duration.Seconds())
	}
}

// InitializeMetrics is a public function to initialize metrics
// Call this once at application startup
func InitializeMetrics() {
	initMetrics()
}

// GetMetricsHandler returns a simple HTTP handler for metrics
func GetMetricsHandler() http.Handler {
	initMetrics()
	return promhttp.HandlerFor(globalRegistry, promhttp.HandlerOpts{})
}
