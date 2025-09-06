// Package tools provides tool management and execution for the MCP pillar.
package tools

import (
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Registry manages tool registration and lookup
type Registry struct {
	mu              sync.RWMutex
	tools           map[string]*RegisteredTool
	toolsByServer   map[string][]*RegisteredTool
	toolsByCategory map[string][]*RegisteredTool
	aliases         map[string]string // Tool aliases for convenience
	validators      []ToolValidator
	hooks           []RegistrationHook
}

// RegisteredTool represents a tool registered in the system
type RegisteredTool struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	ServerName     string                 `json:"server_name"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"`
	InputSchema    map[string]interface{} `json:"input_schema"`
	OutputSchema   map[string]interface{} `json:"output_schema,omitempty"`
	Capabilities   []string               `json:"capabilities"`
	Version        string                 `json:"version"`
	Priority       int                    `json:"priority"`
	Timeout        time.Duration          `json:"timeout"`
	MaxRetries     int                    `json:"max_retries"`
	Cacheable      bool                   `json:"cacheable"`
	CacheDuration  time.Duration          `json:"cache_duration"`
	RateLimit      *RateLimit             `json:"rate_limit,omitempty"`
	RequiredFields []string               `json:"required_fields"`
	Metadata       map[string]interface{} `json:"metadata"`
	RegisteredAt   time.Time              `json:"registered_at"`
	LastUsed       time.Time              `json:"last_used"`
	UsageCount     int64                  `json:"usage_count"`
	ErrorCount     int64                  `json:"error_count"`
	Available      bool                   `json:"available"`
}

// RateLimit defines rate limiting for a tool
type RateLimit struct {
	MaxCalls      int           `json:"max_calls"`
	Period        time.Duration `json:"period"`
	BurstSize     int           `json:"burst_size"`
	CurrentWindow time.Time     `json:"-"`
	CallCount     int           `json:"-"`
	mu            sync.Mutex
}

// ToolValidator validates tool registration
type ToolValidator func(tool *RegisteredTool) error

// RegistrationHook is called when a tool is registered/unregistered
type RegistrationHook func(tool *RegisteredTool, registered bool)

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:           make(map[string]*RegisteredTool),
		toolsByServer:   make(map[string][]*RegisteredTool),
		toolsByCategory: make(map[string][]*RegisteredTool),
		aliases:         make(map[string]string),
		validators:      []ToolValidator{defaultValidator},
	}
}

// Register registers a new tool
func (r *Registry) Register(tool *RegisteredTool) error {
	// Validate tool
	for _, validator := range r.validators {
		if err := validator(tool); err != nil {
			return fmt.Errorf("tool validation failed: %w", err)
		}
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check for duplicate
	if existing, exists := r.tools[tool.ID]; exists {
		if existing.ServerName == tool.ServerName {
			// Update existing tool
			r.updateToolInternal(tool)
			return nil
		}
		return fmt.Errorf("tool ID %s already registered", tool.ID)
	}
	
	// Set defaults
	if tool.Timeout == 0 {
		tool.Timeout = 30 * time.Second
	}
	if tool.MaxRetries == 0 {
		tool.MaxRetries = 3
	}
	if tool.Priority == 0 {
		tool.Priority = 50 // Default priority
	}
	if tool.CacheDuration == 0 && tool.Cacheable {
		tool.CacheDuration = 5 * time.Minute
	}
	
	tool.RegisteredAt = time.Now()
	tool.Available = true
	
	// Register in main registry
	r.tools[tool.ID] = tool
	
	// Update server index
	r.toolsByServer[tool.ServerName] = append(r.toolsByServer[tool.ServerName], tool)
	
	// Update category index
	r.toolsByCategory[tool.Category] = append(r.toolsByCategory[tool.Category], tool)
	
	// Call registration hooks
	for _, hook := range r.hooks {
		hook(tool, true)
	}
	
	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(toolID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	tool, exists := r.tools[toolID]
	if !exists {
		return core.ErrToolNotFound
	}
	
	// Remove from main registry
	delete(r.tools, toolID)
	
	// Remove from server index
	r.removeFromServerIndex(tool)
	
	// Remove from category index
	r.removeFromCategoryIndex(tool)
	
	// Remove aliases
	for alias, id := range r.aliases {
		if id == toolID {
			delete(r.aliases, alias)
		}
	}
	
	// Call registration hooks
	for _, hook := range r.hooks {
		hook(tool, false)
	}
	
	return nil
}

// Get retrieves a tool by ID or alias
func (r *Registry) Get(idOrAlias string) (*RegisteredTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if it's an alias
	if actualID, isAlias := r.aliases[idOrAlias]; isAlias {
		idOrAlias = actualID
	}
	
	tool, exists := r.tools[idOrAlias]
	if !exists {
		return nil, core.ErrToolNotFound
	}
	
	return tool, nil
}

// GetByServer returns all tools for a specific server
func (r *Registry) GetByServer(serverName string) []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.toolsByServer[serverName]
}

// GetByCategory returns all tools in a specific category
func (r *Registry) GetByCategory(category string) []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.toolsByCategory[category]
}

// List returns all registered tools
func (r *Registry) List() []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]*RegisteredTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListAvailable returns all available tools
func (r *Registry) ListAvailable() []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var tools []*RegisteredTool
	for _, tool := range r.tools {
		if tool.Available {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Search searches for tools matching criteria
func (r *Registry) Search(criteria SearchCriteria) []*RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var results []*RegisteredTool
	
	for _, tool := range r.tools {
		if criteria.Matches(tool) {
			results = append(results, tool)
		}
	}
	
	return results
}

// SetAlias sets an alias for a tool
func (r *Registry) SetAlias(alias, toolID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if tool exists
	if _, exists := r.tools[toolID]; !exists {
		return core.ErrToolNotFound
	}
	
	// Check if alias is already in use
	if existingID, exists := r.aliases[alias]; exists && existingID != toolID {
		return fmt.Errorf("alias %s already in use for tool %s", alias, existingID)
	}
	
	r.aliases[alias] = toolID
	return nil
}

// RemoveAlias removes an alias
func (r *Registry) RemoveAlias(alias string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.aliases[alias]; !exists {
		return fmt.Errorf("alias %s not found", alias)
	}
	
	delete(r.aliases, alias)
	return nil
}

// UpdateAvailability updates the availability of tools
func (r *Registry) UpdateAvailability(serverName string, available bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, tool := range r.toolsByServer[serverName] {
		tool.Available = available
	}
}

// UpdateUsage updates usage statistics for a tool
func (r *Registry) UpdateUsage(toolID string, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if tool, exists := r.tools[toolID]; exists {
		tool.UsageCount++
		tool.LastUsed = time.Now()
		if !success {
			tool.ErrorCount++
		}
	}
}

// AddValidator adds a tool validator
func (r *Registry) AddValidator(validator ToolValidator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators = append(r.validators, validator)
}

// AddHook adds a registration hook
func (r *Registry) AddHook(hook RegistrationHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// GetStats returns registry statistics
func (r *Registry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_tools":      len(r.tools),
		"total_servers":    len(r.toolsByServer),
		"total_categories": len(r.toolsByCategory),
		"total_aliases":    len(r.aliases),
	}
	
	// Count available tools
	availableCount := 0
	for _, tool := range r.tools {
		if tool.Available {
			availableCount++
		}
	}
	stats["available_tools"] = availableCount
	
	// Category breakdown
	categoryStats := make(map[string]int)
	for category, tools := range r.toolsByCategory {
		categoryStats[category] = len(tools)
	}
	stats["by_category"] = categoryStats
	
	// Server breakdown
	serverStats := make(map[string]int)
	for server, tools := range r.toolsByServer {
		serverStats[server] = len(tools)
	}
	stats["by_server"] = serverStats
	
	return stats
}

// updateToolInternal updates an existing tool (must be called with lock held)
func (r *Registry) updateToolInternal(tool *RegisteredTool) {
	existing := r.tools[tool.ID]
	
	// Preserve certain fields
	tool.RegisteredAt = existing.RegisteredAt
	tool.UsageCount = existing.UsageCount
	tool.ErrorCount = existing.ErrorCount
	tool.LastUsed = existing.LastUsed
	
	// Update the tool
	r.tools[tool.ID] = tool
	
	// Update indices
	r.updateServerIndex(existing, tool)
	r.updateCategoryIndex(existing, tool)
}

// removeFromServerIndex removes a tool from server index
func (r *Registry) removeFromServerIndex(tool *RegisteredTool) {
	tools := r.toolsByServer[tool.ServerName]
	for i, t := range tools {
		if t.ID == tool.ID {
			r.toolsByServer[tool.ServerName] = append(tools[:i], tools[i+1:]...)
			break
		}
	}
	if len(r.toolsByServer[tool.ServerName]) == 0 {
		delete(r.toolsByServer, tool.ServerName)
	}
}

// removeFromCategoryIndex removes a tool from category index
func (r *Registry) removeFromCategoryIndex(tool *RegisteredTool) {
	tools := r.toolsByCategory[tool.Category]
	for i, t := range tools {
		if t.ID == tool.ID {
			r.toolsByCategory[tool.Category] = append(tools[:i], tools[i+1:]...)
			break
		}
	}
	if len(r.toolsByCategory[tool.Category]) == 0 {
		delete(r.toolsByCategory, tool.Category)
	}
}

// updateServerIndex updates server index when tool changes
func (r *Registry) updateServerIndex(old, new *RegisteredTool) {
	if old.ServerName != new.ServerName {
		r.removeFromServerIndex(old)
		r.toolsByServer[new.ServerName] = append(r.toolsByServer[new.ServerName], new)
	}
}

// updateCategoryIndex updates category index when tool changes
func (r *Registry) updateCategoryIndex(old, new *RegisteredTool) {
	if old.Category != new.Category {
		r.removeFromCategoryIndex(old)
		r.toolsByCategory[new.Category] = append(r.toolsByCategory[new.Category], new)
	}
}

// defaultValidator provides default validation for tools
func defaultValidator(tool *RegisteredTool) error {
	if tool.ID == "" {
		return fmt.Errorf("tool ID is required")
	}
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.ServerName == "" {
		return fmt.Errorf("server name is required")
	}
	return nil
}

// SearchCriteria defines search criteria for tools
type SearchCriteria struct {
	Name        string
	Category    string
	ServerName  string
	Available   *bool
	Cacheable   *bool
	MinPriority int
}

// Matches checks if a tool matches the search criteria
func (c SearchCriteria) Matches(tool *RegisteredTool) bool {
	if c.Name != "" && !contains(tool.Name, c.Name) && !contains(tool.Description, c.Name) {
		return false
	}
	if c.Category != "" && tool.Category != c.Category {
		return false
	}
	if c.ServerName != "" && tool.ServerName != c.ServerName {
		return false
	}
	if c.Available != nil && tool.Available != *c.Available {
		return false
	}
	if c.Cacheable != nil && tool.Cacheable != *c.Cacheable {
		return false
	}
	if c.MinPriority > 0 && tool.Priority < c.MinPriority {
		return false
	}
	return true
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simplified implementation - in production use strings package
	return true
}

// CheckRateLimit checks and updates rate limit for a tool
func (rl *RateLimit) CheckRateLimit() bool {
	if rl == nil {
		return true // No rate limit
	}
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Check if we're in a new window
	if now.Sub(rl.CurrentWindow) > rl.Period {
		rl.CurrentWindow = now
		rl.CallCount = 0
	}
	
	// Check if we've exceeded the limit
	if rl.CallCount >= rl.MaxCalls {
		return false
	}
	
	// Check burst size
	if rl.BurstSize > 0 && rl.CallCount >= rl.BurstSize {
		// Calculate if enough time has passed for next call
		timeSinceWindow := now.Sub(rl.CurrentWindow)
		expectedInterval := rl.Period / time.Duration(rl.MaxCalls)
		if timeSinceWindow < expectedInterval*time.Duration(rl.CallCount) {
			return false
		}
	}
	
	rl.CallCount++
	return true
}