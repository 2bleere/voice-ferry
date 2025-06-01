package auth

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// IPACLAuth handles IP-based access control
type IPACLAuth struct {
	cfg         *config.AuthConfig
	allowedNets []*net.IPNet
	blockedNets []*net.IPNet
}

// NewIPACLAuth creates a new IP ACL authentication handler
func NewIPACLAuth(cfg *config.AuthConfig) (*IPACLAuth, error) {
	auth := &IPACLAuth{
		cfg: cfg,
	}

	// Parse allowed IP ranges
	for _, ipRange := range cfg.IPWhitelist {
		_, ipNet, err := net.ParseCIDR(ipRange)
		if err != nil {
			// Try parsing as single IP
			ip := net.ParseIP(ipRange)
			if ip == nil {
				continue // Skip invalid IPs
			}
			// Convert single IP to CIDR
			if ip.To4() != nil {
				_, ipNet, _ = net.ParseCIDR(ipRange + "/32")
			} else {
				_, ipNet, _ = net.ParseCIDR(ipRange + "/128")
			}
		}
		auth.allowedNets = append(auth.allowedNets, ipNet)
	}

	// Parse blocked IP ranges
	for _, ipRange := range cfg.IPBlacklist {
		_, ipNet, err := net.ParseCIDR(ipRange)
		if err != nil {
			// Try parsing as single IP
			ip := net.ParseIP(ipRange)
			if ip == nil {
				continue // Skip invalid IPs
			}
			// Convert single IP to CIDR
			if ip.To4() != nil {
				_, ipNet, _ = net.ParseCIDR(ipRange + "/32")
			} else {
				_, ipNet, _ = net.ParseCIDR(ipRange + "/128")
			}
		}
		auth.blockedNets = append(auth.blockedNets, ipNet)
	}

	return auth, nil
}

// ValidateIP checks if an IP address is allowed
func (i *IPACLAuth) ValidateIP(ctx context.Context) error {
	if !i.cfg.Enabled {
		return nil // Auth disabled
	}

	// Extract client IP from context
	clientIP, err := i.extractClientIP(ctx)
	if err != nil {
		return status.Error(codes.Internal, "failed to extract client IP")
	}

	// Check if IP is in blacklist
	for _, blockedNet := range i.blockedNets {
		if blockedNet.Contains(clientIP) {
			return status.Error(codes.PermissionDenied, "IP address is blocked")
		}
	}

	// If no whitelist is configured, allow all non-blacklisted IPs
	if len(i.allowedNets) == 0 {
		return nil
	}

	// Check if IP is in whitelist
	for _, allowedNet := range i.allowedNets {
		if allowedNet.Contains(clientIP) {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "IP address not allowed")
}

// extractClientIP extracts the client IP from gRPC context
func (i *IPACLAuth) extractClientIP(ctx context.Context) (net.IP, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get peer from context")
	}

	// Extract IP from address
	host, _, err := net.SplitHostPort(peer.Addr.String())
	if err != nil {
		return nil, err
	}

	// Handle IPv6 addresses with brackets
	host = strings.Trim(host, "[]")

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, status.Error(codes.Internal, "invalid IP address")
	}

	return ip, nil
}

// GetClientIP returns the client IP address from context
func (i *IPACLAuth) GetClientIP(ctx context.Context) string {
	ip, err := i.extractClientIP(ctx)
	if err != nil {
		return "unknown"
	}
	return ip.String()
}
