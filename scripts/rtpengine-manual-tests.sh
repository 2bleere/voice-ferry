#!/bin/bash

# Quick RTPEngine Health Test One-liners
# Save this file and reference these commands

echo "=== RTPEngine Manual Testing Commands ==="
echo ""

echo "1. Basic UDP connectivity test:"
echo "   echo 'test' | nc -u 192.168.1.208 22222"
echo ""

echo "2. RTPEngine ping health check:"
echo "   echo 'd7:command16:{\"command\":\"ping\"}e' | nc -u 192.168.1.208 22222"
echo ""

echo "3. With timeout (5 seconds):"
echo "   echo 'd7:command16:{\"command\":\"ping\"}e' | timeout 5 nc -u -w 5 192.168.1.208 22222"
echo ""

echo "4. Test local RTPEngine (if running locally):"
echo "   echo 'd7:command16:{\"command\":\"ping\"}e' | nc -u 127.0.0.1 22222"
echo ""

echo "5. Check if port is open (TCP test - won't work for health but tests connectivity):"
echo "   nc -z -v 192.168.1.208 22222"
echo ""

echo "6. Using socat (alternative to netcat):"
echo "   echo 'd7:command16:{\"command\":\"ping\"}e' | socat - UDP:192.168.1.208:22222"
echo ""

echo "Expected healthy response contains: '\"result\":\"ok\"'"
echo ""
echo "Note: Replace 192.168.1.208 with your actual RTPEngine host IP"
