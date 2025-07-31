#!/bin/bash
# Web installer for cmcp
# Usage: curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/web-install.sh | bash
#
# Note: This installer will check for GitHub releases first, then fall back to
# building from source if no releases are available.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REPO_OWNER="lopezm94"
REPO_NAME="cmcp"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.cmcp"
TEMP_DIR="/tmp/cmcp-install-$$"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Map OS names
case "$OS" in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}🚀 Installing cmcp...${NC}"
echo "Detected: $OS/$ARCH"

# Create temp directory
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# Get latest release
echo "Fetching latest release..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${YELLOW}No releases found. Installing from source...${NC}"
    
    # Check for required tools
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}Go is required to build from source. Please install Go 1.21 or later.${NC}"
        exit 1
    fi
    
    # Clone and build from source
    echo "Cloning repository..."
    git clone "https://github.com/$REPO_OWNER/$REPO_NAME.git" .
    
    echo "Building cmcp..."
    go build -o cmcp
    
    BINARY_PATH="./cmcp"
else
    # Download pre-built binary
    BINARY_NAME="cmcp-${OS}-${ARCH}"
    DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$LATEST_RELEASE/$BINARY_NAME"
    
    echo "Downloading cmcp $LATEST_RELEASE..."
    if ! curl -L -o cmcp "$DOWNLOAD_URL"; then
        echo -e "${YELLOW}Pre-built binary not available. Installing from source...${NC}"
        
        # Check for required tools
        if ! command -v go >/dev/null 2>&1; then
            echo -e "${RED}Go is required to build from source. Please install Go 1.21 or later.${NC}"
            exit 1
        fi
        
        # Download source and build
        echo "Downloading source..."
        curl -L -o source.tar.gz "https://github.com/$REPO_OWNER/$REPO_NAME/archive/$LATEST_RELEASE.tar.gz"
        tar -xzf source.tar.gz --strip-components=1
        
        echo "Building cmcp..."
        go build -o cmcp
    fi
    
    BINARY_PATH="./cmcp"
fi

# Make binary executable
chmod +x "$BINARY_PATH"

# Check if upgrade or fresh install
IS_UPGRADE=false
if command -v cmcp >/dev/null 2>&1; then
    IS_UPGRADE=true
    CURRENT_VERSION=$(cmcp --version 2>/dev/null || echo "unknown")
    echo -e "${BLUE}Upgrading cmcp ($CURRENT_VERSION → $LATEST_RELEASE)...${NC}"
else
    echo -e "${BLUE}Installing cmcp for the first time...${NC}"
fi

# Install binary
echo -e "${YELLOW}Installing cmcp to $INSTALL_DIR...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    cp "$BINARY_PATH" "$INSTALL_DIR/cmcp"
else
    echo -e "${YELLOW}⚠️  Root permission required to install to $INSTALL_DIR${NC}"
    sudo cp "$BINARY_PATH" "$INSTALL_DIR/cmcp"
fi

# Verify installation
if ! command -v cmcp >/dev/null 2>&1; then
    echo -e "${RED}Installation failed. Please check your PATH includes $INSTALL_DIR${NC}"
    exit 1
fi

# Setup shell completion
SHELL_NAME=$(basename "$SHELL")
echo "Setting up shell completion for $SHELL_NAME..."

case "$SHELL_NAME" in
    bash)
        COMPLETION_DIR=""
        if [ -d "/etc/bash_completion.d" ]; then
            COMPLETION_DIR="/etc/bash_completion.d"
        elif [ -d "/usr/local/etc/bash_completion.d" ]; then
            COMPLETION_DIR="/usr/local/etc/bash_completion.d"
        fi
        
        if [ -n "$COMPLETION_DIR" ]; then
            if [ -w "$COMPLETION_DIR" ]; then
                cmcp completion bash > "$COMPLETION_DIR/cmcp"
            else
                sudo cmcp completion bash > "$COMPLETION_DIR/cmcp"
            fi
            echo -e "${GREEN}✓ Bash completion installed${NC}"
        fi
        ;;
    zsh)
        if [ -n "$FPATH" ]; then
            COMPLETION_FILE="${FPATH%%:*}/_cmcp"
            if [ -w "${FPATH%%:*}" ]; then
                cmcp completion zsh > "$COMPLETION_FILE"
            else
                sudo cmcp completion zsh > "$COMPLETION_FILE"
            fi
            echo -e "${GREEN}✓ Zsh completion installed${NC}"
        fi
        ;;
    fish)
        COMPLETION_DIR="$HOME/.config/fish/completions"
        mkdir -p "$COMPLETION_DIR"
        cmcp completion fish > "$COMPLETION_DIR/cmcp.fish"
        echo -e "${GREEN}✓ Fish completion installed${NC}"
        ;;
esac

# Check for existing config
if [ -f "$CONFIG_DIR/config.json" ]; then
    SERVER_COUNT=$(jq -r '.mcpServers | length' "$CONFIG_DIR/config.json" 2>/dev/null || echo "0")
    echo ""
    echo -e "${GREEN}📁 Configuration preserved: $SERVER_COUNT server(s) available${NC}"
    echo "   Run 'cmcp config list' to see your servers"
fi

# Cleanup
cd /
rm -rf "$TEMP_DIR"

echo ""
echo -e "${GREEN}✅ cmcp installed successfully!${NC}"
echo ""
echo "🔄 Restart your terminal or run 'source ~/.bashrc' for completions to work"
echo ""
echo "Quick start:"
echo "  cmcp --help           # Show help"
echo "  cmcp config list      # List configured servers"
echo "  cmcp start            # Start MCP servers"
echo ""
echo "To uninstall:"
echo "  curl -sSL https://raw.githubusercontent.com/$REPO_OWNER/$REPO_NAME/main/scripts/web-uninstall.sh | bash"