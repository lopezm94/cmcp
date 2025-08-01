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
			expected: "claude mcp add test-server --env API_KEY=*** --env PORT=8080 -- python server.py",
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

				// Check that all env vars are present (with masking for sensitive ones)
				for k, v := range tt.server.Env {
					expectedVal := v
					if isSensitiveKey(k) {
						expectedVal = "***"
					}
					envStr := "--env " + k + "=" + expectedVal
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

func TestBuildStartCommandJSON(t *testing.T) {
	b := NewClaudeCmdBuilder()

	tests := []struct {
		name        string
		server      *config.MCPServer
		pretty      bool
		contains    []string
		notContains []string
	}{
		{
			name: "docker server with env",
			server: &config.MCPServer{
				Command: "docker",
				Args:    []string{"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"},
				Env: map[string]string{
					"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_secret123",
				},
			},
			pretty: false,
			contains: []string{
				"claude mcp add-json test-server",
				`"command":"docker"`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN":"***"`,
			},
			notContains: []string{
				"ghp_secret123", // Should never expose the actual secret
			},
		},
		{
			name: "server with multiple env vars - pretty",
			server: &config.MCPServer{
				Command: "node",
				Args:    []string{"server.js"},
				Env: map[string]string{
					"API_KEY":    "secret",
					"AUTH_TOKEN": "token123",
				},
			},
			pretty: true,
			contains: []string{
				"claude mcp add-json test-server",
				"$API_KEY",
				"$AUTH_TOKEN",
			},
			notContains: []string{
				"secret",   // Should be replaced with $API_KEY
				"token123", // Should be replaced with $AUTH_TOKEN
			},
		},
		{
			name: "pretty json is multiline",
			server: &config.MCPServer{
				Command: "python",
				Args:    []string{"server.py"},
				Env: map[string]string{
					"SECRET_KEY": "mysecret",
				},
			},
			pretty: true,
			contains: []string{
				"{\n", // Should have newlines
				"$SECRET_KEY",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.BuildStartCommandJSON("test-server", tt.server, tt.pretty)

			for _, expected := range tt.contains {
				if !contains(result, expected) {
					t.Errorf("BuildStartCommandJSON() should contain %q, got %v", expected, result)
				}
			}

			for _, notExpected := range tt.notContains {
				if contains(result, notExpected) {
					t.Errorf("BuildStartCommandJSON() should NOT contain %q (security leak!), got %v", notExpected, result)
				}
			}
		})
	}
}

func TestDryRunBehavior(t *testing.T) {
	b := NewClaudeCmdBuilder()

	// Test that servers with env vars use add-json
	serverWithEnv := &config.MCPServer{
		Command: "docker",
		Args:    []string{"run", "-e", "TOKEN", "image"},
		Env: map[string]string{
			"TOKEN": "secret123",
		},
	}

	// Test that servers without env vars use regular add
	serverNoEnv := &config.MCPServer{
		Command: "npx",
		Args:    []string{"-y", "@some/package"},
	}

	// For server with env, should use BuildStartCommandJSON
	cmdWithEnv := b.BuildStartCommandJSON("server1", serverWithEnv, true)
	if !contains(cmdWithEnv, "add-json") {
		t.Error("Server with env vars should use add-json")
	}
	if contains(cmdWithEnv, "secret123") {
		t.Error("Should not expose secret in dry-run")
	}

	// For server without env, should use BuildStartCommand
	cmdNoEnv := b.BuildStartCommand("server2", serverNoEnv)
	if contains(cmdNoEnv, "add-json") {
		t.Error("Server without env vars should use regular add")
	}
	if !contains(cmdNoEnv, "claude mcp add server2 -- npx") {
		t.Error("Should use regular add command format")
	}
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
