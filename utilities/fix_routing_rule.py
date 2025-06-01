#!/usr/bin/env python3
"""
Fix routing rule URI format
"""

import sys
sys.path.append('proto/gen')
import grpc
from b2bua.v1.b2bua_pb2_grpc import RoutingRuleServiceStub
from b2bua.v1 import b2bua_pb2

def fix_routing_rule():
    print('📝 Updating routing rule with proper SIP URI format...')

    channel = grpc.insecure_channel('localhost:50051')
    client = RoutingRuleServiceStub(channel)

    # Update the routing rule with proper SIP URI
    try:
        request = b2bua_pb2.UpdateRoutingRuleRequest(
            rule=b2bua_pb2.RoutingRule(
                rule_id='test-routing',
                name='Test Routing Rule Fixed',
                description='Route test calls from 787 to 999 (with proper SIP URI)',
                priority=100,
                enabled=True,
                conditions=b2bua_pb2.RoutingConditions(
                    from_uri_regex='^sip:787@.*',
                    to_uri_regex='^sip:999@.*'
                ),
                actions=b2bua_pb2.RoutingActions(
                    next_hop_uri='sip:999@127.0.0.1:5060'  # Fixed: added sip: scheme
                )
            )
        )
        
        response = client.UpdateRoutingRule(request)
        print(f'✅ Updated routing rule: {response.rule.rule_id}')
        print(f'   Next hop: {response.rule.actions.next_hop_uri}')
        return True
        
    except Exception as e:
        print(f'❌ Failed to update routing rule: {e}')
        return False

if __name__ == "__main__":
    fix_routing_rule()
