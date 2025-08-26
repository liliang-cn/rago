package tools

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Registry manages the registration and retrieval of tools
type Registry struct {
	tools   map[string]Tool
	mu      sync.RWMutex
	config  *ToolConfig
	limiter *rate.Limiter
	logger  Logger
}

// Logger interface for tool registry logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// DefaultLogger provides a simple logging implementation
type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

// NewRegistry creates a new tool registry with the given configuration
func NewRegistry(config *ToolConfig) *Registry {
	if config == nil {
		defaultConfig := DefaultToolConfig()
		config = &defaultConfig
	}

	// Create rate limiter based on configuration
	var limiter *rate.Limiter
	if config.RateLimit.CallsPerMinute > 0 {
		limiter = rate.NewLimiter(
			rate.Every(time.Minute/time.Duration(config.RateLimit.CallsPerMinute)),
			config.RateLimit.BurstSize,
		)
	}

	registry := &Registry{
		tools:   make(map[string]Tool),
		config:  config,
		limiter: limiter,
		logger:  &DefaultLogger{},
	}

	return registry
}

// SetLogger sets a custom logger for the registry
func (r *Registry) SetLogger(logger Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger = logger
}

// Register adds a new tool to the registry
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if tool already exists
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool with name '%s' already registered", name)
	}

	// Validate tool parameters schema
	if err := r.validateToolParameters(tool.Parameters()); err != nil {
		return fmt.Errorf("invalid tool parameters for '%s': %w", name, err)
	}

	r.tools[name] = tool
	r.logger.Info("Registered tool: %s", name)

	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(r.tools, name)
	r.logger.Info("Unregistered tool: %s", name)

	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns information about all registered tools
func (r *Registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for name, tool := range r.tools {
		info := ToolInfo{
			Name:        name,
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
			Enabled:     r.IsEnabled(name),
			Category:    r.getToolCategory(name),
		}
		tools = append(tools, info)
	}

	return tools
}

// ListEnabled returns information about all enabled tools
func (r *Registry) ListEnabled() []ToolInfo {
	all := r.List()
	var enabled []ToolInfo
	for _, tool := range all {
		if tool.Enabled {
			enabled = append(enabled, tool)
		}
	}
	return enabled
}

// builtinTools defines the list of built-in tools that are enabled by default
var builtinTools = map[string]bool{
	"datetime":        true,
	"sql_query":       true,
	"rag_search":      true,
	"document_count":  true,
	"document_info":   true,
	"stats_query":     true,
	"file_read":       true,
	"file_list":       true,
	"file_write":      true,
	"file_exists":     true,
	"file_stat":       true,
	"file_delete":     true,
	"http_request":    true,
	"web_search":      true,
	"open_url":        true,
}

// IsEnabled checks if a tool is enabled in the configuration
func (r *Registry) IsEnabled(name string) bool {
	if !r.config.Enabled {
		return false
	}

	// Built-in tools are enabled by default, but can be explicitly disabled
	if _, isBuiltin := builtinTools[name]; isBuiltin {
		// Check if explicitly disabled in builtin tool configuration
		if builtinConfig, exists := r.config.BuiltinTools[name]; exists {
			return builtinConfig.Enabled
		}
		// Default enabled for built-in tools
		return true
	}

	// Check if tool is in the enabled list
	if len(r.config.EnabledTools) > 0 {
		for _, enabledTool := range r.config.EnabledTools {
			if enabledTool == name {
				return true
			}
		}
		return false
	}

	// Check custom tool configuration
	if customConfig, exists := r.config.CustomTools[name]; exists {
		return customConfig.Enabled
	}

	// Default to disabled for non-builtin tools
	return false
}

// GetConfig returns the current tool configuration
func (r *Registry) GetConfig() *ToolConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// UpdateConfig updates the tool configuration
func (r *Registry) UpdateConfig(config *ToolConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config
	r.logger.Info("Updated tool configuration")

	return nil
}

// CheckRateLimit checks if the current request is within rate limits
func (r *Registry) CheckRateLimit() error {
	if r.limiter == nil {
		return nil
	}

	if !r.limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// GetToolCount returns the total number of registered tools
func (r *Registry) GetToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// GetEnabledToolCount returns the number of enabled tools
func (r *Registry) GetEnabledToolCount() int {
	count := 0
	for name := range r.tools {
		if r.IsEnabled(name) {
			count++
		}
	}
	return count
}

// validateToolParameters validates the tool parameters schema
func (r *Registry) validateToolParameters(params ToolParameters) error {
	if params.Type != "object" && params.Type != "" {
		return fmt.Errorf("tool parameters type must be 'object' or empty")
	}

	// Validate required fields exist in properties
	for _, required := range params.Required {
		if _, exists := params.Properties[required]; !exists {
			return fmt.Errorf("required parameter '%s' not found in properties", required)
		}
	}

	// Validate each property
	for name, param := range params.Properties {
		if err := r.validateToolParameter(name, param); err != nil {
			return err
		}
	}

	return nil
}

// validateToolParameter validates a single tool parameter
func (r *Registry) validateToolParameter(name string, param ToolParameter) error {
	validTypes := map[string]bool{
		"string":  true,
		"number":  true,
		"integer": true,
		"boolean": true,
		"array":   true,
		"object":  true,
	}

	if !validTypes[param.Type] {
		return fmt.Errorf("invalid parameter type '%s' for parameter '%s'", param.Type, name)
	}

	if param.Description == "" {
		return fmt.Errorf("parameter '%s' must have a description", name)
	}

	// Validate numeric constraints
	if param.Type == "number" || param.Type == "integer" {
		if param.Minimum != nil && param.Maximum != nil && *param.Minimum > *param.Maximum {
			return fmt.Errorf("parameter '%s' minimum value cannot be greater than maximum", name)
		}
	}

	return nil
}

// getToolCategory determines the category of a tool based on its name
func (r *Registry) getToolCategory(name string) string {
	// Built-in categories
	builtinCategories := map[string]string{
		"datetime":       "utility",
		"sql_query":      "database",
		"rag_search":     "rag",
		"document_count": "rag",
		"document_info":  "rag",
		"stats_query":    "rag",
		"file_read":      "file",
		"file_list":      "file",
		"http_request":   "network",
	}

	if category, exists := builtinCategories[name]; exists {
		return category
	}

	return "custom"
}

// Stats returns statistics about the tool registry
func (r *Registry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tools":   len(r.tools),
		"enabled_tools": r.GetEnabledToolCount(),
		"categories":    make(map[string]int),
	}

	categories := make(map[string]int)
	for name := range r.tools {
		category := r.getToolCategory(name)
		categories[category]++
	}
	stats["categories"] = categories

	return stats
}
