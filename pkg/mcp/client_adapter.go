package mcp

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/mcp/server"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// clientAdapter adapts mcp.Client to server.MCPClient interface
type clientAdapter struct {
	client *Client
}

// NewClientAdapter creates a new client adapter
func NewClientAdapter(client *Client) server.MCPClient {
	return &clientAdapter{client: client}
}

// Connect establishes connection to the MCP server
func (a *clientAdapter) Connect(ctx context.Context) error {
	return a.client.Connect(ctx)
}

// Close closes the connection to the MCP server
func (a *clientAdapter) Close() error {
	return a.client.Close()
}

// IsConnected returns whether the client is connected
func (a *clientAdapter) IsConnected() bool {
	return a.client.IsConnected()
}

// GetTools returns the available tools
func (a *clientAdapter) GetTools() map[string]*mcpsdk.Tool {
	return a.client.GetTools()
}

// CallTool calls a tool on the MCP server
func (a *clientAdapter) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*server.ToolResult, error) {
	result, err := a.client.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, err
	}
	
	return &server.ToolResult{
		Success: result.Success,
		Data:    result.Data,
		Error:   result.Error,
	}, nil
}

// GetServerInfo returns information about the connected server
func (a *clientAdapter) GetServerInfo() *mcpsdk.Implementation {
	return a.client.GetServerInfo()
}

// CreateClientFactory creates a client factory function for the server manager
func CreateClientFactory() server.ClientFactory {
	return func(config core.ServerConfig) (server.MCPClient, error) {
		// Convert core.ServerConfig to mcp.ServerConfig
		mcpConfig := &ServerConfig{
			Name:             config.Name,
			Description:      config.Description,
			Command:          config.Command,
			Args:             config.Args,
			WorkingDir:       config.WorkingDir,
			Env:              config.Env,
			AutoStart:        config.AutoStart,
			RestartOnFailure: config.RestartOnFailure,
			MaxRestarts:      config.MaxRestarts,
			RestartDelay:     config.RestartDelay,
			Capabilities:     config.Capabilities,
		}
		
		// Create MCP client
		client, err := NewClient(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP client: %w", err)
		}
		
		// Return adapter
		return NewClientAdapter(client), nil
	}
}