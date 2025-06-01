package rtpengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

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
	Command string                 `json:"command"`
	CallID  string                 `json:"call-id"`
	FromTag string                 `json:"from-tag,omitempty"`
	ToTag   string                 `json:"to-tag,omitempty"`
	SDP     string                 `json:"sdp,omitempty"`
	Flags   []string               `json:"flags,omitempty"`
	Replace []string               `json:"replace,omitempty"`
	Data    map[string]interface{} `json:",inline"`
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

	// Encode command as JSON
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Add bencode wrapper (simplified - in production, use proper bencode library)
	bencoded := fmt.Sprintf("d7:command%d:%se", len(cmdBytes), cmdBytes)

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

	// Parse bencode response (simplified)
	responseData := buffer[:n]

	// Extract JSON from bencode (this is a simplified implementation)
	// In production, use a proper bencode library
	jsonStart := bytes.Index(responseData, []byte("{"))
	if jsonStart == -1 {
		return nil, fmt.Errorf("invalid response format")
	}

	jsonEnd := bytes.LastIndex(responseData, []byte("}"))
	if jsonEnd == -1 {
		return nil, fmt.Errorf("invalid response format")
	}

	jsonData := responseData[jsonStart : jsonEnd+1]

	var response Response
	if err := json.Unmarshal(jsonData, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Result != "ok" {
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
	response, err := c.Ping(ctx, instanceID)
	return err == nil && response.Result == "ok"
}
