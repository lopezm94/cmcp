#!/bin/bash

set -e

echo "Building cmcp..."
go build -o cmcp

echo "Installing cmcp to /usr/local/bin..."
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
echo "✅ cmcp installed successfully!"
echo "🔄 Restart your terminal to enable tab completion"
echo "🚀 Run 'cmcp --help' to get started"