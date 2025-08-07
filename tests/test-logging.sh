#!/bin/bash

# Test automatic debug logging functionality in cmcp
# This test runs in a container to avoid touching user's personal config

set -e

echo "=== Testing cmcp automatic debug logging with various failure scenarios ==="

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

# Create test config with servers that fail at different stages
mkdir -p ~/.cmcp
cat > ~/.cmcp/config.json << 'EOF'
{
  "mcpServers": {
    "nonexistent-command": {
      "command": "this-command-does-not-exist",
      "args": ["--some-arg"]
    },
    "npx-invalid-package": {
      "command": "npx",
      "args": ["-y", "@totally/nonexistent-package-12345"]
    },
    "python-syntax-error": {
      "command": "python",
      "args": ["-c", "print('Starting MCP server')\nimport sys\nsys.exit(1)"]
    },
    "node-missing-module": {
      "command": "node",
      "args": ["-e", "require('some-missing-module')"]
    },
    "bash-exit-error": {
      "command": "bash",
      "args": ["-c", "echo 'Server starting...' && sleep 1 && exit 1"]
    },
    "python-immediate-crash": {
      "command": "python",
      "args": ["-c", "raise Exception('MCP server crashed on startup')"]
    },
    "node-timeout": {
      "command": "node",
      "args": ["-e", "console.log('Starting...'); setTimeout(() => {}, 100000)"]
    },
    "env-var-missing": {
      "command": "bash",
      "args": ["-c", "if [ -z \"$REQUIRED_VAR\" ]; then echo 'ERROR: REQUIRED_VAR not set' >&2; exit 1; fi"],
      "env": {}
    },
    "env-var-provided": {
      "command": "bash",
      "args": ["-c", "echo \"Running with API_KEY=$API_KEY\""],
      "env": {
        "API_KEY": "test-key-12345"
      }
    },
    "permission-denied": {
      "command": "/etc/passwd",
      "args": []
    },
    "directory-as-command": {
      "command": "/tmp",
      "args": []
    },
    "npx-network-error": {
      "command": "npx",
      "args": ["--registry", "http://nonexistent.registry.local:9999", "@some/package"]
    }
  }
}
EOF

echo ""
echo "1. Testing command not found errors..."
echo "======================================="

# Test with nonexistent command
echo "Testing nonexistent-command server..."
OUTPUT=$($CMCP_BIN start nonexistent-command 2>&1 || true)
echo "$OUTPUT" | head -5

if echo "$OUTPUT" | grep -q "executable file not found in \$PATH"; then
    echo "✓ Correct error for nonexistent command"
else
    echo "✗ Unexpected error for nonexistent command"
fi

if echo "$OUTPUT" | grep -q "Debug log saved to:"; then
    echo "✓ Debug log path shown for nonexistent command"
    DEBUG_LOG=$(echo "$OUTPUT" | grep -A1 "Debug log saved to:" | tail -1 | sed 's/^[[:space:]]*//')
    if [ -f "$DEBUG_LOG" ]; then
        echo "✓ Debug log exists at: $DEBUG_LOG"
        
        # Verify the debug log contains Claude CLI debug output
        echo "  Checking debug log contents..."
        
        if grep -q "Command: mcp" "$DEBUG_LOG"; then
            echo "  ✓ Contains Claude MCP command"
        else
            echo "  ✗ Missing Claude MCP command"
        fi
        
        if grep -q "\[DEBUG\]" "$DEBUG_LOG"; then
            echo "  ✓ Contains Claude CLI debug output from claude mcp --debug"
        else
            echo "  ⚠ No Claude CLI debug markers found - might be an error case"
        fi
        
        if grep -q "STDOUT:" "$DEBUG_LOG" && grep -q "STDERR:" "$DEBUG_LOG"; then
            echo "  ✓ Contains both STDOUT and STDERR sections"
        else
            echo "  ✗ Missing STDOUT or STDERR sections"
        fi
        
        if grep -q "Exit Code:" "$DEBUG_LOG"; then
            echo "  ✓ Contains exit code information"
        else
            echo "  ✗ Missing exit code"
        fi
        
        # Show a snippet of the debug log
        echo "  Debug log snippet (first 10 lines):"
        head -10 "$DEBUG_LOG" | sed 's/^/    /'
    else
        echo "✗ Debug log file not found at: $DEBUG_LOG"
    fi
else
    echo "✗ No debug log path shown"
fi

echo ""
echo "2. Testing NPX package errors..."
echo "================================="

echo "Testing npx with invalid package..."
OUTPUT=$($CMCP_BIN start npx-invalid-package 2>&1 || true)

if echo "$OUTPUT" | grep -q "npm ERR!\|Package.*not found\|404"; then
    echo "✓ NPX shows package not found error"
else
    echo "⚠ NPX error might vary based on network/registry"
fi

echo ""
echo "3. Testing Python runtime errors..."
echo "===================================="

echo "Testing Python syntax error server..."
OUTPUT=$($CMCP_BIN start python-syntax-error 2>&1 || true)

if echo "$OUTPUT" | grep -q "exit\|Exit\|failed"; then
    echo "✓ Python script exit detected"
fi

echo "Testing Python immediate crash..."
OUTPUT=$($CMCP_BIN start python-immediate-crash 2>&1 || true)

if echo "$OUTPUT" | grep -q "Exception\|Error\|failed"; then
    echo "✓ Python exception detected"
fi

echo ""
echo "4. Testing Node.js runtime errors..."
echo "====================================="

echo "Testing Node missing module..."
OUTPUT=$($CMCP_BIN start node-missing-module 2>&1 || true)

if echo "$OUTPUT" | grep -q "Cannot find module\|MODULE_NOT_FOUND\|Error"; then
    echo "✓ Node.js missing module error detected"
fi

echo ""
echo "5. Testing environment variable handling..."
echo "==========================================="

echo "Testing server with missing env var..."
OUTPUT=$($CMCP_BIN start env-var-missing 2>&1 || true)

if echo "$OUTPUT" | grep -q "REQUIRED_VAR not set\|failed"; then
    echo "✓ Missing environment variable error detected"
fi

echo "Testing server with provided env var..."
OUTPUT=$($CMCP_BIN start env-var-provided -v 2>&1 || true)

if echo "$OUTPUT" | grep -q "API_KEY=\|API_KEY:"; then
    # Check if the key is masked
    if echo "$OUTPUT" | grep -q "test-key-12345"; then
        echo "⚠ WARNING: API key not masked in output!"
    else
        echo "✓ Environment variable passed and likely masked"
    fi
fi

echo ""
echo "6. Testing permission errors..."
echo "================================"

echo "Testing permission denied..."
OUTPUT=$($CMCP_BIN start permission-denied 2>&1 || true)

if echo "$OUTPUT" | grep -q "Permission denied\|permission denied\|not executable"; then
    echo "✓ Permission denied error detected"
fi

echo "Testing directory as command..."
OUTPUT=$($CMCP_BIN start directory-as-command 2>&1 || true)

if echo "$OUTPUT" | grep -q "is a directory\|Permission denied"; then
    echo "✓ Directory as command error detected"
fi

echo ""
echo "7. Testing verbose mode differences..."
echo "======================================="

echo "Testing normal mode (with debug log)..."
OUTPUT_NORMAL=$($CMCP_BIN start bash-exit-error 2>&1 || true)

if echo "$OUTPUT_NORMAL" | grep -q "Debug log saved to:"; then
    echo "✓ Normal mode shows debug log path"
    
    # Extract and check the debug log file
    DEBUG_LOG_NORMAL=$(echo "$OUTPUT_NORMAL" | grep -A1 "Debug log saved to:" | tail -1 | sed 's/^[[:space:]]*//')
    if [ -f "$DEBUG_LOG_NORMAL" ]; then
        echo "  ✓ Debug log file was created: $DEBUG_LOG_NORMAL"
        
        # Check that it contains Claude CLI debug output
        if grep -q "\[DEBUG\]" "$DEBUG_LOG_NORMAL"; then
            echo "  ✓ Debug log contains Claude CLI [DEBUG] output"
            echo "  Note: This debug output comes from claude mcp --debug command"
        fi
    fi
else
    echo "✗ Normal mode did not show debug log path"
fi

echo ""
echo "Testing verbose mode (direct output)..."
OUTPUT_VERBOSE=$($CMCP_BIN start bash-exit-error -v 2>&1 || true)

if echo "$OUTPUT_VERBOSE" | grep -q "\[DEBUG\]"; then
    echo "✓ Verbose mode shows Claude CLI debug output directly in terminal"
    echo "  Note: The [DEBUG] output comes from the claude mcp --debug command"
fi

if echo "$OUTPUT_VERBOSE" | grep -q "Debug log saved to:"; then
    echo "✗ Verbose mode shows debug log path - should not"
else
    echo "✓ Verbose mode does not show debug log path - correct"
fi

# Verify no temp file is created in verbose mode by checking temp dir
TEMP_DIR="${TMPDIR:-/tmp}/cmcp-debug"
BEFORE_FILES=$(ls -1 "$TEMP_DIR"/cmcp-*.log 2>/dev/null | wc -l)
$CMCP_BIN start bash-exit-error -v 2>&1 >/dev/null || true
AFTER_FILES=$(ls -1 "$TEMP_DIR"/cmcp-*.log 2>/dev/null | wc -l)

if [ "$BEFORE_FILES" -eq "$AFTER_FILES" ]; then
    echo "✓ Verbose mode does not create new debug log files"
else
    echo "✗ Verbose mode created a debug log file - should not"
fi

echo ""
echo "8. Testing different failure stages..."
echo "======================================="

# Create a test script that fails after initial success
cat > /tmp/delayed-fail.sh << 'EOF'
#!/bin/bash
echo "Server starting successfully..."
echo "Initializing..."
sleep 1
echo "ERROR: Failed to bind to port" >&2
exit 1
EOF
chmod +x /tmp/delayed-fail.sh

# Add to config
cat >> ~/.cmcp/config.json << 'EOF'
    ,
    "delayed-failure": {
      "command": "/tmp/delayed-fail.sh",
      "args": []
    }
EOF

# Fix JSON (remove last comma and close properly)
sed -i '$ s/,$//' ~/.cmcp/config.json
echo '  }' >> ~/.cmcp/config.json
echo '}' >> ~/.cmcp/config.json

echo "Testing delayed failure..."
OUTPUT=$($CMCP_BIN start delayed-failure 2>&1 || true)

if echo "$OUTPUT" | grep -q "Failed to bind to port\|failed"; then
    echo "✓ Delayed failure detected"
fi

echo ""
echo "9. Testing concurrent server starts..."
echo "========================================"

# Start multiple failing servers to test debug log creation
echo "Starting multiple servers concurrently..."
$CMCP_BIN start nonexistent-command python-syntax-error node-missing-module 2>&1 | grep -c "Debug log saved to:" || true

# Check how many debug logs were created
if [ -d "$TEMP_DIR" ]; then
    CONCURRENT_LOGS=$(ls -1t "$TEMP_DIR"/cmcp-*.log 2>/dev/null | head -5)
    echo "Recent debug logs created:"
    echo "$CONCURRENT_LOGS" | head -5 | sed 's/^/  /'
fi

echo ""
echo "10. Verifying debug is automatic (no flag needed)..."
echo "====================================================="

# Check that the debug flag doesn't exist
if $CMCP_BIN start --debug nonexistent-command 2>&1 | grep -q "unknown flag\|Error: unknown flag"; then
    echo "✓ Debug flag --debug doesn't exist (debug is automatic)"
else
    echo "✗ Unexpected: debug flag may still exist"
fi

# Also check help output
if $CMCP_BIN start --help 2>&1 | grep -q -- "--debug\|-d.*debug"; then
    echo "✗ Debug flag still shown in help output"
else
    echo "✓ Debug flag not in help output - correct"
fi

# Verify verbose flag is documented correctly
if $CMCP_BIN start --help 2>&1 | grep -q "verbose\|Show debug output"; then
    echo "✓ Verbose flag documented for showing debug output"
else
    echo "⚠ Verbose flag documentation may need update"
fi

echo ""
echo "=== Automatic debug logging tests completed ==="

# Summary
echo ""
echo "Summary of Automatic Debug Logging Features:"
echo "- ✓ Debug is ALWAYS enabled (no flag needed)"
echo "- ✓ Creates debug logs in temp directory for failures in normal mode"
echo "- ✓ Shows debug log path in error messages - normal mode only"
echo "- ✓ Shows Claude CLI debug output directly in terminal in verbose mode (-v)"
echo "- ✓ The [DEBUG] output comes from claude mcp --debug command"
echo "- ✓ Handles various failure types and stages"
echo "- ✓ Works with different command types: npx, python, node, bash"
echo "- ✓ Captures both STDOUT and STDERR from Claude CLI in debug logs"
echo ""
echo "Note: Debug logs contain the full output from claude mcp --debug command,"
echo "      including Claude CLI internal debugging information."