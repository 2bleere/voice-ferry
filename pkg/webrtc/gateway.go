package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/sip"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Gateway represents the WebRTC-to-SIP gateway
type Gateway struct {
	config      *config.WebRTCConfig
	sipServer   *sip.Server
	sessionMgr  *sip.SessionManager
	upgrader    websocket.Upgrader
	sessions    map[string]*WebRTCSession
	mu          sync.RWMutex
	logger      *slog.Logger
	stunServers []string
	turnServers []webrtc.ICEServer
}

// WebRTCSession represents an active WebRTC-to-SIP session
type WebRTCSession struct {
	ID             string                 `json:"id"`
	WebRTCConn     *websocket.Conn        `json:"-"`
	PeerConnection *webrtc.PeerConnection `json:"-"`
	SIPSessionID   string                 `json:"sip_session_id"`
	CallID         string                 `json:"call_id"`
	CallerURI      string                 `json:"caller_uri"`
	CalleeURI      string                 `json:"callee_uri"`
	State          SessionState           `json:"state"`
	CreatedAt      time.Time              `json:"created_at"`
	LastActivity   time.Time              `json:"last_activity"`
	LocalSDP       string                 `json:"local_sdp"`
	RemoteSDP      string                 `json:"remote_sdp"`
	ICEGathering   bool                   `json:"ice_gathering"`
	MediaStats     *MediaStats            `json:"media_stats"`
}

// SessionState represents the state of a WebRTC session
type SessionState int

const (
	SessionStateInitial SessionState = iota
	SessionStateConnecting
	SessionStateConnected
	SessionStateDisconnecting
	SessionStateDisconnected
)

func (s SessionState) String() string {
	switch s {
	case SessionStateInitial:
		return "Initial"
	case SessionStateConnecting:
		return "Connecting"
	case SessionStateConnected:
		return "Connected"
	case SessionStateDisconnecting:
		return "Disconnecting"
	case SessionStateDisconnected:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

// MediaStats represents media statistics for WebRTC session
type MediaStats struct {
	BytesSent       uint64    `json:"bytes_sent"`
	BytesReceived   uint64    `json:"bytes_received"`
	PacketsSent     uint64    `json:"packets_sent"`
	PacketsReceived uint64    `json:"packets_received"`
	PacketsLost     uint64    `json:"packets_lost"`
	Jitter          float64   `json:"jitter"`
	RTT             float64   `json:"rtt"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WebRTCMessage represents messages exchanged over WebSocket
type WebRTCMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// NewGateway creates a new WebRTC-to-SIP gateway
func NewGateway(cfg *config.WebRTCConfig, sipServer *sip.Server, sessionMgr *sip.SessionManager, logger *slog.Logger) (*Gateway, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// TODO: Implement proper origin checking based on config
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	gateway := &Gateway{
		config:      cfg,
		sipServer:   sipServer,
		sessionMgr:  sessionMgr,
		upgrader:    upgrader,
		sessions:    make(map[string]*WebRTCSession),
		logger:      logger,
		stunServers: cfg.STUNServers,
	}

	// Configure TURN servers
	for _, turn := range cfg.TURNServers {
		iceServer := webrtc.ICEServer{
			URLs:       []string{turn.URL},
			Username:   turn.Username,
			Credential: turn.Password,
		}
		gateway.turnServers = append(gateway.turnServers, iceServer)
	}

	return gateway, nil
}

// HandleWebSocket handles WebSocket connections for WebRTC signaling
func (g *Gateway) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}
	defer conn.Close()

	g.logger.Info("New WebRTC WebSocket connection", "remote_addr", r.RemoteAddr)

	// Handle messages
	for {
		var msg WebRTCMessage
		if err := conn.ReadJSON(&msg); err != nil {
			g.logger.Error("Failed to read WebSocket message", "error", err)
			break
		}

		if err := g.handleWebSocketMessage(conn, &msg); err != nil {
			g.logger.Error("Failed to handle WebSocket message", "error", err, "type", msg.Type)
			g.sendError(conn, msg.SessionID, err.Error())
		}
	}
}

// handleWebSocketMessage processes WebSocket messages
func (g *Gateway) handleWebSocketMessage(conn *websocket.Conn, msg *WebRTCMessage) error {
	switch msg.Type {
	case "call":
		return g.handleCall(conn, msg)
	case "answer":
		return g.handleAnswer(conn, msg)
	case "ice_candidate":
		return g.handleICECandidate(conn, msg)
	case "hangup":
		return g.handleHangup(conn, msg)
	case "stats_request":
		return g.handleStatsRequest(conn, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleCall initiates a WebRTC-to-SIP call
func (g *Gateway) handleCall(conn *websocket.Conn, msg *WebRTCMessage) error {
	callData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid call data")
	}

	callerURI, _ := callData["caller_uri"].(string)
	calleeURI, _ := callData["callee_uri"].(string)
	sdpOffer, _ := callData["sdp"].(string)

	if callerURI == "" || calleeURI == "" || sdpOffer == "" {
		return fmt.Errorf("missing required call parameters")
	}

	// Create WebRTC session
	session, err := g.createWebRTCSession(conn, callerURI, calleeURI)
	if err != nil {
		return fmt.Errorf("failed to create WebRTC session: %w", err)
	}

	// Set remote description (WebRTC offer)
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdpOffer}
	if err := session.PeerConnection.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := session.PeerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	if err := session.PeerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	session.LocalSDP = answer.SDP
	session.RemoteSDP = sdpOffer
	session.State = SessionStateConnecting

	// Initiate SIP call
	if err := g.initiateSIPCall(session); err != nil {
		return fmt.Errorf("failed to initiate SIP call: %w", err)
	}

	// Send response to WebRTC client
	response := WebRTCMessage{
		Type:      "call_response",
		SessionID: session.ID,
		Data: map[string]interface{}{
			"sdp":        answer.SDP,
			"session_id": session.ID,
			"status":     "connecting",
		},
	}

	if err := conn.WriteJSON(response); err != nil {
		return fmt.Errorf("failed to send call response: %w", err)
	}

	g.logger.Info("WebRTC call initiated",
		"session_id", session.ID,
		"caller", callerURI,
		"callee", calleeURI)

	// TODO: Add metrics collection

	return nil
}

// createWebRTCSession creates a new WebRTC session
func (g *Gateway) createWebRTCSession(conn *websocket.Conn, callerURI, calleeURI string) (*WebRTCSession, error) {
	sessionID := fmt.Sprintf("webrtc-%d", time.Now().UnixNano())

	// Create WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: g.buildICEServers(),
	}

	// Create peer connection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	session := &WebRTCSession{
		ID:             sessionID,
		WebRTCConn:     conn,
		PeerConnection: pc,
		CallerURI:      callerURI,
		CalleeURI:      calleeURI,
		State:          SessionStateInitial,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		MediaStats:     &MediaStats{UpdatedAt: time.Now()},
	}

	// Configure peer connection event handlers
	g.setupPeerConnectionHandlers(session)

	// Store session
	g.mu.Lock()
	g.sessions[sessionID] = session
	g.mu.Unlock()

	return session, nil
}

// setupPeerConnectionHandlers configures WebRTC peer connection event handlers
func (g *Gateway) setupPeerConnectionHandlers(session *WebRTCSession) {
	pc := session.PeerConnection

	// ICE candidate handler
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			session.ICEGathering = false
			return
		}

		msg := WebRTCMessage{
			Type:      "ice_candidate",
			SessionID: session.ID,
			Data: map[string]interface{}{
				"candidate": candidate.ToJSON(),
			},
		}

		if err := session.WebRTCConn.WriteJSON(msg); err != nil {
			g.logger.Error("Failed to send ICE candidate", "error", err)
		}
	})

	// Connection state change handler
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		g.logger.Info("WebRTC connection state changed",
			"session_id", session.ID,
			"state", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			session.State = SessionStateConnected
			g.logger.Info("WebRTC session connected", "session_id", session.ID)
		case webrtc.PeerConnectionStateDisconnected:
			session.State = SessionStateDisconnecting
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			session.State = SessionStateDisconnected
			g.terminateSession(session.ID)
		}

		session.LastActivity = time.Now()
	})

	// Track handler for incoming media
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		g.logger.Info("Received WebRTC track",
			"session_id", session.ID,
			"kind", track.Kind().String(),
			"codec", track.Codec().MimeType)

		// Start reading RTP packets and forward to SIP via rtpengine
		go g.handleWebRTCTrack(session, track)
	})

	// Data channel handler (for future features)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		g.logger.Info("Received data channel", "session_id", session.ID, "label", dc.Label())
	})
}

// buildICEServers creates ICE server configuration
func (g *Gateway) buildICEServers() []webrtc.ICEServer {
	var servers []webrtc.ICEServer

	// Add STUN servers
	for _, stun := range g.stunServers {
		servers = append(servers, webrtc.ICEServer{URLs: []string{stun}})
	}

	// Add TURN servers
	servers = append(servers, g.turnServers...)

	return servers
}

// initiateSIPCall creates a SIP call for the WebRTC session
func (g *Gateway) initiateSIPCall(session *WebRTCSession) error {
	// Generate Call-ID
	callID := fmt.Sprintf("webrtc-%s@%s", session.ID, "webrtc-gateway")
	session.CallID = callID

	// Convert WebRTC SDP to SIP-compatible SDP
	sipSDP, err := g.convertWebRTCToSIPSDP(session.RemoteSDP)
	if err != nil {
		return fmt.Errorf("failed to convert WebRTC SDP: %w", err)
	}

	// Create SIP session via session manager
	// This would integrate with the existing SIP infrastructure

	// TODO: This would need to interface with the existing SIP server
	// For now, we'll simulate the SIP call initiation
	g.logger.Info("Would initiate SIP call",
		"call_id", callID,
		"from", session.CallerURI,
		"to", session.CalleeURI,
		"sdp_length", len(sipSDP))

	session.SIPSessionID = fmt.Sprintf("sip-session-%s", callID)

	return nil
}

// convertWebRTCToSIPSDP converts WebRTC SDP to SIP-compatible format
func (g *Gateway) convertWebRTCToSIPSDP(webrtcSDP string) (string, error) {
	// This is a simplified conversion - in reality, you'd need more sophisticated
	// SDP manipulation to handle codec negotiation, media formats, etc.

	// For now, return the SDP as-is with minimal modifications
	// In a production system, you'd:
	// 1. Parse the SDP
	// 2. Convert WebRTC-specific attributes
	// 3. Ensure compatibility with SIP standards
	// 4. Handle codec preferences

	return webrtcSDP, nil
}

// handleWebRTCTrack processes incoming WebRTC media
func (g *Gateway) handleWebRTCTrack(session *WebRTCSession, track *webrtc.TrackRemote) {
	defer func() {
		if r := recover(); r != nil {
			g.logger.Error("Panic in WebRTC track handler", "error", r)
		}
	}()

	buffer := make([]byte, 1500)

	for {
		n, _, err := track.Read(buffer)
		if err != nil {
			g.logger.Error("Failed to read from WebRTC track", "error", err)
			break
		}

		// Update statistics
		session.MediaStats.PacketsReceived++
		session.MediaStats.BytesReceived += uint64(n)
		session.MediaStats.UpdatedAt = time.Now()

		// TODO: Forward RTP packets to SIP side via rtpengine
		// This would involve:
		// 1. Getting the rtpengine session for this call
		// 2. Forwarding the RTP packet to the appropriate endpoint
		// 3. Handling any necessary packet transformations
	}
}

// handleAnswer processes SDP answer from WebRTC client
func (g *Gateway) handleAnswer(conn *websocket.Conn, msg *WebRTCMessage) error {
	answerData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid answer data")
	}

	sessionID, _ := answerData["session_id"].(string)
	sdpAnswer, _ := answerData["sdp"].(string)

	if sessionID == "" || sdpAnswer == "" {
		return fmt.Errorf("missing session ID or SDP in answer")
	}

	g.mu.RLock()
	session, exists := g.sessions[sessionID]
	g.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Set remote description
	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdpAnswer}
	if err := session.PeerConnection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	session.RemoteSDP = sdpAnswer
	session.LastActivity = time.Now()

	g.logger.Info("WebRTC answer processed", "session_id", sessionID)
	return nil
}

// handleICECandidate processes ICE candidates from WebRTC client
func (g *Gateway) handleICECandidate(conn *websocket.Conn, msg *WebRTCMessage) error {
	candidateData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid ICE candidate data")
	}

	sessionID, _ := candidateData["session_id"].(string)
	candidate, _ := candidateData["candidate"].(map[string]interface{})

	if sessionID == "" {
		return fmt.Errorf("missing session ID in ICE candidate")
	}

	g.mu.RLock()
	session, exists := g.sessions[sessionID]
	g.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if candidate == nil {
		// End of candidates
		return nil
	}

	// Add ICE candidate
	iceCandidate := webrtc.ICECandidateInit{
		Candidate: candidate["candidate"].(string),
	}

	if sdpMid, ok := candidate["sdpMid"].(string); ok {
		iceCandidate.SDPMid = &sdpMid
	}

	if sdpMLineIndex, ok := candidate["sdpMLineIndex"].(float64); ok {
		index := uint16(sdpMLineIndex)
		iceCandidate.SDPMLineIndex = &index
	}

	if err := session.PeerConnection.AddICECandidate(iceCandidate); err != nil {
		return fmt.Errorf("failed to add ICE candidate: %w", err)
	}

	session.LastActivity = time.Now()
	return nil
}

// handleHangup terminates a WebRTC session
func (g *Gateway) handleHangup(conn *websocket.Conn, msg *WebRTCMessage) error {
	hangupData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid hangup data")
	}

	sessionID, _ := hangupData["session_id"].(string)
	if sessionID == "" {
		return fmt.Errorf("missing session ID in hangup")
	}

	return g.terminateSession(sessionID)
}

// handleStatsRequest returns session statistics
func (g *Gateway) handleStatsRequest(conn *websocket.Conn, msg *WebRTCMessage) error {
	statsData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid stats request data")
	}

	sessionID, _ := statsData["session_id"].(string)
	if sessionID == "" {
		return fmt.Errorf("missing session ID in stats request")
	}

	g.mu.RLock()
	session, exists := g.sessions[sessionID]
	g.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Get WebRTC statistics
	stats := session.PeerConnection.GetStats()

	response := WebRTCMessage{
		Type:      "stats_response",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"webrtc_stats": stats,
			"media_stats":  session.MediaStats,
			"session_info": map[string]interface{}{
				"state":         session.State.String(),
				"created_at":    session.CreatedAt,
				"last_activity": session.LastActivity,
			},
		},
	}

	return conn.WriteJSON(response)
}

// terminateSession terminates a WebRTC session and associated SIP call
func (g *Gateway) terminateSession(sessionID string) error {
	g.mu.Lock()
	session, exists := g.sessions[sessionID]
	if exists {
		delete(g.sessions, sessionID)
	}
	g.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.State = SessionStateDisconnected

	// Close WebRTC peer connection
	if session.PeerConnection != nil {
		if err := session.PeerConnection.Close(); err != nil {
			g.logger.Error("Failed to close peer connection", "error", err)
		}
	}

	// Terminate associated SIP session
	if session.SIPSessionID != "" {
		ctx := context.Background()
		if err := g.sessionMgr.TerminateSession(ctx, session.SIPSessionID); err != nil {
			g.logger.Error("Failed to terminate SIP session", "error", err, "sip_session_id", session.SIPSessionID)
		}
	}

	// Send hangup message to WebRTC client
	hangupMsg := WebRTCMessage{
		Type:      "hangup",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"reason": "session_terminated",
		},
	}

	if session.WebRTCConn != nil {
		if err := session.WebRTCConn.WriteJSON(hangupMsg); err != nil {
			g.logger.Error("Failed to send hangup message", "error", err)
		}
		session.WebRTCConn.Close()
	}

	g.logger.Info("WebRTC session terminated", "session_id", sessionID)

	// TODO: Add metrics collection

	return nil
}

// sendError sends an error message to WebRTC client
func (g *Gateway) sendError(conn *websocket.Conn, sessionID, errorMsg string) {
	errMsg := WebRTCMessage{
		Type:      "error",
		SessionID: sessionID,
		Error:     errorMsg,
	}

	if err := conn.WriteJSON(errMsg); err != nil {
		g.logger.Error("Failed to send error message", "error", err)
	}
}

// GetActiveSessions returns all active WebRTC sessions
func (g *Gateway) GetActiveSessions() map[string]*WebRTCSession {
	g.mu.RLock()
	defer g.mu.RUnlock()

	sessions := make(map[string]*WebRTCSession)
	for id, session := range g.sessions {
		sessions[id] = session
	}

	return sessions
}

// GetSessionByID returns a specific WebRTC session
func (g *Gateway) GetSessionByID(sessionID string) (*WebRTCSession, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	session, exists := g.sessions[sessionID]
	return session, exists
}

// Shutdown gracefully shuts down the WebRTC gateway
func (g *Gateway) Shutdown(ctx context.Context) error {
	g.logger.Info("Shutting down WebRTC gateway")

	// Terminate all active sessions
	g.mu.Lock()
	sessionIDs := make([]string, 0, len(g.sessions))
	for id := range g.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	g.mu.Unlock()

	for _, sessionID := range sessionIDs {
		if err := g.terminateSession(sessionID); err != nil {
			g.logger.Error("Failed to terminate session during shutdown", "error", err, "session_id", sessionID)
		}
	}

	g.logger.Info("WebRTC gateway shutdown complete")
	return nil
}
