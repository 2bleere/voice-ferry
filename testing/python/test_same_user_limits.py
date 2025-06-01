#!/usr/bin/env python3
"""
Test session limits for the same user.
This script properly tests session limits by using the same user for multiple calls.
"""

import sys
import grpc
import time

sys.path.append('proto/gen')
from b2bua.v1.b2bua_pb2_grpc import B2BUACallServiceStub
from b2bua.v1 import b2bua_pb2

class SessionLimitTest:
    def __init__(self):
        self.grpc_channel = grpc.insecure_channel('localhost:50051')
        self.call_client = B2BUACallServiceStub(self.grpc_channel)
        
    def get_active_calls(self):
        """Get list of active calls."""
        try:
            request = b2bua_pb2.GetActiveCallsRequest()
            response = self.call_client.GetActiveCalls(request)
            calls = list(response)
            return calls
        except Exception as e:
            print(f"❌ Error getting active calls: {e}")
            return []
    
    def initiate_call(self, call_num, from_user="user787", to_user="user999"):
        """Initiate a call for the same user."""
        try:
            request = b2bua_pb2.InitiateCallRequest(
                from_uri=f"sip:{from_user}@127.0.0.1",
                to_uri=f"sip:{to_user}@127.0.0.1",
                initial_sdp="v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\n",
                custom_headers={},
                routing_rule_id=""
            )
            
            response = self.call_client.InitiateCall(request)
            return response.call_id, response.status
            
        except grpc.RpcError as e:
            # Check if this is a session limit error
            if "session limit exceeded" in str(e).lower():
                print(f"📞 Call {call_num}: ❌ REJECTED - Session limit exceeded for {from_user}")
                return None, "REJECTED"
            else:
                print(f"📞 Call {call_num}: ❌ gRPC Error: {e}")
                return None, "ERROR"
        except Exception as e:
            print(f"📞 Call {call_num}: ❌ Error: {e}")
            return None, "ERROR"
    
    def terminate_call(self, call_id):
        """Terminate a call."""
        try:
            request = b2bua_pb2.TerminateCallRequest(
                call_id=call_id,
                reason="Test cleanup"
            )
            response = self.call_client.TerminateCall(request)
            return response.success
        except Exception as e:
            print(f"❌ Error terminating call {call_id}: {e}")
            return False
    
    def test_session_limits(self):
        """Test session limits for the same user."""
        print("🧪 Testing Session Limits for Same User")
        print("=" * 60)
        
        # Check initial state
        initial_calls = self.get_active_calls()
        print(f"📊 Initial active calls: {len(initial_calls)}")
        
        # Test with same user (user787)
        same_user = "user787"
        max_calls_to_test = 6  # Test beyond the limit of 3
        
        print(f"\n🚀 Testing {max_calls_to_test} calls from same user: {same_user}")
        print(f"Expected limit: 3 sessions per user")
        print(f"Expected action: reject calls 4, 5, and 6\n")
        
        successful_calls = []
        rejected_calls = []
        
        for i in range(max_calls_to_test):
            call_num = i + 1
            print(f"📞 Initiating call {call_num} from {same_user}...")
            
            call_id, status = self.initiate_call(call_num, same_user)
            
            if call_id:
                successful_calls.append(call_id)
                print(f"   ✅ SUCCESS - Call ID: {call_id}, Status: {status}")
            elif status == "REJECTED":
                rejected_calls.append(call_num)
                print(f"   ❌ REJECTED - Session limit enforced")
            else:
                print(f"   ❌ FAILED - {status}")
            
            # Check current active calls
            current_calls = self.get_active_calls()
            print(f"   📊 Active calls: {len(current_calls)}")
            
            time.sleep(0.5)  # Small delay between calls
        
        print(f"\n📈 Results Summary:")
        print(f"   Successful calls: {len(successful_calls)}")
        print(f"   Rejected calls: {len(rejected_calls)}")
        print(f"   Final active calls: {len(self.get_active_calls())}")
        
        # Analyze results
        print(f"\n🔍 Analysis:")
        if len(successful_calls) == 3 and len(rejected_calls) == 3:
            print("   ✅ Session limits working perfectly!")
            print("   ✅ Exactly 3 calls accepted, 3 calls rejected")
        elif len(successful_calls) <= 3 and len(rejected_calls) > 0:
            print("   ✅ Session limits appear to be working")
            print(f"   ✅ {len(successful_calls)} calls accepted, {len(rejected_calls)} calls rejected")
        else:
            print("   ❌ Session limits may not be working as expected")
            if len(rejected_calls) == 0:
                print("   ❌ No calls were rejected - limits may not be enforced")
        
        # Cleanup
        if successful_calls:
            print(f"\n🧹 Cleaning up {len(successful_calls)} successful calls...")
            for call_id in successful_calls:
                success = self.terminate_call(call_id)
                print(f"   🔚 Terminated {call_id}: {'✅' if success else '❌'}")
        
        # Final verification
        final_calls = self.get_active_calls()
        print(f"\n📊 Final active calls after cleanup: {len(final_calls)}")
        
        return len(successful_calls) <= 3 and len(rejected_calls) > 0

def main():
    print("🚀 SIP B2BUA Session Limits Test - Same User")
    print("This test properly verifies session limits using the same user")
    print("=" * 70)
    
    try:
        print("Creating test instance...")
        test = SessionLimitTest()
        print("Running session limits test...")
        success = test.test_session_limits()
        
        if success:
            print("\n✅ Session limits test PASSED!")
            return 0
        else:
            print("\n❌ Session limits test FAILED!")
            return 1
            
    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == "__main__":
    sys.exit(main())
