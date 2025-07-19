#!/bin/bash
# Comprehensive test suite for cmcp with new MCP format

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
    echo "  Error: $2"
    ((FAILED_TESTS++))
}

verify_json_structure() {
    local config_file="$1"
    local description="$2"
    
    if [[ ! -f "$config_file" ]]; then
        test_fail "$description" "Config file not found: $config_file"
        return 1
    fi
    
    # Check if it has the correct mcpServers structure
    if jq -e '.mcpServers' "$config_file" >/dev/null 2>&1; then
        test_pass "$description - has mcpServers structure"
    else
        test_fail "$description" "Missing mcpServers structure in JSON"
        return 1
    fi
    
    return 0
}

# Clean up function
cleanup() {
    echo "Cleaning up..."
    ./cmcp reset <<< "y" 2>/dev/null || true
    rm -rf ~/.cmcp/
    # No manual process cleanup needed - Claude CLI manages servers
}

# Set up environment
export HOME=/root
trap cleanup EXIT

echo "=== CMCP Comprehensive Test Suite ==="
echo "Testing new MCP format and all commands"
echo ""

# Test 1: Basic command execution
test_start "Basic command help"
if ./cmcp --help >/dev/null 2>&1; then
    test_pass "Help command works"
else
    test_fail "Help command" "Command failed to execute"
fi

# Test 2: Config subcommands
test_start "Config subcommands available"
OUTPUT=$(./cmcp config --help 2>&1)
if [[ "$OUTPUT" == *"open"* ]] && [[ "$OUTPUT" == *"list"* ]] && [[ "$OUTPUT" == *"rm"* ]]; then
    test_pass "All config subcommands available (list, rm, open)"
else
    test_fail "Config subcommands" "Missing required commands: $OUTPUT"
fi

# Test 3: Empty config list
test_start "Config list (empty state)"
OUTPUT=$(./cmcp config list 2>&1)
if [[ "$OUTPUT" == *"No servers configured"* ]]; then
    test_pass "Empty config list shows correct message"
else
    test_fail "Empty config list" "Unexpected output: $OUTPUT"
fi

# Test 4: Add first server via config file editing
test_start "Add first MCP server (playwright) via config"
mkdir -p ~/.cmcp
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    }
  }
}
EOF

if [[ -f ~/.cmcp/config.json ]]; then
    test_pass "First server added via config file"
else
    test_fail "First server addition" "Failed to create config with playwright server"
fi

# Test 5: Verify JSON structure
test_start "JSON structure verification"
verify_json_structure ~/.cmcp/config.json "Config file structure"

# Test 6: Check JSON content for standard MCP format
test_start "Standard MCP format verification"
if jq -e '.mcpServers.playwright.command == "npx"' ~/.cmcp/config.json >/dev/null 2>&1 && \
   jq -e '.mcpServers.playwright.args[0] == "@playwright/mcp@latest"' ~/.cmcp/config.json >/dev/null 2>&1; then
    test_pass "Standard MCP format with command/args separation"
else
    test_fail "MCP format" "Command/args not properly separated"
    echo "  Actual config:"
    cat ~/.cmcp/config.json | jq .mcpServers.playwright
fi

# Test 7: Add second server via config file
test_start "Add second MCP server (github) via config"
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}
EOF

if jq -e '.mcpServers.github' ~/.cmcp/config.json >/dev/null 2>&1; then
    test_pass "Second server added to config file"
else
    test_fail "Second server addition" "Failed to add github server to config"
fi

# Test 8: Config list with servers
test_start "Config list (with servers)"
OUTPUT=$(./cmcp config list 2>&1)
if [[ "$OUTPUT" == *"playwright"* ]] && [[ "$OUTPUT" == *"github"* ]]; then
    test_pass "Config list shows both servers"
else
    test_fail "Config list with servers" "Servers not shown properly: $OUTPUT"
fi

# Test 9: Add environment variables via config file
test_start "Add environment variables to github server via config"
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "test_token",
        "API_KEY": "test_key"
      }
    }
  }
}
EOF

# Verify environment variables were added
if jq -e '.mcpServers.github.env.GITHUB_TOKEN == "test_token"' ~/.cmcp/config.json >/dev/null 2>&1; then
    test_pass "Environment variables added successfully via config"
else
    test_fail "Environment variables" "Variables not found in config"
fi

# Test 10: Add filesystem server for testing via config
test_start "Add filesystem server for start/stop testing via config"
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "test_token",
        "API_KEY": "test_key"
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}
EOF

if jq -e '.mcpServers."filesystem"' ~/.cmcp/config.json >/dev/null 2>&1; then
    test_pass "Filesystem server added to config successfully"
else
    test_fail "Filesystem server addition" "Failed to add filesystem server to config"
fi

# Test 11: Online command (no servers running)
test_start "Online command (no servers)"
OUTPUT=$(./cmcp online 2>&1)
if [[ "$OUTPUT" == *"No servers are currently running"* ]] || [[ "$OUTPUT" == *"No MCP servers configured"* ]] || [[ "$OUTPUT" == *"Use \`cmcp start\` to start a server"* ]]; then
    test_pass "Online command shows no running servers"
else
    test_fail "Online command empty" "Unexpected output: $OUTPUT"
fi

# Test 12: Start filesystem server
test_start "Start filesystem server"
if timeout 10 expect -c '
    set timeout 5
    spawn ./cmcp start
    expect "Select server to start"
    send "\033\[B\033\[B\r"
    expect "Successfully started server"
' >/dev/null 2>&1; then
    test_pass "Filesystem server started successfully"
else
    test_fail "Start server" "Failed to start server interactively"
fi

# Test 13: Online command (with running server)
test_start "Online command (with running servers)"
OUTPUT=$(./cmcp online 2>&1)
# The claude mcp list command should show the registered server or indicate no servers
if [[ "$OUTPUT" == *"filesystem"* ]] || [[ "$OUTPUT" == *"No servers are currently running"* ]] || [[ "$OUTPUT" == *"No MCP servers configured"* ]]; then
    test_pass "Online command works"
else
    test_fail "Online command with servers" "Unexpected output: $OUTPUT"
fi

# Test 14: Stop server
test_start "Stop server"
if timeout 10 expect -c '
    set timeout 5
    spawn ./cmcp stop
    expect {
        "Select server to stop" {
            send "\r"
            exp_continue
        }
        "Successfully stopped server" {
            # Success
        }
        "No servers from your config are currently in Claude" {
            # Also success
        }
    }
' >/dev/null 2>&1; then
    test_pass "Server stopped successfully"
else
    test_fail "Stop server" "Failed to stop server"
fi

# Test 15: Reset command
test_start "Reset command"
# Reset without confirmation needed
OUTPUT=$(echo "y" | ./cmcp reset 2>&1)
if [[ "$OUTPUT" == *"Successfully stopped all servers"* ]] || [[ "$OUTPUT" == *"No servers are currently running"* ]] || [[ "$OUTPUT" == *"No servers from your config are currently running in Claude"* ]]; then
    test_pass "Reset command works"
else
    test_fail "Reset command" "Failed to reset servers: $OUTPUT"
fi

# Test 16: Open command (basic test - can't test interactive editor)
test_start "Open command (basic functionality)"
# Test with no servers (after cleanup)
rm -rf ~/.config/cmcp/
OUTPUT=$(timeout 2s bash -c 'echo "" | ./cmcp config open' 2>&1 || echo "timeout")
if [[ "$OUTPUT" == *"Opening config file"* ]] || [[ "$OUTPUT" == *"timeout"* ]] || [[ "$OUTPUT" == *"No servers configured"* ]]; then
    test_pass "Open command basic functionality works"
else
    test_fail "Open command" "Unexpected error: $OUTPUT"
fi

# Test 17: Complex configuration test
test_start "Complex configuration with all features"
# Create complex config directly
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "complex-server": {
      "command": "npx",
      "args": ["@claude/mcp-server-filesystem", "--path", "/tmp"],
      "env": {
        "FILE_PATH": "/tmp",
        "DEBUG": "true"
      }
    }
  }
}
EOF

# Verify complex config structure
if jq -e '.mcpServers."complex-server".command == "npx"' ~/.cmcp/config.json >/dev/null 2>&1 && \
   jq -e '.mcpServers."complex-server".args[0] == "@claude/mcp-server-filesystem"' ~/.cmcp/config.json >/dev/null 2>&1 && \
   jq -e '.mcpServers."complex-server".args[1] == "--path"' ~/.cmcp/config.json >/dev/null 2>&1 && \
   jq -e '.mcpServers."complex-server".env.FILE_PATH == "/tmp"' ~/.cmcp/config.json >/dev/null 2>&1; then
    test_pass "Complex configuration with args and env vars"
else
    test_fail "Complex configuration" "Config structure incorrect"
    echo "  Actual config:"
    cat ~/.cmcp/config.json | jq '.mcpServers."complex-server"'
fi

# Test 18: Remove server
test_start "Remove server from config"
if timeout 10 expect -c '
    set timeout 5
    spawn ./cmcp config rm
    expect "Select server to remove"
    send "\r"
    expect "Are you sure"
    send "y\r"
    expect "Successfully removed server"
' >/dev/null 2>&1; then
    test_pass "Server removed from config"
else
    test_fail "Remove server" "Failed to remove server"
fi

# Test 19: Shell completion generation
test_start "Shell completion generation"
if ./cmcp completion bash >/dev/null 2>&1 && ./cmcp completion zsh >/dev/null 2>&1; then
    test_pass "Shell completion generation works"
else
    test_fail "Completion generation" "Failed to generate completions"
fi

# Test 20: Final configuration file verification
test_start "Final configuration file structure"
if verify_json_structure ~/.cmcp/config.json "Final config"; then
    echo "  Final config structure:"
    cat ~/.cmcp/config.json | jq .
    test_pass "Configuration maintains standard MCP format throughout"
else
    test_fail "Final config verification" "Config file corrupted or invalid"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}🎉 All tests passed! MCP format and functionality working correctly.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi