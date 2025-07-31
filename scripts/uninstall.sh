#!/bin/bash

set -e

echo "Uninstalling cmcp..."
echo ""
echo "⚠️  Root permission will be required to:"
echo "   • Remove the cmcp binary from /usr/local/bin"
echo "   • Remove shell completions from system directories"
echo ""

# Remove binary
if [[ -f "/usr/local/bin/cmcp" ]]; then
    echo "Removing binary from /usr/local/bin..."
    sudo rm -f /usr/local/bin/cmcp
else
    echo "Binary not found in /usr/local/bin"
fi

# Remove shell completions
echo "Removing shell completions..."

# Zsh completion - check standard locations
ZSH_COMPLETIONS_REMOVED=false
for dir in /opt/homebrew/share/zsh/site-functions /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions ~/.config/cmcp; do
    if [[ -f "$dir/_cmcp" ]]; then
        echo "Removing zsh completion from $dir..."
        if [[ "$dir" == "/opt/homebrew"* || "$dir" == "/usr"* ]]; then
            sudo rm -f "$dir/_cmcp"
        else
            rm -f "$dir/_cmcp"
        fi
        ZSH_COMPLETIONS_REMOVED=true
    fi
done

# Clean up empty directory if we created it
if [[ -d ~/.config/cmcp ]] && [[ -z "$(ls -A ~/.config/cmcp 2>/dev/null)" ]]; then
    rmdir ~/.config/cmcp 2>/dev/null || true
fi

if [[ "$ZSH_COMPLETIONS_REMOVED" == "false" ]]; then
    echo "No zsh completions found"
fi

# Bash completion
if command -v brew >/dev/null 2>&1; then
    # macOS with Homebrew
    COMPLETION_FILE="$(brew --prefix)/etc/bash_completion.d/cmcp"
    if [[ -f "$COMPLETION_FILE" ]]; then
        echo "Removing bash completion (macOS)..."
        sudo rm -f "$COMPLETION_FILE"
    fi
else
    # Linux
    if [[ -f "/etc/bash_completion.d/cmcp" ]]; then
        echo "Removing bash completion (Linux)..."
        sudo rm -f /etc/bash_completion.d/cmcp
    fi
fi

# Fish completion
if [[ -f ~/.config/fish/completions/cmcp.fish ]]; then
    echo "Removing fish completion..."
    rm -f ~/.config/fish/completions/cmcp.fish
fi

# Remove config registry (ask user)
if [[ -f ~/.cmcp/config.json ]]; then
    echo "Found cmcp configuration registry with your MCP servers"
    echo "This contains all your registered server configurations"
    read -p "Remove configuration registry and all registered servers? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Removing configuration registry..."
        rm -rf ~/.cmcp
        echo "All server configurations removed"
    else
        echo "Keeping configuration registry with your server configurations"
        echo "Note: You can reinstall cmcp and your servers will still be available"
    fi
else
    echo "No configuration registry found"
fi

echo ""
echo "✅ cmcp uninstalled successfully!"
echo "🔄 Restart your terminal to complete removal"