# Examples

This document provides detailed examples of using Auto Mock Tools in various scenarios.

## Table of Contents

- [Basic Recording and Playback](#basic-recording-and-playback)
- [Multiple Scenarios](#multiple-scenarios)
- [SSE Recording and Replay](#sse-recording-and-replay)
- [HTTPS and mTLS](#https-and-mtls)
- [Scenario-Based Filtering](#scenario-based-filtering)
- [Integration with Python httpx](#integration-with-python-httpx)
- [Docker Integration](#docker-integration)

## Basic Recording and Playback

### Recording API Traffic

```bash
# Start the proxy
auto-proxy -target https://api.github.com -log-dir github-mocks -port 8080

# In another terminal, make requests through the proxy
curl http://localhost:8080/users/octocat
curl http://localhost:8080/repos/httpx-record/auto-tools-go

# Stop the proxy (Ctrl+C)
```

Mock files will be saved in `github-mocks/default/`.

### Serving Recorded Mocks

```bash
# Start the mock server
auto-mock-server -mock-dir github-mocks -port 8000

# Test the mocks
curl http://localhost:8000/users/octocat
curl http://localhost:8000/repos/httpx-record/auto-tools-go
```

## Multiple Scenarios

### Recording Different Scenarios

```bash
# Start proxy
auto-proxy -target https://api.example.com -log-dir mocks

# Record scenario 1: authenticated user
curl -H "x-mock-id: user-authenticated" \
     http://localhost:8080/api/profile

# Record scenario 2: guest user  
curl -H "x-mock-id: user-guest" \
     http://localhost:8080/api/profile

# Record scenario 3: admin user
curl -H "x-mock-id: user-admin" \
     http://localhost:8080/api/profile
```

### Using Different Scenarios

```bash
# Start mock server
auto-mock-server -mock-dir mocks

# Test different scenarios
curl -H "x-mock-id: user-authenticated" \
     http://localhost:8000/api/profile

curl -H "x-mock-id: user-guest" \
     http://localhost:8000/api/profile

curl -H "x-mock-id: user-admin" \
     http://localhost:8000/api/profile
```

## SSE Recording and Replay

### Recording SSE Stream

```bash
# Start proxy
auto-proxy -target http://sse-server.com -log-dir sse-mocks

# Record SSE stream
curl -N -H "Accept: text/event-stream" \
     -H "x-mock-id: live-updates" \
     http://localhost:8080/events
```

### Replaying with Original Timing

```bash
# Replay with original timing and 10% jitter
auto-mock-server \
    -mock-dir sse-mocks \
    -replay-timing \
    -jitter 0.1

# Consume the SSE stream
curl -N -H "Accept: text/event-stream" \
     -H "x-mock-id: live-updates" \
     http://localhost:8000/events
```

### Replaying Without Timing (Instant)

```bash
# Replay instantly (no delays)
auto-mock-server -mock-dir sse-mocks

curl -N -H "Accept: text/event-stream" \
     -H "x-mock-id: live-updates" \
     http://localhost:8000/events
```

## HTTPS and mTLS

### Recording from HTTPS API with Client Certificates

```bash
# Start proxy with mTLS
auto-proxy \
    -target https://secure-api.example.com \
    -client-cert /path/to/client.crt \
    -client-key /path/to/client.key \
    -log-dir secure-mocks

# Make requests through proxy
curl http://localhost:8080/api/secure-endpoint
```

## Scenario-Based Filtering

### Create Scenario Configuration

Create `scenarios.yml`:

```yaml
scenarios:
  # Match requests with status "ready"
  - name: Ready Status
    method: POST
    path: /api/status
    filter:
      body:
        eq:
          field: status
          value: ready
    response:
      file: mocks/ready-status.json
      delay: 0.5

  # Match requests with status "pending"
  - name: Pending Status
    method: POST
    path: /api/status
    filter:
      body:
        eq:
          field: status
          value: pending
    response:
      file: mocks/pending-status.json
      delay: 1.0

  # Fallback for any other status
  - name: Default Status
    method: POST
    path: /api/status
    response:
      file: mocks/default-status.json
```

### Use Scenario Configuration

```bash
# Start mock server with scenarios
auto-mock-server \
    -mock-config scenarios.yml \
    -port 8000

# Test different scenarios
curl -X POST http://localhost:8000/api/status \
     -H "Content-Type: application/json" \
     -d '{"status": "ready"}'

curl -X POST http://localhost:8000/api/status \
     -H "Content-Type: application/json" \
     -d '{"status": "pending"}'
```

### Complex JSON Filtering

```yaml
scenarios:
  - name: Complex Filter
    method: POST
    path: /api/process
    filter:
      body:
        and:
          - eq:
              field: type
              value: payment
          - rx:
              field: transaction.id
              value: ^TXN-[0-9]{6}$
          - gt:
              field: amount
              value: 100
    response:
      file: mocks/high-value-payment.json
```

## Integration with Python httpx

### Recording from Python Application

```python
import httpx

# Configure httpx to use the proxy
proxies = {
    "http://": "http://localhost:8080",
    "https://": "http://localhost:8080",
}

with httpx.Client(proxies=proxies) as client:
    # All requests will be recorded
    response = client.get("https://api.github.com/users/octocat")
    print(response.json())
    
    # Use different mock IDs for different test scenarios
    headers = {"x-mock-id": "test-scenario-1"}
    response = client.post(
        "https://api.example.com/data",
        json={"key": "value"},
        headers=headers
    )
```

### Testing with Mocks

```python
import httpx

# Point httpx to mock server instead of real API
client = httpx.Client(base_url="http://localhost:8000")

# Same code, but now using mocks
response = client.get("/users/octocat")
assert response.status_code == 200

# Test different scenarios
client.headers["x-mock-id"] = "test-scenario-1"
response = client.post("/data", json={"key": "value"})
assert response.status_code == 201
```

## Docker Integration

### Docker Compose Setup

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mock-server:
    build: .
    ports:
      - "8000:8000"
    volumes:
      - ./mocks:/mocks:ro
    command: ["-mock-dir", "/mocks", "-host", "0.0.0.0"]
    
  proxy:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./recordings:/recordings
    command: [
      "-target", "https://api.example.com",
      "-log-dir", "/recordings",
      "-host", "0.0.0.0"
    ]
```

Create `Dockerfile`:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /auto-proxy ./cmd/proxy
RUN CGO_ENABLED=0 go build -o /auto-mock-server ./cmd/mock

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /auto-proxy /usr/local/bin/
COPY --from=builder /auto-mock-server /usr/local/bin/
ENTRYPOINT ["/usr/locaauto-mock-server"]
```

### Running with Docker

```bash
# Start services
docker-compose up -d

# Use the mock server
curl http://localhost:8000/api/endpoint

# Record new mocks
curl http://localhost:8080/api/new-endpoint

# Stop services
docker-compose down
```

## Performance Testing

### Load Testing Mock Server

```bash
# Start mock server
auto-mock-server -mock-dir mocks -host 0.0.0.0 -port 8000

# Use wrk or ab for load testing
wrk -t4 -c100 -d30s http://localhost:8000/api/endpoint

# Or with Apache Bench
ab -n 100000 -c 100 http://localhost:8000/api/endpoint
```

### Monitoring with Stats Endpoint

```bash
# Get statistics
curl http://localhost:8000/__mock__/stats

# List all available mocks
curl http://localhost:8000/__mock__/list
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Install Auto Tools Go
        run: |
          go install github.com/httpx-record/auto-tools-go/cmd/mock@latest
      
      - name: Start Mock Server
        run: |
          auto-mock-server -mock-dir test-mocks -port 8000 &
          sleep 2
      
      - name: Run Tests
        run: |
          export API_BASE_URL=http://localhost:8000
          make test
```

## Advanced Scenarios

### Content-Type Negotiation

```bash
# Record different content types for same endpoint
curl -H "Accept: application/json" \
     -H "x-mock-id: data" \
     http://localhost:8080/api/data

curl -H "Accept: application/xml" \
     -H "x-mock-id: data" \
     http://localhost:8080/api/data

# Mock server will serve appropriate response based on Accept header
curl -H "Accept: application/json" http://localhost:8000/api/data
curl -H "Accept: application/xml" http://localhost:8000/api/data
```

### Method-Specific Mocks

```bash
# Record different methods for same endpoint
curl -X GET http://localhost:8080/api/resource
curl -X POST -d '{"name":"value"}' http://localhost:8080/api/resource
curl -X PUT -d '{"name":"updated"}' http://localhost:8080/api/resource
curl -X DELETE http://localhost:8080/api/resource

# Mock server automatically serves correct response based on method
curl -X GET http://localhost:8000/api/resource
curl -X POST -d '{"name":"value"}' http://localhost:8000/api/resource
```

## Troubleshooting

### Enable Verbose Logging

Currently, both tools output to stdout. Redirect to files for analysis:

```bash
# Proxy with logging
auto-proxy -target https://api.example.com 2>&1 | tee proxy.log

# Mock server with logging  
auto-mock-server -mock-dir mocks 2>&1 | tee mock.log
```

### Verify Mock Files

```bash
# Check recorded files
ls -lh mocks/*/

# Pretty-print a mock file
cat mocks/default/application_json_*.json | jq .
```

### Debug Scenario Matching

```bash
# List active scenarios
curl http://localhost:8000/__mock__/list | jq .

# Check stats
curl http://localhost:8000/__mock__/stats | jq .
```

### Debug Missing Mocks (404 Logging)

The mock server automatically logs all 404 responses to help identify missing mocks:

```bash
# Start mock server with default log directory (mock_log/)
auto-mock-server -mock-dir mocks

# Or specify custom log directory
auto-mock-server -mock-dir mocks -log-dir debug_logs

# Make requests to test your application
curl http://localhost:8000/api/users/1
curl http://localhost:8000/api/posts

# Check for 404s
ls -lt mock_log/
cat mock_log/application_json_20251125_174411_*.json | jq .
```

**Example 404 log file:**

```json
{
  "request": {
    "request_id": "20251125174411.913912",
    "timestamp": "2025-11-25T14:44:11.913976Z",
    "method": "DELETE",
    "url": "/api/test/delete/456",
    "headers": {
      "Accept": "application/json",
      "x-mock-id": "test-scenario"
    },
    "body": {"id": 456}
  },
  "response": {
    "status_code": 404,
    "headers": {"Content-Type": "application/json"},
    "body": {"error": "No mock found"},
    "delay": 0
  }
}
```

**Create missing mocks from logs:**

```bash
# After identifying missing endpoints in 404 logs,
# record them with the proxy:
auto-proxy -target http://api.example.com -log-dir mocks

# Make the missing request through proxy
curl -H "x-mock-id: test-scenario" \
     -X DELETE http://localhost:8080/api/test/delete/456

# Restart mock server with new mocks
auto-mock-server -mock-dir mocks
```
