package mcp

import (
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
	Enabled                bool           `toml:"enabled" json:"enabled" mapstructure:"enabled"`
	LogLevel               string         `toml:"log_level" json:"log_level" mapstructure:"log_level"`
	DefaultTimeout         time.Duration  `toml:"default_timeout" json:"default_timeout" mapstructure:"default_timeout"`
	MaxConcurrentRequests  int            `toml:"max_concurrent_requests" json:"max_concurrent_requests" mapstructure:"max_concurrent_requests"`
	HealthCheckInterval    time.Duration  `toml:"health_check_interval" json:"health_check_interval" mapstructure:"health_check_interval"`
	Servers                []ServerConfig `toml:"servers" json:"servers" mapstructure:"servers"`
}

// DefaultConfig returns default MCP configuration
func DefaultConfig() Config {
	return Config{
		Enabled:                false, // Start disabled by default
		LogLevel:               "info",
		DefaultTimeout:         30 * time.Second,
		MaxConcurrentRequests:  10,
		HealthCheckInterval:    60 * time.Second,
		Servers:                []ServerConfig{},
	}
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