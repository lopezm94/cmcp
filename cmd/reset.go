package cmd

import (
	"fmt"

	"cmcp/internal/config"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Stop all running MCP servers",
	Long:  `Stop all currently running MCP servers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get our registered servers
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Find which servers from our config are actually running in Claude
		var runningServers []string
		for name := range cfg.MCPServers {
			if manager.IsRunning(name) {
				runningServers = append(runningServers, name)
			}
		}

		if len(runningServers) == 0 {
			fmt.Println("No servers from your config are currently running in Claude.")
			return nil
		}

		fmt.Printf("Found %d running server(s) from your config:\n", len(runningServers))
		for _, name := range runningServers {
			fmt.Printf("  - %s\n", name)
		}

		prompt := promptui.Prompt{
			Label:     "Are you sure you want to stop all servers",
			IsConfirm: true,
		}

		_, err = prompt.Run()
		if err != nil {
			return nil
		}

		fmt.Println("Stopping all servers...")
		if err := manager.StopAllServers(); err != nil {
			return fmt.Errorf("failed to stop servers: %w", err)
		}

		fmt.Println("Successfully stopped all servers.")
		return nil
	},
}