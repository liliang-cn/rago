// Package platform provides unified HTTP handlers for the RAGO AI platform
package platform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

// Handler provides unified platform API endpoints
type Handler struct {
	config *config.Config
}

// NewHandler creates a new platform handler
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		config: cfg,
	}
}

// createClient creates a client for the request
func (h *Handler) createClient() (*client.BaseClient, error) {
	// Create client with config
	return client.NewWithConfig(h.config)
}

// LLMGenerateRequest represents the request for LLM generation
type LLMGenerateRequest struct {
	Prompt      string  `json:"prompt" binding:"required"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// LLMGenerateResponse represents the response for LLM generation
type LLMGenerateResponse struct {
	Content string `json:"content"`
}

// LLMGenerate handles text generation requests
func (h *Handler) LLMGenerate(c *gin.Context) {
	var req LLMGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		err = rago.LLM.StreamWithOptions(
			ctx,
			req.Prompt,
			func(token string) {
				c.SSEvent("message", token)
				c.Writer.Flush()
			},
			&client.GenerateOptions{
				Temperature: req.Temperature,
				MaxTokens:   req.MaxTokens,
			},
		)
		if err != nil {
			c.SSEvent("error", err.Error())
		}
		c.SSEvent("done", "")
		return
	}

	response, err := rago.LLM.GenerateWithOptions(
		ctx,
		req.Prompt,
		&client.GenerateOptions{
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LLMGenerateResponse{Content: response})
}

// LLMChatRequest represents the request for LLM chat
type LLMChatRequest struct {
	Messages    []client.ChatMessage `json:"messages" binding:"required"`
	Temperature float64              `json:"temperature,omitempty"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	Stream      bool                 `json:"stream,omitempty"`
}

// LLMChat handles chat requests
func (h *Handler) LLMChat(c *gin.Context) {
	var req LLMChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		err = rago.LLM.ChatStreamWithOptions(
			ctx,
			req.Messages,
			func(token string) {
				c.SSEvent("message", token)
				c.Writer.Flush()
			},
			&client.GenerateOptions{
				Temperature: req.Temperature,
				MaxTokens:   req.MaxTokens,
			},
		)
		if err != nil {
			c.SSEvent("error", err.Error())
		}
		c.SSEvent("done", "")
		return
	}

	response, err := rago.LLM.ChatWithOptions(
		ctx,
		req.Messages,
		&client.GenerateOptions{
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LLMGenerateResponse{Content: response})
}

// RAGQueryRequest represents the request for RAG query
type RAGQueryRequest struct {
	Query       string            `json:"query" binding:"required"`
	TopK        int               `json:"top_k,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	ShowSources bool              `json:"show_sources,omitempty"`
	Filters     map[string]string `json:"filters,omitempty"`
}

// RAGQueryResponse represents the response for RAG query
type RAGQueryResponse struct {
	Answer  string                 `json:"answer"`
	Sources []client.SearchResult  `json:"sources,omitempty"`
}

// RAGQuery handles RAG query requests
func (h *Handler) RAGQuery(c *gin.Context) {
	var req RAGQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	response, err := rago.RAG.QueryWithOptions(
		ctx,
		req.Query,
		&client.QueryOptions{
			TopK:        req.TopK,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			ShowSources: req.ShowSources,
			Filters:     req.Filters,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := RAGQueryResponse{
		Answer: response.Answer,
	}
	if req.ShowSources {
		resp.Sources = response.Sources
	}

	c.JSON(http.StatusOK, resp)
}

// RAGIngestRequest represents the request for RAG ingestion
type RAGIngestRequest struct {
	Path            string            `json:"path" binding:"required"`
	ChunkSize       int               `json:"chunk_size,omitempty"`
	Overlap         int               `json:"overlap,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	RecursiveDir    bool              `json:"recursive_dir,omitempty"`
	ExcludePatterns []string          `json:"exclude_patterns,omitempty"`
	FileTypes       []string          `json:"file_types,omitempty"`
}

// RAGIngest handles document ingestion requests
func (h *Handler) RAGIngest(c *gin.Context) {
	var req RAGIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes for ingestion
	defer cancel()

	err = rago.RAG.IngestWithOptions(
		ctx,
		req.Path,
		&client.IngestOptions{
			ChunkSize:       req.ChunkSize,
			Overlap:         req.Overlap,
			Metadata:        req.Metadata,
			RecursiveDir:    req.RecursiveDir,
			ExcludePatterns: req.ExcludePatterns,
			FileTypes:       req.FileTypes,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Documents ingested successfully"})
}

// RAGSearchRequest represents the request for RAG search
type RAGSearchRequest struct {
	Query          string            `json:"query" binding:"required"`
	TopK           int               `json:"top_k,omitempty"`
	Threshold      float32           `json:"threshold,omitempty"`
	Filters        map[string]string `json:"filters,omitempty"`
	IncludeContent bool              `json:"include_content,omitempty"`
}

// RAGSearch handles semantic search requests
func (h *Handler) RAGSearch(c *gin.Context) {
	var req RAGSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := rago.RAG.SearchWithOptions(
		ctx,
		req.Query,
		&client.SearchOptions{
			TopK:           req.TopK,
			Threshold:      float64(req.Threshold),
			Filters:        req.Filters,
			IncludeContent: req.IncludeContent,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ToolsListResponse represents the response for tools listing
type ToolsListResponse struct {
	Tools []client.ToolInfo `json:"tools"`
}

// ToolsList handles tool listing requests
func (h *Handler) ToolsList(c *gin.Context) {
	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	// Enable MCP if needed
	if err := rago.EnableMCP(context.Background()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP not available", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tools, err := rago.Tools.ListWithOptions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ToolsListResponse{Tools: tools})
}

// ToolCallRequest represents the request for tool execution
type ToolCallRequest struct {
	Name string                 `json:"name" binding:"required"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// ToolCall handles tool execution requests
func (h *Handler) ToolCall(c *gin.Context) {
	var req ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	// Enable MCP if needed
	if err := rago.EnableMCP(context.Background()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP not available", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := rago.Tools.CallWithOptions(ctx, req.Name, req.Args)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// AgentRunRequest represents the request for agent execution
type AgentRunRequest struct {
	Task    string `json:"task" binding:"required"`
	Verbose bool   `json:"verbose,omitempty"`
}

// AgentRunResponse represents the response for agent execution
type AgentRunResponse struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Steps   []client.StepResult    `json:"steps,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// AgentRun handles agent task execution requests
func (h *Handler) AgentRun(c *gin.Context) {
	var req AgentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	// Enable MCP for agent tools
	if err := rago.EnableMCP(context.Background()); err != nil {
		// Continue anyway, agent might work without tools
		c.Header("X-Warning", fmt.Sprintf("MCP not fully enabled: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes for agent tasks
	defer cancel()

	result, err := rago.Agent.RunWithOptions(
		ctx,
		req.Task,
		&client.AgentOptions{
			Verbose: req.Verbose,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := AgentRunResponse{
		Success: result.Success,
		Output:  result.Output,
		Error:   result.Error,
	}
	if req.Verbose {
		resp.Steps = result.Steps
	}

	c.JSON(http.StatusOK, resp)
}

// AgentPlanRequest represents the request for agent planning
type AgentPlanRequest struct {
	Task string `json:"task" binding:"required"`
}

// AgentPlanResponse represents the response for agent planning
type AgentPlanResponse struct {
	Task  string              `json:"task"`
	Steps []client.PlanStep   `json:"steps"`
}

// AgentPlan handles agent planning requests
func (h *Handler) AgentPlan(c *gin.Context) {
	var req AgentPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rago, err := h.createClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create client: %v", err)})
		return
	}
	defer rago.Close()

	// Enable MCP for agent tools
	if err := rago.EnableMCP(context.Background()); err != nil {
		// Continue anyway, agent might work without tools
		c.Header("X-Warning", fmt.Sprintf("MCP not fully enabled: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	plan, err := rago.Agent.PlanWithOptions(ctx, req.Task, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, AgentPlanResponse{
		Task:  plan.Task,
		Steps: plan.Steps,
	})
}

// PlatformInfo represents platform information
type PlatformInfo struct {
	Version     string   `json:"version"`
	Components  []string `json:"components"`
	LLMProvider string   `json:"llm_provider"`
	RAGEnabled  bool     `json:"rag_enabled"`
	MCPEnabled  bool     `json:"mcp_enabled"`
}

// Info returns platform information
func (h *Handler) Info(c *gin.Context) {
	info := PlatformInfo{
		Version:    "2.0.0",
		Components: []string{"LLM", "RAG", "Tools", "Agent"},
		LLMProvider: h.config.Providers.DefaultLLM,
		RAGEnabled: true,
		MCPEnabled: h.config.MCP.Enabled,
	}

	c.JSON(http.StatusOK, info)
}