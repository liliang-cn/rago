package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Service provides high-level MCP operations
type Service struct {
	manager       *Manager
	llm           domain.Generator
	mcpConfig     *Config
}

// NewService creates a new MCP service
func NewService(mcpConfig *Config, llm domain.Generator) (*Service, error) {
	// Load MCP servers from JSON
	if err := mcpConfig.LoadServersFromJSON(); err != nil {
		return nil, fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Create MCP manager
	manager := NewManager(mcpConfig)

	return &Service{
		manager:   manager,
		llm:       llm,
		mcpConfig: mcpConfig,
	}, nil
}

// StartServers starts MCP servers based on configuration
func (s *Service) StartServers(ctx context.Context, serverNames []string) error {
	if len(serverNames) == 0 {
		// Start all configured servers
		for _, server := range s.mcpConfig.LoadedServers {
			if server.AutoStart {
				serverNames = append(serverNames, server.Name)
			}
		}
	}

	for _, name := range serverNames {
		if _, err := s.manager.StartServer(ctx, name); err != nil {
			return fmt.Errorf("failed to start server %s: %w", name, err)
		}
	}

	return nil
}

// StopServer stops a specific MCP server
func (s *Service) StopServer(serverName string) error {
	return s.manager.StopServer(serverName)
}

// ListServers returns a list of configured servers and their status
func (s *Service) ListServers() []ServerStatus {
	var servers []ServerStatus
	
	for _, config := range s.mcpConfig.LoadedServers {
		client, exists := s.manager.GetClient(config.Name)
		
		status := ServerStatus{
			Name:        config.Name,
			Description: config.Description,
			Command:     strings.Join(config.Command, " "),
			Running:     exists && client != nil && client.IsConnected(),
		}
		
		if status.Running {
			tools := client.GetTools()
			status.ToolCount = len(tools)
			for _, tool := range tools {
				status.Tools = append(status.Tools, ToolSummary{
					Name:        tool.Name,
					Description: tool.Description,
					ServerName:  config.Name,
				})
			}
		}
		
		servers = append(servers, status)
	}
	
	return servers
}

// ServerStatus represents the status of an MCP server
type ServerStatus struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Command     string        `json:"command"`
	Running     bool          `json:"running"`
	ToolCount   int           `json:"tool_count"`
	Tools       []ToolSummary `json:"tools,omitempty"`
}


// GetAvailableTools returns all available tools from all running servers
func (s *Service) GetAvailableTools(ctx context.Context) []AgentToolInfo {
	return s.manager.GetAvailableTools(ctx)
}

// CallTool calls a specific tool by name
func (s *Service) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	return s.manager.CallTool(ctx, toolName, arguments)
}

// ChatOptions configures MCP-enabled chat
type ChatOptions struct {
	Model           string   // LLM model to use
	Temperature     float64  // Generation temperature
	MaxTokens       int      // Maximum tokens
	SystemPrompt    string   // System prompt
	AllowedServers  []string // Specific servers to use
	MaxToolCalls    int      // Maximum tool calls per message
	ShowToolCalls   bool     // Display tool calls in response
}

// DefaultChatOptions returns default chat options
func DefaultChatOptions() *ChatOptions {
	return &ChatOptions{
		Temperature:   0.7,
		MaxTokens:     2000,
		MaxToolCalls:  10,
		ShowToolCalls: true,
	}
}

// Chat performs an MCP-enabled chat interaction
func (s *Service) Chat(ctx context.Context, message string, opts *ChatOptions) (*ChatResponse, error) {
	if opts == nil {
		opts = DefaultChatOptions()
	}

	// Get available tools
	tools := s.manager.GetAvailableTools(ctx)
	
	// Filter tools by allowed servers if specified
	if len(opts.AllowedServers) > 0 {
		filtered := []AgentToolInfo{}
		for _, tool := range tools {
			for _, allowed := range opts.AllowedServers {
				if tool.ServerName == allowed {
					filtered = append(filtered, tool)
					break
				}
			}
		}
		tools = filtered
	}

	// Convert to tool definitions for LLM
	toolDefs := make([]domain.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		toolDefs = append(toolDefs, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  map[string]interface{}{}, // TODO: Add parameter schemas
			},
		})
	}

	// Create messages
	messages := []domain.Message{
		{
			Role:    "user",
			Content: message,
		},
	}

	// Add system prompt if provided
	if opts.SystemPrompt != "" {
		messages = append([]domain.Message{
			{
				Role:    "system",
				Content: opts.SystemPrompt,
			},
		}, messages...)
	}

	// Generate with tools
	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	result, err := s.llm.GenerateWithTools(ctx, messages, toolDefs, genOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Execute tool calls if any
	var executedCalls []ExecutedToolCall
	if len(result.ToolCalls) > 0 && opts.MaxToolCalls > 0 {
		for i, call := range result.ToolCalls {
			if i >= opts.MaxToolCalls {
				break
			}

			// Execute the tool
			toolResult, err := s.manager.CallTool(ctx, call.Function.Name, call.Function.Arguments)
			
			executed := ExecutedToolCall{
				ToolName:  call.Function.Name,
				Arguments: call.Function.Arguments,
				Success:   toolResult != nil && toolResult.Success,
			}
			
			if err != nil {
				executed.Error = err.Error()
			} else if toolResult != nil {
				executed.Result = toolResult.Data
			}
			
			executedCalls = append(executedCalls, executed)
		}
	}

	return &ChatResponse{
		Message:   result.Content,
		ToolCalls: executedCalls,
	}, nil
}

// ChatResponse represents a chat response with potential tool calls
type ChatResponse struct {
	Message   string              `json:"message"`
	ToolCalls []ExecutedToolCall  `json:"tool_calls,omitempty"`
}

// ExecutedToolCall represents a tool call that was executed
type ExecutedToolCall struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    interface{}            `json:"result,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// TestTool tests a specific tool with sample arguments
func (s *Service) TestTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	// Find the tool
	toolInfo, _, err := s.manager.FindToolProvider(ctx, toolName)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Call the tool
	result, err := s.manager.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// Add tool info to result for context
	if result != nil {
		if metadata, ok := result.Data.(map[string]interface{}); ok {
			metadata["_tool_info"] = map[string]string{
				"name":   toolInfo.Name,
				"server": toolInfo.ServerName,
			}
		}
	}

	return result, nil
}

// Close closes the MCP service and all connections
func (s *Service) Close() error {
	if s.manager != nil {
		return s.manager.Close()
	}
	return nil
}

// GetManager returns the underlying manager for advanced operations
func (s *Service) GetManager() *Manager {
	return s.manager
}