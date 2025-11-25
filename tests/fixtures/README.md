# Test Fixtures

This directory contains YAML configuration files used in unit tests for scenario-based mock routing.

## Files

- `mock-example.yml` - Example scenario configuration used in handler and storage tests
- `test-jitter-original.yml` - SSE jitter test with original timing
- `test-jitter-override.yml` - SSE jitter test with delay override
- `test-sse-delay-override.yml` - SSE stream with timing override

## Usage in Tests

These configuration files are referenced from unit tests in `pkg/handlers/` and `pkg/storage/`.
They define scenarios that route requests based on path, method, and JSON body filters.

The response files referenced in these configs are located in `../../test_mocks/`.
