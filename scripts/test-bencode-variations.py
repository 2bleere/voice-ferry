#!/usr/bin/env python3

import socket
import json

def test_bencode_formats():
    """Test different bencode formats to find the correct one"""
    
    host = "192.168.1.208"
    port = 22222
    
    # Test different bencode formats
    test_cases = [
        {
            "name": "Current format (what we're using)",
            "payload": 'd6:cookie16:testcookie12347:command18:{"command":"ping"}e'
        },
        {
            "name": "Cookie as string value (with quotes)",
            "payload": 'd6:cookie18:"testcookie1234"7:command18:{"command":"ping"}e'
        },
        {
            "name": "Different field order",
            "payload": 'd7:command18:{"command":"ping"}6:cookie16:testcookie1234e'
        },
        {
            "name": "Cookie as 'cookie' field in dict",
            "payload": 'd6:cookie16:testcookie1234e{"command":"ping"}'
        },
        {
            "name": "Minimal working example from docs",
            "payload": 'd6:cookie4:test7:command18:{"command":"ping"}e'
        },
        {
            "name": "With call-id as separate bencode field",
            "payload": 'd6:cookie16:testcookie12347:call-id0:7:command18:{"command":"ping"}e'
        },
        {
            "name": "Hex encoded cookie",
            "payload": 'd6:cookie32:74657374636f6f6b69653132333435363738393031327:command18:{"command":"ping"}e'
        }
    ]
    
    print("=== Testing Different Bencode Formats ===\n")
    
    for i, test in enumerate(test_cases, 1):
        print(f"{i}. {test['name']}")
        print(f"   Payload: {test['payload']}")
        print(f"   Length: {len(test['payload'])} bytes")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(2.0)
            
            # Send
            sock.sendto(test['payload'].encode('utf-8'), (host, port))
            
            # Try to receive (might timeout if no response)
            try:
                response, addr = sock.recvfrom(1024)
                response_str = response.decode('utf-8', errors='ignore')
                print(f"   ✅ RESPONSE: {response_str}")
                
                if 'ok' in response_str:
                    print(f"   🎉 SUCCESS! This format works!")
                
            except socket.timeout:
                print(f"   ⏰ No response (timeout)")
                
            sock.close()
            
        except Exception as e:
            print(f"   ❌ ERROR: {e}")
        
        print()

def check_rtpengine_logs():
    """Remind user to check RTPEngine logs"""
    print("="*60)
    print("After running this test, check RTPEngine logs with:")
    print("kubectl logs rtpengine-7f89d94875-pqkpc -n voice-ferry --tail=20")
    print("="*60)

if __name__ == "__main__":
    test_bencode_formats()
    check_rtpengine_logs()
