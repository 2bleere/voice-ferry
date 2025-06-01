// Package auth provides SIP digest authentication functionality
package auth

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/emiago/sipgo/sip"
)

// DigestAuth handles SIP digest authentication
type DigestAuth struct {
	cfg     *config.Config
	users   map[string]*User  // username -> user
	nonces  map[string]*Nonce // nonce -> nonce info
	mu      sync.RWMutex
	nonceMu sync.RWMutex
}

// User represents a SIP user
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Realm    string `json:"realm"`
	Enabled  bool   `json:"enabled"`
}

// Nonce represents a digest authentication nonce
type Nonce struct {
	Value     string
	CreatedAt time.Time
	ClientIP  string
	Used      bool
}

// NewDigestAuth creates a new digest authentication handler
func NewDigestAuth(cfg *config.Config) *DigestAuth {
	auth := &DigestAuth{
		cfg:    cfg,
		users:  make(map[string]*User),
		nonces: make(map[string]*Nonce),
	}

	// Add default test user using the realm from the configuration
	auth.AddUser("787", "12345", cfg.SIP.Auth.Realm, true)

	// Start nonce cleanup goroutine in the background
	go auth.cleanupNonces()

	return auth
}

// AddUser adds a user to the authentication database
func (d *DigestAuth) AddUser(username, password, realm string, enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.users[username] = &User{
		Username: username,
		Password: password,
		Realm:    realm,
		Enabled:  enabled,
	}
}

// GetUser retrieves a user from the authentication database
func (d *DigestAuth) GetUser(username string) (*User, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	user, exists := d.users[username]
	return user, exists
}

// GenerateNonce generates a new nonce for digest authentication
func (d *DigestAuth) GenerateNonce(clientIP string) string {
	// Generate random bytes
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Create nonce from random bytes + timestamp
	nonce := fmt.Sprintf("%x%x", bytes, time.Now().Unix())

	d.nonceMu.Lock()
	defer d.nonceMu.Unlock()

	d.nonces[nonce] = &Nonce{
		Value:     nonce,
		CreatedAt: time.Now(),
		ClientIP:  clientIP,
		Used:      false,
	}

	return nonce
}

// ValidateNonce validates a nonce and marks it as used
func (d *DigestAuth) ValidateNonce(nonce string) bool {
	d.nonceMu.Lock()
	defer d.nonceMu.Unlock()

	nonceInfo, exists := d.nonces[nonce]
	if !exists {
		return false
	}

	// Check if nonce is expired (5 minutes)
	if time.Since(nonceInfo.CreatedAt) > 5*time.Minute {
		delete(d.nonces, nonce)
		return false
	}

	// Mark as used
	nonceInfo.Used = true
	return true
}

// CreateChallenge creates a WWW-Authenticate challenge header
func (d *DigestAuth) CreateChallenge(clientIP string) sip.Header {
	nonce := d.GenerateNonce(clientIP)
	realm := d.cfg.SIP.Auth.Realm

	challenge := fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5, qop="auth"`, realm, nonce)

	return sip.NewHeader("WWW-Authenticate", challenge)
}

// ValidateCredentials validates digest authentication credentials
func (d *DigestAuth) ValidateCredentials(authHeader string, method, uri string) (bool, string) {
	// Parse Authorization header
	params := d.parseAuthHeader(authHeader)
	if params == nil {
		return false, ""
	}

	username := params["username"]
	realm := params["realm"]
	nonce := params["nonce"]
	response := params["response"]

	// Validate nonce
	if !d.ValidateNonce(nonce) {
		return false, username
	}

	// Get user
	user, exists := d.GetUser(username)
	if !exists || !user.Enabled {
		return false, username
	}

	// Validate realm
	if realm != user.Realm {
		return false, username
	}

	// Calculate expected response
	expectedResponse := d.calculateResponse(username, realm, user.Password, method, uri, nonce, params["nc"], params["cnonce"], params["qop"])

	return response == expectedResponse, username
}

// parseAuthHeader parses the Authorization header and extracts parameters
func (d *DigestAuth) parseAuthHeader(authHeader string) map[string]string {
	if !strings.HasPrefix(authHeader, "Digest ") {
		return nil
	}

	params := make(map[string]string)
	content := strings.TrimPrefix(authHeader, "Digest ")

	// Split by comma and parse key=value pairs
	parts := strings.Split(content, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])

			// Remove quotes
			if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
				value = value[1 : len(value)-1]
			}

			params[key] = value
		}
	}

	return params
}

// calculateResponse calculates the digest response
func (d *DigestAuth) calculateResponse(username, realm, password, method, uri, nonce, nc, cnonce, qop string) string {
	// HA1 = MD5(username:realm:password)
	ha1 := d.md5Hash(fmt.Sprintf("%s:%s:%s", username, realm, password))

	// HA2 = MD5(method:uri)
	ha2 := d.md5Hash(fmt.Sprintf("%s:%s", method, uri))

	// Response calculation depends on qop
	var response string
	if qop == "auth" {
		// Response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		response = d.md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	} else {
		// Response = MD5(HA1:nonce:HA2)
		response = d.md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	}

	return response
}

// md5Hash calculates MD5 hash and returns hex string
func (d *DigestAuth) md5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// cleanupNonces periodically removes expired nonces
func (d *DigestAuth) cleanupNonces() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.nonceMu.Lock()
		now := time.Now()
		for nonce, info := range d.nonces {
			// Remove nonces older than 10 minutes
			if now.Sub(info.CreatedAt) > 10*time.Minute {
				delete(d.nonces, nonce)
			}
		}
		d.nonceMu.Unlock()
	}
}

// IsAuthenticationRequired returns whether authentication is required
func (da *DigestAuth) IsAuthenticationRequired() bool {
	return da.cfg.SIP.Auth.Enabled
}

// ExtractUsername extracts the username from a digest authentication header value
func (d *DigestAuth) ExtractUsername(authValue string) (string, bool) {
	// Example header: Digest username="user123", realm="example.com", nonce="abcdef", uri="sip:example.com", response="1234abcd"
	if !strings.HasPrefix(authValue, "Digest ") {
		return "", false
	}

	// Extract username parameter
	usernameStart := strings.Index(authValue, "username=\"")
	if usernameStart == -1 {
		return "", false
	}

	usernameStart += len("username=\"")
	usernameEnd := strings.Index(authValue[usernameStart:], "\"")
	if usernameEnd == -1 {
		return "", false
	}

	username := authValue[usernameStart : usernameStart+usernameEnd]
	return username, true
}
