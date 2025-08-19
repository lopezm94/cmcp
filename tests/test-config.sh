#!/bin/bash

# CMCP Config Preservation Tests
# Tests that configuration fields (including unknown ones) are preserved
# RUNS ONLY IN CONTAINERS - NEVER ON LOCAL SYSTEM

# Safety check - ensure we're in a container or using test path
echo "Debug: CMCP_CONFIG_PATH='$CMCP_CONFIG_PATH'"
echo "Debug: Checking for /.dockerenv: $(ls -la /.dockerenv 2>&1 || echo 'not found')"
echo "Debug: Checking for /run/.containerenv: $(ls -la /run/.containerenv 2>&1 || echo 'not found')"

if [[ -z "$CMCP_CONFIG_PATH" ]] && [[ ! -f /.dockerenv ]] && [[ ! -f /run/.containerenv ]]; then
    echo "ERROR: This test must run in a container or with CMCP_CONFIG_PATH set"
    echo "Use ./test.sh config to run safely in a container"
    exit 1
fi

echo "Safety check passed - running tests..."

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

# Set test config path if not already set
if [[ -z "$CMCP_CONFIG_PATH" ]]; then
    export CMCP_CONFIG_PATH="/tmp/cmcp-test-config.json"
fi

# Clean up function
cleanup() {
    rm -rf "$CMCP_CONFIG_PATH" 2>/dev/null || true
    rm -rf /tmp/cmcp-test-* 2>/dev/null || true
}

# Set up clean environment
cleanup
trap cleanup EXIT

echo "=== CMCP Config Preservation Tests ==="
echo

# Test 1: CWD field preservation
test_start "CWD field preservation"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "crawl4ai-session": {
      "command": "uv",
      "args": ["run", "python", "mcp_crawl4ai.py"],
      "cwd": "/Users/test/Projects/workflow-editor/scraps"
    }
  }
}
EOF

# Load and save config through cmcp
echo "  Running: ${CMCP_BIN:-./cmcp} config open"
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || echo "  Command exit code: $?"

# Debug: Show what's in the config after operation
echo "  Config content after operation:"
cat "$CMCP_CONFIG_PATH" 2>/dev/null || echo "  ERROR: Config file not found at $CMCP_CONFIG_PATH"

# Check if cwd field is preserved
if jq -e '.mcpServers."crawl4ai-session".cwd == "/Users/test/Projects/workflow-editor/scraps"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_pass "CWD field preserved after load/save"
else
    test_fail "CWD field preservation" "CWD field was deleted or modified"
    echo "  Current config:"
    jq '.mcpServers."crawl4ai-session"' "$CMCP_CONFIG_PATH"
fi

# Test 2: Unknown fields preservation
test_start "Unknown fields preservation"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "future-server": {
      "command": "node",
      "args": ["server.js"],
      "env": {"NODE_ENV": "production"},
      "cwd": "/app",
      "timeout": 5000,
      "retries": 3,
      "experimental": true,
      "metadata": {
        "version": "2.0",
        "features": ["auth", "logging"]
      }
    }
  }
}
EOF

# Save original for comparison
cp "$CMCP_CONFIG_PATH" /tmp/cmcp-test-original.json

# Trigger load/save
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || true

# Check all fields are preserved
FIELDS_OK=true
for field in timeout retries experimental metadata; do
    if ! jq -e ".mcpServers.\"future-server\".$field" "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        test_fail "Unknown field preservation" "Field '$field' was deleted"
        FIELDS_OK=false
        break
    fi
done

if [[ "$FIELDS_OK" == "true" ]]; then
    # Check nested metadata
    if jq -e '.mcpServers."future-server".metadata.version == "2.0"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1 && \
       jq -e '.mcpServers."future-server".metadata.features[0] == "auth"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        test_pass "All unknown fields preserved including nested structures"
    else
        test_fail "Nested field preservation" "Nested metadata structure was modified"
    fi
fi

# Test 3: Mixed known and unknown fields
test_start "Mixed known and unknown fields"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "mixed-server": {
      "command": "python",
      "args": ["-m", "server"],
      "env": {"API_KEY": "secret"},
      "cwd": "/workspace",
      "customField1": "value1",
      "customField2": 42,
      "customField3": true,
      "customField4": ["a", "b", "c"],
      "customField5": {
        "nested": "object"
      }
    }
  }
}
EOF

# Trigger load/save
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || true

# Check known fields
KNOWN_OK=true
if ! jq -e '.mcpServers."mixed-server".command == "python"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_fail "Mixed fields" "Known field 'command' was modified"
    KNOWN_OK=false
fi

if ! jq -e '.mcpServers."mixed-server".cwd == "/workspace"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_fail "Mixed fields" "Known field 'cwd' was modified"
    KNOWN_OK=false
fi

# Check unknown fields
UNKNOWN_OK=true
for i in 1 2 3 4 5; do
    if ! jq -e ".mcpServers.\"mixed-server\".customField$i" "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        test_fail "Mixed fields" "Unknown field 'customField$i' was deleted"
        UNKNOWN_OK=false
        break
    fi
done

if [[ "$KNOWN_OK" == "true" ]] && [[ "$UNKNOWN_OK" == "true" ]]; then
    test_pass "Both known and unknown fields preserved correctly"
fi

# Test 4: Empty/null field handling
test_start "Empty and null field handling"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "edge-case-server": {
      "command": "test",
      "args": [],
      "env": {},
      "cwd": "",
      "nullField": null,
      "emptyString": "",
      "emptyArray": [],
      "emptyObject": {}
    }
  }
}
EOF

# Trigger load/save
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || true

# Check that empty values are handled correctly
EMPTY_OK=true

# Check if empty cwd is preserved (should be omitted or empty string)
if jq -e '.mcpServers."edge-case-server" | has("cwd")' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    CWD_VAL=$(jq -r '.mcpServers."edge-case-server".cwd' "$CMCP_CONFIG_PATH")
    if [[ "$CWD_VAL" != "" ]] && [[ "$CWD_VAL" != "null" ]]; then
        test_fail "Empty field handling" "Empty cwd field has unexpected value: $CWD_VAL"
        EMPTY_OK=false
    fi
fi

# Check other empty fields
for field in nullField emptyString emptyArray emptyObject; do
    if ! jq -e ".mcpServers.\"edge-case-server\" | has(\"$field\")" "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
        # Field might be omitted which is okay for empty values
        echo "  Note: Field '$field' was omitted (acceptable for empty values)"
    fi
done

if [[ "$EMPTY_OK" == "true" ]]; then
    test_pass "Empty and null fields handled correctly"
fi

# Test 5: Complex nested structure preservation
test_start "Complex nested structure preservation"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "complex-server": {
      "command": "docker",
      "args": ["run", "image:latest"],
      "env": {
        "KEY1": "value1",
        "KEY2": "value2"
      },
      "cwd": "/docker/workspace",
      "docker": {
        "image": "node:18",
        "volumes": [
          "/host/path:/container/path",
          "/another:/path"
        ],
        "ports": {
          "3000": "3000",
          "8080": "80"
        },
        "network": "bridge"
      },
      "healthcheck": {
        "endpoint": "/health",
        "interval": 30,
        "timeout": 5,
        "retries": 3
      }
    }
  }
}
EOF

# Save a hash of the original structure
ORIGINAL_HASH=$(jq -S '.mcpServers."complex-server"' "$CMCP_CONFIG_PATH" | sha256sum | cut -d' ' -f1)

# Trigger load/save
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || true

# Get hash of the saved structure
SAVED_HASH=$(jq -S '.mcpServers."complex-server"' "$CMCP_CONFIG_PATH" | sha256sum | cut -d' ' -f1)

if [[ "$ORIGINAL_HASH" == "$SAVED_HASH" ]]; then
    test_pass "Complex nested structure preserved exactly"
else
    test_fail "Complex structure preservation" "Structure was modified during load/save"
    echo "  Original hash: $ORIGINAL_HASH"
    echo "  Saved hash: $SAVED_HASH"
fi

# Test 6: Multiple servers with different fields
test_start "Multiple servers with different fields"
cat > "$CMCP_CONFIG_PATH" << 'EOF'
{
  "mcpServers": {
    "server1": {
      "command": "node",
      "args": ["app.js"],
      "customField": "server1-specific"
    },
    "server2": {
      "command": "python",
      "cwd": "/python/app",
      "timeout": 60
    },
    "server3": {
      "command": "ruby",
      "env": {"RUBY_ENV": "production"},
      "debugMode": true,
      "logLevel": "info"
    }
  }
}
EOF

# Trigger load/save
${CMCP_BIN:-./cmcp} config open >/dev/null 2>&1 || true

# Check each server maintains its unique fields
ALL_OK=true

if ! jq -e '.mcpServers.server1.customField == "server1-specific"' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_fail "Multiple servers" "server1 lost its customField"
    ALL_OK=false
fi

if ! jq -e '.mcpServers.server2.timeout == 60' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_fail "Multiple servers" "server2 lost its timeout field"
    ALL_OK=false
fi

if ! jq -e '.mcpServers.server3.debugMode == true' "$CMCP_CONFIG_PATH" >/dev/null 2>&1; then
    test_fail "Multiple servers" "server3 lost its debugMode field"
    ALL_OK=false
fi

if [[ "$ALL_OK" == "true" ]]; then
    test_pass "All servers maintain their unique fields"
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