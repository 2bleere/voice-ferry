package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/2bleere/voice-ferry/internal/handlers"
	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/etcd"
	"github.com/2bleere/voice-ferry/pkg/health"
	"github.com/2bleere/voice-ferry/pkg/logging"
	"github.com/2bleere/voice-ferry/pkg/metrics"
	"github.com/2bleere/voice-ferry/pkg/redis"
	"github.com/2bleere/voice-ferry/pkg/rtpengine"
	"github.com/2bleere/voice-ferry/pkg/sip"
	"github.com/2bleere/voice-ferry/pkg/webrtc"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// Server represents the main B2BUA server
type Server struct {
	cfg           *config.Config
	sipServer     *sip.Server
	grpcServer    *grpc.Server
	webrtcGateway *webrtc.Gateway
	rtpEngine     *rtpengine.Client
	httpServer    *http.Server // for health checks
	webrtcServer  *http.Server // for WebRTC gateway
	metricsServer *metrics.MetricsServer
	etcdClient    *etcd.Client
	redisClient   *redis.Client
	healthManager *health.HealthManager
	logger        *logging.Logger
	sipRunning    bool
}

// New creates a new B2BUA server
func New(cfg *config.Config) (*Server, error) {
	// Create logger
	logger, err := logging.NewLogger(cfg.Logging, "b2bua-server")
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	logger.Info("Starting B2BUA server initialization")

	// Create metrics collector
	metricsCollector := metrics.NewMetricsCollector()
	metricsCollector.SetSystemInfo(cfg.Logging.Version, time.Now().Format(time.RFC3339), "go1.21")

	// Create metrics server
	var metricsServer *metrics.MetricsServer
	if cfg.Metrics.Enabled {
		metricsAddr := fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port)
		metricsServer = metrics.NewMetricsServer(metricsAddr, metricsCollector)
		logger.Info("Metrics server configured", "addr", metricsAddr)
	}

	// Create health manager
	healthManager := health.NewHealthManager(cfg.Logging.Version, logger.Logger)

	// Create SIP server
	sipServer, err := sip.NewServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SIP server: %w", err)
	}

	// Create rtpengine client
	rtpEngine, err := rtpengine.NewClient(cfg.RTPEngine)
	if err != nil {
		logger.Warn("Failed to create rtpengine client", "error", err)
		rtpEngine = nil
	}

	// We'll create and attach the session manager later after Redis client is initialized

	server := &Server{
		cfg:           cfg,
		logger:        logger,
		sipServer:     sipServer,
		rtpEngine:     rtpEngine,
		metricsServer: metricsServer,
		healthManager: healthManager,
		sipRunning:    false,
	}

	// Create gRPC server (services will be registered later after Redis is initialized)
	server.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			handlers.LoggingInterceptor,
			handlers.AuthInterceptor(cfg),
		),
	)

	// Create health check HTTP server with integrated handlers
	healthHandler := health.NewHealthHandler(healthManager)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler.HandleHealth)
	mux.HandleFunc("/health/ready", healthHandler.HandleReadiness)
	mux.HandleFunc("/health/live", healthHandler.HandleLiveness)
	mux.HandleFunc("/health/component", healthHandler.HandleComponentHealth)

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Health.Host, cfg.Health.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create WebRTC HTTP server if WebRTC gateway is enabled
	if server.webrtcGateway != nil {
		webrtcMux := http.NewServeMux()
		webrtcMux.HandleFunc(cfg.WebRTC.WSPath, server.webrtcGateway.HandleWebSocket)

		// Add CORS headers for WebRTC
		corsHandler := func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				origin := r.Header.Get("Origin")
				if cfg.WebRTC.Auth.Enabled {
					// Check allowed origins
					allowed := false
					for _, allowedOrigin := range cfg.WebRTC.Auth.Origins {
						if allowedOrigin == "*" || allowedOrigin == origin {
							allowed = true
							break
						}
					}
					if !allowed {
						http.Error(w, "Origin not allowed", http.StatusForbidden)
						return
					}
				}

				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}

				h.ServeHTTP(w, r)
			})
		}

		server.webrtcServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.WebRTC.Host, cfg.WebRTC.Port),
			Handler:      corsHandler(webrtcMux),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}
	}

	// Create etcd client if enabled
	if cfg.Etcd.Enabled {
		server.etcdClient, err = etcd.NewClient(&cfg.Etcd)
		if err != nil {
			logger.Warn("Failed to create etcd client", "error", err)
		} else {
			logger.Info("Connected to etcd cluster")
			// Register etcd health checker
			healthManager.RegisterChecker(health.NewEtcdHealthChecker(server.etcdClient))
		}
	}

	// Create Redis client if enabled
	if cfg.Redis.Enabled {
		server.redisClient, err = redis.NewClient(&cfg.Redis)
		if err != nil {
			logger.Warn("Failed to create Redis client", "error", err)
		} else {
			logger.Info("Connected to Redis server")
			// Register Redis health checker
			healthManager.RegisterChecker(health.NewRedisHealthChecker(server.redisClient))

			// Create and attach session manager to SIP server
			sessionMgr := sip.NewSessionManager(
				sipServer.GetDialogManager(),
				server.redisClient,
				rtpEngine,
				logger.With("component", "session-manager"),
			)
			sipServer.SetSessionManager(sessionMgr)
			logger.Info("Session manager initialized with Redis",
				"enable_session_limits", cfg.Redis.EnableSessionLimits,
				"max_sessions_per_user", cfg.Redis.MaxSessionsPerUser,
				"session_limit_action", cfg.Redis.SessionLimitAction)

			// Now register gRPC services with Redis support
			callHandler := handlers.NewCallHandler(sipServer, rtpEngine, server.redisClient, sessionMgr)
			routingHandler := handlers.NewRoutingHandler(cfg, sipServer.GetRoutingEngine())
			headerHandler := handlers.NewHeaderHandler(sipServer)
			configHandler := handlers.NewConfigHandler(cfg)
			statusHandler := handlers.NewStatusHandler(sipServer, rtpEngine)

			v1.RegisterB2BUACallServiceServer(server.grpcServer, callHandler)
			v1.RegisterRoutingRuleServiceServer(server.grpcServer, routingHandler)
			v1.RegisterSIPHeaderServiceServer(server.grpcServer, headerHandler)
			v1.RegisterConfigurationServiceServer(server.grpcServer, configHandler)
			v1.RegisterStatusServiceServer(server.grpcServer, statusHandler)

			// Enable reflection for debugging
			reflection.Register(server.grpcServer)
		}
	} else {
		// Register gRPC services without Redis support
		callHandler := handlers.NewCallHandler(sipServer, rtpEngine, nil, nil)
		routingHandler := handlers.NewRoutingHandler(cfg, sipServer.GetRoutingEngine())
		headerHandler := handlers.NewHeaderHandler(sipServer)
		configHandler := handlers.NewConfigHandler(cfg)
		statusHandler := handlers.NewStatusHandler(sipServer, rtpEngine)

		v1.RegisterB2BUACallServiceServer(server.grpcServer, callHandler)
		v1.RegisterRoutingRuleServiceServer(server.grpcServer, routingHandler)
		v1.RegisterSIPHeaderServiceServer(server.grpcServer, headerHandler)
		v1.RegisterConfigurationServiceServer(server.grpcServer, configHandler)
		v1.RegisterStatusServiceServer(server.grpcServer, statusHandler)

		// Enable reflection for debugging
		reflection.Register(server.grpcServer)
	}

	// Create WebRTC gateway if enabled (after Redis client is created)
	if cfg.WebRTC.Enabled {
		// Create session manager for WebRTC gateway
		sessionMgr := sip.NewSessionManager(
			sipServer.GetDialogManager(),
			server.redisClient,
			rtpEngine,
			logger.With("component", "session-manager"),
		)

		server.webrtcGateway, err = webrtc.NewGateway(
			&cfg.WebRTC,
			sipServer,
			sessionMgr,
			logger.With("component", "webrtc-gateway"),
		)
		if err != nil {
			logger.Warn("Failed to create WebRTC gateway", "error", err)
			server.webrtcGateway = nil
		} else {
			logger.Info("WebRTC gateway initialized", "host", cfg.WebRTC.Host, "port", cfg.WebRTC.Port)
		}
	}

	// Register component health checkers
	healthManager.RegisterChecker(health.NewSIPServerHealthChecker(&server.sipRunning))
	if rtpEngine != nil {
		healthManager.RegisterChecker(health.NewRTPEngineHealthChecker(rtpEngine))
	}

	// Update metrics collector with component health
	if metricsCollector != nil {
		metricsCollector.UpdateComponentHealth("sip_server", true)
		metricsCollector.UpdateComponentHealth("etcd", server.etcdClient != nil)
		metricsCollector.UpdateComponentHealth("redis", server.redisClient != nil)
		metricsCollector.UpdateComponentHealth("rtpengine", rtpEngine != nil)
	}

	logger.Info("B2BUA server initialization completed")
	return server, nil
}

// StartSIP starts the SIP server
func (s *Server) StartSIP(ctx context.Context) error {
	s.logger.Info("Starting SIP server", "host", s.cfg.SIP.Host, "port", s.cfg.SIP.Port)
	s.sipRunning = true

	// Update metrics
	if s.metricsServer != nil {
		s.metricsServer.GetCollector().UpdateComponentHealth("sip_server", true)
	}

	return s.sipServer.Start(ctx)
}

// StartGRPC starts the gRPC server
func (s *Server) StartGRPC(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.GRPC.Host, s.cfg.GRPC.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("Starting gRPC server", "addr", addr)

	// Start server in a goroutine
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info("Shutting down gRPC server")
	s.grpcServer.GracefulStop()

	return nil
}

// StartHealthCheck starts the health check HTTP server
func (s *Server) StartHealthCheck(ctx context.Context) error {
	s.logger.Info("Starting health check server", "addr", s.httpServer.Addr)

	// Start health manager
	s.healthManager.Start(ctx)

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Health check server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info("Shutting down health check server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}

// StartMetrics starts the metrics server
func (s *Server) StartMetrics(ctx context.Context) error {
	if s.metricsServer == nil {
		s.logger.Info("Metrics server disabled")
		return nil
	}

	s.logger.Info("Starting metrics server")

	// Start server in a goroutine
	go func() {
		if err := s.metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Metrics server error", "error", err)
		}
	}()

	// Set up graceful shutdown in a separate goroutine
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down metrics server")
		if err := s.metricsServer.Shutdown(context.Background()); err != nil {
			s.logger.Error("Error shutting down metrics server", "error", err)
		}
	}()

	return nil
}

// StartWebRTC starts the WebRTC gateway server
func (s *Server) StartWebRTC(ctx context.Context) error {
	if s.webrtcServer == nil {
		s.logger.Info("WebRTC gateway disabled")
		return nil
	}

	s.logger.Info("Starting WebRTC gateway server", "addr", s.webrtcServer.Addr)

	// Update metrics
	if s.metricsServer != nil {
		s.metricsServer.GetCollector().UpdateComponentHealth("webrtc_gateway", true)
	}

	// Start server in a goroutine
	go func() {
		if err := s.webrtcServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebRTC gateway server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info("Shutting down WebRTC gateway server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown WebRTC gateway gracefully
	if s.webrtcGateway != nil {
		if err := s.webrtcGateway.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Error shutting down WebRTC gateway", "error", err)
		}
	}

	return s.webrtcServer.Shutdown(shutdownCtx)
}

// Shutdown gracefully shuts down all server components
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down B2BUA server")

	// Stop health manager first
	if s.healthManager != nil {
		s.logger.Info("Stopping health manager")
		s.healthManager.Stop()
	}

	// Stop metrics server
	if s.metricsServer != nil {
		s.logger.Info("Stopping metrics server")
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.Error("Error shutting down metrics server", "error", err)
		}
	}

	// Stop accepting new requests
	if s.grpcServer != nil {
		s.logger.Info("Stopping gRPC server")
		s.grpcServer.GracefulStop()
	}

	// Stop HTTP health check server
	if s.httpServer != nil {
		s.logger.Info("Stopping health check server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Error shutting down health check server", "error", err)
		}
	}

	// Stop SIP server
	if s.sipServer != nil {
		s.logger.Info("Stopping SIP server")
		s.sipRunning = false
		// Update metrics
		if s.metricsServer != nil {
			s.metricsServer.GetCollector().UpdateComponentHealth("sip_server", false)
		}
		// SIP server shutdown would be implemented in the sip package
		// For now, we just mark it as stopped
	}

	// Close etcd connection
	if s.etcdClient != nil {
		s.logger.Info("Closing etcd connection")
		if err := s.etcdClient.Close(); err != nil {
			s.logger.Error("Error closing etcd connection", "error", err)
		}
	}

	// Close Redis connection
	if s.redisClient != nil {
		s.logger.Info("Closing Redis connection")
		if err := s.redisClient.Close(); err != nil {
			s.logger.Error("Error closing Redis connection", "error", err)
		}
	}

	// Close rtpengine connection
	if s.rtpEngine != nil {
		s.logger.Info("Closing rtpengine connection")
		s.rtpEngine.Close()
	}

	s.logger.Info("B2BUA server shutdown complete")
	return nil
}

// GetEtcdClient returns the etcd client
func (s *Server) GetEtcdClient() *etcd.Client {
	return s.etcdClient
}

// GetRedisClient returns the Redis client
func (s *Server) GetRedisClient() *redis.Client {
	return s.redisClient
}

// HealthCheck performs health checks on all components
func (s *Server) HealthCheck(ctx context.Context) error {
	// Check SIP server
	// (SIP server health check would be implemented in sip package)

	// Check rtpengine
	instances := s.rtpEngine.GetInstances()
	for _, instance := range instances {
		if instance.Enabled {
			if _, err := s.rtpEngine.Ping(ctx, instance.ID); err != nil {
				return fmt.Errorf("rtpengine instance %s health check failed: %w", instance.ID, err)
			}
		}
	}

	// Check etcd
	if s.etcdClient != nil {
		if err := s.etcdClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("etcd health check failed: %w", err)
		}
	}

	// Check Redis
	if s.redisClient != nil {
		if err := s.redisClient.HealthCheck(ctx); err != nil {
			return fmt.Errorf("redis health check failed: %w", err)
		}
	}

	return nil
}
