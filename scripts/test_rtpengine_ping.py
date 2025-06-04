#!/usr/bin/env python3
"""
Quick test to verify RTPEngine ping is working from within the cluster
"""
import socket
import time

def test_rtpengine_ping():
    # Test direct ping to RTPEngine from cluster perspective
    rtpengine_host = "rtpengine-service.voice-ferry.svc.cluster.local"
    rtpengine_port = 22222
    
    print(f"Testing RTPEngine ping to {rtpengine_host}:{rtpengine_port}")
    
    try:
        # Create UDP socket
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5.0)
        
        # Generate a test cookie
        cookie = "test123"
        
        # Create RTPEngine NG protocol ping command
        # Format: cookie + space + bencode dictionary
        message = f"{cookie} d7:command4:pinge"
        
        print(f"Sending: {message}")
        
        # Send ping
        sock.sendto(message.encode(), (rtpengine_host, rtpengine_port))
        
        # Receive response
        response, addr = sock.recvfrom(1024)
        response_str = response.decode()
        
        print(f"Received: {response_str}")
        
        # Check if response starts with our cookie
        if response_str.startswith(cookie + " "):
            print("✅ Cookie matches")
            bencode_part = response_str[len(cookie) + 1:]
            print(f"Bencode part: {bencode_part}")
            
            if "pong" in bencode_part:
                print("✅ RTPEngine responded with pong - connection working!")
                return True
        
        print("❌ Unexpected response format")
        return False
        
    except socket.timeout:
        print("❌ Timeout - RTPEngine not responding")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
        return False
    finally:
        sock.close()

if __name__ == "__main__":
    test_rtpengine_ping()
