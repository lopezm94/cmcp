package cmd

import (
	"testing"

	"cmcp/internal/mcp"
)

func TestStopCommandBuilder(t *testing.T) {
	// Test the command building functionality
	m := mcp.NewManager()

	tests := []struct {
		name       string
		serverName string
		expected   string
	}{
		{
			name:       "stop command for test-server",
			serverName: "test-server",
			expected:   "claude mcp remove test-server",
		},
		{
			name:       "stop command for server with spaces",
			serverName: "my server",
			expected:   "claude mcp remove my server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := m.BuildStopCommand(tt.serverName)

			if cmd != tt.expected {
				t.Errorf("expected command %q, got: %s", tt.expected, cmd)
			}
		})
	}
}
