#!/usr/bin/env python3

import socket
import time

def simple_connection_test():
    """Very simple test to verify UDP connection works"""
    
    host = "192.168.1.208"
    port = 22222
    
    print(f"=== Simple Connection Test to {host}:{port} ===\n")
    
    # Test 1: Send something that should definitely show up in logs
    print("1. Sending simple test message...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(2.0)
        
        test_msg = "SIMPLE_TEST_MESSAGE_12345"
        sock.sendto(test_msg.encode('utf-8'), (host, port))
        print(f"   Sent: {test_msg}")
        
        try:
            response, addr = sock.recvfrom(1024)
            print(f"   Response: {response}")
        except socket.timeout:
            print("   No response (expected)")
            
        sock.close()
        
    except Exception as e:
        print(f"   Error: {e}")
    
    time.sleep(1)
    
    # Test 2: Send basic bencode that should be recognizable
    print("\n2. Sending basic bencode...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(2.0)
        
        test_bencode = "d4:test7:messag"  # Intentionally broken to see error
        sock.sendto(test_bencode.encode('utf-8'), (host, port))
        print(f"   Sent: {test_bencode}")
        
        try:
            response, addr = sock.recvfrom(1024)
            print(f"   Response: {response}")
        except socket.timeout:
            print("   No response")
            
        sock.close()
        
    except Exception as e:
        print(f"   Error: {e}")
    
    time.sleep(1)
    
    # Test 3: Send our standard bencode format
    print("\n3. Sending standard bencode format...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(2.0)
        
        test_bencode = 'd6:cookie8:simple127:command17:{"command":"ping"}e'
        sock.sendto(test_bencode.encode('utf-8'), (host, port))
        print(f"   Sent: {test_bencode}")
        
        try:
            response, addr = sock.recvfrom(1024)
            print(f"   Response: {response}")
        except socket.timeout:
            print("   No response")
            
        sock.close()
        
    except Exception as e:
        print(f"   Error: {e}")
    
    print("\n" + "="*50)
    print("Now check RTPEngine logs to see if these messages appeared:")
    print("kubectl logs -n voice-ferry $(kubectl get pods -n voice-ferry -l app=rtpengine -o jsonpath='{.items[0].metadata.name}') --tail=10")

if __name__ == "__main__":
    simple_connection_test()
