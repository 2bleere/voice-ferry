#!/usr/bin/env python3

import socket
import struct
import time

def test_alternative_theories():
    """Test alternative theories about why RTPEngine reports 'no cookie'"""
    
    host = "192.168.1.208"
    port = 22222
    timeout = 3.0
    
    print("=== Testing Alternative Theories for RTPEngine Cookie Issue ===\n")
    
    theories = [
        {
            "name": "Theory 1: Binary cookie expected",
            "description": "Maybe RTPEngine expects cookie as binary data, not string",
            "test_func": test_binary_cookie
        },
        {
            "name": "Theory 2: Case sensitivity issue", 
            "description": "Maybe field names are case sensitive",
            "test_func": test_case_sensitivity
        },
        {
            "name": "Theory 3: Protocol version issue",
            "description": "Maybe we need a version field or specific protocol marker",
            "test_func": test_protocol_version
        },
        {
            "name": "Theory 4: Encoding issue",
            "description": "Maybe RTPEngine expects different character encoding",
            "test_func": test_encoding_variations
        },
        {
            "name": "Theory 5: UDP packet structure",
            "description": "Maybe there's a specific UDP packet structure expected",
            "test_func": test_packet_structure
        }
    ]
    
    for theory in theories:
        print(f"🧪 {theory['name']}")
        print(f"   {theory['description']}")
        
        try:
            result = theory['test_func'](host, port, timeout)
            if result:
                print(f"   ✅ This theory might be correct!")
            else:
                print(f"   ❌ This theory doesn't seem to work")
        except Exception as e:
            print(f"   ⚠️  Error testing theory: {e}")
        
        print()
        time.sleep(1)

def test_binary_cookie(host, port, timeout):
    """Test if cookie should be binary data"""
    
    # Try cookie as binary representation
    test_cases = [
        # Cookie as 4-byte binary (32-bit integer)
        b'd6:cookie4:\x00\x01\x02\x037:command17:{"command":"ping"}e',
        
        # Cookie as 8-byte binary (64-bit)
        b'd6:cookie8:\x00\x01\x02\x03\x04\x05\x06\x077:command17:{"command":"ping"}e',
        
        # Cookie as null-terminated string
        b'd6:cookie5:test\x007:command17:{"command":"ping"}e',
    ]
    
    for i, payload in enumerate(test_cases, 1):
        print(f"     Binary test {i}: {payload}")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            sock.sendto(payload, (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"     → Response: {response}")
                return True
            except socket.timeout:
                print(f"     → No response")
                
            sock.close()
            
        except Exception as e:
            print(f"     → Error: {e}")
    
    return False

def test_case_sensitivity(host, port, timeout):
    """Test case sensitivity of field names"""
    
    test_cases = [
        'd6:Cookie4:test7:command17:{"command":"ping"}e',  # Capital C
        'd6:COOKIE4:test7:command17:{"command":"ping"}e',  # All caps
        'd6:cookie4:test7:Command17:{"command":"ping"}e',  # Capital C in command
        'd6:cookie4:test7:COMMAND17:{"command":"ping"}e',  # All caps command
    ]
    
    for i, payload in enumerate(test_cases, 1):
        print(f"     Case test {i}: {payload}")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            sock.sendto(payload.encode('utf-8'), (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"     → Response: {response.decode('utf-8', errors='ignore')}")
                return True
            except socket.timeout:
                print(f"     → No response")
                
            sock.close()
            
        except Exception as e:
            print(f"     → Error: {e}")
    
    return False

def test_protocol_version(host, port, timeout):
    """Test if protocol version or magic bytes are needed"""
    
    test_cases = [
        # With version field
        'd7:version1:17:command17:{"command":"ping"}6:cookie4:teste',
        
        # With protocol field
        'd8:protocol2:ng7:command17:{"command":"ping"}6:cookie4:teste',
        
        # Different bencode structure
        'd2:ng35:d6:cookie4:test7:command17:{"command":"ping"}ee',
    ]
    
    for i, payload in enumerate(test_cases, 1):
        print(f"     Version test {i}: {payload}")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            sock.sendto(payload.encode('utf-8'), (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"     → Response: {response.decode('utf-8', errors='ignore')}")
                return True
            except socket.timeout:
                print(f"     → No response")
                
            sock.close()
            
        except Exception as e:
            print(f"     → Error: {e}")
    
    return False

def test_encoding_variations(host, port, timeout):
    """Test different character encodings"""
    
    payload = 'd6:cookie4:test7:command17:{"command":"ping"}e'
    
    encodings = ['utf-8', 'ascii', 'latin1', 'utf-16']
    
    for encoding in encodings:
        print(f"     Encoding test {encoding}")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            encoded_payload = payload.encode(encoding)
            sock.sendto(encoded_payload, (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"     → Response: {response}")
                return True
            except socket.timeout:
                print(f"     → No response")
            except UnicodeDecodeError:
                print(f"     → Encoding not supported")
                
            sock.close()
            
        except Exception as e:
            print(f"     → Error: {e}")
    
    return False

def test_packet_structure(host, port, timeout):
    """Test if there's a specific packet structure or header needed"""
    
    base_payload = 'd6:cookie4:test7:command17:{"command":"ping"}e'
    
    # Test with different packet structures
    test_cases = [
        # With length prefix (4-byte big-endian)
        struct.pack('>I', len(base_payload)) + base_payload.encode('utf-8'),
        
        # With length prefix (4-byte little-endian) 
        struct.pack('<I', len(base_payload)) + base_payload.encode('utf-8'),
        
        # With 2-byte length prefix
        struct.pack('>H', len(base_payload)) + base_payload.encode('utf-8'),
        
        # With magic bytes
        b'RTPE' + base_payload.encode('utf-8'),
        
        # With null terminator
        base_payload.encode('utf-8') + b'\x00',
    ]
    
    for i, payload in enumerate(test_cases, 1):
        print(f"     Packet test {i}: {payload[:50]}...")
        
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(timeout)
            
            sock.sendto(payload, (host, port))
            
            try:
                response, addr = sock.recvfrom(1024)
                print(f"     → Response: {response}")
                return True
            except socket.timeout:
                print(f"     → No response")
                
            sock.close()
            
        except Exception as e:
            print(f"     → Error: {e}")
    
    return False

if __name__ == "__main__":
    test_alternative_theories()
    
    print("="*70)
    print("🔍 Check RTPEngine logs after this test:")
    print("kubectl logs -n voice-ferry $(kubectl get pods -n voice-ferry -l app=rtpengine -o jsonpath='{.items[0].metadata.name}') --tail=30")
