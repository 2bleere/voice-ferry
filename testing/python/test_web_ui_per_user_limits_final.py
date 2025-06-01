#!/usr/bin/env python3
"""
Final comprehensive test for per-user session limits via Web UI API.
This test validates the complete per-user session limits implementation.
"""

import requests
import json
import sys

# Configuration
WEB_UI_URL = "http://localhost:3001"
USERNAME = "admin"
PASSWORD = "admin123"

def login():
    """Login and get JWT token"""
    login_data = {
        "username": USERNAME,
        "password": PASSWORD
    }
    
    response = requests.post(
        f"{WEB_UI_URL}/api/auth/login",
        json=login_data,
        headers={"Content-Type": "application/json"}
    )
    
    if response.status_code != 200:
        print(f"❌ Login failed: {response.status_code}")
        print(response.text)
        return None
    
    token = response.json().get("token")
    print(f"✅ Login successful, token obtained")
    return token

def test_get_session_limits_config(token):
    """Test getting session limits configuration"""
    print("\n🔍 Testing GET /api/sessions/limits")
    
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    response = requests.get(f"{WEB_UI_URL}/api/sessions/limits", headers=headers)
    
    if response.status_code != 200:
        print(f"❌ Failed to get session limits: {response.status_code}")
        print(response.text)
        return False
    
    config = response.json()
    print(f"✅ Session limits config retrieved:")
    print(f"   - Enabled: {config.get('enabled')}")
    print(f"   - Max sessions per user: {config.get('max_sessions_per_user')}")
    print(f"   - Session limit action: {config.get('session_limit_action')}")
    print(f"   - User limits: {config.get('user_limits')}")
    
    # Verify required fields
    required_fields = ['enabled', 'max_sessions_per_user', 'session_limit_action', 'user_limits']
    for field in required_fields:
        if field not in config:
            print(f"❌ Missing required field: {field}")
            return False
    
    return True, config

def test_get_user_limit(token, username):
    """Test getting user-specific session limit"""
    print(f"\n🔍 Testing GET /api/sessions/limits/{username}")
    
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    response = requests.get(f"{WEB_UI_URL}/api/sessions/limits/{username}", headers=headers)
    
    if response.status_code != 200:
        print(f"❌ Failed to get user limit: {response.status_code}")
        print(response.text)
        return False, None
    
    limit_data = response.json()
    print(f"✅ User limit retrieved:")
    print(f"   - Username: {limit_data.get('username')}")
    print(f"   - Limit: {limit_data.get('limit')}")
    
    return True, limit_data.get('limit')

def test_set_user_limit(token, username, limit):
    """Test setting user-specific session limit"""
    print(f"\n🔍 Testing PUT /api/sessions/limits/{username} (limit: {limit})")
    
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    data = {"limit": limit}
    
    response = requests.put(
        f"{WEB_UI_URL}/api/sessions/limits/{username}",
        json=data,
        headers=headers
    )
    
    if response.status_code != 200:
        print(f"❌ Failed to set user limit: {response.status_code}")
        print(response.text)
        return False
    
    result = response.json()
    print(f"✅ User limit set:")
    print(f"   - Success: {result.get('success')}")
    print(f"   - Username: {result.get('username')}")
    print(f"   - Limit: {result.get('limit')}")
    print(f"   - Message: {result.get('message')}")
    
    return result.get('success') and result.get('limit') == limit

def test_delete_user_limit(token, username):
    """Test deleting user-specific session limit"""
    print(f"\n🔍 Testing DELETE /api/sessions/limits/{username}")
    
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    response = requests.delete(f"{WEB_UI_URL}/api/sessions/limits/{username}", headers=headers)
    
    if response.status_code != 200:
        print(f"❌ Failed to delete user limit: {response.status_code}")
        print(response.text)
        return False
    
    result = response.json()
    print(f"✅ User limit deleted:")
    print(f"   - Success: {result.get('success')}")
    print(f"   - Username: {result.get('username')}")
    print(f"   - Message: {result.get('message')}")
    
    return result.get('success')

def run_comprehensive_test():
    """Run comprehensive per-user session limits test"""
    print("🚀 Starting comprehensive per-user session limits test\n")
    
    # Step 1: Login
    token = login()
    if not token:
        return False
    
    # Step 2: Get initial configuration
    success, initial_config = test_get_session_limits_config(token)
    if not success:
        return False
    
    default_limit = initial_config.get('max_sessions_per_user', 5)
    initial_user_limits = initial_config.get('user_limits', {})
    
    # Step 3: Test getting non-existent user limit (should return default)
    test_username = "testuser123"
    success, limit = test_get_user_limit(token, test_username)
    if not success:
        return False
    
    if limit != default_limit:
        print(f"❌ Expected default limit {default_limit}, got {limit}")
        return False
    print(f"✅ Non-existent user correctly returns default limit")
    
    # Step 4: Set a custom user limit
    custom_limit = 15
    success = test_set_user_limit(token, test_username, custom_limit)
    if not success:
        return False
    
    # Step 5: Verify the custom limit was set
    success, limit = test_get_user_limit(token, test_username)
    if not success or limit != custom_limit:
        print(f"❌ Expected custom limit {custom_limit}, got {limit}")
        return False
    print(f"✅ Custom user limit correctly set and retrieved")
    
    # Step 6: Verify the limit appears in the global config
    success, updated_config = test_get_session_limits_config(token)
    if not success:
        return False
    
    user_limits = updated_config.get('user_limits', {})
    if test_username not in user_limits or user_limits[test_username] != custom_limit:
        print(f"❌ User limit not found in global config: {user_limits}")
        return False
    print(f"✅ User limit correctly appears in global configuration")
    
    # Step 7: Test updating the user limit
    new_limit = 20
    success = test_set_user_limit(token, test_username, new_limit)
    if not success:
        return False
    
    success, limit = test_get_user_limit(token, test_username)
    if not success or limit != new_limit:
        print(f"❌ Expected updated limit {new_limit}, got {limit}")
        return False
    print(f"✅ User limit correctly updated")
    
    # Step 8: Delete the user limit
    success = test_delete_user_limit(token, test_username)
    if not success:
        return False
    
    # Step 9: Verify user now gets default limit
    success, limit = test_get_user_limit(token, test_username)
    if not success or limit != default_limit:
        print(f"❌ Expected default limit {default_limit} after deletion, got {limit}")
        return False
    print(f"✅ User correctly reverted to default limit after deletion")
    
    # Step 10: Verify user is removed from global config
    success, final_config = test_get_session_limits_config(token)
    if not success:
        return False
    
    final_user_limits = final_config.get('user_limits', {})
    if test_username in final_user_limits:
        print(f"❌ User still found in global config after deletion: {final_user_limits}")
        return False
    print(f"✅ User correctly removed from global configuration")
    
    print(f"\n🎉 All per-user session limits tests passed successfully!")
    print(f"\n📊 Test Summary:")
    print(f"   ✅ Authentication working")
    print(f"   ✅ GET /api/sessions/limits - retrieve global config")
    print(f"   ✅ GET /api/sessions/limits/:username - retrieve user limit")
    print(f"   ✅ PUT /api/sessions/limits/:username - set user limit")
    print(f"   ✅ DELETE /api/sessions/limits/:username - delete user limit")
    print(f"   ✅ Default limit fallback working")
    print(f"   ✅ User limit persistence in mock storage")
    print(f"   ✅ User limit removal working")
    
    return True

if __name__ == "__main__":
    try:
        success = run_comprehensive_test()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n⚠️  Test interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Test failed with exception: {e}")
        sys.exit(1)
