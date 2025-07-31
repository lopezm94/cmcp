package cmd

import (
	"strings"
	"testing"

	"cmcp/internal/config"
	"cmcp/internal/mcp"
)

func TestStartCommandBuilder(t *testing.T) {
	// Test the command building functionality
	m := mcp.NewManager()

	tests := []struct {
		name     string
		server   *config.MCPServer
		contains []string
	}{
		{
			name: "npx server",
			server: &config.MCPServer{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			},
			contains: []string{
				"claude mcp add",
				"test-server",
				"--",
				"npx",
				"-y",
				"@modelcontextprotocol/server-memory",
			},
		},
		{
			name: "node server with env",
			server: &config.MCPServer{
				Command: "node",
				Args:    []string{"dist/index.js"},
				Env: map[string]string{
					"API_KEY": "test123",
				},
			},
			contains: []string{
				"claude mcp add",
				"test-server",
				"--env",
				"API_KEY=test123",
				"--",
				"node",
				"dist/index.js",
			},
		},
		{
			name: "python server",
			server: &config.MCPServer{
				Command: "python",
				Args:    []string{"-m", "mcp_server"},
			},
			contains: []string{
				"claude mcp add",
				"test-server",
				"--",
				"python",
				"-m",
				"mcp_server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := m.BuildStartCommand("test-server", tt.server)
			
			for _, expected := range tt.contains {
				if !strings.Contains(cmd, expected) {
					t.Errorf("expected command to contain %q, got: %s", expected, cmd)
				}
			}
		})
	}
}

func TestStartDryRunIntegration(t *testing.T) {
	// Test that dry-run mode prevents actual execution
	// This is more of an integration test to ensure the flag works
	t.Run("dry run flag prevents execution", func(t *testing.T) {
		// The actual command tests with interactive prompts would require
		// more complex setup with mocked config and input
		// For now, we're testing the command building logic above
		t.Skip("Skipping integration test that requires interactive input mocking")
	})
}