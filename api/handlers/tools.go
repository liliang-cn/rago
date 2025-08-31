package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/pkg/tools"
)

// ToolsHandler handles tool-related HTTP requests
type ToolsHandler struct {
	registry *tools.Registry
	executor *tools.Executor
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(registry *tools.Registry, executor *tools.Executor) *ToolsHandler {
	return &ToolsHandler{
		registry: registry,
		executor: executor,
	}
}

// ListTools returns all available tools
func (h *ToolsHandler) ListTools(c *gin.Context) {
	enabled := c.Query("enabled")
	includeDisabled := enabled != "true"

	var toolInfos []tools.ToolInfo
	if includeDisabled {
		toolInfos = h.registry.List()
	} else {
		toolInfos = h.registry.ListEnabled()
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": toolInfos,
		"count": len(toolInfos),
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

	tool, exists := h.registry.Get(toolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "tool not found",
		})
		return
	}

	info := tools.ToolInfo{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  tool.Parameters(),
		Enabled:     h.registry.IsEnabled(toolName),
	}

	c.JSON(http.StatusOK, info)
}

// ExecuteTool directly executes a tool with given arguments
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
		Timeout   string                 `json:"timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format: " + err.Error(),
		})
		return
	}

	// Check if tool exists and is enabled
	tool, exists := h.registry.Get(toolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "tool not found",
		})
		return
	}

	if !h.registry.IsEnabled(toolName) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "tool is disabled",
		})
		return
	}

	// Validate arguments
	if err := tool.Validate(request.Arguments); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid arguments: " + err.Error(),
		})
		return
	}

	// Create execution context
	execCtx := &tools.ExecutionContext{
		RequestID: c.GetHeader("X-Request-ID"),
		UserID:    c.GetHeader("X-User-ID"),
		SessionID: c.GetHeader("X-Session-ID"),
	}

	// Execute tool
	result, err := h.executor.Execute(c.Request.Context(), execCtx, toolName, request.Arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "execution failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tool":    toolName,
		"success": result.Success,
		"data":    result.Data,
		"error":   result.Error,
	})
}

// GetToolStats returns tool execution statistics
func (h *ToolsHandler) GetToolStats(c *gin.Context) {
	stats := h.executor.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetRegistryStats returns registry statistics
func (h *ToolsHandler) GetRegistryStats(c *gin.Context) {
	stats := h.registry.Stats()
	c.JSON(http.StatusOK, stats)
}

// ListExecutions returns current executions
func (h *ToolsHandler) ListExecutions(c *gin.Context) {
	limit := 50 // default limit
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	executions := h.executor.ListExecutions()

	// Apply limit
	if len(executions) > limit {
		executions = executions[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"count":      len(executions),
	})
}

// GetExecution returns information about a specific execution
func (h *ToolsHandler) GetExecution(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "execution ID is required",
		})
		return
	}

	execution, exists := h.executor.GetExecutionInfo(executionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "execution not found",
		})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// CancelExecution cancels a running execution
func (h *ToolsHandler) CancelExecution(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "execution ID is required",
		})
		return
	}

	err := h.executor.CancelExecution(executionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "execution cancelled",
	})
}
