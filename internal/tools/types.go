package tools

import (
	"context"
	"time"
)

// Tool represents a function that can be called by the LLM
type Tool interface {
	// Name returns the unique identifier of the tool
	Name() string
	// Description returns a human-readable description of what the tool does
	Description() string
	// Parameters returns the JSON schema for the tool's parameters
	Parameters() ToolParameters
	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
	// Validate checks if the provided arguments are valid for this tool
	Validate(args map[string]interface{}) error
}

// ToolParameters defines the JSON schema for tool parameters
type ToolParameters struct {
	Type       string                   `json:"type"`
	Properties map[string]ToolParameter `json:"properties"`
	Required   []string                 `json:"required"`
}

// ToolParameter defines a single parameter in the tool schema
type ToolParameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ToolInfo provides metadata about a registered tool
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
	Enabled     bool           `json:"enabled"`
	Category    string         `json:"category"`
}

// ExecutionContext provides context information for tool execution
type ExecutionContext struct {
	RequestID   string            `json:"request_id"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ToolCallLog represents a log entry for tool execution
type ToolCallLog struct {
	RequestID   string                 `json:"request_id"`
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	Result      interface{}            `json:"result,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Elapsed     time.Duration          `json:"elapsed"`
	Timestamp   time.Time              `json:"timestamp"`
	ExecutionID string                 `json:"execution_id"`
}

// ToolConfig represents configuration for tools
type ToolConfig struct {
	Enabled        bool                        `toml:"enabled" mapstructure:"enabled"`
	MaxConcurrency int                         `toml:"max_concurrent_calls" mapstructure:"max_concurrent_calls"`
	CallTimeout    time.Duration               `toml:"call_timeout" mapstructure:"call_timeout"`
	SecurityLevel  string                      `toml:"security_level" mapstructure:"security_level"` // strict, normal, permissive
	EnabledTools   []string                    `toml:"enabled_tools" mapstructure:"enabled_tools"`
	BuiltinTools   map[string]BuiltinToolCfg   `toml:"builtin" mapstructure:"builtin"`
	CustomTools    map[string]CustomToolConfig `toml:"custom" mapstructure:"custom"`
	LogLevel       string                      `toml:"log_level" mapstructure:"log_level"`
	RateLimit      RateLimitConfig             `toml:"rate_limit" mapstructure:"rate_limit"`
	Plugins        PluginConfig                `toml:"plugins" mapstructure:"plugins"`
}

// BuiltinToolCfg represents configuration for a built-in tool
type BuiltinToolCfg struct {
	Enabled    bool              `toml:"enabled" mapstructure:"enabled"`
	Parameters map[string]string `toml:"parameters,omitempty" mapstructure:"parameters"`
}

// CustomToolConfig represents configuration for a custom tool
type CustomToolConfig struct {
	Enabled    bool              `toml:"enabled" mapstructure:"enabled"`
	APIKey     string            `toml:"api_key,omitempty" mapstructure:"api_key"`
	BaseURL    string            `toml:"base_url,omitempty" mapstructure:"base_url"`
	Parameters map[string]string `toml:"parameters,omitempty" mapstructure:"parameters"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	CallsPerMinute int `toml:"calls_per_minute" mapstructure:"calls_per_minute"`
	CallsPerHour   int `toml:"calls_per_hour" mapstructure:"calls_per_hour"`
	BurstSize      int `toml:"burst_size" mapstructure:"burst_size"`
}

// DefaultToolConfig returns default configuration for tools
func DefaultToolConfig() ToolConfig {
	return ToolConfig{
		Enabled:        true,
		MaxConcurrency: 3,
		CallTimeout:    30 * time.Second,
		SecurityLevel:  "normal",
		EnabledTools:   []string{"datetime", "rag_search", "document_info", "file_operations", "sql_query", "http_request", "open_url", "web_search"},
		LogLevel:       "info",
		RateLimit: RateLimitConfig{
			CallsPerMinute: 30,
			CallsPerHour:   300,
			BurstSize:      5,
		},
		BuiltinTools: map[string]BuiltinToolCfg{
			"datetime":      {Enabled: true},
			"rag_search":    {Enabled: true},
			"document_info": {Enabled: true},
			"file_operations": {Enabled: true, Parameters: map[string]string{
				"allowed_paths": "./knowledge,./data,./examples",
				"max_file_size": "10485760", // 10MB
			}},
			"sql_query": {Enabled: false, Parameters: map[string]string{
				"allowed_databases": "main:./data/rag.db",
				"max_rows":          "1000",
				"query_timeout":     "30s",
			}},
			"http_request": {Enabled: true, Parameters: map[string]string{
				"timeout":         "30s",
				"max_body_size":   "10485760", // 10MB
				"user_agent":      "RAGO-HTTP-Tool/1.0",
				"follow_redirect": "true",
			}},
			"open_url": {Enabled: true, Parameters: map[string]string{
				"timeout":         "60s",
				"max_content_len": "102400", // 100KB
				"user_agent":      "RAGO-Web-Tool/1.0",
			}},
			"web_search": {Enabled: true, Parameters: map[string]string{
				"max_results":    "10",
				"search_timeout": "60s",
				"user_agent":     "RAGO-Search-Tool/1.0",
			}},
		},
		Plugins: DefaultPluginConfig(),
	}
}
