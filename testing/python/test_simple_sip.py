#!/usr/bin/env python3
"""
Simple SIP test to debug communication with B2BUA
"""

import socket
import time
import sys

def send_simple_invite():
    """Send a simple SIP INVITE to test B2BUA response."""
    
    print("🧪 Simple SIP Test")
    print("Starting SIP INVITE test...")
    
    # Calculate proper content length
    sdp_content = """v=0
o=test 123456 654321 IN IP4 127.0.0.1
s=Test Session
c=IN IP4 127.0.0.1
t=0 0
m=audio 5556 RTP/AVP 0
a=rtpmap:0 PCMU/8000"""

    content_length = len(sdp_content)
    
    # SIP INVITE message matching routing rules (to 999)
    invite = f"""INVITE sip:999@127.0.0.1:5060 SIP/2.0
Via: SIP/2.0/UDP 127.0.0.1:5555;branch=z9hG4bK-test123
Max-Forwards: 70
From: <sip:787@127.0.0.1:5555>;tag=test123
To: <sip:999@127.0.0.1:5060>
Call-ID: test123@127.0.0.1
CSeq: 1 INVITE
Contact: <sip:787@127.0.0.1:5555>
Content-Type: application/sdp
Content-Length: {content_length}

{sdp_content}""".replace('\n', '\r\n')

    sock = None
    try:
        # Create UDP socket
        print("🔌 Creating UDP socket...")
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(10)
        
        print("📡 Binding to 127.0.0.1:5555...")
        sock.bind(('127.0.0.1', 5555))
        
        print("📞 Sending SIP INVITE to B2BUA...")
        print(f"📋 Target: 787 -> 999 (to match routing rules)")
        print(f"📏 Message length: {len(invite)} bytes")
        print("\n📤 Outgoing message:")
        print("-" * 40)
        print(invite)
        print("-" * 40)
        
        # Send INVITE
        bytes_sent = sock.sendto(invite.encode(), ('127.0.0.1', 5060))
        print(f"✅ Sent {bytes_sent} bytes, waiting for response...")
        
        # Wait for response with timeout
        try:
            response, addr = sock.recvfrom(4096)
            response_str = response.decode()
            
            print(f"📨 Received {len(response)} bytes from {addr}")
            print("\n📥 Incoming response:")
            print("=" * 50)
            print(response_str)
            print("=" * 50)
            
            # Parse response code
            if response_str.startswith('SIP/2.0'):
                lines = response_str.split('\r\n')
                status_line = lines[0]
                print(f"\n📊 Status: {status_line}")
                
                if '200' in status_line:
                    print("✅ Call succeeded (200 OK)")
                elif '486' in status_line or '503' in status_line:
                    print("⚠️ Call rejected (busy/unavailable)")
                elif '404' in status_line:
                    print("❌ Not found (404)")
                elif '401' in status_line or '407' in status_line:
                    print("🔐 Authentication required")
                elif '100' in status_line:
                    print("📞 Trying (100)")
                    
                    # Wait for another response
                    print("⏳ Waiting for final response...")
                    try:
                        response2, addr2 = sock.recvfrom(4096)
                        response2_str = response2.decode()
                        print(f"\n📨 Second response from {addr2}:")
                        print("=" * 50)
                        print(response2_str)
                        print("=" * 50)
                    except socket.timeout:
                        print("⏰ No second response received")
                else:
                    print(f"❓ Other response: {status_line}")
            else:
                print("❓ Unexpected response format")
                
        except socket.timeout:
            print("⏰ Timeout - no response from B2BUA within 10 seconds")
            print("   This could indicate:")
            print("   - B2BUA is not listening on port 5060")
            print("   - Routing rules don't match")
            print("   - SIP message format is incorrect")
            
    except socket.error as e:
        print(f"❌ Socket error: {e}")
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        if sock:
            print("🔌 Closing socket...")
            sock.close()

def main():
    try:
        send_simple_invite()
        print("\n🏁 Test completed")
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
