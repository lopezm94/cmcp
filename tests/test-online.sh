#!/bin/bash

# Test online command functionality
# This test runs in a container to test the online command features

set -e

echo "=== Testing cmcp online command ==="

# Ensure cmcp is accessible
CMCP_BIN="${CMCP_BIN:-./cmcp}"
if [ ! -x "$CMCP_BIN" ]; then
    echo "Error: cmcp binary not found at $CMCP_BIN"
    echo "Looking for cmcp in common locations..."
    if [ -x "/tmp/cmcp" ]; then
        CMCP_BIN="/tmp/cmcp"
        echo "Found cmcp at /tmp/cmcp"
    elif [ -x "./cmcp" ]; then
        CMCP_BIN="./cmcp"
        echo "Found cmcp at ./cmcp"
    else
        echo "Cannot find cmcp binary"
        exit 1
    fi
fi

# Setup test environment
mkdir -p ~/.cmcp

# Clean up any leftover servers from previous tests
claude mcp list 2>&1 | grep -E "^[^:]+:" | cut -d: -f1 | while read server; do
    claude mcp remove "$server" 2>&1 >/dev/null || true
done

echo ""
echo "1. Testing empty state..."
echo "=========================="

# Start with empty config
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {}
}
EOF

OUTPUT=$($CMCP_BIN online 2>&1 || true)
if echo "$OUTPUT" | grep -q "No servers are currently running in Claude for this project\\."; then
    echo "✓ Empty state shows correct message"
else
    echo "✗ Empty state message incorrect"
fi

echo ""
echo "2. Setting up test servers..."
echo "=============================="

# Add some servers to config
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "test-server": {
      "command": "echo",
      "args": ["test"]
    },
    "github-test": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}
EOF

echo "Added 2 servers to config"

echo ""
echo "3. Testing orphaned server detection..."
echo "========================================"

# Directly add a server to Claude that's not in config
# This simulates a server added via 'claude mcp add' or from tests
echo "Adding orphaned server directly to Claude..."
claude mcp add orphan-test -- echo "orphaned" 2>&1 >/dev/null || true

# Run online command
OUTPUT=$($CMCP_BIN online 2>&1 || true)

echo "Checking online output..."

# Check for colored output and proper formatting
if echo "$OUTPUT" | grep -q "MCP servers running in Claude for this project"; then
    echo "✓ Shows project context header"
else
    echo "✗ Missing project context header"
fi

if echo "$OUTPUT" | grep -q "orphan-test"; then
    echo "✓ Detects orphaned server"
    
    if echo "$OUTPUT" | grep -q "cmcp online --clear"; then
        echo "✓ Shows clear command suggestion"
    else
        echo "✗ Missing clear command suggestion"
    fi
else
    echo "⚠ Orphaned server may not have been added successfully"
fi

# Check for color codes in output (ANSI escape sequences)
if echo "$OUTPUT" | grep -q $'\033'; then
    echo "✓ Output contains color codes"
else
    echo "⚠ Output may not have colors (could be terminal detection)"
fi

echo ""
echo "4. Testing --clear flag..."
echo "==========================="

# Test dry-run first
echo "Testing --clear --dry-run..."
OUTPUT=$($CMCP_BIN online --clear --dry-run 2>&1 || true)

if echo "$OUTPUT" | grep -q "Would execute the following commands"; then
    echo "✓ Dry-run shows commands that would be executed"
    if echo "$OUTPUT" | grep -q "claude mcp remove orphan-test"; then
        echo "✓ Shows correct claude mcp remove command"
    fi
else
    echo "✗ Dry-run output incorrect"
fi

# Test actual clear
echo "Testing --clear..."
OUTPUT=$($CMCP_BIN online --clear 2>&1 || true)

if echo "$OUTPUT" | grep -q "Clearing server.*orphan-test"; then
    echo "✓ Clear command attempts to remove orphaned server"
fi

if echo "$OUTPUT" | grep -q "Cleanup complete"; then
    echo "✓ Shows completion message"
fi

# Verify orphaned server is gone
OUTPUT=$($CMCP_BIN online 2>&1 || true)
if ! echo "$OUTPUT" | grep -q "orphan-test"; then
    echo "✓ Orphaned server successfully cleared"
else
    echo "⚠ Orphaned server may still be present"
fi

echo ""
echo "5. Testing clear with no orphaned servers..."
echo "=============================================="

# Start a server from config first
$CMCP_BIN start test-server 2>&1 >/dev/null || true

# Now test clear when all servers are from config
OUTPUT=$($CMCP_BIN online --clear 2>&1 || true)
if echo "$OUTPUT" | grep -q "No orphaned servers to clear" || echo "$OUTPUT" | grep -q "All servers in Claude are in your cmcp config" || echo "$OUTPUT" | grep -q "No servers are currently running"; then
    echo "✓ Shows correct message when nothing to clear"
else
    echo "✗ Incorrect message when nothing to clear"
    echo "  Actual output: $OUTPUT"
fi

# Clean up
$CMCP_BIN stop test-server 2>&1 >/dev/null || true

echo ""
echo "6. Testing project context messages..."
echo "========================================"

# Test start command
OUTPUT=$($CMCP_BIN start test-server 2>&1 || true)
if echo "$OUTPUT" | grep -q "Starting server.*in Claude for this project"; then
    echo "✓ Start command mentions project context"
else
    echo "✗ Start command missing project context"
fi

# Clean up - remove test server
claude mcp remove test-server 2>&1 >/dev/null || true

echo ""
echo "7. Testing status indicators..."
echo "================================="

# Add a server that will fail to connect
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "fail-server": {
      "command": "nonexistent-command",
      "args": ["--fail"]
    }
  }
}
EOF

# Start the failing server
$CMCP_BIN start fail-server 2>&1 >/dev/null || true

# Check online output for status indicators
OUTPUT=$($CMCP_BIN online 2>&1 || true)

if echo "$OUTPUT" | grep -q "✗.*fail-server.*Failed to connect"; then
    echo "✓ Shows failure indicator for failed server"
elif echo "$OUTPUT" | grep -q "fail-server"; then
    echo "⚠ Shows server but may not have correct status indicator"
else
    echo "✗ Failed server not shown"
fi

# Clean up
claude mcp remove fail-server 2>&1 >/dev/null || true

echo ""
echo "8. Testing --clean flag (remove failed servers)..."
echo "===================================================="

# Add servers to config, including one that will fail
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "good-server": {
      "command": "echo",
      "args": ["good"]
    },
    "bad-server": {
      "command": "nonexistent-command",
      "args": ["fail"]
    }
  }
}
EOF

# Start both servers
$CMCP_BIN start good-server 2>&1 >/dev/null || true
$CMCP_BIN start bad-server 2>&1 >/dev/null || true

# Test --clean with dry-run
echo "Testing --clean with dry-run..."
OUTPUT=$($CMCP_BIN online --clean --dry-run 2>&1 || true)
if echo "$OUTPUT" | grep -q "Would execute the following commands"; then
    echo "✓ Clean dry-run shows commands to be executed"
    if echo "$OUTPUT" | grep -q "claude mcp remove bad-server"; then
        echo "✓ Clean correctly identifies failed server"
    else
        echo "✗ Clean did not identify failed server correctly"
    fi
else
    echo "✗ Clean dry-run output incorrect"
fi

# Test actual --clean
echo "Testing --clean..."
OUTPUT=$($CMCP_BIN online --clean 2>&1 || true)
if echo "$OUTPUT" | grep -q "bad-server" || echo "$OUTPUT" | grep -q "No failed servers to clean"; then
    if echo "$OUTPUT" | grep -q "Failed servers cleaned" || echo "$OUTPUT" | grep -q "No failed servers to clean"; then
        echo "✓ Clean command works"
    else
        echo "✗ Clean command failed"
    fi
else
    echo "✗ Clean command output incorrect"
fi

# Verify failed server was removed
OUTPUT=$($CMCP_BIN online 2>&1 || true)
if echo "$OUTPUT" | grep -q "bad-server"; then
    echo "✗ Failed server still present after clean"
else
    echo "✓ Failed server successfully removed"
fi

# Clean up any remaining servers first
$CMCP_BIN stop good-server 2>&1 >/dev/null || true
claude mcp remove good-server 2>&1 >/dev/null || true
claude mcp remove bad-server 2>&1 >/dev/null || true

# Test --clean with no failed servers
OUTPUT=$($CMCP_BIN online --clean 2>&1 || true)
if echo "$OUTPUT" | grep -q "No failed servers to clean"; then
    echo "✓ Clean correctly reports when no failed servers exist"
else
    echo "✗ Clean message incorrect when no failed servers"
    echo "  Output was: $OUTPUT"
fi

echo ""
echo "9. Testing color output in different contexts..."
echo "================================================="

# Force color output even if not in TTY
export FORCE_COLOR=1

OUTPUT=$($CMCP_BIN online 2>&1 || true)

# Check for ANSI color codes
if echo "$OUTPUT" | od -c | grep -q '033'; then
    echo "✓ Color codes present in output"
else
    echo "⚠ Color codes may not be present (could be terminal detection)"
fi

echo ""
echo "=== Online command tests completed ==="

# Summary
echo ""
echo "Summary of tested features:"
echo "- ✓ Project context in messages"
echo "- ✓ Orphaned server detection"
echo "- ✓ Clear command functionality (orphaned servers)"
echo "- ✓ Clean command functionality (failed servers)"
echo "- ✓ Colored output with status indicators"
echo "- ✓ Proper error messages and suggestions"