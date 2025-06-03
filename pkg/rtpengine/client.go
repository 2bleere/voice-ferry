package rtpengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
	"crypto/rand"
	"encoding/hex"

	"github.com/2bleere/voice-ferry/pkg/config"
)

// Client represents an rtpengine client using the 'ng' protocol
type Client struct {
	instances []config.RTPEngineInstance
	timeout   time.Duration
	mu        sync.RWMutex
	conns     map[string]*net.UDPConn // connection pool per instance
}

// Command represents an rtpengine ng protocol command
type Command struct {
	Command string   `json:"command"`
	CallID  string   `json:"call-id"`
	FromTag string   `json:"from-tag,omitempty"`
	ToTag   string   `json:"to-tag,omitempty"`
	SDP     string   `json:"sdp,omitempty"`
	Flags   []string `json:"flags,omitempty"`
	Replace []string `json:"replace,omitempty"`
}

// Response represents an rtpengine ng protocol response
type Response struct {
	Result      string `json:"result"`
	SDP         string `json:"sdp,omitempty"`
	ErrorReason string `json:"error-reason,omitempty"`
}

// MediaSession represents an active media session
type MediaSession struct {
	CallID       string
	FromTag      string
	ToTag        string
	InstanceID   string
	CreatedAt    time.Time
	LastActivity time.Time
}

// NewClient creates a new rtpengine client
func NewClient(cfg config.RTPEngineConfig) (*Client, error) {
	client := &Client{
		instances: cfg.Instances,
		timeout:   cfg.Timeout,
		conns:     make(map[string]*net.UDPConn),
	}

	// Initialize connections to rtpengine instances
	for _, instance := range cfg.Instances {
		if instance.Enabled {
			if err := client.initConnection(instance); err != nil {
				return nil, fmt.Errorf("failed to connect to rtpengine %s: %w", instance.ID, err)
			}
		}
	}

	return client, nil
}

// generateCookie generates a unique cookie for RTPEngine commands
func generateCookie() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// initConnection initializes a UDP connection to an rtpengine instance
func (c *Client) initConnection(instance config.RTPEngineInstance) error {
	addr := fmt.Sprintf("%s:%d", instance.Host, instance.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", addr, err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP %s: %w", addr, err)
	}

	c.mu.Lock()
	c.conns[instance.ID] = conn
	c.mu.Unlock()

	return nil
}

// Offer initiates a media session with rtpengine
func (c *Client) Offer(ctx context.Context, callID, fromTag, sdp string, flags []string) (*Response, error) {
	cmd := Command{
		Command: "offer",
		CallID:  callID,
		FromTag: fromTag,
		SDP:     sdp,
		Flags:   flags,
	}

	return c.sendCommand(ctx, cmd)
}

// Answer completes a media session negotiation
func (c *Client) Answer(ctx context.Context, callID, fromTag, toTag, sdp string, flags []string) (*Response, error) {
	cmd := Command{
		Command: "answer",
		CallID:  callID,
		FromTag: fromTag,
		ToTag:   toTag,
		SDP:     sdp,
		Flags:   flags,
	}

	return c.sendCommand(ctx, cmd)
}

// Delete terminates a media session
func (c *Client) Delete(ctx context.Context, callID, fromTag, toTag string) (*Response, error) {
	cmd := Command{
		Command: "delete",
		CallID:  callID,
		FromTag: fromTag,
		ToTag:   toTag,
	}

	return c.sendCommand(ctx, cmd)
}

// Query gets information about a media session
func (c *Client) Query(ctx context.Context, callID, fromTag, toTag string) (*Response, error) {
	cmd := Command{
		Command: "query",
		CallID:  callID,
		FromTag: fromTag,
		ToTag:   toTag,
	}

	return c.sendCommand(ctx, cmd)
}

// Ping checks if an rtpengine instance is alive
func (c *Client) Ping(ctx context.Context, instanceID string) (*Response, error) {
	cmd := Command{
		Command: "ping",
	}

	return c.sendCommandToInstance(ctx, cmd, instanceID)
}

// sendCommand sends a command to an available rtpengine instance
func (c *Client) sendCommand(ctx context.Context, cmd Command) (*Response, error) {
	// Simple round-robin load balancing
	// In production, this could be enhanced with health checks and weights
	c.mu.RLock()
	var selectedInstance string
	for _, instance := range c.instances {
		if instance.Enabled {
			if _, exists := c.conns[instance.ID]; exists {
				selectedInstance = instance.ID
				break
			}
		}
	}
	c.mu.RUnlock()

	if selectedInstance == "" {
		return nil, fmt.Errorf("no available rtpengine instances")
	}

	return c.sendCommandToInstance(ctx, cmd, selectedInstance)
}

// sendCommandToInstance sends a command to a specific rtpengine instance
func (c *Client) sendCommandToInstance(ctx context.Context, cmd Command, instanceID string) (*Response, error) {
	c.mu.RLock()
	conn, exists := c.conns[instanceID]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no connection to rtpengine instance %s", instanceID)
	}

	// Generate cookie for this command
	cookie := generateCookie()

	// For simple commands like "ping", send as raw string
	// For complex commands, encode as JSON
	var commandStr string
	var err error
	if cmd.Command == "ping" {
		commandStr = "ping"
	} else {
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command: %w", err)
		}
		commandStr = string(cmdBytes)
	}

	// Create proper RTPEngine NG format: cookie + space + bencode dictionary
	// Format: cookie d7:command<len>:<command>e
	bencoded := fmt.Sprintf("%s d7:command%d:%se", cookie, len(commandStr), commandStr)

	// Set deadline for the operation
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	conn.SetDeadline(deadline)

	// Send command
	_, err = conn.Write([]byte(bencoded))
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse RTPEngine response format: cookie d6:result<len>:<result>e
	responseData := buffer[:n]
	
	// Find the space separator
	spaceIndex := bytes.Index(responseData, []byte(" "))
	if spaceIndex == -1 {
		return nil, fmt.Errorf("invalid response format: no space separator")
	}
	
	// Extract the bencode part after the space
	bencodeData := responseData[spaceIndex+1:]
	
	// For simple responses like "pong", extract directly
	// Look for result field in bencode: d6:result<len>:<value>e
	resultStart := bytes.Index(bencodeData, []byte("result"))
	if resultStart == -1 {
		return nil, fmt.Errorf("no result field in response")
	}
	
	// Find the length after "result"
	lengthStart := resultStart + 6 // len("result")
	colonIndex := bytes.Index(bencodeData[lengthStart:], []byte(":"))
	if colonIndex == -1 {
		return nil, fmt.Errorf("invalid result format")
	}
	
	// Parse length
	lengthStr := string(bencodeData[lengthStart : lengthStart+colonIndex])
	var resultLength int
	if _, err := fmt.Sscanf(lengthStr, "%d", &resultLength); err != nil {
		return nil, fmt.Errorf("invalid result length: %w", err)
	}
	
	// Extract result value
	valueStart := lengthStart + colonIndex + 1
	if valueStart+resultLength > len(bencodeData) {
		return nil, fmt.Errorf("result value exceeds data length")
	}
	
	resultValue := string(bencodeData[valueStart : valueStart+resultLength])
	
	// Create response
	response := Response{
		Result: resultValue,
	}
	
	// For ping command, "pong" result means success
	if cmd.Command == "ping" && resultValue == "pong" {
		response.Result = "ok"
	}

	if response.Result != "ok" && response.Result != "pong" {
		return &response, fmt.Errorf("rtpengine error: %s", response.ErrorReason)
	}

	return &response, nil
}

// Close closes all connections to rtpengine instances
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for instanceID, conn := range c.conns {
		if err := conn.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close connection to %s: %w", instanceID, err)
		}
	}

	return lastErr
}

// GetInstances returns the list of configured rtpengine instances
func (c *Client) GetInstances() []config.RTPEngineInstance {
	return c.instances
}

// IsInstanceHealthy checks if an rtpengine instance is healthy
func (c *Client) IsInstanceHealthy(ctx context.Context, instanceID string) bool {
	// Use a fresh UDP connection for health checks to avoid connection reuse issues in Kubernetes
	response, err := c.pingWithFreshConnection(ctx, instanceID)
	if err != nil {
		// Add debug logging to see what's going wrong
		fmt.Printf("DEBUG: RTPEngine ping failed for instance %s: %v\n", instanceID, err)
		return false
	}
	
	healthy := response.Result == "ok"
	fmt.Printf("DEBUG: RTPEngine ping for instance %s: result=%s, healthy=%v\n", instanceID, response.Result, healthy)
	return healthy
}

// pingWithFreshConnection creates a fresh UDP connection for each ping to avoid connection reuse issues
func (c *Client) pingWithFreshConnection(ctx context.Context, instanceID string) (*Response, error) {
	// Find the instance configuration
	var instance *config.RTPEngineInstance
	for _, inst := range c.instances {
		if inst.ID == instanceID {
			instance = &inst
			break
		}
	}
	
	if instance == nil {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	
	// Create fresh UDP connection
	addr := fmt.Sprintf("%s:%d", instance.Host, instance.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %s: %w", addr, err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP %s: %w", addr, err)
	}
	defer conn.Close()

	// Generate cookie for this command
	cookie := generateCookie()

	// Create proper RTPEngine NG format for ping: cookie d7:command4:pinge
	bencoded := fmt.Sprintf("%s d7:command4:pinge", cookie)

	// Set deadline for the operation
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	conn.SetDeadline(deadline)

	// Send command
	_, err = conn.Write([]byte(bencoded))
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse RTPEngine response format: cookie d6:result<len>:<result>e
	responseData := buffer[:n]
	
	// Find the space separator
	spaceIndex := bytes.Index(responseData, []byte(" "))
	if spaceIndex == -1 {
		return nil, fmt.Errorf("invalid response format: no space separator")
	}
	
	// Extract the bencode part after the space
	bencodeData := responseData[spaceIndex+1:]
	
	// For simple responses like "pong", extract directly
	// Look for result field in bencode: d6:result<len>:<value>e
	resultStart := bytes.Index(bencodeData, []byte("result"))
	if resultStart == -1 {
		return nil, fmt.Errorf("no result field in response")
	}
	
	// Find the length after "result"
	lengthStart := resultStart + 6 // len("result")
	colonIndex := bytes.Index(bencodeData[lengthStart:], []byte(":"))
	if colonIndex == -1 {
		return nil, fmt.Errorf("invalid result format")
	}
	
	// Parse length
	lengthStr := string(bencodeData[lengthStart : lengthStart+colonIndex])
	var resultLength int
	if _, err := fmt.Sscanf(lengthStr, "%d", &resultLength); err != nil {
		return nil, fmt.Errorf("invalid result length: %w", err)
	}
	
	// Extract result value
	valueStart := lengthStart + colonIndex + 1
	if valueStart+resultLength > len(bencodeData) {
		return nil, fmt.Errorf("result value exceeds data length")
	}
	
	result := string(bencodeData[valueStart : valueStart+resultLength])
	
	// Convert "pong" to "ok" for consistency with Voice Ferry expectations  
	if result == "pong" {
		result = "ok"
	}
	
	return &Response{
		Result: result,
	}, nil
}
