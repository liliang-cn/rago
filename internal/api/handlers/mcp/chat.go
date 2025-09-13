package mcp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	mcppkg "github.com/liliang-cn/rago/v2/pkg/mcp"
)

// ChatWithMCPRequest represents a chat request with MCP tools
type ChatWithMCPRequest struct {
	Message      string                 `json:"message" binding:"required"`
	Options      *MCPChatOptions        `json:"options"`
	Context      map[string]interface{} `json:"context"`
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
	Content       string                   `json:"content"`
	FinalResponse string                   `json:"final_response,omitempty"`
	ToolCalls     []MCPToolCallResult      `json:"tool_calls,omitempty"`
	Thinking      string                   `json:"thinking,omitempty"`
	HasThinking   bool                     `json:"has_thinking"`
}

// MCPToolCallResult represents the result of an MCP tool call
type MCPToolCallResult struct {
	ToolName  string                 `json:"tool_name"`
	Args      map[string]interface{} `json:"args"`
	Result    interface{}            `json:"result"`
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

	// Get available tools
	availableTools := h.mcpAPI.ListTools()
	
	// Filter tools if allowed list is specified
	toolsToUse := availableTools
	if len(req.Options.AllowedTools) > 0 {
		filtered := []mcppkg.ToolSummary{}
		for _, tool := range availableTools {
			for _, allowed := range req.Options.AllowedTools {
				if tool.Name == allowed {
					filtered = append(filtered, tool)
					break
				}
			}
		}
		toolsToUse = filtered
	}

	// Process the message and determine if tools are needed
	// This is a simplified implementation - in reality, you'd use an LLM to determine tool usage
	response := MCPChatResponse{
		Content:     "Processing with MCP tools...",
		HasThinking: req.Options.ShowThinking,
	}

	// Example: Check if the message mentions files or web content
	if containsKeywords(req.Message, []string{"file", "read", "write", "fetch", "http"}) {
		// Determine which tool to use based on the message
		// This is simplified - you'd normally use LLM to determine the right tool and args
		
		if containsKeywords(req.Message, []string{"read", "file"}) && hasToolSummaryNamed(toolsToUse, "filesystem_read_file") {
			// Example tool call
			toolCall := MCPToolCallResult{
				ToolName: "filesystem_read_file",
				Args: map[string]interface{}{
					"path": "README.md", // This would be extracted from the message
				},
			}
			
			// Execute the tool
			result, err := h.mcpAPI.CallToolWithTimeout(toolCall.ToolName, toolCall.Args, 30*time.Second)
			if err != nil {
				toolCall.Success = false
				toolCall.Error = err.Error()
			} else {
				toolCall.Success = true
				toolCall.Result = result
			}
			
			response.ToolCalls = append(response.ToolCalls, toolCall)
		}
	}

	// Generate final response based on tool results
	if len(response.ToolCalls) > 0 {
		response.FinalResponse = "I've executed the requested tools. Here are the results."
	} else {
		response.FinalResponse = "I can help you with that, but no tools were needed for this request."
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
	Filters      map[string]interface{} `json:"filters"`
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
		"sources": []interface{}{},
		"tool_calls": []interface{}{},
	}

	// If tools are enabled, we could call them based on the query
	if req.EnableTools {
		// Example: If query asks about current time, use time tool
		if containsKeywords(req.Query, []string{"time", "date", "today"}) {
			if hasToolInAPI(h.mcpAPI, "time_get_current_time") {
				result, err := h.mcpAPI.CallToolWithTimeout("time_get_current_time", nil, 5*time.Second)
				if err == nil {
					response["tool_calls"] = append(response["tool_calls"].([]interface{}), gin.H{
						"tool": "time_get_current_time",
						"result": result,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// Helper functions
func containsKeywords(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if contains(text, keyword) {
			return true
		}
	}
	return false
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr || len(text) > 0 && len(substr) > 0 && indexOf(text, substr) >= 0)
}

func indexOf(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasToolSummaryNamed(tools []mcppkg.ToolSummary, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func hasToolInAPI(api *mcppkg.MCPLibraryAPI, name string) bool {
	tools := api.ListTools()
	return hasToolSummaryNamed(tools, name)
}