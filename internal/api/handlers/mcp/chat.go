package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	mcppkg "github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// ChatWithMCPRequest represents a chat request with MCP tools
type ChatWithMCPRequest struct {
	Message        string                 `json:"message" binding:"required"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Options        *MCPChatOptions        `json:"options"`
	Context        map[string]any `json:"context"`
}

// MCPChatOptions represents options for MCP-enhanced chat
type MCPChatOptions struct {
	Temperature  float64  `json:"temperature"`
	MaxTokens    int      `json:"max_tokens"`
	ShowThinking bool     `json:"show_thinking"`
	AllowedTools []string `json:"allowed_tools"`
	MaxToolCalls int      `json:"max_tool_calls"`
}

// MCPChatResponse represents the response from MCP chat
type MCPChatResponse struct {
	Content        string                   `json:"content"`
	FinalResponse  string                   `json:"final_response,omitempty"`
	ToolCalls      []MCPToolCallResult      `json:"tool_calls,omitempty"`
	Thinking       string                   `json:"thinking,omitempty"`
	HasThinking    bool                     `json:"has_thinking"`
	ConversationID string                   `json:"conversation_id"`
}

// MCPToolCallResult represents the result of an MCP tool call
type MCPToolCallResult struct {
	ToolName  string                 `json:"tool_name"`
	Args      map[string]any `json:"args"`
	Result    any                    `json:"result"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
}

// ChatWithMCP handles chat requests that can use MCP tools
func (h *MCPHandler) ChatWithMCP(c *gin.Context) {
	var req ChatWithMCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set defaults
	if req.Options == nil {
		req.Options = &MCPChatOptions{
			Temperature:  0.7,
			MaxTokens:    500,
			ShowThinking: false,
			MaxToolCalls: 5,
		}
	}

	// Generate conversation ID if not provided
	if req.ConversationID == "" {
		req.ConversationID = uuid.New().String()
	}

	// Get available tools
	availableTools := h.mcpAPI.ListTools()
	
	// Filter tools if allowed list is specified
	toolsToUse := availableTools
	if len(req.Options.AllowedTools) > 0 {
		filtered := []mcppkg.ToolSummary{}
		for _, tool := range availableTools {
			if slices.Contains(req.Options.AllowedTools, tool.Name) {
				filtered = append(filtered, tool)
			}
		}
		toolsToUse = filtered
	}

	// Check if LLM service is available
	if h.llmService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "LLM service not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	// Load conversation history
	var messages []domain.Message
	if h.convStore != nil && req.ConversationID != "" {
		if conv, err := h.convStore.GetConversation(req.ConversationID); err == nil {
			messages = h.buildMessageHistory(conv)
		}
	}

	// Add current user message
	messages = append(messages, domain.Message{
		Role:    "user",
		Content: req.Message,
	})

	// Convert MCP tools to domain tool definitions
	var toolDefinitions []domain.ToolDefinition
	if len(toolsToUse) > 0 {
		toolDefinitions = h.convertMCPToolsToDefinitions(toolsToUse)
	}

	// Prepare system prompt
	systemPrompt := h.buildSystemPrompt(req.Options.ShowThinking)
	if len(messages) == 0 || messages[0].Role != "system" {
		messages = append([]domain.Message{{
			Role:    "system",
			Content: systemPrompt,
		}}, messages...)
	}

	// Generate response with tool support
	genOpts := &domain.GenerationOptions{
		Temperature: req.Options.Temperature,
		MaxTokens:   req.Options.MaxTokens,
		ToolChoice:  "auto", // Let the LLM decide when to use tools
	}

	var finalResponse string
	var toolCalls []MCPToolCallResult
	var thinking string
	var currentContent strings.Builder

	// Execute LLM with tools
	if len(toolDefinitions) > 0 {
		result, err := h.llmService.GenerateWithTools(ctx, messages, toolDefinitions, genOpts)
		if err != nil {
			log.Printf("LLM generation failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to generate response: " + err.Error(),
			})
			return
		}

		currentContent.WriteString(result.Content)

		// Process tool calls if any
		if len(result.ToolCalls) > 0 && req.Options.MaxToolCalls > 0 {
			toolCalls = h.executeToolCalls(ctx, result.ToolCalls, req.Options.MaxToolCalls)
			
			// Add tool results to messages
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			})

			for i, toolCall := range result.ToolCalls {
				if i < len(toolCalls) {
					toolResult := toolCalls[i]
					resultContent := fmt.Sprintf("Tool '%s' result: %v", toolResult.ToolName, toolResult.Result)
					if !toolResult.Success {
						resultContent = fmt.Sprintf("Tool '%s' error: %s", toolResult.ToolName, toolResult.Error)
					}
					messages = append(messages, domain.Message{
						Role:       "tool",
						Content:    resultContent,
						ToolCallID: toolCall.ID,
					})
				}
			}

			// Get final response after tool execution
			finalResult, err := h.llmService.GenerateWithTools(ctx, messages, nil, genOpts)
			if err != nil {
				finalResponse = currentContent.String()
			} else {
				finalResponse = finalResult.Content
			}
		} else {
			finalResponse = result.Content
		}
	} else {
		// No tools available, just generate response
		prompt := messages[len(messages)-1].Content
		if len(messages) > 1 {
			// Build conversation context
			var contextBuilder strings.Builder
			for _, msg := range messages[:len(messages)-1] {
				contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
			}
			contextBuilder.WriteString("user: ")
			contextBuilder.WriteString(prompt)
			prompt = contextBuilder.String()
		}
		
		resp, err := h.llmService.Generate(ctx, prompt, genOpts)
		if err != nil {
			log.Printf("LLM generation failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to generate response: " + err.Error(),
			})
			return
		}
		finalResponse = resp
	}

	// Extract thinking if present
	if req.Options.ShowThinking {
		finalResponse, thinking = h.extractThinking(finalResponse)
	}

	response := MCPChatResponse{
		Content:        finalResponse,
		FinalResponse:  finalResponse,
		ToolCalls:      toolCalls,
		Thinking:       thinking,
		HasThinking:    thinking != "",
		ConversationID: req.ConversationID,
	}

	// Set conversation ID in response
	response.ConversationID = req.ConversationID

	// Save conversation if conversation store is available
	if h.convStore != nil {
		go h.saveConversation(req.ConversationID, req.Message, response)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// QueryWithMCPRequest represents a RAG query request with MCP enhancement
type QueryWithMCPRequest struct {
	Query        string                 `json:"query" binding:"required"`
	TopK         int                    `json:"top_k"`
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	EnableTools  bool                   `json:"enable_tools"`
	AllowedTools []string               `json:"allowed_tools"`
	Filters      map[string]any `json:"filters"`
}

// QueryWithMCP handles RAG queries enhanced with MCP tools
func (h *MCPHandler) QueryWithMCP(c *gin.Context) {
	var req QueryWithMCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set defaults
	if req.TopK <= 0 {
		req.TopK = 5
	}
	if req.Temperature <= 0 {
		req.Temperature = 0.7
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 500
	}

	// Context would be used for actual tool execution
	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	// defer cancel()

	// This would integrate with the RAG processor and use MCP tools
	// For now, return a placeholder response
	response := gin.H{
		"answer": "This would be the RAG response enhanced with MCP tools.",
		"sources": []any{},
		"tool_calls": []any{},
	}

	// TODO: If tools are enabled, integrate with LLM to determine tool usage
	if req.EnableTools {
		// Placeholder: This should use an LLM to intelligently determine which tools to call
		response["message"] = "Tool usage requires LLM integration for intelligent tool selection"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// saveConversation saves the conversation to the store
func (h *MCPHandler) saveConversation(conversationID, userMessage string, response MCPChatResponse) {
	if h.convStore == nil {
		return
	}

	// Get existing conversation or create new one
	conv, err := h.convStore.GetConversation(conversationID)
	if err != nil {
		// Create new conversation
		conv = &store.Conversation{
			ID:       conversationID,
			Title:    truncateString(userMessage, 100),
			Messages: []store.ConversationMessage{},
			Metadata: map[string]any{
				"type": "mcp_chat",
			},
		}
	}

	now := time.Now().Unix()

	// Add user message
	userMsg := store.ConversationMessage{
		Role:      "user",
		Content:   userMessage,
		Timestamp: now,
	}
	conv.Messages = append(conv.Messages, userMsg)

	// Add assistant response
	assistantContent := response.FinalResponse
	if assistantContent == "" {
		assistantContent = response.Content
	}

	assistantMsg := store.ConversationMessage{
		Role:      "assistant",
		Content:   assistantContent,
		Thinking:  response.Thinking,
		Timestamp: now,
	}

	// Add tool call information to metadata if present
	if len(response.ToolCalls) > 0 {
		if assistantMsg.Sources == nil {
			assistantMsg.Sources = []store.RAGSource{}
		}
		for _, toolCall := range response.ToolCalls {
			source := store.RAGSource{
				ID:      toolCall.ToolName,
				Source:  "MCP Tool: " + toolCall.ToolName,
				Content: "Tool executed successfully",
				Score:   1.0,
			}
			if !toolCall.Success {
				source.Content = "Tool execution failed: " + toolCall.Error
				source.Score = 0.0
			}
			assistantMsg.Sources = append(assistantMsg.Sources, source)
		}
	}

	conv.Messages = append(conv.Messages, assistantMsg)

	// Save conversation
	if err := h.convStore.SaveConversation(conv); err != nil {
		// Log error but don't fail the request
		// In production, you might want to use a proper logger
		return
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}