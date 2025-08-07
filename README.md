# cmcp - CLI MCP Manager

A command-line tool for managing Model Context Protocol (MCP) servers on your system.

## Features

- **Claude CLI Integration** - Registers/unregisters servers with Claude CLI for seamless use
- **Persistent Configuration** - Store MCP server configurations independently from Claude
- **Interactive server selection** - Select servers to start/stop using arrow keys
- **Manual config editing** - Open config file directly in nano with `cmcp config open`
- **Shell autocompletion** - Full command completion support for bash, zsh, fish, and PowerShell
- **Standard MCP format** - Compatible with Claude Desktop and other MCP tools
- **Advanced Troubleshooting** - Built-in diagnostics help identify and fix MCP connection issues
- **Debug Logging** - Automatic debug output capture for troubleshooting failures
- **Orphaned Server Cleanup** - Detect and clear servers from Claude that aren't in your config

## Requirements

- **Claude CLI** - Required for server registration (uses `claude mcp add/remove/list`)
- **Go 1.21+** - For building the tool
- **nano** (preferred) or any text editor

## Supported Operating Systems

- **macOS** (Intel and Apple Silicon)
  - Full support including Homebrew integration
  - Tested on macOS 12+ (Monterey and later)
  
- **Linux** (x86_64 and ARM64)
  - Ubuntu, Debian, Fedora, Arch, and other major distributions
  - Requires standard Unix tools (bash, grep, etc.)
  
- **Windows** - Not currently supported
  - The tool relies on Unix shell scripts and Claude CLI
  - Windows users can use WSL2 (Windows Subsystem for Linux)

## Installation

### Quick Install (Recommended)

Install cmcp with a single command:

```bash
# Install
curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/scripts/web-install.sh | bash

# Uninstall
curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/scripts/web-uninstall.sh | bash
```

### Manual Installation

```bash
# Clone and install cmcp
git clone https://github.com/lopezm94/cmcp.git
cd cmcp

# Install or upgrade (automatically detects existing installation)
./scripts/install.sh

# To uninstall completely
./scripts/uninstall.sh
```

### Installation Scripts

- **`./scripts/install.sh`** - Install or upgrade cmcp
  - Automatically detects if this is a fresh install or upgrade
  - Builds and installs the binary to `/usr/local/bin`
  - Sets up shell completions for your shell
  - **Always preserves existing server configurations**
  - For upgrades: shows version info and offers to stop running servers
  - For fresh installs: creates the configuration directory `~/.cmcp`

- **`./scripts/uninstall.sh`** - Remove cmcp from your system
  - Removes the cmcp binary
  - Removes shell completions
  - Optionally removes configuration (asks for confirmation)
  - If you keep the configuration, you can reinstall later and retain all server settings

## Usage

This tool stores MCP server configurations and registers them with Claude CLI when starting.

### How it works

1. **cmcp** stores server configurations in its own config file
2. **cmcp start** uses `claude mcp add` to register servers with Claude for the current project
3. **cmcp stop** uses `claude mcp remove` to unregister servers from Claude for the current project
4. **cmcp online** uses `claude mcp list` to show Claude's registered servers with status indicators

### Configure and Start MCP Servers

```bash
# Open config file to add servers manually
cmcp config open
# Add servers in the standard MCP format - example:
# {
#   "mcpServers": {
#     "github": {
#       "command": "npx",
#       "args": ["-y", "@modelcontextprotocol/server-github"],
#       "env": {
#         "GITHUB_TOKEN": "ghp_your_token_here"
#       }
#     }
#   }
# }

# Start the server (registers with Claude)
cmcp start
# Select: github

# Verify it's running (shows Claude's registered servers)
cmcp online
```

## Manage Configuration

```bash
# Edit configuration file directly
cmcp config open
# Opens ~/.cmcp/config.json in nano

# List configured servers
cmcp config list

# Remove a server (interactive selection)
cmcp config rm
```

### Example Configuration

Edit your config file to add servers like these:

```json
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },
    "filesystem": {
      "command": "npx",
      "args": ["@claude/mcp-server-filesystem"],
      "env": {
        "ALLOWED_DIRECTORIES": "/home/user/documents,/home/user/projects"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    },
    "sqlite": {
      "command": "npx",
      "args": ["@claude/mcp-server-sqlite", "--db", "/path/to/database.db"]
    }
  }
}
```

### Manage Servers

```bash
# Start a server (interactive selection, registers with Claude for this project)
cmcp start

# Start with verbose output to see debug output directly
cmcp start -v

# Stop a running server (interactive selection, unregisters from Claude)
cmcp stop

# Show all servers registered in Claude for this project with colored status indicators
cmcp online

# Clear orphaned servers (not in your config) from Claude
cmcp online --clear

# Remove failed servers from Claude
cmcp online --clean

# Stop all running servers (unregisters all from Claude for this project)
cmcp reset
```

### Troubleshooting MCP Connections

cmcp includes advanced diagnostics and automatic debug logging:

#### Automatic Debug Logging
Debug output is always captured when commands fail:
- In **normal mode**: Debug logs are saved to `/tmp/cmcp-debug/` and the path is shown in error messages
- In **verbose mode** (`-v`): Debug output from Claude CLI is shown directly in the terminal

```bash
# Normal mode - debug log saved to file on error
cmcp start github
# If it fails: ✗ Failed to start server 'github' (debug log: /tmp/cmcp-debug/cmcp-start-github-20250807-150625.log)

# Verbose mode - see debug output directly
cmcp start -v github
```

The diagnostics provide intelligent analysis for common issues:

- **Docker servers**: Checks if Docker daemon is running, image availability, environment variables
- **Node.js servers**: Verifies node/npx installation, script existence, dependencies
- **Python servers**: Checks Python installation, script availability, requirements
- **General issues**: Permission errors, port conflicts, missing environment variables

Example output when troubleshooting:
```
Starting server 'github'...
✗ Failed to start server 'github':
server 'github' failed to start: failed to connect

Health Check: github: docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server - ✗ Failed to connect

Error Output:
docker: Cannot connect to the Docker daemon at unix:///var/run/docker.sock.

Possible solutions:
  1. Docker daemon is not running. Please start Docker Desktop or the Docker service.
  2. Check that required environment variables are set in your shell
```

#### Managing Orphaned and Failed Servers

**Orphaned servers** (added directly with `claude mcp add` or from tests, not in your cmcp config):

```bash
# View all servers with status indicators
cmcp online
# Output:
# MCP servers running in Claude for this project:
# Project: /path/to/your/project
# 
# ✓ github: npx -y @modelcontextprotocol/server-github - Connected
# ✗ test-fail: nonexistent-command --fail - Failed to connect
# 
# ⚠ Found 1 server(s) in Claude that are not in your cmcp config:
#   - test-fail
# 
# To clear these servers from Claude, run:
#   $ cmcp online --clear

# Clear orphaned servers
cmcp online --clear
# or with dry-run to preview
cmcp online --clear --dry-run
```

**Failed servers** (servers in your config that are failing to connect):

```bash
# Remove all failed servers from Claude
cmcp online --clean

# Preview what would be removed
cmcp online --clean --dry-run
```

**Security Note**: All sensitive information (API keys, tokens, passwords) are automatically masked in verbose and debug output to prevent accidental exposure.


### Shell Completion

Shell completion is automatically installed by `./install.sh` and adds automcompletion to zsh.

For manual setup:
```bash
# Zsh
cmcp completion zsh > "${fpath[1]}/_cmcp"
```

## Configuration

Configuration is stored in `~/.cmcp/config.json` using the **standard MCP format**:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    }
  }
}
```

**Benefits of the standard format:**
- ✅ Compatible with Claude Desktop configuration
- ✅ Can copy servers between cmcp and Claude
- ✅ Edit config file manually for advanced setups
- ✅ Industry standard MCP configuration

## Testing

Run comprehensive tests in an isolated container:

```bash
# Run all tests (automatically detects Podman or Docker)
./test.sh

# Run specific tests
./test.sh online              # Test online command features
./test.sh logging             # Test automatic logging functionality
./test.sh unit comprehensive  # Run multiple specific tests

# Available test names:
# - unit: Go unit tests
# - comprehensive: Comprehensive functionality tests
# - install: Install/uninstall script tests
# - logging: Automatic logging tests
# - web: Web install/uninstall tests
# - online: Online command tests
```

The test suite covers:
- All command functionality
- Interactive prompts
- Server lifecycle management
- Configuration persistence
- Error handling
- Debug logging
- Orphaned server cleanup

Tests run in a clean environment and don't affect your system.

## License

MIT
