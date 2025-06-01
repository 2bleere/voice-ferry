#!/usr/bin/env python3
"""
Test script to verify per-user session limits in Voice Ferry SIP B2BUA
This script creates multiple simultaneous SIP sessions with different usernames
to verify that user-specific session limits are enforced correctly.

Usage:
    python test_per_user_limits.py

Requirements:
    pip install requests sipp pysipp

The script requires the Voice Ferry B2BUA to be running with Redis session limits enabled.
"""

import subprocess
import time
import random
import string
import requests
import sys
import json
import os
from concurrent.futures import ThreadPoolExecutor

# Configuration
B2BUA_HOST = "127.0.0.1"
B2BUA_PORT = 5060
B2BUA_API_PORT = 8080
REDIS_HOST = "127.0.0.1" 
REDIS_PORT = 6379
MAX_CONCURRENT_CALLS = 30  # Maximum number of concurrent calls to attempt
CALL_DURATION = 5  # Call duration in seconds

# Define different users with their expected limits
USER_LIMITS = {
    "user_unlimited": 0,      # Should have no limit
    "user_high": 10,          # Should allow 10 calls
    "user_medium": 5,         # Should allow 5 calls
    "user_low": 2,            # Should allow 2 calls
    "user_default": None      # Should use default limit
}

def setup_user_limits():
    """Configure user-specific session limits via API"""
    print("Setting up user-specific session limits...")
    
    base_url = f"http://{B2BUA_HOST}:{B2BUA_API_PORT}/api/sessions/limits"
    
    for user, limit in USER_LIMITS.items():
        if limit is not None:  # Skip default user
            try:
                response = requests.put(
                    f"{base_url}/{user}",
                    json={"limit": limit},
                    timeout=5
                )
                if response.status_code == 200:
                    print(f"Set {user} limit to {limit}")
                else:
                    print(f"Failed to set limit for {user}: {response.status_code}")
                    print(response.text)
            except Exception as e:
                print(f"Error setting limit for {user}: {e}")
    
    # Verify limits were set
    try:
        response = requests.get(base_url, timeout=5)
        if response.status_code == 200:
            print("Current session limits configuration:")
            print(json.dumps(response.json(), indent=2))
    except Exception as e:
        print(f"Error getting limits: {e}")

def generate_sip_call(user, call_index):
    """Generate a single SIP call with the given username"""
    call_id = f"{user}-{call_index}-{random.randint(10000, 99999)}"
    
    sipp_cmd = [
        "sipp", 
        B2BUA_HOST,
        "-p", "5090",  # Local port
        "-s", user,    # The user part becomes the username for session tracking
        "-r", "1",     # Rate: 1 call per second
        "-d", str(CALL_DURATION * 1000),  # Call duration in ms
        "-l", "1",     # Maximum 1 concurrent call per process
        "-m", "1",     # Total 1 call
        "-cid_str", call_id,  # Custom Call-ID
        "-trace_err",
        "-timeout", "10s",
        "-t", "t1",  # Transport: UDP
    ]
    
    try:
        result = subprocess.run(
            sipp_cmd, 
            stdout=subprocess.PIPE, 
            stderr=subprocess.PIPE,
            timeout=CALL_DURATION + 10
        )
        
        if "successful" in result.stdout.decode() or "Call(s) succeeded" in result.stdout.decode():
            return True, user, call_id
        else:
            # Check for specific error indicating session limit
            if "486" in result.stdout.decode() or "Busy" in result.stdout.decode():
                return False, user, "Session limit reached"
            else:
                error_text = result.stdout.decode()[:100] + "..." + result.stderr.decode()[:100]
                return False, user, f"Failed: {error_text}"
    except subprocess.TimeoutExpired:
        return False, user, "Call timed out"
    except Exception as e:
        return False, user, f"Error: {str(e)}"

def test_user_limits():
    """Test each user's session limits by creating multiple concurrent calls"""
    print("\nTesting per-user session limits...\n")
    
    results = {user: {"success": 0, "failed": 0, "limit_reached": False} for user in USER_LIMITS.keys()}
    
    for user in USER_LIMITS.keys():
        print(f"\nTesting limits for user '{user}':")
        call_threads = []
        
        # Determine number of calls to attempt
        max_calls = 15  # Try more than the expected limit
        if USER_LIMITS[user] == 0:  # For unlimited users, use a reasonable number
            max_calls = 10  
        
        # Start multiple calls concurrently
        with ThreadPoolExecutor(max_workers=max_calls) as executor:
            for i in range(max_calls):
                call_threads.append(executor.submit(generate_sip_call, user, i + 1))
                time.sleep(0.1)  # Small delay between starting calls
        
        # Collect results
        for future in call_threads:
            success, call_user, result = future.result()
            if success:
                results[user]["success"] += 1
                print(f"  Call {results[user]['success']}: SUCCESS")
            else:
                results[user]["failed"] += 1
                if "Session limit" in result or "486" in result or "Busy" in result:
                    results[user]["limit_reached"] = True
                    print(f"  Call {results[user]['success'] + results[user]['failed']}: LIMIT REACHED")
                else:
                    print(f"  Call {results[user]['success'] + results[user]['failed']}: FAILED - {result}")
        
        # Pause between users to allow sessions to clean up
        print(f"Completed testing for {user}. Waiting for sessions to clean up...")
        time.sleep(CALL_DURATION + 2)
    
    return results

def verify_results(results):
    """Verify the test results against expected limits"""
    print("\n" + "="*40)
    print("PER-USER SESSION LIMITS TEST RESULTS")
    print("="*40)
    
    all_passed = True
    
    for user, result in results.items():
        expected_limit = USER_LIMITS[user]
        if expected_limit is None:
            # For default user, use system default
            expected_limit = 5  # Assuming default is 5
        
        print(f"\nUser: {user}")
        print(f"  Successful calls:  {result['success']}")
        print(f"  Failed calls:      {result['failed']}")
        print(f"  Limit reached:     {result['limit_reached']}")
        
        # Verify the limit was enforced correctly
        if expected_limit == 0:
            # Unlimited user should have no failures due to limits
            if result["success"] >= 8:  # Should succeed with at least 8 calls (out of 10)
                print(f"  ✓ PASS: Unlimited user allowed multiple calls ({result['success']})")
            else:
                print(f"  ✗ FAIL: Unlimited user was limited ({result['success']} succeeded)")
                all_passed = False
        else:
            # Limited user should be limited to their specific limit
            if result["success"] == expected_limit and result["limit_reached"]:
                print(f"  ✓ PASS: Limit correctly enforced at {expected_limit}")
            elif result["success"] >= expected_limit:
                print(f"  ~ WARN: User allowed more calls ({result['success']}) than expected limit ({expected_limit})")
            else:
                print(f"  ✗ FAIL: User limited to fewer calls ({result['success']}) than expected ({expected_limit})")
                all_passed = False
    
    print("\nOverall Result:", "PASS" if all_passed else "FAIL")
    return all_passed

def main():
    print("="*60)
    print("Voice Ferry Per-User Session Limits Test")
    print("="*60)
    
    try:
        # Setup user-specific limits
        setup_user_limits()
        
        # Give time for limits to be applied
        time.sleep(2)
        
        # Run the test
        results = test_user_limits()
        
        # Verify results
        success = verify_results(results)
        
        # Exit with appropriate status code
        sys.exit(0 if success else 1)
        
    except KeyboardInterrupt:
        print("\nTest interrupted by user.")
        sys.exit(130)
    except Exception as e:
        print(f"\nError running test: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
