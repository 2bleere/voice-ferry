#!/usr/bin/env python3

import socket
import json
import os
import random
import string

def generate_cookie(length=16):
    """Generate a random cookie string"""
    return ''.join(random.choices(string.ascii_lowercase + string.digits, k=length))

def create_bencode_command(cookie, json_data):
    """Create a bencode formatted command"""
    json_str = json.dumps(json_data, separators=(',', ':'))
    return f"d6:cookie{len(cookie)}:{cookie}7:command{len(json_str)}:{json_str}e"

def test_rtpengine_format(host, port, format_name, payload):
    """Test a specific bencode format against RTPEngine"""
    print(f"\n=== Testing {format_name} ===")
    print(f"Payload: {payload}")
    print(f"Length: {len(payload)} bytes")
    
    try:
        # Create UDP socket
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5.0)  # Longer timeout
        
        # Send the command
        bytes_sent = sock.sendto(payload.encode('utf-8'), (host, port))
        print(f"Sent {bytes_sent} bytes")
        
        # Receive response
        response, addr = sock.recvfrom(1024)
        response_str = response.decode('utf-8', errors='ignore')
        
        print(f"Response ({len(response)} bytes): {response_str}")
        
        # Check if successful (look for both "ok" result and absence of error)
        if 'result' in response_str:
            if 'ok' in response_str:
                print(f"✅ SUCCESS: {format_name} worked!")
                return True
            else:
                print(f"❌ FAILED: {format_name} returned non-ok result")
                return False
        else:
            # No result field - might be an error
            print(f"❌ FAILED: {format_name} - no result field in response")
            return False
            
    except socket.timeout:
        print(f"❌ TIMEOUT: {format_name} timed out (no response received)")
        return False
    except Exception as e:
        print(f"❌ ERROR: {format_name} failed with error: {e}")
        return False
    finally:
        if 'sock' in locals():
            sock.close()

def main():
    print("=== RTPEngine Bencode Format Testing ===")
    
    # RTPEngine connection details
    # Try different possible addresses
    test_addresses = [
        ("192.168.1.208", 22222),  # Direct pod IP
        ("10.43.56.159", 22222),   # Service IP from kubectl get svc
    ]
    
    # Different test formats to try
    test_formats = [
        {
            "name": "Current Implementation (Clean JSON)",
            "json": {"command": "ping", "call-id": ""},
        },
        {
            "name": "Minimal Ping",
            "json": {"command": "ping"},
        },
        {
            "name": "Ping with call-id only",
            "json": {"command": "ping", "call-id": "test-call-123"},
        },
        {
            "name": "Cookie in JSON (duplicate)",
            "json": {"command": "ping", "call-id": "", "cookie": "json-cookie-123"},
        },
    ]
    
    # Test each address
    for host, port in test_addresses:
        print(f"\n{'='*60}")
        print(f"Testing RTPEngine at {host}:{port}")
        print(f"{'='*60}")
        
        # Test connection first
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(1.0)
            sock.connect((host, port))
            sock.close()
            print(f"✅ Connection to {host}:{port} successful")
        except Exception as e:
            print(f"❌ Cannot connect to {host}:{port}: {e}")
            continue
        
        # Test each format
        success_count = 0
        for format_info in test_formats:
            cookie = generate_cookie()
            payload = create_bencode_command(cookie, format_info["json"])
            
            if test_rtpengine_format(host, port, format_info["name"], payload):
                success_count += 1
        
        print(f"\nResults for {host}:{port}: {success_count}/{len(test_formats)} formats succeeded")
        
        if success_count > 0:
            print(f"✅ Found working format(s) at {host}:{port}!")
            break
    
    print(f"\n{'='*60}")
    print("Testing complete!")

if __name__ == "__main__":
    main()
