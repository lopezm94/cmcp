package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var onlineDryRun bool

var onlineCmd = &cobra.Command{
	Use:   "online",
	Short: "Show currently running MCP servers",
	Long:  `Display a list of all MCP servers that are currently running in Claude.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle dry-run mode
		if onlineDryRun {
			yellow := color.New(color.FgYellow)
			yellow.Println("Would execute the following command:")
			fmt.Println()
			fmt.Printf("$ %s\n", manager.BuildListCommand())
			return nil
		}
		
		// Execute claude mcp list and capture output
		claudePath := "claude"
		if path, err := exec.LookPath("claude"); err == nil {
			claudePath = path
		}
		
		var stdout, stderr bytes.Buffer
		claudeCmd := exec.Command(claudePath, "mcp", "list")
		claudeCmd.Stdout = &stdout
		claudeCmd.Stderr = &stderr
		
		err := claudeCmd.Run()
		
		// Get the output
		output := stdout.String()
		errorOutput := stderr.String()
		
		// Replace Claude's message with our own
		if strings.Contains(output, "No MCP servers configured. Use `claude mcp add` to add a server.") {
			color.Yellow("No servers are currently running. Use `cmcp start` to start a server.")
			return nil
		}
		if strings.Contains(errorOutput, "No MCP servers configured. Use `claude mcp add` to add a server.") {
			color.Yellow("No servers are currently running. Use `cmcp start` to start a server.")
			return nil
		}
		
		// Write the modified output (which contains the list of servers)
		if output != "" {
			fmt.Fprint(cmd.OutOrStdout(), output)
		}
		if errorOutput != "" {
			fmt.Fprint(cmd.ErrOrStderr(), errorOutput)
		}
		
		if err != nil {
			return fmt.Errorf("failed to list Claude MCP servers: %w", err)
		}
		
		return nil
	},
}

func init() {
	onlineCmd.Flags().BoolVarP(&onlineDryRun, "dry-run", "n", false, "Show command that would be executed without running it")
}