#!/bin/bash
# Web uninstaller for cmcp
# Usage: curl -sSL https://raw.githubusercontent.com/lopezm94/cmcp/main/scripts/web-uninstall.sh | bash
#
# This script downloads the cmcp repository and runs the uninstall.sh script

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
TEMP_DIR="/tmp/cmcp-uninstall-$$"

echo -e "${BLUE}🗑️  Uninstalling cmcp via web uninstaller...${NC}"

# Check if cmcp is installed
if ! command -v cmcp >/dev/null 2>&1; then
    echo -e "${YELLOW}cmcp is not installed.${NC}"
    exit 0
fi

# Create temp directory
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# Check if git is available
if ! command -v git >/dev/null 2>&1; then
    echo -e "${RED}Git is required for uninstallation. Please install git first.${NC}"
    exit 1
fi

# Clone the repository
echo "Downloading uninstaller..."
if ! git clone "https://github.com/$REPO_OWNER/$REPO_NAME.git" . >/dev/null 2>&1; then
    echo -e "${RED}Failed to download cmcp repository${NC}"
    exit 1
fi

# Run the uninstall script
echo "Running uninstaller..."
if ./scripts/uninstall.sh; then
    # Cleanup
    cd /
    rm -rf "$TEMP_DIR"
else
    # Cleanup on failure
    cd /
    rm -rf "$TEMP_DIR"
    exit 1
fi