#!/bin/bash

set -e

# Source color definitions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/colors.sh"

print_header "Uninstalling cmcp..."
echo
print_warning "Root permission will be required to:"
print_detail "• Remove the cmcp binary from /usr/local/bin"
print_detail "• Remove shell completions from system directories"
echo

# Remove binary
if [[ -f "/usr/local/bin/cmcp" ]]; then
    print_step "Removing binary from /usr/local/bin..."
    sudo -p "Password: " rm -f /usr/local/bin/cmcp
else
    print_info "Binary not found in /usr/local/bin"
fi

# Remove shell completions
print_step "Removing shell completions..."

# Zsh completion - check standard locations
ZSH_COMPLETIONS_REMOVED=false
for dir in /opt/homebrew/share/zsh/site-functions /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions ~/.config/cmcp; do
    if [[ -f "$dir/_cmcp" ]]; then
        print_detail "Removing zsh completion from $dir..."
        if [[ "$dir" == "/opt/homebrew"* || "$dir" == "/usr"* ]]; then
            sudo -p "Password: " rm -f "$dir/_cmcp"
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
    print_info "No zsh completions found"
fi

# Bash completion
if command -v brew >/dev/null 2>&1; then
    # macOS with Homebrew
    COMPLETION_FILE="$(brew --prefix)/etc/bash_completion.d/cmcp"
    if [[ -f "$COMPLETION_FILE" ]]; then
        print_detail "Removing bash completion (macOS)..."
        sudo -p "Password: " rm -f "$COMPLETION_FILE"
    fi
else
    # Linux
    if [[ -f "/etc/bash_completion.d/cmcp" ]]; then
        print_detail "Removing bash completion (Linux)..."
        sudo -p "Password: " rm -f /etc/bash_completion.d/cmcp
    fi
fi

# Fish completion
if [[ -f ~/.config/fish/completions/cmcp.fish ]]; then
    print_detail "Removing fish completion..."
    rm -f ~/.config/fish/completions/cmcp.fish
fi

# Remove config registry (ask user)
if [[ -f ~/.cmcp/config.json ]]; then
    print_warning "Found cmcp configuration registry with your MCP servers"
    print_detail "This contains all your registered server configurations"
    echo -en "   ${YELLOW}Remove configuration registry and all registered servers? (y/N): ${RESET}"
    read -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_detail "Removing configuration registry..."
        rm -rf ~/.cmcp
        print_success "All server configurations removed"
    else
        print_info "Keeping configuration registry with your server configurations"
        print_detail "Note: You can reinstall cmcp and your servers will still be available"
    fi
else
    print_info "No configuration registry found"
fi

echo
print_success "cmcp uninstalled successfully!"
print_info "Restart your terminal to complete removal"