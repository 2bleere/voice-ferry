#!/usr/bin/env python3

print("🧪 Starting Session Limits Test")

try:
    import sys
    import grpc
    import time
    
    print("✅ Basic imports successful")
    
    sys.path.append('proto/gen')
    from b2bua.v1.b2bua_pb2_grpc import B2BUACallServiceStub
    from b2bua.v1 import b2bua_pb2
    
    print("✅ gRPC imports successful")
    
    # Test connection
    channel = grpc.insecure_channel('localhost:50051')
    call_client = B2BUACallServiceStub(channel)
    
    print("✅ gRPC client created")
    
    # Test basic API call
    print("📊 Getting active calls...")
    request = b2bua_pb2.GetActiveCallsRequest()
    response = call_client.GetActiveCalls(request)
    calls = list(response)
    print(f"✅ Found {len(calls)} active calls")
    
    # Test session limits by initiating multiple calls
    print("\n🚀 Testing session limits...")
    active_calls = []
    
    for i in range(3):
        print(f"📞 Initiating call {i+1}...")
        
        request = b2bua_pb2.InitiateCallRequest(
            from_uri=f'sip:user{i+787}@127.0.0.1',
            to_uri='sip:user999@127.0.0.1',
            initial_sdp='v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\n',
            custom_headers={},
            routing_rule_id=''
        )
        
        response = call_client.InitiateCall(request)
        active_calls.append(response.call_id)
        
        print(f"   ✅ Call {i+1} initiated: {response.call_id} (Status: {response.status})")
        
        # Check current active calls
        active_request = b2bua_pb2.GetActiveCallsRequest()
        active_response = call_client.GetActiveCalls(active_request)
        current_calls = list(active_response)
        print(f"   📊 Current active calls: {len(current_calls)}")
        
        time.sleep(0.5)
    
    print(f"\n📈 Total initiated calls: {len(active_calls)}")
    
    # Cleanup
    print("\n🧹 Cleaning up calls...")
    for call_id in active_calls:
        terminate_request = b2bua_pb2.TerminateCallRequest(
            call_id=call_id,
            reason='Test cleanup'
        )
        terminate_response = call_client.TerminateCall(terminate_request)
        print(f"   🔚 Terminated {call_id}: {terminate_response.success}")
    
    # Final check
    final_request = b2bua_pb2.GetActiveCallsRequest()
    final_response = call_client.GetActiveCalls(final_request)
    final_calls = list(final_response)
    print(f"\n📊 Final active calls: {len(final_calls)}")
    
    print("\n✅ Session limits test completed!")
    
except Exception as e:
    print(f"\n❌ Error: {e}")
    import traceback
    traceback.print_exc()
