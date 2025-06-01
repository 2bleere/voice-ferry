package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/2bleere/voice-ferry/internal/server"
	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/metrics"
)

const (
	defaultConfigPath = "/etc/b2bua/config.yaml"
	defaultSIPPort    = 5060
	defaultGRPCPort   = 50051
)

// Build-time variables
var (
	version   = "dev"
	buildTime = "unknown"
	commitSHA = "unknown"
)

func main() {
	var (
		configPath  = flag.String("config", defaultConfigPath, "Path to configuration file")
		sipPort     = flag.Int("sip-port", defaultSIPPort, "SIP listening port")
		grpcPort    = flag.Int("grpc-port", defaultGRPCPort, "gRPC API port")
		debug       = flag.Bool("debug", false, "Enable debug logging")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()
	// Initialize metrics only once at startup
	metrics.InitializeMetrics()
	// Show version if requested
	if *showVersion {
		log.Printf("SIP B2BUA version: %s, build time: %s, commit: %s", version, buildTime, commitSHA)
		return
	}
	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override with command line flags
	if *sipPort != defaultSIPPort {
		cfg.SIP.Port = *sipPort
	}
	if *grpcPort != defaultGRPCPort {
		cfg.GRPC.Port = *grpcPort
	}
	cfg.Debug = *debug

	// Setup logging
	if cfg.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Printf("Starting SIP B2BUA v%s (built: %s, commit: %s)", version, buildTime, commitSHA)
	//log.Printf("Starting SIP B2BUA v1.0.0")
	log.Printf("SIP Port: %d, gRPC Port: %d", cfg.SIP.Port, cfg.GRPC.Port)
	if cfg.WebRTC.Enabled {
		log.Printf("WebRTC Gateway: enabled on port %d", cfg.WebRTC.Port)
	}

	// Create server instance
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start SIP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.StartSIP(ctx); err != nil {
			log.Printf("SIP server error: %v", err)
		}
	}()

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.StartGRPC(ctx); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start health check server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.StartHealthCheck(ctx); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()

	// Start metrics server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.StartMetrics(ctx); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Start WebRTC server if enabled
	if cfg.WebRTC.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.StartWebRTC(ctx); err != nil {
				log.Printf("WebRTC server error: %v", err)
			}
		}()
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Received shutdown signal, gracefully shutting down...")

	// Cancel context
	cancel()

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown server components
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Shutdown completed")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
	}
}
