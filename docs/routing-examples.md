# Voice Ferry - Routing Rules Examples

This directory contains example routing rules for the Voice Ferry SIP B2BUA system.

## Example 1: Basic Number-based Routing

This rule routes calls to different gateways based on the destination number prefix:

```yaml
rule_id: "us_domestic"
priority: 100
name: "US Domestic Routing"
description: "Route US domestic calls to carrier gateway"
enabled: true
conditions:
  request_uri_regex: "^sip:\\+?1[2-9][0-9]{9}@"
  source_ips:
    - "192.168.1.0/24"
    - "10.0.0.0/8"
actions:
  next_hop_uri: "sip:gateway1.carrier.com:5060"
  add_headers:
    X-Route-Type: "domestic"
    X-Carrier: "primary"
  rtpengine_flags: "trust-address replace-origin replace-session-connection"
```

## Example 2: Time-based Routing

This rule provides different routing during business hours:

```yaml
rule_id: "business_hours"
priority: 200
name: "Business Hours Routing"
description: "Route to premium gateway during business hours"
enabled: true
conditions:
  request_uri_regex: "^sip:\\+?1[2-9][0-9]{9}@"
  time_condition:
    days_of_week: [1, 2, 3, 4, 5]  # Monday-Friday
    start_time: "08:00"
    end_time: "18:00"
actions:
  next_hop_uri: "sip:premium-gateway.carrier.com:5060"
  add_headers:
    X-Route-Type: "premium"
    X-Time-Based: "business-hours"
  rtpengine_flags: "trust-address replace-origin replace-session-connection ICE=force"
```

## Example 3: Header-based Routing

This rule routes based on custom SIP headers:

```yaml
rule_id: "premium_customer"
priority: 300
name: "Premium Customer Routing"
description: "Route premium customers to dedicated infrastructure"
enabled: true
conditions:
  header_conditions:
    X-Customer-Type: "premium"
    P-Access-Network-Info: ".*fiber.*"
actions:
  next_hop_uri: "sip:premium.internal.com:5060"
  add_headers:
    X-Route-Type: "premium-customer"
    X-QOS-Level: "high"
  remove_headers:
    - "X-Internal-Route"
  rtpengine_flags: "trust-address replace-origin replace-session-connection DTLS=passive"
```

## Example 4: Call Rejection Rule

This rule blocks calls from specific sources:

```yaml
rule_id: "block_spam"
priority: 500
name: "Spam Block"
description: "Block known spam sources"
enabled: true
conditions:
  from_uri_regex: "^sip:.*@(spam-domain\\.com|bad-actor\\.net)"
  source_ips:
    - "198.51.100.0/24"  # Known spam network
actions:
  response_code: 403
  response_reason: "Forbidden - Spam Source"
```

## Example 5: International Routing

This rule routes international calls with specific handling:

```yaml
rule_id: "international"
priority: 50
name: "International Routing"
description: "Route international calls to international gateway"
enabled: true
conditions:
  request_uri_regex: "^sip:\\+?(?!1)[1-9][0-9]{7,14}@"  # Non-US numbers
  source_ips:
    - "0.0.0.0/0"  # Allow from anywhere
actions:
  next_hop_uri: "sip:intl-gateway.carrier.com:5060"
  add_headers:
    X-Route-Type: "international"
    X-Billing-Plan: "premium"
    P-Charging-Vector: "icid-value=intl-$(call-id)"
  rtpengine_flags: "trust-address replace-origin replace-session-connection record-call=/var/recordings/"
```

## Example 6: Emergency Services

This rule ensures emergency calls are properly routed:

```yaml
rule_id: "emergency"
priority: 1000
name: "Emergency Services"
description: "Route emergency calls to E911 gateway"
enabled: true
conditions:
  request_uri_regex: "^sip:9(11|33)@"
actions:
  next_hop_uri: "sip:e911.psap.gov:5060"
  add_headers:
    X-Route-Type: "emergency"
    X-Priority: "critical"
    P-Emergency-Info: "$(remote-ip)"
  rtpengine_flags: "trust-address replace-origin replace-session-connection record-call=/var/emergency-recordings/"
```

## Example 7: LRN-based Routing

This rule routes calls based on Location Routing Number (LRN) from the `rn` field in the R-URI header or by NPA-NXX from the To header:

```yaml
rule_id: "lrn_routing"
priority: 150
name: "LRN-based Routing"
description: "Route calls based on LRN from rn parameter or NPA-NXX from To header"
enabled: true
conditions:
  # Match calls with rn parameter in R-URI (LRN routing)
  request_uri_regex: "^sip:.*[;?]rn=([2-9][0-9]{9}).*@"
actions:
  next_hop_uri: "sip:lrn-gateway.carrier.com:5060"
  add_headers:
    X-Route-Type: "lrn"
    X-LRN-Source: "rn-parameter"
    X-Original-Called: "$(request-uri)"
  rtpengine_flags: "trust-address replace-origin replace-session-connection"
---
rule_id: "npa_nxx_routing"
priority: 140
name: "NPA-NXX based Routing"
description: "Route calls based on NPA-NXX (area code + exchange) from To header"
enabled: true
conditions:
  # Match calls with specific NPA-NXX patterns in To header
  # This example routes calls to specific area code/exchange combinations
  to_header_regex: "^.*sip:\\+?1?([2-9][0-9]{2})([2-9][0-9]{2})[0-9]{4}@.*"
  # Match specific NPA-NXX ranges (e.g., 212-555, 646-123, 718-987)
  to_header_regex: "^.*sip:\\+?1?(212555|646123|718987)[0-9]{4}@.*"
actions:
  next_hop_uri: "sip:npa-nxx-gateway.carrier.com:5060"
  add_headers:
    X-Route-Type: "npa-nxx"
    X-NPA-NXX: "$(to-header-npa-nxx)"
    X-Routing-Method: "geographic"
  rtpengine_flags: "trust-address replace-origin replace-session-connection"
---
rule_id: "lrn_fallback_routing"
priority: 130
name: "LRN Fallback Routing"
description: "Advanced LRN routing with fallback to NPA-NXX if no rn parameter"
enabled: true
conditions:
  # Complex condition: Check for rn parameter OR extract NPA-NXX from To header
  request_uri_regex: "^sip:.*([;?]rn=([2-9][0-9]{9}).*@|.*@)"
  to_header_regex: "^.*sip:\\+?1?([2-9][0-9]{2})([2-9][0-9]{2})[0-9]{4}@.*"
actions:
  # Route to LRN-aware gateway that can handle both scenarios
  next_hop_uri: "sip:lrn-aware.carrier.com:5060"
  add_headers:
    X-Route-Type: "lrn-or-npa-nxx"
    X-LRN-Available: "$(has-rn-parameter)"
    X-Fallback-Method: "npa-nxx"
    P-Routing-Info: "LRN=$(rn-value);NPA=$(npa);NXX=$(nxx)"
  # Add custom routing information for downstream processing
  set_headers:
    Route: "<sip:lrn-processor.internal.com;lr>"
  rtpengine_flags: "trust-address replace-origin replace-session-connection"
```

### LRN Routing Explanation

**Location Routing Number (LRN) Routing:**
- LRN is used in number portability scenarios where a phone number has been ported to a different carrier
- The `rn` parameter in the R-URI contains the actual routing number for the current carrier
- Format: `sip:15551234567;rn=15559876543@example.com`
- The LRN (15559876543) indicates which switch/carrier currently serves the number

**NPA-NXX Routing:**
- NPA = Numbering Plan Area (area code, e.g., 212)
- NXX = Exchange code (first 3 digits after area code, e.g., 555)
- Together they identify a specific geographic or carrier-based routing destination
- Used when LRN information is not available or for geographic routing

**Use Cases:**
1. **Number Portability**: Route ported numbers using LRN data
2. **Geographic Routing**: Route based on NPA-NXX when LRN unavailable
3. **Carrier Selection**: Choose optimal carrier based on destination routing data
4. **Rate Optimization**: Apply different billing rates based on LRN vs. geographic routing

**Header Variables Available:**
- `$(rn-value)`: Extracted LRN from rn parameter
- `$(npa)`: Area code from To header
- `$(nxx)`: Exchange code from To header
- `$(has-rn-parameter)`: Boolean indicating if rn parameter exists
- `$(request-uri)`: Original Request-URI for logging
````markdown

## Loading Rules via gRPC API

You can load these rules using the gRPC API:

```bash
# Using grpcurl
grpcurl -plaintext -d '{
  "rule": {
    "rule_id": "us_domestic",
    "priority": 100,
    "name": "US Domestic Routing",
    "description": "Route US domestic calls to carrier gateway",
    "enabled": true,
    "conditions": {
      "request_uri_regex": "^sip:\\\\+?1[2-9][0-9]{9}@",
      "source_ips": ["192.168.1.0/24", "10.0.0.0/8"]
    },
    "actions": {
      "next_hop_uri": "sip:gateway1.carrier.com:5060",
      "add_headers": {
        "X-Route-Type": "domestic",
        "X-Carrier": "primary"
      },
      "rtpengine_flags": "trust-address replace-origin replace-session-connection"
    }
  }
}' localhost:50051 b2bua.v1.RoutingRuleService/AddRoutingRule
```

## Testing Rules

Test your routing rules with different SIP request scenarios:

```bash
# Test domestic number
sipp -sf scenarios/test_invite.xml -s +12125551234 192.168.1.100:5060

# Test international number  
sipp -sf scenarios/test_invite.xml -s +4412345678 192.168.1.100:5060

# Test emergency number
sipp -sf scenarios/test_invite.xml -s 911 192.168.1.100:5060
```

## Rule Priority Guidelines

- Emergency services: 1000+
- Security/blocking rules: 500-999
- Premium customer routing: 300-499
- Time-based routing: 200-299
- Geographic/carrier routing: 100-199
- Default/fallback routing: 1-99
