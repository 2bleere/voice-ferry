#!/usr/bin/env python3

import sys
import grpc
sys.path.append('proto/gen')

from b2bua.v1.b2bua_pb2_grpc import RoutingRuleServiceStub
from b2bua.v1 import b2bua_pb2

def fix_next_hop_uris():
    """Fix the next hop URIs to use simpler format without user part."""
    try:
        # Create gRPC channel
        channel = grpc.insecure_channel('localhost:50051')
        client = RoutingRuleServiceStub(channel)
        
        print("🔧 Fixing next hop URIs to remove user part...")
        
        # List current rules
        response = client.ListRoutingRules(b2bua_pb2.ListRoutingRulesRequest())
        
        rules_to_update = []
        for rule in response.rules:
            if "@127.0.0.1" in rule.actions.next_hop_uri:
                rules_to_update.append(rule)
                
        print(f"📋 Found {len(rules_to_update)} rules to update")
        
        for rule in rules_to_update:
            old_uri = rule.actions.next_hop_uri
            # Change from sip:user999@127.0.0.1:5060 to sip:127.0.0.1:5060
            new_uri = "sip:127.0.0.1:5060"
            
            print(f"\n🔧 Updating rule: {rule.rule_id}")
            print(f"   Old URI: {old_uri}")
            print(f"   New URI: {new_uri}")
            
            # Create updated rule
            updated_rule = b2bua_pb2.RoutingRule(
                rule_id=rule.rule_id,
                name=rule.name,
                description=rule.description,
                priority=rule.priority,
                conditions=rule.conditions,
                actions=b2bua_pb2.RoutingActions(next_hop_uri=new_uri),
                enabled=rule.enabled
            )
            
            # Update the rule
            update_request = b2bua_pb2.UpdateRoutingRuleRequest(rule=updated_rule)
            
            try:
                client.UpdateRoutingRule(update_request)
                print(f"   ✅ Successfully updated")
            except Exception as e:
                print(f"   ❌ Error updating: {e}")
        
        return True
        
    except Exception as e:
        print(f"❌ Error fixing URIs: {e}")
        return False

def main():
    print("🔧 Fixing Next Hop URI formats\n")
    
    if not fix_next_hop_uris():
        return 1
    
    print("\n✅ All next hop URIs fixed!")
    return 0

if __name__ == "__main__":
    sys.exit(main())
