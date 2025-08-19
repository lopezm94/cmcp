package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type MCPServer struct {
	Command string                 `json:"command"`
	Args    []string               `json:"args,omitempty"`
	Env     map[string]string      `json:"env,omitempty"`
	Cwd     string                 `json:"cwd,omitempty"`
	Extra   map[string]interface{} `json:"-"` // Stores any additional fields
}

type Config struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

var configPath string

func init() {
	// Allow override via environment variable for testing
	if envPath := os.Getenv("CMCP_CONFIG_PATH"); envPath != "" {
		configPath = envPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Sprintf("failed to get user home directory: %v", err))
		}
		configPath = filepath.Join(homeDir, ".cmcp", "config.json")
	}
}

func GetConfigPath() (string, error) {
	return configPath, nil
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve unknown fields
func (s *MCPServer) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if cmd, ok := raw["command"].(string); ok {
		s.Command = cmd
		delete(raw, "command")
	}

	if args, ok := raw["args"].([]interface{}); ok {
		s.Args = make([]string, len(args))
		for i, arg := range args {
			if str, ok := arg.(string); ok {
				s.Args[i] = str
			}
		}
		delete(raw, "args")
	}

	if env, ok := raw["env"].(map[string]interface{}); ok {
		s.Env = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				s.Env[k] = str
			}
		}
		delete(raw, "env")
	}

	if cwd, ok := raw["cwd"].(string); ok {
		s.Cwd = cwd
		delete(raw, "cwd")
	}

	// Store any remaining fields in Extra
	if len(raw) > 0 {
		s.Extra = raw
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to include extra fields
func (s MCPServer) MarshalJSON() ([]byte, error) {
	// Start with extra fields if any
	result := make(map[string]interface{})
	for k, v := range s.Extra {
		result[k] = v
	}

	// Add known fields (these will override any duplicates in Extra)
	result["command"] = s.Command
	if len(s.Args) > 0 {
		result["args"] = s.Args
	}
	if len(s.Env) > 0 {
		result["env"] = s.Env
	}
	if s.Cwd != "" {
		result["cwd"] = s.Cwd
	}

	return json.Marshal(result)
}

func Load() (*Config, error) {
	if err := ensureConfigDir(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config without saving - let caller decide what to do
			cfg := &Config{MCPServers: make(map[string]MCPServer)}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServer)
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func ensureConfigDir() error {
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0755)
}

func (c *Config) FindServer(name string) (*MCPServer, bool) {
	server, exists := c.MCPServers[name]
	if exists {
		return &server, true
	}
	return nil, false
}

func (c *Config) AddServer(name string, server MCPServer) error {
	if _, exists := c.MCPServers[name]; exists {
		return fmt.Errorf("server '%s' already exists", name)
	}
	c.MCPServers[name] = server
	return Save(c)
}

func (c *Config) RemoveServer(name string) error {
	if _, exists := c.MCPServers[name]; !exists {
		return fmt.Errorf("server '%s' not found", name)
	}
	delete(c.MCPServers, name)
	return Save(c)
}

func (c *Config) GetServerNames() []string {
	names := make([]string, 0, len(c.MCPServers))
	for name := range c.MCPServers {
		names = append(names, name)
	}
	return names
}

func (c *Config) UpdateServerEnv(name string, env map[string]string) error {
	server, exists := c.MCPServers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if server.Env == nil {
		server.Env = make(map[string]string)
	}

	// Merge environment variables
	for k, v := range env {
		server.Env[k] = v
	}

	c.MCPServers[name] = server
	return Save(c)
}
