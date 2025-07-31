package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"cmcp/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	stopVerbose bool
	stopDryRun  bool
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
			color.Yellow("No servers from your config are currently in Claude.")
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
			color.Yellow("No servers selected.")
			return nil
		}

		// Handle dry-run mode
		if stopDryRun {
			yellow := color.New(color.FgYellow)
			yellow.Println("Would execute the following commands:")
			fmt.Println()
			
			for _, serverName := range selectedServers {
				command := manager.BuildStopCommand(serverName)
				fmt.Printf("$ %s\n", command)
			}
			return nil
		}
		
		// Stop each selected server
		var errors []error
		var stopped []string
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		
		for _, serverName := range selectedServers {
			cyan.Printf("Stopping server '%s'...\n", serverName)
			
			if err := manager.StopServer(serverName, stopVerbose); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop '%s': %w", serverName, err))
			} else {
				stopped = append(stopped, serverName)
				green.Printf("✓ Successfully stopped server '%s'\n", serverName)
			}
		}

		if len(stopped) > 0 {
			fmt.Printf("\nStopped %d server(s): %v\n", len(stopped), stopped)
		}
		
		if len(errors) > 0 {
			red.Printf("\nErrors occurred:\n")
			for _, err := range errors {
				red.Printf("  • %v\n", err)
			}
		}

		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVarP(&stopVerbose, "verbose", "v", false, "Show verbose output including command details")
	stopCmd.Flags().BoolVarP(&stopDryRun, "dry-run", "n", false, "Show commands that would be executed without running them")
}