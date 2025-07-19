#!/bin/bash

set -e

echo "🚀 Upgrading cmcp..."
echo "This will update cmcp while preserving your configuration registry"
echo ""

# Check if cmcp is currently installed
if ! command -v cmcp >/dev/null 2>&1; then
    echo "❌ cmcp is not currently installed"
    echo "Use ./install.sh to install cmcp for the first time"
    exit 1
fi

# Show current version info
echo "Current installation:"
CURRENT_VERSION=$(cmcp help | head -1 || echo "Unknown version")
echo "  $CURRENT_VERSION"
echo ""

# Check if config exists and show info
if [[ -f ~/.cmcp/config.json ]]; then
    echo "📁 Configuration registry found"
    SERVER_COUNT=$(jq -r '.mcpServers | length' ~/.cmcp/config.json 2>/dev/null || echo "unknown")
    echo "  • Registered servers: $SERVER_COUNT"
    echo "  • Location: ~/.cmcp/config.json"
    echo "  • ✅ Configuration will be preserved during upgrade"
else
    echo "📁 No existing configuration registry found"
fi
echo ""

# Confirm upgrade
read -p "Continue with upgrade? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Upgrade cancelled"
    exit 0
fi

echo ""
echo "🔄 Starting upgrade process..."

# Step 1: Build the new version
echo "1️⃣  Building new version..."
if ! go build -o cmcp; then
    echo "❌ Failed to build cmcp"
    exit 1
fi
echo "   ✅ Build successful"

# Step 2: Stop any running servers (optional, with user consent)
if command -v cmcp >/dev/null 2>&1; then
    echo ""
    echo "2️⃣  Checking for running servers..."
    
    # Create a temporary script to check for running servers
    RUNNING_SERVERS=$(./cmcp online 2>/dev/null | grep -v "No servers" | grep -v "No MCP servers" || true)
    
    if [[ -n "$RUNNING_SERVERS" ]]; then
        echo "   ⚠️  Found running MCP servers"
        echo "   It's recommended to stop them before upgrading"
        read -p "   Stop all running servers? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "   Stopping servers..."
            ./cmcp reset <<< "y" >/dev/null 2>&1 || true
            echo "   ✅ Servers stopped"
        else
            echo "   ⚠️  Continuing with servers running"
        fi
    else
        echo "   ✅ No running servers found"
    fi
fi

# Step 3: Install the new binary
echo ""
echo "3️⃣  Installing new binary..."
if [[ -w "/usr/local/bin" ]]; then
    cp cmcp /usr/local/bin/cmcp
else
    sudo cp cmcp /usr/local/bin/cmcp
fi
echo "   ✅ Binary updated at /usr/local/bin/cmcp"

# Step 4: Update shell completions (only if they were previously installed)
echo ""
echo "4️⃣  Updating shell completions..."

# Check if completions were previously installed and update them
COMPLETIONS_UPDATED=false

# Zsh completion
for dir in /opt/homebrew/share/zsh/site-functions /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions ~/.config/cmcp; do
    if [[ -f "$dir/_cmcp" ]]; then
        echo "   Updating zsh completion at $dir..."
        if [[ "$dir" == "/opt/homebrew"* || "$dir" == "/usr"* ]]; then
            cmcp completion zsh | sudo tee "$dir/_cmcp" >/dev/null
        else
            cmcp completion zsh > "$dir/_cmcp"
        fi
        COMPLETIONS_UPDATED=true
    fi
done

# Bash completion
if command -v brew >/dev/null 2>&1; then
    # macOS with Homebrew
    COMPLETION_FILE="$(brew --prefix)/etc/bash_completion.d/cmcp"
    if [[ -f "$COMPLETION_FILE" ]]; then
        echo "   Updating bash completion (macOS)..."
        cmcp completion bash | sudo tee "$COMPLETION_FILE" >/dev/null
        COMPLETIONS_UPDATED=true
    fi
else
    # Linux
    if [[ -f "/etc/bash_completion.d/cmcp" ]]; then
        echo "   Updating bash completion (Linux)..."
        cmcp completion bash | sudo tee /etc/bash_completion.d/cmcp >/dev/null
        COMPLETIONS_UPDATED=true
    fi
fi

# Fish completion
if [[ -f ~/.config/fish/completions/cmcp.fish ]]; then
    echo "   Updating fish completion..."
    mkdir -p ~/.config/fish/completions
    cmcp completion fish > ~/.config/fish/completions/cmcp.fish
    COMPLETIONS_UPDATED=true
fi

if [[ "$COMPLETIONS_UPDATED" == "true" ]]; then
    echo "   ✅ Shell completions updated"
else
    echo "   ℹ️  No existing completions found to update"
fi

# Step 5: Verify installation
echo ""
echo "5️⃣  Verifying installation..."
if command -v cmcp >/dev/null 2>&1; then
    NEW_VERSION=$(cmcp help | head -1 || echo "Unknown version")
    echo "   ✅ cmcp is working"
    echo "   📦 $NEW_VERSION"
else
    echo "   ❌ cmcp not found in PATH"
    exit 1
fi

# Step 6: Show configuration status
echo ""
echo "6️⃣  Configuration status..."
if [[ -f ~/.cmcp/config.json ]]; then
    SERVER_COUNT=$(jq -r '.mcpServers | length' ~/.cmcp/config.json 2>/dev/null || echo "unknown")
    echo "   ✅ Configuration registry preserved"
    echo "   📋 $SERVER_COUNT registered servers available"
else
    echo "   ℹ️  No configuration registry (this is normal for new installations)"
fi

echo ""
echo "🎉 cmcp upgraded successfully!"
echo ""
echo "📋 What's next:"
echo "   • Your server configurations have been preserved"
echo "   • Run 'cmcp config list' to see your registered servers"
echo "   • Run 'cmcp start' to start servers"
echo "   • Restart your terminal to ensure completions work"
echo ""
echo "💡 Tip: Use 'cmcp help' to see all available commands"