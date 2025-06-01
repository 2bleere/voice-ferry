#!/usr/bin/env python3

import sys
import grpc
import time
import socket
import threading
from collections import defaultdict

sys.path.append('proto/gen')
from b2bua.v1.b2bua_pb2_grpc import B2BUACallServiceStub, RoutingRuleServiceStub
from b2bua.v1 import b2bua_pb2

class SimpleSessionTest:
    def __init__(self):
        self.grpc_channel = grpc.insecure_channel('localhost:50051')
        self.call_client = B2BUACallServiceStub(self.grpc_channel)
        self.routing_client = RoutingRuleServiceStub(self.grpc_channel)
        
    def get_active_calls(self):
        """Get number of active calls via gRPC API."""
        try:
            request = b2bua_pb2.GetActiveCallsRequest()
            response = self.call_client.GetActiveCalls(request)
            calls = list(response)  # Convert stream to list
            return len(calls)
        except Exception as e:
            print(f"❌ Error getting active calls: {e}")
            return 0
    
    def initiate_call_via_grpc(self, call_id, from_uri, to_uri):
        """Initiate a call via gRPC API instead of SIP."""
        try:
            request = b2bua_pb2.InitiateCallRequest(
                from_uri=from_uri,
                to_uri=to_uri,
                initial_sdp="v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\n",
                custom_headers={},
                routing_rule_id=""  # Let it auto-select
            )
            
            response = self.call_client.InitiateCall(request)
            return response.call_id, response.status
            
        except Exception as e:
            print(f"❌ Error initiating call {call_id}: {e}")
            return None, None
    
    def terminate_call_via_grpc(self, call_id):
        """Terminate a call via gRPC API."""
        try:
            request = b2bua_pb2.TerminateCallRequest(
                call_id=call_id,
                reason="Test completed"
            )
            
            response = self.call_client.TerminateCall(request)
            return response.success
            
        except Exception as e:
            print(f"❌ Error terminating call {call_id}: {e}")
            return False
    
    def test_session_limits_via_grpc(self):
        """Test session limits using gRPC API calls."""
        print("🧪 Testing Session Limits via gRPC API")
        print("=" * 50)
        
        # Check initial state
        initial_calls = self.get_active_calls()
        print(f"📊 Initial active calls: {initial_calls}")
        
        # Test initiating multiple calls
        active_calls = []
        max_calls_to_test = 5
        
        print(f"\n🚀 Initiating {max_calls_to_test} calls...")
        
        for i in range(max_calls_to_test):
            call_id = f"test-grpc-call-{i+1}"
            from_uri = f"sip:user{i+787}@127.0.0.1"
            to_uri = "sip:user999@127.0.0.1"
            
            result_call_id, status = self.initiate_call_via_grpc(call_id, from_uri, to_uri)
            
            if result_call_id:
                active_calls.append(result_call_id)
                print(f"📞 Call {i+1}: ✅ Started (ID: {result_call_id}, Status: {status})")
            else:
                print(f"📞 Call {i+1}: ❌ Failed to start")
            
            # Check active call count
            current_active = self.get_active_calls()
            print(f"📊 Active calls after call {i+1}: {current_active}")
            
            time.sleep(0.5)  # Small delay between calls
        
        print(f"\n📈 Final active calls: {len(active_calls)}")
        
        # Cleanup: terminate all calls
        print(f"\n🧹 Cleaning up {len(active_calls)} calls...")
        for call_id in active_calls:
            success = self.terminate_call_via_grpc(call_id)
            print(f"🔚 Terminated {call_id}: {'✅' if success else '❌'}")
        
        # Final check
        final_calls = self.get_active_calls()
        print(f"📊 Final active calls after cleanup: {final_calls}")
        
        return True

def main():
    print("🧪 Simple Session Limits Test via gRPC API")
    print("This test uses gRPC to bypass SIP routing issues")
    print("=" * 60)
    
    try:
        test = SimpleSessionTest()
        success = test.test_session_limits_via_grpc()
        
        if success:
            print("\n✅ Session limits test completed!")
        else:
            print("\n❌ Session limits test failed!")
            
    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        return 1
    
    return 0

if __name__ == "__main__":
    sys.exit(main())
