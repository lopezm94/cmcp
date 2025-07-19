package mcp

import (
	"fmt"
	"os"
	"os/exec"

	"cmcp/internal/config"
)

type Manager struct {
	// No need to track servers - Claude manages them
}

func NewManager() *Manager {
	return &Manager{}
}

// findClaude returns the claude command path
func findClaude() string {
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}
	return "claude" // fallback
}

func (m *Manager) StartServer(name string, server *config.MCPServer) error {
	// Build the claude mcp add command
	args := []string{"mcp", "add", name, server.Command}
	args = append(args, server.Args...)
	
	// Add environment variables as options
	if server.Env != nil {
		for k, v := range server.Env {
			args = append(args, "--env", fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	// Execute claude mcp add
	cmd := exec.Command(findClaude(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add server '%s' to Claude: %w", name, err)
	}

	return nil
}

func (m *Manager) StopServer(name string) error {
	// First check if server exists in Claude
	if !m.IsRunning(name) {
		return fmt.Errorf("server '%s' is not registered in Claude", name)
	}

	// Execute claude mcp remove
	cmd := exec.Command(findClaude(), "mcp", "remove", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove server '%s' from Claude: %w", name, err)
	}

	return nil
}

func (m *Manager) GetRunningServers() []string {
	// This method is no longer used since we delegate to claude mcp list
	// Keeping for backward compatibility but returns empty
	return []string{}
}

func (m *Manager) StopAllServers() error {
	// Get list of servers from config to remove them all
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var errors []error
	for name := range cfg.MCPServers {
		// Check if this server is in Claude before trying to remove
		if m.IsRunning(name) {
			cmd := exec.Command(findClaude(), "mcp", "remove", name)
			if err := cmd.Run(); err != nil {
				errors = append(errors, fmt.Errorf("failed to remove server '%s' from Claude: %w", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errors)
	}
	return nil
}

func (m *Manager) IsRunning(name string) bool {
	// Check if server is registered in Claude by running claude mcp get
	cmd := exec.Command(findClaude(), "mcp", "get", name)
	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	// If the command succeeds, the server exists in Claude
	return err == nil
}