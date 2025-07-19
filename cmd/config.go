package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"cmcp/internal/config"
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
			fmt.Println("No servers configured")
			return nil
		}

		serverNames := cfg.GetServerNames()

		prompt := promptui.Select{
			Label: "Select server to remove",
			Items: serverNames,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}

		serverName := serverNames[idx]

		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to remove '%s'", serverName),
			IsConfirm: true,
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			return nil
		}

		if err := cfg.RemoveServer(serverName); err != nil {
			return err
		}

		fmt.Printf("Successfully removed server '%s'\n", serverName)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCPServers) == 0 {
			fmt.Println("No servers configured")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCOMMAND")
		for name, server := range cfg.MCPServers {
			cmd := server.Command
			if len(server.Args) > 0 {
				cmd = fmt.Sprintf("%s %s", server.Command, strings.Join(server.Args, " "))
			}
			fmt.Fprintf(w, "%s\t%s\n", name, cmd)
		}
		w.Flush()

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

		fmt.Printf("Opening config file...\n")
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

		fmt.Printf("Config file reformatted successfully.\n")
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

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRmCmd)
	configCmd.AddCommand(configOpenCmd)
}