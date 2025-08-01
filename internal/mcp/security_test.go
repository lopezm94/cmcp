package mcp

import (
	"strings"
	"testing"
)

func TestMaskSensitiveArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "no sensitive data",
			args:     []string{"mcp", "add", "test", "--", "npx", "server"},
			expected: []string{"mcp", "add", "test", "--", "npx", "server"},
		},
		{
			name:     "mask API_KEY",
			args:     []string{"mcp", "add", "test", "--env", "API_KEY=secret123", "--", "python", "server.py"},
			expected: []string{"mcp", "add", "test", "--env", "API_KEY=***", "--", "python", "server.py"},
		},
		{
			name:     "mask multiple sensitive vars",
			args:     []string{"--env", "GITHUB_TOKEN=ghp_abc123", "--env", "PORT=8080", "--env", "SECRET_KEY=xyz789"},
			expected: []string{"--env", "GITHUB_TOKEN=***", "--env", "PORT=8080", "--env", "SECRET_KEY=***"},
		},
		{
			name:     "mask PASSWORD",
			args:     []string{"--env", "DB_PASSWORD=mypass123"},
			expected: []string{"--env", "DB_PASSWORD=***"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveArgs(tt.args)
			if !slicesEqual(result, tt.expected) {
				t.Errorf("MaskSensitiveArgs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "no env section",
			json:     `{"command":"npx","args":["server"]}`,
			expected: `{"args":["server"],"command":"npx"}`,
		},
		{
			name:     "mask token in env",
			json:     `{"command":"docker","args":["run"],"env":{"GITHUB_TOKEN":"ghp_secret123"}}`,
			expected: `{"args":["run"],"command":"docker","env":{"GITHUB_TOKEN":"***"}}`,
		},
		{
			name:     "mask multiple sensitive vars",
			json:     `{"command":"node","env":{"API_KEY":"abc","PORT":"8080","SECRET":"xyz"}}`,
			expected: `{"command":"node","env":{"API_KEY":"***","PORT":"8080","SECRET":"***"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MaskSensitiveJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("MaskSensitiveJSON() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("MaskSensitiveJSON() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestMaskSensitiveJSONPretty(t *testing.T) {
	json := `{"command":"docker","args":["run","-e","GITHUB_TOKEN"],"env":{"GITHUB_TOKEN":"ghp_secret123"}}`

	result, err := MaskSensitiveJSONPretty([]byte(json), "  ")
	if err != nil {
		t.Fatalf("MaskSensitiveJSONPretty() error = %v", err)
	}

	// Check that it's pretty printed
	if !strings.Contains(result, "\n") {
		t.Error("Result should be multi-line")
	}

	// Check that sensitive value is replaced with bash variable
	if !strings.Contains(result, "$GITHUB_TOKEN") {
		t.Error("Should contain $GITHUB_TOKEN bash variable")
	}

	// Check that quotes are removed from bash variable
	if strings.Contains(result, `"$GITHUB_TOKEN"`) {
		t.Error("Bash variable should not be quoted")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"API_KEY", true},
		{"GITHUB_TOKEN", true},
		{"SECRET_PASSWORD", true},
		{"AUTH_CREDENTIAL", true},
		{"PORT", false},
		{"DEBUG", false},
		{"NODE_ENV", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			if result != tt.sensitive {
				t.Errorf("isSensitiveKey(%s) = %v, want %v", tt.key, result, tt.sensitive)
			}
		})
	}
}

// Helper function to compare slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
