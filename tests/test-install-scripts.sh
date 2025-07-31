#!/bin/bash
# Test suite for install and uninstall scripts

# Don't exit on error - we want to run all tests
set +e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Use temp files to track test results across subshells
PASS_COUNT_FILE=/tmp/pass_count_$$
FAIL_COUNT_FILE=/tmp/fail_count_$$
echo "0" > "$PASS_COUNT_FILE"
echo "0" > "$FAIL_COUNT_FILE"

# Test helper functions
test_start() {
    echo -e "${YELLOW}Testing: $1${NC}"
}

test_pass() {
    echo -e "${GREEN}✓ PASSED: $1${NC}"
    local count=$(cat "$PASS_COUNT_FILE")
    echo $((count + 1)) > "$PASS_COUNT_FILE"
}

test_fail() {
    echo -e "${RED}✗ FAILED: $1${NC}"
    echo "  Error: $2"
    local count=$(cat "$FAIL_COUNT_FILE")
    echo $((count + 1)) > "$FAIL_COUNT_FILE"
}

# Setup test environment
ORIGINAL_PATH=$PATH
ORIGINAL_HOME=$HOME
TEST_HOME="/tmp/cmcp-test-home-$$"
TEST_BIN="/tmp/cmcp-test-bin-$$"

setup_test_env() {
    # Create test directories
    mkdir -p "$TEST_HOME"
    mkdir -p "$TEST_BIN"
    
    # Set test environment
    export HOME="$TEST_HOME"
    # Set PATH without the container's cmcp locations
    export PATH="$TEST_BIN:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    
    # Create a mock sudo that doesn't require password
    cat > "$TEST_BIN/sudo" << 'EOF'
#!/bin/bash
# Mock sudo for testing - execute the command as-is
exec "$@"
EOF
    chmod +x "$TEST_BIN/sudo"
    
    # Remove any existing cmcp from PATH to ensure clean test
    rm -f /app/cmcp /tmp/cmcp /usr/local/bin/cmcp 2>/dev/null || true
    # Also hide the real cmcp by creating a dummy that returns error
    cat > "$TEST_BIN/cmcp-hidden" << 'EOF'
#!/bin/bash
exit 1
EOF
    chmod +x "$TEST_BIN/cmcp-hidden"
}

cleanup_test_env() {
    # Restore environment
    export HOME="$ORIGINAL_HOME"
    export PATH="$ORIGINAL_PATH"
    
    # Clean up test directories
    rm -rf "$TEST_HOME"
    rm -rf "$TEST_BIN"
    
    # Restore backed up cmcp if it exists
    if [[ -f /app/cmcp.bak ]]; then
        mv /app/cmcp.bak /app/cmcp 2>/dev/null || true
    fi
}

# Trap to ensure cleanup
trap cleanup_test_env EXIT

echo "=== Install/Uninstall Scripts Test Suite ==="
echo "Testing script behaviors in isolated environment"
echo ""

# Setup
setup_test_env

# Copy source to writable location once
cp -r /app /tmp/cmcp-src

# Move existing cmcp aside temporarily
if command -v cmcp >/dev/null 2>&1; then
    CMCP_PATH=$(which cmcp)
    mv "$CMCP_PATH" "${CMCP_PATH}.bak" 2>/dev/null || true
fi

# Test 1: Fresh install without existing config
test_start "Fresh install - no existing config"
(
    cd /tmp/cmcp-src
    # Build first
    go build -o cmcp
    
    # Mock the install directory
    mkdir -p "$TEST_BIN/../local/bin"
    
    # Run install - should not detect existing cmcp
    OUTPUT=$(HOME="$TEST_HOME" ./scripts/install.sh 2>&1 | tail -20)
    
    if [[ "$OUTPUT" == *"cmcp installed successfully"* ]] && [[ ! "$OUTPUT" == *"Existing configuration found"* ]]; then
        test_pass "Fresh install completed without mentioning existing config"
    else
        test_fail "Fresh install" "Unexpected output: $OUTPUT"
    fi
)

# Test 2: Install with existing config
test_start "Install with existing config preservation"
(
    cd /tmp/cmcp-src
    # Create existing config
    mkdir -p "$TEST_HOME/.cmcp"
    cat > "$TEST_HOME/.cmcp/config.json" << 'EOF'
{
  "mcpServers": {
    "test-server": {
      "command": "test",
      "args": ["arg1"]
    },
    "another-server": {
      "command": "another",
      "args": ["arg2"]
    }
  }
}
EOF
    
    # Temporarily hide cmcp to test fresh install with existing config
    if command -v cmcp >/dev/null 2>&1; then
        CMCP_TMP=$(which cmcp)
        mv "$CMCP_TMP" "${CMCP_TMP}.tmp2" 2>/dev/null || true
    fi
    
    # Run install
    OUTPUT=$(HOME="$TEST_HOME" ./scripts/install.sh 2>&1 | tail -20)
    
    # Restore cmcp
    if [[ -f "${CMCP_TMP}.tmp2" ]]; then
        mv "${CMCP_TMP}.tmp2" "$CMCP_TMP" 2>/dev/null || true
    fi
    
    if [[ "$OUTPUT" == *"Configuration preserved: 2 server(s) available"* ]]; then
        # Verify config still exists
        if [[ -f "$TEST_HOME/.cmcp/config.json" ]]; then
            SERVER_COUNT=$(jq -r '.mcpServers | length' "$TEST_HOME/.cmcp/config.json")
            if [[ "$SERVER_COUNT" == "2" ]]; then
                test_pass "Install preserved existing configuration"
            else
                test_fail "Config preservation" "Server count changed: $SERVER_COUNT"
            fi
        else
            test_fail "Config preservation" "Config file was deleted"
        fi
    else
        test_fail "Install with config" "Did not detect existing config: $OUTPUT"
    fi
)

# Test 3: Uninstall prompt behavior - keep config
test_start "Uninstall - keep configuration (default N)"
(
    cd /tmp/cmcp-src
    # Setup: Install first
    mkdir -p "$TEST_BIN/../local/bin"
    cp cmcp "$TEST_BIN/../local/bin/"
    
    # Create config
    mkdir -p "$TEST_HOME/.cmcp"
    cat > "$TEST_HOME/.cmcp/config.json" << 'EOF'
{
  "mcpServers": {
    "test": {
      "command": "test"
    }
  }
}
EOF
    
    # Run uninstall with 'n' response (keep config)
    OUTPUT=$(echo "n" | HOME="$TEST_HOME" ./scripts/uninstall.sh 2>&1)
    
    if [[ "$OUTPUT" == *"Keeping configuration registry"* ]] && [[ -f "$TEST_HOME/.cmcp/config.json" ]]; then
        test_pass "Uninstall kept configuration when user selected 'n'"
    else
        test_fail "Uninstall keep config" "Config was removed or wrong message"
    fi
)

# Test 4: Uninstall prompt behavior - remove config
test_start "Uninstall - remove configuration (y)"
(
    cd /tmp/cmcp-src
    # Setup: Install first
    mkdir -p "$TEST_BIN/../local/bin"
    cp cmcp "$TEST_BIN/../local/bin/"
    
    # Create config
    mkdir -p "$TEST_HOME/.cmcp"
    cat > "$TEST_HOME/.cmcp/config.json" << 'EOF'
{
  "mcpServers": {
    "test": {
      "command": "test"
    }
  }
}
EOF
    
    # Run uninstall with 'y' response (remove config)
    OUTPUT=$(echo "y" | HOME="$TEST_HOME" ./scripts/uninstall.sh 2>&1)
    
    if [[ "$OUTPUT" == *"All server configurations removed"* ]] && [[ ! -d "$TEST_HOME/.cmcp" ]]; then
        test_pass "Uninstall removed configuration when user selected 'y'"
    else
        test_fail "Uninstall remove config" "Config still exists or wrong message"
    fi
)

# Test 5: Install as upgrade preserves configuration
test_start "Install as upgrade - configuration preservation"
(
    cd /tmp/cmcp-src
    # Setup: Install first
    mkdir -p "$TEST_BIN/../local/bin"
    cp cmcp "$TEST_BIN/../local/bin/"
    
    # Create config with 3 servers
    mkdir -p "$TEST_HOME/.cmcp"
    cat > "$TEST_HOME/.cmcp/config.json" << 'EOF'
{
  "mcpServers": {
    "server1": {
      "command": "cmd1",
      "args": ["arg1"]
    },
    "server2": {
      "command": "cmd2",
      "env": {
        "KEY": "value"
      }
    },
    "server3": {
      "command": "cmd3"
    }
  }
}
EOF
    
    # Make cmcp available in PATH for upgrade detection
    cat > "$TEST_BIN/cmcp" << 'EOF'
#!/bin/bash
if [[ "$1" == "help" ]]; then
    echo "cmcp version 1.0.0"
elif [[ "$1" == "online" ]]; then
    echo "No servers are currently running"
fi
exit 0
EOF
    chmod +x "$TEST_BIN/cmcp"
    
    # Run install (should detect as upgrade), 'n' to not stop servers
    OUTPUT=$(echo "n" | HOME="$TEST_HOME" ./scripts/install.sh 2>&1)
    
    if [[ "$OUTPUT" == *"Detected existing cmcp installation"* ]] && [[ "$OUTPUT" == *"upgraded successfully"* ]] && [[ "$OUTPUT" == *"Configuration preserved: 3 server(s) available"* ]]; then
        # Verify config still intact
        if [[ -f "$TEST_HOME/.cmcp/config.json" ]]; then
            SERVER_COUNT=$(jq -r '.mcpServers | length' "$TEST_HOME/.cmcp/config.json")
            HAS_ENV=$(jq -r '.mcpServers.server2.env.KEY' "$TEST_HOME/.cmcp/config.json")
            if [[ "$SERVER_COUNT" == "3" ]] && [[ "$HAS_ENV" == "value" ]]; then
                test_pass "Install as upgrade preserved all configuration including env vars"
            else
                test_fail "Upgrade config check" "Config was modified during upgrade"
            fi
        else
            test_fail "Upgrade preservation" "Config file missing after upgrade"
        fi
    else
        test_fail "Install as upgrade" "Wrong output or config not preserved: $OUTPUT"
    fi
)

# Test 6: Root permission messages
test_start "Root permission explanations"
(
    cd /tmp/cmcp-src
    
    # Check install script
    INSTALL_MSG=$(./scripts/install.sh 2>&1 | grep -A2 "Root permission required" || echo "")
    if [[ "$INSTALL_MSG" == *"Install the cmcp binary"* ]] && [[ "$INSTALL_MSG" == *"shell completions"* ]]; then
        test_pass "Install script explains root permissions"
    else
        test_fail "Install root message" "Missing or incorrect permission explanation"
    fi
    
    # Check uninstall script
    UNINSTALL_MSG=$(echo "n" | ./scripts/uninstall.sh 2>&1 | grep -A2 "Root permission will be required" || echo "")
    if [[ "$UNINSTALL_MSG" == *"Remove the cmcp binary"* ]] && [[ "$UNINSTALL_MSG" == *"shell completions"* ]]; then
        test_pass "Uninstall script explains root permissions"
    else
        test_fail "Uninstall root message" "Missing or incorrect permission explanation"
    fi
    
    # Check install script in upgrade mode
    # Create mock cmcp to trigger upgrade mode
    cat > "$TEST_BIN/cmcp" << 'EOF'
#!/bin/bash
echo "cmcp version test"
EOF
    chmod +x "$TEST_BIN/cmcp"
    
    INSTALL_UPGRADE_MSG=$(echo "n" | ./scripts/install.sh 2>&1 | grep -A2 "Root permission required" || echo "")
    if [[ "$INSTALL_UPGRADE_MSG" == *"Install the cmcp binary"* ]]; then
        test_pass "Install script (upgrade mode) explains root permissions"
    else
        test_fail "Install upgrade root message" "Missing or incorrect permission explanation"
    fi
)

# Test 7: Install script doesn't require existing cmcp
test_start "Install works without existing cmcp"
(
    cd /tmp/cmcp-src
    # Ensure cmcp is not in PATH
    rm -f "$TEST_BIN/cmcp"
    
    # Temporarily hide any existing cmcp
    HIDDEN_CMCP=""
    if command -v cmcp >/dev/null 2>&1; then
        HIDDEN_CMCP=$(which cmcp)
        mv "$HIDDEN_CMCP" "${HIDDEN_CMCP}.hidden7" 2>/dev/null || true
    fi
    
    OUTPUT=$(HOME="$TEST_HOME" ./scripts/install.sh 2>&1 | tail -5)
    
    # Restore cmcp
    if [[ -n "$HIDDEN_CMCP" ]] && [[ -f "${HIDDEN_CMCP}.hidden7" ]]; then
        mv "${HIDDEN_CMCP}.hidden7" "$HIDDEN_CMCP" 2>/dev/null || true
    fi
    if [[ "$OUTPUT" == *"cmcp installed successfully"* ]]; then
        test_pass "Install works without existing cmcp"
    else
        test_fail "Install without cmcp" "Installation failed"
    fi
)

# Test 8: Install shows different behavior when cmcp exists
test_start "Install detects existing cmcp vs fresh install"
(
    cd /tmp/cmcp-src
    # First test: no existing cmcp
    rm -f "$TEST_BIN/cmcp"
    rm -f "$TEST_BIN/../local/bin/cmcp"
    
    # Hide any existing cmcp
    HIDDEN_CMCP=""
    if command -v cmcp >/dev/null 2>&1; then
        HIDDEN_CMCP=$(which cmcp)
        mv "$HIDDEN_CMCP" "${HIDDEN_CMCP}.hidden8" 2>/dev/null || true
    fi
    
    OUTPUT=$(HOME="$TEST_HOME" ./scripts/install.sh 2>&1 | head -5)
    
    # Restore hidden cmcp
    if [[ -n "$HIDDEN_CMCP" ]] && [[ -f "${HIDDEN_CMCP}.hidden8" ]]; then
        mv "${HIDDEN_CMCP}.hidden8" "$HIDDEN_CMCP" 2>/dev/null || true
    fi
    
    if [[ "$OUTPUT" == *"Installing cmcp..."* ]] && [[ "$OUTPUT" != *"Detected existing"* ]]; then
        test_pass "Install correctly shows fresh install mode"
    else
        test_fail "Fresh install detection" "Should not detect existing installation"
    fi
    
    # Second test: with existing cmcp
    cat > "$TEST_BIN/cmcp" << 'EOF'
#!/bin/bash
if [[ "$1" == "help" ]]; then
    echo "cmcp version test"
fi
EOF
    chmod +x "$TEST_BIN/cmcp"
    
    OUTPUT=$(HOME="$TEST_HOME" ./scripts/install.sh 2>&1 | head -5)
    if [[ "$OUTPUT" == *"Detected existing cmcp installation"* ]]; then
        test_pass "Install correctly detects existing cmcp"
    else
        test_fail "Existing cmcp detection" "Should detect existing installation"
    fi
)

# Read final counts
TOTAL_PASSED=$(cat "$PASS_COUNT_FILE")
TOTAL_FAILED=$(cat "$FAIL_COUNT_FILE")

# Clean up temp files
rm -f "$PASS_COUNT_FILE" "$FAIL_COUNT_FILE"

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "${GREEN}Passed: $TOTAL_PASSED${NC}"
echo -e "${RED}Failed: $TOTAL_FAILED${NC}"

if [[ $TOTAL_FAILED -eq 0 ]] && [[ $TOTAL_PASSED -gt 0 ]]; then
    echo -e "${GREEN}🎉 All install/uninstall tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi