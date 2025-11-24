# Test Utilities

Test servers and certificates for integration testing.

## Structure

```
testutils/
├── servers/          # Go source code for test servers
│   ├── sse_test_server.go           # Basic SSE server (HTTP)
│   ├── sse_test_server_fasthttp.go  # FastHTTP-based SSE server
│   ├── sse_test_server_https.go     # SSE server with HTTPS
│   ├── mtls_test_server.go          # mTLS test server
│   └── mtls_sse_server.go           # mTLS + SSE server
├── certs/           # SSL/TLS certificates for testing
│   ├── ca-cert.pem               # Certificate Authority
│   ├── ca-key.pem                # CA private key
│   ├── server-cert.pem           # Server certificate
│   ├── server-key.pem            # Server private key
│   ├── client-cert.pem           # Client certificate (for mTLS)
│   └── client-key.pem            # Client private key (for mTLS)
└── bin/             # Compiled binaries (generated, not in git)
    ├── sse_test_server
    ├── mtls_test_server
    └── ...
```

## Building

Build all test utilities:
```bash
make build-testutils
```

This compiles all server binaries from `servers/` into `bin/`.

## Test Servers

### SSE Test Server (Basic)
```bash
cd testutils/bin
./sse_test_server
```
- Port: 5555
- Protocol: HTTP
- Streams 5 SSE events with 100ms delay

### SSE Test Server (FastHTTP)
```bash
cd testutils/bin
./sse_test_server_fasthttp
```
- Port: 5556
- Protocol: HTTP
- High-performance version using fasthttp

### SSE Test Server (HTTPS)
```bash
cd testutils/bin
./sse_test_server_https
```
- Port: 5557
- Protocol: HTTPS
- Requires certificates from `certs/`

### mTLS Test Server
```bash
cd testutils/bin
./mtls_test_server
```
- Port: 5558
- Protocol: HTTPS with mutual TLS
- Requires client certificate verification
- Test with: `curl --cert ../certs/client-cert.pem --key ../certs/client-key.pem --cacert ../certs/ca-cert.pem https://localhost:5558/test`

### mTLS SSE Server
```bash
cd testutils/bin
./mtls_sse_server
```
- Port: 5559
- Protocol: HTTPS with mutual TLS + SSE
- Combines mTLS authentication with SSE streaming

## Certificates

All certificates in `certs/` are self-signed and **for testing only**. 

- `ca-cert.pem` / `ca-key.pem` - Certificate Authority
- `server-cert.pem` / `server-key.pem` - Server certificates
- `client-cert.pem` / `client-key.pem` - Client certificates for mTLS

To regenerate certificates, use `openssl` commands or a certificate generation script.

## Usage in Integration Tests

Integration tests in `tests/integration/` automatically build and use these utilities:

```bash
cd tests/integration
./test_sse.sh          # Uses sse_test_server
./test_full_workflow.sh # Full proxy + mock server test
```

The `Makefile` target `test-integration` handles building and running all integration tests.
