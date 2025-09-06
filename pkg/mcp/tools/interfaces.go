package tools

import "context"

// MCPClient defines the interface for MCP client operations used by tools
type MCPClient interface {
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error)
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Success bool
	Data    interface{}
	Error   string
}

// ServerManager interface for accessing MCP servers
type ServerManager interface {
	GetServer(name string) (MCPClient, error)
}