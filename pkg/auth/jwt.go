// Package auth provides authentication and authorization functionality
package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// Claims represents JWT custom claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// JWTAuth handles JWT authentication and authorization
type JWTAuth struct {
	cfg        *config.AuthConfig
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// NewJWTAuth creates a new JWT authentication handler
func NewJWTAuth(cfg *config.AuthConfig) (*JWTAuth, error) {
	auth := &JWTAuth{
		cfg: cfg,
	}

	// Load keys if provided
	if cfg.JWT.PublicKeyPath != "" {
		// TODO: Load public key from file
		// For now, we'll use a simple secret
	}

	if cfg.JWT.PrivateKeyPath != "" {
		// TODO: Load private key from file
		// For now, we'll use a simple secret
	}

	return auth, nil
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTAuth) ValidateToken(tokenString string) (*Claims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.cfg.JWT.SigningKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GenerateToken generates a new JWT token for a user
func (j *JWTAuth) GenerateToken(userID, username string, roles, permissions []string) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.cfg.JWT.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.cfg.JWT.Expiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.cfg.JWT.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ExtractTokenFromContext extracts JWT token from gRPC metadata
func (j *JWTAuth) ExtractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders, ok := md["authorization"]
	if !ok || len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	return authHeaders[0], nil
}

// ValidatePermission checks if the user has the required permission
func (j *JWTAuth) ValidatePermission(claims *Claims, requiredPermission string) bool {
	if !j.cfg.Enabled {
		return true // Auth disabled
	}

	// Check if user has admin role
	for _, role := range claims.Roles {
		if role == "admin" || role == "superuser" {
			return true
		}
	}

	// Check specific permission
	for _, permission := range claims.Permissions {
		if permission == requiredPermission || permission == "*" {
			return true
		}
	}

	return false
}

// ValidateRole checks if the user has the required role
func (j *JWTAuth) ValidateRole(claims *Claims, requiredRole string) bool {
	if !j.cfg.Enabled {
		return true // Auth disabled
	}

	for _, role := range claims.Roles {
		if role == requiredRole || role == "admin" || role == "superuser" {
			return true
		}
	}

	return false
}

// AuthenticateContext validates authentication from gRPC context
func (j *JWTAuth) AuthenticateContext(ctx context.Context) (*Claims, error) {
	if !j.cfg.Enabled {
		return &Claims{
			UserID:      "system",
			Username:    "system",
			Roles:       []string{"admin"},
			Permissions: []string{"*"},
		}, nil
	}

	token, err := j.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	claims, err := j.ValidateToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return claims, nil
}
