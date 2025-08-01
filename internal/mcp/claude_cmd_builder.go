package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"cmcp/internal/config"
	"github.com/fatih/color"
)

type ClaudeCmdBuilder struct {
	// Builder for Claude CLI commands
}

func NewClaudeCmdBuilder() *ClaudeCmdBuilder {
	return &ClaudeCmdBuilder{}
}

// findClaude returns the claude command path
func findClaude() string {
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}
	return "claude" // fallback
}

func (b *ClaudeCmdBuilder) StartServer(name string, server *config.MCPServer, verbose bool) error {
	var args []string
	var commandStr string

	// Decide whether to use add-json or regular add
	useAddJSON := len(server.Env) > 0

	if useAddJSON {
		// Use add-json for servers with environment variables
		args = b.buildStartArgsJSON(name, server)

		// Show command if verbose with pretty-printed JSON
		if verbose {
			// Print the command prefix and JSON separately to avoid color code issues
			fmt.Printf("  Command: claude mcp add-json %s ", name)
			b.printPrettyJSON(server)
		}
	} else {
		// Use regular add for simple servers
		args = b.buildStartArgs(name, server)

		// Show command if verbose
		if verbose {
			commandStr = b.BuildStartCommand(name, server)
			fmt.Printf("  Command: %s\n", commandStr)
		}
	}

	// Execute claude mcp add/add-json
	cmd := exec.Command(findClaude(), args...)

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Handle output based on verbose flag and error state
	if err != nil {
		// On error, show the full command and stderr (with masked values)
		if !verbose {
			if useAddJSON {
				commandStr = b.BuildStartCommandJSON(name, server, false)
			} else {
				commandStr = b.BuildStartCommand(name, server)
			}
			fmt.Printf("  Command failed: %s\n", commandStr)
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(os.Stderr, "%s", stderr.String())
		}
		return fmt.Errorf("failed to add server '%s' to Claude", name)
	}

	// In verbose mode, parse and show only relevant info
	if verbose && stdout.Len() > 0 {
		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Skip the duplicate "Added stdio MCP server..." line
			if strings.Contains(line, "Added stdio MCP server") {
				continue
			}
			// Show file modifications with indentation
			if strings.Contains(line, "File modified:") {
				fmt.Printf("  %s\n", line)
			} else {
				// Show other output as-is
				fmt.Println(line)
			}
		}
	}

	// Verify server started successfully with diagnostics
	if err := b.VerifyServerStartedWithDiagnostics(name, server); err != nil {
		return err
	}

	return nil
}

// VerifyServerStarted checks if a server is actually running after being added
func (b *ClaudeCmdBuilder) VerifyServerStarted(name string) error {
	// Give the server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Try up to 3 times with increasing delays
	for attempt := 0; attempt < 3; attempt++ {
		// Run claude mcp list and check if server is connected
		cmd := exec.Command(findClaude(), "mcp", "list")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list servers: %w", err)
		}

		// Parse the output to check server status
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for the server in the output
			if strings.Contains(line, name+":") {
				// Check if it shows as connected (✓) or failed (✗)
				if strings.Contains(line, "✗") || strings.Contains(line, "Failed") {
					return fmt.Errorf("failed to connect")
				}
				if strings.Contains(line, "✓") || strings.Contains(line, "Connected") {
					return nil // Server is connected
				}
			}
		}

		// If not found or not connected yet, wait before retrying
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	// After retries, assume failure
	return fmt.Errorf("failed to connect after 3 attempts")
}

// VerifyServerStartedWithDiagnostics checks if a server is running and provides diagnostics on failure
func (b *ClaudeCmdBuilder) VerifyServerStartedWithDiagnostics(name string, server *config.MCPServer) error {
	err := b.VerifyServerStarted(name)
	if err == nil {
		return nil // Server started successfully
	}

	// Get diagnostic information
	diag, _ := GetServerDiagnostics(name, server.Command, server.Args)
	if diag != nil {
		diagInfo := FormatDiagnostics(diag)
		if diagInfo != "" {
			// Return just the diagnostic info, not the original error
			return fmt.Errorf("%s", strings.TrimSpace(diagInfo))
		}
	}

	return err
}

func (b *ClaudeCmdBuilder) StopServer(name string, verbose bool) error {
	// First check if server exists in Claude
	if !b.IsRunning(name) {
		return fmt.Errorf("server '%s' is not registered in Claude", name)
	}

	// Build the command
	commandStr := b.BuildStopCommand(name)
	args := []string{"mcp", "remove", name}

	// Show command if verbose
	if verbose {
		fmt.Printf("  Command: %s\n", commandStr)
	}

	// Execute claude mcp remove
	cmd := exec.Command(findClaude(), args...)

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Handle output based on verbose flag and error state
	if err != nil {
		// On error, show the full command and stderr
		if !verbose {
			fmt.Printf("  Command failed: %s\n", commandStr)
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(os.Stderr, "%s", stderr.String())
		}
		return fmt.Errorf("failed to remove server '%s' from Claude", name)
	}

	// In verbose mode, parse and show only relevant info
	if verbose && stdout.Len() > 0 {
		output := stdout.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			// Skip the duplicate "Removed MCP server..." line
			if strings.Contains(line, "Removed MCP server") {
				continue
			}
			// Show file modifications with indentation
			if strings.Contains(line, "File modified:") {
				fmt.Printf("  %s\n", line)
			} else {
				// Show other output as-is
				fmt.Println(line)
			}
		}
	}

	return nil
}

func (b *ClaudeCmdBuilder) GetRunningServers() []string {
	// This method is no longer used since we delegate to claude mcp list
	// Keeping for backward compatibility but returns empty
	return []string{}
}

func (b *ClaudeCmdBuilder) StopAllServers() error {
	// Get list of servers from config to remove them all
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var errors []error
	for name := range cfg.MCPServers {
		// Check if this server is in Claude before trying to remove
		if b.IsRunning(name) {
			// Use StopServer with verbose=false for reset command
			if err := b.StopServer(name, false); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errors)
	}
	return nil
}

func (b *ClaudeCmdBuilder) IsRunning(name string) bool {
	// Check if server is registered in Claude by running claude mcp get
	cmd := exec.Command(findClaude(), "mcp", "get", name)
	// Suppress output
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	// If the command succeeds, the server exists in Claude
	return err == nil
}

// buildStartArgs constructs the arguments for starting a server
func (b *ClaudeCmdBuilder) buildStartArgs(name string, server *config.MCPServer) []string {
	// Build the claude mcp add command
	args := []string{"mcp", "add", name}

	// Add environment variables as options
	if server.Env != nil {
		for k, v := range server.Env {
			args = append(args, "--env", fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Add the command and its arguments
	// Use -- to separate claude options from server command args
	args = append(args, "--", server.Command)
	args = append(args, server.Args...)

	return args
}

// BuildStartCommand constructs the command to start a server without executing it
func (b *ClaudeCmdBuilder) BuildStartCommand(name string, server *config.MCPServer) string {
	args := b.buildStartArgs(name, server)
	// Mask sensitive values in args
	maskedArgs := MaskSensitiveArgs(args)
	return fmt.Sprintf("claude %s", strings.Join(maskedArgs, " "))
}

// BuildStopCommand constructs the command to stop a server without executing it
func (b *ClaudeCmdBuilder) BuildStopCommand(name string) string {
	return fmt.Sprintf("claude mcp remove %s", name)
}

// BuildListCommand constructs the command to list servers without executing it
func (b *ClaudeCmdBuilder) BuildListCommand() string {
	return "claude mcp list"
}

// BuildResetCommands constructs the commands to remove multiple servers without executing them
func (b *ClaudeCmdBuilder) BuildResetCommands(serverNames []string) []string {
	var commands []string
	for _, name := range serverNames {
		commands = append(commands, b.BuildStopCommand(name))
	}
	return commands
}

// buildStartArgsJSON constructs the arguments for starting a server using add-json
func (b *ClaudeCmdBuilder) buildStartArgsJSON(name string, server *config.MCPServer) []string {
	// Create the JSON structure
	serverJSON := map[string]interface{}{
		"command": server.Command,
		"args":    server.Args,
	}

	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}

	// Marshal to JSON
	jsonData, _ := json.Marshal(serverJSON)

	return []string{"mcp", "add-json", name, string(jsonData)}
}

// BuildStartCommandJSON constructs the add-json command for display
func (b *ClaudeCmdBuilder) BuildStartCommandJSON(name string, server *config.MCPServer, pretty bool) string {
	if pretty {
		// Pretty print with colors and bash variables
		return b.buildPrettyJSONCommand(name, server)
	}

	// Regular JSON with masked values
	serverJSON := map[string]interface{}{
		"command": server.Command,
		"args":    server.Args,
	}

	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}

	jsonData, _ := json.Marshal(serverJSON)
	maskedJSON, _ := MaskSensitiveJSON(jsonData)

	return fmt.Sprintf("claude mcp add-json %s '%s'", name, string(maskedJSON))
}

// buildPrettyJSONCommand creates a colored, pretty-printed JSON command
func (b *ClaudeCmdBuilder) buildPrettyJSONCommand(name string, server *config.MCPServer) string {
	// Create the JSON structure
	serverJSON := map[string]interface{}{
		"command": server.Command,
		"args":    server.Args,
	}

	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}

	// Marshal to JSON for pretty printing
	jsonData, _ := json.Marshal(serverJSON)
	prettyJSON, _ := MaskSensitiveJSONPretty(jsonData, "  ")

	// Apply colors
	lines := strings.Split(prettyJSON, "\n")
	coloredLines := make([]string, len(lines))

	blue := color.New(color.FgBlue).SprintFunc()
	// green := color.New(color.FgGreen).SprintFunc() // Reserved for future use
	yellow := color.New(color.FgYellow).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	for i, line := range lines {
		// Color the JSON structure
		colored := line

		// Color property names (e.g., "command":)
		colored = strings.ReplaceAll(colored, `"command":`, blue(`"command"`)+gray(":"))
		colored = strings.ReplaceAll(colored, `"args":`, blue(`"args"`)+gray(":"))
		colored = strings.ReplaceAll(colored, `"env":`, blue(`"env"`)+gray(":"))

		// Color environment variable keys
		for key := range server.Env {
			colored = strings.ReplaceAll(colored, `"`+key+`":`, blue(`"`+key+`"`)+gray(":"))
		}

		// Color bash variables (e.g., $GITHUB_TOKEN)
		// bashVarRegex := `\$[A-Z_]+[A-Z0-9_]*` // Reserved for regex-based coloring
		colored = strings.ReplaceAll(colored, "$", yellow("$"))

		// Color string values (but not bash variables)
		// This is a bit tricky, so we'll keep it simple for now

		// Color structural elements
		colored = strings.ReplaceAll(colored, "{", gray("{"))
		colored = strings.ReplaceAll(colored, "}", gray("}"))
		colored = strings.ReplaceAll(colored, "[", gray("["))
		colored = strings.ReplaceAll(colored, "]", gray("]"))
		colored = strings.ReplaceAll(colored, ",", gray(","))

		coloredLines[i] = colored
	}

	// Join the colored lines
	coloredJSON := strings.Join(coloredLines, "\n")

	return fmt.Sprintf("claude mcp add-json %s '%s'", name, coloredJSON)
}

// PrintPrettyJSONPublic prints the colored JSON configuration to stdout (public version)
func (b *ClaudeCmdBuilder) PrintPrettyJSONPublic(server *config.MCPServer) {
	b.printPrettyJSON(server)
}

// printPrettyJSON prints the colored JSON configuration to stdout
func (b *ClaudeCmdBuilder) printPrettyJSON(server *config.MCPServer) {
	// Create the JSON structure
	serverJSON := map[string]interface{}{
		"command": server.Command,
		"args":    server.Args,
	}

	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}

	// Marshal to JSON for pretty printing
	jsonData, _ := json.Marshal(serverJSON)
	prettyJSON, _ := MaskSensitiveJSONPretty(jsonData, "  ")

	// Apply colors
	lines := strings.Split(prettyJSON, "\n")

	blue := color.New(color.FgBlue).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	for _, line := range lines {
		// Color the JSON structure
		colored := line

		// Color property names (e.g., "command":)
		colored = strings.ReplaceAll(colored, `"command":`, blue(`"command"`)+gray(":"))
		colored = strings.ReplaceAll(colored, `"args":`, blue(`"args"`)+gray(":"))
		colored = strings.ReplaceAll(colored, `"env":`, blue(`"env"`)+gray(":"))

		// Color environment variable keys
		for key := range server.Env {
			colored = strings.ReplaceAll(colored, `"`+key+`":`, blue(`"`+key+`"`)+gray(":"))
		}

		// Color bash variables (e.g., $GITHUB_TOKEN)
		colored = strings.ReplaceAll(colored, "$", yellow("$"))

		// Color structural elements
		colored = strings.ReplaceAll(colored, "{", gray("{"))
		colored = strings.ReplaceAll(colored, "}", gray("}"))
		colored = strings.ReplaceAll(colored, "[", gray("["))
		colored = strings.ReplaceAll(colored, "]", gray("]"))
		colored = strings.ReplaceAll(colored, ",", gray(","))

		fmt.Println(colored)
	}
}

