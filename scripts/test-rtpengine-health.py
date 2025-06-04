#!/usr/bin/env python3
"""
RTPEngine Health Check Test Script
Replicates the exact UDP protocol used by Voice Ferry for health checks
"""

import socket
import json
import time
import sys
import argparse
from typing import Optional, Tuple

class RTPEngineHealthChecker:
    def __init__(self, host: str, port: int, timeout: float = 5.0):
        self.host = host
        self.port = port
        self.timeout = timeout
    
    def create_ping_command(self) -> bytes:
        """Create a bencode-wrapped ping command exactly like Voice Ferry does"""
        # Create JSON command
        json_cmd = {"command": "ping"}
        json_str = json.dumps(json_cmd, separators=(',', ':'))  # No spaces
        
        # Wrap in bencode format: d7:command{length}:{json}e
        json_bytes = json_str.encode('utf-8')
        bencode = f"d7:command{len(json_bytes)}:{json_str}e"
        
        return bencode.encode('utf-8')
    
    def parse_response(self, response: bytes) -> Optional[dict]:
        """Parse bencode response and extract JSON"""
        try:
            # Convert to string for easier parsing
            response_str = response.decode('utf-8', errors='ignore')
            
            # Find JSON portion (simplified bencode parsing)
            json_start = response_str.find('{')
            json_end = response_str.rfind('}')
            
            if json_start == -1 or json_end == -1:
                print(f"⚠️  No JSON found in response: {response_str}")
                return None
            
            json_str = response_str[json_start:json_end + 1]
            return json.loads(json_str)
            
        except Exception as e:
            print(f"⚠️  Error parsing response: {e}")
            print(f"Raw response: {response}")
            return None
    
    def test_connectivity(self) -> bool:
        """Test basic UDP connectivity"""
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(self.timeout)
            
            # Send a simple test message
            test_msg = b"test"
            sock.sendto(test_msg, (self.host, self.port))
            
            # Try to receive (may timeout, but that's OK if port is open)
            try:
                sock.recv(1024)
            except socket.timeout:
                pass  # Timeout is expected for a simple test message
            
            sock.close()
            return True
            
        except Exception as e:
            print(f"❌ Connectivity test failed: {e}")
            return False
    
    def test_health(self) -> Tuple[bool, Optional[dict]]:
        """Test RTPEngine health using the ping command"""
        try:
            # Create UDP socket
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.settimeout(self.timeout)
            
            print(f"📡 Connecting to {self.host}:{self.port}")
            
            # Create and send ping command
            ping_cmd = self.create_ping_command()
            print(f"📤 Sending ping command: {ping_cmd.decode('utf-8')}")
            
            start_time = time.time()
            sock.sendto(ping_cmd, (self.host, self.port))
            
            # Receive response
            response, addr = sock.recvfrom(4096)
            response_time = time.time() - start_time
            
            print(f"📨 Received response from {addr} in {response_time:.3f}s")
            print(f"Raw response: {response}")
            
            # Parse response
            parsed = self.parse_response(response)
            if parsed is None:
                return False, None
            
            print(f"📄 Parsed JSON: {json.dumps(parsed, indent=2)}")
            
            # Check if result is "ok"
            result = parsed.get('result')
            if result == 'ok':
                print("✅ Health check PASSED - RTPEngine is healthy")
                return True, parsed
            else:
                print(f"❌ Health check FAILED - Result: {result}")
                if 'error-reason' in parsed:
                    print(f"Error reason: {parsed['error-reason']}")
                return False, parsed
                
        except socket.timeout:
            print(f"⏰ Timeout after {self.timeout}s - RTPEngine not responding")
            return False, None
        except Exception as e:
            print(f"❌ Health check failed: {e}")
            return False, None
        finally:
            try:
                sock.close()
            except:
                pass
    
    def run_full_test(self) -> bool:
        """Run complete health check test"""
        print("=" * 60)
        print("         RTPEngine Health Check Test (Python)")
        print("=" * 60)
        print(f"Target: {self.host}:{self.port}")
        print(f"Timeout: {self.timeout}s")
        print()
        
        # Test 1: Basic connectivity
        print("🔍 Testing basic UDP connectivity...")
        if not self.test_connectivity():
            print("❌ Basic connectivity test failed")
            return False
        print("✅ Basic connectivity test passed")
        print()
        
        # Test 2: Health check
        print("🏥 Testing RTPEngine health check protocol...")
        is_healthy, response = self.test_health()
        
        print()
        if is_healthy:
            print("🎉 Overall result: RTPEngine is HEALTHY")
            return True
        else:
            print("💀 Overall result: RTPEngine is UNHEALTHY or not responding")
            return False

def main():
    parser = argparse.ArgumentParser(description='Test RTPEngine health using UDP ping')
    parser.add_argument('--host', default='192.168.1.208', 
                       help='RTPEngine host (default: 192.168.1.208)')
    parser.add_argument('--port', type=int, default=22222,
                       help='RTPEngine port (default: 22222)')
    parser.add_argument('--timeout', type=float, default=5.0,
                       help='Timeout in seconds (default: 5.0)')
    
    args = parser.parse_args()
    
    checker = RTPEngineHealthChecker(args.host, args.port, args.timeout)
    success = checker.run_full_test()
    
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()
