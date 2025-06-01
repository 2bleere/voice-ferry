#!/bin/bash

# Test SIP authentication with the Go B2BUA server
# This script tests REGISTER requests with and without authentication

SERVER_IP="127.0.0.1"
SERVER_PORT="5060"  # Container port is exposed on host as 5060
TEST_USER="787"
TEST_PASSWORD="12345"
REALM="sip-b2bua.local"

echo "Testing SIP B2BUA Authentication..."
echo "=================================="

# Test 1: REGISTER without authentication (should get 401)
echo "Test 1: REGISTER without authentication (expecting 401 Unauthorized)"
echo "======================================================================"

# Create a basic REGISTER request
cat > /tmp/register_test1.sip << EOF
REGISTER sip:${REALM} SIP/2.0
Via: SIP/2.0/UDP ${SERVER_IP}:${SERVER_PORT};branch=z9hG4bK-test1
Max-Forwards: 70
From: <sip:${TEST_USER}@${REALM}>;tag=test1
To: <sip:${TEST_USER}@${REALM}>
Contact: <sip:${TEST_USER}@${SERVER_IP}:${SERVER_PORT}>
Call-ID: test1@${SERVER_IP}
CSeq: 1 REGISTER
Content-Length: 0

EOF

echo "Sending REGISTER without credentials:"
echo "-------------------------------------"
cat /tmp/register_test1.sip

# Send the SIP message using netcat (if available) or socat
if command -v nc &> /dev/null; then
    echo ""
    echo "Response:"
    echo "--------"
    nc -u -w 3 ${SERVER_IP} ${SERVER_PORT} < /tmp/register_test1.sip
elif command -v socat &> /dev/null; then
    echo ""
    echo "Response:"
    echo "--------"
    socat -T 3 - UDP4:${SERVER_IP}:${SERVER_PORT} < /tmp/register_test1.sip
else
    echo "Neither 'nc' nor 'socat' available for sending SIP messages"
    echo "Please install one of these tools to test SIP communication"
fi

echo ""
echo "=================================="
echo "Check Docker logs for authentication messages:"
echo "docker logs sip-b2bua"
