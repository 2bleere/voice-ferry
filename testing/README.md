# Testing Directory

This directory contains all test files organized by type:

## Subdirectories

### `python/`
Contains Python test scripts:
- Session limit tests
- Redis integration tests
- SIP client tests
- Web UI tests
- Authentication tests
- Comprehensive test suites

### `shell/`
Contains shell script tests:
- SIP authentication tests
- Integration tests
- Register tests

### `sip/`
Contains SIP configuration files:
- Test SIP configurations
- Authentication test files
- Register test configurations

### `javascript/`
Contains JavaScript/Node.js test files:
- Web UI test scripts
- Server test utilities
- Node.js integration tests

### `logs/`
Contains test log files and output from various test runs.

## Usage

Run tests from the project root directory. Most Python tests can be executed directly:

```bash
# Run specific test
python testing/python/test_session_limits.py

# Run shell tests
bash testing/shell/test_sip_auth.sh
```

## Note

These tests are designed to work with the Voice Ferry development environment. Ensure all services (Redis, etcd, B2BUA) are running before executing tests.
