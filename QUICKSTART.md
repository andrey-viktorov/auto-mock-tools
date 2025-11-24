# Quick Start Guide

Get started with Auto Mock Tools in 5 minutes!

## Installation

### Option 1: Run Directly (Recommended)

```bash
# No installation needed!
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest -target http://api.example.com
```

### Option 2: Install Globally

```bash
# Install once, use everywhere
go install github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest
go install github.com/andrey-viktorov/auto-mock-tools/cmd/auto-mock-server@latest
```

## Record Your First Mock

### 1. Start the Proxy

```bash
# If installed
auto-proxy -target https://api.github.com -log-dir mocks

# Or run directly
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest \
  -target https://api.github.com -log-dir mocks
```

The proxy is now running on `http://localhost:8080`

### 2. Make Requests

```bash
# In another terminal
curl http://localhost:8080/users/octocat
```

Mocks are saved to `mocks/default/application_json_*.json`

### 3. Stop the Proxy

Press `Ctrl+C` in the proxy terminal.

## Use Your Mocks

### 1. Start Mock Server

```bash
# If installed
auto-mock-server -mock-dir mocks

# Or run directly
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-mock-server@latest -mock-dir mocks
```

Mock server is now running on `http://localhost:8000`

### 2. Test Your Mocks

```bash
# Same request, but from mocks
curl http://localhost:8000/users/octocat
```

🎉 You're now using mocks instead of the real API!

## Next Steps

### Record Multiple Scenarios

Use the `x-mock-id` header to organize mocks:

```bash
# Start proxy
auto-proxy -target https://api.example.com -log-dir mocks

# Record scenario 1
curl -H "x-mock-id: success-case" http://localhost:8080/api/endpoint

# Record scenario 2
curl -H "x-mock-id: error-case" http://localhost:8080/api/endpoint
```

Use scenarios:

```bash
# Start mock server
auto-mock-server -mock-dir mocks

# Get success case
curl -H "x-mock-id: success-case" http://localhost:8000/api/endpoint

# Get error case
curl -H "x-mock-id: error-case" http://localhost:8000/api/endpoint
```

### Record SSE Streams

```bash
# Start proxy
auto-proxy -target http://sse-server.com -log-dir sse-mocks

# Record SSE
curl -N -H "Accept: text/event-stream" http://localhost:8080/events
```

Replay with timing:

```bash
# Replay with original timing
auto-mock-server -mock-dir sse-mocks -replay-timing

curl -N -H "Accept: text/event-stream" http://localhost:8000/events
```

### Check Stats

```bash
# Get statistics
curl http://localhost:8000/__mock__/stats

# List all mocks
curl http://localhost:8000/__mock__/list
```

## Common Use Cases

### API Testing

Replace real API calls with mocks in tests:

```python
# tests/conftest.py
import pytest
import httpx

@pytest.fixture
def api_client():
    # Point to mock server instead of real API
    return httpx.Client(base_url="http://localhost:8000")

# tests/test_api.py
def test_get_user(api_client):
    response = api_client.get("/users/octocat")
    assert response.status_code == 200
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml
- name: Start Mock Server
  run: |
    auto-mock-server -mock-dir test-mocks &
    sleep 2

- name: Run Tests
  env:
    API_BASE_URL: http://localhost:8000
  run: make test
```

### Development Without Internet

Record mocks once, then develop offline:

```bash
# Once: record production API
./bin/auto-proxy -target https://production-api.com -log-dir prod-mocks

# Daily: use mocks (no internet needed)
./bin/auto-mock-server -mock-dir prod-mocks
```

## Troubleshooting

### Port Already in Use

Change the port:

```bash
./bin/auto-proxy -target https://api.example.com -port 8888
./bin/auto-mock-server -mock-dir mocks -port 9000
```

### No Mocks Found

Check the mock directory:

```bash
ls -la mocks/
```

Ensure you're using the same `x-mock-id` and `Accept` headers.

### Mock Not Matching

Check what's available:

```bash
curl http://localhost:8000/__mock__/list | jq .
```

## Learn More

- [Full Documentation](README.md)
- [Detailed Examples](EXAMPLES.md)
- [Contributing Guide](CONTRIBUTING.md)

## Help

```bash
# Proxy help
auto-proxy -h

# Mock server help
auto-mock-server -h
```

Happy mocking! 🚀
