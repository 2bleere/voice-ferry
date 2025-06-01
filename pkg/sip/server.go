package sip

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/2bleere/voice-ferry/pkg/auth"
	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/routing"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// Server represents the SIP server component
type Server struct {
	cfg           *config.Config
	ua            *sipgo.UserAgent
	srv           *sipgo.Server
	handlers      map[sip.RequestMethod]Handler
	dialogs       *DialogManager
	routingEngine *routing.Engine
	digestAuth    *auth.DigestAuth
	sessionMgr    *SessionManager
	mu            sync.RWMutex
}

// Handler defines the interface for SIP request handlers
type Handler interface {
	Handle(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error
}

// HandlerFunc is an adapter to allow ordinary functions to be used as Handlers
type HandlerFunc func(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error

// Handle calls the function
func (f HandlerFunc) Handle(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	return f(ctx, req, tx)
}

// NewServer creates a new SIP server
func NewServer(cfg *config.Config) (*Server, error) {
	// Create user agent
	ua, err := sipgo.NewUA()
	if err != nil {
		return nil, fmt.Errorf("failed to create user agent: %w", err)
	}

	// Create server
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	server := &Server{
		cfg:           cfg,
		ua:            ua,
		srv:           srv,
		handlers:      make(map[sip.RequestMethod]Handler),
		dialogs:       NewDialogManager(),
		routingEngine: routing.NewEngine(),
		digestAuth:    auth.NewDigestAuth(cfg),
	}

	// Register default handlers
	server.RegisterHandler(sip.INVITE, HandlerFunc(server.handleInvite))
	server.RegisterHandler(sip.ACK, HandlerFunc(server.handleACK))
	server.RegisterHandler(sip.BYE, HandlerFunc(server.handleBYE))
	server.RegisterHandler(sip.CANCEL, HandlerFunc(server.handleCANCEL))
	server.RegisterHandler(sip.REGISTER, HandlerFunc(server.handleREGISTER))
	server.RegisterHandler(sip.OPTIONS, HandlerFunc(server.handleOPTIONS))

	return server, nil
}

// RegisterHandler registers a handler for a specific SIP method
func (s *Server) RegisterHandler(method sip.RequestMethod, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// Start starts the SIP server
func (s *Server) Start(ctx context.Context) error {
	// Listen on configured address and port
	addr := fmt.Sprintf("%s:%d", s.cfg.SIP.Host, s.cfg.SIP.Port)

	var transport string
	switch s.cfg.SIP.Transport {
	case "UDP":
		transport = "udp"
	case "TCP":
		transport = "tcp"
	case "TLS":
		transport = "tls"
	default:
		transport = "udp"
	}

	log.Printf("Starting SIP server on %s (%s)", addr, transport)

	// Register handlers for each method
	for method, handler := range s.handlers {
		s.srv.OnRequest(method, func(req *sip.Request, tx sip.ServerTransaction) {
			s.handleRequest(ctx, req, tx, handler)
		})
	}

	// Start listening
	return s.srv.ListenAndServe(ctx, transport, addr)
}

// handleRequest routes incoming SIP requests to appropriate handlers
func (s *Server) handleRequest(ctx context.Context, req *sip.Request, tx sip.ServerTransaction, handler Handler) {
	if err := handler.Handle(ctx, req, tx); err != nil {
		log.Printf("Handler error for %s: %v", req.Method, err)
		res := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		tx.Respond(res)
	}
}

// handleInvite handles INVITE requests - core B2BUA logic
func (s *Server) handleInvite(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling INVITE from %s to %s", req.From().Address.String(), req.To().Address.String())

	// Check authentication and extract username
	username := ""

	// If authentication is enabled, extract username from From header or auth header
	if s.digestAuth.IsAuthenticationRequired() {
		// Try to get username from the From header URI
		fromURI := req.From().Address
		// Use the User field from the URI if available
		if fromURI.User != "" {
			username = fromURI.User
		}

		// If we have an auth header, use the authenticated username
		if authHeader := req.GetHeader("Authorization"); authHeader != nil {
			if u, ok := s.digestAuth.ExtractUsername(authHeader.Value()); ok && u != "" {
				username = u // Use authenticated username if available
			}
		}
	}

	// Create dialog for incoming leg (UAS)
	dialog := s.dialogs.CreateDialog(req, true)

	// Apply routing rules to determine next hop
	nextHop, err := s.applyRoutingRules(req)
	if err != nil {
		log.Printf("Routing failed: %v", err)
		res := sip.NewResponseFromRequest(req, 404, "Not Found", nil)
		tx.Respond(res)
		return err
	}

	// If we have session manager, check session limits and create session
	if s.sessionMgr != nil {
		// Check if the user has exceeded their session limit
		callID := req.CallID().Value()

		// Try to create session (will check limits if Redis is configured)
		_, err := s.sessionMgr.CreateSession(ctx, callID, dialog, username)
		if err != nil {
			log.Printf("Session limit error: %v", err)
			// Check if it's a session limit error
			if strings.Contains(err.Error(), "session limit exceeded") {
				// Return a 503 Service Unavailable for session limit error
				res := sip.NewResponseFromRequest(req, 503, "Maximum Sessions Exceeded", nil)
				tx.Respond(res)
				return fmt.Errorf("session limit exceeded for user %s", username)
			}
		}
	}

	// Create outgoing INVITE (UAC)
	outgoingReq := s.createOutgoingInvite(req, nextHop)

	// Send outgoing INVITE
	client, err := sipgo.NewClient(s.ua)
	if err != nil {
		res := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		tx.Respond(res)
		return err
	}

	clientTx, err := client.TransactionRequest(context.Background(), outgoingReq)
	if err != nil {
		res := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		tx.Respond(res)
		return err
	}

	// Handle responses from outgoing leg
	go s.handleOutgoingResponse(ctx, clientTx, tx, dialog, req)

	return nil
}

// handleACK handles ACK requests
func (s *Server) handleACK(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling ACK for call %s", req.CallID())

	dialog := s.dialogs.FindDialog(req.CallID().Value(), req.From().Address.String(), req.To().Address.String())
	if dialog == nil {
		log.Printf("No dialog found for ACK")
		return nil
	}

	// Forward ACK to outgoing leg
	return s.forwardRequest(req, dialog)
}

// handleBYE handles BYE requests
func (s *Server) handleBYE(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling BYE for call %s", req.CallID())

	dialog := s.dialogs.FindDialog(req.CallID().Value(), req.From().Address.String(), req.To().Address.String())
	if dialog == nil {
		res := sip.NewResponseFromRequest(req, 481, "Call/Transaction Does Not Exist", nil)
		if err := tx.Respond(res); err != nil {
			log.Printf("Failed to respond to transaction: %v", err)
		}
		return nil
	}

	// Forward BYE to other leg
	err := s.forwardRequest(req, dialog)
	if err != nil {
		res := sip.NewResponseFromRequest(req, 500, "Internal Server Error", nil)
		if err2 := tx.Respond(res); err2 != nil {
			log.Printf("Failed to respond to transaction: %v", err2)
		}
		return err
	}

	// Send 200 OK
	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(res); err != nil {
		log.Printf("Failed to respond to transaction: %v", err)
	}

	// Clean up dialog and media session
	s.dialogs.RemoveDialog(dialog.ID)
	// TODO: Send delete command to rtpengine

	return nil
}

// handleCANCEL handles CANCEL requests
func (s *Server) handleCANCEL(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling CANCEL for call %s", req.CallID())

	// Send 200 OK to CANCEL
	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(res); err != nil {
		log.Printf("Failed to respond to transaction: %v", err)
	}

	// TODO: Forward CANCEL to outgoing leg

	return nil
}

// handleREGISTER handles REGISTER requests
func (s *Server) handleREGISTER(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling REGISTER from %s", req.From().Address.String())

	// Check if authentication is required
	authRequired := s.digestAuth.IsAuthenticationRequired()
	log.Printf("DEBUG: Authentication required: %v", authRequired)

	if authRequired {
		// Extract authorization header
		authHeader := req.GetHeader("Authorization")
		log.Printf("DEBUG: Authorization header present: %v", authHeader != nil)

		if authHeader == nil {
			// No authorization header, send challenge
			log.Printf("No authorization header, sending challenge to %s", req.From().Address.String())
			res := sip.NewResponseFromRequest(req, 401, "Unauthorized", nil)

			// Add WWW-Authenticate header
			clientIP := s.extractSourceIP(req)
			challenge := s.digestAuth.CreateChallenge(clientIP)
			res.AppendHeader(challenge)

			if err := tx.Respond(res); err != nil {
				log.Printf("Failed to respond to transaction: %v", err)
			}
			return nil
		}

		// Validate credentials
		authValue := authHeader.Value()
		method := req.Method.String()
		uri := req.Recipient.String()

		valid, username := s.digestAuth.ValidateCredentials(authValue, method, uri)
		if !valid {
			log.Printf("Authentication failed for user %s from %s", username, req.From().Address.String())
			res := sip.NewResponseFromRequest(req, 403, "Forbidden", nil)
			if err := tx.Respond(res); err != nil {
				log.Printf("Failed to respond to transaction: %v", err)
			}
			return nil
		}

		log.Printf("Authentication successful for user %s from %s", username, req.From().Address.String())
	}

	// Authentication successful or not required
	// TODO: Store registration in Redis

	res := sip.NewResponseFromRequest(req, 200, "OK", nil)

	// Add Contact header with expires
	if req.Contact() != nil {
		res.AppendHeader(&sip.ContactHeader{
			Address: req.Contact().Address,
			Params:  sip.HeaderParams{"expires": "3600"},
		})
	}

	if err := tx.Respond(res); err != nil {
		log.Printf("Failed to respond to transaction: %v", err)
	}
	return nil
}

// handleOPTIONS handles OPTIONS requests
func (s *Server) handleOPTIONS(ctx context.Context, req *sip.Request, tx sip.ServerTransaction) error {
	log.Printf("Handling OPTIONS from %s", req.Source())

	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	// For now, just add the methods as a simple string header
	res.AppendHeader(sip.NewHeader("Allow", "INVITE, ACK, BYE, CANCEL, REGISTER, OPTIONS"))

	if err := tx.Respond(res); err != nil {
		log.Printf("Failed to respond to transaction: %v", err)
	}
	return nil
}

// applyRoutingRules applies configured routing rules to determine next hop
func (s *Server) applyRoutingRules(req *sip.Request) (string, error) {
	// Extract source IP from the request
	sourceIP := s.extractSourceIP(req)

	// Use routing engine to find matching rule
	result, err := s.routingEngine.RouteRequest(req, sourceIP)
	if err != nil {
		log.Printf("No routing rule matched for request from %s to %s: %v",
			req.From().Address.String(), req.To().Address.String(), err)
		// Return default route as fallback
		return "sip:127.0.0.1:5061", nil
	}

	log.Printf("Matched routing rule: %s (%s) for request %s -> %s",
		result.RuleID, result.RuleName, req.From().Address.String(), req.To().Address.String())

	// Check if call should be rejected
	if result.ShouldReject() {
		return "", fmt.Errorf("call rejected by rule %s: %d %s",
			result.RuleID, result.ResponseCode, result.ResponseReason)
	}

	// Return next hop URI
	if result.NextHopURI != "" {
		return result.NextHopURI, nil
	}

	return "", fmt.Errorf("no next hop defined in routing rule %s", result.RuleID)
}

// extractSourceIP extracts the source IP from a SIP request
func (s *Server) extractSourceIP(req *sip.Request) string {
	// Try to get from Via header first
	if via := req.Via(); via != nil {
		if host := via.Host; host != "" {
			return host
		}
	}

	// Fallback to connection info if available
	// This would need to be passed from the transport layer
	// For now, return localhost as placeholder
	return "127.0.0.1"
}

// createOutgoingInvite creates an outgoing INVITE based on incoming request
func (s *Server) createOutgoingInvite(incomingReq *sip.Request, nextHop string) *sip.Request {
	// Parse next hop URI
	var nextHopURI sip.Uri
	if err := sip.ParseUri(nextHop, &nextHopURI); err != nil {
		log.Printf("Failed to parse next hop URI %s: %v", nextHop, err)
		// Fallback to a basic URI
		nextHopURI.Host = "127.0.0.1"
		nextHopURI.Port = 5061
		nextHopURI.Scheme = "sip"
	}

	// Create new request
	outgoingReq := sip.NewRequest(sip.INVITE, nextHopURI)

	// Copy relevant headers
	outgoingReq.SetDestination(nextHop)

	// Copy From (but we might want to modify it)
	outgoingReq.AppendHeader(incomingReq.From())

	// Copy To
	outgoingReq.AppendHeader(incomingReq.To())

	// Generate new Call-ID for B2BUA leg (or reuse for transparency)
	outgoingReq.AppendHeader(incomingReq.CallID())

	// Copy body (SDP)
	if len(incomingReq.Body()) > 0 {
		// TODO: Process SDP through rtpengine
		outgoingReq.SetBody(incomingReq.Body())
		if contentType := incomingReq.ContentType(); contentType != nil {
			outgoingReq.AppendHeader(contentType)
		}
	}

	return outgoingReq
}

// forwardRequest forwards a request to the other call leg
func (s *Server) forwardRequest(req *sip.Request, dialog *Dialog) error {
	// TODO: Implement request forwarding logic
	log.Printf("Forwarding %s request for dialog %s", req.Method, dialog.ID)
	return nil
}

// handleOutgoingResponse handles responses from outgoing call leg
func (s *Server) handleOutgoingResponse(ctx context.Context, clientTx sip.ClientTransaction, serverTx sip.ServerTransaction, dialog *Dialog, originalReq *sip.Request) {
	for {
		select {
		case res := <-clientTx.Responses():
			// Forward response to incoming leg
			forwardedRes := s.createForwardedResponse(res, originalReq)

			if err := serverTx.Respond(forwardedRes); err != nil {
				log.Printf("Failed to forward response: %v", err)
				return
			}

			// If final response, we're done
			if res.StatusCode >= 200 {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// createForwardedResponse creates a response to forward to the other leg
func (s *Server) createForwardedResponse(originalRes *sip.Response, originalReq *sip.Request) *sip.Response {
	// Create new response based on original request
	forwardedRes := sip.NewResponseFromRequest(originalReq, int(originalRes.StatusCode), originalRes.Reason, nil)

	// Copy body if present (SDP answer)
	if len(originalRes.Body()) > 0 {
		// TODO: Process SDP through rtpengine
		forwardedRes.SetBody(originalRes.Body())
		if contentType := originalRes.ContentType(); contentType != nil {
			forwardedRes.AppendHeader(contentType)
		}
	}

	return forwardedRes
}

// SetSessionManager sets the session manager for the SIP server
func (s *Server) SetSessionManager(sessionMgr *SessionManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionMgr = sessionMgr
}

// GetDialogManager returns the dialog manager
func (s *Server) GetDialogManager() *DialogManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dialogs
}

// GetRoutingEngine returns the routing engine
func (s *Server) GetRoutingEngine() *routing.Engine {
	return s.routingEngine
}

// Close shuts down the SIP server
func (s *Server) Close() error {
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}
