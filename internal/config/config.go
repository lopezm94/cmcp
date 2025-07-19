package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type Config struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

var configPath string

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	configPath = filepath.Join(homeDir, ".cmcp", "config.json")
}

func GetConfigPath() (string, error) {
	return configPath, nil
}

func Load() (*Config, error) {
	if err := ensureConfigDir(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{MCPServers: make(map[string]MCPServer)}
			return cfg, Save(cfg)
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