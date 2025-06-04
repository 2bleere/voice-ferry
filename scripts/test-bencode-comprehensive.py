#!/usr/bin/env python3

import socket
import json
import time

def test_comprehensive_bencode_formats():
    """Comprehensive test of bencode formats based on analysis of RTPEngine source and common patterns"""
    
    host = "192.168.1.208"
    port = 22222
    
    print("=== Comprehensive RTPEngine Bencode Format Testing ===\n")
    
    # Test cases based on various bencode interpretations
    test_cases = [
        {
            "name": "Standard: Cookie as raw string",
            "payload": 'd6:cookie10:abc1234567:command18:{"command":"ping"}e',
            "description": "Cookie as raw binary string, command as JSON string"
        },
        {
            "name": "Alternative: Cookie with call-id in bencode",
            "payload": 'd6:cookie10:abc12345677:call-id4:test7:command18:{"command":"ping"}e',
            "description": "Both cookie and call-id at bencode level"
        },
        {
            "name": "Variation: Different field order",
            "payload": 'd7:command18:{"command":"ping"}6:cookie10:abc123456e',
            "description": "Command field first in bencode dictionary"
        },
        {
            "name": "Minimal: Only required fields",
            "payload": 'd6:cookie8:testcook7:command17:{"command":"ping"}e',
            "description": "Minimal required fields only"
        },
        {
            "name": "Binary cookie: Hex-decoded cookie",
            # Cookie 'test1234' as hex bytes: 74657374313233340a
            "payload": 'd6:cookie9:test12347:command17:{"command":"ping"}e',
            "description": "Short alphanumeric cookie"
        },
        {
            "name": "Empty call-id in JSON",
            "payload": 'd6:cookie8:testcook7:command36:{"command":"ping","call-id":""}e',
            "description": "Include empty call-id in JSON"
        },
        {
            "name": "Call-id in both places",
            "payload": 'd6:cookie8:testcook7:call-id4:test7:command36:{"command":"ping","call-id":"test"}e',
            "description": "Call-id in both bencode and JSON"
        },
        {
            "name": "Ultra-minimal ping",
            "payload": 'd6:cookie4:test7:command15:{"command":"p"}e',
            "description": "Shortest possible valid format"
        },
        {
            "name": "With transaction ID",
            "payload": 'd6:cookie12:test123456789:transact12:transaction017:command17:{"command":"ping"}e',
            "description": "Include transaction ID field"
        },
        {
            "name": "Full format with all fields",
            "payload": 'd6:cookie16:testcookie123457:call-id8:testcall8:from-tag7:fromtag6:to-tag5:totag7:command75:{"command":"ping","call-id":"testcall","from-tag":"fromtag","to-tag":"totag"}e',
            "description": "All possible fields included"
        }
    ]
    
    # Test each format
    for i, test in enumerate(test_cases, 1):
        print(f"{i}. {test['name']}")
        print(f"   Description: {test['description']}")
        print(f"   Payload: {test['payload']}")
        print(f"   Length: {len(test['payload'])} bytes")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(3.0)  # Longer timeout
            
            # Send
            sock.sendto(test['payload'].encode('utf-8'), (host, port))
            
            # Try to receive response
            try:
                response, addr = sock.recvfrom(1024)
                response_str = response.decode('utf-8', errors='ignore')
                print(f"   ✅ RESPONSE: {response_str}")
                
                # Check for success indicators
                if any(word in response_str.lower() for word in ['ok', 'pong', 'result']):
                    print(f"   🎉 SUCCESS! This format appears to work!")
                    return test  # Return the working format
                    
            except socket.timeout:
                print(f"   ⏰ No response (timeout)")
                
            sock.close()
            
        except Exception as e:
            print(f"   ❌ ERROR: {e}")
        
        print()
        time.sleep(0.5)  # Small delay between tests
    
    return None

def check_rtpengine_response():
    """Check what RTPEngine logs show"""
    print("="*70)
    print("🔍 Check RTPEngine logs after running this test:")
    print("kubectl logs -n voice-ferry $(kubectl get pods -n voice-ferry -l app=rtpengine -o jsonpath='{.items[0].metadata.name}') --tail=15")
    print()
    print("Look for:")
    print("- Any format that doesn't show 'no cookie' error")
    print("- Any actual response/acknowledgment from RTPEngine")
    print("- Different error messages that might give us clues")

def test_working_kamailio_format():
    """Test format known to work with Kamailio -> RTPEngine"""
    print("\n" + "="*50)
    print("Testing known working Kamailio format:")
    
    # This is based on Kamailio rtpengine module source
    cookie = "kamailio_test"
    command_json = '{"command": "ping"}'
    
    # Format: d{cookie_len}:cookie{cookie}{command_len}:command{json_len}:{json}e
    bencode = f'd6:cookie{len(cookie)}:{cookie}7:command{len(command_json)}:{command_json}e'
    
    print(f"Kamailio-style bencode: {bencode}")
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(3.0)
        
        sock.sendto(bencode.encode('utf-8'), ("192.168.1.208", 22222))
        
        try:
            response, addr = sock.recvfrom(1024)
            print(f"✅ RESPONSE: {response.decode('utf-8', errors='ignore')}")
        except socket.timeout:
            print("⏰ No response")
            
        sock.close()
        
    except Exception as e:
        print(f"❌ ERROR: {e}")

if __name__ == "__main__":
    working_format = test_comprehensive_bencode_formats()
    
    if not working_format:
        test_working_kamailio_format()
    
    check_rtpengine_response()
