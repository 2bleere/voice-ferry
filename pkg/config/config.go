package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the B2BUA configuration
type Config struct {
	Debug     bool            `yaml:"debug"`
	SIP       SIPConfig       `yaml:"sip"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	WebRTC    WebRTCConfig    `yaml:"webrtc"`
	Health    HealthConfig    `yaml:"health"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Logging   LoggingConfig   `yaml:"logging"`
	Etcd      EtcdConfig      `yaml:"etcd"`
	Redis     RedisConfig     `yaml:"redis"`
	RTPEngine RTPEngineConfig `yaml:"rtpengine"`
	Security  SecurityConfig  `yaml:"security"`
	Auth      AuthConfig      `yaml:"auth"`
}

// SIPConfig contains SIP-related configuration
type SIPConfig struct {
	Host      string        `yaml:"host"`
	Port      int           `yaml:"port"`
	Transport string        `yaml:"transport"` // UDP, TCP, TLS, WS, WSS
	TLS       TLSConfig     `yaml:"tls"`
	Timeouts  TimeoutConfig `yaml:"timeouts"`
	Auth      SIPAuthConfig `yaml:"auth"`
}

// GRPCConfig contains gRPC API configuration
type GRPCConfig struct {
	Host string         `yaml:"host"`
	Port int            `yaml:"port"`
	TLS  TLSConfig      `yaml:"tls"`
	Auth GRPCAuthConfig `yaml:"auth"`
}

// GRPCAuthConfig contains gRPC authentication configuration
type GRPCAuthConfig struct {
	Enabled   bool   `yaml:"enabled"`
	JWTSecret string `yaml:"jwt_secret"`
}

// WebRTCConfig contains WebRTC gateway configuration
type WebRTCConfig struct {
	Enabled     bool               `yaml:"enabled"`
	Host        string             `yaml:"host"`
	Port        int                `yaml:"port"`
	WSPath      string             `yaml:"ws_path"`
	STUNServers []string           `yaml:"stun_servers"`
	TURNServers []TURNServerConfig `yaml:"turn_servers"`
	TLS         TLSConfig          `yaml:"tls"`
	Auth        WebRTCAuthConfig   `yaml:"auth"`
}

// TURNServerConfig represents a TURN server configuration
type TURNServerConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// WebRTCAuthConfig contains WebRTC authentication configuration
type WebRTCAuthConfig struct {
	Enabled   bool     `yaml:"enabled"`
	JWTSecret string   `yaml:"jwt_secret"`
	Origins   []string `yaml:"allowed_origins"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// MetricsConfig contains metrics and monitoring configuration
type MetricsConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Path         string        `yaml:"path"`
	UpdatePeriod time.Duration `yaml:"update_period"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level         string `yaml:"level"`          // debug, info, warn, error
	Format        string `yaml:"format"`         // json, text
	Output        string `yaml:"output"`         // stdout, stderr, file
	File          string `yaml:"file"`           // log file path when output is file
	IncludeSource bool   `yaml:"include_source"` // include source code location
	Version       string `yaml:"version"`        // application version
	InstanceID    string `yaml:"instance_id"`    // instance identifier
}

// EtcdConfig contains etcd connection configuration
type EtcdConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Endpoints   []string      `yaml:"endpoints"`
	DialTimeout time.Duration `yaml:"dial_timeout"`
	Username    string        `yaml:"username"`
	Password    string        `yaml:"password"`
	TLS         TLSConfig     `yaml:"tls"`
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Enabled             bool           `yaml:"enabled"`
	Host                string         `yaml:"host"`
	Port                int            `yaml:"port"`
	Password            string         `yaml:"password"`
	Database            int            `yaml:"database"`
	PoolSize            int            `yaml:"pool_size"`
	MinIdleConns        int            `yaml:"min_idle_conns"`
	EnableSessionLimits bool           `yaml:"enable_session_limits"` // Enable per-user session limits
	MaxSessionsPerUser  int            `yaml:"max_sessions_per_user"` // Default max allowed active sessions per user
	UserSessionLimits   map[string]int `yaml:"user_session_limits"`   // Per-user session limits map
	SessionLimitAction  string         `yaml:"session_limit_action"`  // Action when limit reached: "reject" or "terminate_oldest"
	MaxIdleTime         int            `yaml:"max_idle_time"`         // in seconds
	ConnMaxLifetime     int            `yaml:"conn_max_lifetime"`     // in seconds
	Timeout             int            `yaml:"timeout"`               // in seconds
	TLS                 TLSConfig      `yaml:"tls"`
}

// RTPEngineConfig contains rtpengine configuration
type RTPEngineConfig struct {
	Instances []RTPEngineInstance `yaml:"instances"`
	Timeout   time.Duration       `yaml:"timeout"`
}

// RTPEngineInstance represents a single rtpengine instance
type RTPEngineInstance struct {
	ID      string `yaml:"id"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Weight  int    `yaml:"weight"`
	Enabled bool   `yaml:"enabled"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	JWT JWTConfig         `yaml:"jwt"`
	SIP SIPSecurityConfig `yaml:"sip"`
}

// JWTConfig contains JWT authentication configuration
type JWTConfig struct {
	SigningKey string        `yaml:"signing_key"`
	Expiration time.Duration `yaml:"expiration"`
	Issuer     string        `yaml:"issuer"`
}

// SIPSecurityConfig contains SIP security configuration
type SIPSecurityConfig struct {
	IPACLs     []IPACL          `yaml:"ip_acls"`
	DigestAuth DigestAuthConfig `yaml:"digest_auth"`
}

// IPACL represents an IP access control list entry
type IPACL struct {
	Name     string   `yaml:"name"`
	Action   string   `yaml:"action"` // allow, deny
	Networks []string `yaml:"networks"`
}

// DigestAuthConfig contains SIP digest authentication configuration
type DigestAuthConfig struct {
	Realm   string `yaml:"realm"`
	Enabled bool   `yaml:"enabled"`
}

// SIPAuthConfig contains SIP authentication configuration
type SIPAuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Realm   string `yaml:"realm"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// TimeoutConfig contains various timeout settings
type TimeoutConfig struct {
	Transaction  time.Duration `yaml:"transaction"`
	Dialog       time.Duration `yaml:"dialog"`
	Registration time.Duration `yaml:"registration"`
}

// AuthConfig contains authentication and authorization configuration
type AuthConfig struct {
	Enabled     bool          `yaml:"enabled"`
	IPWhitelist []string      `yaml:"ip_whitelist"`
	IPBlacklist []string      `yaml:"ip_blacklist"`
	JWT         JWTAuthConfig `yaml:"jwt"`
}

// JWTAuthConfig contains JWT-specific authentication configuration
type JWTAuthConfig struct {
	PublicKeyPath  string        `yaml:"public_key_path"`
	PrivateKeyPath string        `yaml:"private_key_path"`
	SigningKey     string        `yaml:"signing_key"`
	Expiration     time.Duration `yaml:"expiration"`
	Issuer         string        `yaml:"issuer"`
}

// Load loads configuration from file or returns default configuration
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	// If file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	return &Config{
		Debug: false,
		SIP: SIPConfig{
			Host:      "0.0.0.0",
			Port:      5060,
			Transport: "UDP",
			Timeouts: TimeoutConfig{
				Transaction:  32 * time.Second,
				Dialog:       1800 * time.Second, // 30 minutes
				Registration: 3600 * time.Second, // 1 hour
			},
			Auth: SIPAuthConfig{
				Enabled: false,
				Realm:   "sip.example.com", // Default realm
			},
		},
		GRPC: GRPCConfig{
			Host: "0.0.0.0",
			Port: 50051,
			Auth: GRPCAuthConfig{
				Enabled: false,
			},
		},
		WebRTC: WebRTCConfig{
			Enabled: false,
			Host:    "0.0.0.0",
			Port:    8081,
			WSPath:  "/ws",
			STUNServers: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
			TURNServers: []TURNServerConfig{},
			Auth: WebRTCAuthConfig{
				Enabled: false,
				Origins: []string{"*"},
			},
		},
		Health: HealthConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    9090,
			Path:    "/metrics",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Etcd: EtcdConfig{
			Enabled:     false,
			Endpoints:   []string{"127.0.0.1:2379"},
			DialTimeout: 5 * time.Second,
		},
		Redis: RedisConfig{
			Enabled:             false,
			Host:                "127.0.0.1",
			Port:                6379,
			Database:            0,
			PoolSize:            10,
			MinIdleConns:        5,
			MaxIdleTime:         300,
			ConnMaxLifetime:     3600,
			Timeout:             5,
			EnableSessionLimits: false,
			MaxSessionsPerUser:  5,
			SessionLimitAction:  "reject",
		},
		RTPEngine: RTPEngineConfig{
			Instances: []RTPEngineInstance{
				{
					ID:      "rtpengine-1",
					Host:    "127.0.0.1",
					Port:    22222,
					Weight:  100,
					Enabled: true,
				},
			},
			Timeout: 5 * time.Second,
		},
		Security: SecurityConfig{
			JWT: JWTConfig{
				SigningKey: "your-secret-key",
				Expiration: 24 * time.Hour,
				Issuer:     "b2bua",
			},
			SIP: SIPSecurityConfig{
				IPACLs: []IPACL{
					{
						Name:     "localhost",
						Action:   "allow",
						Networks: []string{"127.0.0.0/8", "::1/128"},
					},
				},
				DigestAuth: DigestAuthConfig{
					Realm:   "sip.example.com",
					Enabled: false,
				},
			},
		},
		Auth: AuthConfig{
			Enabled: true,
			IPWhitelist: []string{
				"127.0.0.1",
				"::1",
			},
			IPBlacklist: []string{
				"192.168.1.100",
			},
			JWT: JWTAuthConfig{
				PublicKeyPath:  "/etc/ssl/public.key",
				PrivateKeyPath: "/etc/ssl/private.key",
				SigningKey:     "your-signing-key",
				Expiration:     24 * time.Hour,
				Issuer:         "b2bua",
			},
		},
	}
}
