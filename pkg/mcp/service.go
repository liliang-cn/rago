// Package mcp provides tool definitions for LLM function calling.
// MCP (Model Context Protocol) provides tool specifications that LLMs 
// can use through their tool calling APIs.
package mcp

import (
	"fmt"
	"log"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Service provides MCP tool definitions for LLM tool calling.
type Service struct {
	config core.MCPConfig
	tools  []core.ToolInfo
}

// NewService creates a new MCP service for tool definitions.
func NewService(config core.MCPConfig) (*Service, error) {
	service := &Service{
		config: config,
	}
	
	// Load tool definitions if mcpServers.json is configured
	if config.ServersPath != "" {
		tools, err := LoadToolsFromMCP(config.ServersPath)
		if err != nil {
			// Don't fail, just log
			log.Printf("[MCP] Could not load tools from %s: %v\n", config.ServersPath, err)
			service.tools = []core.ToolInfo{}
		} else {
			service.tools = tools
			log.Printf("[MCP] Loaded %d tool definitions\n", len(tools))
		}
	}
	
	return service, nil
}

// GetTools returns all available tool definitions for LLM tool calling.
func (s *Service) GetTools() []core.ToolInfo {
	return s.tools
}

// GetToolsForLLM returns tool definitions formatted for LLM tool calling.
// This is the main method for getting tools to pass to the LLM.
func (s *Service) GetToolsForLLM() []core.ToolInfo {
	return s.tools
}

// AddTool adds a custom tool definition.
func (s *Service) AddTool(tool core.ToolInfo) {
	s.tools = append(s.tools, tool)
}

// FindTool finds a tool by name.
func (s *Service) FindTool(name string) (*core.ToolInfo, bool) {
	for _, tool := range s.tools {
		if tool.Name == name {
			return &tool, true
		}
	}
	return nil, false
}

// SearchTools searches for tools matching a query.
func (s *Service) SearchTools(query string) []core.ToolInfo {
	var results []core.ToolInfo
	for _, tool := range s.tools {
		// Simple substring search in name or description
		if contains(tool.Name, query) || contains(tool.Description, query) {
			results = append(results, tool)
		}
	}
	return results
}

// GetToolsByCategory returns tools in a specific category.
func (s *Service) GetToolsByCategory(category string) []core.ToolInfo {
	var results []core.ToolInfo
	for _, tool := range s.tools {
		if tool.Category == category {
			results = append(results, tool)
		}
	}
	return results
}

// ReloadTools reloads tool definitions from the configured source.
func (s *Service) ReloadTools() error {
	if s.config.ServersPath == "" {
		return fmt.Errorf("no servers path configured")
	}
	
	tools, err := LoadToolsFromMCP(s.config.ServersPath)
	if err != nil {
		return fmt.Errorf("failed to reload tools: %w", err)
	}
	
	s.tools = tools
	fmt.Printf("[MCP] Reloaded %d tool definitions\n", len(tools))
	return nil
}

// GetMetrics returns metrics about tool definitions.
func (s *Service) GetMetrics() map[string]interface{} {
	categories := make(map[string]int)
	servers := make(map[string]int)
	
	for _, tool := range s.tools {
		if tool.Category != "" {
			categories[tool.Category]++
		}
		if tool.ServerName != "" {
			servers[tool.ServerName]++
		}
	}
	
	return map[string]interface{}{
		"total_tools":      len(s.tools),
		"tools_by_category": categories,
		"tools_by_server":  servers,
	}
}


// Helper function for case-insensitive substring search
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}