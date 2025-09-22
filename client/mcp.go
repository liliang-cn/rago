package client

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// MCPClient provides advanced MCP functionality for interactive features
type MCPClient struct {
	service *mcp.Service
	enabled bool
}

// IsInitialized returns whether the MCP client is initialized and enabled
func (m *MCPClient) IsInitialized() bool {
	return m != nil && m.enabled && m.service != nil
}

// GetToolDefinitions returns tool definitions in domain format for LLM (advanced feature)
func (m *MCPClient) GetToolDefinitions(ctx context.Context) []domain.ToolDefinition {
	if !m.IsInitialized() {
		return nil
	}

	tools := m.service.GetAvailableTools(ctx)
	var toolDefs []domain.ToolDefinition

	for _, tool := range tools {
		toolDef := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  map[string]interface{}{}, // TODO: Add parameter schemas
			},
		}
		toolDefs = append(toolDefs, toolDef)
	}

	return toolDefs
}

// CallTool calls an MCP tool with the specified arguments
func (m *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.ToolResult, error) {
	if !m.IsInitialized() {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	return m.service.CallTool(ctx, toolName, args)
}

// EnableMCPFull enables MCP functionality for the rago client with full features
func (c *BaseClient) EnableMCPFull(ctx context.Context) error {
	if !c.config.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	if c.mcpClient != nil && c.mcpClient.enabled {
		return nil // Already enabled
	}

	if c.mcpService == nil {
		return fmt.Errorf("MCP service not available")
	}

	// Start MCP servers
	if err := c.mcpService.StartServers(ctx, nil); err != nil {
		return fmt.Errorf("failed to start MCP servers: %w", err)
	}

	c.mcpClient = &MCPClient{
		service: c.mcpService,
		enabled: true,
	}

	return nil
}

// DisableMCP disables MCP functionality
func (c *BaseClient) DisableMCP() error {
	if c.mcpClient == nil || !c.mcpClient.enabled {
		return nil
	}

	// MCP service lifecycle is managed at the service layer
	// Just mark as disabled in client

	c.mcpClient.enabled = false
	return nil
}

// IsMCPEnabled returns whether MCP is enabled and ready
func (c *BaseClient) IsMCPEnabled() bool {
	return c.mcpClient != nil && c.mcpClient.enabled && c.config.MCP.Enabled
}

// ListMCPTools returns all available MCP tools
func (c *BaseClient) ListMCPTools() ([]mcp.ToolSummary, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	servers := c.mcpService.ListServers()
	var tools []mcp.ToolSummary
	for _, server := range servers {
		tools = append(tools, server.Tools...)
	}
	return tools, nil
}

// GetMCPToolsForLLM returns MCP tools in LLM-compatible format
func (c *BaseClient) GetMCPToolsForLLM() ([]map[string]interface{}, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	tools := c.mcpService.GetAvailableTools(context.Background())
	var llmTools []map[string]interface{}
	for _, tool := range tools {
		llmTool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  map[string]interface{}{},
			},
		}
		llmTools = append(llmTools, llmTool)
	}
	return llmTools, nil
}

// CallMCPTool calls an MCP tool with the specified arguments
func (c *BaseClient) CallMCPTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.ToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	return c.mcpService.CallTool(ctx, toolName, args)
}

// CallMCPToolWithTimeout calls an MCP tool with a timeout
func (c *BaseClient) CallMCPToolWithTimeout(toolName string, args map[string]interface{}, timeout time.Duration) (*mcp.ToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.mcpService.CallTool(ctx, toolName, args)
}

// GetMCPServerStatus returns the status of MCP servers
func (c *BaseClient) GetMCPServerStatus() (map[string]bool, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	servers := c.mcpService.ListServers()
	status := make(map[string]bool)
	for _, server := range servers {
		status[server.Name] = server.Running
	}
	return status, nil
}

// BatchCallMCPTools calls multiple MCP tools in parallel
func (c *BaseClient) BatchCallMCPTools(ctx context.Context, calls []ToolCall) ([]mcp.MCPToolResult, error) {
	if !c.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP is not enabled")
	}

	results := make([]mcp.MCPToolResult, len(calls))
	errChan := make(chan error, len(calls))

	for i, call := range calls {
		go func(index int, toolCall ToolCall) {
			result, err := c.mcpService.CallTool(ctx, toolCall.ToolName, toolCall.Args)
			if err != nil {
				errChan <- fmt.Errorf("tool %s failed: %w", toolCall.ToolName, err)
				return
			}
			if result != nil {
				results[index] = mcp.MCPToolResult{
					Success: result.Success,
					Data:    result.Data,
					Error:   result.Error,
				}
			}
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

// ToolCall represents a tool call request for lib usage
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args"`
}

// MCPChatOptions contains options for MCP chat
type MCPChatOptions struct {
	Temperature  float64
	MaxTokens    int
	ShowThinking bool
	AllowedTools []string
}

// ChatWithMCP performs a direct chat with MCP tools, bypassing RAG
func (c *BaseClient) ChatWithMCP(message string, opts *MCPChatOptions) (*MCPChatResponse, error) {
	if opts == nil {
		opts = &MCPChatOptions{
			Temperature:  0.7,
			MaxTokens:    30000,
			ShowThinking: true,
		}
	}

	ctx := context.Background()

	// Initialize LLM service
	_, llmService, _, err := providers.InitializeProviders(ctx, c.config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	// Create MCP tool manager
	mcpManager := mcp.NewMCPToolManager(&c.config.MCP)

	// Get available tools
	toolsMap := mcpManager.ListTools()

	// Convert to slice for filtering
	var tools []*mcp.MCPToolWrapper
	for _, tool := range toolsMap {
		tools = append(tools, tool)
	}

	// Filter tools if allowed-tools is specified
	if len(opts.AllowedTools) > 0 {
		allowedSet := make(map[string]bool)
		for _, tool := range opts.AllowedTools {
			allowedSet[tool] = true
		}

		var filteredTools []*mcp.MCPToolWrapper
		for _, tool := range tools {
			if allowedSet[tool.Name()] {
				filteredTools = append(filteredTools, tool)
			}
		}
		tools = filteredTools
	}

	// Build tool definitions for LLM
	var toolDefinitions []domain.ToolDefinition
	for _, tool := range tools {
		definition := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Schema(),
			},
		}
		toolDefinitions = append(toolDefinitions, definition)
	}

	// Prepare messages
	messages := []domain.Message{
		{
			Role:    "user",
			Content: message,
		},
	}

	// Generation options
	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		Think:       &opts.ShowThinking,
	}

	// Call LLM with tools
	result, err := llmService.GenerateWithTools(ctx, messages, toolDefinitions, genOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	response := &MCPChatResponse{
		Content:     result.Content,
		ToolCalls:   make([]MCPToolCallResult, 0),
		Thinking:    "",
		HasThinking: opts.ShowThinking,
	}

	// Handle tool calls
	if len(result.ToolCalls) > 0 {
		for _, toolCall := range result.ToolCalls {
			// Execute tool call via MCP
			mcpResult, err := mcpManager.CallTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments)

			toolCallResult := MCPToolCallResult{
				ToolName:  toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
				Success:   err == nil,
			}

			if err != nil {
				toolCallResult.Error = err.Error()
			} else {
				toolCallResult.Result = mcpResult.Data
				if mcpResult.Error != "" {
					toolCallResult.Error = mcpResult.Error
					toolCallResult.Success = false
				}
			}

			response.ToolCalls = append(response.ToolCalls, toolCallResult)
		}

		// If we have tool results, send follow-up request for final response
		if len(response.ToolCalls) > 0 {
			// Build tool result messages
			var toolResults []domain.Message
			for i, toolCall := range result.ToolCalls {
				var resultContent string
				if response.ToolCalls[i].Success {
					if response.ToolCalls[i].Result != nil {
						resultContent = fmt.Sprintf("%v", response.ToolCalls[i].Result)
					} else {
						resultContent = "Tool executed successfully"
					}
				} else {
					resultContent = fmt.Sprintf("Error: %s", response.ToolCalls[i].Error)
				}

				toolResults = append(toolResults, domain.Message{
					Role:       "tool",
					Content:    resultContent,
					ToolCallID: toolCall.ID,
				})
			}

			// Append assistant message with tool calls
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			})

			// Append tool results
			messages = append(messages, toolResults...)

			followUpResult, err := llmService.GenerateWithTools(ctx, messages, toolDefinitions, genOpts)
			if err == nil {
				response.FinalResponse = followUpResult.Content
			}
		}
	}

	return response, nil
}

// MCPChatResponse represents the response from MCP chat
type MCPChatResponse struct {
	Content       string              `json:"content"`
	FinalResponse string              `json:"final_response,omitempty"`
	ToolCalls     []MCPToolCallResult `json:"tool_calls,omitempty"`
	Thinking      string              `json:"thinking,omitempty"`
	HasThinking   bool                `json:"has_thinking"`
}

// MCPToolCallResult represents the result of an MCP tool call
type MCPToolCallResult struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Success   bool                   `json:"success"`
}
