#!/usr/bin/env python3
"""
Complete SIP REGISTER authentication test
Tests both challenge and authenticated response
"""

import socket
import hashlib
import uuid
import re

def md5_hash(data):
    """Calculate MD5 hash"""
    return hashlib.md5(data.encode()).hexdigest()

def calculate_digest_response(username, realm, password, method, uri, nonce, nc="00000001", cnonce=None, qop="auth"):
    """Calculate digest authentication response"""
    if cnonce is None:
        cnonce = str(uuid.uuid4()).replace('-', '')[:8]
    
    # HA1 = MD5(username:realm:password)
    ha1 = md5_hash(f"{username}:{realm}:{password}")
    
    # HA2 = MD5(method:uri)
    ha2 = md5_hash(f"{method}:{uri}")
    
    # Response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
    response = md5_hash(f"{ha1}:{nonce}:{nc}:{cnonce}:{qop}:{ha2}")
    
    return response, cnonce

def send_sip_message(message):
    """Send SIP message and return response"""
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(5.0)
    
    try:
        sock.sendto(message.encode(), ('127.0.0.1', 5060))
        response, addr = sock.recvfrom(4096)
        return response.decode()
    except Exception as e:
        return f"Error: {e}"
    finally:
        sock.close()

def parse_www_authenticate(header):
    """Parse WWW-Authenticate header"""
    # Extract realm and nonce from Digest challenge
    realm_match = re.search(r'realm="([^"]*)"', header)
    nonce_match = re.search(r'nonce="([^"]*)"', header)
    
    realm = realm_match.group(1) if realm_match else None
    nonce = nonce_match.group(1) if nonce_match else None
    
    return realm, nonce

def test_sip_authentication():
    """Test complete SIP authentication flow"""
    print("🔐 SIP REGISTER Authentication Test")
    print("=" * 50)
    
    # Test parameters
    username = "787"
    password = "12345"
    realm = "sip-b2bua.local"
    call_id = f"test-{uuid.uuid4()}"
    
    # Step 1: Send REGISTER without authentication
    print("\n📤 Step 1: REGISTER without authentication")
    print("-" * 40)
    
    register_msg = f"""REGISTER sip:{realm} SIP/2.0\r
Via: SIP/2.0/UDP 127.0.0.1:5080;branch=z9hG4bK-{uuid.uuid4()}\r
From: <sip:{username}@{realm}>;tag=test-{uuid.uuid4()}\r
To: <sip:{username}@{realm}>\r
Call-ID: {call_id}\r
CSeq: 1 REGISTER\r
Content-Length: 0\r
\r
"""
    
    response1 = send_sip_message(register_msg)
    print("Response:")
    print(response1)
    
    # Check for 401 and extract challenge
    if "401 Unauthorized" not in response1:
        print("❌ Expected 401 Unauthorized response")
        return
    
    www_auth_match = re.search(r'WWW-Authenticate: (.+)', response1)
    if not www_auth_match:
        print("❌ No WWW-Authenticate header found")
        return
    
    www_auth = www_auth_match.group(1)
    challenge_realm, nonce = parse_www_authenticate(www_auth)
    
    print(f"\n✅ Challenge received:")
    print(f"   Realm: {challenge_realm}")
    print(f"   Nonce: {nonce}")
    
    # Step 2: Send REGISTER with authentication
    print("\n📤 Step 2: REGISTER with digest authentication")
    print("-" * 40)
    
    method = "REGISTER"
    uri = f"sip:{realm}"
    response_hash, cnonce = calculate_digest_response(username, challenge_realm, password, method, uri, nonce)
    
    auth_header = f'Digest username="{username}", realm="{challenge_realm}", nonce="{nonce}", uri="{uri}", response="{response_hash}", algorithm=MD5, cnonce="{cnonce}", nc=00000001, qop=auth'
    
    register_auth_msg = f"""REGISTER sip:{realm} SIP/2.0\r
Via: SIP/2.0/UDP 127.0.0.1:5080;branch=z9hG4bK-{uuid.uuid4()}\r
From: <sip:{username}@{realm}>;tag=test-{uuid.uuid4()}\r
To: <sip:{username}@{realm}>\r
Call-ID: {call_id}\r
CSeq: 2 REGISTER\r
Authorization: {auth_header}\r
Content-Length: 0\r
\r
"""
    
    response2 = send_sip_message(register_auth_msg)
    print("Response:")
    print(response2)
    
    # Check result
    if "200 OK" in response2:
        print("\n✅ Authentication successful! REGISTER accepted.")
    elif "403 Forbidden" in response2:
        print("\n❌ Authentication failed - credentials rejected")
    else:
        print(f"\n❓ Unexpected response: {response2[:50]}...")
    
    print(f"\n📊 Test Summary:")
    print(f"   Username: {username}")
    print(f"   Password: {password}")
    print(f"   Realm: {challenge_realm}")
    print(f"   Calculated response: {response_hash}")

if __name__ == "__main__":
    test_sip_authentication()
