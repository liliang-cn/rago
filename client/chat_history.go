package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// ConversationHistory manages conversation state for multi-round chats
type ConversationHistory struct {
	Messages   []domain.Message
	MaxHistory int
}

// NewConversationHistory creates a new conversation history
func NewConversationHistory(systemPrompt string, maxHistory int) *ConversationHistory {
	if maxHistory <= 0 {
		maxHistory = 50
	}

	return &ConversationHistory{
		Messages: []domain.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
		},
		MaxHistory: maxHistory,
	}
}

// AddUserMessage adds a user message to the history
func (h *ConversationHistory) AddUserMessage(content string) {
	h.Messages = append(h.Messages, domain.Message{
		Role:    "user",
		Content: content,
	})
	h.trimHistory()
}

// AddAssistantMessage adds an assistant message to the history
func (h *ConversationHistory) AddAssistantMessage(content string, toolCalls []domain.ToolCall) {
	msg := domain.Message{
		Role:    "assistant",
		Content: content,
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	h.Messages = append(h.Messages, msg)
	h.trimHistory()
}

// AddToolMessage adds a tool result message to the history
func (h *ConversationHistory) AddToolMessage(content string, toolCallID string) {
	h.Messages = append(h.Messages, domain.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	})
	h.trimHistory()
}

// trimHistory keeps the conversation history within MaxHistory limit
func (h *ConversationHistory) trimHistory() {
	if len(h.Messages) > h.MaxHistory {
		// Keep system message and trim old messages
		h.Messages = append(h.Messages[:1], h.Messages[len(h.Messages)-h.MaxHistory+1:]...)
	}
}

// Clear resets the conversation history, keeping only the system prompt
func (h *ConversationHistory) Clear() {
	if len(h.Messages) > 0 && h.Messages[0].Role == "system" {
		h.Messages = h.Messages[:1]
	} else {
		h.Messages = nil
	}
}

// ChatWithHistory performs a chat with conversation history
func (c *Client) ChatWithHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions) (string, error) {
	if c.llm == nil {
		return "", fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		}
	}

	// Add user message to history
	history.AddUserMessage(message)

	// Generate response using full conversation history
	// Use GenerateWithTools with empty tools to support message history
	result, err := c.llm.GenerateWithTools(ctx, history.Messages, nil, opts)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	response := result.Content

	// Add assistant response to history
	history.AddAssistantMessage(response, nil)

	return response, nil
}

// StreamChatWithHistory performs streaming chat with conversation history
func (c *Client) StreamChatWithHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		}
	}

	// Add user message to history
	history.AddUserMessage(message)

	// Collect the full response for history
	var fullResponse strings.Builder
	wrappedCallback := func(chunk string, toolCalls []domain.ToolCall) error {
		fullResponse.WriteString(chunk)
		callback(chunk)
		return nil
	}

	// Stream response using StreamWithTools
	err := c.llm.StreamWithTools(ctx, history.Messages, nil, opts, wrappedCallback)
	if err != nil {
		return fmt.Errorf("streaming failed: %w", err)
	}

	// Add complete response to history
	history.AddAssistantMessage(fullResponse.String(), nil)

	return nil
}

// ChatWithRAGHistory performs a RAG-enhanced chat with conversation history
func (c *Client) ChatWithRAGHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions) (string, []SearchResult, error) {
	if c.llm == nil {
		return "", nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		}
	}

	// Search knowledge base
	searchOpts := &SearchOptions{
		TopK:            5,
		IncludeMetadata: true,
	}

	searchResults, err := c.Search(ctx, message, searchOpts)
	if err != nil {
		// Log warning but continue without context
		fmt.Printf("Warning: Search failed: %v\n", err)
		searchResults = []SearchResult{}
	}

	// Build augmented message with context
	augmentedMessage := message
	if len(searchResults) > 0 {
		context := "\nRelevant information from knowledge base:\n"
		for i, result := range searchResults {
			context += fmt.Sprintf("\n[%d] %s (Score: %.2f)\n", i+1, result.Content, result.Score)
		}
		augmentedMessage = fmt.Sprintf("Context:%s\n\nQuestion: %s", context, message)
	}

	// Add augmented message to history
	history.AddUserMessage(augmentedMessage)

	// Generate response with full history using GenerateWithTools
	result, err := c.llm.GenerateWithTools(ctx, history.Messages, nil, opts)
	if err != nil {
		return "", searchResults, fmt.Errorf("generation failed: %w", err)
	}

	response := result.Content

	// Add assistant response to history
	history.AddAssistantMessage(response, nil)

	return response, searchResults, nil
}

// StreamChatWithRAGHistory performs streaming RAG-enhanced chat with conversation history
func (c *Client) StreamChatWithRAGHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions, callback func(string)) ([]SearchResult, error) {
	if c.llm == nil {
		return nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		}
	}

	// Search knowledge base
	searchOpts := &SearchOptions{
		TopK:            5,
		IncludeMetadata: true,
	}

	searchResults, err := c.Search(ctx, message, searchOpts)
	if err != nil {
		fmt.Printf("Warning: Search failed: %v\n", err)
		searchResults = []SearchResult{}
	}

	// Build augmented message with context
	augmentedMessage := message
	if len(searchResults) > 0 {
		context := "\nRelevant information from knowledge base:\n"
		for i, result := range searchResults {
			context += fmt.Sprintf("\n[%d] %s (Score: %.2f)\n", i+1, result.Content, result.Score)
		}
		augmentedMessage = fmt.Sprintf("Context:%s\n\nQuestion: %s", context, message)
	}

	// Add augmented message to history
	history.AddUserMessage(augmentedMessage)

	// Collect the full response for history
	var fullResponse strings.Builder
	wrappedCallback := func(chunk string, toolCalls []domain.ToolCall) error {
		fullResponse.WriteString(chunk)
		callback(chunk)
		return nil
	}

	// Stream response using StreamWithTools
	err = c.llm.StreamWithTools(ctx, history.Messages, nil, opts, wrappedCallback)
	if err != nil {
		return searchResults, fmt.Errorf("streaming failed: %w", err)
	}

	// Add complete response to history
	history.AddAssistantMessage(fullResponse.String(), nil)

	return searchResults, nil
}

// ChatWithMCPHistory performs a chat with MCP tools and conversation history
func (c *Client) ChatWithMCPHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions) (string, []domain.ToolCall, error) {
	if c.llm == nil {
		return "", nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to MCP tools. Use the available tools to help answer questions and complete tasks.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
			ToolChoice:  "auto",
		}
	}

	// Check if MCP is enabled and get tools
	var tools []domain.ToolDefinition
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		tools = c.mcpClient.GetToolDefinitions(ctx)
	}

	if len(tools) == 0 {
		// Fall back to regular chat if no tools available
		response, err := c.ChatWithHistory(ctx, message, history, opts)
		return response, nil, err
	}

	// Add user message to history
	history.AddUserMessage(message)

	// Generate response with tools
	result, err := c.llm.GenerateWithTools(ctx, history.Messages, tools, opts)
	if err != nil {
		return "", nil, fmt.Errorf("generation with tools failed: %w", err)
	}

	// Handle tool calls if present
	if len(result.ToolCalls) > 0 {
		// Add assistant message with tool calls
		history.AddAssistantMessage(result.Content, result.ToolCalls)

		// Execute each tool call
		for _, tc := range result.ToolCalls {
			toolResult, err := c.mcpClient.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)

			var toolResultContent string
			if err != nil {
				toolResultContent = fmt.Sprintf("Error: %v", err)
			} else if toolResult.Success {
				toolResultContent = fmt.Sprintf("%v", toolResult.Data)
			} else {
				toolResultContent = fmt.Sprintf("Error: %s", toolResult.Error)
			}

			// Add tool result to history
			history.AddToolMessage(toolResultContent, tc.ID)
		}

		// Get final response after tool execution
		finalResult, err := c.llm.GenerateWithTools(ctx, history.Messages, tools, opts)
		if err != nil {
			return result.Content, result.ToolCalls, nil // Return partial result
		}

		// Add final assistant response
		history.AddAssistantMessage(finalResult.Content, nil)

		return finalResult.Content, result.ToolCalls, nil
	}

	// No tool calls, just add response to history
	history.AddAssistantMessage(result.Content, nil)

	return result.Content, nil, nil
}

// StreamChatWithMCPHistory performs streaming chat with MCP tools and conversation history
func (c *Client) StreamChatWithMCPHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions, callback func(string)) ([]domain.ToolCall, error) {
	if c.llm == nil {
		return nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to MCP tools. Use the available tools to help answer questions and complete tasks.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
			ToolChoice:  "auto",
		}
	}

	// Check if MCP is enabled and get tools
	var tools []domain.ToolDefinition
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		tools = c.mcpClient.GetToolDefinitions(ctx)
	}

	if len(tools) == 0 {
		// Fall back to regular streaming chat if no tools available
		err := c.StreamChatWithHistory(ctx, message, history, opts, callback)
		return nil, err
	}

	// Add user message to history
	history.AddUserMessage(message)

	// Collect the full response and tool calls for history
	var fullResponse strings.Builder
	var collectedToolCalls []domain.ToolCall
	wrappedCallback := func(chunk string, toolCalls []domain.ToolCall) error {
		fullResponse.WriteString(chunk)
		if len(toolCalls) > 0 {
			collectedToolCalls = toolCalls
		}
		callback(chunk)
		return nil
	}

	// Stream response with tools
	err := c.llm.StreamWithTools(ctx, history.Messages, tools, opts, wrappedCallback)
	if err != nil {
		return nil, fmt.Errorf("streaming with tools failed: %w", err)
	}

	// Handle tool calls if present
	if len(collectedToolCalls) > 0 {
		// Add assistant message with tool calls
		history.AddAssistantMessage(fullResponse.String(), collectedToolCalls)

		// Execute each tool call
		for _, tc := range collectedToolCalls {
			toolResult, err := c.mcpClient.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)

			var toolResultContent string
			if err != nil {
				toolResultContent = fmt.Sprintf("Error: %v", err)
			} else if toolResult.Success {
				toolResultContent = fmt.Sprintf("%v", toolResult.Data)
			} else {
				toolResultContent = fmt.Sprintf("Error: %s", toolResult.Error)
			}

			// Add tool result to history
			history.AddToolMessage(toolResultContent, tc.ID)
		}

		// Get final response after tool execution - need to stream this too
		fullResponse.Reset()
		finalCallback := func(chunk string, toolCalls []domain.ToolCall) error {
			fullResponse.WriteString(chunk)
			callback(chunk)
			return nil
		}

		err = c.llm.StreamWithTools(ctx, history.Messages, tools, opts, finalCallback)
		if err != nil {
			return collectedToolCalls, nil // Return partial result
		}

		// Add final assistant response
		history.AddAssistantMessage(fullResponse.String(), nil)

		return collectedToolCalls, nil
	}

	// No tool calls, just add response to history
	history.AddAssistantMessage(fullResponse.String(), nil)

	return nil, nil
}

// ChatWithRAGAndMCPHistory performs a RAG-enhanced chat with MCP tools and conversation history
func (c *Client) ChatWithRAGAndMCPHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions) (string, []SearchResult, []domain.ToolCall, error) {
	if c.llm == nil {
		return "", nil, nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to both a knowledge base and MCP tools. Use the provided context and available tools to answer questions accurately and complete tasks.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
			ToolChoice:  "auto",
		}
	}

	// Search knowledge base first
	searchOpts := &SearchOptions{
		TopK:            5,
		IncludeMetadata: true,
	}

	searchResults, err := c.Search(ctx, message, searchOpts)
	if err != nil {
		fmt.Printf("Warning: Search failed: %v\n", err)
		searchResults = []SearchResult{}
	}

	// Build augmented message with context
	augmentedMessage := message
	if len(searchResults) > 0 {
		context := "\nRelevant information from knowledge base:\n"
		for i, result := range searchResults {
			context += fmt.Sprintf("\n[%d] %s (Score: %.2f)\n", i+1, result.Content, result.Score)
		}
		augmentedMessage = fmt.Sprintf("Context:%s\n\nQuestion: %s", context, message)
	}

	// Check if MCP is enabled and get tools
	var tools []domain.ToolDefinition
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		tools = c.mcpClient.GetToolDefinitions(ctx)
	}

	// Add augmented message to history
	history.AddUserMessage(augmentedMessage)

	// If no tools, just do RAG with message history
	if len(tools) == 0 {
		result, err := c.llm.GenerateWithTools(ctx, history.Messages, nil, opts)
		if err != nil {
			return "", searchResults, nil, fmt.Errorf("generation failed: %w", err)
		}
		history.AddAssistantMessage(result.Content, nil)
		return result.Content, searchResults, nil, nil
	}

	// Generate response with tools
	result, err := c.llm.GenerateWithTools(ctx, history.Messages, tools, opts)
	if err != nil {
		return "", searchResults, nil, fmt.Errorf("generation with tools failed: %w", err)
	}

	// Handle tool calls if present
	if len(result.ToolCalls) > 0 {
		// Add assistant message with tool calls
		history.AddAssistantMessage(result.Content, result.ToolCalls)

		// Execute each tool call
		for _, tc := range result.ToolCalls {
			toolResult, err := c.mcpClient.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)

			var toolResultContent string
			if err != nil {
				toolResultContent = fmt.Sprintf("Error: %v", err)
			} else if toolResult.Success {
				toolResultContent = fmt.Sprintf("%v", toolResult.Data)
			} else {
				toolResultContent = fmt.Sprintf("Error: %s", toolResult.Error)
			}

			// Add tool result to history
			history.AddToolMessage(toolResultContent, tc.ID)
		}

		// Get final response after tool execution
		finalResult, err := c.llm.GenerateWithTools(ctx, history.Messages, tools, opts)
		if err != nil {
			return result.Content, searchResults, result.ToolCalls, nil // Return partial result
		}

		// Add final assistant response
		history.AddAssistantMessage(finalResult.Content, nil)

		return finalResult.Content, searchResults, result.ToolCalls, nil
	}

	// No tool calls, just add response to history
	history.AddAssistantMessage(result.Content, nil)

	return result.Content, searchResults, nil, nil
}

// StreamChatWithRAGAndMCPHistory performs streaming RAG-enhanced chat with MCP tools and conversation history
func (c *Client) StreamChatWithRAGAndMCPHistory(ctx context.Context, message string, history *ConversationHistory, opts *domain.GenerationOptions, callback func(string)) ([]SearchResult, []domain.ToolCall, error) {
	if c.llm == nil {
		return nil, nil, fmt.Errorf("LLM service not initialized")
	}

	if history == nil {
		history = NewConversationHistory("You are a helpful assistant with access to both a knowledge base and MCP tools. Use the provided context and available tools to answer questions accurately and complete tasks.", 50)
	}

	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
			ToolChoice:  "auto",
		}
	}

	// Search knowledge base first
	searchOpts := &SearchOptions{
		TopK:            5,
		IncludeMetadata: true,
	}

	searchResults, err := c.Search(ctx, message, searchOpts)
	if err != nil {
		fmt.Printf("Warning: Search failed: %v\n", err)
		searchResults = []SearchResult{}
	}

	// Build augmented message with context
	augmentedMessage := message
	if len(searchResults) > 0 {
		context := "\nRelevant information from knowledge base:\n"
		for i, result := range searchResults {
			context += fmt.Sprintf("\n[%d] %s (Score: %.2f)\n", i+1, result.Content, result.Score)
		}
		augmentedMessage = fmt.Sprintf("Context:%s\n\nQuestion: %s", context, message)
	}

	// Check if MCP is enabled and get tools
	var tools []domain.ToolDefinition
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		tools = c.mcpClient.GetToolDefinitions(ctx)
	}

	// Add augmented message to history
	history.AddUserMessage(augmentedMessage)

	// If no tools, just do streaming RAG with message history
	if len(tools) == 0 {
		var fullResponse strings.Builder
		wrappedCallback := func(chunk string, toolCalls []domain.ToolCall) error {
			fullResponse.WriteString(chunk)
			callback(chunk)
			return nil
		}

		err := c.llm.StreamWithTools(ctx, history.Messages, nil, opts, wrappedCallback)
		if err != nil {
			return searchResults, nil, fmt.Errorf("streaming failed: %w", err)
		}
		history.AddAssistantMessage(fullResponse.String(), nil)
		return searchResults, nil, nil
	}

	// Collect the full response and tool calls for history
	var fullResponse strings.Builder
	var collectedToolCalls []domain.ToolCall
	wrappedCallback := func(chunk string, toolCalls []domain.ToolCall) error {
		fullResponse.WriteString(chunk)
		if len(toolCalls) > 0 {
			collectedToolCalls = toolCalls
		}
		callback(chunk)
		return nil
	}

	// Stream response with tools
	err = c.llm.StreamWithTools(ctx, history.Messages, tools, opts, wrappedCallback)
	if err != nil {
		return searchResults, nil, fmt.Errorf("streaming with tools failed: %w", err)
	}

	// Handle tool calls if present
	if len(collectedToolCalls) > 0 {
		// Add assistant message with tool calls
		history.AddAssistantMessage(fullResponse.String(), collectedToolCalls)

		// Execute each tool call
		for _, tc := range collectedToolCalls {
			toolResult, err := c.mcpClient.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)

			var toolResultContent string
			if err != nil {
				toolResultContent = fmt.Sprintf("Error: %v", err)
			} else if toolResult.Success {
				toolResultContent = fmt.Sprintf("%v", toolResult.Data)
			} else {
				toolResultContent = fmt.Sprintf("Error: %s", toolResult.Error)
			}

			// Add tool result to history
			history.AddToolMessage(toolResultContent, tc.ID)
		}

		// Get final response after tool execution - need to stream this too
		fullResponse.Reset()
		finalCallback := func(chunk string, toolCalls []domain.ToolCall) error {
			fullResponse.WriteString(chunk)
			callback(chunk)
			return nil
		}

		err = c.llm.StreamWithTools(ctx, history.Messages, tools, opts, finalCallback)
		if err != nil {
			return searchResults, collectedToolCalls, nil // Return partial result
		}

		// Add final assistant response
		history.AddAssistantMessage(fullResponse.String(), nil)

		return searchResults, collectedToolCalls, nil
	}

	// No tool calls, just add response to history
	history.AddAssistantMessage(fullResponse.String(), nil)

	return searchResults, nil, nil
}
