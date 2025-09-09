package mcp

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	mcppkg "github.com/liliang-cn/rago/v2/pkg/mcp"
)

// MCPHandler handles MCP-related HTTP requests
type MCPHandler struct {
	mcpService *mcppkg.MCPService
	mcpAPI     *mcppkg.MCPLibraryAPI
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(config *mcppkg.Config) (*MCPHandler, error) {
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
	}, nil
}

// ListTools returns all available MCP tools
func (h *MCPHandler) ListTools(c *gin.Context) {
	tools := h.mcpAPI.ListTools()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tools": tools,
			"count": len(tools),
		},
	})
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
	Timeout int            `json:"timeout"` // timeout in seconds for all calls
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"servers": statuses,
			"enabled": h.mcpService.IsEnabled(),
		},
	})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tools": tools,
			"count": len(tools),
		},
	})
}

// GetToolsByServer returns tools from a specific server
func (h *MCPHandler) GetToolsByServer(c *gin.Context) {
	serverName := c.Param("server")

	tools := h.mcpService.GetToolsByServer(serverName)
	if len(tools) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Server not found or no tools available",
		})
		return
	}

	// Convert to simplified format
	toolList := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolList = append(toolList, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"schema":      tool.Schema(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"server": serverName,
			"tools":  toolList,
			"count":  len(toolList),
		},
	})
}

// Close shuts down the MCP handler
func (h *MCPHandler) Close() error {
	return h.mcpAPI.Stop()
}
