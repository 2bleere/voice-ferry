#!/usr/bin/env python3
"""
Test RTPEngine with the CORRECT format from the manual:
Two parts separated by a single space: cookie + bencode dictionary
"""

import socket
import time
import json

RTPENGINE_HOST = "192.168.1.208"
RTPENGINE_PORT = 22222

def create_rtpengine_message(cookie, command_dict):
    """Create message in correct RTPEngine format: 'cookie d<bencode>e'"""
    
    # Convert command to JSON string
    command_json = json.dumps(command_dict, separators=(',', ':'))
    
    # Create bencode dictionary with just the command
    bencode_dict = f'd7:command{len(command_json)}:{command_json}e'
    
    # Format: cookie + space + bencode
    message = f'{cookie} {bencode_dict}'
    
    return message

def create_minimal_rtpengine_message(cookie, command_str):
    """Create minimal message with raw command string"""
    
    # Create bencode dictionary with raw command
    bencode_dict = f'd7:command{len(command_str)}:{command_str}e'
    
    # Format: cookie + space + bencode
    message = f'{cookie} {bencode_dict}'
    
    return message

def test_correct_rtpengine_format():
    """Test the correct RTPEngine NG protocol format"""
    
    print("=== Correct RTPEngine NG Protocol Format Test ===")
    print("Format: 'cookie d<bencode_dictionary>e'")
    print(f"Target: {RTPENGINE_HOST}:{RTPENGINE_PORT}")
    print()
    
    # Generate unique session cookie
    session_cookie = f"test_{int(time.time())}"
    
    test_cases = [
        # Test 1: Basic ping with JSON command
        ("JSON ping", create_rtpengine_message(session_cookie, {"command": "ping"})),
        
        # Test 2: Raw ping command
        ("Raw ping", create_minimal_rtpengine_message(session_cookie, "ping")),
        
        # Test 3: Query command
        ("Query", create_rtpengine_message(session_cookie, {"command": "query"})),
        
        # Test 4: Simple numeric cookie
        ("Numeric cookie", create_rtpengine_message("12345678", {"command": "ping"})),
        
        # Test 5: Very short cookie
        ("Short cookie", create_rtpengine_message("test", {"command": "ping"})),
        
        # Test 6: Offer command (more complex)
        ("Offer command", create_rtpengine_message(session_cookie, {
            "command": "offer",
            "call-id": "test-call",
            "sdp": "v=0"
        })),
    ]
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        for i, (description, message) in enumerate(test_cases, 1):
            print(f"{i}. {description}:")
            print(f"   Message: {message}")
            print(f"   Length: {len(message)} bytes")
            
            try:
                # Send the packet
                sent = sock.sendto(message.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
                print(f"   ✓ Sent {sent} bytes")
                
                # Try to receive a response
                try:
                    data, addr = sock.recvfrom(1024)
                    response = data.decode('utf-8', errors='ignore')
                    print(f"   ✓ RESPONSE: {response}")
                except socket.timeout:
                    print("   - No response (timeout - but this is expected)")
                
            except Exception as e:
                print(f"   ✗ Failed: {e}")
            
            print()
            time.sleep(1)
    
    except Exception as e:
        print(f"Socket error: {e}")
    finally:
        if 'sock' in locals():
            sock.close()

def show_format_comparison():
    """Show the format difference"""
    
    print("=== Format Comparison ===")
    
    cookie = "testcookie"
    command = {"command": "ping"}
    command_json = json.dumps(command, separators=(',', ':'))
    
    print("WRONG (what we were doing):")
    wrong_format = f'd6:cookie{len(cookie)}:{cookie}7:command{len(command_json)}:{command_json}e'
    print(f"  {wrong_format}")
    
    print("\nCORRECT (per manual):")
    bencode_dict = f'd7:command{len(command_json)}:{command_json}e'
    correct_format = f'{cookie} {bencode_dict}'
    print(f"  {correct_format}")
    
    print(f"\nStructure:")
    print(f"  Cookie: '{cookie}'")
    print(f"  Space: ' '")
    print(f"  Bencode: '{bencode_dict}'")

if __name__ == "__main__":
    show_format_comparison()
    print()
    
    print("Starting test in 3 seconds...")
    print("Monitor logs with: kubectl logs -n voice-ferry deployment/rtpengine -f")
    time.sleep(3)
    
    try:
        test_correct_rtpengine_format()
        
        print("\n=== Next Steps ===")
        print("Check RTPEngine logs for:")
        print("- Successful command processing (no 'no cookie' errors)")
        print("- Response messages")
        print("- 'duplicate' detection on retransmission")
        
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
