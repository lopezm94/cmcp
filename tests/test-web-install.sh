#!/bin/bash
# Test web-based install/uninstall scripts - optimized for running inside test container
# This version runs sequentially with cleanup between tests, no separate container needed

set +e

# Auto-generate unique paths based on script name
TEST_NAME=$(basename "$0" .sh | sed 's/^test-//')
export CMCP_CONFIG_PATH="/tmp/cmcp-test-${TEST_NAME}/config.json"
export TEST_DIR="/tmp/cmcp-test-${TEST_NAME}"

# Setup test environment
mkdir -p "$TEST_DIR"
mkdir -p "$(dirname "$CMCP_CONFIG_PATH")"

# Use CMCP_BIN if provided, otherwise use ./cmcp
CMCP="${CMCP_BIN:-./cmcp}"

# Cleanup on exit
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Web Install/Uninstall Test Suite ===${NC}"
echo "Testing curl-based installation with environment cleanup"
echo ""

# Use temp files to track test results across subshells
PASS_COUNT_FILE=/tmp/web_pass_count_$$
FAIL_COUNT_FILE=/tmp/web_fail_count_$$
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
    echo -e "${RED}  Error: $2${NC}"
    local count=$(cat "$FAIL_COUNT_FILE")
    echo $((count + 1)) > "$FAIL_COUNT_FILE"
}

# Cleanup function to ensure clean environment between tests
cleanup_environment() {
    # Remove cmcp binary
    sudo rm -f /usr/local/bin/cmcp 2>/dev/null || true
    
    # Remove config directory
    rm -rf ~/.cmcp 2>/dev/null || true
    
    # Clean up any temp directories from web scripts
    rm -rf /tmp/cmcp-install-* 2>/dev/null || true
    rm -rf /tmp/cmcp-uninstall-* 2>/dev/null || true
    
    # Ensure cmcp is not in PATH
    hash -r
}

# Mock git clone to use local source instead of downloading
mock_git_clone() {
    local REPO_URL="$1"
    local TARGET_DIR="${2:-.}"
    
    if [[ "$REPO_URL" == *"github.com/lopezm94/cmcp"* ]]; then
        # Copy local source to target directory (excluding the binary)
        cp -r /app/* "$TARGET_DIR/" 2>/dev/null || true
        # Remove the macOS binary if it was copied
        rm -f "$TARGET_DIR/cmcp" 2>/dev/null || true
        return 0
    else
        # Fall back to real git clone for other repos
        git clone "$REPO_URL" "$TARGET_DIR"
    fi
}

# Override git command for this script
git() {
    if [[ "$1" == "clone" ]]; then
        shift
        mock_git_clone "$@"
    else
        command git "$@"
    fi
}

# Export the function so it's available to subshells
export -f git
export -f mock_git_clone

# Ensure we start clean
cleanup_environment

# Remove /app from PATH to avoid macOS binary conflicts
export PATH=$(echo "$PATH" | sed 's|/app:||g')

# Test 1: Fresh web install
test_start "Web install (fresh)"
(
    cd /tmp
    OUTPUT=$(bash /app/scripts/web-install.sh 2>&1)
    
    if [[ "$OUTPUT" == *"cmcp installed successfully"* ]] || [[ "$OUTPUT" == *"cmcp upgraded successfully"* ]]; then
        if command -v cmcp >/dev/null 2>&1; then
            test_pass "Web install completed"
        else
            test_fail "Web install" "cmcp not found in PATH after installation"
        fi
    else
        test_fail "Web install" "Installation failed or cmcp not found: $OUTPUT"
    fi
)
cleanup_environment

# Test 2: Web install with existing config
test_start "Web install with existing config"
(
    # Create config
    mkdir -p ~/.cmcp
    cat > "$CMCP_CONFIG_PATH" << 'JSON'
{
  "mcpServers": {
    "test1": {"command": "test", "args": ["arg1"]},
    "test2": {"command": "test2", "args": ["arg2"]}
  }
}
JSON
    
    cd /tmp
    OUTPUT=$(bash /app/scripts/web-install.sh 2>&1)
    
    if [[ "$OUTPUT" == *"Configuration preserved: 2 server(s) available"* ]]; then
        # Verify config still exists
        if [[ -f "$CMCP_CONFIG_PATH" ]] && grep -q "test1" "$CMCP_CONFIG_PATH"; then
            test_pass "Web install preserved existing config"
        else
            test_fail "Config preservation" "Config file was modified"
        fi
    else
        test_fail "Config preservation" "Did not preserve config: $OUTPUT"
    fi
)
cleanup_environment

# Test 3: Web uninstall (keep config)
test_start "Web uninstall - keep config"
(
    # Install first
    cd /tmp
    bash /app/scripts/web-install.sh >/dev/null 2>&1 || true
    
    # Create config
    mkdir -p ~/.cmcp
    echo '{"mcpServers":{}}' > "$CMCP_CONFIG_PATH"
    
    # Uninstall (pipe "n" to keep config)
    OUTPUT=$(echo "n" | bash /app/scripts/web-uninstall.sh 2>&1)
    
    if [[ "$OUTPUT" == *"Keeping configuration registry"* ]] && 
       [[ "$OUTPUT" == *"uninstalled successfully"* ]] && 
       [[ -f "$CMCP_CONFIG_PATH" ]]; then
        test_pass "Web uninstall kept config when requested"
    else
        test_fail "Uninstall keep config" "Config removed or wrong output: $OUTPUT"
    fi
)
cleanup_environment

# Test 4: Web uninstall (remove config)
test_start "Web uninstall - remove config"
(
    # Install first
    cd /tmp
    bash /app/scripts/web-install.sh >/dev/null 2>&1 || true
    
    # Create config
    mkdir -p ~/.cmcp
    echo '{"mcpServers":{}}' > "$CMCP_CONFIG_PATH"
    
    # Uninstall (pipe "y" to remove config)
    OUTPUT=$(echo "y" | bash /app/scripts/web-uninstall.sh 2>&1)
    
    if [[ "$OUTPUT" == *"All server configurations removed"* ]] && 
       [[ "$OUTPUT" == *"uninstalled successfully"* ]] && 
       [[ ! -d ~/.cmcp ]]; then
        test_pass "Web uninstall removed config when requested"
    else
        test_fail "Uninstall remove config" "Config still exists or wrong output: $OUTPUT"
    fi
)
cleanup_environment

# Test 5: Web install error handling (no git)
test_start "Web install error - no git available"
(
    # Unset our git function for this test
    unset -f git
    
    # Temporarily hide real git
    GIT_PATH=$(which git)
    sudo mv "$GIT_PATH" "${GIT_PATH}.bak" 2>/dev/null || true
    
    cd /tmp
    OUTPUT=$(bash /app/scripts/web-install.sh 2>&1 || true)
    
    # Restore git
    sudo mv "${GIT_PATH}.bak" "$GIT_PATH" 2>/dev/null || true
    
    if [[ "$OUTPUT" == *"Git is required"* ]]; then
        test_pass "Web install correctly requires git"
    else
        test_fail "Git requirement" "Did not detect missing git: $OUTPUT"
    fi
)
cleanup_environment

# Test 6: Web uninstall when cmcp not installed
test_start "Web uninstall with no cmcp"
(
    cd /tmp
    OUTPUT=$(bash /app/scripts/web-uninstall.sh 2>&1 || true)
    
    if [[ "$OUTPUT" == *"Binary not found"* ]] || [[ "$OUTPUT" == *"cmcp is not installed"* ]] || [[ "$OUTPUT" == *"cmcp uninstalled successfully"* ]]; then
        test_pass "Web uninstall correctly handles missing cmcp"
    else
        test_fail "Uninstall no cmcp" "Unexpected output: $OUTPUT"
    fi
)
cleanup_environment

# Read final counts
TOTAL_PASSED=$(cat "$PASS_COUNT_FILE")
TOTAL_FAILED=$(cat "$FAIL_COUNT_FILE")

# Clean up temp files
rm -f "$PASS_COUNT_FILE" "$FAIL_COUNT_FILE"

# Summary
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "${GREEN}Passed: ${TOTAL_PASSED}${NC}"
echo -e "${RED}Failed: ${TOTAL_FAILED}${NC}"

if [ $TOTAL_FAILED -eq 0 ] && [ $TOTAL_PASSED -gt 0 ]; then
    echo -e "${GREEN}🎉 All web install/uninstall tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi