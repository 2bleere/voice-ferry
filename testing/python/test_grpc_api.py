#!/usr/bin/env python3

import sys
import grpc
sys.path.append('proto/gen')

from b2bua.v1.b2bua_pb2_grpc import RoutingRuleServiceStub
from b2bua.v1 import b2bua_pb2

def test_grpc_connection():
    """Test gRPC connection and list existing routing rules."""
    try:
        # Create gRPC channel
        channel = grpc.insecure_channel('localhost:50051')
        client = RoutingRuleServiceStub(channel)
        
        # Test connection by listing routing rules
        print("🔗 Testing gRPC connection...")
        response = client.ListRoutingRules(b2bua_pb2.ListRoutingRulesRequest())
        print(f"✅ Successfully connected to gRPC API!")
        print(f"📋 Found {len(response.rules)} existing routing rules:")
        
        for rule in response.rules:
            print(f"   Rule ID: {rule.rule_id}")
            print(f"   Name: {rule.name}")
            print(f"   Priority: {rule.priority}")
            print(f"   Description: {rule.description}")
            print(f"   Enabled: {rule.enabled}")
            if rule.conditions:
                print(f"   Request URI Regex: {rule.conditions.request_uri_regex}")
                print(f"   From URI Regex: {rule.conditions.from_uri_regex}")
                print(f"   To URI Regex: {rule.conditions.to_uri_regex}")
            if rule.actions:
                print(f"   Next Hop URI: {rule.actions.next_hop_uri}")
            print("   ---")
            
        return client, True
        
    except Exception as e:
        print(f"❌ Error connecting to gRPC API: {e}")
        return None, False

def create_routing_rule(client):
    """Create a routing rule for test calls from 787 to 999."""
    try:
        print("\n📝 Creating routing rule for calls to 999...")
        
        # Create routing conditions
        conditions = b2bua_pb2.RoutingConditions(
            request_uri_regex="^sip:999@.*",  # Match calls to user 999
            from_uri_regex=".*",  # Match any from URI
            to_uri_regex=".*"     # Match any to URI
        )
        
        # Create routing actions
        actions = b2bua_pb2.RoutingActions(
            next_hop_uri="sip:999@127.0.0.1:5060"  # Route to local endpoint
        )
        
        # Create routing rule
        rule = b2bua_pb2.RoutingRule(
            rule_id="test-rule-999",
            name="Test Rule for 999",
            description="Route calls to 999 for session limits testing",
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
        print(f"   Priority: {response.rule.priority}")
        
        return True
        
    except Exception as e:
        print(f"❌ Error creating routing rule: {e}")
        return False

def main():
    print("🧪 Testing gRPC API and creating routing rules for session limits test\n")
    print("📍 Starting main function...")
    
    # Test connection
    client, success = test_grpc_connection()
    if not success:
        print("❌ Connection test failed")
        return 1
    
    # Create routing rule if needed
    if not create_routing_rule(client):
        print("❌ Routing rule creation failed")
        return 1
    
    print("\n✅ gRPC API test completed successfully!")
    print("🚀 Ready to run session limits test!")
    return 0

if __name__ == "__main__":
    sys.exit(main())
