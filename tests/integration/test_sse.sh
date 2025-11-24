#!/bin/bash
set -e

echo "=== Testing SSE (Server-Sent Events) Recording ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

MOCK_DIR="test_sse_mocks"
PROXY_PORT=8084
TEST_SERVER_PORT=5555

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    pkill -f "auto-proxy.*$PROXY_PORT" 2>/dev/null || true
    pkill -f "sse_test_server" 2>/dev/null || true
    rm -rf "$MOCK_DIR"
}

trap cleanup EXIT

# Build proxy
if [ ! -f "../../bin/auto-proxy" ]; then
    echo -e "${BLUE}Building proxy...${NC}"
    cd ../.. && make build-proxy && cd tests/integration
fi

# Build SSE test server if needed
cd ../../testutils/bin
if [ ! -f "./sse_test_server" ]; then
    echo -e "${BLUE}Building SSE test server...${NC}"
    cd ../servers && go build -o ../bin/sse_test_server sse_test_server.go && cd ../bin
fi

# Start SSE test server
echo -e "${BLUE}Starting SSE test server on port $TEST_SERVER_PORT...${NC}"
./sse_test_server &
SSE_PID=$!
sleep 2
cd ../../tests/integration

# Clean start
rm -rf "$MOCK_DIR"
mkdir -p "$MOCK_DIR"

# Start proxy
echo -e "${GREEN}Starting recording proxy...${NC}"
../../bin/auto-proxy -target "http://127.0.0.1:$TEST_SERVER_PORT" -log-dir "$MOCK_DIR" -port $PROXY_PORT > /dev/null 2>&1 &
PROXY_PID=$!
sleep 2

echo -e "${YELLOW}Testing SSE recording...${NC}"
echo "  Making SSE request with mock-id: sse-test"
echo ""

# Make SSE request (will stream for a few seconds)
timeout 5s curl -s -H "Accept: text/event-stream" -H "x-mock-id: sse-test" \
  "http://127.0.0.1:$PROXY_PORT/events" || true

echo ""
echo -e "${GREEN}Stopping proxy and test server...${NC}"
kill $PROXY_PID 2>/dev/null || true
kill $SSE_PID 2>/dev/null || true
sleep 1

# Verify structure
echo ""
echo -e "${GREEN}Verifying SSE mock files...${NC}"
echo ""
echo "Files created:"
find "$MOCK_DIR" -name "*.json" -type f

echo ""
SSE_FILE=$(find "$MOCK_DIR" -name "text_event-stream*.json" -type f | head -1)
if [ -n "$SSE_FILE" ]; then
    echo -e "${GREEN}SSE mock file content:${NC}"
    echo "File: $SSE_FILE"
    cat "$SSE_FILE" | jq '.'
    
    echo ""
    echo -e "${GREEN}Event count:${NC}"
    cat "$SSE_FILE" | jq '.response.body | length'
else
    echo -e "${YELLOW}No SSE file found${NC}"
fi

echo ""
echo -e "${GREEN}✓ SSE test complete!${NC}"
