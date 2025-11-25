.PHONY: build-proxy build-mock build build-testutils run-proxy run-mock clean test test-integration test-all help

# Binary names and paths
BIN_DIR=bin
PROXY_BINARY=$(BIN_DIR)/auto-proxy
MOCK_BINARY=$(BIN_DIR)/auto-mock-server

# Default values
LOG_DIR?=mocks
TARGET?=http://httpbin.org
PROXY_PORT?=8080
MOCK_PORT?=8000

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: build-proxy build-mock ## Build both proxy and mock server

build-proxy: ## Build the recording proxy
	@mkdir -p $(BIN_DIR)
	go build -o $(PROXY_BINARY) -v ./cmd/auto-proxy
	chmod +x $(PROXY_BINARY)

build-mock: ## Build the mock server
	@mkdir -p $(BIN_DIR)
	go build -o $(MOCK_BINARY) -v ./cmd/auto-mock-server
	chmod +x $(MOCK_BINARY)

build-optimized: ## Build optimized binaries (smaller size)
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(PROXY_BINARY) -v ./cmd/auto-proxy
	go build -ldflags="-s -w" -o $(MOCK_BINARY) -v ./cmd/auto-mock-server
	chmod +x $(PROXY_BINARY) $(MOCK_BINARY)

# Run targets
run-proxy: build-proxy ## Build and run the recording proxy
	./$(PROXY_BINARY) -target $(TARGET) -log-dir $(LOG_DIR) -port $(PROXY_PORT)

run-mock: build-mock ## Build and run the mock server
	./$(MOCK_BINARY) -mock-dir $(LOG_DIR) -port $(MOCK_PORT)

# Development targets
clean: ## Remove binaries, logs, and build artifacts
	go clean
	rm -rf $(BIN_DIR)
	rm -rf mocks/

test: ## Run all tests
	go test -v ./pkg/... ./cmd/...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./pkg/... ./cmd/...
	go tool cover -html=coverage.out

test-integration: build build-testutils ## Run integration tests (requires build)
	@echo "Running integration tests..."
	cd tests/integration && bash test_basic_recording.sh
	cd tests/integration && bash test_request_logging.sh
	cd tests/integration && bash test_sse.sh
	cd tests/integration && bash test_full_workflow.sh

test-all: test test-integration ## Run all tests (unit + integration)

# Build test utilities
build-testutils: ## Build test utilities (SSE servers, etc.)
	@mkdir -p testutils/bin
	cd testutils/servers/sse_test_server && go build -o ../../bin/sse_test_server
	cd testutils/servers/sse_test_server_fasthttp && go build -o ../../bin/sse_test_server_fasthttp
	cd testutils/servers/sse_test_server_https && go build -o ../../bin/sse_test_server_https
	cd testutils/servers/mtls_test_server && go build -o ../../bin/mtls_test_server
	cd testutils/servers/mtls_sse_server && go build -o ../../bin/mtls_sse_server
	chmod +x testutils/bin/*

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linters
	golangci-lint run || true

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

# Platform-specific builds
build-linux: ## Build for Linux (both tools)
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-proxy-linux-amd64 ./cmd/auto-proxy
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-mock-server-linux-amd64 ./cmd/auto-mock-server
	chmod +x $(BIN_DIR)/auto-proxy-linux-amd64 $(BIN_DIR)/auto-mock-server-linux-amd64

build-darwin: ## Build for macOS (both tools)
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-proxy-darwin-amd64 ./cmd/auto-proxy
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-proxy-darwin-arm64 ./cmd/auto-proxy
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-mock-server-darwin-amd64 ./cmd/auto-mock-server
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-mock-server-darwin-arm64 ./cmd/auto-mock-server
	chmod +x $(BIN_DIR)/auto-proxy-darwin-* $(BIN_DIR)/auto-mock-server-darwin-*

build-windows: ## Build for Windows (both tools)
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-proxy-windows-amd64.exe ./cmd/auto-proxy
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/auto-mock-server-windows-amd64.exe ./cmd/auto-mock-server
	chmod +x $(BIN_DIR)/auto-proxy-windows-amd64.exe $(BIN_DIR)/auto-mock-server-windows-amd64.exe

build-all: build-linux build-darwin build-windows ## Build for all platforms

.DEFAULT_GOAL := help
