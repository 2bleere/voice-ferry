#!/usr/bin/env python3
"""
Test RTPEngine with known valid commands.
The logs show it's parsing our format correctly but rejecting the command content.
"""

import socket
import time

RTPENGINE_HOST = "192.168.1.208"
RTPENGINE_PORT = 22222

def create_rtpengine_message(cookie, command_str):
    """Create message in correct RTPEngine format: 'cookie d<bencode>e'"""
    
    # Create bencode dictionary with raw command (not JSON)
    bencode_dict = f'd7:command{len(command_str)}:{command_str}e'
    
    # Format: cookie + space + bencode
    message = f'{cookie} {bencode_dict}'
    
    return message

def test_valid_rtpengine_commands():
    """Test with known RTPEngine commands"""
    
    print("=== Testing Valid RTPEngine Commands ===")
    print("Based on RTPEngine documentation and common SIP media proxy commands")
    print(f"Target: {RTPENGINE_HOST}:{RTPENGINE_PORT}")
    print()
    
    session_cookie = f"test_{int(time.time())}"
    
    # Known RTPEngine commands (from documentation)
    test_commands = [
        ("ping", "ping"),
        ("version", "version"),
        ("list", "list"),
        ("query", "query"),
        ("offer", "offer"),
        ("answer", "answer"),
        ("delete", "delete"),
        ("statistics", "statistics"),
        ("start recording", "start recording"),
        ("stop recording", "stop recording"),
    ]
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(3)
        
        for i, (description, command) in enumerate(test_commands, 1):
            print(f"{i}. Testing '{command}' command:")
            
            message = create_rtpengine_message(session_cookie, command)
            print(f"   Message: {message}")
            
            try:
                sent = sock.sendto(message.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
                print(f"   ✓ Sent {sent} bytes")
                
                # Try to receive response
                try:
                    data, addr = sock.recvfrom(1024)
                    response = data.decode('utf-8', errors='ignore')
                    print(f"   ✓ RESPONSE: {response}")
                except socket.timeout:
                    print("   - No response")
                
            except Exception as e:
                print(f"   ✗ Send failed: {e}")
            
            print()
            time.sleep(1)
            
            # Special handling for commands that might need parameters
            if command in ["offer", "answer", "delete"]:
                print(f"   Note: '{command}' typically requires additional parameters")
    
    except Exception as e:
        print(f"Socket error: {e}")
    finally:
        if 'sock' in locals():
            sock.close()

def test_minimal_commands():
    """Test the most basic commands first"""
    
    print("\n=== Testing Minimal Commands ===")
    
    session_cookie = "test123"
    minimal_commands = ["ping", "list", "version"]
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        for command in minimal_commands:
            print(f"Testing minimal '{command}':")
            
            message = create_rtpengine_message(session_cookie, command)
            print(f"  {message}")
            
            sent = sock.sendto(message.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
            print(f"  ✓ Sent {sent} bytes")
            
            time.sleep(0.5)
    
    except Exception as e:
        print(f"Error: {e}")
    finally:
        if 'sock' in locals():
            sock.close()

if __name__ == "__main__":
    try:
        test_valid_rtpengine_commands()
        test_minimal_commands()
        
        print("\n=== Check Logs ===")
        print("kubectl logs -n voice-ferry deployment/rtpengine --tail=20")
        
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
