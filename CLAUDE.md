# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

cmcp (CLI MCP Manager) is a command-line tool written in Go for managing Model Context Protocol (MCP) servers. It acts as a wrapper around Claude CLI to provide persistent configuration and easier management of MCP servers.

## Key Commands

### Build and Development
```bash
# Build the binary
go build -o cmcp

# Run unit tests
go test ./...

# Run comprehensive tests in container (Docker or Podman)
./test.sh

# Run unit tests locally (fast)
./test-unit.sh
```

### Installation
```bash
# Install/upgrade (preserves configs)
./scripts/install.sh

# Uninstall
./scripts/uninstall.sh
```

## Architecture

### Core Components

1. **cmd/** - Cobra command definitions
   - `root.go` - Main command structure and completion setup
   - `start.go` - Start servers with `claude mcp add`/`claude mcp add-json`
   - `stop.go` - Stop servers with `claude mcp remove`
   - `online.go` - List running servers with `claude mcp list`
   - `reset.go` - Stop all servers
   - `config.go` - Manage persistent configuration

2. **internal/mcp/** - MCP server management
   - `claude_cmd_builder.go` - Builds and executes Claude CLI commands
   - `security.go` - Masks sensitive data in output
   - `diagnostics.go` - Intelligent error diagnostics for Docker/Node/Python servers

3. **internal/config/** - Configuration management
   - `config.go` - Handles ~/.cmcp/config.json using standard MCP format

### Key Design Patterns

- **Claude CLI Integration**: All server operations delegate to `claude mcp` commands
- **Persistent Config**: Stores server definitions in ~/.cmcp/config.json
- **Security**: Automatically masks API keys and tokens in verbose output
- **Smart Diagnostics**: Detects common issues (Docker not running, missing deps, etc.)

### Configuration Format

Uses standard MCP server configuration format:
```json
{
  "mcpServers": {
    "server-name": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_..."
      }
    }
  }
}
```

## Testing Strategy

- **Unit Tests**: Test individual components (claude_cmd_builder_test.go, security_test.go, diagnostics_test.go)
- **Integration Tests**: Run in containers to test full command flow
- **Install Tests**: Verify installation/uninstall scripts work correctly

## Important Notes

- Always use `claude mcp` commands for server operations, never implement MCP protocol directly
- Preserve existing configurations during upgrades
- Test commands with dry-run flags before making changes
- Security-sensitive information should always be masked in output