# MCP Kubernetes Tests

This directory contains tests for the MCP Kubernetes server components.

## Running Tests

You can run the tests using Python's built-in unittest module:

```bash
# Run all tests
python -m unittest discover

# Run a specific test file
python -m unittest tests/test_security.py

# Run a specific test case
python -m unittest tests.test_security.TestSecurityConfig.test_denied_list_takes_precedence
```
