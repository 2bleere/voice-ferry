#!/usr/bin/env python3
"""
Create and test PROPERLY formatted bencode for RTPEngine.
Focus on getting the bencode structure exactly right.
"""

import socket
import time
import json

RTPENGINE_HOST = "192.168.1.208"
RTPENGINE_PORT = 22222

def create_proper_bencode(cookie, command_dict):
    """Create properly formatted bencode string"""
    
    # Convert command to JSON string
    command_json = json.dumps(command_dict, separators=(',', ':'))
    
    # Build bencode dictionary format: d<key><value><key><value>e
    # Keys must be in lexicographical order in bencode
    
    parts = []
    parts.append('d')  # Start dictionary
    
    # Add 'command' key first (lexicographically before 'cookie')
    parts.append(f'7:command{len(command_json)}:{command_json}')
    
    # Add 'cookie' key
    parts.append(f'6:cookie{len(cookie)}:{cookie}')
    
    parts.append('e')  # End dictionary
    
    return ''.join(parts)

def create_minimal_bencode(cookie, command_str):
    """Create minimal bencode with raw command string"""
    
    parts = []
    parts.append('d')  # Start dictionary
    
    # Add 'command' key first
    parts.append(f'7:command{len(command_str)}:{command_str}')
    
    # Add 'cookie' key
    parts.append(f'6:cookie{len(cookie)}:{cookie}')
    
    parts.append('e')  # End dictionary
    
    return ''.join(parts)

def test_proper_bencode_formats():
    """Test properly structured bencode formats"""
    
    print("=== Proper Bencode Format Test ===")
    print(f"Target: {RTPENGINE_HOST}:{RTPENGINE_PORT}")
    print()
    
    # Generate unique session cookie
    session_cookie = f"test_{int(time.time())}"
    
    test_cases = [
        # Test 1: Minimal ping with JSON
        ("JSON ping command", create_proper_bencode(session_cookie, {"command": "ping"})),
        
        # Test 2: Raw string command
        ("Raw ping command", create_minimal_bencode(session_cookie, "ping")),
        
        # Test 3: Query command
        ("Query command", create_proper_bencode(session_cookie, {"command": "query"})),
        
        # Test 4: Different cookie format
        ("Numeric cookie", create_proper_bencode("12345678", {"command": "ping"})),
        
        # Test 5: Very simple cookie
        ("Simple cookie", create_proper_bencode("test", {"command": "ping"})),
    ]
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(3)
        
        for i, (description, bencode_data) in enumerate(test_cases, 1):
            print(f"{i}. {description}:")
            print(f"   Bencode: {bencode_data}")
            print(f"   Length: {len(bencode_data)} bytes")
            
            try:
                # Send the packet
                sent = sock.sendto(bencode_data.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
                print(f"   ✓ Sent {sent} bytes")
                
                # Try to receive a response (in case RTPEngine sends one)
                try:
                    data, addr = sock.recvfrom(1024)
                    print(f"   ✓ Received response: {data.decode('utf-8', errors='ignore')}")
                except socket.timeout:
                    print("   - No response (timeout)")
                
            except Exception as e:
                print(f"   ✗ Failed: {e}")
            
            print()
            time.sleep(1)
    
    except Exception as e:
        print(f"Socket error: {e}")
    finally:
        if 'sock' in locals():
            sock.close()

def validate_bencode_structure():
    """Simple bencode validation"""
    
    print("=== Bencode Validation ===")
    
    test_bencode = create_proper_bencode("testcookie", {"command": "ping"})
    print(f"Generated bencode: {test_bencode}")
    
    # Manual validation
    if test_bencode.startswith('d') and test_bencode.endswith('e'):
        print("✓ Proper dictionary format")
    else:
        print("✗ Invalid dictionary format")
    
    # Check for proper string lengths
    parts = test_bencode[1:-1]  # Remove 'd' and 'e'
    print(f"Dictionary content: {parts}")

if __name__ == "__main__":
    validate_bencode_structure()
    print()
    
    print("Starting test in 3 seconds...")
    print("Monitor logs with: kubectl logs -n voice-ferry deployment/rtpengine -f")
    time.sleep(3)
    
    try:
        test_proper_bencode_formats()
        
        print("\n=== Next Steps ===")
        print("Check RTPEngine logs for:")
        print("- 'duplicate' messages (SUCCESS)")
        print("- 'no cookie' errors (STILL FAILING)")
        print("- Any other response patterns")
        
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
