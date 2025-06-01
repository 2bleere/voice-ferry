#!/usr/bin/env python3

import requests
import json
import sys

def test_web_ui_session_limits():
    """Test per-user session limits through the web-ui API"""
    
    base_url = "http://localhost:3001"
    
    print("🚀 Testing Per-User Session Limits via Web-UI API")
    print("=" * 55)
    
    # Step 1: Login to get token
    print("🔑 Step 1: Authentication")
    login_data = {
        "username": "admin",
        "password": "admin123"
    }
    
    try:
        response = requests.post(f"{base_url}/api/auth/login", json=login_data)
        if response.status_code == 200:
            auth_data = response.json()
            if auth_data.get("success"):
                token = auth_data["token"]
                print(f"✅ Login successful, token obtained")
                headers = {"Authorization": f"Bearer {token}"}
            else:
                print(f"❌ Login failed: {auth_data.get('error', 'Unknown error')}")
                return False
        else:
            print(f"❌ Login request failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Login error: {e}")
        return False
    
    # Step 2: Test setting per-user session limits
    print("\n📝 Step 2: Setting Per-User Session Limits")
    test_users = {
        "alice": 5,
        "bob": 10,
        "charlie": 2,
        "diana": 15
    }
    
    success_count = 0
    for username, limit in test_users.items():
        try:
            response = requests.put(
                f"{base_url}/api/sessions/limits/{username}",
                headers=headers,
                json={"limit": limit}
            )
            
            if response.status_code == 200:
                print(f"✅ Set limit for {username}: {limit}")
                success_count += 1
            else:
                print(f"❌ Failed to set limit for {username}: {response.status_code}")
                if response.text:
                    try:
                        error_data = response.json()
                        print(f"   Error: {error_data.get('error', 'Unknown error')}")
                    except:
                        print(f"   Response: {response.text}")
        except Exception as e:
            print(f"❌ Error setting limit for {username}: {e}")
    
    print(f"\n   Successfully set {success_count}/{len(test_users)} user limits")
    
    # Step 3: Test retrieving per-user session limits
    print("\n📊 Step 3: Retrieving Per-User Session Limits")
    retrieved_count = 0
    for username, expected_limit in test_users.items():
        try:
            response = requests.get(
                f"{base_url}/api/sessions/limits/{username}",
                headers=headers
            )
            
            if response.status_code == 200:
                data = response.json()
                actual_limit = data.get("limit")
                if actual_limit == expected_limit:
                    print(f"✅ {username}: {actual_limit} (matches expected)")
                    retrieved_count += 1
                else:
                    print(f"⚠️  {username}: {actual_limit} (expected {expected_limit})")
            else:
                print(f"❌ Failed to get limit for {username}: {response.status_code}")
                if response.text:
                    try:
                        error_data = response.json()
                        print(f"   Error: {error_data.get('error', 'Unknown error')}")
                    except:
                        print(f"   Response: {response.text}")
        except Exception as e:
            print(f"❌ Error getting limit for {username}: {e}")
    
    print(f"\n   Successfully retrieved {retrieved_count}/{len(test_users)} user limits")
    
    # Step 4: Test updating limits
    print("\n🔄 Step 4: Updating User Session Limits")
    update_tests = {
        "alice": 8,
        "bob": 12
    }
    
    updated_count = 0
    for username, new_limit in update_tests.items():
        try:
            response = requests.put(
                f"{base_url}/api/sessions/limits/{username}",
                headers=headers,
                json={"limit": new_limit}
            )
            
            if response.status_code == 200:
                print(f"✅ Updated {username} limit to {new_limit}")
                updated_count += 1
            else:
                print(f"❌ Failed to update limit for {username}: {response.status_code}")
        except Exception as e:
            print(f"❌ Error updating limit for {username}: {e}")
    
    print(f"\n   Successfully updated {updated_count}/{len(update_tests)} user limits")
    
    # Step 5: Test deleting user limits
    print("\n🗑️  Step 5: Deleting User Session Limits")
    delete_tests = ["charlie", "diana"]
    
    deleted_count = 0
    for username in delete_tests:
        try:
            response = requests.delete(
                f"{base_url}/api/sessions/limits/{username}",
                headers=headers
            )
            
            if response.status_code == 200:
                print(f"✅ Deleted limit for {username}")
                deleted_count += 1
            else:
                print(f"❌ Failed to delete limit for {username}: {response.status_code}")
        except Exception as e:
            print(f"❌ Error deleting limit for {username}: {e}")
    
    print(f"\n   Successfully deleted {deleted_count}/{len(delete_tests)} user limits")
    
    # Step 6: Verify deletions
    print("\n🔍 Step 6: Verifying Deletions")
    verification_count = 0
    for username in delete_tests:
        try:
            response = requests.get(
                f"{base_url}/api/sessions/limits/{username}",
                headers=headers
            )
            
            if response.status_code == 404:
                print(f"✅ {username} limit correctly removed")
                verification_count += 1
            elif response.status_code == 200:
                data = response.json()
                if data.get("limit") is None or data.get("error"):
                    print(f"✅ {username} limit correctly removed")
                    verification_count += 1
                else:
                    print(f"❌ {username} limit still exists: {data.get('limit')}")
            else:
                print(f"⚠️  {username} verification unclear: {response.status_code}")
        except Exception as e:
            print(f"❌ Error verifying deletion for {username}: {e}")
    
    print(f"\n   Successfully verified {verification_count}/{len(delete_tests)} deletions")
    
    # Summary
    total_tests = len(test_users) + len(test_users) + len(update_tests) + len(delete_tests) + len(delete_tests)
    total_passed = success_count + retrieved_count + updated_count + deleted_count + verification_count
    success_rate = (total_passed / total_tests) * 100
    
    print(f"\n📋 Test Results Summary:")
    print(f"   Total Tests: {total_tests}")
    print(f"   Passed: {total_passed}")
    print(f"   Success Rate: {success_rate:.1f}%")
    
    if success_rate >= 80:
        print("🎉 Per-user session limits implementation is working well!")
        return True
    else:
        print("⚠️  Some issues detected with per-user session limits implementation")
        return False

if __name__ == "__main__":
    success = test_web_ui_session_limits()
    sys.exit(0 if success else 1)
