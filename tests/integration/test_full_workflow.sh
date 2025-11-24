#!/bin/bash
set -e

echo "=== Testing Full Workflow: Record → Mock Server ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

MOCK_DIR="test_workflow_mocks"
PROXY_PORT=8083
MOCK_PORT=9003

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    pkill -f "auto-proxy.*$PROXY_PORT" 2>/dev/null || true
    pkill -f "auto-mock-server.*$MOCK_PORT" 2>/dev/null || true
    rm -rf "$MOCK_DIR"
}

trap cleanup EXIT

# Build both
echo -e "${BLUE}Building proxy and mock server...${NC}"
cd ../.. && make build && cd tests/integration

# Clean start
rm -rf "$MOCK_DIR"
mkdir -p "$MOCK_DIR"

# === PHASE 1: Recording ===
echo ""
echo -e "${GREEN}=== PHASE 1: Recording Traffic ===${NC}"
echo -e "${BLUE}Starting proxy...${NC}"
../../bin/auto-proxy -target http://httpbin.org -log-dir "$MOCK_DIR" -port $PROXY_PORT > /dev/null 2>&1 &
PROXY_PID=$!
sleep 2

echo -e "${YELLOW}Recording requests...${NC}"
echo "  1. GET /get with mock-id: api-v1"
curl -s -H "x-mock-id: api-v1" "http://127.0.0.1:$PROXY_PORT/get?version=1" > /dev/null
echo "  2. GET /uuid with mock-id: api-v1"
curl -s -H "x-mock-id: api-v1" "http://127.0.0.1:$PROXY_PORT/uuid" > /dev/null
echo "  3. POST /post with mock-id: api-v2"
curl -s -X POST -H "x-mock-id: api-v2" -H "Content-Type: application/json" \
  -d '{"test":"data"}' "http://127.0.0.1:$PROXY_PORT/post" > /dev/null
echo "  4. GET /get with mock-id: api-v2"
curl -s -H "x-mock-id: api-v2" "http://127.0.0.1:$PROXY_PORT/get?version=2" > /dev/null

echo -e "${BLUE}Stopping proxy...${NC}"
kill $PROXY_PID 2>/dev/null || true
sleep 1

# Show recorded structure
echo ""
echo -e "${GREEN}Recorded mocks:${NC}"
tree "$MOCK_DIR" 2>/dev/null || find "$MOCK_DIR" -type f -name "*.json"

# === PHASE 2: Mock Server ===
echo ""
echo -e "${GREEN}=== PHASE 2: Serving Mocks ===${NC}"
echo -e "${BLUE}Starting mock server...${NC}"
../../bin/auto-mock-server -mock-dir "$MOCK_DIR" -port $MOCK_PORT > /dev/null 2>&1 &
MOCK_PID=$!
sleep 2

echo -e "${YELLOW}Testing mock responses...${NC}"
echo ""

echo -e "${YELLOW}1. GET /get with mock-id: api-v1${NC}"
RESPONSE=$(curl -s -H "x-mock-id: api-v1" "http://127.0.0.1:$MOCK_PORT/get")
echo "  URL: $(echo $RESPONSE | jq -r '.url')"
echo "  Args: $(echo $RESPONSE | jq -r '.args')"

echo ""
echo -e "${YELLOW}2. GET /uuid with mock-id: api-v1${NC}"
UUID=$(curl -s -H "x-mock-id: api-v1" "http://127.0.0.1:$MOCK_PORT/uuid" | jq -r '.uuid')
echo "  UUID: $UUID"

echo ""
echo -e "${YELLOW}3. POST /post with mock-id: api-v2${NC}"
JSON_ECHO=$(curl -s -H "x-mock-id: api-v2" "http://127.0.0.1:$MOCK_PORT/post" | jq -r '.json')
echo "  JSON: $JSON_ECHO"

echo ""
echo -e "${YELLOW}4. GET /get with mock-id: api-v2${NC}"
URL2=$(curl -s -H "x-mock-id: api-v2" "http://127.0.0.1:$MOCK_PORT/get" | jq -r '.url')
echo "  URL: $URL2"

# Check stats
echo ""
echo -e "${GREEN}Mock server stats:${NC}"
curl -s "http://127.0.0.1:$MOCK_PORT/__mock__/stats" | jq '.'

echo ""
echo -e "${BLUE}Stopping mock server...${NC}"
kill $MOCK_PID 2>/dev/null || true

echo ""
echo -e "${GREEN}✓ Full workflow test complete!${NC}"
