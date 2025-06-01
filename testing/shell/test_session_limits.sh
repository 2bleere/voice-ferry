#!/bin/bash

# Test script to verify user session limits functionality
# This script creates multiple concurrent SIP sessions for a single user
# to test the session limit enforcement

SERVER_IP="127.0.0.1"
SERVER_PORT="5060"
TEST_USER="787"
TEST_PASSWORD="12345"
REALM="sip-b2bua.local"

echo "=============================================="
echo "Testing SIP User Session Limits"
echo "=============================================="
echo "User: $TEST_USER"
echo "Limit: 3 sessions per user (from config)"
echo "Action: reject"

# Function to generate a random string for branch/call-id
generate_random() {
    local length=${1:-8}
    openssl rand -hex $((length/2)) 2>/dev/null || echo "$(date +%s)$(( RANDOM % 1000 ))"
}

# Function to send a SIP INVITE with authentication
send_invite() {
    local call_num=$1
    local branch="z9hG4bK-$(generate_random 8)"
    local call_id="test-session-limit-$(generate_random 8)@$SERVER_IP"
    local tag="tag-$(generate_random 8)"
    
    echo -e "\n📤 Sending INVITE #$call_num (Call-ID: $call_id)..."
    
    # Create SDP content
    local sdp="v=0\r\no=- 1234567890 1234567890 IN IP4 $SERVER_IP\r\ns=Session SDP\r\nc=IN IP4 $SERVER_IP\r\nt=0 0\r\nm=audio 49170 RTP/AVP 0 8 97\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:8 PCMA/8000\r\na=rtpmap:97 iLBC/8000\r\na=sendrecv\r\n"
    
    # Calculate content length (account for \r\n)
    local content_length=190
    
    # Create INVITE request with proper SIP formatting
    local sip_msg="INVITE sip:999@$REALM SIP/2.0\r\nVia: SIP/2.0/UDP $SERVER_IP:$SERVER_PORT;branch=$branch\r\nMax-Forwards: 70\r\nFrom: <sip:${TEST_USER}@${REALM}>;tag=$tag\r\nTo: <sip:999@${REALM}>\r\nContact: <sip:${TEST_USER}@$SERVER_IP:$SERVER_PORT>\r\nCall-ID: $call_id\r\nCSeq: 1 INVITE\r\nContent-Type: application/sdp\r\nContent-Length: $content_length\r\n\r\n$sdp"
    
    # Send the request and capture the response
    echo "Response (Call #$call_num):"
    echo "-----------------------"
    printf "%s" "$sip_msg" | timeout 3 nc -u $SERVER_IP $SERVER_PORT || echo "No response (timeout or connection issue)"
    sleep 1
}

# First let's create multiple auth sessions
echo "Creating multiple call sessions for the same user..."
for i in $(seq 1 5); do
    send_invite $i
    echo "Waiting for processing..."
    sleep 1
done

echo -e "\n✅ Check server logs for session limit enforcement..."
echo "Command to check logs: docker-compose -f docker-compose.dev.yml logs --tail=30 b2bua"

# Clean up temp files
rm -f /tmp/invite_*.sip
