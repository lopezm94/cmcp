#!/bin/bash

set -e

# Source color definitions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/colors.sh"

# Detect if this is an upgrade or fresh install
IS_UPGRADE=false
if command -v cmcp >/dev/null 2>&1; then
    IS_UPGRADE=true
    print_header "🔄 Detected existing cmcp installation"
    CURRENT_VERSION=$(cmcp help | head -1 || echo "Unknown version")
    print_detail "Current: $CURRENT_VERSION"
    echo
    
    # Check for running servers if upgrading
    RUNNING_SERVERS=$(cmcp online 2>/dev/null | grep -v "No servers" | grep -v "No MCP servers" || true)
    if [[ -n "$RUNNING_SERVERS" ]]; then
        print_warning "Found running MCP servers"
        print_detail "It's recommended to stop them before upgrading"
        echo -en "   ${YELLOW}Stop all running servers? (y/N): ${RESET}"
        read -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_detail "Stopping servers..."
            cmcp reset <<< "y" >/dev/null 2>&1 || true
            print_success "Servers stopped"
        fi
    fi
    echo
else
    print_header "🚀 Installing cmcp..."
fi

print_step "Building cmcp..."
go build -o cmcp

echo
print_step "Installing cmcp to /usr/local/bin..."
print_warning "Root permission required to:"
print_detail "• Install the cmcp binary to /usr/local/bin (system-wide access)"
print_detail "• Set up shell completions in system directories"
echo
# Use -p flag to provide a custom prompt
sudo -p "Password: " cp cmcp /usr/local/bin/

print_step "Setting up shell completion..."

# Detect shell and install completion
if [[ "$SHELL" == *"zsh"* ]]; then
    print_info "Detected zsh - installing completion..."
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
        print_detail "Installing to $ZSH_COMP_DIR/_cmcp..."
        cmcp completion zsh | sudo -p "Password: " tee "$ZSH_COMP_DIR/_cmcp" > /dev/null
        print_success "Zsh completion installed to standard location"
    else
        # Fallback to user directory
        print_detail "No standard completion directory found, using user directory..."
        mkdir -p ~/.config/cmcp
        cmcp completion zsh > ~/.config/cmcp/_cmcp
        # Add to fpath if not already there
        if ! grep -q "~/.config/cmcp" ~/.zshrc 2>/dev/null; then
            echo 'fpath=(~/.config/cmcp $fpath)' >> ~/.zshrc
            echo 'autoload -U compinit && compinit' >> ~/.zshrc
        fi
    fi
    print_info "Restart your terminal or run 'source ~/.zshrc' to enable completion"
elif [[ "$SHELL" == *"bash"* ]]; then
    print_info "Detected bash - installing completion..."
    if command -v brew >/dev/null 2>&1; then
        # macOS with Homebrew
        COMPLETION_DIR="$(brew --prefix)/etc/bash_completion.d"
        sudo -p "Password: " mkdir -p "$COMPLETION_DIR"
        cmcp completion bash | sudo -p "Password: " tee "$COMPLETION_DIR/cmcp" > /dev/null
        print_success "Bash completion installed"
    else
        # Linux
        sudo -p "Password: " mkdir -p /etc/bash_completion.d
        cmcp completion bash | sudo -p "Password: " tee /etc/bash_completion.d/cmcp > /dev/null
        print_success "Bash completion installed"
    fi
    print_info "Restart your terminal to enable completion"
elif [[ "$SHELL" == *"fish"* ]]; then
    print_info "Detected fish - installing completion..."
    mkdir -p ~/.config/fish/completions
    cmcp completion fish > ~/.config/fish/completions/cmcp.fish
    print_success "Fish completion installed"
    print_info "Restart your terminal to enable completion"
else
    print_warning "Shell not detected"
    print_detail "You can manually install completion with:"
    print_command "cmcp completion [bash|zsh|fish] > [completion-file]"
fi

echo
if [[ "$IS_UPGRADE" == "true" ]]; then
    print_success "cmcp upgraded successfully!"
    NEW_VERSION=$(cmcp help | head -1 || echo "Unknown version")
    print_detail "New: $NEW_VERSION"
else
    print_success "cmcp installed successfully!"
fi
print_info "Restart your terminal to ensure tab completion works"

# Check if existing config exists
if [[ -f ~/.cmcp/config.json ]]; then
    SERVER_COUNT=$(jq -r '.mcpServers | length' ~/.cmcp/config.json 2>/dev/null || echo "0")
    echo
    print_header "📁 Configuration preserved: $SERVER_COUNT server(s) available"
    print_command "cmcp config list"
    print_detail "to see your servers"
elif [[ "$IS_UPGRADE" == "false" ]]; then
    echo
    print_header "🚀 Getting Started"
    print_command "cmcp --help"
    print_detail "to see available commands"
fi