package sip

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/2bleere/voice-ferry/pkg/redis"
	"github.com/2bleere/voice-ferry/pkg/rtpengine"
)

// CallSession represents a complete B2BUA call session with media
type CallSession struct {
	ID           string                 `json:"id"`
	CallID       string                 `json:"call_id"`
	CallerLeg    *DialogLeg             `json:"caller_leg"`
	CalleeLeg    *DialogLeg             `json:"callee_leg"`
	MediaSession *MediaSessionInfo      `json:"media_session"`
	State        CallState              `json:"state"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// DialogLeg represents one leg of a B2BUA call
type DialogLeg struct {
	DialogID   string    `json:"dialog_id"`
	LocalTag   string    `json:"local_tag"`
	RemoteTag  string    `json:"remote_tag"`
	LocalURI   string    `json:"local_uri"`
	RemoteURI  string    `json:"remote_uri"`
	ContactURI string    `json:"contact_uri"`
	RouteSet   []string  `json:"route_set"`
	State      string    `json:"state"`
	LastSDP    string    `json:"last_sdp"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MediaSessionInfo represents media session details
type MediaSessionInfo struct {
	RTPEngineInstanceID string                 `json:"rtpengine_instance_id"`
	OfferSDP            string                 `json:"offer_sdp"`
	AnswerSDP           string                 `json:"answer_sdp"`
	CallerMediaAddr     string                 `json:"caller_media_addr"`
	CalleeMediaAddr     string                 `json:"callee_media_addr"`
	Codecs              []string               `json:"codecs"`
	Statistics          map[string]interface{} `json:"statistics"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// CallState represents the overall state of a B2BUA call
type CallState int

const (
	CallStateInitial CallState = iota
	CallStateRinging
	CallStateConnected
	CallStateHolding
	CallStateTerminating
	CallStateTerminated
)

func (s CallState) String() string {
	switch s {
	case CallStateInitial:
		return "Initial"
	case CallStateRinging:
		return "Ringing"
	case CallStateConnected:
		return "Connected"
	case CallStateHolding:
		return "Holding"
	case CallStateTerminating:
		return "Terminating"
	case CallStateTerminated:
		return "Terminated"
	default:
		return "Unknown"
	}
}

// SessionManager manages call sessions with Redis persistence
type SessionManager struct {
	dialogManager *DialogManager
	redisClient   *redis.Client
	rtpengine     *rtpengine.Client
	logger        *slog.Logger
}

// NewSessionManager creates a new session manager
func NewSessionManager(dm *DialogManager, redisClient *redis.Client, rtpClient *rtpengine.Client, logger *slog.Logger) *SessionManager {
	return &SessionManager{
		dialogManager: dm,
		redisClient:   redisClient,
		rtpengine:     rtpClient,
		logger:        logger,
	}
}

// CreateSession creates a new call session
func (sm *SessionManager) CreateSession(ctx context.Context, callID string, callerDialog *Dialog, username string) (*CallSession, error) {
	sessionID := fmt.Sprintf("session-%s", callID)

	// Check session limits if Redis is available and username is provided
	if sm.redisClient != nil && username != "" {
		allowed, err := sm.redisClient.CheckSessionLimit(ctx, username)
		if err != nil {
			sm.logger.Error("Failed to check session limit", "error", err, "username", username)
			// Continue despite error to avoid blocking calls
		} else if !allowed {
			sm.logger.Info("Session limit exceeded for user", "username", username)
			return nil, fmt.Errorf("session limit exceeded for user %s", username)
		}
	}

	session := &CallSession{
		ID:     sessionID,
		CallID: callID,
		CallerLeg: &DialogLeg{
			DialogID:  callerDialog.ID,
			LocalTag:  callerDialog.LocalTag,
			RemoteTag: callerDialog.RemoteTag,
			LocalURI:  callerDialog.LocalURI,
			RemoteURI: callerDialog.RemoteURI,
			RouteSet:  callerDialog.RouteSet,
			State:     callerDialog.State.String(),
			UpdatedAt: time.Now(),
		},
		State:        CallStateInitial,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Store username if provided
	if username != "" {
		session.Metadata["username"] = username
	}

	// Store session in Redis
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to store session in Redis", "error", err, "session_id", sessionID)
			// Continue without Redis persistence if it fails
		}
	}

	return session, nil
}

// AddCalleeDialog adds the callee dialog to an existing session
func (sm *SessionManager) AddCalleeDialog(ctx context.Context, sessionID string, calleeDialog *Dialog) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.CalleeLeg = &DialogLeg{
		DialogID:  calleeDialog.ID,
		LocalTag:  calleeDialog.LocalTag,
		RemoteTag: calleeDialog.RemoteTag,
		LocalURI:  calleeDialog.LocalURI,
		RemoteURI: calleeDialog.RemoteURI,
		RouteSet:  calleeDialog.RouteSet,
		State:     calleeDialog.State.String(),
		UpdatedAt: time.Now(),
	}

	session.LastActivity = time.Now()

	// Link dialogs in memory
	sm.dialogManager.LinkDialogs(
		sm.dialogManager.FindDialogByID(session.CallerLeg.DialogID),
		sm.dialogManager.FindDialogByID(session.CalleeLeg.DialogID),
	)

	// Update session in Redis
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to update session in Redis", "error", err, "session_id", sessionID)
		}
	}

	return nil
}

// ProcessOffer processes an SDP offer through rtpengine
func (sm *SessionManager) ProcessOffer(ctx context.Context, sessionID, sdp string, flags []string) (string, error) {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	if sm.rtpengine == nil {
		sm.logger.Warn("RTPEngine not available, returning original SDP")
		return sdp, nil
	}

	// Create media session if not exists
	if session.MediaSession == nil {
		session.MediaSession = &MediaSessionInfo{
			CreatedAt: time.Now(),
		}
	}

	// Process SDP through rtpengine
	response, err := sm.rtpengine.Offer(ctx, session.CallID, session.CallerLeg.LocalTag, sdp, flags)
	if err != nil {
		sm.logger.Error("RTPEngine offer failed", "error", err, "call_id", session.CallID)
		return sdp, err // Return original SDP on error
	}

	// Update session with media information
	session.MediaSession.OfferSDP = sdp
	session.MediaSession.UpdatedAt = time.Now()
	session.CallerLeg.LastSDP = sdp
	session.CallerLeg.UpdatedAt = time.Now()
	session.LastActivity = time.Now()

	// Store updated session
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to update session after offer", "error", err, "session_id", sessionID)
		}
	}

	sm.logger.Info("Processed SDP offer through rtpengine",
		"session_id", sessionID,
		"call_id", session.CallID,
		"original_sdp_len", len(sdp),
		"processed_sdp_len", len(response.SDP))

	return response.SDP, nil
}

// ProcessAnswer processes an SDP answer through rtpengine
func (sm *SessionManager) ProcessAnswer(ctx context.Context, sessionID, sdp string, flags []string) (string, error) {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	if sm.rtpengine == nil {
		sm.logger.Warn("RTPEngine not available, returning original SDP")
		return sdp, nil
	}

	if session.MediaSession == nil {
		return "", fmt.Errorf("no media session found for answer processing")
	}

	var toTag string
	if session.CalleeLeg != nil {
		toTag = session.CalleeLeg.LocalTag
	}

	// Process SDP through rtpengine
	response, err := sm.rtpengine.Answer(ctx, session.CallID, session.CallerLeg.LocalTag, toTag, sdp, flags)
	if err != nil {
		sm.logger.Error("RTPEngine answer failed", "error", err, "call_id", session.CallID)
		return sdp, err
	}

	// Update session with answer information
	session.MediaSession.AnswerSDP = sdp
	session.MediaSession.UpdatedAt = time.Now()
	if session.CalleeLeg != nil {
		session.CalleeLeg.LastSDP = sdp
		session.CalleeLeg.UpdatedAt = time.Now()
	}
	session.LastActivity = time.Now()

	// Store updated session
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to update session after answer", "error", err, "session_id", sessionID)
		}
	}

	sm.logger.Info("Processed SDP answer through rtpengine",
		"session_id", sessionID,
		"call_id", session.CallID,
		"original_sdp_len", len(sdp),
		"processed_sdp_len", len(response.SDP))

	return response.SDP, nil
}

// UpdateSessionState updates the call session state
func (sm *SessionManager) UpdateSessionState(ctx context.Context, sessionID string, state CallState) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.State = state
	session.LastActivity = time.Now()

	// Store updated session
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to update session state", "error", err, "session_id", sessionID)
			return err
		}
	}

	sm.logger.Info("Updated session state", "session_id", sessionID, "state", state.String())
	return nil
}

// TerminateSession terminates a call session and cleans up media
func (sm *SessionManager) TerminateSession(ctx context.Context, sessionID string) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Clean up RTPEngine media session
	if sm.rtpengine != nil && session.MediaSession != nil {
		var toTag string
		if session.CalleeLeg != nil {
			toTag = session.CalleeLeg.LocalTag
		}

		_, err := sm.rtpengine.Delete(ctx, session.CallID, session.CallerLeg.LocalTag, toTag)
		if err != nil {
			sm.logger.Error("Failed to delete RTPEngine session", "error", err, "call_id", session.CallID)
		}
	}

	// Update session state
	session.State = CallStateTerminated
	session.LastActivity = time.Now()

	// Store final session state
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to store final session state", "error", err, "session_id", sessionID)
		}

		// Clean up session from active sessions after a delay (for debugging/auditing)
		go func() {
			time.Sleep(5 * time.Minute)
			sm.redisClient.DeleteSession(context.Background(), sessionID)
		}()
	}

	// Remove dialogs from dialog manager
	if session.CallerLeg != nil {
		sm.dialogManager.RemoveDialog(session.CallerLeg.DialogID)
	}
	if session.CalleeLeg != nil {
		sm.dialogManager.RemoveDialog(session.CalleeLeg.DialogID)
	}

	sm.logger.Info("Terminated session", "session_id", sessionID, "call_id", session.CallID)
	return nil
}

// GetSession retrieves a session from Redis or memory
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*CallSession, error) {
	if sm.redisClient != nil {
		sessionData, err := sm.redisClient.GetSession(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session CallSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

// GetSessionByCallID retrieves a session by Call-ID
func (sm *SessionManager) GetSessionByCallID(ctx context.Context, callID string) (*CallSession, error) {
	sessionID := fmt.Sprintf("session-%s", callID)
	return sm.GetSession(ctx, sessionID)
}

// storeSession stores a session in Redis
func (sm *SessionManager) storeSession(ctx context.Context, session *CallSession) error {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store with 24-hour TTL
	return sm.redisClient.StoreSessionString(ctx, session.ID, string(sessionData), 24*time.Hour)
}

// GetActiveSessions returns all active sessions
func (sm *SessionManager) GetActiveSessions(ctx context.Context) ([]*CallSession, error) {
	if sm.redisClient == nil {
		return nil, fmt.Errorf("Redis client not available")
	}

	sessionIDs, err := sm.redisClient.GetActiveSessionIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active session IDs: %w", err)
	}

	var sessions []*CallSession
	for _, sessionID := range sessionIDs {
		session, err := sm.GetSession(ctx, sessionID)
		if err == nil && session.State != CallStateTerminated {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// UpdateSessionMetadata updates session metadata
func (sm *SessionManager) UpdateSessionMetadata(ctx context.Context, sessionID string, key string, value interface{}) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}

	session.Metadata[key] = value
	session.LastActivity = time.Now()

	// Store updated session
	if sm.redisClient != nil {
		if err := sm.storeSession(ctx, session); err != nil {
			sm.logger.Error("Failed to update session metadata", "error", err, "session_id", sessionID)
			return err
		}
	}

	return nil
}

// GetSessionStatistics returns session statistics
func (sm *SessionManager) GetSessionStatistics(ctx context.Context) (map[string]interface{}, error) {
	sessions, err := sm.GetActiveSessions(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_active_sessions": len(sessions),
		"sessions_by_state":     make(map[string]int),
		"avg_session_duration":  0.0,
	}

	stateCount := make(map[string]int)
	var totalDuration time.Duration

	for _, session := range sessions {
		state := session.State.String()
		stateCount[state]++
		totalDuration += time.Since(session.CreatedAt)
	}

	stats["sessions_by_state"] = stateCount
	if len(sessions) > 0 {
		stats["avg_session_duration"] = totalDuration.Seconds() / float64(len(sessions))
	}

	return stats, nil
}
