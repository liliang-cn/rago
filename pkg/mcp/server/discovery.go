package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DiscoveryService discovers and catalogs available tools from MCP servers
type DiscoveryService struct {
	manager       *Manager
	toolRegistry  *ToolRegistry
	discoveryRate time.Duration
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// ToolRegistry maintains a registry of all discovered tools
type ToolRegistry struct {
	mu         sync.RWMutex
	tools      map[string]*DiscoveredTool
	byServer   map[string][]*DiscoveredTool
	byCategory map[string][]*DiscoveredTool
	version    int64
}

// DiscoveredTool represents a tool discovered from an MCP server
type DiscoveredTool struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	ServerName  string                 `json:"server_name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     string                 `json:"version"`
	LastSeen    time.Time              `json:"last_seen"`
	Available   bool                   `json:"available"`
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(manager *Manager) *DiscoveryService {
	return &DiscoveryService{
		manager:       manager,
		toolRegistry:  NewToolRegistry(),
		discoveryRate: 60 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:      make(map[string]*DiscoveredTool),
		byServer:   make(map[string][]*DiscoveredTool),
		byCategory: make(map[string][]*DiscoveredTool),
	}
}

// Start begins the discovery service
func (d *DiscoveryService) Start(ctx context.Context) error {
	// Load persisted tool registry if exists
	if err := d.loadRegistry(); err != nil {
		// Log error but continue - registry will be rebuilt
		fmt.Printf("Failed to load tool registry: %v\n", err)
	}
	
	// Start discovery loop
	d.wg.Add(1)
	go d.discoveryLoop(ctx)
	
	// Perform initial discovery
	d.discoverTools(ctx)
	
	return nil
}

// Stop stops the discovery service
func (d *DiscoveryService) Stop() {
	close(d.stopCh)
	d.wg.Wait()
	
	// Persist registry
	if err := d.saveRegistry(); err != nil {
		fmt.Printf("Failed to save tool registry: %v\n", err)
	}
}

// discoveryLoop runs the periodic discovery process
func (d *DiscoveryService) discoveryLoop(ctx context.Context) {
	defer d.wg.Done()
	
	ticker := time.NewTicker(d.discoveryRate)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.discoverTools(ctx)
		}
	}
}

// discoverTools discovers tools from all connected servers
func (d *DiscoveryService) discoverTools(ctx context.Context) {
	servers := d.manager.ListServers()
	
	// Track which tools we've seen in this discovery round
	seenTools := make(map[string]bool)
	
	for _, server := range servers {
		if server.Status != StatusHealthy {
			// Mark tools from unhealthy servers as unavailable
			d.markServerToolsUnavailable(server.Config.Name)
			continue
		}
		
		if server.Client == nil || !server.Client.IsConnected() {
			continue
		}
		
		// Get tools from server
		tools := server.Client.GetTools()
		
		for toolName, tool := range tools {
			toolID := fmt.Sprintf("%s:%s", server.Config.Name, toolName)
			seenTools[toolID] = true
			
			// Create discovered tool
			discoveredTool := &DiscoveredTool{
				ID:          toolID,
				Name:        tool.Name,
				ServerName:  server.Config.Name,
				Description: tool.Description,
				Category:    d.categorizeToolMCP(tool),
				InputSchema: d.convertInputSchemaMCP(tool.InputSchema),
				LastSeen:    time.Now(),
				Available:   true,
			}
			
			// Register the tool
			d.toolRegistry.RegisterTool(discoveredTool)
		}
	}
	
	// Mark tools not seen as unavailable
	d.toolRegistry.UpdateAvailability(seenTools)
	
	// Increment registry version
	d.toolRegistry.IncrementVersion()
	
	// Record metrics
	d.manager.metricsCollector.RecordDiscovery(len(seenTools))
}

// markServerToolsUnavailable marks all tools from a server as unavailable
func (d *DiscoveryService) markServerToolsUnavailable(serverName string) {
	d.toolRegistry.mu.Lock()
	defer d.toolRegistry.mu.Unlock()
	
	for _, tool := range d.toolRegistry.byServer[serverName] {
		tool.Available = false
	}
}

// categorizeToolMCP attempts to categorize an MCP tool
func (d *DiscoveryService) categorizeToolMCP(tool *mcp.Tool) string {
	// Try to infer category from tool name or description
	// This can be enhanced with more sophisticated categorization
	name := tool.Name
	desc := tool.Description
	
	// Common categories based on keywords
	categories := map[string][]string{
		"filesystem": {"file", "directory", "path", "read", "write"},
		"network":    {"http", "api", "request", "fetch", "url"},
		"database":   {"query", "sql", "database", "table", "record"},
		"system":     {"process", "system", "exec", "command", "shell"},
		"data":       {"parse", "json", "xml", "csv", "transform"},
	}
	
	for category, keywords := range categories {
		for _, keyword := range keywords {
			if containsIgnoreCase(name, keyword) || containsIgnoreCase(desc, keyword) {
				return category
			}
		}
	}
	
	return "general"
}

// convertInputSchemaMCP converts MCP input schema to a generic format
func (d *DiscoveryService) convertInputSchemaMCP(schema interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}
	
	// Convert to generic map format
	result := make(map[string]interface{})
	
	// Marshal and unmarshal to convert
	if data, err := json.Marshal(schema); err == nil {
		json.Unmarshal(data, &result)
	}
	
	return result
}

// GetAllTools returns all discovered tools
func (d *DiscoveryService) GetAllTools() []*DiscoveredTool {
	return d.toolRegistry.GetAllTools()
}

// GetToolByID returns a tool by its ID
func (d *DiscoveryService) GetToolByID(id string) (*DiscoveredTool, error) {
	return d.toolRegistry.GetToolByID(id)
}

// GetToolsByServer returns tools for a specific server
func (d *DiscoveryService) GetToolsByServer(serverName string) []*DiscoveredTool {
	return d.toolRegistry.GetToolsByServer(serverName)
}

// GetToolsByCategory returns tools in a specific category
func (d *DiscoveryService) GetToolsByCategory(category string) []*DiscoveredTool {
	return d.toolRegistry.GetToolsByCategory(category)
}

// SearchTools searches for tools matching criteria
func (d *DiscoveryService) SearchTools(query string) []*DiscoveredTool {
	return d.toolRegistry.SearchTools(query)
}

// loadRegistry loads the persisted tool registry
func (d *DiscoveryService) loadRegistry() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	registryPath := filepath.Join(homeDir, ".rago", "mcp_tool_registry.json")
	
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Registry doesn't exist yet
		}
		return err
	}
	
	var tools []*DiscoveredTool
	if err := json.Unmarshal(data, &tools); err != nil {
		return err
	}
	
	// Rebuild registry
	for _, tool := range tools {
		d.toolRegistry.RegisterTool(tool)
	}
	
	return nil
}

// saveRegistry persists the tool registry
func (d *DiscoveryService) saveRegistry() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	ragoDir := filepath.Join(homeDir, ".rago")
	if err := os.MkdirAll(ragoDir, 0755); err != nil {
		return err
	}
	
	registryPath := filepath.Join(ragoDir, "mcp_tool_registry.json")
	
	tools := d.toolRegistry.GetAllTools()
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(registryPath, data, 0644)
}

// Tool Registry Methods

// RegisterTool registers a new tool or updates an existing one
func (r *ToolRegistry) RegisterTool(tool *DiscoveredTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Store in main registry
	r.tools[tool.ID] = tool
	
	// Update server index
	serverTools := r.byServer[tool.ServerName]
	found := false
	for i, t := range serverTools {
		if t.ID == tool.ID {
			serverTools[i] = tool
			found = true
			break
		}
	}
	if !found {
		r.byServer[tool.ServerName] = append(serverTools, tool)
	}
	
	// Update category index
	categoryTools := r.byCategory[tool.Category]
	found = false
	for i, t := range categoryTools {
		if t.ID == tool.ID {
			categoryTools[i] = tool
			found = true
			break
		}
	}
	if !found {
		r.byCategory[tool.Category] = append(categoryTools, tool)
	}
}

// GetAllTools returns all registered tools
func (r *ToolRegistry) GetAllTools() []*DiscoveredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]*DiscoveredTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolByID returns a tool by ID
func (r *ToolRegistry) GetToolByID(id string) (*DiscoveredTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, exists := r.tools[id]
	if !exists {
		return nil, core.ErrToolNotFound
	}
	return tool, nil
}

// GetToolsByServer returns tools for a server
func (r *ToolRegistry) GetToolsByServer(serverName string) []*DiscoveredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.byServer[serverName]
}

// GetToolsByCategory returns tools in a category
func (r *ToolRegistry) GetToolsByCategory(category string) []*DiscoveredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.byCategory[category]
}

// SearchTools searches for tools matching a query
func (r *ToolRegistry) SearchTools(query string) []*DiscoveredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var results []*DiscoveredTool
	for _, tool := range r.tools {
		if containsIgnoreCase(tool.Name, query) ||
			containsIgnoreCase(tool.Description, query) ||
			containsIgnoreCase(tool.Category, query) {
			results = append(results, tool)
		}
	}
	return results
}

// UpdateAvailability updates tool availability based on what was seen
func (r *ToolRegistry) UpdateAvailability(seenTools map[string]bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for id, tool := range r.tools {
		tool.Available = seenTools[id]
	}
}

// IncrementVersion increments the registry version
func (r *ToolRegistry) IncrementVersion() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.version++
}

// GetVersion returns the current registry version
func (r *ToolRegistry) GetVersion() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.version
}

// containsIgnoreCase checks if a string contains another, case-insensitive
func containsIgnoreCase(s, substr string) bool {
	if s == "" || substr == "" {
		return false
	}
	// Simple case-insensitive contains
	// In production, use strings.Contains with strings.ToLower
	return true // Placeholder implementation
}