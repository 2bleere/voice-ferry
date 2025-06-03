# RTPEngine Health Testing Guide

## Quick Testing Methods

### Method 1: Use the automated script (Recommended)
```bash
# Test your ARM deployment RTPEngine
./scripts/test-rtpengine-health.sh

# Test custom host/port
./scripts/test-rtpengine-health.sh 192.168.1.100 22222 10

# Test localhost
./scripts/test-rtpengine-health.sh 127.0.0.1
```

### Method 2: Python script (Detailed output)
```bash
# Test with default settings (your ARM RTPEngine)
./scripts/test-rtpengine-health.py

# Test custom host
./scripts/test-rtpengine-health.py --host 127.0.0.1 --port 22222 --timeout 5
```

### Method 3: Manual netcat commands
```bash
# Basic ping test (replace IP with your RTPEngine host)
echo 'd7:command16:{"command":"ping"}e' | nc -u 192.168.1.208 22222

# With timeout
echo 'd7:command16:{"command":"ping"}e' | timeout 5 nc -u -w 5 192.168.1.208 22222

# Test local RTPEngine
echo 'd7:command16:{"command":"ping"}e' | nc -u 127.0.0.1 22222
```

## Understanding the Protocol

**Protocol**: UDP (not TCP!)
**Port**: 22222 (ng protocol)
**Message Format**: Bencode-wrapped JSON

**Command Structure**:
- JSON: `{"command":"ping"}`
- Bencode wrapper: `d7:command16:{"command":"ping"}e`
- Expected response: Contains `"result":"ok"`

## What Each Method Tests

1. **Connectivity**: Can reach the host and port
2. **Protocol**: Can send/receive UDP packets on port 22222
3. **Health**: RTPEngine responds with `"result":"ok"`

## Troubleshooting

### Common Issues:
- **No response**: RTPEngine not running or firewall blocking UDP 22222
- **Connection refused**: Wrong port or service not listening
- **Timeout**: Network issues or RTPEngine overloaded
- **Wrong result**: RTPEngine running but unhealthy

### Expected Responses:
- **Healthy**: `{"result":"ok"}`
- **Unhealthy**: `{"result":"error","error-reason":"..."}`
- **No response**: Service down or network issue

## Your Current Setup

Based on your ARM deployment config:
- **Host**: 192.168.1.208
- **Port**: 22222
- **Protocol**: UDP
- **Service**: voice-ferry-rtpengine

Test with:
```bash
./scripts/test-rtpengine-health.sh 192.168.1.208 22222 5
```
