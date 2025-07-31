package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"cmcp/internal/config"
	"cmcp/internal/mcp"
	"github.com/spf13/cobra"
)

var manager = mcp.NewManager()

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start MCP servers",
	Long:  `Start one or more MCP servers from your registered servers. Shows only servers that are not currently running.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCPServers) == 0 {
			fmt.Println("No servers configured. Use 'cmcp config open' to add servers.")
			return nil
		}

		var availableServers []string
		var serverLabels []string

		for name := range cfg.MCPServers {
			if !manager.IsRunning(name) {
				availableServers = append(availableServers, name)
				serverLabels = append(serverLabels, name)
			}
		}

		if len(availableServers) == 0 {
			fmt.Println("All registered servers are already running.")
			return nil
		}

		var selectedServers []string
		prompt := &survey.MultiSelect{
			Message: "Select servers to start (use space to select, enter to confirm):",
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

		// Start each selected server
		var errors []error
		var started []string
		for _, serverName := range selectedServers {
			selectedServer, _ := cfg.FindServer(serverName)
			fmt.Printf("Starting server '%s'...\n", serverName)
			
			if err := manager.StartServer(serverName, selectedServer); err != nil {
				errors = append(errors, fmt.Errorf("failed to start '%s': %w", serverName, err))
			} else {
				started = append(started, serverName)
				fmt.Printf("✓ Successfully started server '%s'\n", serverName)
			}
		}

		if len(started) > 0 {
			fmt.Printf("\nStarted %d server(s): %v\n", len(started), started)
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