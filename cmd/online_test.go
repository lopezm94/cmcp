package cmd

import (
	"testing"

	"cmcp/internal/mcp"
)

func TestOnlineCommandBuilder(t *testing.T) {
	// Test the command building functionality
	m := mcp.NewManager()

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "list command",
			expected: "claude mcp list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := m.BuildListCommand()

			if cmd != tt.expected {
				t.Errorf("expected command %q, got: %s", tt.expected, cmd)
			}
		})
	}
}
