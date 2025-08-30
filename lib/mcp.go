package rago

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/internal/mcp"
)

// MCPClient provides MCP (Model Context Protocol) functionality
type MCPClient struct {
	api     *mcp.MCPLibraryAPI
	enabled bool
}

// EnableMCP enables MCP functionality for the rago client
func (c *Client) EnableMCP(ctx context.Context) error {
	if !c.config.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}
	
	if c.mcpClient != nil && c.mcpClient.enabled {
		return nil // Already enabled
	}
	
	api := mcp.NewMCPLibraryAPI(&c.config.MCP)
	if err := api.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP service: %w", err)
	}
	
	c.mcpClient = &MCPClient{
		api:     api,
		enabled: true,
	}
	
	return nil
}

// DisableMCP disables MCP functionality
func (c *Client) DisableMCP() error {
	if c.mcpClient == nil || !c.mcpClient.enabled {
		return nil
	}
	
	if err := c.mcpClient.api.Stop(); err != nil {
		return fmt.Errorf("failed to stop MCP service: %w", err)
	}
	
	c.mcpClient.enabled = false
	return nil
}

// IsMCPEnabled returns whether MCP is enabled and ready
func (c *Client) IsMCPEnabled() bool {
	return c.mcpClient != nil && c.mcpClient.enabled && c.config.MCP.Enabled
}

// ListMCPTools returns all available MCP tools
func (c *Client) ListMCPTools() ([]mcp.ToolSummary, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	return c.mcpClient.api.ListTools(), nil
}

// GetMCPToolsForLLM returns MCP tools in LLM-compatible format
func (c *Client) GetMCPToolsForLLM() ([]map[string]interface{}, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	return c.mcpClient.api.GetToolsForLLMIntegration(), nil
}

// CallMCPTool calls an MCP tool with the specified arguments
func (c *Client) CallMCPTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.MCPToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	return c.mcpClient.api.CallTool(ctx, toolName, args)
}

// CallMCPToolWithTimeout calls an MCP tool with a timeout
func (c *Client) CallMCPToolWithTimeout(toolName string, args map[string]interface{}, timeout time.Duration) (*mcp.MCPToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	return c.mcpClient.api.CallToolWithTimeout(toolName, args, timeout)
}

// GetMCPServerStatus returns the status of MCP servers
func (c *Client) GetMCPServerStatus() (map[string]bool, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	return c.mcpClient.api.GetServerStatuses(), nil
}

// BatchCallMCPTools calls multiple MCP tools in parallel
func (c *Client) BatchCallMCPTools(ctx context.Context, calls []ToolCall) ([]mcp.MCPToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}
	
	// Convert to mcp.ToolCall format
	mcpCalls := make([]mcp.ToolCall, len(calls))
	for i, call := range calls {
		mcpCalls[i] = mcp.ToolCall{
			ToolName: call.ToolName,
			Args:     call.Args,
		}
	}
	
	return c.mcpClient.api.BatchCall(ctx, mcpCalls)
}

// ToolCall represents a tool call request for lib usage
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args"`
}