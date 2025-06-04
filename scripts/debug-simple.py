#!/usr/bin/env python3
"""
Simple debug test for RTPEngine commands
"""

import socket
import time

def test_simple_ping():
    print("=== Simple RTPEngine Test ===")
    
    host = "192.168.1.208"
    port = 22222
    
    # Create the message: cookie + space + bencode
    cookie = "test123"
    command = "ping"
    bencode = f'd7:command{len(command)}:{command}e'
    message = f'{cookie} {bencode}'
    
    print(f"Host: {host}:{port}")
    print(f"Cookie: '{cookie}'")
    print(f"Command: '{command}'")
    print(f"Bencode: '{bencode}'")
    print(f"Full message: '{message}'")
    print(f"Message length: {len(message)} bytes")
    print()
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        print("Sending message...")
        sent = sock.sendto(message.encode('utf-8'), (host, port))
        print(f"✓ Sent {sent} bytes")
        
        print("Waiting for response...")
        try:
            data, addr = sock.recvfrom(1024)
            response = data.decode('utf-8', errors='ignore')
            print(f"✓ RESPONSE: '{response}'")
        except socket.timeout:
            print("- No response (timeout)")
        
        sock.close()
        
    except Exception as e:
        print(f"✗ Error: {e}")

if __name__ == "__main__":
    test_simple_ping()
    print("\nCheck logs with:")
    print("kubectl logs -n voice-ferry deployment/rtpengine --tail=5")
