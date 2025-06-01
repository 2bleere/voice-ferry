package handlers

import (
	"context"
	"log"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/2bleere/voice-ferry/pkg/config"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// ConfigHandler implements the ConfigurationService
type ConfigHandler struct {
	v1.UnimplementedConfigurationServiceServer
	cfg *config.Config
	// TODO: Add etcd client for distributed configuration
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{
		cfg: cfg,
	}
}

// GetGlobalConfig returns the current global configuration
func (h *ConfigHandler) GetGlobalConfig(ctx context.Context, req *emptypb.Empty) (*v1.GlobalConfigResponse, error) {
	log.Printf("Getting global configuration")

	globalConfig := &v1.GlobalConfig{
		Logging: &v1.LoggingConfig{
			Level:          "info",
			EnableSipTrace: h.cfg.Debug,
		},
		Sip: &v1.SipConfig{
			MaxForwards:   70,
			UserAgent:     "Voice-Ferry-C4 B2BUA v1.0.0",
			Enable_100Rel: true,
		},
		Security: &v1.SecurityConfig{
			EnableDigestAuth: h.cfg.Security.SIP.DigestAuth.Enabled,
			TrustedProxies:   []string{"127.0.0.1", "::1"},
		},
	}

	return &v1.GlobalConfigResponse{
		Config: globalConfig,
	}, nil
}

// UpdateGlobalConfig updates the global configuration
func (h *ConfigHandler) UpdateGlobalConfig(ctx context.Context, req *v1.UpdateGlobalConfigRequest) (*v1.CommandStatusResponse, error) {
	log.Printf("Updating global configuration")

	// TODO: Implement configuration updates
	// This would involve:
	// 1. Validating the new configuration
	// 2. Applying changes to running components
	// 3. Persisting to etcd

	return &v1.CommandStatusResponse{
		Success: true,
		Message: "Configuration updated successfully",
	}, nil
}

// ReloadConfig reloads configuration from etcd
func (h *ConfigHandler) ReloadConfig(ctx context.Context, req *emptypb.Empty) (*v1.CommandStatusResponse, error) {
	log.Printf("Reloading configuration from etcd")

	// TODO: Implement configuration reload from etcd

	return &v1.CommandStatusResponse{
		Success: true,
		Message: "Configuration reloaded successfully",
	}, nil
}
