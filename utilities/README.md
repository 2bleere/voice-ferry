# Utilities Directory

This directory contains utility scripts and helper tools for the Voice Ferry project.

## Files

### User Management
- `add_user787_rule.py` - Script to add user 787 routing rule
- `add_user888_rule.py` - Script to add user 888 routing rule

### Routing Fixes
- `fix_next_hop_uris.py` - Utility to fix next hop URI configurations
- `fix_routing_rule.py` - General routing rule fix utility

### Testing Utilities
- `quick_test.py` - Quick functionality test script
- `verify_session_limits.py` - Session limit verification utility
- `simple_session_test.py` - Simple session testing script
- `simple_sip_test.py` - Basic SIP functionality test

### Development Tools
- `simple_auth_test.sh` - Simple authentication testing
- `simple_sip_test.sh` - Basic SIP testing script

## Usage

Most utilities can be run directly from the project root:

```bash
# Add user routing rules
python utilities/add_user787_rule.py
python utilities/add_user888_rule.py

# Run quick tests
python utilities/quick_test.py

# Verify session limits
python utilities/verify_session_limits.py
```

## Note

These utilities are designed to work with the Voice Ferry development environment. Ensure proper configuration and running services before executing utilities.
