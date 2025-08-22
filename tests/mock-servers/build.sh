#!/bin/bash
# Build mock MCP servers for testing

echo "Building mock MCP servers..."

# Go to the directory containing the mock servers
cd "$(dirname "$0")"

# Build the basic mock server as a standalone binary
CGO_ENABLED=0 go build -o mock-mcp-basic mock-mcp-basic.go
if [ $? -eq 0 ]; then
    echo "✓ Built mock-mcp-basic"
else
    echo "✗ Failed to build mock-mcp-basic"
    exit 1
fi

# Build the failing mock server as a standalone binary
CGO_ENABLED=0 go build -o mock-mcp-failing mock-mcp-failing.go
if [ $? -eq 0 ]; then
    echo "✓ Built mock-mcp-failing"
else
    echo "✗ Failed to build mock-mcp-failing"
    exit 1
fi

echo "Mock servers built successfully!"