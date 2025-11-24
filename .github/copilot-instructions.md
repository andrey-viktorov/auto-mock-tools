# Auto Tools Go - AI Coding Assistant Guidelines

## Project Overview

High-performance HTTP recording proxy and mock server built with Go and fasthttp. Part of the httpx-record ecosystem, providing ~50K RPS mock serving with zero-allocation design.

**Architecture:** Two standalone binaries (`auto-proxy`, `auto-mock-server`) that share common libraries (`pkg/storage`, `pkg/proxy`, `pkg/handlers`).

## Core Components

### 1. Recording Proxy (`cmd/proxy`, `pkg/proxy/`)
- **Purpose:** Reverse proxy that records HTTP traffic to JSON files organized by `x-mock-id` header
- **Entry:** `cmd/proxy/main.go` - CLI parsing, server setup, graceful shutdown
- **Handler:** `pkg/proxy/handler.go` - Request forwarding, mTLS support, SSE streaming
- **Recorder:** `pkg/proxy/recorder.go` - File I/O, JSON serialization, SSE event parsing
- **File naming:** `<mock_id>/<content-type>_<timestamp>_<random>.json`

### 2. Mock Server (`cmd/mock`, `pkg/handlers/`, `pkg/storage/`)
- **Purpose:** Serves recorded mocks with optional timing replay and SSE streaming
- **Entry:** `cmd/mock/main.go` - CLI parsing, storage initialization
- **Storage:** `pkg/storage/storage.go` - In-memory index, zero-allocation lookups, pre-serialized responses
- **Handlers:** `pkg/handlers/handlers.go` - Request routing, SSE streaming with pooled writers
- **Indexing:** `path|mockID|contentType` composite key for O(1) lookups

### 3. File Format
Each recorded interaction is a single JSON file containing request + response:
```json
{
  "request": {"request_id": "...", "method": "GET", "url": "...", "headers": {...}, "body": "..."},
  "response": {"status_code": 200, "headers": {...}, "body": {...}, "delay": 0.123}
}
```

**SSE format:** Response body is array of `{"data": {...}, "timestamp": 1.5}` objects for timing replay.

## Performance Philosophy

**Zero-allocation hot path:** The mock server's request handling avoids allocations through:
- Pre-serialized response bodies (`[]byte`) at startup
- Direct `[]byte` operations (no string conversions in `FindResponseBytes`)
- Pooled SSE stream writers (`sync.Pool`)
- Pre-computed lowercase header keys for fast lookup
- Composite string keys for indexing (single allocation per key, reused)

**When editing performance-critical code:** Always work with `[]byte` in handlers, avoid `string()` conversions, use `bytes.Equal()` not `==`.

## Build & Test Workflow

```bash
# Development
make build              # Build both binaries
make test               # Unit tests only
make test-integration   # Requires build-testutils first
make test-all           # Unit + integration

# Integration tests live in tests/integration/
# Requires test servers: make build-testutils
```

**Test utilities:** `testutils/servers/` contains SSE and mTLS test servers. Build with `make build-testutils` before running integration tests.

## Key Patterns & Conventions

### 1. Header Handling
- **`x-mock-id` is internal:** Never forwarded to upstream (proxy), never returned to client (mock server)
- Always stored in request headers and file structure
- Default value: `"default"` when header absent

### 2. SSE (Server-Sent Events) Support
- **Proxy:** Detects via `Accept: text/event-stream`, uses raw TCP/TLS streaming (`handleSSEStreaming`)
- **Mock:** Replays with timing via `SetBodyStreamWriter` + pooled `sseStreamWriter`
- Events stored as array with timestamps for replay: `[{"data": {...}, "timestamp": 1.5}, ...]`

### 3. Content-Type Negotiation
- Uses `Accept` header from client to match stored mocks
- Normalizes by stripping charset/params: `application/json; charset=utf-8` → `application/json`
- Sanitizes for filenames: `text/event-stream` → `text_event-stream`

### 4. Method Filtering
- Multiple mocks can exist for same path/mock_id/content-type with different methods
- `FindResponseBytes` filters by method case-insensitively via `equalFoldBytes`

### 5. Timing Replay
- Controlled by CLI flags: `-replay-timing` and `-jitter 0.1` (±10%)
- Applied as sleep before response (non-SSE) or between SSE events
- SSE timing uses event timestamps relative to request start
- **Scenario override:** `delay` in scenario config overrides timing from log file
  - For SSE: timestamps are proportionally rescaled (e.g., 2.0s → 1.0s scales all events by 0.5x)
  - For non-SSE: directly replaces the delay before response

## CLI Patterns

Both tools use consistent flag naming:
- `-host` / `-port` - Server binding (default: 127.0.0.1:8080 for proxy, :8000 for mock)
- `-log-dir` / `-mock-dir` - File storage directory
- `-target` - Proxy target URL (required)
- `-replay-timing` / `-jitter` - Mock server timing control

## Testing Strategy

1. **Unit tests:** `pkg/*/` with `_test.go` files (use `go test ./pkg/...`)
2. **Integration tests:** `tests/integration/*.sh` scripts that:
   - Build binaries (`make build`)
   - Build test utilities (`make build-testutils`)
   - Start test servers (SSE, mTLS)
   - Run proxy → record → mock server workflow
   - Verify output

## Special Endpoints (Mock Server)

- `GET /__mock__/stats` - Pre-serialized statistics (total responses, unique paths/mock_ids)
- `GET /__mock__/list` - Pre-serialized list of all mocks with metadata

## Dependencies

- **fasthttp:** Core HTTP library (valyala/fasthttp) - provides `RequestCtx`, zero-copy buffers
- **Standard library:** No other external deps (crypto/tls for mTLS, encoding/json, bufio, sync)

## Scenario-Based Mock Configuration

The mock server supports advanced scenario-based routing via YAML config (`-mock-config` flag). When scenarios are enabled, the traditional `x-mock-id` lookup is disabled.

**Key features:**
- **Filter-based routing:** Use JSONFilter expressions to match request bodies
- **Declaration order:** First matching scenario wins
- **Timing override:** Override `delay` from log files per scenario
- **SSE proportional scaling:** When overriding timing for SSE, all event timestamps are scaled proportionally

**Config format:**
```yaml
scenarios:
  - name: Scenario Name
    method: POST           # Optional, defaults to method in response file
    path: /api/endpoint
    filter:                # Optional JSONFilter expression
      body:
        eq:
          field: status
          value: ready
    response:
      file: path/to/response.json
      delay: 1.5  # Optional timing override
```

**Implementation:** `pkg/storage/scenario.go` - loads config, applies filters, overrides timing.

## Common Tasks

### Adding a new CLI flag
1. Add to `flag.String/Int/Bool` in `cmd/*/main.go`
2. Pass to constructor (e.g., `proxy.NewProxyHandler`)
3. Store in struct and use in handler logic

### Modifying storage indexing
Edit `makeIndexKey` and `makeIndexKeyFromBytes` in `pkg/storage/storage.go`. Current format: `"path|mockID|contentType"`.

### Adding a new special endpoint
Add path comparison in `Router()` in `pkg/handlers/handlers.go`, create handler function following `StatsHandler` pattern.

## Cross-Platform Builds

`Makefile` provides targets for all platforms:
- `build-linux`, `build-darwin`, `build-windows` - Platform-specific
- `build-all` - All platforms
- Optimized builds use `-ldflags="-s -w"` for smaller binaries

## Security Notes

- **mTLS:** Client certs loaded via `-client-cert`/`-client-key` flags (proxy only)
- **TLS config:** `InsecureSkipVerify: true` by default (test/dev usage)
- **Mock files:** May contain sensitive data - secure the storage directory
- **x-mock-id:** Not exposed to external systems (stripped from upstream/client)
