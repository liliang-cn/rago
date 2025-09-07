package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Handler handles MCP-related HTTP requests
type Handler struct {
	client *client.Client
}

// NewHandler creates a new MCP handler
func NewHandler(client *client.Client) *Handler {
	return &Handler{
		client: client,
	}
}

// ListServers lists all MCP servers
// @Summary List MCP servers
// @Description Get a list of all registered MCP servers
// @Tags MCP
// @Produce json
// @Success 200 {array} core.ServerInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/servers [get]
func (h *Handler) ListServers(c *gin.Context) {
	servers := h.client.MCP().ListServers()
	c.JSON(http.StatusOK, servers)
}

// RegisterServer registers a new MCP server
// @Summary Register MCP server
// @Description Register a new MCP server
// @Tags MCP
// @Accept json
// @Produce json
// @Param request body core.ServerConfig true "Server configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/servers [post]
func (h *Handler) RegisterServer(c *gin.Context) {
	var config core.ServerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.client.MCP().RegisterServer(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "server registered successfully"})
}

// UnregisterServer unregisters an MCP server
// @Summary Unregister MCP server
// @Description Remove an MCP server by name
// @Tags MCP
// @Param name path string true "Server name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/servers/{name} [delete]
func (h *Handler) UnregisterServer(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server name required"})
		return
	}

	err := h.client.MCP().UnregisterServer(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "server unregistered successfully"})
}

// GetServerHealth gets health status of a specific server
// @Summary Get server health
// @Description Get health status of a specific MCP server
// @Tags MCP
// @Param name path string true "Server name"
// @Produce json
// @Success 200 {object} core.HealthStatus
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/servers/{name}/health [get]
func (h *Handler) GetServerHealth(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server name required"})
		return
	}

	health := h.client.MCP().GetServerHealth(name)
	c.JSON(http.StatusOK, health)
}

// ListTools lists all available tools
// @Summary List tools
// @Description Get a list of all available MCP tools
// @Tags MCP
// @Produce json
// @Success 200 {array} core.ToolInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools [get]
func (h *Handler) ListTools(c *gin.Context) {
	tools := h.client.MCP().ListTools()
	c.JSON(http.StatusOK, tools)
}

// GetTool gets information about a specific tool
// @Summary Get tool info
// @Description Get detailed information about a specific tool
// @Tags MCP
// @Param name path string true "Tool name"
// @Produce json
// @Success 200 {object} core.ToolInfo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools/{name} [get]
func (h *Handler) GetTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name required"})
		return
	}

	tool, err := h.client.MCP().GetTool(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tool)
}

// CallTool executes a tool
// @Summary Call a tool
// @Description Execute an MCP tool with the provided parameters
// @Tags MCP
// @Accept json
// @Produce json
// @Param name path string true "Tool name"
// @Param request body core.ToolCallRequest true "Tool call request"
// @Success 200 {object} core.ToolCallResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools/{name}/call [post]
func (h *Handler) CallTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name required"})
		return
	}

	var req core.ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set tool name from path
	req.ToolName = name

	resp, err := h.client.MCP().CallTool(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CallToolAsync executes a tool asynchronously
// @Summary Call tool async
// @Description Execute an MCP tool asynchronously and return immediately
// @Tags MCP
// @Accept json
// @Produce json
// @Param name path string true "Tool name"
// @Param request body core.ToolCallRequest true "Tool call request"
// @Success 202 {object} AsyncResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools/{name}/call-async [post]
func (h *Handler) CallToolAsync(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name required"})
		return
	}

	var req core.ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set tool name from path
	req.ToolName = name

	// Create a channel to receive the response
	respChan, err := h.client.MCP().CallToolAsync(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate a task ID
	taskID := fmt.Sprintf("task_%d", time.Now().Unix())

	// Store the channel for later retrieval (in production, use a proper task manager)
	// For now, we'll just return the task ID
	
	// Start a goroutine to handle the async response
	go func() {
		select {
		case resp := <-respChan:
			// In production, store the response for later retrieval
			_ = resp
		case <-time.After(5 * time.Minute):
			// Timeout after 5 minutes
		}
	}()

	c.JSON(http.StatusAccepted, AsyncResponse{
		TaskID:  taskID,
		Status:  "processing",
		Message: "Tool execution started",
	})
}

// CallToolsBatch executes multiple tools in batch
// @Summary Batch call tools
// @Description Execute multiple MCP tools in a single batch operation
// @Tags MCP
// @Accept json
// @Produce json
// @Param request body BatchToolCallRequest true "Batch tool call request"
// @Success 200 {array} core.ToolCallResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools/batch [post]
func (h *Handler) CallToolsBatch(c *gin.Context) {
	var req BatchToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	responses, err := h.client.MCP().CallToolsBatch(c.Request.Context(), req.Requests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, responses)
}

// StreamToolExecution streams tool execution results
// @Summary Stream tool execution
// @Description Stream real-time tool execution results using Server-Sent Events
// @Tags MCP
// @Accept json
// @Produce text/event-stream
// @Param name path string true "Tool name"
// @Param request body core.ToolCallRequest true "Tool call request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mcp/tools/{name}/stream [post]
func (h *Handler) StreamToolExecution(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool name required"})
		return
	}

	var req core.ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Set tool name from path
	req.ToolName = name

	// Execute tool asynchronously
	respChan, err := h.client.MCP().CallToolAsync(c.Request.Context(), req)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Stream progress updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case resp := <-respChan:
			// Send the final response
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "event: result\ndata: %s\n\n", data)
			flusher.Flush()
			
			// Send done event
			fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			flusher.Flush()
			return

		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"status\":\"processing\"}\n\n")
			flusher.Flush()

		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

// Request/Response types
type AsyncResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type BatchToolCallRequest struct {
	Requests []core.ToolCallRequest `json:"requests" binding:"required,min=1"`
}