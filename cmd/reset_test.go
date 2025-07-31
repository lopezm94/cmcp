package cmd

import (
	"fmt"
	"testing"

	"cmcp/internal/mcp"
)

func TestResetCommandBuilder(t *testing.T) {
	// Test the command building functionality for reset
	// Reset uses BuildStopCommand for each server
	m := mcp.NewManager()

	servers := []string{"server1", "server2", "server3"}

	for _, serverName := range servers {
		cmd := m.BuildStopCommand(serverName)
		expected := fmt.Sprintf("claude mcp remove %s", serverName)

		if cmd != expected {
			t.Errorf("expected command %q, got: %s", expected, cmd)
		}
	}
}
