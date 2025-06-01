#!/usr/bin/env python3
"""
Simple test to verify session limits are working.
"""

import grpc
import time
import sys
import traceback

try:
    from b2bua.v1.call_service_pb2 import InitiateCallRequest, TerminateCallRequest
    from b2bua.v1.call_service_pb2_grpc import B2BUACallServiceStub
    print("✅ Successfully imported gRPC modules")
except ImportError as e:
    print(f"❌ Failed to import gRPC modules: {e}")
    print("Make sure protobuf files are generated")
    sys.exit(1)

def test_session_limits():
    """Test session limits with basic error handling."""
    print("🧪 Testing Session Limits")
    print("=" * 40)
    
    try:
        # Connect to server
        channel = grpc.insecure_channel("localhost:50051")
        stub = B2BUACallServiceStub(channel)
        print("✅ Connected to B2BUA server")
        
        # Test calls
        username = "user787"
        active_calls = []
        
        for i in range(6):
            try:
                request = InitiateCallRequest(
                    from_uri=f"sip:{username}@127.0.0.1",
                    to_uri="sip:user999@127.0.0.1",
                    sdp="v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\nm=audio 5004 RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\n"
                )
                
                response = stub.InitiateCall(request)
                
                if response.call_id:
                    active_calls.append(response.call_id)
                    print(f"📞 Call {i+1}: ✅ SUCCESS ({response.call_id})")
                else:
                    print(f"📞 Call {i+1}: ❌ FAILED (no call ID)")
                    
            except grpc.RpcError as e:
                if "session limit exceeded" in str(e.details()).lower():
                    print(f"📞 Call {i+1}: ❌ REJECTED (session limit)")
                else:
                    print(f"📞 Call {i+1}: ❌ ERROR: {e.details()}")
            except Exception as e:
                print(f"📞 Call {i+1}: ❌ EXCEPTION: {e}")
            
            time.sleep(0.2)
        
        print(f"\n📊 Results: {len(active_calls)} successful calls")
        
        # Cleanup
        for call_id in active_calls:
            try:
                stub.TerminateCall(TerminateCallRequest(call_id=call_id))
            except:
                pass
        
        channel.close()
        print("🧹 Cleanup completed")
        
        return len(active_calls) <= 3
        
    except Exception as e:
        print(f"❌ Test failed: {e}")
        traceback.print_exc()
        return False

if __name__ == "__main__":
    print("🚀 Session Limits Test")
    success = test_session_limits()
    print(f"\n{'✅ PASSED' if success else '❌ FAILED'}")
    sys.exit(0 if success else 1)
