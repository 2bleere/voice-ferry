#!/usr/bin/env python3
"""
Simple test script to verify per-user session limits using built-in Python libraries only.
Tests the Redis client directly and API endpoints without external dependencies.
This script outputs detailed debug information.
"""

import json
import sys
import socket
import time
import subprocess
from urllib.request import urlopen, Request
from urllib.parse import urlencode
from urllib.error import HTTPError, URLError

# Configuration
B2BUA_API_HOST = "127.0.0.1"
B2BUA_API_PORT = 8080
REDIS_HOST = "127.0.0.1"
REDIS_PORT = 6379

class SessionLimitsTest:
    def __init__(self):
        self.api_base = f"http://{B2BUA_API_HOST}:{B2BUA_API_PORT}/api"
        self.test_results = []
        print(f"Initialized test with API base URL: {self.api_base}")
        print(f"Redis target: {REDIS_HOST}:{REDIS_PORT}")
    
    def log_result(self, test_name, success, message):
        """Log test result"""
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status} {test_name}: {message}")
        self.test_results.append({
            'test': test_name,
            'success': success,
            'message': message
        })
    
    def make_api_request(self, endpoint, method='GET', data=None):
        """Make HTTP API request"""
        url = f"{self.api_base}/{endpoint.lstrip('/')}"
        print(f"DEBUG: Making {method} request to {url}")
        
        try:
            if method == 'GET':
                print(f"DEBUG: Opening GET URL {url}")
                response = urlopen(url, timeout=5)
                print(f"DEBUG: Got response code {response.getcode()}")
            elif method == 'PUT':
                print(f"DEBUG: Creating PUT request to {url} with data: {data}")
                req = Request(url, data=json.dumps(data).encode('utf-8') if data else None)
                req.add_header('Content-Type', 'application/json')
                req.get_method = lambda: 'PUT'
                response = urlopen(req, timeout=5)
                print(f"DEBUG: Got PUT response code {response.getcode()}")
            elif method == 'DELETE':
                print(f"DEBUG: Creating DELETE request to {url}")
                req = Request(url)
                req.get_method = lambda: 'DELETE'
                response = urlopen(req, timeout=5)
                print(f"DEBUG: Got DELETE response code {response.getcode()}")
            else:
                raise ValueError(f"Unsupported method: {method}")
            
            content = response.read().decode('utf-8')
            print(f"DEBUG: Response content: {content[:200]}...")
            return json.loads(content)
        
        except HTTPError as e:
            print(f"DEBUG: HTTP error {e.code} for {url}")
            error_body = e.read().decode('utf-8') if e.fp else str(e)
            print(f"DEBUG: Error body: {error_body}")
            return {'error': f"HTTP {e.code}: {error_body}"}
        except URLError as e:
            print(f"DEBUG: URL error for {url}: {e.reason}")
            return {'error': f"Connection error: {e.reason}"}
        except Exception as e:
            print(f"DEBUG: Exception for {url}: {str(e)}")
            return {'error': f"Request failed: {str(e)}"}
    
    def check_redis_connection(self):
        """Check if Redis is accessible"""
        print(f"DEBUG: Testing Redis connection to {REDIS_HOST}:{REDIS_PORT}")
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            print(f"DEBUG: Connecting to Redis...")
            result = sock.connect_ex((REDIS_HOST, REDIS_PORT))
            sock.close()
            
            if result == 0:
                print(f"DEBUG: Redis connection successful")
                self.log_result("Redis Connection", True, f"Redis is accessible at {REDIS_HOST}:{REDIS_PORT}")
                return True
            else:
                print(f"DEBUG: Redis connection failed with result {result}")
                self.log_result("Redis Connection", False, f"Cannot connect to Redis at {REDIS_HOST}:{REDIS_PORT}")
                return False
        except Exception as e:
            print(f"DEBUG: Redis connection exception: {str(e)}")
            self.log_result("Redis Connection", False, f"Redis connection test failed: {str(e)}")
            return False
    
    def check_api_connection(self):
        """Check if B2BUA API is accessible"""
        print(f"DEBUG: Testing API connection to {self.api_base}")
        try:
            response = self.make_api_request("/sessions/limits")
            if 'error' not in response:
                print(f"DEBUG: API connection successful")
                self.log_result("API Connection", True, f"B2BUA API is accessible at {self.api_base}")
                return True
            else:
                print(f"DEBUG: API connection failed with error: {response.get('error', 'Unknown error')}")
                self.log_result("API Connection", False, f"API error: {response['error']}")
                return False
        except Exception as e:
            print(f"DEBUG: API connection exception: {str(e)}")
            self.log_result("API Connection", False, f"API connection test failed: {str(e)}")
            return False
    
    def run_basic_connectivity_test(self):
        """Just run the connectivity tests"""
        print("\n🧪 Testing Basic Connectivity\n")
        
        # Test 1: Check connections
        redis_ok = self.check_redis_connection()
        api_ok = self.check_api_connection()
        
        print("\n📋 Connectivity Test Summary:")
        print(f"   Redis: {'✅ Connected' if redis_ok else '❌ Not Available'}")
        print(f"   API: {'✅ Connected' if api_ok else '❌ Not Available'}")
        
        return redis_ok and api_ok

def main():
    print("🚀 Voice Ferry Per-User Session Limits - Basic Connectivity Test")
    print("=" * 60)
    
    tester = SessionLimitsTest()
    success = tester.run_basic_connectivity_test()
    
    if not success:
        print("\n⚠️ Basic connectivity test failed.")
        print("Recommendations:")
        print("1. Make sure Redis is running (redis-server or Docker)")
        print("2. Make sure B2BUA server is running (go run cmd/b2bua/main.go or Docker)")
        print("3. Verify port configurations match the test script")
        print(f"   - Redis expected at {REDIS_HOST}:{REDIS_PORT}")
        print(f"   - API expected at {B2BUA_API_HOST}:{B2BUA_API_PORT}")
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
