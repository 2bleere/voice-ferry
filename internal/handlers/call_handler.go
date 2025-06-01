package handlers

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/2bleere/voice-ferry/pkg/redis"
	"github.com/2bleere/voice-ferry/pkg/rtpengine"
	"github.com/2bleere/voice-ferry/pkg/sip"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// CallHandler implements the B2BUACallService
type CallHandler struct {
	v1.UnimplementedB2BUACallServiceServer
	sipServer   *sip.Server
	rtpEngine   *rtpengine.Client
	redisClient *redis.Client
	sessionMgr  *sip.SessionManager
}

// NewCallHandler creates a new call handler
func NewCallHandler(sipServer *sip.Server, rtpEngine *rtpengine.Client, redisClient *redis.Client, sessionMgr *sip.SessionManager) *CallHandler {
	return &CallHandler{
		sipServer:   sipServer,
		rtpEngine:   rtpEngine,
		redisClient: redisClient,
		sessionMgr:  sessionMgr,
	}
}

// InitiateCall initiates a new SIP call
func (h *CallHandler) InitiateCall(ctx context.Context, req *v1.InitiateCallRequest) (*v1.InitiateCallResponse, error) {
	log.Printf("Initiating call from %s to %s", req.FromUri, req.ToUri)

	// Extract username from the From URI for session limit checking
	username := extractUsernameFromURI(req.FromUri)
	if username == "" {
		log.Printf("Warning: Could not extract username from URI: %s", req.FromUri)
	}

	// Generate unique call and leg IDs
	callID := fmt.Sprintf("call-%d", generateCallID())
	legID := fmt.Sprintf("leg-%d", generateLegID())

	// Check session limits if Redis client is available and username is provided
	if h.redisClient != nil && username != "" {
		allowed, err := h.redisClient.CheckSessionLimit(ctx, username)
		if err != nil {
			log.Printf("Error checking session limit for user %s: %v", username, err)
			// Continue despite error to avoid blocking calls
		} else if !allowed {
			log.Printf("Session limit exceeded for user %s", username)
			return &v1.InitiateCallResponse{
				CallId:       "",
				LegId:        "",
				ResultingSdp: "",
				Status:       v1.CallStatus_CALL_STATUS_FAILED,
			}, fmt.Errorf("session limit exceeded for user %s", username)
		}
	}

	// Store session in Redis if available
	if h.redisClient != nil {
		// Store both call state and session data with user tracking
		callState := map[string]interface{}{
			"call_id":       callID,
			"leg_id":        legID,
			"from_uri":      req.FromUri,
			"to_uri":        req.ToUri,
			"username":      username,
			"status":        v1.CallStatus_CALL_STATUS_INITIATING.String(),
			"created_at":    time.Now().Unix(),
			"last_activity": time.Now().Unix(),
		}

		// Store call state
		err := h.redisClient.StoreCallState(ctx, callID, callState)
		if err != nil {
			log.Printf("Warning: Failed to store call state in Redis: %v", err)
			// Continue without Redis persistence
		} else {
			log.Printf("Stored call state for call %s in Redis", callID)
		}

		// Store session for user tracking (this is what enables session limits)
		if username != "" {
			sessionID := fmt.Sprintf("session-%s", callID)
			sessionData := map[string]interface{}{
				"session_id":    sessionID,
				"call_id":       callID,
				"username":      username,
				"from_uri":      req.FromUri,
				"to_uri":        req.ToUri,
				"status":        v1.CallStatus_CALL_STATUS_INITIATING.String(),
				"created_at":    time.Now().Unix(),
				"last_activity": time.Now().Unix(),
			}

			err = h.redisClient.StoreSession(ctx, sessionID, sessionData, 24*time.Hour)
			if err != nil {
				log.Printf("Warning: Failed to store session in Redis: %v", err)
			} else {
				log.Printf("Stored session %s for user %s in Redis", sessionID, username)
			}
		}
	}

	// TODO: Implement actual call initiation logic
	// This would involve:
	// 1. Creating outgoing INVITE
	// 2. Processing SDP through rtpengine
	// 3. Managing call state with session manager

	return &v1.InitiateCallResponse{
		CallId:       callID,
		LegId:        legID,
		ResultingSdp: req.InitialSdp, // TODO: Process through rtpengine
		Status:       v1.CallStatus_CALL_STATUS_INITIATING,
	}, nil
}

// TerminateCall terminates an active call
func (h *CallHandler) TerminateCall(ctx context.Context, req *v1.TerminateCallRequest) (*v1.TerminateCallResponse, error) {
	log.Printf("Terminating call %s", req.CallId)

	// Remove call state from Redis if available
	if h.redisClient != nil {
		// First, get call state to track username for session management
		var callState map[string]interface{}
		var username string
		err := h.redisClient.GetCallState(ctx, req.CallId, &callState)
		if err == nil {
			// Track session termination
			if un, ok := callState["username"].(string); ok && un != "" {
				username = un
				log.Printf("Terminating session for user %s (call %s)", username, req.CallId)
			}
		}

		// Delete call state
		err = h.redisClient.DeleteCallState(ctx, req.CallId)
		if err != nil {
			log.Printf("Warning: Failed to delete call state from Redis: %v", err)
		} else {
			log.Printf("Removed call state for call %s from Redis", req.CallId)
		}

		// Delete session data for session limit tracking
		if username != "" {
			sessionID := fmt.Sprintf("session-%s", req.CallId)
			err = h.redisClient.DeleteSession(ctx, sessionID)
			if err != nil {
				log.Printf("Warning: Failed to delete session from Redis: %v", err)
			} else {
				log.Printf("Removed session %s for user %s from Redis", sessionID, username)
			}
		}
	}

	// TODO: Implement actual call termination logic
	// This would involve:
	// 1. Sending BYE to both legs
	// 2. Cleaning up rtpengine session
	// 3. Removing dialog state

	return &v1.TerminateCallResponse{
		Success: true,
		Message: "Call terminated successfully",
	}, nil
}

// GetActiveCalls returns a stream of active calls
func (h *CallHandler) GetActiveCalls(req *v1.GetActiveCallsRequest, stream v1.B2BUACallService_GetActiveCallsServer) error {
	log.Printf("Getting active calls with filter: %s", req.Filter)

	// Get active calls from Redis if available
	if h.redisClient != nil {
		callIDs, err := h.redisClient.ListActiveCalls(stream.Context())
		if err != nil {
			log.Printf("Error listing active calls from Redis: %v", err)
			return fmt.Errorf("failed to list active calls: %w", err)
		}

		log.Printf("Found %d active calls in Redis", len(callIDs))

		// Stream each call back to the client
		for _, callID := range callIDs {
			var callState map[string]interface{}
			err := h.redisClient.GetCallState(stream.Context(), callID, &callState)
			if err != nil {
				log.Printf("Error getting call state for %s: %v", callID, err)
				continue
			}

			// Convert call state to ActiveCallInfo
			call := &v1.ActiveCallInfo{
				CallId:       callID,
				Status:       parseCallStatus(getStringValue(callState, "status")),
				StartTime:    timestamppb.New(parseUnixTime(getNumericValue(callState, "created_at"))),
				LastActivity: timestamppb.New(parseUnixTime(getNumericValue(callState, "last_activity"))),
			}

			// Set URIs if available
			if fromUri := getStringValue(callState, "from_uri"); fromUri != "" {
				call.FromUri = fromUri
			}
			if toUri := getStringValue(callState, "to_uri"); toUri != "" {
				call.ToUri = toUri
			}

			// Apply filter if specified
			if req.Filter != "" {
				if !matchesFilter(call, req.Filter) {
					continue
				}
			}

			// Send the call info
			if err := stream.Send(call); err != nil {
				log.Printf("Error sending call info: %v", err)
				return err
			}
		}

		log.Printf("Streamed %d active calls", len(callIDs))
	} else {
		log.Printf("Redis client not available, returning empty call list")
	}

	return nil
}

// GetCallDetails returns detailed information about a specific call
func (h *CallHandler) GetCallDetails(ctx context.Context, req *v1.GetCallDetailsRequest) (*v1.CallDetailsResponse, error) {
	log.Printf("Getting call details for %s", req.CallId)

	// TODO: Implement call details lookup
	// This would involve:
	// 1. Finding dialog by call ID
	// 2. Getting SIP message history
	// 3. Getting media statistics from rtpengine

	return &v1.CallDetailsResponse{
		CallInfo: &v1.ActiveCallInfo{
			CallId:       req.CallId,
			FromUri:      "sip:unknown@example.com",
			ToUri:        "sip:unknown@example.com",
			Status:       v1.CallStatus_CALL_STATUS_CONNECTED,
			StartTime:    timestamppb.Now(),
			LastActivity: timestamppb.Now(),
		},
		// TODO: Add SIP messages and media info
	}, nil
}

// generateCallID generates a unique call ID
func generateCallID() int64 {
	// Simple implementation - use timestamp
	// In production, use proper UUID generation
	return timestamppb.Now().GetSeconds()
}

// generateLegID generates a unique leg ID
func generateLegID() int64 {
	// Simple implementation - use timestamp
	// In production, use proper UUID generation
	return int64(timestamppb.Now().GetNanos())
}

// Helper functions

// extractUsernameFromURI extracts the username from a SIP URI
func extractUsernameFromURI(uri string) string {
	// Match sip:username@domain pattern
	re := regexp.MustCompile(`sip:([^@]+)@`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseCallStatus converts string status to CallStatus enum
func parseCallStatus(status string) v1.CallStatus {
	switch strings.ToLower(status) {
	case "call_status_initiating":
		return v1.CallStatus_CALL_STATUS_INITIATING
	case "call_status_ringing":
		return v1.CallStatus_CALL_STATUS_RINGING
	case "call_status_connected":
		return v1.CallStatus_CALL_STATUS_CONNECTED
	case "call_status_disconnecting":
		return v1.CallStatus_CALL_STATUS_DISCONNECTING
	case "call_status_terminated":
		return v1.CallStatus_CALL_STATUS_TERMINATED
	case "call_status_failed":
		return v1.CallStatus_CALL_STATUS_FAILED
	default:
		return v1.CallStatus_CALL_STATUS_UNSPECIFIED
	}
}

// parseUnixTime converts Unix timestamp to time.Time
func parseUnixTime(timestamp float64) time.Time {
	if timestamp == 0 {
		return time.Now()
	}
	return time.Unix(int64(timestamp), 0)
}

// getStringValue safely gets a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getNumericValue safely gets a numeric value from a map
func getNumericValue(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
		if num, ok := val.(int64); ok {
			return float64(num)
		}
		if num, ok := val.(int); ok {
			return float64(num)
		}
	}
	return 0
}

// matchesFilter checks if a call matches the given filter
func matchesFilter(call *v1.ActiveCallInfo, filter string) bool {
	// Simple filter implementation - match against call ID, from URI, or to URI
	filter = strings.ToLower(filter)
	return strings.Contains(strings.ToLower(call.CallId), filter) ||
		strings.Contains(strings.ToLower(call.FromUri), filter) ||
		strings.Contains(strings.ToLower(call.ToUri), filter)
}
