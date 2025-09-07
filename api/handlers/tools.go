package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ToolsHandler handles tool-related HTTP requests using V3 client
type ToolsHandler struct {
	client *client.Client
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(c *client.Client) *ToolsHandler {
	return &ToolsHandler{
		client: c,
	}
}

// ListTools returns all available tools
func (h *ToolsHandler) ListTools(c *gin.Context) {
	if h.client.MCP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "MCP service not available",
		})
		return
	}

	tools := h.client.MCP().ListTools()

	c.JSON(http.StatusOK, gin.H{
		"tools": tools,
		"count": len(tools),
	})
}

// GetTool returns information about a specific tool
func (h *ToolsHandler) GetTool(c *gin.Context) {
	toolName := c.Param("name")
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tool name is required",
		})
		return
	}

	if h.client.MCP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "MCP service not available",
		})
		return
	}

	tool, err := h.client.MCP().GetTool(toolName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "tool not found",
		})
		return
	}

	c.JSON(http.StatusOK, tool)
}

// ExecuteTool executes a tool with given arguments
func (h *ToolsHandler) ExecuteTool(c *gin.Context) {
	toolName := c.Param("name")
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tool name is required",
		})
		return
	}

	var request struct {
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format: " + err.Error(),
		})
		return
	}

	if h.client.MCP() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "MCP service not available",
		})
		return
	}

	// Execute tool via MCP
	toolCall := core.ToolCall{
		Name:      toolName,
		Arguments: request.Arguments,
	}

	result, err := h.client.MCP().CallTool(c.Request.Context(), toolCall)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "execution failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tool":   toolName,
		"result": result,
	})
}
