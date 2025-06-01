#!/usr/bin/env python3

import grpc
import sys
sys.path.append('proto/gen')
from b2bua.v1.b2bua_pb2_grpc import B2BUACallServiceStub
from b2bua.v1 import b2bua_pb2

def test_call_apis():
    try:
        print('🧪 Testing B2BUA Call APIs...')
        channel = grpc.insecure_channel('localhost:50051')
        call_client = B2BUACallServiceStub(channel)
        
        # Test initiating a call
        request = b2bua_pb2.InitiateCallRequest(
            from_uri='sip:user787@127.0.0.1',
            to_uri='sip:user999@127.0.0.1',
            initial_sdp='v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\n',
            custom_headers={},
            routing_rule_id=''
        )
        
        print('Making InitiateCall request...')
        response = call_client.InitiateCall(request)
        
        print(f'✅ Call initiated successfully!')
        print(f'   Call ID: {response.call_id}')
        print(f'   Leg ID: {response.leg_id}')
        print(f'   Status: {response.status}')
        
        # Now check active calls
        print('\n📊 Checking active calls...')
        active_request = b2bua_pb2.GetActiveCallsRequest()
        active_response = call_client.GetActiveCalls(active_request)
        calls = list(active_response)
        print(f'Found {len(calls)} active calls')
        
        # Test terminating the call
        print(f'\n🔚 Terminating call {response.call_id}...')
        terminate_request = b2bua_pb2.TerminateCallRequest(
            call_id=response.call_id,
            reason='Test completed'
        )
        terminate_response = call_client.TerminateCall(terminate_request)
        print(f'Terminate success: {terminate_response.success}')
        print(f'Message: {terminate_response.message}')
        
        # Final check of active calls
        print('\n📊 Final check of active calls...')
        final_active_response = call_client.GetActiveCalls(active_request)
        final_calls = list(final_active_response)
        print(f'Found {len(final_calls)} active calls after termination')
        
        return True
        
    except Exception as e:
        print(f'❌ Error: {e}')
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    success = test_call_apis()
    if success:
        print('\n✅ All tests passed!')
    else:
        print('\n❌ Tests failed!')
