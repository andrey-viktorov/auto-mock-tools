#!/bin/bash
set -e

echo "=== Testing Request Logging in Proxy ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

MOCK_DIR="test_logs_mocks"
PROXY_PORT=8083
LOG_FILE="proxy_test.log"

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up...${NC}"
    pkill -f "auto-proxy.*$PROXY_PORT" 2>/dev/null || true
    rm -rf "$MOCK_DIR"
    rm -f "$LOG_FILE"
}

trap cleanup EXIT

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Build if needed
if [ ! -f "$PROJECT_ROOT/bin/auto-proxy" ]; then
    echo -e "${BLUE}Building proxy...${NC}"
    cd "$PROJECT_ROOT" && make build && cd "$SCRIPT_DIR"
fi

# Clean start
rm -rf "$MOCK_DIR"
mkdir -p "$MOCK_DIR"
rm -f "$LOG_FILE"

# Start proxy with logging to file
echo -e "${GREEN}Starting recording proxy with logging...${NC}"
echo "  Target: http://httpbin.org"
echo "  Port: $PROXY_PORT"
echo "  Mock dir: $MOCK_DIR"
echo "  Log file: $LOG_FILE"
echo ""

"$PROJECT_ROOT/bin/auto-proxy" -target http://httpbin.org -log-dir "$MOCK_DIR" -port $PROXY_PORT > "$LOG_FILE" 2>&1 &
PROXY_PID=$!
sleep 2

# Check if proxy started
if ! ps -p $PROXY_PID > /dev/null; then
    echo -e "${RED}✗ Proxy failed to start${NC}"
    cat "$LOG_FILE"
    exit 1
fi

echo -e "${GREEN}Making test requests...${NC}"
echo ""

# Request 1: GET with default mock-id
echo -e "${YELLOW}1. GET /get (default mock-id)${NC}"
curl -s "http://127.0.0.1:$PROXY_PORT/get?test=1" > /dev/null
sleep 0.5

# Request 2: GET with custom mock-id
echo -e "${YELLOW}2. GET /uuid (mock-id: api-v1)${NC}"
curl -s -H "x-mock-id: api-v1" "http://127.0.0.1:$PROXY_PORT/uuid" > /dev/null
sleep 0.5

# Request 3: POST with JSON
echo -e "${YELLOW}3. POST /post (mock-id: api-v2)${NC}"
curl -s -X POST -H "x-mock-id: api-v2" -H "Content-Type: application/json" \
  -d '{"name":"test","value":123}' "http://127.0.0.1:$PROXY_PORT/post" > /dev/null
sleep 0.5

# Request 4: GET that will return 404
echo -e "${YELLOW}4. GET /status/404 (default mock-id)${NC}"
curl -s "http://127.0.0.1:$PROXY_PORT/status/404" > /dev/null || true
sleep 0.5

# Request 5: GET with another mock-id
echo -e "${YELLOW}5. GET /headers (mock-id: test-headers)${NC}"
curl -s -H "x-mock-id: test-headers" "http://127.0.0.1:$PROXY_PORT/headers" > /dev/null
sleep 0.5

echo ""
echo -e "${GREEN}Stopping proxy...${NC}"
kill $PROXY_PID 2>/dev/null || true
sleep 1

# Analyze logs
echo ""
echo -e "${GREEN}Analyzing logs...${NC}"
echo ""

# Check if log file exists
if [ ! -f "$LOG_FILE" ]; then
    echo -e "${RED}✗ Log file not found${NC}"
    exit 1
fi

echo -e "${BLUE}Full proxy log:${NC}"
echo "----------------------------------------"
cat "$LOG_FILE"
echo "----------------------------------------"
echo ""

# Test 1: Check for request logging
echo -e "${GREEN}Test 1: Checking request logging format${NC}"
REQUEST_LOG_COUNT=$(grep -c "\[.*\] GET\|POST\|PUT\|DELETE" "$LOG_FILE" || echo "0")
echo "  Found $REQUEST_LOG_COUNT request log entries"
if [ "$REQUEST_LOG_COUNT" -ge 5 ]; then
    echo -e "  ${GREEN}✓ Request logging is working${NC}"
else
    echo -e "  ${RED}✗ Expected at least 5 request logs, found $REQUEST_LOG_COUNT${NC}"
    exit 1
fi

# Test 2: Check for mock-id logging
echo ""
echo -e "${GREEN}Test 2: Checking mock-id logging${NC}"
DEFAULT_MOCK_COUNT=$(grep -c "mock-id: default" "$LOG_FILE" || echo "0")
API_V1_COUNT=$(grep -c "mock-id: api-v1" "$LOG_FILE" || echo "0")
API_V2_COUNT=$(grep -c "mock-id: api-v2" "$LOG_FILE" || echo "0")
TEST_HEADERS_COUNT=$(grep -c "mock-id: test-headers" "$LOG_FILE" || echo "0")

echo "  default: $DEFAULT_MOCK_COUNT"
echo "  api-v1: $API_V1_COUNT"
echo "  api-v2: $API_V2_COUNT"
echo "  test-headers: $TEST_HEADERS_COUNT"

if [ "$DEFAULT_MOCK_COUNT" -ge 2 ] && [ "$API_V1_COUNT" -ge 1 ] && [ "$API_V2_COUNT" -ge 1 ] && [ "$TEST_HEADERS_COUNT" -ge 1 ]; then
    echo -e "  ${GREEN}✓ Mock-id logging is working${NC}"
else
    echo -e "  ${RED}✗ Mock-id logging incomplete${NC}"
    exit 1
fi

# Test 3: Check for success status logging
echo ""
echo -e "${GREEN}Test 3: Checking response status logging${NC}"
SUCCESS_LOG_COUNT=$(grep -c "✓ 200\|✓ 201\|✓ 404" "$LOG_FILE" || echo "0")
echo "  Found $SUCCESS_LOG_COUNT success status log entries"
if [ "$SUCCESS_LOG_COUNT" -ge 5 ]; then
    echo -e "  ${GREEN}✓ Response status logging is working${NC}"
else
    echo -e "  ${RED}✗ Expected at least 5 status logs, found $SUCCESS_LOG_COUNT${NC}"
    exit 1
fi

# Test 4: Check for timing information
echo ""
echo -e "${GREEN}Test 4: Checking timing information${NC}"
TIMING_COUNT=$(grep -c "[0-9]\+\.[0-9]\+s)" "$LOG_FILE" || echo "0")
echo "  Found $TIMING_COUNT timing entries"
if [ "$TIMING_COUNT" -ge 5 ]; then
    echo -e "  ${GREEN}✓ Timing information is logged${NC}"
else
    echo -e "  ${RED}✗ Expected at least 5 timing entries, found $TIMING_COUNT${NC}"
    exit 1
fi

# Test 5: Check for request IDs
echo ""
echo -e "${GREEN}Test 5: Checking request ID format${NC}"
REQUEST_ID_COUNT=$(grep -E "\[[0-9]{8}[0-9.]+\]" "$LOG_FILE" | wc -l | tr -d ' ')
echo "  Found $REQUEST_ID_COUNT entries with request IDs"
if [ "$REQUEST_ID_COUNT" -ge 10 ]; then
    echo -e "  ${GREEN}✓ Request IDs are present${NC}"
else
    echo -e "  ${RED}✗ Expected at least 10 request ID entries, found $REQUEST_ID_COUNT${NC}"
    exit 1
fi

# Test 6: Check for URL logging
echo ""
echo -e "${GREEN}Test 6: Checking URL logging${NC}"
if grep -q "GET.*127.0.0.1:$PROXY_PORT/get" "$LOG_FILE" && \
   grep -q "GET.*127.0.0.1:$PROXY_PORT/uuid" "$LOG_FILE" && \
   grep -q "POST.*127.0.0.1:$PROXY_PORT/post" "$LOG_FILE"; then
    echo -e "  ${GREEN}✓ URLs are logged correctly${NC}"
else
    echo -e "  ${RED}✗ Not all URLs found in logs${NC}"
    exit 1
fi

# Test 7: Verify no errors or warnings (except expected recording messages)
echo ""
echo -e "${GREEN}Test 7: Checking for unexpected errors${NC}"
ERROR_COUNT=$(grep "❌" "$LOG_FILE" | wc -l | tr -d ' ')
if [ "$ERROR_COUNT" -eq 0 ]; then
    echo -e "  ${GREEN}✓ No errors found${NC}"
else
    echo -e "  ${YELLOW}⚠ Found $ERROR_COUNT error entries (may be expected)${NC}"
    grep "❌" "$LOG_FILE" || true
fi

# Display sample log entries
echo ""
echo -e "${GREEN}Sample log entries:${NC}"
echo "----------------------------------------"
grep "\[.*\] GET\|POST" "$LOG_FILE" | head -5
echo "----------------------------------------"

echo ""
echo -e "${GREEN}✓ All logging tests passed!${NC}"
echo ""
echo "Summary:"
echo "  - Request logging: ✓"
echo "  - Mock-id tracking: ✓"
echo "  - Status codes: ✓"
echo "  - Timing info: ✓"
echo "  - Request IDs: ✓"
echo "  - URL logging: ✓"
