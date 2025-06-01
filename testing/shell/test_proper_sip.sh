#!/bin/bash

# Proper SIP REGISTER test with correct CRLF line endings
echo "Testing SIP REGISTER with proper formatting..."

# Create temporary file with proper SIP message
cat > /tmp/register_test.sip << 'EOF'
REGISTER sip:sip-b2bua.local SIP/2.0
Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK-test1
Max-Forwards: 70
From: <sip:787@sip-b2bua.local>;tag=test1
To: <sip:787@sip-b2bua.local>
Contact: <sip:787@127.0.0.1:5060>
Call-ID: test1@127.0.0.1
CSeq: 1 REGISTER
Content-Length: 0

EOF

# Convert to proper CRLF line endings and add the extra CRLF at the end
echo "Converting to proper SIP format..."
sed 's/$/\r/' /tmp/register_test.sip > /tmp/register_test_crlf.sip
echo -e "\r" >> /tmp/register_test_crlf.sip

echo "Sending SIP REGISTER with proper CRLF endings..."
cat /tmp/register_test_crlf.sip | nc -u -w 3 127.0.0.1 5060

echo "Waiting for response..."
sleep 2

echo "Check server logs for authentication challenge..."
