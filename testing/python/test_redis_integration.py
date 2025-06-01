#!/usr/bin/env python3

import requests
import json
import sys

def test_redis_integration():
    """Test SessionLimitsService with Redis backend"""
    
    BASE_URL = "http://localhost:3001"
    
    print("🧪 Testing SessionLimitsService Redis Integration")
    print("=" * 50)
    
    # 1. Authenticate
    print("1. Authenticating...")
    auth_response = requests.post(
        f"{BASE_URL}/api/auth/login",
        json={"username": "admin", "password": "admin123"}
    )
    
    if auth_response.status_code != 200:
        print(f"❌ Authentication failed: {auth_response.status_code}")
        return False
        
    token = auth_response.json()["token"]
    headers = {"Authorization": f"Bearer {token}"}
    print("✅ Authentication successful")
    
    # 2. Get all user limits (should show default)
    print("\n2. Getting all user limits...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits", headers=headers)
    if response.status_code == 200:
        limits = response.json()
        print(f"✅ Current limits: {json.dumps(limits, indent=2)}")
    else:
        print(f"❌ Failed to get limits: {response.status_code}")
        return False
    
    # 3. Set a user limit
    print("\n3. Setting limit for testuser to 12...")
    response = requests.put(
        f"{BASE_URL}/api/sessions/limits/testuser",
        headers=headers,
        json={"limit": 12}
    )
    if response.status_code == 200:
        result = response.json()
        print(f"✅ Set result: {json.dumps(result, indent=2)}")
    else:
        print(f"❌ Failed to set limit: {response.status_code}")
        return False
    
    # 4. Get specific user limit
    print("\n4. Getting testuser's limit...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits/testuser", headers=headers)
    if response.status_code == 200:
        limit = response.json()
        print(f"✅ testuser's limit: {limit}")
        if limit.get("limit") != 12:
            print(f"❌ Expected 12, got {limit.get('limit')}")
            return False
    else:
        print(f"❌ Failed to get user limit: {response.status_code}")
        return False
    
    # 5. Verify it appears in all limits
    print("\n5. Verifying testuser appears in all limits...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits", headers=headers)
    if response.status_code == 200:
        limits = response.json()
        print(f"✅ All limits: {json.dumps(limits, indent=2)}")
        user_limits = limits.get("user_limits", {})
        if "testuser" not in user_limits:
            print("❌ testuser not found in all limits")
            return False
        if user_limits["testuser"] != 12:
            print(f"❌ Expected testuser limit 12, got {user_limits['testuser']}")
            return False
    else:
        print(f"❌ Failed to get all limits: {response.status_code}")
        return False
    
    # 6. Delete user limit
    print("\n6. Deleting testuser's limit...")
    response = requests.delete(f"{BASE_URL}/api/sessions/limits/testuser", headers=headers)
    if response.status_code == 200:
        result = response.json()
        print(f"✅ Delete result: {json.dumps(result, indent=2)}")
    else:
        print(f"❌ Failed to delete limit: {response.status_code}")
        return False
    
    # 7. Verify it's back to default
    print("\n7. Verifying testuser is back to default...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits/testuser", headers=headers)
    if response.status_code == 200:
        limit = response.json()
        print(f"✅ testuser's limit after delete: {limit}")
        expected_default = 5  # from config
        if limit.get("limit") != expected_default:
            print(f"❌ Expected default {expected_default}, got {limit.get('limit')}")
            return False
    else:
        print(f"❌ Failed to get user limit after delete: {response.status_code}")
        return False
    
    # 8. Verify it's removed from all limits
    print("\n8. Verifying testuser is removed from all limits...")
    response = requests.get(f"{BASE_URL}/api/sessions/limits", headers=headers)
    if response.status_code == 200:
        limits = response.json()
        print(f"✅ All limits after delete: {json.dumps(limits, indent=2)}")
        if "testuser" in limits:
            print("❌ testuser still appears in all limits (should be removed)")
            return False
    else:
        print(f"❌ Failed to get all limits after delete: {response.status_code}")
        return False
    
    print("\n🎉 All Redis integration tests passed!")
    print("✅ SessionLimitsService is successfully connected to Redis")
    return True

if __name__ == "__main__":
    try:
        success = test_redis_integration()
        sys.exit(0 if success else 1)
    except Exception as e:
        print(f"❌ Test failed with exception: {e}")
        sys.exit(1)
