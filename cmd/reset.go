package cmd

import (
	"fmt"

	"cmcp/internal/config"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var resetDryRun bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Stop all running MCP servers in Claude for this project",
	Long:  `Stop all currently running MCP servers in Claude for the current project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get our registered servers
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Find which servers from our config are actually running in Claude
		var runningServers []string
		for name := range cfg.MCPServers {
			if builder.IsRunning(name) {
				runningServers = append(runningServers, name)
			}
		}

		if len(runningServers) == 0 {
			color.Yellow("No servers from your config are currently running in Claude for this project.")
			return nil
		}

		color.Cyan("Found %d running server(s) from your config in Claude for this project:\n", len(runningServers))
		for _, name := range runningServers {
			fmt.Printf("  - %s\n", name)
		}

		// Handle dry-run mode
		if resetDryRun {
			yellow := color.New(color.FgYellow)
			yellow.Println("\nWould execute the following commands:")
			fmt.Println()

			// Build commands for all running servers
			commands := builder.BuildResetCommands(runningServers)
			for _, cmd := range commands {
				fmt.Printf("$ %s\n", cmd)
			}
			return nil
		}

		prompt := promptui.Prompt{
			Label:     "Are you sure you want to stop all servers in Claude for this project",
			IsConfirm: true,
		}

		_, err = prompt.Run()
		if err != nil {
			return nil
		}

		color.Cyan("Stopping all servers...")
		if err := builder.StopAllServers(); err != nil {
			return fmt.Errorf("failed to stop servers: %w", err)
		}

		color.Green("Successfully stopped all servers.")
		return nil
	},
}

func init() {
	resetCmd.Flags().BoolVarP(&resetDryRun, "dry-run", "n", false, "Show commands that would be executed without running them")
}
