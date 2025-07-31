#!/bin/bash
# Web uninstaller for cmcp
# Usage: curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/web-uninstall.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.cmcp"

echo -e "${BLUE}🗑️  Uninstalling cmcp...${NC}"

# Check if cmcp is installed
if ! command -v cmcp >/dev/null 2>&1; then
    echo -e "${YELLOW}cmcp is not installed.${NC}"
    exit 0
fi

# Get installation path
CMCP_PATH=$(which cmcp)
echo "Found cmcp at: $CMCP_PATH"

# Ask about configuration
REMOVE_CONFIG="n"
if [ -d "$CONFIG_DIR" ]; then
    echo ""
    echo -e "${YELLOW}Configuration found at $CONFIG_DIR${NC}"
    echo -n "Do you want to remove your configuration? (y/N): "
    
    # Read from stdin (works with piped input)
    if [ -t 0 ]; then
        read -r REMOVE_CONFIG
    else
        # If piped, read from stdin
        read -r REMOVE_CONFIG
        echo "$REMOVE_CONFIG"
    fi
fi

# Remove binary
echo -e "${YELLOW}Removing cmcp binary...${NC}"
if [ -w "$(dirname "$CMCP_PATH")" ]; then
    rm -f "$CMCP_PATH"
else
    echo -e "${YELLOW}⚠️  Root permission required to remove cmcp from $(dirname "$CMCP_PATH")${NC}"
    sudo rm -f "$CMCP_PATH"
fi

# Remove shell completions
SHELL_NAME=$(basename "$SHELL")
echo "Removing shell completions..."

case "$SHELL_NAME" in
    bash)
        for dir in /etc/bash_completion.d /usr/local/etc/bash_completion.d; do
            if [ -f "$dir/cmcp" ]; then
                if [ -w "$dir" ]; then
                    rm -f "$dir/cmcp"
                else
                    sudo rm -f "$dir/cmcp"
                fi
                echo -e "${GREEN}✓ Removed bash completion${NC}"
            fi
        done
        ;;
    zsh)
        if [ -n "$FPATH" ]; then
            for dir in ${FPATH//:/ }; do
                if [ -f "$dir/_cmcp" ]; then
                    if [ -w "$dir" ]; then
                        rm -f "$dir/_cmcp"
                    else
                        sudo rm -f "$dir/_cmcp"
                    fi
                    echo -e "${GREEN}✓ Removed zsh completion${NC}"
                fi
            done
        fi
        ;;
    fish)
        COMPLETION_FILE="$HOME/.config/fish/completions/cmcp.fish"
        if [ -f "$COMPLETION_FILE" ]; then
            rm -f "$COMPLETION_FILE"
            echo -e "${GREEN}✓ Removed fish completion${NC}"
        fi
        ;;
esac

# Remove configuration if requested
if [[ "$REMOVE_CONFIG" == "y" ]] || [[ "$REMOVE_CONFIG" == "Y" ]]; then
    echo -e "${YELLOW}Removing configuration...${NC}"
    rm -rf "$CONFIG_DIR"
    echo -e "${GREEN}✓ Configuration removed${NC}"
else
    if [ -d "$CONFIG_DIR" ]; then
        echo -e "${BLUE}Configuration preserved at $CONFIG_DIR${NC}"
    fi
fi

# Verify removal
if command -v cmcp >/dev/null 2>&1; then
    echo -e "${RED}Warning: cmcp is still accessible. You may have multiple installations.${NC}"
    echo "Found at: $(which cmcp)"
else
    echo ""
    echo -e "${GREEN}✅ cmcp has been successfully uninstalled!${NC}"
fi

echo ""
echo "Thank you for using cmcp!"