package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerType represents the type of MCP server connection
type ServerType string

const (
	ServerTypeStdio ServerType = "stdio"
	ServerTypeHTTP  ServerType = "http"
)

// ServerConfig represents the configuration for an MCP server
type ServerConfig struct {
	Name             string            `toml:"name" json:"name" mapstructure:"name"`
	Description      string            `toml:"description" json:"description" mapstructure:"description"`
	Type             ServerType        `toml:"type" json:"type" mapstructure:"type"`                       // "stdio" or "http"
	Command          []string          `toml:"command" json:"command" mapstructure:"command"`               // For stdio type
	Args             []string          `toml:"args" json:"args" mapstructure:"args"`                       // For stdio type
	URL              string            `toml:"url" json:"url" mapstructure:"url"`                           // For http type
	Headers          map[string]string `toml:"headers" json:"headers" mapstructure:"headers"`               // For http type
	WorkingDir       string            `toml:"working_dir" json:"working_dir" mapstructure:"working_dir"`   // For stdio type
	Env              map[string]string `toml:"env" json:"env" mapstructure:"env"`                           // For stdio type
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
	Servers               []string       `toml:"servers" json:"servers" mapstructure:"servers"`                                     // Array of server config file paths
	ServersConfigPath     string         `toml:"servers_config_path" json:"servers_config_path" mapstructure:"servers_config_path"` // Deprecated: use Servers instead
	LoadedServers         []ServerConfig `toml:"-" json:"-" mapstructure:"-"`                                                       // Internal: loaded server configurations
	mu                    sync.Mutex     `toml:"-" json:"-" mapstructure:"-"`                                                       // Protects LoadedServers
}

// DefaultConfig returns default MCP configuration
func DefaultConfig() Config {
	return Config{
		Enabled:               false, // Start disabled by default
		LogLevel:              "info",
		DefaultTimeout:        30 * time.Second,
		MaxConcurrentRequests: 10,
		HealthCheckInterval:   60 * time.Second,
		Servers:               []string{"./mcpServers.json"}, // Default to mcpServers.json in current directory
		ServersConfigPath:     "",                            // Deprecated
		LoadedServers:         []ServerConfig{},
	}
}

// SimpleServerConfig represents a simplified server configuration for JSON files
type SimpleServerConfig struct {
	Type       string            `json:"type,omitempty"`        // "stdio" or "http", defaults to "stdio"
	Command    string            `json:"command,omitempty"`     // For stdio type
	Args       []string          `json:"args,omitempty"`        // For stdio type
	URL        string            `json:"url,omitempty"`         // For http type
	Headers    map[string]string `json:"headers,omitempty"`     // For http type
	WorkingDir string            `json:"working_dir,omitempty"` // For stdio type
	Env        map[string]string `json:"env,omitempty"`         // For stdio type
}

// JSONServersConfig represents the root structure of the JSON MCP servers config file
type JSONServersConfig struct {
	MCPServers map[string]SimpleServerConfig `json:"mcpServers"`
}

// LoadServersFromJSON loads MCP server configurations from JSON files
func (c *Config) LoadServersFromJSON() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear loaded servers
	c.LoadedServers = []ServerConfig{}

	// Load from new Servers array
	for _, serverFile := range c.Servers {
		if err := c.loadServerFile(serverFile); err != nil {
			return fmt.Errorf("failed to load server file %s: %w", serverFile, err)
		}
	}

	// Backward compatibility: also load from ServersConfigPath if set
	if c.ServersConfigPath != "" {
		if err := c.loadServerFile(c.ServersConfigPath); err != nil {
			return fmt.Errorf("failed to load server file %s: %w", c.ServersConfigPath, err)
		}
	}

	return nil
}

// GetLoadedServers returns a copy of loaded servers with thread-safe access
func (c *Config) GetLoadedServers() []ServerConfig {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Return a copy to prevent external modification
	servers := make([]ServerConfig, len(c.LoadedServers))
	copy(servers, c.LoadedServers)
	return servers
}

// loadServerFile loads a single server configuration file
func (c *Config) loadServerFile(serverFile string) error {
	// Resolve path - check current directory first, then ~/.rago/
	configPath := serverFile
	if !filepath.IsAbs(configPath) {
		// Try current directory first
		if _, err := os.Stat(configPath); err != nil {
			// Try ~/.rago/ and ~/.rago/config/ directories
			homeDir, err := os.UserHomeDir()
			if err == nil {
				// 1. ~/.rago/mcpServers.json
				ragoPath := filepath.Join(homeDir, ".rago", configPath)
				if _, err := os.Stat(ragoPath); err == nil {
					configPath = ragoPath
				} else {
					// 2. ~/.rago/config/mcpServers.json
					configPathSub := filepath.Join(homeDir, ".rago", "config", configPath)
					if _, err := os.Stat(configPathSub); err == nil {
						configPath = configPathSub
					}
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
		// Determine server type
		serverType := ServerTypeStdio
		if simpleConfig.Type != "" {
			serverType = ServerType(simpleConfig.Type)
		} else if simpleConfig.URL != "" {
			// Auto-detect HTTP type if URL is provided
			serverType = ServerTypeHTTP
		}

		serverConfig := ServerConfig{
			Name:             name,
			Description:      fmt.Sprintf("MCP server: %s", name),
			Type:             serverType,
			Command:          []string{},
			Args:             simpleConfig.Args,
			URL:              simpleConfig.URL,
			Headers:          simpleConfig.Headers,
			WorkingDir:       simpleConfig.WorkingDir,
			Env:              simpleConfig.Env,
			AutoStart:        true, // Default to auto-start for JSON-configured servers
			RestartOnFailure: true,
			MaxRestarts:      3,
			RestartDelay:     5 * time.Second,
			Capabilities:     []string{}, // Will be discovered at runtime
		}

		// Set command based on type
		if serverType == ServerTypeStdio && simpleConfig.Command != "" {
			serverConfig.Command = []string{simpleConfig.Command}
		}

		// Add to loaded servers list
		c.LoadedServers = append(c.LoadedServers, serverConfig)
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
