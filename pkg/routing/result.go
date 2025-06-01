package routing

import (
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// RoutingResult represents the result of a routing decision
type RoutingResult struct {
	// Rule identification
	RuleID      string
	RuleName    string
	MatchedRule *v1.RoutingRule

	// Routing actions
	NextHopURI     string
	AddHeaders     map[string]string
	RemoveHeaders  []string
	RtpengineFlags string

	// Response actions (for rejecting calls)
	ResponseCode   int32
	ResponseReason string
}

// ShouldReject returns true if this result indicates the call should be rejected
func (r *RoutingResult) ShouldReject() bool {
	return r.ResponseCode >= 400 && r.ResponseCode < 700
}

// IsRedirect returns true if this result indicates a redirect response
func (r *RoutingResult) IsRedirect() bool {
	return r.ResponseCode >= 300 && r.ResponseCode < 400
}

// ShouldRoute returns true if this result provides a next hop for routing
func (r *RoutingResult) ShouldRoute() bool {
	return r.NextHopURI != "" && !r.ShouldReject()
}
