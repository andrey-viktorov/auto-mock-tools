#!/bin/bash
set -e

echo "=== Testing Basic Proxy Recording with New Storage ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

MOCK_DIR="test_mocks"
PROXY_PORT=8082

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    pkill -f "auto-proxy.*$PROXY_PORT" 2>/dev/null || true
    rm -rf "$MOCK_DIR"
}

trap cleanup EXIT

# Build if needed
if [ ! -f "../../bin/auto-proxy" ]; then
    echo -e "${BLUE}Building proxy...${NC}"
    cd ../.. && make build-proxy && cd tests/integration
fi

# Clean start
rm -rf "$MOCK_DIR"
mkdir -p "$MOCK_DIR"

# Start proxy
echo -e "${GREEN}Starting recording proxy...${NC}"
echo "  Target: http://httpbin.org"
echo "  Port: $PROXY_PORT"
echo "  Mock dir: $MOCK_DIR"
echo ""

../../bin/auto-proxy -target http://httpbin.org -log-dir "$MOCK_DIR" -port $PROXY_PORT > /dev/null 2>&1 &
PROXY_PID=$!
sleep 2

echo -e "${GREEN}Making test requests...${NC}"

# Request 1: GET with default mock-id
echo -e "${YELLOW}1. GET /get (default mock-id)${NC}"
curl -s "http://127.0.0.1:$PROXY_PORT/get?test=1" | jq -r '.url' || true
sleep 1

# Request 2: GET with custom mock-id
echo -e "${YELLOW}2. GET /get (mock-id: custom-test)${NC}"
curl -s -H "x-mock-id: custom-test" "http://127.0.0.1:$PROXY_PORT/get?test=2" | jq -r '.url' || true
sleep 1

# Request 3: POST with JSON
echo -e "${YELLOW}3. POST /post (mock-id: post-test)${NC}"
curl -s -X POST -H "x-mock-id: post-test" -H "Content-Type: application/json" \
  -d '{"name":"test","value":123}' "http://127.0.0.1:$PROXY_PORT/post" | jq -r '.json' || true
sleep 1

# Request 4: Another GET with custom-test
echo -e "${YELLOW}4. GET /uuid (mock-id: custom-test)${NC}"
curl -s -H "x-mock-id: custom-test" "http://127.0.0.1:$PROXY_PORT/uuid" | jq -r '.uuid' || true
sleep 1

echo ""
echo -e "${GREEN}Stopping proxy...${NC}"
kill $PROXY_PID 2>/dev/null || true
sleep 1

# Verify structure
echo ""
echo -e "${GREEN}Verifying mock structure...${NC}"
echo ""
echo "Directory structure:"
tree "$MOCK_DIR" 2>/dev/null || find "$MOCK_DIR" -type f

echo ""
echo -e "${GREEN}Mock-id directories:${NC}"
ls -la "$MOCK_DIR"

echo ""
echo -e "${GREEN}Sample file content (first found):${NC}"
FIRST_FILE=$(find "$MOCK_DIR" -name "*.json" -type f | head -1)
if [ -n "$FIRST_FILE" ]; then
    echo "File: $FIRST_FILE"
    cat "$FIRST_FILE" | jq '.'
fi

echo ""
echo -e "${GREEN}✓ Test complete!${NC}"
