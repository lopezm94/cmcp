package cmd

import (
	"fmt"
	"os"
	"strings"

	"cmcp/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	onlineDryRun bool
	onlineClear  bool
	onlineClean  bool
)

var onlineCmd = &cobra.Command{
	Use:   "online",
	Short: "Show currently running MCP servers",
	Long:  `Display a list of all MCP servers that are currently running in Claude for this project.
	
Use --clear to remove servers from Claude that are not in your cmcp config.
Use --clean to remove servers that are failing to connect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle dry-run mode for list command only
		if onlineDryRun && !onlineClear && !onlineClean {
			yellow := color.New(color.FgYellow)
			yellow.Println("Would execute the following command:")
			fmt.Println()
			fmt.Printf("$ %s\n", builder.BuildListCommand())
			return nil
		}

		// Load config to check which servers are in our config
		cfg, err := config.Load()
		if err != nil {
			// Continue even if config load fails - we can still show Claude servers
			cfg = &config.Config{MCPServers: make(map[string]config.MCPServer)}
		}

		// Get server statuses from Claude
		servers, err := builder.GetServerStatuses(cfg)
		if err != nil {
			// Check if it's the "no servers" case
			if strings.Contains(err.Error(), "No MCP servers configured") {
				color.Yellow("No servers are currently running in Claude for this project.")
				fmt.Println("Use 'cmcp start' to start a server.")
				return nil
			}
			return fmt.Errorf("failed to get server statuses: %w", err)
		}

		if len(servers) == 0 {
			// Special handling for --clean flag when no servers are running
			if onlineClean {
				color.Green("✓ No failed servers to clean.")
				fmt.Println()
				color.Cyan("Note: --clean only removes servers that are failing to connect.")
				return nil
			}
			// Special handling for --clear flag when no servers are running  
			if onlineClear {
				color.Green("✓ No orphaned servers to clear.")
				fmt.Println()
				color.Cyan("Note: --clear only removes servers that are NOT in your cmcp config.")
				return nil
			}
			color.Yellow("No servers are currently running in Claude for this project.")
			fmt.Println("Use 'cmcp start' to start a server.")
			return nil
		}

		// Track servers not in config
		var orphanedServers []string
		for _, server := range servers {
			if !server.InConfig {
				orphanedServers = append(orphanedServers, server.Name)
			}
		}

		// Handle --clear flag
		if onlineClear {
			if len(orphanedServers) == 0 {
				color.Green("✓ No orphaned servers to clear.")
				fmt.Println()
				color.Cyan("Note: --clear only removes servers that are NOT in your cmcp config.")
				fmt.Println("Failed servers that are in your config will remain until you stop them.")
				return nil
			}

			// Show what will be cleared
			color.Yellow("The following servers will be cleared from Claude:")
			for _, name := range orphanedServers {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()

			if onlineDryRun {
				yellow := color.New(color.FgYellow)
				yellow.Println("Would execute the following commands:")
				for _, name := range orphanedServers {
					fmt.Printf("$ claude mcp remove %s\n", name)
				}
				return nil
			}

			// Clear each orphaned server
			cyan := color.New(color.FgCyan)
			green := color.New(color.FgGreen)
			red := color.New(color.FgRed)
			
			for _, name := range orphanedServers {
				cyan.Printf("Clearing server '%s' from Claude...\n", name)
				if err := builder.StopServer(name, false); err != nil {
					red.Printf("✗ Failed to clear server '%s': %v\n", name, err)
				} else {
					green.Printf("✓ Cleared server '%s'\n", name)
				}
			}
			
			fmt.Println()
			color.Green("✓ Cleanup complete!")
			return nil
		}

		// Handle --clean flag (remove failed servers)
		if onlineClean {
			// Track failed servers
			var failedServers []string
			for _, server := range servers {
				if server.Status == "failed" && server.InConfig {
					failedServers = append(failedServers, server.Name)
				}
			}

			if len(failedServers) == 0 {
				color.Green("✓ No failed servers to clean.")
				fmt.Println()
				color.Cyan("Note: --clean only removes servers that are failing to connect.")
				return nil
			}

			// Show what will be cleaned
			color.Yellow("The following failed servers will be removed from Claude:")
			for _, name := range failedServers {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()

			if onlineDryRun {
				yellow := color.New(color.FgYellow)
				yellow.Println("Would execute the following commands:")
				for _, name := range failedServers {
					fmt.Printf("$ claude mcp remove %s\n", name)
				}
				return nil
			}

			// Clean each failed server
			cyan := color.New(color.FgCyan)
			green := color.New(color.FgGreen)
			red := color.New(color.FgRed)
			
			for _, name := range failedServers {
				cyan.Printf("Removing failed server '%s' from Claude...\n", name)
				if err := builder.StopServer(name, false); err != nil {
					red.Printf("✗ Failed to remove server '%s': %v\n", name, err)
				} else {
					green.Printf("✓ Removed failed server '%s'\n", name)
				}
			}
			
			fmt.Println()
			color.Green("✓ Failed servers cleaned!")
			return nil
		}

		// Normal online display mode
		// Get current directory for context
		cwd, _ := os.Getwd()
		
		// Print header with project context
		fmt.Println()
		color.Cyan("MCP servers running in Claude for this project:")
		grayColor := color.New(color.FgHiBlack)
		grayColor.Printf("Project: %s\n", cwd)
		fmt.Println()

		// Define colors for different statuses
		greenCheck := color.New(color.FgGreen).Sprint("✓")
		redCross := color.New(color.FgRed).Sprint("✗")
		yellowDot := color.New(color.FgYellow).Sprint("•")
		cyanColor := color.New(color.FgCyan)

		// Print servers
		for _, server := range servers {
			// Determine status icon
			statusIcon := yellowDot
			statusText := "Unknown"
			statusColor := color.New(color.FgYellow)
			
			switch server.Status {
			case "connected":
				statusIcon = greenCheck
				statusText = "Connected"
				statusColor = color.New(color.FgGreen)
			case "failed":
				statusIcon = redCross
				statusText = "Failed to connect"
				statusColor = color.New(color.FgRed)
			}

			// Print server info
			if server.InConfig {
				fmt.Printf("%s %s: ", statusIcon, cyanColor.Sprint(server.Name))
			} else {
				fmt.Printf("%s %s: ", statusIcon, color.New(color.FgYellow).Sprint(server.Name))
			}
			
			// Truncate command if too long
			command := server.Command
			if len(command) > 50 {
				command = command[:47] + "..."
			}
			fmt.Printf("%s - %s\n", grayColor.Sprint(command), statusColor.Sprint(statusText))
		}

		// If there are orphaned servers, show how to clear them
		if len(orphanedServers) > 0 {
			fmt.Println()
			yellowWarning := color.New(color.FgYellow)
			yellowWarning.Printf("⚠ Found %d server(s) in Claude that are not in your cmcp config:\n", len(orphanedServers))
			
			for _, name := range orphanedServers {
				fmt.Printf("  - %s\n", name)
			}
			
			fmt.Println()
			fmt.Println("To clear these servers from Claude, run:")
			fmt.Printf("  $ %s\n", color.New(color.FgCyan).Sprint("cmcp online --clear"))
		}

		return nil
	},
}

func init() {
	onlineCmd.Flags().BoolVarP(&onlineDryRun, "dry-run", "n", false, "Show command that would be executed without running it")
	onlineCmd.Flags().BoolVarP(&onlineClear, "clear", "c", false, "Clear orphaned servers (servers in Claude but NOT in your cmcp config)")
	onlineCmd.Flags().BoolVar(&onlineClean, "clean", false, "Remove failed servers from Claude")
}