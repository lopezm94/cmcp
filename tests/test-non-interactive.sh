#!/bin/bash

# CMCP Non-Interactive Mode Tests
# Tests commands with arguments instead of interactive prompts

# Auto-generate unique paths based on script name
TEST_NAME=$(basename "$0" .sh | sed 's/^test-//')
export CMCP_CONFIG_PATH="/tmp/cmcp-test-${TEST_NAME}/config.json"
export TEST_DIR="/tmp/cmcp-test-${TEST_NAME}"

# Setup test environment  
mkdir -p "$TEST_DIR"
mkdir -p "$(dirname "$CMCP_CONFIG_PATH")"

# Ensure config directory exists for all operations
mkdir -p "$(dirname "$CMCP_CONFIG_PATH")"

# Use the provided binary or default
CMCP="${CMCP_BIN:-./cmcp}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test functions
test_start() {
    echo -e "${YELLOW}▶ Testing: $1${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
}

test_pass() {
    echo -e "${GREEN}✓ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

test_fail() {
    echo -e "${RED}✗ $1${NC}"
    echo "  Error: $2"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Clean up function
cleanup() {
    rm -rf "$TEST_DIR" 2>/dev/null || true
}

# Set up trap for cleanup on exit
trap cleanup EXIT

echo "=== CMCP Non-Interactive Mode Tests ==="
echo

# Setup: Create test configuration
test_start "Setup test configuration"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "test-server1": {
      "command": "node",
      "args": ["server1.js"]
    },
    "test-server2": {
      "command": "python",
      "args": ["server2.py"]
    },
    "test-server3": {
      "command": "ruby",
      "args": ["server3.rb"]
    }
  }
}
EOF

if [[ -f "$CMCP_CONFIG_PATH" ]]; then
    test_pass "Test configuration created"
else
    test_fail "Setup" "Failed to create test configuration"
    exit 1
fi

# Test 1: config rm with single server (non-interactive)
test_start "config rm - single server non-interactive"
OUTPUT=$("$CMCP" config rm test-server1 <<< "y" 2>&1)
if echo "$OUTPUT" | grep -q "Successfully removed"; then
    # Verify server was actually removed
    if jq -e '.mcpServers."test-server1"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        test_fail "config rm single" "Server still exists in config"
    else
        test_pass "Single server removed non-interactively"
    fi
else
    test_fail "config rm single" "Command failed"
fi

# Test 2: config rm with multiple servers (non-interactive)
test_start "config rm - multiple servers non-interactive"
OUTPUT=$("$CMCP" config rm test-server2 test-server3 <<< "y" 2>&1)
if echo "$OUTPUT" | grep -q "Successfully removed 2 server"; then
    # Verify servers were removed
    if jq -e '.mcpServers."test-server2"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1 || \
       jq -e '.mcpServers."test-server3"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        test_fail "config rm multiple" "Some servers still exist in config"
    else
        test_pass "Multiple servers removed non-interactively"
    fi
else
    test_fail "config rm multiple" "Command failed"
    echo "  Output: $OUTPUT"
fi

# Test 3: config rm with non-existent server
test_start "config rm - non-existent server"
# Recreate config for this test
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "real-server": {
      "command": "node",
      "args": ["server.js"]
    }
  }
}
EOF

OUTPUT=$("$CMCP" config rm non-existent 2>&1)
if echo "$OUTPUT" | grep -q "not found in configuration"; then
    test_pass "Properly handles non-existent server"
else
    test_fail "config rm non-existent" "Should have reported server not found"
    echo "  Output: $OUTPUT"
fi

# Test 4: start command non-interactive mode
test_start "start - non-interactive with server names"
# Note: We can't actually test starting servers without Claude, but we can test the command accepts arguments
OUTPUT=$("$CMCP" start real-server --dry-run 2>&1)
if echo "$OUTPUT" | grep -q "Would execute" || echo "$OUTPUT" | grep -q "not found"; then
    test_pass "Start command accepts server names as arguments"
else
    test_fail "start non-interactive" "Command doesn't seem to accept arguments"
    echo "  Output: $OUTPUT"
fi

# Test 5: stop command non-interactive mode
test_start "stop - non-interactive with server names"
OUTPUT=$("$CMCP" stop real-server --dry-run 2>&1)
if echo "$OUTPUT" | grep -q "Would execute" || echo "$OUTPUT" | grep -q "not running"; then
    test_pass "Stop command accepts server names as arguments"
else
    test_fail "stop non-interactive" "Command doesn't seem to accept arguments"
    echo "  Output: $OUTPUT"
fi

# Test 6: Multiple operations in sequence
test_start "Sequential non-interactive operations"
# Create fresh config
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "seq-test1": {
      "command": "node",
      "args": ["test1.js"]
    },
    "seq-test2": {
      "command": "python",
      "args": ["test2.py"]
    },
    "seq-test3": {
      "command": "ruby",
      "args": ["test3.rb"]
    }
  }
}
EOF

# Remove servers in sequence
SUCCESS=true
for server in seq-test1 seq-test2 seq-test3; do
    if ! "$CMCP" config rm "$server" <<< "y" >/dev/null 2>&1; then
        SUCCESS=false
        break
    fi
done

if [[ "$SUCCESS" == "true" ]]; then
    # Check all servers are gone
    SERVER_COUNT=$(jq -r '.mcpServers | length' "$CMCP_CONFIG_PATH" 2>/dev/null || echo "0")
    if [[ "$SERVER_COUNT" == "0" ]]; then
        test_pass "Sequential operations work correctly"
    else
        test_fail "Sequential operations" "Some servers remain: $SERVER_COUNT"
    fi
else
    test_fail "Sequential operations" "Failed to remove servers sequentially"
fi

# Restore original config if it existed
if [[ -f "$CMCP_CONFIG_PATH".bak ]]; then
    mv "$CMCP_CONFIG_PATH".bak "$CMCP_CONFIG_PATH"
else
    rm -f "$CMCP_CONFIG_PATH"
fi

# Print summary
echo
echo "=== Test Summary ==="
echo "Tests run: $TESTS_RUN"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi