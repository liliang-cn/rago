package unified

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Handler handles unified multi-pillar operations
type Handler struct {
	client *client.Client
}

// NewHandler creates a new unified handler
func NewHandler(client *client.Client) *Handler {
	return &Handler{
		client: client,
	}
}

// Chat handles chat requests using LLM + RAG + MCP
// @Summary Chat with RAG and tools
// @Description Process chat messages with optional RAG context and tool usage
// @Tags Unified
// @Accept json
// @Produce json
// @Param request body core.ChatRequest true "Chat request"
// @Success 200 {object} core.ChatResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat [post]
func (h *Handler) Chat(c *gin.Context) {
	var req core.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.Chat(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StreamChat handles streaming chat requests
// @Summary Stream chat responses
// @Description Stream chat responses with RAG context and tool usage
// @Tags Unified
// @Accept json
// @Produce text/event-stream
// @Param request body core.ChatRequest true "Chat request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/chat/stream [post]
func (h *Handler) StreamChat(c *gin.Context) {
	var req core.ChatRequest
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

	// Stream chat responses
	err := h.client.StreamChat(c.Request.Context(), req, func(chunk core.StreamChunk) error {
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ProcessDocument handles document processing requests
// @Summary Process document
// @Description Process a document using RAG ingestion and agent workflows
// @Tags Unified
// @Accept json
// @Produce json
// @Param request body core.DocumentRequest true "Document processing request"
// @Success 200 {object} core.DocumentResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/process [post]
func (h *Handler) ProcessDocument(c *gin.Context) {
	var req core.DocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.ProcessDocument(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ExecuteTask handles complex task execution
// @Summary Execute task
// @Description Execute a complex task using agents, tools, and RAG
// @Tags Unified
// @Accept json
// @Produce json
// @Param request body core.TaskRequest true "Task execution request"
// @Success 200 {object} core.TaskResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/task [post]
func (h *Handler) ExecuteTask(c *gin.Context) {
	var req core.TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.ExecuteTask(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Query handles intelligent query requests
// @Summary Intelligent query
// @Description Execute an intelligent query across all pillars
// @Tags Unified
// @Accept json
// @Produce json
// @Param request body QueryRequest true "Query request"
// @Success 200 {object} QueryResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/query [post]
func (h *Handler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This is a unified query that can use all pillars
	// 1. Search RAG for relevant context
	// 2. Use LLM to generate response
	// 3. Execute tools if needed
	// 4. Run agent workflows if complex

	ctx := c.Request.Context()
	response := QueryResponse{
		Query: req.Query,
	}

	// Search RAG if enabled
	if req.UseRAG {
		searchReq := core.SearchRequest{
			Query: req.Query,
			Limit: 5,
		}
		searchResp, err := h.client.RAG().Search(ctx, searchReq)
		if err == nil && len(searchResp.Results) > 0 {
			response.Context = make([]string, 0, len(searchResp.Results))
			for _, result := range searchResp.Results {
				response.Context = append(response.Context, result.Content)
			}
		}
	}

	// Generate response with LLM
	genReq := core.GenerationRequest{
		Prompt:      req.Query,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	// Add context if available
	if len(response.Context) > 0 {
		contextStr := "Context:\n"
		for _, ctx := range response.Context {
			contextStr += ctx + "\n"
		}
		genReq.Prompt = contextStr + "\nQuery: " + req.Query
	}

	genResp, err := h.client.LLM().Generate(ctx, genReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response.Answer = genResp.Text
	response.Metadata = map[string]interface{}{
		"model":       genResp.Model,
		"usage":       genResp.Usage,
		"rag_enabled": req.UseRAG,
	}

	c.JSON(http.StatusOK, response)
}

// Analyze performs comprehensive analysis
// @Summary Analyze content
// @Description Perform comprehensive analysis using all pillars
// @Tags Unified
// @Accept json
// @Produce json
// @Param request body AnalyzeRequest true "Analysis request"
// @Success 200 {object} AnalyzeResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/analyze [post]
func (h *Handler) Analyze(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	response := AnalyzeResponse{
		Content: req.Content,
		Results: make(map[string]interface{}),
	}

	// Use different pillars based on analysis type
	for _, analysisType := range req.Types {
		switch analysisType {
		case "summary":
			// Use LLM to generate summary
			genReq := core.GenerationRequest{
				Prompt:    fmt.Sprintf("Summarize the following content:\n\n%s", req.Content),
				MaxTokens: 500,
			}
			genResp, err := h.client.LLM().Generate(ctx, genReq)
			if err == nil {
				response.Results["summary"] = genResp.Text
			}

		case "entities":
			// Use LLM to extract entities
			genReq := core.GenerationRequest{
				Prompt:    fmt.Sprintf("Extract all entities (people, places, organizations) from:\n\n%s", req.Content),
				MaxTokens: 300,
			}
			genResp, err := h.client.LLM().Generate(ctx, genReq)
			if err == nil {
				response.Results["entities"] = genResp.Text
			}

		case "sentiment":
			// Use LLM for sentiment analysis
			genReq := core.GenerationRequest{
				Prompt:    fmt.Sprintf("Analyze the sentiment of the following text:\n\n%s", req.Content),
				MaxTokens: 100,
			}
			genResp, err := h.client.LLM().Generate(ctx, genReq)
			if err == nil {
				response.Results["sentiment"] = genResp.Text
			}

		case "keywords":
			// Extract keywords
			genReq := core.GenerationRequest{
				Prompt:    fmt.Sprintf("Extract key terms and concepts from:\n\n%s", req.Content),
				MaxTokens: 200,
			}
			genResp, err := h.client.LLM().Generate(ctx, genReq)
			if err == nil {
				response.Results["keywords"] = genResp.Text
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// Request/Response types
type QueryRequest struct {
	Query       string  `json:"query" binding:"required"`
	UseRAG      bool    `json:"use_rag"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
}

type QueryResponse struct {
	Query    string                 `json:"query"`
	Answer   string                 `json:"answer"`
	Context  []string               `json:"context,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AnalyzeRequest struct {
	Content string   `json:"content" binding:"required"`
	Types   []string `json:"types" binding:"required"` // summary, entities, sentiment, keywords
}

type AnalyzeResponse struct {
	Content string                 `json:"content"`
	Results map[string]interface{} `json:"results"`
}