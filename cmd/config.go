package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"cmcp/internal/config"
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
	Use:   "rm",
	Short: "Remove an MCP server from configuration",
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
		cyan := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		gray := color.New(color.FgHiBlack).SprintFunc()

		// Get server names and check which are running
		serverNames := cfg.GetServerNames()
		runningServers := make(map[string]bool)
		for _, name := range serverNames {
			if builder.IsRunning(name) {
				runningServers[name] = true
			}
		}

		// Create colored items for the select prompt
		items := make([]string, len(serverNames))
		for i, name := range serverNames {
			if runningServers[name] {
				items[i] = fmt.Sprintf("%s %s %s", green("●"), name, gray("(running)"))
			} else {
				items[i] = fmt.Sprintf("%s %s", gray("○"), name)
			}
		}

		prompt := promptui.Select{
			Label: cyan("Select server to remove"),
			Items: items,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}

		serverName := serverNames[idx]

		// Show warning if server is running
		if runningServers[serverName] {
			fmt.Printf("\n%s %s\n", red("⚠"), red("Warning: This server is currently running"))
			fmt.Printf("%s\n\n", gray("It will be stopped if you proceed"))
		}

		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to remove %s", cyan(serverName)),
			IsConfirm: true,
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			return nil
		}

		// Stop server if running
		if runningServers[serverName] {
			fmt.Printf("%s %s\n", gray("→"), gray("Stopping server..."))
			builder.StopServer(serverName, false)
		}

		if err := cfg.RemoveServer(serverName); err != nil {
			return err
		}

		fmt.Printf("\n%s %s\n", green("✓"), green(fmt.Sprintf("Successfully removed server '%s'", serverName)))
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
		fmt.Println("\n")

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
		_, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load/create config: %w", err)
		}

		color.Cyan("Opening config file...\n")
		if err := openInEditor(configPath, ""); err != nil {
			return err
		}

		// After editing, reload and reformat the JSON file
		cfg, err := config.Load()
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
