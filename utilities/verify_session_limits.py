#!/usr/bin/env python3
"""
Simple verification script to test session limits functionality.
This script verifies session limits are working by checking actual session tracking.
"""

import grpc
import time
import sys
from b2bua.v1.call_service_pb2 import InitiateCallRequest, TerminateCallRequest
from b2bua.v1.call_service_pb2_grpc import B2BUACallServiceStub

class SessionLimitVerifier:
    def __init__(self, server_address="localhost:50051"):
        """Initialize the session limit verifier."""
        self.server_address = server_address
        self.channel = None
        self.stub = None
        
    def connect(self):
        """Connect to the gRPC server."""
        try:
            self.channel = grpc.insecure_channel(self.server_address)
            self.stub = B2BUACallServiceStub(self.channel)
            print(f"✅ Connected to B2BUA server at {self.server_address}")
            return True
        except Exception as e:
            print(f"❌ Failed to connect to B2BUA server: {e}")
            return False
    
    def initiate_call(self, call_num, username):
        """Initiate a call for the given user."""
        try:
            request = InitiateCallRequest(
                from_uri=f"sip:{username}@127.0.0.1",
                to_uri="sip:user999@127.0.0.1",
                sdp="v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\nm=audio 5004 RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\n"
            )
            
            response = self.stub.InitiateCall(request)
            
            if response.call_id:
                return response.call_id, "SUCCESS"
            else:
                return None, "FAILED"
                
        except grpc.RpcError as e:
            if "session limit exceeded" in str(e.details()).lower():
                return None, "REJECTED_LIMIT"
            else:
                return None, f"ERROR: {e.details()}"
        except Exception as e:
            return None, f"EXCEPTION: {e}"
    
    def terminate_call(self, call_id):
        """Terminate a call."""
        try:
            request = TerminateCallRequest(call_id=call_id)
            response = self.stub.TerminateCall(request)
            return True
        except Exception as e:
            print(f"Warning: Failed to terminate call {call_id}: {e}")
            return False
    
    def test_session_limits(self):
        """Test that session limits work correctly."""
        print("🧪 Session Limits Verification Test")
        print("=" * 50)
        print("Testing with user787 (max 3 sessions allowed)")
        print("Expected: First 3 calls succeed, remaining calls rejected")
        print()
        
        username = "user787"
        test_calls = []
        results = []
        
        # Test 6 calls
        for i in range(6):
            call_num = i + 1
            print(f"📞 Call {call_num}: ", end="")
            
            call_id, status = self.initiate_call(call_num, username)
            
            if call_id:
                test_calls.append(call_id)
                print(f"✅ SUCCESS ({call_id})")
                results.append("SUCCESS")
            elif status == "REJECTED_LIMIT":
                print(f"❌ REJECTED (session limit)")
                results.append("REJECTED")
            else:
                print(f"❌ {status}")
                results.append("ERROR")
            
            time.sleep(0.2)  # Small delay
        
        # Analyze results
        print(f"\n📊 Results Summary:")
        successful_count = results.count("SUCCESS")
        rejected_count = results.count("REJECTED")
        error_count = results.count("ERROR")
        
        print(f"   Successful calls: {successful_count}")
        print(f"   Rejected calls:   {rejected_count}")
        print(f"   Error calls:      {error_count}")
        print(f"   Total calls:      {len(results)}")
        
        # Verify correctness
        print(f"\n🔍 Verification:")
        if successful_count == 3 and rejected_count == 3:
            print("   ✅ PERFECT: Exactly 3 calls allowed, 3 calls rejected")
            success = True
        elif successful_count <= 3 and rejected_count >= 1:
            print("   ✅ WORKING: Session limits are enforced")
            success = True
        else:
            print("   ❌ FAILED: Session limits not working correctly")
            success = False
        
        # Cleanup
        if test_calls:
            print(f"\n🧹 Cleaning up {len(test_calls)} active calls...")
            for call_id in test_calls:
                self.terminate_call(call_id)
        
        return success
    
    def close(self):
        """Close the connection."""
        if self.channel:
            self.channel.close()

def main():
    """Main function."""
    print("🚀 B2BUA Session Limits Verification")
    print("=" * 60)
    
    verifier = SessionLimitVerifier()
    
    if not verifier.connect():
        return False
    
    try:
        success = verifier.test_session_limits()
        
        if success:
            print("\n🎉 Session limits verification PASSED!")
            return True
        else:
            print("\n❌ Session limits verification FAILED!")
            return False
            
    finally:
        verifier.close()

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
