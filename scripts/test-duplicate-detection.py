#!/usr/bin/env python3
"""
Test RTPEngine with systematic approach to find the working bencode format.
Focus on the 'duplicate' message pattern found in logs.
"""

import socket
import time
import hashlib
import struct

RTPENGINE_HOST = "192.168.1.208"
RTPENGINE_PORT = 22222

def create_bencode_variations():
    """Create systematic variations to test the exact working format"""
    
    # Generate a unique cookie for this session
    session_id = str(int(time.time()))
    base_cookie = f"test_{session_id}"
    
    variations = []
    
    # Test 1: Exact format from successful duplicate detection
    # From logs: "Detected command from 192.168.1.74:58920 as a duplicate"
    # This suggests the previous command was processed successfully
    cmd1 = '{"command":"ping"}'
    bencode1 = f'd6:cookie{len(base_cookie)}:{base_cookie}7:command{len(cmd1)}:{cmd1}e'
    variations.append(("Basic ping format", bencode1))
    
    # Test 2: Add call-id (common in SIP)
    call_id = "testcall"
    cmd2 = f'{{"command":"ping","call-id":"{call_id}"}}'
    bencode2 = f'd6:cookie{len(base_cookie)}:{base_cookie}7:call-id{len(call_id)}:{call_id}7:command{len(cmd2)}:{cmd2}e'
    variations.append(("With call-id", bencode2))
    
    # Test 3: Try with transaction ID (some protocols need this)
    transaction_id = f"trans_{session_id}"
    bencode3 = f'd6:cookie{len(base_cookie)}:{base_cookie}12:transaction-id{len(transaction_id)}:{transaction_id}7:command{len(cmd1)}:{cmd1}e'
    variations.append(("With transaction ID", bencode3))
    
    # Test 4: Minimal format
    bencode4 = f'd6:cookie{len(base_cookie)}:{base_cookie}7:command4:pinge'
    variations.append(("Minimal ping", bencode4))
    
    # Test 5: Try different cookie length/format
    short_cookie = "test123"
    bencode5 = f'd6:cookie{len(short_cookie)}:{short_cookie}7:command{len(cmd1)}:{cmd1}e'
    variations.append(("Short cookie", bencode5))
    
    # Test 6: Numeric cookie
    numeric_cookie = str(int(time.time() * 1000))
    bencode6 = f'd6:cookie{len(numeric_cookie)}:{numeric_cookie}7:command{len(cmd1)}:{cmd1}e'
    variations.append(("Numeric cookie", bencode6))
    
    return variations

def send_bencode_with_retry(sock, data, retries=2):
    """Send bencode data with retry logic to trigger duplicate detection"""
    
    print(f"  Sending: {data}")
    print(f"  Length: {len(data)} bytes")
    
    try:
        # Send first attempt
        sent = sock.sendto(data.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
        print(f"  ✓ Sent {sent} bytes")
        
        # Wait a moment
        time.sleep(0.5)
        
        # Send duplicate to trigger the duplicate detection
        # This should help us see if the first one was processed successfully
        sent2 = sock.sendto(data.encode('utf-8'), (RTPENGINE_HOST, RTPENGINE_PORT))
        print(f"  ✓ Sent duplicate {sent2} bytes (looking for duplicate detection)")
        
        return True
        
    except Exception as e:
        print(f"  ✗ Send failed: {e}")
        return False

def test_systematic_bencode():
    """Test systematic bencode variations"""
    
    print("=== Systematic RTPEngine Bencode Test ===")
    print(f"Target: {RTPENGINE_HOST}:{RTPENGINE_PORT}")
    print("Strategy: Send each format twice to trigger duplicate detection")
    print("Success = seeing 'duplicate' message in RTPEngine logs")
    print()
    
    variations = create_bencode_variations()
    
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        
        for i, (description, bencode_data) in enumerate(variations, 1):
            print(f"{i}. Testing: {description}")
            
            success = send_bencode_with_retry(sock, bencode_data)
            
            if success:
                print("  → Check RTPEngine logs for 'duplicate' message")
            else:
                print("  → Failed to send")
            
            print()
            time.sleep(2)  # Wait between tests
            
    except Exception as e:
        print(f"Socket error: {e}")
    finally:
        if 'sock' in locals():
            sock.close()

def monitor_logs_suggestion():
    """Print log monitoring command"""
    print("\n" + "="*60)
    print("TO MONITOR RTPENGINE LOGS:")
    print("kubectl logs -n voice-ferry deployment/rtpengine -f")
    print("\nLOOK FOR:")
    print("- 'Detected command ... as a duplicate' ← SUCCESS!")
    print("- 'Received invalid NG data (no cookie)' ← FAILURE")
    print("- Any other response patterns")
    print("="*60)

if __name__ == "__main__":
    monitor_logs_suggestion()
    print()
    
    try:
        test_systematic_bencode()
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
    except Exception as e:
        print(f"Test failed: {e}")
