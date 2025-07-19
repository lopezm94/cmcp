# cmcp - CLI MCP Manager

A command-line tool for managing Model Context Protocol (MCP) servers on your system.

## Features

- **Claude CLI Integration** - Registers/unregisters servers with Claude CLI for seamless use
- **Persistent Configuration** - Store MCP server configurations independently from Claude
- **Interactive server selection** - Select servers to start/stop using arrow keys
- **Manual config editing** - Open config file directly in nano with `cmcp config open`
- **Shell autocompletion** - Full command completion support for bash, zsh, fish, and PowerShell
- **Standard MCP format** - Compatible with Claude Desktop and other MCP tools

## Requirements

- **Claude CLI** - Required for server registration (uses `claude mcp add/remove/list`)
- **Go 1.21+** - For building the tool
- **nano** (preferred) or any text editor

## Installation

```bash
# Clone and install cmcp
git clone <repository-url>
cd global-cli

# Install automatically (includes shell completion)
./install.sh

# To uninstall later
./uninstall.sh
```

## Usage

This tool stores MCP server configurations and registers them with Claude CLI when starting.

### How it works

1. **cmcp** stores server configurations in its own config file
2. **cmcp start** uses `claude mcp add` to register servers with Claude
3. **cmcp stop** uses `claude mcp remove` to unregister servers from Claude  
4. **cmcp online** uses `claude mcp list` to show Claude's registered servers

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
    "github": {
      "command": "npx", 
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["@claude/mcp-server-filesystem"]
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
# Start a server (interactive selection, registers with Claude)
cmcp start

# Stop a running server (interactive selection, unregisters from Claude)
cmcp stop

# Show all servers registered in Claude
cmcp online

# Stop all running servers (unregisters all from Claude)
cmcp reset
```


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

See `example-config.json` for more examples with real MCP servers.

## Testing

Run comprehensive tests in an isolated container:

```bash
# Run all tests (automatically detects Podman or Docker)
./test.sh
```

The test suite covers:
- All command functionality
- Interactive prompts
- Server lifecycle management
- Configuration persistence
- Error handling

Tests run in a clean environment and don't affect your system.

## License

MIT
