package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"cmcp/internal/config"
	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage MCP server configuration",
	Long:  `List, remove, or edit MCP servers in your configuration.`,
}

var configRmCmd = &cobra.Command{
	Use:   "rm [server-name...]",
	Short: "Remove MCP servers from configuration",
	Long:  `Remove one or more MCP servers from configuration.
You can specify server names as arguments or run without arguments for interactive selection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCPServers) == 0 {
			yellow := color.New(color.FgYellow).SprintFunc()
			gray := color.New(color.FgHiBlack).SprintFunc()
			fmt.Printf("%s\n", yellow("No servers configured"))
			fmt.Printf("%s %s\n", gray("→"), gray("Use 'cmcp config open' to add servers"))
			return nil
		}

		// Color functions
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		gray := color.New(color.FgHiBlack)
		yellow := color.New(color.FgYellow)

		var selectedServers []string
		runningServers := make(map[string]bool)

		// Check which servers are running
		for name := range cfg.MCPServers {
			if builder.IsRunning(name) {
				runningServers[name] = true
			}
		}

		// If server names are provided as arguments, use those (non-interactive mode)
		if len(args) > 0 {
			for _, serverName := range args {
				// Check if server exists in config
				if _, exists := cfg.MCPServers[serverName]; !exists {
					return fmt.Errorf("server '%s' not found in configuration", serverName)
				}
				selectedServers = append(selectedServers, serverName)
			}
		} else {
			// Interactive mode - multi-select
			serverNames := cfg.GetServerNames()
			
			if len(serverNames) == 0 {
				yellow.Println("No servers to remove.")
				return nil
			}

			// Create labeled options showing running status
			var options []string
			for _, name := range serverNames {
				if runningServers[name] {
					options = append(options, fmt.Sprintf("%s (running)", name))
				} else {
					options = append(options, name)
				}
			}

			prompt := &survey.MultiSelect{
				Message: "Select servers to remove (use space to select, enter to confirm):",
				Options: options,
			}

			var selectedOptions []string
			err = survey.AskOne(prompt, &selectedOptions, survey.WithPageSize(10))
			if err != nil {
				return err
			}

			// Extract server names from selected options (remove " (running)" suffix)
			for _, option := range selectedOptions {
				serverName := strings.TrimSuffix(option, " (running)")
				selectedServers = append(selectedServers, serverName)
			}
		}

		if len(selectedServers) == 0 {
			yellow.Println("No servers selected.")
			return nil
		}

		// Show what will be removed
		fmt.Println()
		cyan.Println("The following servers will be removed:")
		for _, name := range selectedServers {
			if runningServers[name] {
				fmt.Printf("  • %s %s\n", name, gray.Sprint("(will be stopped)"))
			} else {
				fmt.Printf("  • %s\n", name)
			}
		}
		fmt.Println()

		// Confirm removal
		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to remove %d server(s)", len(selectedServers)),
			IsConfirm: true,
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			return nil
		}

		// Remove each selected server
		var removed []string
		var errors []error

		for _, serverName := range selectedServers {
			// Stop server if running
			if runningServers[serverName] {
				cyan.Printf("Stopping server '%s'...\n", serverName)
				if err := builder.StopServer(serverName, false); err != nil {
					red.Printf("Warning: Failed to stop server '%s': %v\n", serverName, err)
					// Continue with removal anyway
				}
			}

			if err := cfg.RemoveServer(serverName); err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", serverName, err))
			} else {
				removed = append(removed, serverName)
			}
		}

		// Show results
		fmt.Println()
		if len(removed) > 0 {
			green.Printf("✓ Successfully removed %d server(s): %v\n", len(removed), removed)
		}

		if len(errors) > 0 {
			red.Printf("✗ Failed to remove %d server(s):\n", len(errors))
			for _, err := range errors {
				red.Printf("  • %v\n", err)
			}
			return fmt.Errorf("some servers could not be removed")
		}

		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCPServers) == 0 {
			color.Yellow("No servers configured")
			return nil
		}

		// Color functions
		blue := color.New(color.FgBlue).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		gray := color.New(color.FgHiBlack).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()

		// Get server names and check which are running
		runningServers := make(map[string]bool)
		for name := range cfg.MCPServers {
			if builder.IsRunning(name) {
				runningServers[name] = true
			}
		}

		// Show summary first
		runningCount := len(runningServers)
		totalCount := len(cfg.MCPServers)
		
		fmt.Printf("%s %s", bold(fmt.Sprintf("%d", totalCount)), gray("server(s) configured"))
		if runningCount > 0 {
			fmt.Printf(" • %s %s", green(fmt.Sprintf("%d", runningCount)), gray("running"))
		}
		fmt.Println()
		fmt.Println()

		// List servers
		for name, server := range cfg.MCPServers {
			// Status indicator and name
			if runningServers[name] {
				fmt.Printf("%s %s\n", green("●"), bold(name))
			} else {
				fmt.Printf("%s %s\n", gray("○"), bold(name))
			}

			// Command on the next line with indentation
			fmt.Printf("  %s", blue(server.Command))
			if len(server.Args) > 0 {
				fmt.Printf(" %s", strings.Join(server.Args, " "))
			}
			fmt.Println()

			// Environment variables if any
			if len(server.Env) > 0 {
				fmt.Printf("  %s", gray("env:"))
				for i, key := range getSortedKeys(server.Env) {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf(" %s", yellow(key))
				}
				fmt.Println()
			}
			
			fmt.Println() // Empty line between servers
		}

		return nil
	},
}

var configOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the config file in an editor",
	Long:  `Open the configuration file in nano editor. Optionally select a specific server to jump to.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}

		// Ensure config file exists by loading it
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// If config doesn't exist yet, create it
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to create config: %w", err)
			}
		}

		color.Cyan("Opening config file...\n")
		if err := openInEditor(configPath, ""); err != nil {
			return err
		}

		// After editing, reload and reformat the JSON file
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to reload config after editing: %w", err)
		}

		// Save the config to ensure proper formatting
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to reformat config: %w", err)
		}

		color.Green("Config file reformatted successfully.\n")
		return nil
	},
}

func openInEditor(configPath, serverName string) error {
	// Check if nano is available, fallback to other editors
	var editorCmd *exec.Cmd

	if _, err := exec.LookPath("nano"); err == nil {
		if serverName != "" {
			// Search for the server name in nano
			editorCmd = exec.Command("nano", "+/"+serverName, configPath)
		} else {
			editorCmd = exec.Command("nano", configPath)
		}
	} else if editor := os.Getenv("EDITOR"); editor != "" {
		editorCmd = exec.Command(editor, configPath)
	} else if _, err := exec.LookPath("vim"); err == nil {
		if serverName != "" {
			editorCmd = exec.Command("vim", "+/"+serverName, configPath)
		} else {
			editorCmd = exec.Command("vim", configPath)
		}
	} else if _, err := exec.LookPath("vi"); err == nil {
		editorCmd = exec.Command("vi", configPath)
	} else {
		return fmt.Errorf("no suitable editor found. Please install nano or set $EDITOR environment variable")
	}

	// Set up the editor to use the current terminal
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

func getSortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRmCmd)
	configCmd.AddCommand(configOpenCmd)
}
