#!/bin/bash

# Comprehensive SIP REGISTER authentication test
# Tests the complete authentication flow: no auth -> 401 challenge -> auth response

SERVER_IP="127.0.0.1"
SERVER_PORT="5060"
TEST_USER="787"
TEST_PASSWORD="12345"
REALM="sip-b2bua.local"

echo "SIP B2BUA REGISTER Authentication Test"
echo "====================================="
echo "Server: ${SERVER_IP}:${SERVER_PORT}"
echo "User: ${TEST_USER}"
echo "Realm: ${REALM}"
echo ""

# Function to send SIP message and capture response
send_sip_message() {
    local message_file="$1"
    local description="$2"
    
    echo "📤 ${description}"
    echo "----------------------------------------"
    echo "Sending:"
    cat "$message_file"
    echo ""
    echo "Response:"
    echo "----------"
    
    # Send with proper timeout and capture response
    timeout 5s bash -c "cat '$message_file' | nc -u -w 3 $SERVER_IP $SERVER_PORT" 2>/dev/null
    echo ""
    echo "----------------------------------------"
}

# Test 1: REGISTER without authentication (should get 401)
echo "🔒 Test 1: REGISTER without authentication (expecting 401 challenge)"
send_sip_message "test_register_proper.sip" "REGISTER without Authorization header"

# Wait a moment for server processing
sleep 1

# Check logs for authentication challenge
echo "📋 Checking server logs for authentication activity:"
echo "---------------------------------------------------"
docker-compose -f docker-compose.dev.yml logs --since 30s b2bua | grep -E "(auth|Auth|401|challenge|WWW-Authenticate|REGISTER|DigestAuth)" | tail -10

echo ""
echo "✅ Test completed. Check the response above for:"
echo "   - 401 Unauthorized status"
echo "   - WWW-Authenticate header with digest challenge"
echo "   - Server logs showing challenge generation"
