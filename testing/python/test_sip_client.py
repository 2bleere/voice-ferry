#!/usr/bin/env python3

import socket
import time
import hashlib
import random

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

def send_sip_register(include_auth=False, nonce=None):
    """Send SIP REGISTER message, optionally with authentication."""
    branch = f"z9hG4bK-test{random.randint(1000, 9999)}"
    tag = f"test{random.randint(1000, 9999)}"
    call_id = f"test{random.randint(1000, 9999)}@127.0.0.1"
    
    # Base SIP REGISTER message
    sip_message = (
        "REGISTER sip:sip-b2bua.local SIP/2.0\r\n"
        f"Via: SIP/2.0/UDP 127.0.0.1:5060;branch={branch}\r\n"
        "Max-Forwards: 70\r\n"
        f"From: <sip:787@sip-b2bua.local>;tag={tag}\r\n"
        "To: <sip:787@sip-b2bua.local>\r\n"
        "Contact: <sip:787@127.0.0.1:5060>\r\n"
        f"Call-ID: {call_id}\r\n"
        "CSeq: 1 REGISTER\r\n"
    )
    
    # Add authentication header if provided
    if include_auth and nonce:
        username = "787"
        realm = "sip-b2bua.local"
        password = "12345"
        method = "REGISTER"
        uri = "sip:sip-b2bua.local"
        
        response = calculate_digest_response(username, realm, password, nonce, method, uri)
        
        auth_header = (
            f'Authorization: Digest username="{username}", '
            f'realm="{realm}", '
            f'nonce="{nonce}", '
            f'uri="{uri}", '
            f'response="{response}"\r\n'
        )
        sip_message += auth_header
    
    sip_message += "Content-Length: 0\r\n\r\n"  # End of headers
    
    print(f"Sending SIP REGISTER message {'with authentication' if include_auth else '(no auth)'}...")
    print(f"Message length: {len(sip_message)} bytes")
    print("Message content:")
    print(sip_message)
    
    # Create UDP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(10.0)
    
    try:
        # Send to SIP server
        sock.sendto(sip_message.encode('utf-8'), ('127.0.0.1', 5060))
        print("Message sent successfully!")
        
        # Try to receive response
        try:
            response, addr = sock.recvfrom(4096)
            response_text = response.decode('utf-8')
            print(f"\nReceived response from {addr}:")
            print(response_text)
            
            # Parse the response for nonce if it's a 401 challenge
            if "401 Unauthorized" in response_text:
                print("\nReceived authentication challenge!")
                # Extract nonce from WWW-Authenticate header
                lines = response_text.split('\r\n')
                for line in lines:
                    if line.startswith('WWW-Authenticate:'):
                        print(f"Challenge header: {line}")
                        # Simple nonce extraction
                        if 'nonce="' in line:
                            start = line.find('nonce="') + 7
                            end = line.find('"', start)
                            extracted_nonce = line[start:end]
                            print(f"Extracted nonce: {extracted_nonce}")
                            return extracted_nonce
            elif "200 OK" in response_text:
                print("\nAuthentication successful!")
            
        except socket.timeout:
            print("No response received within timeout period")
            
    except Exception as e:
        print(f"Error sending message: {e}")
    finally:
        sock.close()
    
    return None

def test_authentication_flow():
    """Test the complete SIP authentication flow."""
    print("=== Testing SIP Authentication Flow ===\n")
    
    # Step 1: Send REGISTER without authentication (should get 401)
    print("Step 1: Sending initial REGISTER request...")
    nonce = send_sip_register(include_auth=False)
    
    if nonce:
        print(f"\nStep 2: Sending REGISTER with authentication using nonce: {nonce}")
        time.sleep(1)
        send_sip_register(include_auth=True, nonce=nonce)
    else:
        print("No nonce received, cannot proceed with authentication")

if __name__ == "__main__":
    test_authentication_flow()
