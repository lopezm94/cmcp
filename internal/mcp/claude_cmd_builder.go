package mcp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cmcp/internal/config"
	"github.com/fatih/color"
)

type ClaudeCmdBuilder struct {
	// Builder for Claude CLI commands
}

// ServerStatus represents the status of a server in Claude
type ServerStatus struct {
	Name      string
	Command   string
	Status    string // "connected", "failed", "unknown"
	InConfig  bool
}

func NewClaudeCmdBuilder() *ClaudeCmdBuilder {
	return &ClaudeCmdBuilder{}
}

// createDebugLogFile creates a temp file for debug output and returns the path
func (b *ClaudeCmdBuilder) createDebugLogFile(operation string) (string, error) {
	// Create temp directory for cmcp debug logs
	tempDir := filepath.Join(os.TempDir(), "cmcp-debug")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create debug temp dir: %w", err)
	}

	// Create temp file with timestamp and operation name
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("cmcp-%s-%s.log", operation, timestamp)
	logPath := filepath.Join(tempDir, filename)

	// Create the file
	file, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to create debug log file: %w", err)
	}
	file.Close()

	return logPath, nil
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

	// Create debug log file only if not verbose
	var debugLogPath string
	var debugLogErr error
	if !verbose {
		debugLogPath, debugLogErr = b.createDebugLogFile("start-" + name)
	}

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

	// Always add --debug flag for better error diagnostics
	args = append([]string{args[0], args[1], "--debug"}, args[2:]...)

	// Execute claude mcp add/add-json
	cmd := exec.Command(findClaude(), args...)

	// Capture output or show directly based on verbose flag
	var stdout, stderr strings.Builder
	if verbose {
		// In verbose mode, show output directly
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println()  // Add newline before debug output
	} else {
		// In normal mode, capture output for logging
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()

	// Write debug output to log file only if not verbose
	if !verbose && debugLogErr == nil {
		debugContent := fmt.Sprintf("Command: %s\nExit Code: %v\n\nSTDOUT:\n%s\n\nSTDERR:\n%s\n", 
			strings.Join(args, " "), err, stdout.String(), stderr.String())
		ioutil.WriteFile(debugLogPath, []byte(debugContent), 0644)
	}

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
			
			if stderr.Len() > 0 {
				fmt.Fprintf(os.Stderr, "%s", stderr.String())
			}

			// Include debug log path in error message if available
			errorMsg := fmt.Sprintf("failed to add server '%s' to Claude", name)
			if debugLogErr == nil {
				errorMsg += fmt.Sprintf("\n\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n\033[0;90m  View this file for detailed error information\033[0m", debugLogPath)
			}
			return fmt.Errorf(errorMsg)
		} else {
			// In verbose mode, error was already shown, just return simple error
			return fmt.Errorf("failed to add server '%s' to Claude", name)
		}
	}

	// In non-verbose mode, parse and show only relevant info
	if !verbose && stdout.Len() > 0 {
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
			}
		}
	}

	// Verify server started successfully with diagnostics
	var verifyDebugPath string
	if !verbose && debugLogErr == nil {
		verifyDebugPath = debugLogPath
	}
	if err := b.VerifyServerStartedWithDiagnosticsVerbose(name, server, verbose, verifyDebugPath); err != nil {
		return err
	}

	return nil
}

// VerifyServerStarted checks if a server is actually running after being added
func (b *ClaudeCmdBuilder) VerifyServerStarted(name string) error {
	return b.VerifyServerStartedVerbose(name, false)
}

// VerifyServerStartedVerbose checks if a server is running with optional verbose output
func (b *ClaudeCmdBuilder) VerifyServerStartedVerbose(name string, verbose bool) error {
	// Give the server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Create debug log file only if not verbose
	var debugLogPath string
	var debugLogErr error
	if !verbose {
		debugLogPath, debugLogErr = b.createDebugLogFile("verify-" + name)
	}

	// Try up to 3 times with increasing delays
	for attempt := 0; attempt < 3; attempt++ {
		// Run claude mcp list with debug and check if server is connected
		cmd := exec.Command(findClaude(), "mcp", "list", "--debug")
		
		var output []byte
		var err error
		
		if verbose {
			// In verbose mode, show output directly
			if attempt == 0 {
				fmt.Println("\nVerifying server connection...")
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		} else {
			// In normal mode, capture output for logging
			output, err = cmd.CombinedOutput()
			
			// Log the verification attempt if we have a debug log
			if debugLogErr == nil {
				debugContent := fmt.Sprintf("Verification attempt %d:\nCommand: claude mcp list --debug\nOutput:\n%s\nError: %v\n\n", 
					attempt+1, string(output), err)
				// Append to existing log file
				file, appendErr := os.OpenFile(debugLogPath, os.O_APPEND|os.O_WRONLY, 0644)
				if appendErr == nil {
					file.WriteString(debugContent)
					file.Close()
				}
			}
		}

		if err != nil {
			if !verbose && debugLogErr == nil {
				return fmt.Errorf("failed to list servers: %w\n\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n\033[0;90m  View this file for detailed error information\033[0m", err, debugLogPath)
			}
			return fmt.Errorf("failed to list servers: %w", err)
		}

		// Parse the output to check server status
		// For verbose mode, we need to re-run to capture output for parsing
		if verbose {
			cmd := exec.Command(findClaude(), "mcp", "list")
			output, _ = cmd.Output()
		}
		
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for the server in the output
			if strings.Contains(line, name+":") {
				// Check if it shows as connected (✓) or failed (✗)
				if strings.Contains(line, "✗") || strings.Contains(line, "Failed") {
					errorMsg := "failed to connect"
					if !verbose && debugLogErr == nil {
						errorMsg += fmt.Sprintf("\n\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n\033[0;90m  View this file for detailed connection diagnostics\033[0m", debugLogPath)
					}
					return fmt.Errorf(errorMsg)
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
	errorMsg := "failed to connect after 3 attempts"
	if !verbose && debugLogErr == nil {
		errorMsg += fmt.Sprintf("\n\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n\033[0;90m  View this file for detailed connection diagnostics\033[0m", debugLogPath)
	}
	return fmt.Errorf(errorMsg)
}

// VerifyServerStartedWithDiagnostics checks if a server is running and provides diagnostics on failure
func (b *ClaudeCmdBuilder) VerifyServerStartedWithDiagnostics(name string, server *config.MCPServer) error {
	return b.VerifyServerStartedWithDiagnosticsVerbose(name, server, false, "")
}

// VerifyServerStartedWithDiagnosticsVerbose checks if a server is running and provides diagnostics on failure
func (b *ClaudeCmdBuilder) VerifyServerStartedWithDiagnosticsVerbose(name string, server *config.MCPServer, verbose bool, debugLogPath string) error {
	err := b.VerifyServerStartedVerbose(name, verbose)
	if err == nil {
		return nil // Server started successfully
	}

	// Get diagnostic information
	diag, _ := GetServerDiagnostics(name, server.Command, server.Args)
	if diag != nil {
		var diagInfo string
		if !verbose && debugLogPath != "" {
			// Include debug log path in the diagnostic output
			diagInfo = FormatDiagnosticsWithDebugLog(diag, debugLogPath)
		} else {
			diagInfo = FormatDiagnostics(diag)
		}
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

	// Create debug log file only if not verbose
	var debugLogPath string
	var debugLogErr error
	if !verbose {
		debugLogPath, debugLogErr = b.createDebugLogFile("stop-" + name)
	}

	// Build the command
	commandStr := b.BuildStopCommand(name)
	args := []string{"mcp", "remove", "--debug", name}

	// Show command if verbose
	if verbose {
		fmt.Printf("  Command: %s\n", commandStr)
		fmt.Println()  // Add newline before debug output
	}

	// Execute claude mcp remove
	cmd := exec.Command(findClaude(), args...)

	// Capture output or show directly based on verbose flag
	var stdout, stderr strings.Builder
	if verbose {
		// In verbose mode, show output directly
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// In normal mode, capture output for logging
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()

	// Write debug output to log file only if not verbose
	if !verbose && debugLogErr == nil {
		debugContent := fmt.Sprintf("Command: %s\nExit Code: %v\n\nSTDOUT:\n%s\n\nSTDERR:\n%s\n", 
			strings.Join(args, " "), err, stdout.String(), stderr.String())
		ioutil.WriteFile(debugLogPath, []byte(debugContent), 0644)
	}

	// Handle output based on verbose flag and error state
	if err != nil {
		if !verbose {
			// On error, show the full command and stderr
			fmt.Printf("  Command failed: %s\n", commandStr)
			
			if stderr.Len() > 0 {
				fmt.Fprintf(os.Stderr, "%s", stderr.String())
			}

			// Include debug log path in error message if available
			errorMsg := fmt.Sprintf("failed to remove server '%s' from Claude", name)
			if debugLogErr == nil {
				errorMsg += fmt.Sprintf("\n\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n\033[0;90m  View this file for detailed error information\033[0m", debugLogPath)
			}
			return fmt.Errorf(errorMsg)
		} else {
			// In verbose mode, error was already shown
			return fmt.Errorf("failed to remove server '%s' from Claude", name)
		}
	}

	// In non-verbose mode, parse and show only relevant info
	if !verbose && stdout.Len() > 0 {
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

// IsRunningWithDebugLog checks if server is running and optionally logs to debug file
func (b *ClaudeCmdBuilder) IsRunningWithDebugLog(name string, logDebug bool) (bool, string) {
	if !logDebug {
		return b.IsRunning(name), ""
	}

	// Create debug log file for this check
	debugLogPath, debugLogErr := b.createDebugLogFile("check-" + name)

	// Check if server is registered in Claude by running claude mcp get with debug
	cmd := exec.Command(findClaude(), "mcp", "get", "--debug", name)
	output, err := cmd.CombinedOutput()

	// Log the check if we have a debug log
	if debugLogErr == nil {
		debugContent := fmt.Sprintf("Command: claude mcp get --debug %s\nOutput:\n%s\nError: %v\n", 
			name, string(output), err)
		ioutil.WriteFile(debugLogPath, []byte(debugContent), 0644)
	}

	logPath := ""
	if debugLogErr == nil {
		logPath = debugLogPath
	}

	// If the command succeeds, the server exists in Claude
	return err == nil, logPath
}

// GetServerStatuses parses claude mcp list output and returns server statuses
func (b *ClaudeCmdBuilder) GetServerStatuses(cfg *config.Config) ([]ServerStatus, error) {
	// Execute claude mcp list and capture output
	cmd := exec.Command(findClaude(), "mcp", "list")
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	var servers []ServerStatus
	lines := strings.Split(string(output), "\n")
	
	// Skip the "Checking MCP server health..." line if present
	startIndex := 0
	for i, line := range lines {
		if strings.Contains(line, "Checking MCP server health") {
			startIndex = i + 1
			// Skip empty line after header
			if startIndex < len(lines) && strings.TrimSpace(lines[startIndex]) == "" {
				startIndex++
			}
			break
		}
	}
	
	// Parse each server line
	for i := startIndex; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		// Parse lines like: "test-fail: nonexistent-command --fail - ✗ Failed to connect"
		// or: "github: docker run ... - ✓ Connected"
		parts := strings.Split(line, ": ")
		if len(parts) < 2 {
			continue
		}
		
		serverName := strings.TrimSpace(parts[0])
		remainder := strings.Join(parts[1:], ": ")
		
		// Find the last " - " to separate command from status
		lastDashIndex := strings.LastIndex(remainder, " - ")
		if lastDashIndex == -1 {
			continue
		}
		
		command := strings.TrimSpace(remainder[:lastDashIndex])
		statusPart := strings.TrimSpace(remainder[lastDashIndex+3:])
		
		// Determine status
		status := "unknown"
		if strings.Contains(statusPart, "✓") || strings.Contains(statusPart, "Connected") {
			status = "connected"
		} else if strings.Contains(statusPart, "✗") || strings.Contains(statusPart, "Failed") {
			status = "failed"
		}
		
		// Check if server is in config
		inConfig := false
		if cfg != nil {
			_, inConfig = cfg.MCPServers[serverName]
		}
		
		servers = append(servers, ServerStatus{
			Name:     serverName,
			Command:  command,
			Status:   status,
			InConfig: inConfig,
		})
	}
	
	return servers, nil
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
	// Create the JSON structure starting with any extra fields
	serverJSON := make(map[string]interface{})
	for k, v := range server.Extra {
		serverJSON[k] = v
	}

	// Add known fields (these will override any duplicates in Extra)
	serverJSON["command"] = server.Command
	if len(server.Args) > 0 {
		serverJSON["args"] = server.Args
	}
	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}
	if server.Cwd != "" {
		serverJSON["cwd"] = server.Cwd
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
		colored = strings.ReplaceAll(colored, `"cwd":`, blue(`"cwd"`)+gray(":"))

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
	// Create the JSON structure starting with any extra fields
	serverJSON := make(map[string]interface{})
	for k, v := range server.Extra {
		serverJSON[k] = v
	}

	// Add known fields
	serverJSON["command"] = server.Command
	if len(server.Args) > 0 {
		serverJSON["args"] = server.Args
	}
	if len(server.Env) > 0 {
		serverJSON["env"] = server.Env
	}
	if server.Cwd != "" {
		serverJSON["cwd"] = server.Cwd
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
		colored = strings.ReplaceAll(colored, `"cwd":`, blue(`"cwd"`)+gray(":"))

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

