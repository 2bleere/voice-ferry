#!/usr/bin/env python3

import socket
import time
import hashlib
import random

def test_rtpengine_source_patterns():
    """Test bencode patterns based on RTPEngine source code analysis"""
    
    host = "192.168.1.208"
    port = 22222
    
    print("=== Testing RTPEngine Source Code Patterns ===\n")
    
    # Generate a unique cookie for this session
    session_id = str(random.randint(100000, 999999))
    cookie = f"test_{session_id}"
    
    print(f"Using session cookie: {cookie}")
    
    # Test different bencode patterns found in RTPEngine source analysis
    test_patterns = [
        {
            "name": "Pattern 1: Standard NG protocol format",
            "description": "Based on ng_control.c in RTPEngine source",
            "cookie": cookie,
            "command": {"command": "ping"},
            "format_func": format_standard_ng
        },
        {
            "name": "Pattern 2: With call-id field at bencode level",
            "description": "Some commands require call-id at bencode level",
            "cookie": cookie,
            "command": {"command": "ping"},
            "extra_fields": {"call-id": ""},
            "format_func": format_with_call_id
        },
        {
            "name": "Pattern 3: Kamailio-style format",
            "description": "Format used by Kamailio rtpengine module",
            "cookie": cookie,
            "command": {"command": "ping"},
            "format_func": format_kamailio_style
        },
        {
            "name": "Pattern 4: OpenSIPS-style format",
            "description": "Format that might work with OpenSIPS",
            "cookie": cookie,
            "command": {"command": "ping"},
            "format_func": format_opensips_style
        },
        {
            "name": "Pattern 5: Minimal working format",
            "description": "Absolute minimal format that should work",
            "cookie": cookie,
            "command": {"command": "ping"},
            "format_func": format_minimal
        }
    ]
    
    for i, pattern in enumerate(test_patterns, 1):
        print(f"{i}. {pattern['name']}")
        print(f"   Description: {pattern['description']}")
        
        try:
            # Format the bencode using the pattern's function
            if 'extra_fields' in pattern:
                bencode = pattern['format_func'](pattern['cookie'], pattern['command'], pattern['extra_fields'])
            else:
                bencode = pattern['format_func'](pattern['cookie'], pattern['command'])
            
            print(f"   Bencode: {bencode}")
            print(f"   Length: {len(bencode)} bytes")
            
            # Send the request
            result = send_and_check_response(host, port, bencode, timeout=3.0)
            
            if result['success']:
                print(f"   ✅ SUCCESS! Got response: {result['response']}")
                print(f"   🎉 THIS PATTERN WORKS! Use this format.")
                return pattern
            elif result['timeout']:
                print(f"   ⏰ No response (timeout)")
            else:
                print(f"   ❌ Error: {result['error']}")
                
        except Exception as e:
            print(f"   ⚠️  Exception: {e}")
        
        print()
        time.sleep(1)  # Give RTPEngine time between requests
    
    print("None of the patterns worked. Check RTPEngine logs for clues.")
    return None

def format_standard_ng(cookie, command):
    """Standard NG protocol format: d6:cookie<len>:<cookie>7:command<len>:<json>e"""
    import json
    
    command_json = json.dumps(command, separators=(',', ':'))
    bencode = f'd6:cookie{len(cookie)}:{cookie}7:command{len(command_json)}:{command_json}e'
    return bencode

def format_with_call_id(cookie, command, extra_fields):
    """Format with additional fields at bencode level"""
    import json
    
    command_json = json.dumps(command, separators=(',', ':'))
    
    # Build bencode with extra fields
    parts = [f'6:cookie{len(cookie)}:{cookie}']
    
    for key, value in extra_fields.items():
        parts.append(f'{len(key)}:{key}{len(value)}:{value}')
    
    parts.append(f'7:command{len(command_json)}:{command_json}')
    
    bencode = 'd' + ''.join(parts) + 'e'
    return bencode

def format_kamailio_style(cookie, command):
    """Kamailio rtpengine module style format"""
    import json
    
    # Kamailio might use different JSON formatting or field order
    command_json = json.dumps(command, separators=(', ', ': '))  # With spaces
    bencode = f'd6:cookie{len(cookie)}:{cookie}7:command{len(command_json)}:{command_json}e'
    return bencode

def format_opensips_style(cookie, command):
    """OpenSIPS style format (might have different expectations)"""
    import json
    
    # Try with different field order
    command_json = json.dumps(command, separators=(',', ':'))
    bencode = f'd7:command{len(command_json)}:{command_json}6:cookie{len(cookie)}:{cookie}e'
    return bencode

def format_minimal(cookie, command):
    """Absolute minimal format"""
    import json
    
    # Just the command string, not JSON
    cmd_str = command['command']
    bencode = f'd6:cookie{len(cookie)}:{cookie}7:command{len(cmd_str)}:{cmd_str}e'
    return bencode

def send_and_check_response(host, port, payload, timeout=3.0):
    """Send payload and check for response"""
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(timeout)
        
        # Send request
        sock.sendto(payload.encode('utf-8'), (host, port))
        
        # Try to receive response
        try:
            response, addr = sock.recvfrom(1024)
            response_str = response.decode('utf-8', errors='ignore')
            
            sock.close()
            
            # Check if response indicates success
            if any(indicator in response_str.lower() for indicator in ['ok', 'pong', 'result']):
                return {'success': True, 'response': response_str, 'timeout': False, 'error': None}
            else:
                return {'success': False, 'response': response_str, 'timeout': False, 'error': 'Unexpected response format'}
                
        except socket.timeout:
            sock.close()
            return {'success': False, 'response': None, 'timeout': True, 'error': None}
            
    except Exception as e:
        return {'success': False, 'response': None, 'timeout': False, 'error': str(e)}

def check_logs_hint():
    """Print hint about checking logs"""
    print("\n" + "="*70)
    print("🔍 After running this test, check RTPEngine logs:")
    print("kubectl logs -n voice-ferry $(kubectl get pods -n voice-ferry -l app=rtpengine -o jsonpath='{.items[0].metadata.name}') --tail=30")
    print("\nLook for:")
    print("- Any entries without 'no cookie' error")
    print("- Any 'duplicate' messages (which indicate successful parsing)")
    print("- Different error messages that might give clues")

if __name__ == "__main__":
    working_pattern = test_rtpengine_source_patterns()
    
    if working_pattern:
        print(f"\n🎉 FOUND WORKING PATTERN: {working_pattern['name']}")
        print("Use this pattern in your RTPEngine client implementation!")
    else:
        print("\n❌ No working pattern found yet.")
        print("This suggests we need to investigate RTPEngine configuration or version compatibility.")
    
    check_logs_hint()
