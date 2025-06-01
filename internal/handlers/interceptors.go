package handlers

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// LoggingInterceptor logs gRPC requests
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("gRPC call: %s", info.FullMethod)

	resp, err := handler(ctx, req)

	if err != nil {
		log.Printf("gRPC error for %s: %v", info.FullMethod, err)
	}

	return resp, err
}

// AuthInterceptor handles authentication for gRPC requests
func AuthInterceptor(cfg *config.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health checks
		if info.FullMethod == "/v1.v1.StatusService/HealthCheck" {
			return handler(ctx, req)
		}

		// If authentication is disabled, skip auth checks
		if !cfg.GRPC.Auth.Enabled {
			return handler(ctx, req)
		}

		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Check for authorization header
		auth := md.Get("authorization")
		if len(auth) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		// TODO: Implement JWT validation
		// For now, accept any non-empty auth header
		if auth[0] == "" {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header")
		}

		return handler(ctx, req)
	}
}
