package mcp

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// TODO: Integrate with actual LLM for intelligent tool selection
	// This is a placeholder implementation that should be replaced with:
	// 1. LLM-based tool selection and argument extraction
	// 2. Multi-turn conversation handling
	// 3. Proper error handling and validation
	
	response := MCPChatResponse{
		Content:        "MCP chat functionality is available but requires LLM integration for intelligent tool usage.",
		FinalResponse:  "To use MCP tools effectively, this endpoint needs to be integrated with an LLM service that can determine which tools to call based on user messages.",
		HasThinking:    req.Options.ShowThinking,
		ToolCalls:      []MCPToolCallResult{},
	}

	// For now, return a helpful message about the available tools
	if len(toolsToUse) > 0 {
		var toolNames []string
		for _, tool := range toolsToUse {
			toolNames = append(toolNames, tool.Name)
		}
		response.Content = fmt.Sprintf("Available MCP tools: %v. Integration with LLM needed for automatic tool selection.", toolNames)
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