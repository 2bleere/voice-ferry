#!/usr/bin/env python3
"""
Simple test script to verify per-user session limits using built-in Python libraries only.
Tests the Redis client directly and API endpoints without external dependencies.
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
        
        try:
            if method == 'GET':
                response = urlopen(url, timeout=10)
            elif method == 'PUT':
                req = Request(url, data=json.dumps(data).encode('utf-8') if data else None)
                req.add_header('Content-Type', 'application/json')
                req.get_method = lambda: 'PUT'
                response = urlopen(req, timeout=10)
            elif method == 'DELETE':
                req = Request(url)
                req.get_method = lambda: 'DELETE'
                response = urlopen(req, timeout=10)
            else:
                raise ValueError(f"Unsupported method: {method}")
            
            return json.loads(response.read().decode('utf-8'))
        
        except HTTPError as e:
            error_body = e.read().decode('utf-8') if e.fp else str(e)
            return {'error': f"HTTP {e.code}: {error_body}"}
        except URLError as e:
            return {'error': f"Connection error: {e.reason}"}
        except Exception as e:
            return {'error': f"Request failed: {str(e)}"}
    
    def check_redis_connection(self):
        """Check if Redis is accessible"""
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(5)
            result = sock.connect_ex((REDIS_HOST, REDIS_PORT))
            sock.close()
            
            if result == 0:
                self.log_result("Redis Connection", True, f"Redis is accessible at {REDIS_HOST}:{REDIS_PORT}")
                return True
            else:
                self.log_result("Redis Connection", False, f"Cannot connect to Redis at {REDIS_HOST}:{REDIS_PORT}")
                return False
        except Exception as e:
            self.log_result("Redis Connection", False, f"Redis connection test failed: {str(e)}")
            return False
    
    def check_api_connection(self):
        """Check if B2BUA API is accessible"""
        try:
            response = self.make_api_request("/sessions/limits")
            if 'error' not in response:
                self.log_result("API Connection", True, f"B2BUA API is accessible at {self.api_base}")
                return True
            else:
                self.log_result("API Connection", False, f"API error: {response['error']}")
                return False
        except Exception as e:
            self.log_result("API Connection", False, f"API connection test failed: {str(e)}")
            return False
    
    def test_get_default_limits(self):
        """Test getting default session limits configuration"""
        response = self.make_api_request("/sessions/limits")
        
        if 'error' in response:
            self.log_result("Get Default Limits", False, f"Failed to get limits: {response['error']}")
            return False
        
        # Check for expected fields
        expected_fields = ['enable_session_limits', 'max_sessions_per_user']
        missing_fields = [field for field in expected_fields if field not in response]
        
        if missing_fields:
            self.log_result("Get Default Limits", False, f"Missing fields: {missing_fields}")
            return False
        
        self.log_result("Get Default Limits", True, 
                       f"Default limit: {response.get('max_sessions_per_user', 'unknown')}, "
                       f"Enabled: {response.get('enable_session_limits', 'unknown')}")
        return response
    
    def test_set_user_limit(self, username, limit):
        """Test setting user-specific session limit"""
        data = {'limit': limit}
        response = self.make_api_request(f"/sessions/limits/{username}", method='PUT', data=data)
        
        if 'error' in response:
            self.log_result(f"Set User Limit ({username})", False, f"Failed to set limit: {response['error']}")
            return False
        
        self.log_result(f"Set User Limit ({username})", True, f"Set limit to {limit}")
        return True
    
    def test_get_user_limit(self, username, expected_limit=None):
        """Test getting user-specific session limit"""
        response = self.make_api_request(f"/sessions/limits/{username}")
        
        if 'error' in response:
            self.log_result(f"Get User Limit ({username})", False, f"Failed to get limit: {response['error']}")
            return False
        
        actual_limit = response.get('limit')
        if expected_limit is not None and actual_limit != expected_limit:
            self.log_result(f"Get User Limit ({username})", False, 
                           f"Expected {expected_limit}, got {actual_limit}")
            return False
        
        self.log_result(f"Get User Limit ({username})", True, f"Limit: {actual_limit}")
        return actual_limit
    
    def test_delete_user_limit(self, username):
        """Test deleting user-specific session limit"""
        response = self.make_api_request(f"/sessions/limits/{username}", method='DELETE')
        
        if 'error' in response:
            self.log_result(f"Delete User Limit ({username})", False, f"Failed to delete limit: {response['error']}")
            return False
        
        self.log_result(f"Delete User Limit ({username})", True, "Limit deleted successfully")
        return True
    
    def test_session_stats(self):
        """Test getting session statistics"""
        response = self.make_api_request("/sessions/stats")
        
        if 'error' in response:
            self.log_result("Session Stats", False, f"Failed to get stats: {response['error']}")
            return False
        
        # Check for basic stats fields
        stats_info = []
        if 'total_sessions' in response:
            stats_info.append(f"Total: {response['total_sessions']}")
        if 'active_users' in response:
            stats_info.append(f"Users: {response['active_users']}")
        
        self.log_result("Session Stats", True, f"Stats: {', '.join(stats_info) if stats_info else 'Available'}")
        return response
    
    def run_per_user_limits_tests(self):
        """Run comprehensive per-user limits tests"""
        print("\n🧪 Starting Per-User Session Limits Tests\n")
        
        # Test 1: Check connections
        redis_ok = self.check_redis_connection()
        api_ok = self.check_api_connection()
        
        if not api_ok:
            print("\n❌ Cannot proceed without API connection")
            return False
        
        # Test 2: Get default configuration
        default_config = self.test_get_default_limits()
        if not default_config:
            print("\n❌ Cannot get default configuration")
            return False
        
        # Test 3: Test user-specific limits
        test_users = [
            ('alice', 10),
            ('bob', 3),
            ('charlie', 0),  # Unlimited
            ('diana', 1)     # Very restricted
        ]
        
        print("\n📝 Testing User-Specific Limits...")
        
        # Set limits for test users
        for username, limit in test_users:
            self.test_set_user_limit(username, limit)
        
        # Verify limits were set correctly
        for username, expected_limit in test_users:
            self.test_get_user_limit(username, expected_limit)
        
        # Test 4: Test session statistics
        print("\n📊 Testing Session Statistics...")
        self.test_session_stats()
        
        # Test 5: Clean up - delete user limits
        print("\n🧹 Cleaning Up Test Data...")
        for username, _ in test_users:
            self.test_delete_user_limit(username)
        
        # Verify limits were deleted (should return to default)
        default_limit = default_config.get('max_sessions_per_user', 5)
        for username, _ in test_users:
            self.test_get_user_limit(username, default_limit)
        
        # Summary
        print(f"\n📋 Test Summary:")
        passed = sum(1 for result in self.test_results if result['success'])
        total = len(self.test_results)
        
        print(f"   Tests Passed: {passed}/{total}")
        print(f"   Success Rate: {passed/total*100:.1f}%")
        
        if passed == total:
            print("\n🎉 All tests passed! Per-user session limits are working correctly.")
            return True
        else:
            print(f"\n⚠️  {total-passed} test(s) failed. Check the implementation.")
            print("\nFailed tests:")
            for result in self.test_results:
                if not result['success']:
                    print(f"   - {result['test']}: {result['message']}")
            return False

def main():
    print("🚀 Voice Ferry Per-User Session Limits Test")
    print("=" * 50)
    
    tester = SessionLimitsTest()
    success = tester.run_per_user_limits_tests()
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
