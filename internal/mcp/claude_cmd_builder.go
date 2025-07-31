package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cmcp/internal/config"
)

type ClaudeCmdBuilder struct {
	// Builder for Claude CLI commands
}

func NewClaudeCmdBuilder() *ClaudeCmdBuilder {
	return &ClaudeCmdBuilder{}
}

// findClaude returns the claude command path
func findClaude() string {
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}
	return "claude" // fallback
}

func (b *ClaudeCmdBuilder) StartServer(name string, server *config.MCPServer, verbose bool) error {
	// Build the command args
	args := b.buildStartArgs(name, server)
	
	// Show command if verbose
	if verbose {
		commandStr := b.BuildStartCommand(name, server)
		fmt.Printf("  Command: %s\n", commandStr)
	}
	
	// Execute claude mcp add
	cmd := exec.Command(findClaude(), args...)
	
	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	// Handle output based on verbose flag and error state
	if err != nil {
		// On error, show the full command and stderr
		if !verbose {
			commandStr := b.BuildStartCommand(name, server)
			fmt.Printf("  Command failed: %s\n", commandStr)
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(os.Stderr, "%s", stderr.String())
		}
		return fmt.Errorf("failed to add server '%s' to Claude", name)
	}
	
	// In verbose mode, parse and show only relevant info
	if verbose && stdout.Len() > 0 {
		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Skip the duplicate "Added stdio MCP server..." line
			if strings.Contains(line, "Added stdio MCP server") {
				continue
			}
			// Show file modifications with indentation
			if strings.Contains(line, "File modified:") {
				fmt.Printf("  %s\n", line)
			} else {
				// Show other output as-is
				fmt.Println(line)
			}
		}
	}

	return nil
}

func (b *ClaudeCmdBuilder) StopServer(name string, verbose bool) error {
	// First check if server exists in Claude
	if !b.IsRunning(name) {
		return fmt.Errorf("server '%s' is not registered in Claude", name)
	}

	// Build the command
	commandStr := b.BuildStopCommand(name)
	args := []string{"mcp", "remove", name}
	
	// Show command if verbose
	if verbose {
		fmt.Printf("  Command: %s\n", commandStr)
	}

	// Execute claude mcp remove
	cmd := exec.Command(findClaude(), args...)
	
	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	// Handle output based on verbose flag and error state
	if err != nil {
		// On error, show the full command and stderr
		if !verbose {
			fmt.Printf("  Command failed: %s\n", commandStr)
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(os.Stderr, "%s", stderr.String())
		}
		return fmt.Errorf("failed to remove server '%s' from Claude", name)
	}
	
	// In verbose mode, parse and show only relevant info
	if verbose && stdout.Len() > 0 {
		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Skip the duplicate "Removed MCP server..." line
			if strings.Contains(line, "Removed MCP server") {
				continue
			}
			// Show file modifications with indentation
			if strings.Contains(line, "File modified:") {
				fmt.Printf("  %s\n", line)
			} else {
				// Show other output as-is
				fmt.Println(line)
			}
		}
	}

	return nil
}

func (b *ClaudeCmdBuilder) GetRunningServers() []string {
	// This method is no longer used since we delegate to claude mcp list
	// Keeping for backward compatibility but returns empty
	return []string{}
}

func (b *ClaudeCmdBuilder) StopAllServers() error {
	// Get list of servers from config to remove them all
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var errors []error
	for name := range cfg.MCPServers {
		// Check if this server is in Claude before trying to remove
		if b.IsRunning(name) {
			// Use StopServer with verbose=false for reset command
			if err := b.StopServer(name, false); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errors)
	}
	return nil
}

func (b *ClaudeCmdBuilder) IsRunning(name string) bool {
	// Check if server is registered in Claude by running claude mcp get
	cmd := exec.Command(findClaude(), "mcp", "get", name)
	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	// If the command succeeds, the server exists in Claude
	return err == nil
}

// buildStartArgs constructs the arguments for starting a server
func (b *ClaudeCmdBuilder) buildStartArgs(name string, server *config.MCPServer) []string {
	// Build the claude mcp add command
	args := []string{"mcp", "add", name}
	
	// Add environment variables as options
	if server.Env != nil {
		for k, v := range server.Env {
			args = append(args, "--env", fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	// Add the command and its arguments
	// Use -- to separate claude options from server command args
	args = append(args, "--", server.Command)
	args = append(args, server.Args...)
	
	return args
}

// BuildStartCommand constructs the command to start a server without executing it
func (b *ClaudeCmdBuilder) BuildStartCommand(name string, server *config.MCPServer) string {
	args := b.buildStartArgs(name, server)
	return fmt.Sprintf("claude %s", strings.Join(args, " "))
}

// BuildStopCommand constructs the command to stop a server without executing it
func (b *ClaudeCmdBuilder) BuildStopCommand(name string) string {
	return fmt.Sprintf("claude mcp remove %s", name)
}

// BuildListCommand constructs the command to list servers without executing it
func (b *ClaudeCmdBuilder) BuildListCommand() string {
	return "claude mcp list"
}

// BuildResetCommands constructs the commands to remove multiple servers without executing them
func (b *ClaudeCmdBuilder) BuildResetCommands(serverNames []string) []string {
	var commands []string
	for _, name := range serverNames {
		commands = append(commands, b.BuildStopCommand(name))
	}
	return commands
}