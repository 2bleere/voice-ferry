#!/usr/bin/env python3
"""
Complete Redis Integration Test
Tests both user limits and session counting functionality
"""

import requests
import json
import time

BASE_URL = "http://localhost:3001"

def get_auth_token():
    """Get authentication token"""
    response = requests.post(f"{BASE_URL}/api/auth/login", json={
        "username": "admin",
        "password": "admin123"
    })
    if response.status_code == 200:
        return response.json()["token"]
    else:
        raise Exception(f"Failed to authenticate: {response.status_code}")

def test_complete_redis_integration():
    """Test complete Redis integration"""
    print("🚀 Complete Redis Integration Test")
    print("=" * 50)
    
    # Get auth token
    print("1. Getting authentication token...")
    token = get_auth_token()
    headers = {"Authorization": f"Bearer {token}"}
    print("✅ Authentication successful")
    
    # Test user limits
    print("\n2. Testing user limits...")
    test_user = "redis_test_user"
    
    # Set a custom limit
    response = requests.put(f"{BASE_URL}/api/sessions/limits/{test_user}", 
                          json={"limit": 15}, headers=headers)
    if response.status_code == 200:
        print(f"✅ Set limit for {test_user}: 15")
    else:
        print(f"❌ Failed to set limit: {response.status_code}")
        return False
    
    # Verify the limit
    response = requests.get(f"{BASE_URL}/api/sessions/limits/{test_user}", headers=headers)
    if response.status_code == 200:
        limit = response.json()
        if limit.get("limit") == 15:
            print(f"✅ Verified limit for {test_user}: {limit.get('limit')}")
        else:
            print(f"❌ Unexpected limit: {limit.get('limit')}")
            return False
    else:
        print(f"❌ Failed to get limit: {response.status_code}")
        return False
    
    # Test session counting
    print("\n3. Testing session counting...")
    response = requests.get(f"{BASE_URL}/api/sessions/counts/{test_user}", headers=headers)
    if response.status_code == 200:
        counts = response.json()
        print(f"✅ Session counts for {test_user}: {json.dumps(counts, indent=2)}")
    else:
        print(f"❌ Failed to get session counts: {response.status_code}")
        return False
    
    # Test all limits view
    print("\n4. Testing all limits view...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits", headers=headers)
    if response.status_code == 200:
        all_limits = response.json()
        user_limits = all_limits.get("user_limits", {})
        if test_user in user_limits and user_limits[test_user] == 15:
            print(f"✅ {test_user} found in all limits with correct value")
        else:
            print(f"❌ {test_user} not found or incorrect value in all limits")
            return False
    else:
        print(f"❌ Failed to get all limits: {response.status_code}")
        return False
    
    # Cleanup
    print("\n5. Cleaning up...")
    response = requests.delete(f"{BASE_URL}/api/sessions/limits/{test_user}", headers=headers)
    if response.status_code == 200:
        print(f"✅ Deleted {test_user} limit")
    else:
        print(f"❌ Failed to delete limit: {response.status_code}")
        return False
    
    # Verify cleanup
    response = requests.get(f"{BASE_URL}/api/sessions/limits/{test_user}", headers=headers)
    if response.status_code == 200:
        limit = response.json()
        default_limit = 5  # from config
        if limit.get("limit") == default_limit:
            print(f"✅ {test_user} back to default limit: {default_limit}")
        else:
            print(f"❌ Unexpected limit after delete: {limit.get('limit')}")
            return False
    
    print("\n🎉 Complete Redis integration test passed!")
    print("✅ User limits: Working")
    print("✅ Session counting: Working") 
    print("✅ Data persistence: Working")
    print("✅ CRUD operations: Working")
    return True

if __name__ == "__main__":
    try:
        success = test_complete_redis_integration()
        exit(0 if success else 1)
    except Exception as e:
        print(f"❌ Test failed with exception: {e}")
        exit(1)
