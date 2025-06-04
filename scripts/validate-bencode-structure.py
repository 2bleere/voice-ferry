#!/usr/bin/env python3
"""
Deep dive into bencode structure validation.
Test if our bencode is actually valid by parsing it ourselves.
"""

def parse_bencode(data):
    """Parse bencode data to validate structure"""
    try:
        pos = 0
        
        def parse_value():
            nonlocal pos
            if pos >= len(data):
                raise ValueError("Unexpected end of data")
            
            char = data[pos]
            
            if char == 'd':  # Dictionary
                pos += 1
                result = {}
                while pos < len(data) and data[pos] != 'e':
                    key = parse_value()
                    if not isinstance(key, str):
                        raise ValueError("Dictionary keys must be strings")
                    value = parse_value()
                    result[key] = value
                if pos >= len(data):
                    raise ValueError("Unterminated dictionary")
                pos += 1  # Skip 'e'
                return result
                
            elif char == 'l':  # List
                pos += 1
                result = []
                while pos < len(data) and data[pos] != 'e':
                    result.append(parse_value())
                if pos >= len(data):
                    raise ValueError("Unterminated list")
                pos += 1  # Skip 'e'
                return result
                
            elif char == 'i':  # Integer
                pos += 1
                end = data.find('e', pos)
                if end == -1:
                    raise ValueError("Unterminated integer")
                result = int(data[pos:end])
                pos = end + 1
                return result
                
            elif char.isdigit():  # String
                colon = data.find(':', pos)
                if colon == -1:
                    raise ValueError("Invalid string format")
                length = int(data[pos:colon])
                pos = colon + 1
                if pos + length > len(data):
                    raise ValueError("String length exceeds data")
                result = data[pos:pos + length]
                pos += length
                return result
            else:
                raise ValueError(f"Unknown bencode type: {char}")
        
        result = parse_value()
        if pos != len(data):
            raise ValueError(f"Extra data after parsing: {data[pos:]}")
        return result
        
    except Exception as e:
        return f"Parse error: {e}"

def validate_bencode_formats():
    """Test various bencode formats and validate their structure"""
    
    print("=== Bencode Structure Validation ===")
    
    # Test formats from our previous attempts
    test_formats = [
        # Basic format
        'd6:cookie8:testcook7:command18:{"command":"ping"}e',
        
        # Simple cookie
        'd6:cookie4:test7:command18:{"command":"ping"}e',
        
        # Different cookie lengths
        'd6:cookie12:testcookie127:command18:{"command":"ping"}e',
        
        # Minimal ping
        'd6:cookie8:testcook7:command4:pinge',
        
        # RTPEngine specific format attempt
        'd6:cookie10:1234567890e7:command18:{"command":"ping"}e',
        
        # Try without JSON (raw command)
        'd6:cookie8:testcook7:command4:pinge',
    ]
    
    for i, bencode in enumerate(test_formats, 1):
        print(f"\n{i}. Testing: {bencode}")
        print(f"   Length: {len(bencode)} bytes")
        
        # Parse the bencode
        parsed = parse_bencode(bencode)
        
        if isinstance(parsed, dict):
            print("   ✓ Valid bencode structure")
            print(f"   → Parsed: {parsed}")
            
            # Check required fields
            if 'cookie' in parsed and 'command' in parsed:
                print("   ✓ Has required cookie and command")
            else:
                print("   ✗ Missing required fields")
                print(f"   → Keys: {list(parsed.keys())}")
                
        else:
            print(f"   ✗ Invalid bencode: {parsed}")

def test_rtpengine_specific_formats():
    """Test formats specifically mentioned in RTPEngine documentation"""
    
    print("\n=== RTPEngine Specific Format Tests ===")
    
    # Based on RTPEngine documentation and source code analysis
    specific_formats = [
        # Standard NG control format
        ('Standard NG', 'd6:cookie8:12345678e7:command4:pinge'),
        
        # With session info
        ('With call-id', 'd7:call-id8:testcall6:cookie8:12345678e7:command4:pinge'),
        
        # Offer format (common RTPEngine command)
        ('Offer command', 'd6:cookie8:testcook7:command31:{"command":"offer","sdp":"test"}e'),
        
        # Query format 
        ('Query command', 'd6:cookie8:testcook7:command20:{"command":"query"}e'),
    ]
    
    for name, bencode in specific_formats:
        print(f"\n{name}:")
        print(f"  Format: {bencode}")
        
        parsed = parse_bencode(bencode)
        if isinstance(parsed, dict):
            print("  ✓ Valid structure")
            print(f"  → {parsed}")
        else:
            print(f"  ✗ Invalid: {parsed}")

if __name__ == "__main__":
    validate_bencode_formats()
    test_rtpengine_specific_formats()
    
    print("\n=== Analysis ===")
    print("If all formats are structurally valid, the issue might be:")
    print("1. RTPEngine expects a specific cookie format/value")
    print("2. There's a checksum or authentication mechanism")
    print("3. The command field needs specific JSON structure")
    print("4. RTPEngine version incompatibility")
    print("5. Network-level issue (packet corruption)")
