#!/bin/bash

# Simple SIP test using echo and netcat
echo "Testing simple SIP REGISTER..."

# Create a proper SIP REGISTER message
SIP_MSG="REGISTER sip:sip-b2bua.local SIP/2.0\r
Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK-test1\r
Max-Forwards: 70\r
From: <sip:787@sip-b2bua.local>;tag=test1\r
To: <sip:787@sip-b2bua.local>\r
Contact: <sip:787@127.0.0.1:5060>\r
Call-ID: test1@127.0.0.1\r
CSeq: 1 REGISTER\r
Content-Length: 0\r
\r
"

echo "Sending SIP REGISTER..."
printf "%s" "$SIP_MSG" | nc -u 127.0.0.1 5060

echo "Done."
