package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"cmcp/internal/config"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop running MCP servers",
	Long:  `Stop one or more running MCP servers. Shows only servers that are currently running.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get our registered servers
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Find which servers from our config are in Claude
		var runningServers []string
		for name := range cfg.MCPServers {
			if manager.IsRunning(name) {
				runningServers = append(runningServers, name)
			}
		}

		if len(runningServers) == 0 {
			fmt.Println("No servers from your config are currently in Claude.")
			return nil
		}

		var serverLabels []string
		for _, name := range runningServers {
			serverLabels = append(serverLabels, name)
		}

		var selectedServers []string
		prompt := &survey.MultiSelect{
			Message: "Select servers to stop (use space to select, enter to confirm):",
			Options: serverLabels,
		}

		err = survey.AskOne(prompt, &selectedServers, survey.WithPageSize(10))
		if err != nil {
			return err
		}

		if len(selectedServers) == 0 {
			fmt.Println("No servers selected.")
			return nil
		}

		// Stop each selected server
		var errors []error
		var stopped []string
		for _, serverName := range selectedServers {
			fmt.Printf("Stopping server '%s'...\n", serverName)
			
			if err := manager.StopServer(serverName); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop '%s': %w", serverName, err))
			} else {
				stopped = append(stopped, serverName)
				fmt.Printf("✓ Successfully stopped server '%s'\n", serverName)
			}
		}

		if len(stopped) > 0 {
			fmt.Printf("\nStopped %d server(s): %v\n", len(stopped), stopped)
		}
		
		if len(errors) > 0 {
			fmt.Printf("\nErrors occurred:\n")
			for _, err := range errors {
				fmt.Printf("  • %v\n", err)
			}
		}

		return nil
	},
}