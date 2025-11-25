# Auto Mock Tools

[![CI](https://github.com/andrey-viktorov/auto-mock-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/andrey-viktorov/auto-mock-tools/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/andrey-viktorov/auto-mock-tools)](https://goreportcard.com/report/github.com/andrey-viktorov/auto-mock-tools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/andrey-viktorov/auto-mock-tools.svg)](https://pkg.go.dev/github.com/andrey-viktorov/auto-mock-tools)
[![Go Version](https://img.shields.io/github/go-mod/go-version/andrey-viktorov/auto-mock-tools)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/andrey-viktorov/auto-mock-tools)](https://github.com/andrey-viktorov/auto-mock-tools/releases)

Automatic HTTP traffic recording and mocking toolkit. High-performance tools built with fasthttp for recording HTTP requests/responses and serving them as mocks.

## 🚀 Tools

### 1. **Auto Proxy** - Recording Reverse Proxy
HTTP reverse proxy that records all traffic to structured JSON files organized by mock_id.

### 2. **Auto Mock Server** - High-Performance Mock Server
Fast mock server (~50K RPS) that serves recorded traffic with SSE support and timing replay.

## ✨ Features

- ⚡ **High Performance** - Built on fasthttp with zero-allocation design
- 📁 **Structured Storage** - Files organized as `mock_id/<content-type>_<timestamp>_<random>.json`
- 🏷️ **x-mock-id Support** - Automatic scenario identification via headers
- 🔍 **Scenario Filters** - Optional YAML-defined JSON body filters with `jsonfilter-go`
- 🌊 **SSE Support** - Full Server-Sent Events support with event timestamps
- ⏱️ **Timing Replay** - Record and replay request/response timing with jitter
- 🔒 **HTTPS/mTLS** - Support for HTTPS upstream and mutual TLS authentication
- 📖 **Human-Readable** - Each mock file can be edited manually
- 📝 **404 Logging** - Automatic logging of unmatched requests for mock creation
- 🎭 **Complete Workflow** - Record with proxy → Replay with mock server

## 📁 Project Structure

```
auto-mock-tools/
├── cmd/                    # Application entrypoints
│   ├── proxy/             # Recording proxy binary
│   └── mock/              # Mock server binary
├── pkg/                   # Shared libraries
│   ├── storage/           # Mock storage (reading/serving)
│   ├── proxy/             # Proxy & recording logic
│   └── handlers/          # Mock server HTTP handlers
├── testutils/             # Testing utilities
│   ├── servers/           # Test servers (SSE, mTLS, etc.)
│   ├── certs/             # SSL certificates for testing
│   └── bin/               # Compiled test binaries (gitignored)
└── tests/
    └── integration/       # Integration test scripts
```

## 📦 Installation

### Quick Start (No Installation Required)

Run directly without cloning or installing:

```bash
# Run recording proxy
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest -target http://api.example.com

# Run mock server
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-mock-server@latest -mock-dir mocks
```

### Install Globally

Install once, use everywhere:

```bash
# Install both tools
go install github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest
go install github.com/andrey-viktorov/auto-mock-tools/cmd/auto-mock-server@latest

# Now use them anywhere
auto-proxy -target http://api.example.com
auto-mock-server -mock-dir mocks
```

### From Source

```bash
# Clone and build
git clone https://github.com/andrey-viktorov/auto-mock-tools.git
cd auto-mock-tools
make build

# Binaries will be in bin/
./bin/auto-proxy -target http://api.example.com
./bin/auto-mock-server -mock-dir mocks
```

### Pre-built Binaries

Download pre-built binaries for your platform from the [Releases](https://github.com/andrey-viktorov/auto-mock-tools/releases) page.

## 🔄 Quick Start

> 📖 **New to Auto Mock Tools?** Check out the [Quick Start Guide](QUICKSTART.md) for a 5-minute introduction!  
> 💡 For detailed examples, see [EXAMPLES.md](EXAMPLES.md)

### 1. Record Traffic with Proxy

```bash
# Run directly (no installation)
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-proxy@latest \
  -target http://httpbin.org -log-dir mocks -port 8080

# Or if installed
auto-proxy -target http://httpbin.org -log-dir mocks -port 8080

# Make requests through the proxy
curl http://localhost:8080/get
curl -H "x-mock-id: test-1" http://localhost:8080/post -d '{"key":"value"}'
```

Recorded files will be in `mocks/` directory:
```
mocks/
  default/
    application_json_20251123_120000_abc123.json
  test-1/
    application_json_20251123_120001_def456.json
```

### 2. Serve Mocks

```bash
# Run directly (no installation)
go run github.com/andrey-viktorov/auto-mock-tools/cmd/auto-mock-server@latest \
  -mock-dir mocks -port 8000

# Or if installed
auto-mock-server -mock-dir mocks -port 8000

# Use the mocks
curl http://localhost:8000/get
curl -H "x-mock-id: test-1" http://localhost:8000/post
```

## 📖 Usage

### Auto Proxy (Recording)

```bash
# Basic usage (target is REQUIRED)
auto-proxy -target http://api.example.com

# Custom directory and port
auto-proxy -target http://localhost:3000 -log-dir recordings -port 8888

# With mTLS client certificate
auto-proxy -target https://secure-api.com \
  -client-cert client.crt \
  -client-key client.key

# On all interfaces
auto-proxy -target http://api.example.com -host 0.0.0.0 -port 8080
```

**CLI Options:**
```
-target string      Target URL to proxy requests to (REQUIRED)
-log-dir string     Directory to store recorded mock files (default "mocks")
-host string        Host to bind the proxy to (default "127.0.0.1")
-port int           Port to bind the proxy to (default 8080)
-client-cert string Path to client certificate file for mTLS (optional)
-client-key string  Path to client key file for mTLS (optional)
```

### Auto Mock Server

```bash
# Basic usage
auto-mock-server -mock-dir mocks

# Custom port and host
auto-mock-server -mock-dir mocks -host 0.0.0.0 -port 9000

# With timing replay and jitter
auto-mock-server -mock-dir mocks -replay-timing -jitter 0.1
```

**CLI Options:**
```
-mock-dir string    Directory containing recorded mock files (default "mocks")
-mock-config string YAML file that defines scenario filters; disables x-mock-id lookup when set
-log-dir string     Directory to store 404 request/response logs (default "mock_log")
-host string        Host to bind the server to (default "127.0.0.1")
-port int           Port to bind the server to (default 8000)
-replay-timing      Replay original request/response timing (latency)
-jitter float       Add random jitter to timing, 0.0-1.0 (0.1 = ±10%)
```

## 🧩 Scenario-Based Filtering

Provide `-mock-config tests/fixtures/mock-example.yml` to switch the mock server from
`x-mock-id` lookups to declarative JSON body scenarios. Each scenario is
evaluated in file order and the first match wins.

- **name** – identifier shown in `/__mock__/list` and stats
- **method** – HTTP verb (defaults to the recorded method if omitted)
- **path** – request path to match (`/users/1`, `/api/v1/status`, ...)
- **filter.body** – [jsonfilter-go](https://pkg.go.dev/github.com/andrey-viktorov/jsonfilter-go) tree;
  omit to match any body. Use [gjson path syntax](https://github.com/tidwall/gjson#path-syntax) without `$` prefix (e.g., `processing.state` not `$.processing.state`)
- **response.file** – recorded JSON file; paths are resolved relative to the
  YAML file

```yaml
scenarios:
  - name: Status Ready With Valid ID
    method: POST
    path: /api/v1/status
    filter:
      body:
        and:
          - eq:
              field: processing.state
              value: done
          - rx:
              field: payload.id
              value: ^[A-Z]{3}-[0-9]{4}$
    response:
      file: test_mocks/api-v1/application_json_20251122_233842_8e3ce990.json
      # Optional: override delay from log file
      # For SSE responses, timing is redistributed proportionally across events
      delay: 1.5

  - name: Status Fallback Default
    method: POST
    path: /api/v1/status
    response:
      file: test_mocks/default/application_json_20251122_233842_059b6fbd.json
```

When scenarios are enabled:

1. `x-mock-id` is ignored.
2. The request body is streamed directly into the JSON filter.
3. Accept negotiation is skipped—the selected response dictates headers.
4. `delay` can be overridden per scenario:
   - For regular responses: directly replaces the delay before response
   - For SSE: all event timestamps are scaled proportionally (e.g., 2.0s → 1.0s = 0.5x scaling)
5. `jitter` is applied to the total delay:
   - For regular responses: adds ±N% variance to the delay
   - For SSE: all event timestamps are scaled by the same jitter factor (e.g., 5% jitter = 0.95x to 1.05x scaling)

Use `/__mock__/stats` and `/__mock__/list` to verify which scenarios are active.

## 🎭 Mock Server API

### Regular Endpoints

Make requests with `x-mock-id` and `Accept` headers:

```bash
# With mock_id and content-type
curl -H "x-mock-id: user-1" -H "Accept: application/json" \
     http://localhost:8000/users/1

# Without headers (uses defaults: mock_id="default", content-type="application/json")
curl http://localhost:8000/users/1

# Different content-type
curl -H "x-mock-id: user-1" -H "Accept: application/xml" \
     http://localhost:8000/users/1
```

### Special Endpoints

#### `GET /__mock__/stats`
Returns statistics about loaded mocks:
```json
{
  "total_responses": 42,
  "unique_paths": 8,
  "unique_mock_ids": 3,
  "paths": ["/users/1", "/posts", ...]
}
```

#### `GET /__mock__/list`
Lists all loaded mock responses:
```json
{
  "total": 42,
  "mocks": [
    {
      "request_id": "20251123120000.123456789",
      "path": "/users/1",
      "method": "GET",
      "mock_id": "user-1",
      "content_type": "application/json",
      "status_code": 200,
      "full_url": "http://api.example.com/users/1"
    },
    ...
  ]
}
```

## 📁 File Format

Each recorded request/response is stored in a single JSON file:

```json
{
  "request": {
    "request_id": "20251123120000.123456789",
    "timestamp": "2025-11-23T12:00:00.123456789Z",
    "method": "GET",
    "url": "http://api.example.com/users/1",
    "headers": {
      "Accept": "application/json",
      "x-mock-id": "user-1"
    },
    "body": ""
  },
  "response": {
    "request_id": "20251123120000.123456789",
    "timestamp": "2025-11-23T12:00:00.234567890Z",
    "status_code": 200,
    "headers": {
      "Content-Type": "application/json",
      "x-mock-id": "user-1"
    },
    "body": {
      "id": 1,
      "name": "John Doe"
    },
    "delay": 0.123
  }
}
```

### SSE (Server-Sent Events) Format

For SSE responses, events are stored with timestamps:

```json
{
  "response": {
    "status_code": 200,
    "headers": {
      "Content-Type": "text/event-stream"
    },
    "body": [
      {
        "data": {"message": "Event 1"},
        "timestamp": 0.0
      },
      {
        "data": {"message": "Event 2"},
        "timestamp": 1.5
      }
    ],
    "delay": 3.2
  }
}
```

## 📝 404 Request Logging

### Overview

The mock server automatically logs all requests that result in 404 (no mock found) to help you identify missing mocks. Each 404 response is recorded in the same format as proxy recordings, making it easy to create new mocks from failed requests.

### Usage

```bash
# Default: logs saved to mock_log/ directory
auto-mock-server -mock-dir mocks

# Custom log directory
auto-mock-server -mock-dir mocks -log-dir failed_requests

# Disable logging by using empty string
auto-mock-server -mock-dir mocks -log-dir ""
```

### Log File Format

Each 404 request is logged to a separate JSON file:

```
mock_log/
  application_json_20251125_174411_d214ec70.json
  text_html_20251125_174315_cd06f8ed.json
```

**File naming:** `<content-type>_<timestamp>_<random>.json` (based on `Accept` header)

**File content:**
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
    "request_id": "20251125174411.913912",
    "timestamp": "2025-11-25T14:44:11.913985Z",
    "status_code": 404,
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {"error": "No mock found"},
    "delay": 0
  }
}
```

### Use Cases

**1. Debugging missing mocks:**
```bash
# Start mock server with logging
auto-mock-server -mock-dir mocks -log-dir debug_logs

# Run your tests
curl http://localhost:8000/api/new-endpoint

# Check logged requests
ls -lt debug_logs/
cat debug_logs/application_json_*.json
```

**2. Creating mocks from failed requests:**
```bash
# After identifying missing endpoints in logs,
# record them with the proxy:
auto-proxy -target http://api.example.com -log-dir mocks

# Make the missing request through proxy
curl http://localhost:8080/api/new-endpoint

# Restart mock server with new mocks
auto-mock-server -mock-dir mocks
```

**3. Monitoring test coverage:**
```bash
# Clear logs before test run
rm -rf mock_log

# Run tests against mock server
make test

# Review any missing mocks
find mock_log -name '*.json' | wc -l
```

### Implementation Details

- Logs are written asynchronously (non-blocking)
- Failed logging doesn't affect request handling
- Directory created automatically if it doesn't exist
- Same file format as proxy recordings for consistency
- `x-mock-id` header preserved in logs if present
- Content-Type derived from `Accept` header for file naming

## 🌊 SSE (Server-Sent Events) Support

### Recording SSE with Proxy

```bash
# Start proxy
auto-proxy -target http://sse-server.com -log-dir mocks

# Record SSE stream
curl -H "Accept: text/event-stream" \
     -H "x-mock-id: sse-test" \
     http://localhost:8080/events
```

### Replaying SSE with Mock Server

```bash
# Without timing replay (instant)
auto-mock-server -mock-dir mocks

# With timing replay (events sent at recorded intervals)
auto-mock-server -mock-dir mocks -replay-timing

# With timing + jitter (±10% variance)
auto-mock-server -mock-dir mocks -replay-timing -jitter 0.1
```

```bash
# Consume SSE from mock
curl -H "Accept: text/event-stream" \
     -H "x-mock-id: sse-test" \
     http://localhost:8000/events
```

### SSE Timing Behavior

For SSE responses, `delay` represents the **total duration** of the stream (last event timestamp).

**Timing replay (`-replay-timing`):**
- Events are sent at their recorded intervals
- Each event has a `timestamp` relative to stream start
- Original timing is preserved from recording

**Jitter (`-jitter 0.1`):**
- Applied **once** to the total delay, not per event
- All event timestamps are scaled proportionally by the same factor
- Example: 10% jitter means total duration varies ±10% (0.9x to 1.1x)
- Ensures natural variance while maintaining relative event spacing

**Delay override (in scenario config):**
- Event timestamps are scaled proportionally when loading config
- Example: 1.0s → 0.5s scales all timestamps by 0.5x (done once at startup)
- Jitter is then applied to the overridden delay

## 🛠️ Development

### Build Commands

```bash
# Build both tools
make build

# Build individually
make build-proxy
make build-mock

# Build optimized (smaller binaries)
make build-optimized

# Run unit tests
make test

# Run integration tests (requires httpbin.org access)
make test-integration

# Run all tests (unit + integration)
make test-all

# Run with coverage
make test-coverage

# Build test utilities (SSE servers, etc.)
make build-testutils

# Format code
make fmt

# Clean build artifacts
make clean
```

### Platform-Specific Builds

```bash
# Build for Linux
make build-linux

# Build for macOS (Intel + ARM)
make build-darwin

# Build for Windows
make build-windows

# Build for all platforms
make build-all
```

### Project Structure

```
auto-mock-tools/
├── cmd/
│   ├── auto-proxy/     # Recording proxy main
│   └── auto-mock-server/  # Mock server main
├── pkg/
│   ├── storage/        # Shared storage logic
│   ├── proxy/          # Proxy handler & recorder
│   └── handlers/       # Mock server handlers
├── testutils/          # Test utilities
├── go.mod
├── Makefile
└── README.md
```

## 🔧 Advanced Usage

### Using with Python httpx

```python
import httpx

# Record requests
with httpx.Client(
    proxies={"http://": "http://localhost:8080"},
    headers={"x-mock-id": "scenario-1"}
) as client:
    response = client.get("http://api.example.com/users/1")
```

### Method Filtering

Mock server supports method filtering. Multiple mocks with same path/mock_id/content-type but different methods:

```
mocks/user-1/
  application_json_20251123_120000_abc123.json  # GET request
  application_json_20251123_120001_def456.json  # POST request
```

Mock server will return the appropriate response based on request method.

### Content-Type Negotiation

Mock server uses `Accept` header for content-type matching:

```bash
# Get JSON response
curl -H "Accept: application/json" http://localhost:8000/users/1

# Get XML response (if available)
curl -H "Accept: application/xml" http://localhost:8000/users/1
```

## 🔒 Security Notes

- `x-mock-id` header is **not forwarded** to upstream server
- `x-mock-id` header is **not returned** to client (only stored in files)
- For mTLS, ensure client certificates are properly secured
- Mock files may contain sensitive data - secure the `mocks/` directory

## ⚡ Performance

### Mock Server Benchmarks

- **~50,000 RPS** on standard hardware
- **1 allocation per request** (map key lookup only)
- Zero-copy body serving (pre-serialized)
- SSE with timing replay: ~1000 concurrent streams

### Optimization Features

- Pre-serialized response bodies
- Pooled SSE stream writers
- Direct []byte operations (no string conversions)
- Pre-computed lowercase header keys
- Cached stats/mock list JSON

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

Please read our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) before submitting a pull request.

Quick start:
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`make test-all`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## 🔒 Security

For security issues, please review our [Security Policy](SECURITY.md) before reporting.

## 🔗 Related Projects

- Main project: [httpx-record](https://github.com/httpx-record/httpx-record) - Python HTTPX recording transport
- Python mock server: [auto_mock_server](https://github.com/httpx-record/auto_mock_server) - Python/Starlette version

## 🙏 Acknowledgments

- Built with [fasthttp](https://github.com/valyala/fasthttp) - Fast HTTP package for Go
- JSON filtering powered by [jsonfilter-go](https://github.com/andrey-viktorov/jsonfilter-go)
