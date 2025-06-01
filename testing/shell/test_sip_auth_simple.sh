#!/bin/bash

# Test SIP authentication with the B2BUA server
# Simple script that sends a REGISTER without auth credentials

echo "Sending unauthenticated REGISTER to test authentication challenge..."

# Create a temporary REGISTER request file
cat > /tmp/register_test.sip << EOF
REGISTER sip:sip-b2bua.local SIP/2.0
Via: SIP/2.0/UDP 127.0.0.1:12345;branch=z9hG4bK-test-123
Max-Forwards: 70
From: <sip:787@sip-b2bua.local>;tag=test-tag
To: <sip:787@sip-b2bua.local>
Call-ID: auth-test-$(date +%s)@127.0.0.1
CSeq: 1 REGISTER
Contact: <sip:787@127.0.0.1:12345>
Expires: 3600
Content-Length: 0

EOF

# Try using netcat to send the SIP message
echo "Sending SIP REGISTER request..."

# Add verbose output
echo "Request being sent:"
echo "==================="
cat /tmp/register_test.sip
echo "==================="

# Send the request and capture the response
echo "Response from server:"
echo "===================="
nc -u -w 5 127.0.0.1 5060 < /tmp/register_test.sip
echo "===================="

# Add debug info
echo
echo "Checking server logs for authentication related messages:"
docker compose -f docker-compose.dev.yml logs --since 30s b2bua | grep -E "auth|AUTH|401|challenge|WWW-Authenticate|DigestAuth|IsAuthenticationRequired|SIP.Auth"
