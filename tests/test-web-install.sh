#!/bin/bash
# Test web-based install/uninstall scripts in isolated container

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Web Install/Uninstall Test Suite ===${NC}"
echo "Testing curl-based installation in isolated environment"
echo -e "${YELLOW}Note: Tests build from source, which may take a few minutes${NC}"
echo ""

FAILED_TESTS=0
PASSED_TESTS=0

# Test helper functions
test_start() {
    echo -e "${YELLOW}Testing: $1${NC}"
}

test_pass() {
    echo -e "${GREEN}✓ PASSED: $1${NC}"
    ((PASSED_TESTS++))
}

test_fail() {
    echo -e "${RED}✗ FAILED: $1${NC}"
    echo -e "${RED}  Error: $2${NC}"
    ((FAILED_TESTS++))
}

# Build test container
echo "Building test container..."
DOCKERFILE_PATH="$(pwd)/tests/Dockerfile.web-test"
cat > "$DOCKERFILE_PATH" << 'EOF'
FROM golang:1.21-alpine

# Install required packages
RUN apk add --no-cache bash curl git jq sudo nodejs npm

# Install Claude CLI (simulated)
RUN npm install -g @anthropic-ai/claude-code

# Create non-root user for testing
RUN adduser -D testuser && \
    echo "testuser ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

# Set up directories
RUN mkdir -p /usr/local/bin && \
    chmod 755 /usr/local/bin

WORKDIR /workspace

# Switch to non-root user
USER testuser
ENV HOME=/home/testuser
EOF

# Build container
if command -v podman >/dev/null 2>&1; then
    CONTAINER_CMD="podman"
else
    CONTAINER_CMD="docker"
fi

$CONTAINER_CMD build -f "$DOCKERFILE_PATH" -t cmcp-web-test .

# Test 1: Fresh web install
test_start "Web install (fresh)"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c "cd /tmp && bash /workspace/scripts/web-install.sh 2>&1")

if [[ "$OUTPUT" == *"cmcp installed successfully"* ]]; then
    test_pass "Web install completed"
else
    test_fail "Web install" "Installation failed: $OUTPUT"
fi

# Test 2: Web install with existing config
test_start "Web install with existing config"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c '
        # Create config
        mkdir -p ~/.cmcp
        cat > ~/.cmcp/config.json << "JSON"
{
  "mcpServers": {
    "test1": {"command": "test", "args": ["arg1"]},
    "test2": {"command": "test2", "args": ["arg2"]}
  }
}
JSON
        # Run install
        cd /tmp && bash /workspace/scripts/web-install.sh 2>&1 || true
    ')

if [[ "$OUTPUT" == *"Configuration preserved: 2 server(s) available"* ]]; then
    test_pass "Web install preserved existing config"
else
    test_fail "Config preservation" "Config not preserved: $OUTPUT"
fi

# Test 3: Web uninstall (keep config)
test_start "Web uninstall - keep config"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c '
        # Install first
        cd /tmp && bash /workspace/scripts/web-install.sh >/dev/null 2>&1 || true
        # Create config
        mkdir -p ~/.cmcp
        echo "{\"mcpServers\":{}}" > ~/.cmcp/config.json
        # Uninstall (pipe "n" to keep config)
        echo "n" | bash /workspace/scripts/web-uninstall.sh 2>&1 || true
    ')

if [[ "$OUTPUT" == *"Configuration preserved"* ]] && [[ "$OUTPUT" == *"successfully uninstalled"* ]]; then
    test_pass "Web uninstall kept config when requested"
else
    test_fail "Uninstall keep config" "Unexpected output: $OUTPUT"
fi

# Test 4: Web uninstall (remove config)
test_start "Web uninstall - remove config"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c '
        # Install first
        cd /tmp && bash /workspace/scripts/web-install.sh >/dev/null 2>&1 || true
        # Create config
        mkdir -p ~/.cmcp
        echo "{\"mcpServers\":{}}" > ~/.cmcp/config.json
        # Uninstall (pipe "y" to remove config)
        echo "y" | bash /workspace/scripts/web-uninstall.sh 2>&1 || true
    ')

if [[ "$OUTPUT" == *"Configuration removed"* ]] && [[ "$OUTPUT" == *"successfully uninstalled"* ]]; then
    test_pass "Web uninstall removed config when requested"
else
    test_fail "Uninstall remove config" "Unexpected output: $OUTPUT"
fi

# Test 5: Install from source (no releases)
test_start "Web install from source"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c "cd /tmp && cp -r /workspace . && cd workspace && bash web-install.sh 2>&1 || true")

if [[ "$OUTPUT" == *"Building cmcp"* ]] && [[ "$OUTPUT" == *"installed successfully"* ]]; then
    test_pass "Web install built from source"
else
    test_fail "Install from source" "Build failed: $OUTPUT"
fi

# Test 6: OS/Arch detection
test_start "OS/Architecture detection"
OUTPUT=$($CONTAINER_CMD run --rm \
    -v "$(pwd):/workspace:ro" \
    cmcp-web-test \
    bash -c "cd /tmp && bash /workspace/scripts/web-install.sh 2>&1 | grep 'Detected:' || true")

if [[ "$OUTPUT" == *"linux"* ]]; then
    test_pass "OS detection works"
else
    test_fail "OS detection" "Failed to detect OS: $OUTPUT"
fi

# Cleanup
rm -f "$DOCKERFILE_PATH"

# Summary
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 All web install/uninstall tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi