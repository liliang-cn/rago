package mcp

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	mcppkg "github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// MCPHandler handles MCP-related HTTP requests
type MCPHandler struct {
	mcpService *mcppkg.MCPService
	mcpAPI     *mcppkg.MCPLibraryAPI
	convStore  *store.ConversationStore
	llmService *llm.Service
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(config *mcppkg.Config, convStore *store.ConversationStore) (*MCPHandler, error) {
	mcpAPI := mcppkg.NewMCPLibraryAPI(config)
	mcpService := mcppkg.NewMCPService(config)

	// Initialize MCP service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mcpAPI.Start(ctx); err != nil {
		return nil, err
	}

	return &MCPHandler{
		mcpService: mcpService,
		mcpAPI:     mcpAPI,
		convStore:  convStore,
	}, nil
}

// NewMCPHandlerWithLLM creates a new MCP handler with LLM support
func NewMCPHandlerWithLLM(config *mcppkg.Config, convStore *store.ConversationStore, llmService *llm.Service) (*MCPHandler, error) {
	handler, err := NewMCPHandler(config, convStore)
	if err != nil {
		return nil, err
	}
	handler.llmService = llmService
	return handler, nil
}

// ListTools returns all available MCP tools
func (h *MCPHandler) ListTools(c *gin.Context) {
	tools := h.mcpAPI.ListTools()

	// Ensure tools is never nil
	if tools == nil {
		tools = []mcppkg.ToolSummary{}
	}

	handlers.SendListResponse(c, tools, len(tools))
}

// GetTool returns details of a specific tool
func (h *MCPHandler) GetTool(c *gin.Context) {
	toolName := c.Param("name")

	tools := h.mcpService.GetAvailableTools()
	tool, exists := tools[toolName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Tool not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":        tool.Name(),
			"description": tool.Description(),
			"server":      tool.ServerName(),
			"schema":      tool.Schema(),
		},
	})
}

// CallToolRequest represents a tool call request
type CallToolRequest struct {
	ToolName string                 `json:"tool_name" binding:"required"`
	Args     map[string]interface{} `json:"args"`
	Timeout  int                    `json:"timeout"` // timeout in seconds
}

// CallTool executes an MCP tool
func (h *MCPHandler) CallTool(c *gin.Context) {
	var req CallToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set timeout
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// Call tool with timeout
	result, err := h.mcpAPI.CallToolWithTimeout(req.ToolName, req.Args, timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// BatchCallRequest represents a batch tool call request
type BatchCallRequest struct {
	Calls   []mcppkg.ToolCall `json:"calls" binding:"required"`
	Timeout int               `json:"timeout"` // timeout in seconds for all calls
}

// BatchCallTools executes multiple MCP tools in parallel
func (h *MCPHandler) BatchCallTools(c *gin.Context) {
	var req BatchCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set timeout
	timeout := 60 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute batch calls
	results, err := h.mcpAPI.BatchCall(ctx, req.Calls)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"results": results,
			"count":   len(results),
		},
	})
}

// GetServerStatus returns the status of all MCP servers
func (h *MCPHandler) GetServerStatus(c *gin.Context) {
	statuses := h.mcpService.GetServerStatus()

	// Ensure statuses is never nil
	if statuses == nil {
		statuses = make(map[string]bool)
	}

	handlers.SendListResponse(c, gin.H{
		"servers": statuses,
		"enabled": h.mcpService.IsEnabled(),
	}, len(statuses))
}

// StartServerRequest represents a server start request
type StartServerRequest struct {
	ServerName string `json:"server_name" binding:"required"`
}

// StartServer starts a specific MCP server
func (h *MCPHandler) StartServer(c *gin.Context) {
	var req StartServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.mcpService.StartServer(ctx, req.ServerName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server started successfully",
	})
}

// StopServerRequest represents a server stop request
type StopServerRequest struct {
	ServerName string `json:"server_name" binding:"required"`
}

// StopServer stops a specific MCP server
func (h *MCPHandler) StopServer(c *gin.Context) {
	var req StopServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.mcpService.StopServer(req.ServerName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server stopped successfully",
	})
}

// GetToolsForLLM returns tools formatted for LLM integration
func (h *MCPHandler) GetToolsForLLM(c *gin.Context) {
	tools := h.mcpAPI.GetToolsForLLMIntegration()

	// Ensure tools is never nil
	if tools == nil {
		tools = []map[string]interface{}{}
	}

	handlers.SendListResponse(c, tools, len(tools))
}

// GetToolsByServer returns tools from a specific server
func (h *MCPHandler) GetToolsByServer(c *gin.Context) {
	serverName := c.Param("server")

	tools := h.mcpService.GetToolsByServer(serverName)
	
	// Convert to simplified format
	toolList := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolList = append(toolList, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"schema":      tool.Schema(),
		})
	}

	// Return empty list instead of 404 for consistency
	handlers.SendListResponse(c, gin.H{
		"server": serverName,
		"tools":  toolList,
	}, len(toolList))
}

// Close shuts down the MCP handler
func (h *MCPHandler) Close() error {
	return h.mcpAPI.Stop()
}

// Helper methods for LLM-based chat

// buildMessageHistory converts conversation history to domain messages
func (h *MCPHandler) buildMessageHistory(conv *store.Conversation) []domain.Message {
	var messages []domain.Message
	for _, msg := range conv.Messages {
		dmsg := domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
		// Handle tool calls in history if needed
		if msg.Role == "assistant" && len(msg.Sources) > 0 {
			// Convert sources to tool calls if they represent tool results
			for _, source := range msg.Sources {
				if strings.HasPrefix(source.Source, "MCP Tool:") {
					// This was a tool call result
					// You might want to reconstruct tool calls here
				}
			}
		}
		messages = append(messages, dmsg)
	}
	return messages
}

// convertMCPToolsToDefinitions converts MCP tools to domain tool definitions
func (h *MCPHandler) convertMCPToolsToDefinitions(tools []mcppkg.ToolSummary) []domain.ToolDefinition {
	var definitions []domain.ToolDefinition
	for _, tool := range tools {
		// Get detailed tool info
		availableTools := h.mcpService.GetAvailableTools()
		if mcpTool, exists := availableTools[tool.Name]; exists {
			schema := mcpTool.Schema()
			
			// Convert schema to parameters
			parameters := make(map[string]interface{})
			// Schema is already a map[string]interface{}
			if inputSchema, ok := schema["inputSchema"]; ok {
				if inputParams, ok := inputSchema.(map[string]interface{}); ok {
					parameters = inputParams
				}
			}
			
			definitions = append(definitions, domain.ToolDefinition{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  parameters,
				},
			})
		}
	}
	return definitions
}

// buildSystemPrompt creates a system prompt for the LLM
func (h *MCPHandler) buildSystemPrompt(showThinking bool) string {
	var prompt strings.Builder
	prompt.WriteString("You are a helpful AI assistant with access to MCP (Model Context Protocol) tools.\n")
	prompt.WriteString("You can use these tools to help answer user questions and perform tasks.\n")
	prompt.WriteString("When you need to use a tool, make sure to provide the correct arguments.\n")
	prompt.WriteString("After using tools, provide a clear and helpful response based on the results.\n")
	
	if showThinking {
		prompt.WriteString("\n<thinking>\n")
		prompt.WriteString("Show your reasoning process when deciding which tools to use.\n")
		prompt.WriteString("</thinking>\n")
	}
	
	return prompt.String()
}

// executeToolCalls executes the tool calls requested by the LLM
func (h *MCPHandler) executeToolCalls(_ context.Context, toolCalls []domain.ToolCall, maxCalls int) []MCPToolCallResult {
	var results []MCPToolCallResult
	
	for i, toolCall := range toolCalls {
		if i >= maxCalls {
			break
		}
		
		start := time.Now()
		
		// Execute the tool
		result, err := h.mcpAPI.CallToolWithTimeout(toolCall.Function.Name, toolCall.Function.Arguments, 30*time.Second)
		
		toolResult := MCPToolCallResult{
			ToolName: toolCall.Function.Name,
			Args:     toolCall.Function.Arguments,
			Duration: time.Since(start).String(),
		}
		
		if err != nil {
			toolResult.Success = false
			toolResult.Error = err.Error()
			log.Printf("Tool call failed: %s - %v", toolCall.Function.Name, err)
		} else {
			toolResult.Success = true
			toolResult.Result = result
		}
		
		results = append(results, toolResult)
	}
	
	return results
}

// extractThinking extracts thinking tags from the response
func (h *MCPHandler) extractThinking(content string) (string, string) {
	thinkingStart := strings.Index(content, "<thinking>")
	thinkingEnd := strings.Index(content, "</thinking>")
	
	if thinkingStart >= 0 && thinkingEnd > thinkingStart {
		thinking := content[thinkingStart+10 : thinkingEnd]
		cleanContent := content[:thinkingStart] + content[thinkingEnd+11:]
		return strings.TrimSpace(cleanContent), strings.TrimSpace(thinking)
	}
	
	return content, ""
}
