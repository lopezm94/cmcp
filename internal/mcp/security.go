package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Common sensitive environment variable patterns
var sensitivePatterns = []string{
	"TOKEN",
	"KEY",
	"SECRET",
	"PASSWORD",
	"PAT",
	"CREDENTIAL",
	"AUTH",
}

// Common environment variable mappings for display
var envVarMappings = map[string]string{
	"GITHUB_PERSONAL_ACCESS_TOKEN": "GITHUB_TOKEN",
	"GITHUB_TOKEN":                 "GITHUB_TOKEN",
	"ANTHROPIC_API_KEY":            "ANTHROPIC_KEY",
	"OPENAI_API_KEY":               "OPENAI_KEY",
	"GROQ_API_KEY":                 "GROQ_KEY",
	"GOOGLE_API_KEY":               "GOOGLE_KEY",
	"AWS_ACCESS_KEY_ID":            "AWS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY":        "AWS_SECRET_KEY",
}

// isSensitiveKey checks if an environment variable key contains sensitive patterns
func isSensitiveKey(key string) bool {
	upperKey := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}
	return false
}

// maskValue returns a masked version of a sensitive value
func maskValue(value string) string {
	if len(value) == 0 {
		return ""
	}
	return "***"
}

// getBashVariable returns a bash variable name for a known env var, or a generic one
func getBashVariable(key string) string {
	if bashVar, ok := envVarMappings[key]; ok {
		return "$" + bashVar
	}
	// For unknown vars, create a reasonable bash variable name
	// e.g., MY_API_TOKEN -> $MY_API_TOKEN
	return "$" + key
}

// MaskSensitiveArgs masks sensitive values in command arguments for display
func MaskSensitiveArgs(args []string) []string {
	masked := make([]string, len(args))
	copy(masked, args)

	for i := 0; i < len(masked); i++ {
		// Check for --env KEY=VALUE pattern
		if strings.HasPrefix(masked[i], "--env") && i+1 < len(masked) {
			parts := strings.SplitN(masked[i+1], "=", 2)
			if len(parts) == 2 && isSensitiveKey(parts[0]) {
				masked[i+1] = parts[0] + "=" + maskValue(parts[1])
			}
		}
		// Check for -e KEY=VALUE pattern
		if strings.HasPrefix(masked[i], "-e") && strings.Contains(masked[i], "=") {
			parts := strings.SplitN(masked[i], "=", 2)
			keyPart := strings.TrimPrefix(parts[0], "-e")
			if isSensitiveKey(keyPart) {
				masked[i] = parts[0] + "=" + maskValue(parts[1])
			}
		}
	}

	return masked
}

// MaskSensitiveJSON masks sensitive values in JSON data
func MaskSensitiveJSON(jsonData []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return jsonData, err
	}

	// Mask environment variables
	if envMap, ok := data["env"].(map[string]interface{}); ok {
		maskedEnv := make(map[string]interface{})
		for key, value := range envMap {
			if isSensitiveKey(key) {
				maskedEnv[key] = maskValue(value.(string))
			} else {
				maskedEnv[key] = value
			}
		}
		data["env"] = maskedEnv
	}

	return json.Marshal(data)
}

// MaskSensitiveJSONPretty creates pretty JSON with bash variables for sensitive values
func MaskSensitiveJSONPretty(jsonData []byte, indent string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}

	// Replace sensitive env values with bash variables
	if envMap, ok := data["env"].(map[string]interface{}); ok {
		maskedEnv := make(map[string]interface{})
		for key, value := range envMap {
			if isSensitiveKey(key) {
				// Use bash variable instead of masking
				maskedEnv[key] = getBashVariable(key)
			} else {
				maskedEnv[key] = value
			}
		}
		data["env"] = maskedEnv
	}

	// Marshal with indentation
	prettyJSON, err := json.MarshalIndent(data, "", indent)
	if err != nil {
		return "", err
	}

	// Post-process to remove quotes around bash variables
	result := string(prettyJSON)
	for key := range envVarMappings {
		bashVar := getBashVariable(key)
		// Replace "$VAR" with $VAR (remove quotes)
		result = strings.ReplaceAll(result, `"`+bashVar+`"`, bashVar)
	}
	// Also handle generic bash variables
	re := regexp.MustCompile(`"\$([A-Z_]+[A-Z0-9_]*)"`)
	result = re.ReplaceAllString(result, "$$$1")

	return result, nil
}

