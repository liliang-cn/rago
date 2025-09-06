package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient defines the interface for MCP client operations
type MCPClient interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	GetTools() map[string]*mcp.Tool
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error)
	GetServerInfo() *mcp.Implementation
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Success bool
	Data    interface{}
	Error   string
}