package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/pkg/tools"
)

// ListAvailableTools returns a list of all available tools in the system
func (c *Client) ListAvailableTools() []tools.ToolInfo {
	if c.processor == nil {
		return []tools.ToolInfo{}
	}
	return c.processor.GetToolRegistry().List()
}

// ListEnabledTools returns a list of currently enabled tools
func (c *Client) ListEnabledTools() []tools.ToolInfo {
	if c.processor == nil {
		return []tools.ToolInfo{}
	}
	return c.processor.GetToolRegistry().ListEnabled()
}

// ExecuteTool executes a tool with the given arguments
func (c *Client) ExecuteTool(toolName string, args map[string]interface{}) (*tools.ToolResult, error) {
	if c.processor == nil {
		return nil, fmt.Errorf("processor not initialized")
	}

	registry := c.processor.GetToolRegistry()
	if !registry.IsEnabled(toolName) {
		return nil, fmt.Errorf("tool '%s' is not enabled", toolName)
	}

	tool, exists := registry.Get(toolName)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// GetToolStats returns statistics about tool usage
func (c *Client) GetToolStats() map[string]interface{} {
	if c.processor == nil {
		return map[string]interface{}{
			"available_tools": 0,
			"enabled_tools":   0,
			"error":          "processor not initialized",
		}
	}

	registry := c.processor.GetToolRegistry()
	availableTools := registry.List()
	enabledTools := registry.ListEnabled()

	stats := map[string]interface{}{
		"available_tools": len(availableTools),
		"enabled_tools":   len(enabledTools),
		"tools_list":      make(map[string]interface{}),
	}

	toolsList := stats["tools_list"].(map[string]interface{})
	for _, tool := range availableTools {
		toolsList[tool.Name] = map[string]interface{}{
			"description": tool.Description,
			"enabled":     registry.IsEnabled(tool.Name),
		}
	}

	return stats
}