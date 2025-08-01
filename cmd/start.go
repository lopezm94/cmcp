package cmd

import (
	"fmt"

	"cmcp/internal/config"
	"cmcp/internal/mcp"
	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	builder = mcp.NewClaudeCmdBuilder()
	verbose bool
	dryRun  bool
	debug   bool
)

var startCmd = &cobra.Command{
	Use:          "start",
	Short:        "Start MCP servers",
	Long:         `Start one or more MCP servers from your registered servers. Shows only servers that are not currently running.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCPServers) == 0 {
			color.Yellow("No servers configured. Use 'cmcp config open' to add servers.")
			return nil
		}

		var availableServers []string
		var serverLabels []string

		for name := range cfg.MCPServers {
			if !builder.IsRunning(name) {
				availableServers = append(availableServers, name)
				serverLabels = append(serverLabels, name)
			}
		}

		if len(availableServers) == 0 {
			color.Yellow("All registered servers are already running.")
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
			color.Yellow("No servers selected.")
			return nil
		}

		// Handle dry-run mode
		if dryRun {
			yellow := color.New(color.FgYellow)
			yellow.Println("Would execute the following commands:")
			fmt.Println()

			for _, serverName := range selectedServers {
				selectedServer, _ := cfg.FindServer(serverName)

				// Use appropriate command based on whether server has env vars
				if len(selectedServer.Env) > 0 {
					fmt.Printf("$ claude mcp add-json %s ", serverName)
					builder.PrintPrettyJSONPublic(selectedServer)
					fmt.Println() // Extra line after pretty JSON
				} else {
					command := builder.BuildStartCommand(serverName, selectedServer)
					fmt.Printf("$ %s\n", command)
				}
			}
			return nil
		}

		// Start each selected server
		var errors []error
		var started []string
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)

		for _, serverName := range selectedServers {
			selectedServer, _ := cfg.FindServer(serverName)
			cyan.Printf("Starting server '%s'...\n", serverName)

			if err := builder.StartServer(serverName, selectedServer, verbose || debug); err != nil {
				if debug {
					// Show full error with diagnostics
					red.Printf("✗ Server '%s' diagnostics:\n%v\n", serverName, err)
				} else {
					// Show concise error
					red.Printf("✗ Failed to start server '%s': %v\n", serverName, err)
				}
				errors = append(errors, fmt.Errorf("%s", serverName))
			} else {
				started = append(started, serverName)
				green.Printf("✓ Successfully started server '%s'\n", serverName)
			}
		}

		if len(started) > 0 {
			fmt.Printf("\nStarted %d server(s): %v\n", len(started), started)
		}

		if len(errors) > 0 {
			red.Printf("\nFailed to start %d server(s):\n", len(errors))
			for _, err := range errors {
				red.Printf("  • %v\n", err)
			}
		}

		return nil
	},
}

func init() {
	startCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output including command details")
	startCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show commands that would be executed without running them")
	startCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Show detailed diagnostic information for failures")
}

