#!/bin/bash
# Web installer for cmcp
# Usage: curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/scripts/web-install.sh | bash
#
# This script downloads the cmcp repository and runs the install.sh script

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
TEMP_DIR="/tmp/cmcp-install-$$"

echo -e "${BLUE}🚀 Installing cmcp via web installer...${NC}"

# Create temp directory
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# Check if git is available
if ! command -v git >/dev/null 2>&1; then
    echo -e "${RED}Git is required for installation. Please install git first.${NC}"
    exit 1
fi

# Clone the repository
echo "Downloading cmcp..."
if ! git clone "https://github.com/$REPO_OWNER/$REPO_NAME.git" . >/dev/null 2>&1; then
    echo -e "${RED}Failed to download cmcp repository${NC}"
    exit 1
fi

# Run the install script
echo "Running installer..."
if ./scripts/install.sh; then
    # Cleanup
    cd /
    rm -rf "$TEMP_DIR"
    
    echo ""
    echo "To uninstall:"
    echo "  curl -sSL https://raw.githubusercontent.com/$REPO_OWNER/$REPO_NAME/main/scripts/web-uninstall.sh | bash"
else
    # Cleanup on failure
    cd /
    rm -rf "$TEMP_DIR"
    exit 1
fi