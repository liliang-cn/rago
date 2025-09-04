package mcp

import (
	"context"
	"fmt"
	"time"
)

// MCPService provides high-level MCP functionality for library usage
type MCPService struct {
	toolManager *MCPToolManager
	config      *Config
}

// NewMCPService creates a new MCP service instance
func NewMCPService(config *Config) *MCPService {
	return &MCPService{
		toolManager: NewMCPToolManager(config),
		config:      config,
	}
}

// Initialize starts the MCP service and auto-start servers
func (s *MCPService) Initialize(ctx context.Context) error {
	if !s.config.Enabled {
		return fmt.Errorf("MCP service is disabled")
	}

	return s.toolManager.Start(ctx)
}

// CallTool calls an MCP tool by name with arguments
func (s *MCPService) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*MCPToolResult, error) {
	return s.toolManager.CallTool(ctx, toolName, args)
}

// GetAvailableTools returns all available MCP tools
func (s *MCPService) GetAvailableTools() map[string]*MCPToolWrapper {
	return s.toolManager.ListTools()
}

// GetToolsByServer returns tools from a specific server
func (s *MCPService) GetToolsByServer(serverName string) map[string]*MCPToolWrapper {
	return s.toolManager.ListToolsByServer(serverName)
}

// GetToolsForLLM returns tools formatted for LLM function calling
func (s *MCPService) GetToolsForLLM() []map[string]interface{} {
	return s.toolManager.GetToolsForLLM()
}

// StartServer starts a specific MCP server
func (s *MCPService) StartServer(ctx context.Context, serverName string) error {
	return s.toolManager.StartServer(ctx, serverName)
}

// StopServer stops a specific MCP server
func (s *MCPService) StopServer(serverName string) error {
	return s.toolManager.StopServer(serverName)
}

// GetServerStatus returns the connection status of all servers
func (s *MCPService) GetServerStatus() map[string]bool {
	return s.toolManager.GetServerStatus()
}

// IsEnabled returns whether MCP service is enabled
func (s *MCPService) IsEnabled() bool {
	return s.config.Enabled
}

// GetConfig returns the MCP configuration
func (s *MCPService) GetConfig() *Config {
	return s.config
}

// Close shuts down all MCP servers and cleans up resources
func (s *MCPService) Close() error {
	return s.toolManager.Close()
}

// MCPLibraryAPI provides a simplified API for programmatic usage
type MCPLibraryAPI struct {
	service *MCPService
}

// NewMCPLibraryAPI creates a new MCP library API
func NewMCPLibraryAPI(config *Config) *MCPLibraryAPI {
	return &MCPLibraryAPI{
		service: NewMCPService(config),
	}
}

// Start initializes the MCP service
func (api *MCPLibraryAPI) Start(ctx context.Context) error {
	return api.service.Initialize(ctx)
}

// CallTool is a convenience method for calling MCP tools
func (api *MCPLibraryAPI) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*MCPToolResult, error) {
	return api.service.CallTool(ctx, toolName, args)
}

// CallToolWithTimeout calls a tool with a specific timeout
func (api *MCPLibraryAPI) CallToolWithTimeout(toolName string, args map[string]interface{}, timeout time.Duration) (*MCPToolResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return api.service.CallTool(ctx, toolName, args)
}

// ListTools returns available tools in a simple format
func (api *MCPLibraryAPI) ListTools() []ToolSummary {
	tools := api.service.GetAvailableTools()
	summaries := make([]ToolSummary, 0, len(tools))

	for _, tool := range tools {
		summaries = append(summaries, ToolSummary{
			Name:        tool.Name(),
			Description: tool.Description(),
			ServerName:  tool.ServerName(),
		})
	}

	return summaries
}

// GetToolsForLLMIntegration returns tools formatted for LLM integration
func (api *MCPLibraryAPI) GetToolsForLLMIntegration() []map[string]interface{} {
	return api.service.GetToolsForLLM()
}

// IsAvailable checks if MCP service is available and ready
func (api *MCPLibraryAPI) IsAvailable() bool {
	return api.service.IsEnabled()
}

// GetServerStatuses returns server connection statuses
func (api *MCPLibraryAPI) GetServerStatuses() map[string]bool {
	return api.service.GetServerStatus()
}

// Stop shuts down the MCP service
func (api *MCPLibraryAPI) Stop() error {
	return api.service.Close()
}

// ToolSummary provides a simple view of an MCP tool
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerName  string `json:"server_name"`
}

// QuickCallOptions provides options for quick tool calls
type QuickCallOptions struct {
	Timeout    time.Duration          `json:"timeout"`
	ServerName string                 `json:"server_name,omitempty"`
	Args       map[string]interface{} `json:"args"`
}

// QuickCall provides a simple way to call MCP tools
func (api *MCPLibraryAPI) QuickCall(toolName string, options QuickCallOptions) (*MCPToolResult, error) {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return api.CallToolWithTimeout(toolName, options.Args, timeout)
}

// BatchCall calls multiple tools in parallel
func (api *MCPLibraryAPI) BatchCall(ctx context.Context, calls []ToolCall) ([]MCPToolResult, error) {
	results := make([]MCPToolResult, len(calls))
	errChan := make(chan error, len(calls))

	for i, call := range calls {
		go func(index int, toolCall ToolCall) {
			result, err := api.service.CallTool(ctx, toolCall.ToolName, toolCall.Args)
			if err != nil {
				errChan <- fmt.Errorf("tool %s failed: %w", toolCall.ToolName, err)
				return
			}
			results[index] = *result
			errChan <- nil
		}(i, call)
	}

	// Wait for all calls to complete
	for i := 0; i < len(calls); i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	return results, nil
}

// ToolCall represents a tool call request
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args"`
}

// Example usage functions for documentation:

// ExampleBasicUsage demonstrates basic MCP API usage
func ExampleBasicUsage(config *Config) error {
	// Create API instance
	api := NewMCPLibraryAPI(config)

	// Start MCP service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := api.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP: %w", err)
	}
	defer func() {
		if err := api.Stop(); err != nil {
			fmt.Printf("failed to stop mcp api: %v\n", err)
		}
	}()

	// List available tools
	tools := api.ListTools()
	fmt.Printf("Available tools: %d\n", len(tools))

	// Call a tool
	result, err := api.QuickCall("mcp_sqlite_query", QuickCallOptions{
		Timeout: 10 * time.Second,
		Args: map[string]interface{}{
			"query": "SELECT COUNT(*) FROM users",
		},
	})
	if err != nil {
		return fmt.Errorf("tool call failed: %w", err)
	}

	fmt.Printf("Tool result: success=%v, data=%v\n", result.Success, result.Data)
	return nil
}

// ExampleLLMIntegration demonstrates LLM integration usage
func ExampleLLMIntegration(config *Config) ([]map[string]interface{}, error) {
	api := NewMCPLibraryAPI(config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := api.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP: %w", err)
	}
	defer func() {
		if err := api.Stop(); err != nil {
			fmt.Printf("failed to stop mcp api: %v\n", err)
		}
	}()

	// Get tools formatted for LLM function calling
	llmTools := api.GetToolsForLLMIntegration()

	return llmTools, nil
}
