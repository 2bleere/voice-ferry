#!/usr/bin/env python3

import socket
import time

def test_basic_bencode_structures():
    """Test very basic bencode structures to understand parsing"""
    
    host = "192.168.1.208"
    port = 22222
    
    print("=== Testing Basic Bencode Structures ===\n")
    
    # Very basic tests to understand bencode parsing
    test_cases = [
        {
            "name": "Empty dictionary",
            "payload": "de",
            "description": "Simplest possible bencode dictionary"
        },
        {
            "name": "Single string field",
            "payload": "d4:test4:datae",
            "description": "Dictionary with one string field"
        },
        {
            "name": "Cookie only",
            "payload": "d6:cookie4:teste",
            "description": "Dictionary with just cookie field"
        },
        {
            "name": "Command only",
            "payload": "d7:command4:pinge",
            "description": "Dictionary with just command field"
        },
        {
            "name": "Both cookie and command - simple",
            "payload": "d6:cookie4:test7:command4:pinge",
            "description": "Both fields with simple string values"
        },
        {
            "name": "Cookie + JSON command (proper length)",
            "payload": 'd6:cookie4:test7:command17:{"command":"ping"}e',
            "description": "Cookie with properly encoded JSON command"
        },
        {
            "name": "Different cookie format",
            "payload": 'd6:cookie12:test_cookie_17:command17:{"command":"ping"}e',
            "description": "Longer cookie with underscores"
        },
        {
            "name": "Numeric cookie",
            "payload": 'd6:cookie8:123456787:command17:{"command":"ping"}e',
            "description": "Numeric cookie as string"
        }
    ]
    
    for i, test in enumerate(test_cases, 1):
        print(f"{i}. {test['name']}")
        print(f"   Description: {test['description']}")
        print(f"   Payload: {test['payload']}")
        print(f"   Length: {len(test['payload'])} bytes")
        
        # Validate bencode structure
        if not validate_bencode_structure(test['payload']):
            print(f"   ⚠️  WARNING: Invalid bencode structure detected!")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(2.0)
            
            # Send
            sock.sendto(test['payload'].encode('utf-8'), (host, port))
            
            # Try to receive response
            try:
                response, addr = sock.recvfrom(1024)
                response_str = response.decode('utf-8', errors='ignore')
                print(f"   ✅ RESPONSE: {response_str}")
                
                if 'ok' in response_str.lower():
                    print(f"   🎉 SUCCESS! This format works!")
                    
            except socket.timeout:
                print(f"   ⏰ No response (timeout)")
                
            sock.close()
            
        except Exception as e:
            print(f"   ❌ ERROR: {e}")
        
        print()
        time.sleep(0.3)

def validate_bencode_structure(bencode_str):
    """Basic validation of bencode structure"""
    if not bencode_str.startswith('d'):
        return False
    if not bencode_str.endswith('e'):
        return False
    
    # Check that string lengths match actual content
    i = 1  # Skip initial 'd'
    while i < len(bencode_str) - 1:  # Skip final 'e'
        if bencode_str[i].isdigit():
            # Find the key length
            colon_pos = bencode_str.find(':', i)
            if colon_pos == -1:
                return False
            
            key_len = int(bencode_str[i:colon_pos])
            key_start = colon_pos + 1
            key_end = key_start + key_len
            
            if key_end >= len(bencode_str):
                return False
            
            # Move to value
            i = key_end
            if i >= len(bencode_str) - 1:
                break
                
            # Check if value is a string (starts with digit) or nested structure
            if bencode_str[i].isdigit():
                # String value
                val_colon = bencode_str.find(':', i)
                if val_colon == -1:
                    return False
                val_len = int(bencode_str[i:val_colon])
                val_start = val_colon + 1
                val_end = val_start + val_len
                
                if val_end > len(bencode_str):
                    return False
                    
                i = val_end
            else:
                # For simplicity, assume other structures are valid
                break
        else:
            break
    
    return True

def test_rtpengine_specific_formats():
    """Test formats that might be specific to RTPEngine expectations"""
    
    print("\n" + "="*50)
    print("Testing RTPEngine-specific format expectations:")
    
    # Based on looking at error messages, try different approaches
    formats = [
        # Maybe RTPEngine expects a specific field order?
        "d7:command17:{\"command\":\"ping\"}6:cookie8:test1234e",
        
        # Maybe it needs specific cookie format?
        "d6:cookie32:00112233445566778899aabbccddeeff7:command17:{\"command\":\"ping\"}e",
        
        # Maybe call-id is required even if empty?
        "d6:cookie8:test12347:call-id0:7:command17:{\"command\":\"ping\"}e",
        
        # Maybe it needs transaction ID?
        "d6:cookie8:test12347:command17:{\"command\":\"ping\"}11:transaction8:trans123e",
    ]
    
    for i, fmt in enumerate(formats, 1):
        print(f"\nRTPEngine format {i}: {fmt}")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(2.0)
            
            sock.sendto(fmt.encode('utf-8'), (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"✅ RESPONSE: {response.decode('utf-8', errors='ignore')}")
            except socket.timeout:
                print("⏰ No response")
                
            sock.close()
            
        except Exception as e:
            print(f"❌ ERROR: {e}")

if __name__ == "__main__":
    test_basic_bencode_structures()
    test_rtpengine_specific_formats()
    
    print("\n" + "="*70)
    print("🔍 Check RTPEngine logs:")
    print("kubectl logs -n voice-ferry $(kubectl get pods -n voice-ferry -l app=rtpengine -o jsonpath='{.items[0].metadata.name}') --tail=20")
