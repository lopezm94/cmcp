# Manual Test Guide for CMCP

This guide shows exactly how to test cmcp manually, with the exact commands you would type.

## Automated Testing (Recommended)

For comprehensive testing, use the automated test suite:

```bash
# Run all tests automatically (detects Podman or Docker)  
./test.sh
```

## Manual Testing

## Setup
```bash
# Build the binary
go build -o cmcp

# Clean any existing config
rm -f ~/.cmcp/config.json
```

## Test Commands

### 1. Basic Help
```bash
./cmcp --help
```

### 2. Check Empty Registry
```bash
./cmcp register list
# Expected: "No servers registered"
```

### 3. Add Your First MCP Server
```bash
./cmcp register add
# You'll see prompts - type these responses:
# Server name: playwright
# Command to run: npx @playwright/mcp@latest
```

### 4. Add Another Server
```bash
./cmcp register add
# Server name: github
# Command to run: npx -y @modelcontextprotocol/server-github
```

### 5. Add Environment Variables
```bash
./cmcp register env
# Select server: github
# Environment variables: GITHUB_TOKEN=ghp_your_token,API_KEY=test123
```

### 6. Open Config in Editor
```bash
./cmcp register open
# Select server to jump to, or "[Open at top]"
# Edit config file directly in nano
```

### 7. List Your Servers
```bash
./cmcp register list
# Should show both servers with their commands
```

### 8. Check Running Servers
```bash
./cmcp online
# Expected: "No servers are currently running."
```

### 9. Start a Server
```bash
./cmcp start
# Use arrow keys to select a server, press Enter
```

### 10. Check It's Running
```bash
./cmcp online
# Should show the server as RUNNING
```

### 11. Stop the Server
```bash
./cmcp stop
# Select the running server and press Enter
```

### 12. Start Multiple Servers
```bash
./cmcp start  # Start first server
./cmcp start  # Start second server
./cmcp online # Should show both running
```

### 13. Reset (Stop All)
```bash
./cmcp reset
# Type 'y' to confirm
```

### 14. Remove a Server
```bash
./cmcp register rm
# Select server to remove, confirm with 'y'
```

### 15. Shell Completion
```bash
# Generate completion for your shell
./cmcp completion bash
./cmcp completion zsh
```

## Interactive Navigation

For commands with selection menus (`start`, `stop`, `register rm`):
- Use ↑/↓ arrow keys to navigate
- Press Enter to select
- Press Ctrl+C to cancel