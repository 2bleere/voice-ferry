# SIP B2BUA Routing System Documentation

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Rule Structure](#rule-structure)
4. [Condition Matching](#condition-matching)
5. [Actions](#actions)
6. [Routing Engine](#routing-engine)
7. [Integration with SIP Server](#integration-with-sip-server)
8. [API Usage](#api-usage)
9. [Configuration Examples](#configuration-examples)
10. [Best Practices](#best-practices)
11. [Troubleshooting](#troubleshooting)

## Overview

The SIP B2BUA routing system is a powerful, flexible engine that determines how incoming SIP requests are processed and routed. It uses a rule-based approach where each rule contains conditions that must be matched and actions to be executed when a match occurs.

### Key Features
- **Priority-based routing**: Rules are evaluated in priority order (higher priority first)
- **Flexible matching**: Support for URI patterns, source IP filtering, header conditions, and time-based routing
- **Rich actions**: Route to next hop, manipulate headers, set RTPEngine flags, or reject calls
- **Real-time management**: Add, update, and delete rules via gRPC API
- **Regular expression support**: Use regex patterns for advanced URI matching

## Architecture

The routing system consists of several key components:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   SIP Server    │───▶│  Routing Engine  │───▶│  Routing Result │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Request Context │    │   Rule Storage   │    │    Actions      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Components

1. **Routing Engine** (`pkg/routing/engine.go`): Core logic for rule evaluation
2. **Rule Storage**: In-memory storage with thread-safe access
3. **Routing Result** (`pkg/routing/result.go`): Contains matched rule and actions
4. **gRPC Handler** (`internal/handlers/routing_handler.go`): API for rule management
5. **SIP Integration** (`pkg/sip/server.go`): Integration with SIP message processing

## Rule Structure

Each routing rule is defined using the following protobuf structure:

```protobuf
message RoutingRule {
  string id = 1;                    // Unique identifier
  string name = 2;                  // Human-readable name
  string description = 3;           // Optional description
  uint32 priority = 4;              // Priority (higher = evaluated first)
  bool enabled = 5;                 // Enable/disable rule
  RoutingConditions conditions = 6;  // Matching conditions
  RoutingActions actions = 7;       // Actions to execute
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}
```

### Rule Priority

Rules are evaluated in **descending priority order** (higher numbers first). This allows you to:
- Place specific rules before general ones
- Override default behavior for specific scenarios
- Implement fallback routing

**Example Priority Scheme:**
- 1000-1999: Emergency routing
- 800-999: Customer-specific rules
- 500-799: Geographic routing
- 100-499: Default routing
- 1-99: Fallback rules

## Condition Matching

Conditions determine when a rule should be applied. Multiple conditions can be specified, and **all conditions must match** for the rule to be triggered.

### Available Conditions

```protobuf
message RoutingConditions {
  UriCondition uri = 1;              // Match request URI
  repeated string source_ips = 2;    // Source IP addresses
  HeaderConditions headers = 3;      // SIP header conditions
  TimeCondition time = 4;            // Time-based routing
}
```

#### 1. URI Condition

Matches against the Request-URI of incoming SIP requests.

```protobuf
message UriCondition {
  string pattern = 1;        // Regex pattern or exact match
  bool use_regex = 2;        // Enable regex matching
  bool case_sensitive = 3;   // Case-sensitive matching
}
```

**Examples:**
- Exact match: `sip:1234@example.com`
- Regex pattern: `^sip:\+1[0-9]{10}@.*` (US phone numbers)
- Wildcard: `sip:.*@internal\.domain\.com`

#### 2. Source IP Filtering

Filter requests based on originating IP address.

```protobuf
repeated string source_ips = 2;  // CIDR notation supported
```

**Examples:**
- Single IP: `192.168.1.100`
- CIDR range: `10.0.0.0/8`
- Multiple IPs: `["192.168.1.0/24", "172.16.0.1"]`

#### 3. Header Conditions

Match against SIP headers using exact match or regex patterns.

```protobuf
message HeaderConditions {
  map<string, HeaderCondition> headers = 1;
}

message HeaderCondition {
  string value = 1;          // Expected header value
  bool use_regex = 2;        // Enable regex matching
  bool case_sensitive = 3;   // Case-sensitive matching
}
```

**Examples:**
- User-Agent matching: `headers["User-Agent"] = {value: "^MyPhone.*", use_regex: true}`
- Call-ID filtering: `headers["Call-ID"] = {value: "specific-call-id"}`

#### 4. Time-Based Conditions

Route calls based on time of day, day of week, or date ranges.

```protobuf
message TimeCondition {
  repeated string days_of_week = 1;  // MON, TUE, WED, THU, FRI, SAT, SUN
  string start_time = 2;             // HH:MM format
  string end_time = 3;               // HH:MM format
  string timezone = 4;               // IANA timezone (e.g., "America/New_York")
}
```

**Examples:**
- Business hours: `start_time: "09:00", end_time: "17:00", days_of_week: ["MON", "TUE", "WED", "THU", "FRI"]`
- Weekend routing: `days_of_week: ["SAT", "SUN"]`

## Actions

Actions define what happens when a rule matches. Multiple actions can be specified and are executed in order.

### Available Actions

```protobuf
message RoutingActions {
  NextHopAction next_hop = 1;          // Route to destination
  HeaderActions headers = 2;           // Manipulate headers
  RTPEngineActions rtpengine = 3;      // RTPEngine configuration
  RejectAction reject = 4;             // Reject the call
}
```

#### 1. Next Hop Routing

Defines where to route the call.

```protobuf
message NextHopAction {
  string destination_uri = 1;    // Target SIP URI
  string outbound_proxy = 2;     // Optional outbound proxy
  uint32 timeout = 3;            // Request timeout (seconds)
  TransportType transport = 4;   // UDP, TCP, TLS, WS, WSS
}
```

**Examples:**
- Direct routing: `destination_uri: "sip:1234@carrier.com"`
- Proxy routing: `outbound_proxy: "sip:proxy.provider.com:5060"`
- Secure routing: `transport: TLS`

#### 2. Header Manipulation

Add, modify, or remove SIP headers.

```protobuf
message HeaderActions {
  map<string, string> add = 1;     // Add headers
  map<string, string> set = 2;     // Set/modify headers
  repeated string remove = 3;      // Remove headers
}
```

**Examples:**
- Add custom header: `add["X-Customer-ID"] = "12345"`
- Modify From header: `set["From"] = "sip:anonymous@privacy.invalid"`
- Remove header: `remove = ["User-Agent"]`

#### 3. RTPEngine Actions

Configure RTPEngine behavior for media handling.

```protobuf
message RTPEngineActions {
  repeated string flags = 1;       // RTPEngine flags
  string set_id = 2;              // RTPEngine set identifier
  map<string, string> options = 3; // Additional options
}
```

**Common RTPEngine flags:**
- `"trust-address"`: Trust the source address
- `"SIP-source-address"`: Use SIP source address
- `"replace-origin"`: Replace origin in SDP
- `"replace-session-connection"`: Replace session connection
- `"ICE=remove"`: Remove ICE attributes
- `"DTLS=off"`: Disable DTLS

#### 4. Call Rejection

Reject the call with a specific SIP response code.

```protobuf
message RejectAction {
  uint32 code = 1;      // SIP response code
  string reason = 2;    // Reason phrase
}
```

**Common rejection codes:**
- 403: Forbidden
- 404: Not Found
- 486: Busy Here
- 503: Service Unavailable

## Routing Engine

The routing engine (`pkg/routing/engine.go`) is the core component that evaluates rules and returns routing decisions.

### Engine Interface

```go
type Engine interface {
    AddRule(rule *RoutingRule) error
    UpdateRule(rule *RoutingRule) error
    DeleteRule(id string) error
    GetRule(id string) (*RoutingRule, error)
    ListRules() ([]*RoutingRule, error)
    Route(ctx context.Context, req *RouteRequest) (*RouteResult, error)
}
```

### Route Request Context

The engine evaluates rules against the request context:

```go
type RouteRequest struct {
    RequestURI  string            // SIP Request-URI
    Method      string            // SIP method (INVITE, REGISTER, etc.)
    SourceIP    string            // Source IP address
    Headers     map[string]string // SIP headers
    Timestamp   time.Time         // Request timestamp
}
```

### Routing Algorithm

1. **Sort rules by priority** (descending order)
2. **Evaluate each rule** until a match is found
3. **Check all conditions** (URI, source IP, headers, time)
4. **Return first matching rule** with its actions
5. **Return no match** if no rules match

```go
func (e *engine) Route(ctx context.Context, req *RouteRequest) (*RouteResult, error) {
    // Sort rules by priority (highest first)
    rules := e.getSortedRules()
    
    for _, rule := range rules {
        if !rule.Enabled {
            continue
        }
        
        // Check all conditions
        if e.matchesConditions(rule.Conditions, req) {
            return &RouteResult{
                Rule:    rule,
                Actions: rule.Actions,
                Matched: true,
            }, nil
        }
    }
    
    return &RouteResult{Matched: false}, nil
}
```

## Integration with SIP Server

The routing engine integrates with the SIP server through the `applyRoutingRules` function in `pkg/sip/server.go`.

### Request Processing Flow

1. **SIP request arrives** at the server
2. **Extract routing context** (URI, headers, source IP)
3. **Call routing engine** with request context
4. **Apply routing actions** based on result
5. **Forward or reject** the request

```go
func (s *Server) applyRoutingRules(req sip.Request) (*RouteResult, error) {
    routeReq := &RouteRequest{
        RequestURI: req.RequestURI().String(),
        Method:     string(req.Method()),
        SourceIP:   req.Source(),
        Headers:    extractHeaders(req),
        Timestamp:  time.Now(),
    }
    
    return s.routingEngine.Route(context.Background(), routeReq)
}
```

### Action Execution

When a rule matches, the server executes the specified actions:

```go
func (s *Server) executeActions(req sip.Request, result *RouteResult) error {
    actions := result.Actions
    
    // Apply header modifications
    if actions.Headers != nil {
        s.applyHeaderActions(req, actions.Headers)
    }
    
    // Handle next hop routing
    if actions.NextHop != nil {
        return s.routeToNextHop(req, actions.NextHop)
    }
    
    // Handle call rejection
    if actions.Reject != nil {
        return s.rejectCall(req, actions.Reject)
    }
    
    return nil
}
```

## API Usage

The routing system provides a gRPC API for managing rules at runtime.

### gRPC Service Definition

```protobuf
service RoutingService {
  rpc CreateRule(CreateRuleRequest) returns (CreateRuleResponse);
  rpc GetRule(GetRuleRequest) returns (GetRuleResponse);
  rpc UpdateRule(UpdateRuleRequest) returns (UpdateRuleResponse);
  rpc DeleteRule(DeleteRuleRequest) returns (DeleteRuleResponse);
  rpc ListRules(ListRulesRequest) returns (ListRulesResponse);
  rpc TestRoute(TestRouteRequest) returns (TestRouteResponse);
}
```

### Example API Usage

#### Creating a Rule

```bash
# Using grpcurl
grpcurl -plaintext -d '{
  "rule": {
    "id": "emergency-routing",
    "name": "Emergency Numbers",
    "priority": 1000,
    "enabled": true,
    "conditions": {
      "uri": {
        "pattern": "^sip:(911|112)@.*",
        "use_regex": true
      }
    },
    "actions": {
      "next_hop": {
        "destination_uri": "sip:emergency@911.gov",
        "transport": "TLS"
      }
    }
  }
}' localhost:50051 b2bua.v1.RoutingService/CreateRule
```

#### Testing Routes

```bash
grpcurl -plaintext -d '{
  "request_uri": "sip:911@example.com",
  "method": "INVITE",
  "source_ip": "192.168.1.100",
  "headers": {
    "User-Agent": "MyPhone/1.0"
  }
}' localhost:50051 b2bua.v1.RoutingService/TestRoute
```

## Configuration Examples

### Basic Routing Rules

#### 1. Geographic Routing

Route calls based on area codes to regional carriers:

```yaml
# US East Coast (area codes 212, 646, 718, 917)
id: "us-east-coast"
name: "US East Coast Routing"
priority: 500
enabled: true
conditions:
  uri:
    pattern: "^sip:\\+1(212|646|718|917)[0-9]{7}@.*"
    use_regex: true
actions:
  next_hop:
    destination_uri: "sip:east.carrier.com"
    transport: "UDP"
```

#### 2. Time-Based Routing

Route to different destinations based on business hours:

```yaml
# Business hours routing
id: "business-hours"
name: "Business Hours Routing"
priority: 600
enabled: true
conditions:
  time:
    days_of_week: ["MON", "TUE", "WED", "THU", "FRI"]
    start_time: "09:00"
    end_time: "17:00"
    timezone: "America/New_York"
actions:
  next_hop:
    destination_uri: "sip:business@company.com"
```

#### 3. Customer-Specific Routing

Route calls from specific customers differently:

```yaml
# VIP customer routing
id: "vip-customer"
name: "VIP Customer Routing"
priority: 800
enabled: true
conditions:
  source_ips: ["203.0.113.0/24"]
  headers:
    headers:
      "X-Customer-Tier":
        value: "VIP"
        case_sensitive: true
actions:
  next_hop:
    destination_uri: "sip:vip.queue@callcenter.com"
    timeout: 60
  headers:
    add:
      "X-Priority": "high"
      "X-Queue": "vip"
```

#### 4. Security Filtering

Block calls from known bad actors:

```yaml
# Block malicious sources
id: "security-block"
name: "Security Block List"
priority: 1500
enabled: true
conditions:
  source_ips: ["192.0.2.0/24", "198.51.100.0/24"]
actions:
  reject:
    code: 403
    reason: "Forbidden"
```

#### 5. LRN and NPA-NXX Routing

Route calls based on Location Routing Number (LRN) or Number Plan Area/Exchange (NPA-NXX):

```yaml
# LRN-based routing for number portability
id: "lrn-routing"
name: "LRN-based Routing"
priority: 400
enabled: true
conditions:
  uri:
    pattern: "^sip:.*[;?]rn=([2-9][0-9]{9}).*@"
    use_regex: true
actions:
  next_hop:
    destination_uri: "sip:lrn-gateway.carrier.com"
  headers:
    add:
      "X-Route-Type": "lrn"
      "X-LRN-Source": "rn-parameter"

# NPA-NXX based routing fallback
id: "npa-nxx-routing"
name: "NPA-NXX Geographic Routing"
priority: 390
enabled: true
conditions:
  headers:
    headers:
      "To":
        value: "^.*sip:\\+?1?([2-9][0-9]{2})([2-9][0-9]{2})[0-9]{4}@.*"
        use_regex: true
actions:
  next_hop:
    destination_uri: "sip:npa-nxx-gateway.carrier.com"
  headers:
    add:
      "X-Route-Type": "npa-nxx"
      "X-Routing-Method": "geographic"
```

### Advanced Routing Scenarios

#### 1. Load Balancing

Distribute calls across multiple carriers:

```yaml
# Carrier A (60% traffic)
id: "carrier-a"
name: "Primary Carrier"
priority: 300
enabled: true
conditions:
  headers:
    headers:
      "X-Load-Balance":
        value: "^[0-5].*"
        use_regex: true
actions:
  next_hop:
    destination_uri: "sip:carriera.com"

# Carrier B (40% traffic)
id: "carrier-b"
name: "Secondary Carrier"
priority: 299
enabled: true
actions:
  next_hop:
    destination_uri: "sip:carrierb.com"
```

#### 2. Codec Transcoding

Route calls requiring transcoding to specific servers:

```yaml
# G.729 codec routing
id: "g729-transcoding"
name: "G.729 Transcoding"
priority: 700
enabled: true
conditions:
  headers:
    headers:
      "Content-Type":
        value: ".*G729.*"
        use_regex: true
actions:
  next_hop:
    destination_uri: "sip:transcoder.internal.com"
  rtpengine:
    flags: ["transcode-G729", "replace-origin"]
```

#### 3. Emergency Override

Override normal routing for emergency situations:

```yaml
# Emergency maintenance mode
id: "emergency-override"
name: "Emergency Maintenance"
priority: 2000
enabled: false  # Enable during maintenance
actions:
  reject:
    code: 503
    reason: "Service Temporarily Unavailable"
```

## Best Practices

### 1. Rule Organization

- **Use descriptive names** and IDs for rules
- **Add detailed descriptions** explaining the purpose
- **Group related rules** with similar priority ranges
- **Use consistent naming conventions**

### 2. Priority Management

- **Reserve high priorities** (1500+) for security and emergency rules
- **Use priority ranges** for different rule categories
- **Leave gaps** between priorities for future insertions
- **Document your priority scheme**

### 3. Condition Design

- **Be as specific as possible** to avoid unintended matches
- **Test regex patterns** thoroughly before deployment
- **Use case-insensitive matching** unless specifically needed
- **Combine multiple conditions** for precise targeting

### 4. Action Safety

- **Validate destination URIs** before creating rules
- **Use appropriate timeouts** for different scenarios
- **Test header modifications** to ensure compatibility
- **Monitor rejected calls** to verify security rules

### 5. Testing and Validation

- **Use the TestRoute API** to verify rule behavior
- **Test with real SIP messages** before production deployment
- **Monitor routing metrics** to detect issues
- **Keep backup copies** of working rule sets

### 6. Performance Considerations

- **Limit the number of rules** to maintain performance
- **Place frequently matched rules** at higher priorities
- **Optimize regex patterns** for speed
- **Regularly review and cleanup** unused rules

## Troubleshooting

### Common Issues

#### 1. Rule Not Matching

**Problem**: Rule exists but doesn't match expected requests.

**Debugging steps:**
1. Use the TestRoute API with actual request data
2. Check rule priority and enabled status
3. Verify condition syntax (especially regex patterns)
4. Check for case sensitivity issues
5. Validate source IP format (CIDR notation)

**Example debug test:**
```bash
grpcurl -plaintext -d '{
  "request_uri": "sip:1234@example.com",
  "method": "INVITE",
  "source_ip": "192.168.1.100"
}' localhost:50051 b2bua.v1.RoutingService/TestRoute
```

#### 2. Regex Pattern Issues

**Problem**: Regex patterns not matching as expected.

**Common mistakes:**
- Forgetting to escape special characters
- Using wrong anchors (^ and $)
- Case sensitivity issues
- Invalid regex syntax

**Testing approach:**
```bash
# Test regex patterns separately
echo "sip:+15551234567@example.com" | grep -E "^sip:\\+1[0-9]{10}@.*"
```

#### 3. Header Matching Problems

**Problem**: Header conditions not working correctly.

**Debugging steps:**
1. Check exact header names (case-sensitive)
2. Verify header values format
3. Ensure headers are present in the request
4. Test with simple exact matches first

#### 4. Time-Based Routing Issues

**Problem**: Time conditions not triggering correctly.

**Common causes:**
- Incorrect timezone specification
- Server time vs. condition timezone mismatch
- Invalid time format (must be HH:MM)
- Wrong day of week abbreviations

#### 5. Performance Issues

**Problem**: Routing engine responding slowly.

**Optimization strategies:**
- Reduce the total number of rules
- Optimize regex patterns
- Place most common rules at higher priorities
- Remove unused or redundant rules

### Logging and Monitoring

Enable debug logging to trace routing decisions:

```yaml
logging:
  level: debug
  include_source: true
```

Key metrics to monitor:
- Route evaluation time
- Rule match rates
- Rejected call counts
- Error rates by rule

### Emergency Procedures

#### Disable All Rules
```bash
# Get all rules and disable them
grpcurl -plaintext localhost:50051 b2bua.v1.RoutingService/ListRules | \
jq -r '.rules[].id' | \
xargs -I {} grpcurl -plaintext -d '{"id": "{}", "enabled": false}' \
localhost:50051 b2bua.v1.RoutingService/UpdateRule
```

#### Emergency Bypass Rule
Create a high-priority rule that routes all traffic to a backup destination:

```bash
grpcurl -plaintext -d '{
  "rule": {
    "id": "emergency-bypass",
    "name": "Emergency Bypass",
    "priority": 9999,
    "enabled": true,
    "actions": {
      "next_hop": {
        "destination_uri": "sip:backup.carrier.com"
      }
    }
  }
}' localhost:50051 b2bua.v1.RoutingService/CreateRule
```

## Conclusion

The SIP B2BUA routing system provides a powerful and flexible framework for call routing decisions. By understanding the rule structure, condition matching logic, and available actions, you can implement sophisticated routing policies that meet your specific requirements.

Key takeaways:
- Rules are evaluated by priority (highest first)
- All conditions in a rule must match for it to trigger
- Multiple actions can be executed when a rule matches
- The gRPC API allows real-time rule management
- Proper testing and monitoring are essential for reliable operation

For additional examples and use cases, refer to the `routing-examples.md` file in this documentation directory.
