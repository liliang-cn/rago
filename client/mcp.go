package client

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/pkg/domain"
	"github.com/liliang-cn/rago/pkg/mcp"
	"github.com/liliang-cn/rago/pkg/utils"
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

// MCPChatOptions contains options for MCP chat
type MCPChatOptions struct {
	Temperature  float64
	MaxTokens    int
	ShowThinking bool
	AllowedTools []string
}

// ChatWithMCP performs a direct chat with MCP tools, bypassing RAG
func (c *Client) ChatWithMCP(message string, opts *MCPChatOptions) (*MCPChatResponse, error) {
	if opts == nil {
		opts = &MCPChatOptions{
			Temperature:  0.7,
			MaxTokens:    1000,
			ShowThinking: true,
		}
	}

	ctx := context.Background()

	// Initialize LLM service
	_, llmService, _, err := utils.InitializeProviders(ctx, c.config)
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
