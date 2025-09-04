package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig represents the configuration for an MCP server
type ServerConfig struct {
	Name             string            `toml:"name" json:"name" mapstructure:"name"`
	Description      string            `toml:"description" json:"description" mapstructure:"description"`
	Command          []string          `toml:"command" json:"command" mapstructure:"command"`
	Args             []string          `toml:"args" json:"args" mapstructure:"args"`
	WorkingDir       string            `toml:"working_dir" json:"working_dir" mapstructure:"working_dir"`
	Env              map[string]string `toml:"env" json:"env" mapstructure:"env"`
	AutoStart        bool              `toml:"auto_start" json:"auto_start" mapstructure:"auto_start"`
	RestartOnFailure bool              `toml:"restart_on_failure" json:"restart_on_failure" mapstructure:"restart_on_failure"`
	MaxRestarts      int               `toml:"max_restarts" json:"max_restarts" mapstructure:"max_restarts"`
	RestartDelay     time.Duration     `toml:"restart_delay" json:"restart_delay" mapstructure:"restart_delay"`
	Capabilities     []string          `toml:"capabilities" json:"capabilities" mapstructure:"capabilities"`
}

// Config represents the overall MCP configuration
type Config struct {
	Enabled               bool           `toml:"enabled" json:"enabled" mapstructure:"enabled"`
	LogLevel              string         `toml:"log_level" json:"log_level" mapstructure:"log_level"`
	DefaultTimeout        time.Duration  `toml:"default_timeout" json:"default_timeout" mapstructure:"default_timeout"`
	MaxConcurrentRequests int            `toml:"max_concurrent_requests" json:"max_concurrent_requests" mapstructure:"max_concurrent_requests"`
	HealthCheckInterval   time.Duration  `toml:"health_check_interval" json:"health_check_interval" mapstructure:"health_check_interval"`
	ServersConfigPath     string         `toml:"servers_config_path" json:"servers_config_path" mapstructure:"servers_config_path"` // Path to external JSON config
	Servers               []ServerConfig `toml:"servers" json:"servers" mapstructure:"servers"`
}

// DefaultConfig returns default MCP configuration
func DefaultConfig() Config {
	return Config{
		Enabled:               false, // Start disabled by default
		LogLevel:              "info",
		DefaultTimeout:        30 * time.Second,
		MaxConcurrentRequests: 10,
		HealthCheckInterval:   60 * time.Second,
		ServersConfigPath:     "", // Empty by default, can be set to "./mcpServers.json" or similar
		Servers:               []ServerConfig{},
	}
}

// SimpleServerConfig represents a simplified server configuration for JSON files
type SimpleServerConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// JSONServersConfig represents the root structure of the JSON MCP servers config file
type JSONServersConfig struct {
	MCPServers map[string]SimpleServerConfig `json:"mcpServers"`
}

// LoadServersFromJSON loads MCP server configurations from a JSON file
func (c *Config) LoadServersFromJSON() error {
	if c.ServersConfigPath == "" {
		return nil // No external config specified
	}

	// Resolve path - check current directory first, then ~/.rago/
	configPath := c.ServersConfigPath
	if !filepath.IsAbs(configPath) {
		// Try current directory first
		if _, err := os.Stat(configPath); err != nil {
			// Try ~/.rago/ directory
			homeDir, err := os.UserHomeDir()
			if err == nil {
				ragoPath := filepath.Join(homeDir, ".rago", configPath)
				if _, err := os.Stat(ragoPath); err == nil {
					configPath = ragoPath
				}
			}
		}
	}

	// Read the JSON file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read MCP servers config file %s: %w", configPath, err)
	}

	// Parse JSON
	var jsonConfig JSONServersConfig
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return fmt.Errorf("failed to parse MCP servers config JSON: %w", err)
	}

	// Convert to ServerConfig and append to existing servers
	for name, simpleConfig := range jsonConfig.MCPServers {
		serverConfig := ServerConfig{
			Name:             name,
			Description:      fmt.Sprintf("MCP server: %s", name),
			Command:          []string{simpleConfig.Command},
			Args:             simpleConfig.Args,
			WorkingDir:       simpleConfig.WorkingDir,
			Env:              simpleConfig.Env,
			AutoStart:        true, // Default to auto-start for JSON-configured servers
			RestartOnFailure: true,
			MaxRestarts:      3,
			RestartDelay:     5 * time.Second,
			Capabilities:     []string{}, // Will be discovered at runtime
		}

		// Add to servers list
		c.Servers = append(c.Servers, serverConfig)
	}

	return nil
}

// Client represents an MCP client connection to a server
type Client struct {
	config    *ServerConfig
	session   *mcp.ClientSession
	tools     map[string]*mcp.Tool
	connected bool
}

// ToolInfo represents information about an MCP tool
type ToolInfo struct {
	ServerName  string                 `json:"server_name"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	LastUsed    time.Time              `json:"last_used"`
	UsageCount  int64                  `json:"usage_count"`
}

// ToolResult represents the result of an MCP tool call
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}
