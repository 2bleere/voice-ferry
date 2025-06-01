#!/usr/bin/env python3

import sys
import grpc
sys.path.append('proto/gen')

from b2bua.v1.b2bua_pb2_grpc import RoutingRuleServiceStub
from b2bua.v1 import b2bua_pb2

def create_routing_rule_for_user888():
    """Create a routing rule for calls from user888 to user999."""
    try:
        # Create gRPC channel
        channel = grpc.insecure_channel('localhost:50051')
        client = RoutingRuleServiceStub(channel)
        
        print("📝 Creating routing rule for user888 -> user999...")
        
        # Create routing conditions
        conditions = b2bua_pb2.RoutingConditions(
            from_uri_regex="^sip:user888@.*",  # Match calls from user888
            to_uri_regex="^sip:user999@.*"     # Match calls to user999
        )
        
        # Create routing actions
        actions = b2bua_pb2.RoutingActions(
            next_hop_uri="sip:user999@127.0.0.1:5060"  # Route to local endpoint
        )
        
        # Create routing rule
        rule = b2bua_pb2.RoutingRule(
            rule_id="test-rule-user888",
            name="Test Rule for user888",
            description="Route calls from user888 to user999 for session limits testing",
            priority=100,
            conditions=conditions,
            actions=actions,
            enabled=True
        )
        
        # Create add routing rule request
        request = b2bua_pb2.AddRoutingRuleRequest(rule=rule)
        
        response = client.AddRoutingRule(request)
        print(f"✅ Successfully created routing rule!")
        print(f"   Rule ID: {response.rule.rule_id}")
        print(f"   Name: {response.rule.name}")
        print(f"   From URI Pattern: {response.rule.conditions.from_uri_regex}")
        print(f"   To URI Pattern: {response.rule.conditions.to_uri_regex}")
        print(f"   Next Hop: {response.rule.actions.next_hop_uri}")
        
        return True
        
    except Exception as e:
        if "already exists" in str(e).lower():
            print(f"✅ Routing rule already exists")
            return True
        print(f"❌ Error creating routing rule: {e}")
        return False

def main():
    print("🔧 Adding routing rule for user888\n")
    
    if not create_routing_rule_for_user888():
        return 1
    
    print("\n✅ Routing rule for user888 created successfully!")
    return 0

if __name__ == "__main__":
    sys.exit(main())
