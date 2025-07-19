#!/bin/bash

set -e

# Detect if this is an upgrade or fresh install
IS_UPGRADE=false
if command -v cmcp >/dev/null 2>&1; then
    IS_UPGRADE=true
    echo "🔄 Detected existing cmcp installation"
    CURRENT_VERSION=$(cmcp help | head -1 || echo "Unknown version")
    echo "   Current: $CURRENT_VERSION"
    echo ""
    
    # Check for running servers if upgrading
    RUNNING_SERVERS=$(cmcp online 2>/dev/null | grep -v "No servers" | grep -v "No MCP servers" || true)
    if [[ -n "$RUNNING_SERVERS" ]]; then
        echo "⚠️  Found running MCP servers"
        echo "   It's recommended to stop them before upgrading"
        read -p "   Stop all running servers? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "   Stopping servers..."
            cmcp reset <<< "y" >/dev/null 2>&1 || true
            echo "   ✅ Servers stopped"
        fi
    fi
    echo ""
else
    echo "🚀 Installing cmcp..."
fi

echo "Building cmcp..."
go build -o cmcp

echo ""
echo "Installing cmcp to /usr/local/bin..."
echo "⚠️  Root permission required to:"
echo "   • Install the cmcp binary to /usr/local/bin (system-wide access)"
echo "   • Set up shell completions in system directories"
echo ""
sudo cp cmcp /usr/local/bin/

echo "Setting up shell completion..."

# Detect shell and install completion
if [[ "$SHELL" == *"zsh"* ]]; then
    echo "Detected zsh - installing completion..."
    # Find the best completion directory
    ZSH_COMP_DIR=""
    for dir in /opt/homebrew/share/zsh/site-functions /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions; do
        if [[ -d "$dir" ]]; then
            ZSH_COMP_DIR="$dir"
            break
        fi
    done
    
    if [[ -n "$ZSH_COMP_DIR" ]]; then
        # Install to standard location
        echo "Installing to $ZSH_COMP_DIR/_cmcp..."
        cmcp completion zsh | sudo tee "$ZSH_COMP_DIR/_cmcp" > /dev/null
        echo "Zsh completion installed to standard location."
    else
        # Fallback to user directory
        echo "No standard completion directory found, using user directory..."
        mkdir -p ~/.config/cmcp
        cmcp completion zsh > ~/.config/cmcp/_cmcp
        # Add to fpath if not already there
        if ! grep -q "~/.config/cmcp" ~/.zshrc 2>/dev/null; then
            echo 'fpath=(~/.config/cmcp $fpath)' >> ~/.zshrc
            echo 'autoload -U compinit && compinit' >> ~/.zshrc
        fi
    fi
    echo "Zsh completion installed. Restart your terminal or run 'source ~/.zshrc'"
elif [[ "$SHELL" == *"bash"* ]]; then
    echo "Detected bash - installing completion..."
    if command -v brew >/dev/null 2>&1; then
        # macOS with Homebrew
        COMPLETION_DIR="$(brew --prefix)/etc/bash_completion.d"
        sudo mkdir -p "$COMPLETION_DIR"
        cmcp completion bash | sudo tee "$COMPLETION_DIR/cmcp" > /dev/null
        echo "Bash completion installed. Restart your terminal."
    else
        # Linux
        sudo mkdir -p /etc/bash_completion.d
        cmcp completion bash | sudo tee /etc/bash_completion.d/cmcp > /dev/null
        echo "Bash completion installed. Restart your terminal."
    fi
elif [[ "$SHELL" == *"fish"* ]]; then
    echo "Detected fish - installing completion..."
    mkdir -p ~/.config/fish/completions
    cmcp completion fish > ~/.config/fish/completions/cmcp.fish
    echo "Fish completion installed. Restart your terminal."
else
    echo "Shell not detected. You can manually install completion with:"
    echo "  cmcp completion [bash|zsh|fish] > [completion-file]"
fi

echo ""
if [[ "$IS_UPGRADE" == "true" ]]; then
    echo "✅ cmcp upgraded successfully!"
    NEW_VERSION=$(cmcp help | head -1 || echo "Unknown version")
    echo "   New: $NEW_VERSION"
else
    echo "✅ cmcp installed successfully!"
fi
echo "🔄 Restart your terminal to ensure tab completion works"

# Check if existing config exists
if [[ -f ~/.cmcp/config.json ]]; then
    SERVER_COUNT=$(jq -r '.mcpServers | length' ~/.cmcp/config.json 2>/dev/null || echo "0")
    echo ""
    echo "📁 Configuration preserved: $SERVER_COUNT server(s) available"
    echo "   Run 'cmcp config list' to see your servers"
elif [[ "$IS_UPGRADE" == "false" ]]; then
    echo "🚀 Run 'cmcp --help' to get started"
fi