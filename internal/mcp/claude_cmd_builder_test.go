package mcp

import (
	"testing"

	"cmcp/internal/config"
)

func TestBuildStartCommand(t *testing.T) {
	b := NewClaudeCmdBuilder()

	tests := []struct {
		name     string
		server   *config.MCPServer
		expected string
	}{
		{
			name: "simple command without args",
			server: &config.MCPServer{
				Command: "npx",
				Args:    []string{},
			},
			expected: "claude mcp add test-server -- npx",
		},
		{
			name: "command with args",
			server: &config.MCPServer{
				Command: "npx",
				Args:    []string{"-y", "@upstash/context7-mcp"},
			},
			expected: "claude mcp add test-server -- npx -y @upstash/context7-mcp",
		},
		{
			name: "command with environment variables",
			server: &config.MCPServer{
				Command: "python",
				Args:    []string{"server.py"},
				Env: map[string]string{
					"API_KEY": "secret123",
					"PORT":    "8080",
				},
			},
			expected: "claude mcp add test-server --env API_KEY=secret123 --env PORT=8080 -- python server.py",
		},
		{
			name: "complex command with args and env",
			server: &config.MCPServer{
				Command: "node",
				Args:    []string{"--experimental-modules", "server.mjs", "--port", "3000"},
				Env: map[string]string{
					"NODE_ENV": "production",
				},
			},
			expected: "claude mcp add test-server --env NODE_ENV=production -- node --experimental-modules server.mjs --port 3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.BuildStartCommand("test-server", tt.server)
			
			// Note: Map iteration order is not guaranteed, so for tests with multiple env vars,
			// we should check that all parts are present rather than exact string match
			if tt.server.Env != nil && len(tt.server.Env) > 1 {
				// Check that the command starts correctly
				if !contains(result, "claude mcp add test-server") {
					t.Errorf("Command should start with 'claude mcp add test-server'")
				}
				
				// Check that all env vars are present
				for k, v := range tt.server.Env {
					envStr := "--env " + k + "=" + v
					if !contains(result, envStr) {
						t.Errorf("Missing environment variable: %s", envStr)
					}
				}
				
				// Check that the command and args are present
				if !contains(result, "-- "+tt.server.Command) {
					t.Errorf("Missing command separator and command")
				}
			} else {
				// For deterministic cases, do exact match
				if result != tt.expected {
					t.Errorf("BuildStartCommand() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestBuildStopCommand(t *testing.T) {
	b := NewClaudeCmdBuilder()

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "test-server",
			expected: "claude mcp remove test-server",
		},
		{
			name:     "context7",
			expected: "claude mcp remove context7",
		},
		{
			name:     "server-with-dashes",
			expected: "claude mcp remove server-with-dashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.BuildStopCommand(tt.name)
			if result != tt.expected {
				t.Errorf("BuildStopCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildListCommand(t *testing.T) {
	b := NewClaudeCmdBuilder()
	
	expected := "claude mcp list"
	result := b.BuildListCommand()
	
	if result != expected {
		t.Errorf("BuildListCommand() = %v, want %v", result, expected)
	}
}

func TestBuildResetCommands(t *testing.T) {
	b := NewClaudeCmdBuilder()

	t.Run("reset builds remove commands for specified servers", func(t *testing.T) {
		// List of servers to reset
		serverNames := []string{"server1", "server2", "server3"}
		
		// Get the reset commands
		commands := b.BuildResetCommands(serverNames)
		
		// Should generate remove commands for each server
		if len(commands) != 3 {
			t.Errorf("expected 3 commands, got %d", len(commands))
		}
		
		// Verify the exact commands
		expectedCommands := []string{
			"claude mcp remove server1",
			"claude mcp remove server2",
			"claude mcp remove server3",
		}
		
		for i, cmd := range commands {
			if cmd != expectedCommands[i] {
				t.Errorf("command %d: expected %q, got %q", i, expectedCommands[i], cmd)
			}
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}