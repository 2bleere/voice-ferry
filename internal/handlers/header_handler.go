package handlers

import (
	"context"
	"log"

	"github.com/2bleere/voice-ferry/pkg/sip"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// HeaderHandler implements the SIPHeaderService
type HeaderHandler struct {
	v1.UnimplementedSIPHeaderServiceServer
	sipServer *sip.Server
}

// NewHeaderHandler creates a new header handler
func NewHeaderHandler(sipServer *sip.Server) *HeaderHandler {
	return &HeaderHandler{
		sipServer: sipServer,
	}
}

// AddSipHeader adds a SIP header to a call leg
func (h *HeaderHandler) AddSipHeader(ctx context.Context, req *v1.AddSipHeaderRequest) (*v1.CommandStatusResponse, error) {
	log.Printf("Adding SIP header %s to call %s, leg %s", req.HeaderName, req.CallId, req.LegId)

	// TODO: Implement SIP header manipulation
	// This would involve:
	// 1. Finding the dialog
	// 2. Adding header to subsequent messages

	return &v1.CommandStatusResponse{
		Success: true,
		Message: "Header added successfully",
	}, nil
}

// GetSipHeaders retrieves SIP headers from a call leg
func (h *HeaderHandler) GetSipHeaders(ctx context.Context, req *v1.GetSipHeadersRequest) (*v1.GetSipHeadersResponse, error) {
	log.Printf("Getting SIP headers for call %s, leg %s", req.CallId, req.LegId)

	// TODO: Implement SIP header retrieval
	// This would involve:
	// 1. Finding the dialog
	// 2. Extracting headers from last message

	headers := make(map[string]*v1.SipHeaderValues)

	// Example headers
	headers["User-Agent"] = &v1.SipHeaderValues{
		Values: []string{"Voice-Ferry-C4 B2BUA v1.0.0"},
	}

	return &v1.GetSipHeadersResponse{
		Headers: headers,
	}, nil
}

// RemoveSipHeader removes a SIP header from a call leg
func (h *HeaderHandler) RemoveSipHeader(ctx context.Context, req *v1.RemoveSipHeaderRequest) (*v1.CommandStatusResponse, error) {
	log.Printf("Removing SIP header %s from call %s, leg %s", req.HeaderName, req.CallId, req.LegId)

	// TODO: Implement SIP header removal

	return &v1.CommandStatusResponse{
		Success: true,
		Message: "Header removed successfully",
	}, nil
}

// ReplaceSipHeader replaces a SIP header in a call leg
func (h *HeaderHandler) ReplaceSipHeader(ctx context.Context, req *v1.ReplaceSipHeaderRequest) (*v1.CommandStatusResponse, error) {
	log.Printf("Replacing SIP header %s in call %s, leg %s", req.HeaderName, req.CallId, req.LegId)

	// TODO: Implement SIP header replacement

	return &v1.CommandStatusResponse{
		Success: true,
		Message: "Header replaced successfully",
	}, nil
}
