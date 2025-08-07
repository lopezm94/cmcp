package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DiagnosticInfo contains detailed information about a server failure
type DiagnosticInfo struct {
	ServerName    string
	Command       string
	Args          []string
	Error         error
	StdErr        string
	StdOut        string
	Suggestions   []string
	HealthCheck   string
}

// GetServerDiagnostics attempts to gather diagnostic information for a failed server
func GetServerDiagnostics(name string, cmd string, args []string) (*DiagnosticInfo, error) {
	diag := &DiagnosticInfo{
		ServerName:  name,
		Command:     cmd,
		Args:        args,
		Suggestions: []string{},
	}

	// First, check if the server exists in Claude's config
	getCmd := exec.Command(findClaude(), "mcp", "get", name)
	_, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		diag.Error = fmt.Errorf("server not found in Claude config: %v", getErr)
		diag.Suggestions = append(diag.Suggestions, "Server may not be properly registered with Claude")
		return diag, nil
	}

	// Parse the health check info from claude mcp list
	listCmd := exec.Command(findClaude(), "mcp", "list")
	listOut, listErr := listCmd.CombinedOutput()
	if listErr == nil {
		lines := strings.Split(string(listOut), "\n")
		for _, line := range lines {
			if strings.Contains(line, name+":") {
				diag.HealthCheck = strings.TrimSpace(line)
				// Just store the health check info, don't add redundant suggestion
				break
			}
		}
	}

	// Try to run the server command directly to get specific error
	if cmd == "docker" {
		diag.Suggestions = append(diag.Suggestions, getDiagnosticsForDocker(args)...)
	} else if cmd == "node" || cmd == "npx" {
		diag.Suggestions = append(diag.Suggestions, getDiagnosticsForNode(cmd, args)...)
	} else if cmd == "python" || cmd == "python3" {
		diag.Suggestions = append(diag.Suggestions, getDiagnosticsForPython(cmd, args)...)
	}

	// Try running the command with a timeout to capture any startup errors
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	testCmd := exec.CommandContext(ctx, cmd, args...)
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	testErr := testCmd.Run()
	diag.StdOut = stdout.String()
	diag.StdErr = stderr.String()

	if testErr != nil {
		diag.Error = testErr
		// Analyze common error patterns
		analyzeDiagnosticErrors(diag)
	}

	return diag, nil
}

// getDiagnosticsForDocker provides Docker-specific diagnostics
func getDiagnosticsForDocker(args []string) []string {
	suggestions := []string{}

	// Check if Docker daemon is running
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		suggestions = append(suggestions, "Docker daemon is not running. Please start Docker Desktop or the Docker service.")
		return suggestions
	}

	// Check if the image exists
	for _, arg := range args {
		if strings.HasPrefix(arg, "ghcr.io/") || strings.Contains(arg, ":") {
			// This looks like an image name
			checkCmd := exec.Command("docker", "image", "inspect", arg)
			if err := checkCmd.Run(); err != nil {
				suggestions = append(suggestions, fmt.Sprintf("Docker image '%s' not found. Try: docker pull %s", arg, arg))
			}
			break
		}
	}

	// Check for environment variable issues
	hasEnvVars := false
	for _, arg := range args {
		if arg == "-e" || strings.HasPrefix(arg, "--env") {
			hasEnvVars = true
			break
		}
	}
	if hasEnvVars {
		suggestions = append(suggestions, "Check that required environment variables are set in your shell")
	}

	return suggestions
}

// getDiagnosticsForNode provides Node.js-specific diagnostics
func getDiagnosticsForNode(cmd string, args []string) []string {
	suggestions := []string{}

	// Check if node/npx is installed
	checkCmd := exec.Command("which", cmd)
	if err := checkCmd.Run(); err != nil {
		suggestions = append(suggestions, fmt.Sprintf("%s not found. Please install Node.js.", cmd))
		return suggestions
	}

	// Check for common node/npm issues
	if len(args) > 0 {
		scriptPath := args[0]
		if strings.HasSuffix(scriptPath, ".js") || strings.HasSuffix(scriptPath, ".mjs") {
			// Check if the script file exists
			if _, err := exec.Command("test", "-f", scriptPath).Output(); err != nil {
				suggestions = append(suggestions, fmt.Sprintf("Script file '%s' not found", scriptPath))
			}
		}
		
		// Check for package.json if it's a local path
		if !strings.HasPrefix(scriptPath, "@") && !strings.Contains(scriptPath, "/") {
			if _, err := exec.Command("test", "-f", "package.json").Output(); err == nil {
				suggestions = append(suggestions, "Run 'npm install' to install dependencies")
			}
		}
	}

	return suggestions
}

// getDiagnosticsForPython provides Python-specific diagnostics
func getDiagnosticsForPython(cmd string, args []string) []string {
	suggestions := []string{}

	// Check if python is installed
	checkCmd := exec.Command("which", cmd)
	if err := checkCmd.Run(); err != nil {
		suggestions = append(suggestions, fmt.Sprintf("%s not found. Please install Python.", cmd))
		return suggestions
	}

	// Check for common python issues
	if len(args) > 0 && strings.HasSuffix(args[0], ".py") {
		// Check if the script file exists
		if _, err := exec.Command("test", "-f", args[0]).Output(); err != nil {
			suggestions = append(suggestions, fmt.Sprintf("Python script '%s' not found", args[0]))
		}
		
		// Check for requirements.txt
		if _, err := exec.Command("test", "-f", "requirements.txt").Output(); err == nil {
			suggestions = append(suggestions, "Run 'pip install -r requirements.txt' to install dependencies")
		}
	}

	return suggestions
}

// analyzeDiagnosticErrors analyzes common error patterns and provides suggestions
func analyzeDiagnosticErrors(diag *DiagnosticInfo) {
	errStr := diag.StdErr + diag.StdOut
	
	// Permission errors
	if strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "Permission denied") {
		diag.Suggestions = append(diag.Suggestions, "Permission denied. Check file permissions or try running with appropriate privileges.")
	}
	
	// Network errors
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "Connection refused") {
		diag.Suggestions = append(diag.Suggestions, "Connection refused. Check if the service is running and accessible.")
	}
	
	// Port binding errors
	if strings.Contains(errStr, "address already in use") || strings.Contains(errStr, "bind: address already in use") {
		diag.Suggestions = append(diag.Suggestions, "Port already in use. Check for conflicting services or change the port.")
	}
	
	// Module/dependency errors
	if strings.Contains(errStr, "ModuleNotFoundError") || strings.Contains(errStr, "Cannot find module") {
		diag.Suggestions = append(diag.Suggestions, "Missing dependencies. Install required packages for your project.")
	}
	
	// Environment variable errors
	if strings.Contains(errStr, "environment variable") || strings.Contains(errStr, "env var") {
		diag.Suggestions = append(diag.Suggestions, "Missing or invalid environment variables. Check your configuration.")
	}
}

// FormatDiagnostics formats diagnostic information for display
func FormatDiagnostics(diag *DiagnosticInfo) string {
	var sb strings.Builder
	
	// Start with a clear indication this is a connection failure
	sb.WriteString("Connection failed\n")
	
	if diag.HealthCheck != "" {
		sb.WriteString(fmt.Sprintf("\n\033[1;33mHealth check output:\033[0m\n  %s\n", diag.HealthCheck))
	}
	
	if diag.StdErr != "" {
		// Mask sensitive information before displaying
		maskedErr := maskSensitiveOutput(diag.StdErr)
		sb.WriteString(fmt.Sprintf("\n\033[1;31mServer error:\033[0m\n%s\n", maskedErr))
	} else if diag.Error != nil {
		sb.WriteString(fmt.Sprintf("\n\033[1;31mError:\033[0m %v\n", diag.Error))
	}
	
	if len(diag.Suggestions) > 0 {
		sb.WriteString("\n\033[1;35mPossible solutions:\033[0m\n")
		for i, suggestion := range diag.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
	}
	
	return sb.String()
}

// FormatDiagnosticsWithDebugLog formats diagnostic information with optional debug log path
func FormatDiagnosticsWithDebugLog(diag *DiagnosticInfo, debugLogPath string) string {
	var sb strings.Builder
	
	// Start with a clear indication this is a connection failure
	sb.WriteString("Connection failed\n")
	
	if diag.HealthCheck != "" {
		sb.WriteString(fmt.Sprintf("\n\033[1;33mHealth check output:\033[0m\n  %s\n", diag.HealthCheck))
	}
	
	if diag.StdErr != "" {
		// Mask sensitive information before displaying
		maskedErr := maskSensitiveOutput(diag.StdErr)
		sb.WriteString(fmt.Sprintf("\n\033[1;31mServer error:\033[0m\n%s\n", maskedErr))
	} else if diag.Error != nil {
		sb.WriteString(fmt.Sprintf("\n\033[1;31mError:\033[0m %v\n", diag.Error))
	}
	
	if len(diag.Suggestions) > 0 {
		sb.WriteString("\n\033[1;35mPossible solutions:\033[0m\n")
		for i, suggestion := range diag.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
	}
	
	// Add debug log path as a separate, clearly marked section
	if debugLogPath != "" {
		sb.WriteString(fmt.Sprintf("\n\033[0;36mℹ Debug log saved to:\033[0m\n  %s\n", debugLogPath))
		sb.WriteString("\033[0;90m  View this file for detailed connection diagnostics and Claude CLI debug output\033[0m\n")
	}
	
	return sb.String()
}

// maskSensitiveOutput masks sensitive information in output
func maskSensitiveOutput(output string) string {
	// This is a simple implementation - in practice, you'd want to be more sophisticated
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		upperLine := strings.ToUpper(line)
		for _, pattern := range sensitivePatterns {
			patternIdx := strings.Index(upperLine, pattern)
			if patternIdx >= 0 {
				// Find the separator after the pattern
				remainingLine := line[patternIdx:]
				if idx := strings.IndexAny(remainingLine, "=:"); idx >= 0 {
					// Mask after the pattern's separator
					actualIdx := patternIdx + idx
					lines[i] = line[:actualIdx+1] + " ***"
					break // Only mask once per line
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}