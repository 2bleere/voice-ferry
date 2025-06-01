#!/bin/bash

# Simple test script to verify SIP authentication is working
SERVER_IP="127.0.0.1"
SERVER_PORT="5060"  # Port where container's 5060 is mapped

echo "Testing SIP authentication..."
echo "============================"
echo "Sending REGISTER request without authentication..."
cat test_auth.sip

# Use netcat to send the SIP message
if command -v nc &> /dev/null; then
    echo -e "\nSending REGISTER request..."
    echo -e "\nResponse:"
    echo -e "----------"
    # Add proper SIP message termination (double CRLF)
    (cat test_auth.sip; echo -e "\r\n"; sleep 2) | nc -u $SERVER_IP $SERVER_PORT
else
    echo "Netcat (nc) not found. Please install netcat to run this test."
    exit 1
fi

echo -e "\n\nChecking container logs for authentication challenge..."
docker-compose -f docker-compose.dev.yml logs --tail=20 b2bua | grep -E "401|Unauthorized|auth|challenge|WWW-Authenticate"
