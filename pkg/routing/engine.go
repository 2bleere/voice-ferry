package routing

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
	"github.com/emiago/sipgo/sip"
)

// Engine represents the routing engine
type Engine struct {
	rules map[string]*v1.RoutingRule
	mu    sync.RWMutex
}

// NewEngine creates a new routing engine
func NewEngine() *Engine {
	return &Engine{
		rules: make(map[string]*v1.RoutingRule),
	}
}

// AddRule adds a routing rule to the engine
func (e *Engine) AddRule(rule *v1.RoutingRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.RuleId] = rule
}

// UpdateRule updates an existing routing rule
func (e *Engine) UpdateRule(rule *v1.RoutingRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[rule.RuleId]; !exists {
		return fmt.Errorf("rule %s not found", rule.RuleId)
	}

	e.rules[rule.RuleId] = rule
	return nil
}

// RemoveRule removes a routing rule from the engine
func (e *Engine) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	delete(e.rules, ruleID)
	return nil
}

// GetRules returns all rules
func (e *Engine) GetRules() []*v1.RoutingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*v1.RoutingRule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	return rules
}

// SetRules replaces all rules
func (e *Engine) SetRules(rules []*v1.RoutingRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make(map[string]*v1.RoutingRule)
	for _, rule := range rules {
		e.rules[rule.RuleId] = rule
	}
}

// RouteRequest finds the best matching rule for a SIP request
func (e *Engine) RouteRequest(req *sip.Request, sourceIP string) (*RoutingResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Convert map to slice and sort by priority
	rules := make([]*v1.RoutingRule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}

	// Sort by priority (higher priority first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if e.matchesConditions(req, sourceIP, rule.Conditions) {
			return e.buildRoutingResult(rule), nil
		}
	}

	return nil, fmt.Errorf("no matching routing rule found")
}

// matchesConditions checks if a request matches the rule conditions
func (e *Engine) matchesConditions(req *sip.Request, sourceIP string, conditions *v1.RoutingConditions) bool {
	if conditions == nil {
		return true
	}

	// Check request URI regex
	if conditions.RequestUriRegex != "" {
		if !e.matchesRegex(req.Recipient.String(), conditions.RequestUriRegex) {
			return false
		}
	}

	// Check From URI regex
	if conditions.FromUriRegex != "" {
		fromHeader := req.From()
		if fromHeader == nil || !e.matchesRegex(fromHeader.Address.String(), conditions.FromUriRegex) {
			return false
		}
	}

	// Check To URI regex
	if conditions.ToUriRegex != "" {
		toHeader := req.To()
		if toHeader == nil || !e.matchesRegex(toHeader.Address.String(), conditions.ToUriRegex) {
			return false
		}
	}

	// Check source IP conditions
	if len(conditions.SourceIps) > 0 {
		if !e.matchesSourceIP(sourceIP, conditions.SourceIps) {
			return false
		}
	}

	// Check header conditions
	if len(conditions.HeaderConditions) > 0 {
		if !e.matchesHeaders(req, conditions.HeaderConditions) {
			return false
		}
	}

	// Check time conditions
	if conditions.TimeCondition != nil {
		if !e.matchesTimeCondition(conditions.TimeCondition) {
			return false
		}
	}

	return true
}

// matchesRegex checks if a string matches a regex pattern
func (e *Engine) matchesRegex(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(text)
}

// matchesSourceIP checks if source IP matches any of the allowed IPs/CIDRs
func (e *Engine) matchesSourceIP(sourceIP string, allowedIPs []string) bool {
	sourceIPAddr := net.ParseIP(sourceIP)
	if sourceIPAddr == nil {
		return false
	}

	for _, allowedIP := range allowedIPs {
		// Check if it's a CIDR
		if strings.Contains(allowedIP, "/") {
			_, cidr, err := net.ParseCIDR(allowedIP)
			if err != nil {
				continue
			}
			if cidr.Contains(sourceIPAddr) {
				return true
			}
		} else {
			// Direct IP match
			allowedIPAddr := net.ParseIP(allowedIP)
			if allowedIPAddr != nil && allowedIPAddr.Equal(sourceIPAddr) {
				return true
			}
		}
	}

	return false
}

// matchesHeaders checks if request headers match the conditions
func (e *Engine) matchesHeaders(req *sip.Request, headerConditions map[string]string) bool {
	for headerName, expectedValue := range headerConditions {
		headers := req.GetHeaders(headerName)
		if len(headers) == 0 {
			return false
		}

		// Check if any header value matches (using regex)
		found := false
		for _, header := range headers {
			if e.matchesRegex(header.Value(), expectedValue) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// matchesTimeCondition checks if current time matches the time condition
func (e *Engine) matchesTimeCondition(timeCondition *v1.TimeCondition) bool {
	now := time.Now()

	// Check day of week
	if len(timeCondition.DaysOfWeek) > 0 {
		currentDay := int32(now.Weekday())
		found := false
		for _, day := range timeCondition.DaysOfWeek {
			if day == currentDay {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if timeCondition.StartTime != "" && timeCondition.EndTime != "" {
		currentTime := now.Format("15:04")
		if !e.isTimeInRange(currentTime, timeCondition.StartTime, timeCondition.EndTime) {
			return false
		}
	}

	return true
}

// isTimeInRange checks if current time is within the specified range
func (e *Engine) isTimeInRange(currentTime, startTime, endTime string) bool {
	current, err1 := e.parseTime(currentTime)
	start, err2 := e.parseTime(startTime)
	end, err3 := e.parseTime(endTime)

	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}

	// Handle overnight time ranges (e.g., 22:00 to 06:00)
	if end <= start {
		return current >= start || current <= end
	}

	return current >= start && current <= end
}

// parseTime parses time in HH:MM format to minutes since midnight
func (e *Engine) parseTime(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	return hours*60 + minutes, nil
}

// buildRoutingResult creates a routing result from a matched rule
func (e *Engine) buildRoutingResult(rule *v1.RoutingRule) *RoutingResult {
	result := &RoutingResult{
		RuleID:      rule.RuleId,
		RuleName:    rule.Name,
		MatchedRule: rule,
	}

	if rule.Actions != nil {
		result.NextHopURI = rule.Actions.NextHopUri
		result.AddHeaders = rule.Actions.AddHeaders
		result.RemoveHeaders = rule.Actions.RemoveHeaders
		result.RtpengineFlags = rule.Actions.RtpengineFlags
		result.ResponseCode = rule.Actions.ResponseCode
		result.ResponseReason = rule.Actions.ResponseReason
	}

	return result
}
