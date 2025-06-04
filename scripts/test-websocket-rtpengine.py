#!/usr/bin/env python3
"""
Test RTPEngine WebSocket connection to avoid UDP bencode issues.
This approach might be more reliable than UDP NG protocol.
"""

import asyncio
import websockets
import json
import time
import sys

RTPENGINE_HOST = "192.168.1.208"
RTPENGINE_WS_PORT = 8080  # Default WebSocket port for RTPEngine

async def test_websocket_connection():
    """Test WebSocket connection to RTPEngine"""
    
    # Test both WebSocket protocols mentioned in the configuration
    protocols = [
        "rtpengine-ng",
        "rtpengine-ng-plain"
    ]
    
    for protocol in protocols:
        print(f"\n=== Testing WebSocket with protocol: {protocol} ===")
        
        try:
            uri = f"ws://{RTPENGINE_HOST}:{RTPENGINE_WS_PORT}"
            print(f"Connecting to: {uri}")
            print(f"Using protocol: {protocol}")
            
            # Try to connect with the specific protocol
            async with websockets.connect(
                uri, 
                subprotocols=[protocol]
            ) as websocket:
                print(f"✓ Connected successfully with protocol {protocol}")
                
                # Test ping command
                ping_command = {
                    "command": "ping",
                    "cookie": "websocket_test_" + str(int(time.time()))
                }
                
                print(f"Sending: {json.dumps(ping_command)}")
                await websocket.send(json.dumps(ping_command))
                
                # Wait for response
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=5)
                    print(f"✓ Received response: {response}")
                    return True
                except asyncio.TimeoutError:
                    print("⚠ No response received within 5 seconds")
                    
        except websockets.exceptions.ConnectionClosed as e:
            print(f"✗ Connection closed: {e}")
        except websockets.exceptions.InvalidStatusCode as e:
            print(f"✗ Invalid status code: {e}")
        except websockets.exceptions.WebSocketException as e:
            print(f"✗ WebSocket error: {e}")
        except ConnectionRefusedError:
            print(f"✗ Connection refused - WebSocket might not be enabled on port {RTPENGINE_WS_PORT}")
        except Exception as e:
            print(f"✗ Error: {e}")
    
    return False

async def test_alternative_ports():
    """Test alternative WebSocket ports"""
    
    alternative_ports = [8080, 8081, 9080, 22223, 22224]
    
    print(f"\n=== Testing Alternative WebSocket Ports ===")
    
    for port in alternative_ports:
        try:
            uri = f"ws://{RTPENGINE_HOST}:{port}"
            print(f"Testing port {port}...")
            
            async with websockets.connect(uri) as websocket:
                print(f"✓ Connected to port {port}")
                return port
                
        except Exception as e:
            print(f"✗ Port {port}: {type(e).__name__}")
    
    return None

async def main():
    print("=== RTPEngine WebSocket Connection Test ===")
    print(f"Target: {RTPENGINE_HOST}")
    
    # First test standard WebSocket connection
    success = await test_websocket_connection()
    
    if not success:
        # Try alternative ports
        working_port = await test_alternative_ports()
        if working_port:
            print(f"\n✓ Found working WebSocket port: {working_port}")
        else:
            print(f"\n✗ No working WebSocket ports found")
            print("This suggests RTPEngine might not have WebSocket interface enabled")
            print("or it's configured differently than expected.")

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
    except Exception as e:
        print(f"Test failed: {e}")
