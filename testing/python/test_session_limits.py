#!/usr/bin/env python3

import socket
import time
import hashlib
import random
import threading

def md5_hash(text):
    """Calculate MD5 hash of text."""
    return hashlib.md5(text.encode('utf-8')).hexdigest()

def calculate_digest_response(username, realm, password, nonce, method, uri):
    """Calculate SIP digest authentication response."""
    # Calculate HA1
    ha1 = md5_hash(f"{username}:{realm}:{password}")
    
    # Calculate HA2  
    ha2 = md5_hash(f"{method}:{uri}")
    
    # Calculate response
    response = md5_hash(f"{ha1}:{nonce}:{ha2}")
    
    return response

def send_sip_invite(call_num, username="787", password="12345"):
    """Send SIP INVITE message to test session limits."""
    branch = f"z9hG4bK-test{random.randint(1000, 9999)}"
    tag = f"test{random.randint(1000, 9999)}"
    call_id = f"session-limit-test-{call_num}-{random.randint(1000, 9999)}@127.0.0.1"
    
    # Create SDP content
    sdp_content = (
        "v=0\r\n"
        "o=- 1234567890 1234567890 IN IP4 127.0.0.1\r\n"
        "s=Session SDP\r\n"
        "c=IN IP4 127.0.0.1\r\n"
        "t=0 0\r\n"
        "m=audio 49170 RTP/AVP 0 8 97\r\n"
        "a=rtpmap:0 PCMU/8000\r\n"
        "a=rtpmap:8 PCMA/8000\r\n"
        "a=rtpmap:97 iLBC/8000\r\n"
        "a=sendrecv\r\n"
    )
    
    content_length = len(sdp_content.encode('utf-8'))
    
    # Create SIP INVITE message
    sip_message = (
        "INVITE sip:999@sip-b2bua.local SIP/2.0\r\n"
        f"Via: SIP/2.0/UDP 127.0.0.1:5060;branch={branch}\r\n"
        "Max-Forwards: 70\r\n"
        f"From: <sip:{username}@sip-b2bua.local>;tag={tag}\r\n"
        "To: <sip:999@sip-b2bua.local>\r\n"
        f"Contact: <sip:{username}@127.0.0.1:5060>\r\n"
        f"Call-ID: {call_id}\r\n"
        "CSeq: 1 INVITE\r\n"
        "Content-Type: application/sdp\r\n"
        f"Content-Length: {content_length}\r\n"
        "\r\n"
        f"{sdp_content}"
    )
    
    print(f"📤 Sending INVITE #{call_num} (Call-ID: {call_id})")
    
    # Create UDP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(2.0)
    
    try:
        # Send to SIP server
        sock.sendto(sip_message.encode('utf-8'), ('127.0.0.1', 5060))
        
        # Try to receive response
        try:
            response, addr = sock.recvfrom(4096)
            response_text = response.decode('utf-8')
            
            # Extract status code
            first_line = response_text.split('\r\n')[0]
            if "SIP/2.0" in first_line:
                status_parts = first_line.split(' ', 2)
                if len(status_parts) >= 3:
                    status_code = status_parts[1]
                    status_text = status_parts[2]
                    
                    if status_code == "503":
                        print(f"   ❌ INVITE #{call_num}: {status_code} {status_text} (SESSION LIMIT EXCEEDED)")
                        return "rejected"
                    elif status_code == "401":
                        print(f"   🔐 INVITE #{call_num}: {status_code} {status_text} (Authentication required)")
                        return "auth_required"
                    elif status_code == "200":
                        print(f"   ✅ INVITE #{call_num}: {status_code} {status_text} (Accepted)")
                        return "accepted"
                    elif status_code == "100" or status_code == "180" or status_code == "183":
                        print(f"   ⏳ INVITE #{call_num}: {status_code} {status_text} (In progress)")
                        return "in_progress"
                    else:
                        print(f"   ⚠️  INVITE #{call_num}: {status_code} {status_text}")
                        return "other"
            
            print(f"   📋 INVITE #{call_num}: Unexpected response format")
            print(f"      Response: {response_text[:100]}...")
            return "unexpected"
            
        except socket.timeout:
            print(f"   ⏰ INVITE #{call_num}: No response (timeout)")
            return "timeout"
            
    except Exception as e:
        print(f"   💥 INVITE #{call_num}: Error - {e}")
        return "error"
    finally:
        sock.close()

def test_session_limits():
    """Test session limits by sending multiple INVITE requests from the same user."""
    print("=" * 60)
    print("🧪 Testing SIP User Session Limits")
    print("=" * 60)
    print("User: 787")
    print("Expected Limit: 3 sessions per user (from config)")
    print("Expected Action: reject")
    print()
    
    results = []
    
    # Send 5 INVITE requests to test the limit
    for i in range(1, 6):
        print(f"Sending INVITE #{i}...")
        result = send_sip_invite(i)
        results.append(result)
        time.sleep(1)  # Small delay between requests
    
    print()
    print("=" * 60)
    print("📊 Test Results Summary:")
    print("=" * 60)
    
    for i, result in enumerate(results, 1):
        status_emoji = {
            "rejected": "❌",
            "accepted": "✅", 
            "auth_required": "🔐",
            "in_progress": "⏳",
            "timeout": "⏰",
            "error": "💥",
            "other": "⚠️",
            "unexpected": "📋"
        }.get(result, "❓")
        
        print(f"   INVITE #{i}: {status_emoji} {result}")
    
    print()
    
    # Analyze results
    rejected_count = results.count("rejected")
    accepted_count = results.count("accepted") + results.count("in_progress")
    auth_required_count = results.count("auth_required")
    
    print("🔍 Analysis:")
    if auth_required_count > 0:
        print(f"   - {auth_required_count} requests required authentication (expected)")
        print("   - Session limits may not be tested if authentication is blocking requests")
    
    if rejected_count >= 2:  # Expecting 2 rejections (4th and 5th requests)
        print(f"   - ✅ Session limiting appears to be working! ({rejected_count} requests rejected)")
        print("   - This suggests the limit is being enforced correctly")
    elif rejected_count > 0:
        print(f"   - ⚠️  Some requests were rejected ({rejected_count}), but not as expected")
    else:
        print("   - ❌ No requests were rejected due to session limits")
        print("   - Session limiting may not be working or authentication is preventing testing")
    
    if accepted_count > 0:
        print(f"   - {accepted_count} requests were accepted/processed")
    
    print()
    print("💡 Next Steps:")
    print("   - Check server logs: docker logs sip-b2bua --tail=20")
    print("   - Look for 'session limit' messages in the logs")
    print("   - Verify Redis configuration and session tracking")

if __name__ == "__main__":
    test_session_limits()
