#!/usr/bin/env python3
"""
Minimal session limits test
"""

import socket
import time
import threading
from concurrent.futures import ThreadPoolExecutor

def send_sip_invite(from_user, call_id):
    """Send a SIP INVITE and return basic result."""
    try:
        # Create SIP message
        invite = f"""INVITE sip:999@127.0.0.1:5060 SIP/2.0\r
Via: SIP/2.0/UDP 127.0.0.1:{5500 + call_id};branch=z9hG4bK-{call_id}\r
Max-Forwards: 70\r
From: <sip:{from_user}@127.0.0.1:{5500 + call_id}>;tag={call_id}\r
To: <sip:999@127.0.0.1:5060>\r
Call-ID: {call_id}@127.0.0.1\r
CSeq: 1 INVITE\r
Contact: <sip:{from_user}@127.0.0.1:{5500 + call_id}>\r
Content-Length: 0\r
\r
"""
        
        # Send via UDP
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        sock.bind(('127.0.0.1', 5500 + call_id))
        
        print(f"📞 Sending call {call_id}: {from_user} -> 999")
        sock.sendto(invite.encode(), ('127.0.0.1', 5060))
        
        # Wait for response
        response, addr = sock.recvfrom(4096)
        response_str = response.decode()
        
        sock.close()
        
        if "100 Trying" in response_str:
            print(f"✅ Call {call_id}: Accepted (100 Trying)")
            return "ACCEPTED"
        elif "486" in response_str or "503" in response_str:
            print(f"⛔ Call {call_id}: Rejected ({response_str.split()[1]})")
            return "REJECTED"
        else:
            print(f"❓ Call {call_id}: Unknown response")
            return "UNKNOWN"
            
    except socket.timeout:
        print(f"⏰ Call {call_id}: Timeout")
        return "TIMEOUT"
    except Exception as e:
        print(f"❌ Call {call_id}: Error - {e}")
        return "ERROR"

def test_session_limits():
    """Test session limits with concurrent calls."""
    print("🧪 Session Limits Test")
    print("Testing with user787 (should have max 3 sessions)")
    print("=" * 50)
    
    # Send multiple concurrent calls
    results = []
    with ThreadPoolExecutor(max_workers=6) as executor:
        futures = []
        
        # Send 8 calls rapidly
        for i in range(8):
            future = executor.submit(send_sip_invite, "user787", i + 1)
            futures.append(future)
            time.sleep(0.1)  # Small delay between calls
            
        # Collect results
        for i, future in enumerate(futures):
            try:
                result = future.result(timeout=10)
                results.append(result)
            except Exception as e:
                print(f"❌ Call {i+1} failed: {e}")
                results.append("ERROR")
    
    # Analyze results
    print("\n📊 Results Summary:")
    print(f"Total calls: {len(results)}")
    print(f"Accepted: {results.count('ACCEPTED')}")
    print(f"Rejected: {results.count('REJECTED')}")
    print(f"Timeout: {results.count('TIMEOUT')}")
    print(f"Error: {results.count('ERROR')}")
    
    if results.count('REJECTED') > 0:
        print("✅ Session limits appear to be working (some calls rejected)")
    elif results.count('ACCEPTED') <= 3:
        print("✅ Session limits may be working (≤3 calls accepted)")
    else:
        print("❌ Session limits may not be working (too many calls accepted)")

if __name__ == "__main__":
    test_session_limits()
