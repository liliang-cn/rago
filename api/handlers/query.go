package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/pkg/domain"
	"github.com/liliang-cn/rago/pkg/processor"
)

type QueryHandler struct {
	processor *processor.Service
}

func NewQueryHandler(p *processor.Service) *QueryHandler {
	return &QueryHandler{processor: p}
}

func (h *QueryHandler) Handle(c *gin.Context) {
	var req domain.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	if req.Stream {
		h.handleStream(c, req)
		return
	}

	// Use QueryWithTools if tools are enabled and requested
	var resp domain.QueryResponse
	var err error
	if req.ToolsEnabled {
		resp, err = h.processor.QueryWithTools(c.Request.Context(), req)
	} else {
		resp, err = h.processor.Query(c.Request.Context(), req)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process query: " + err.Error(),
		})
		return
	}

	// 确保sources总是数组
	if resp.Sources == nil {
		resp.Sources = []domain.Chunk{}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *QueryHandler) SearchOnly(c *gin.Context) {
	var req struct {
		Query   string                 `json:"query"`
		TopK    int                    `json:"top_k"`
		Filters map[string]interface{} `json:"filters,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	ctx := c.Request.Context()

	// Create a simple search request using the processor
	queryReq := domain.QueryRequest{
		Query:   req.Query,
		TopK:    req.TopK,
		Filters: req.Filters,
	}

	// Use processor's search functionality - we'll need to add a search-only method
	resp, err := h.processor.Query(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search: " + err.Error(),
		})
		return
	}

	// 确保总是返回数组，即使为空
	sources := resp.Sources
	if sources == nil {
		sources = []domain.Chunk{}
	}

	c.JSON(http.StatusOK, sources)
}

func (h *QueryHandler) HandleStream(c *gin.Context) {
	var req domain.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	// Force streaming mode
	req.Stream = true
	h.handleStream(c, req)
}

func (h *QueryHandler) handleStream(c *gin.Context, req domain.QueryRequest) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	// Use StreamQueryWithTools if tools are enabled and requested
	var err error
	if req.ToolsEnabled {
		err = h.processor.StreamQueryWithTools(ctx, req, func(token string) {
			if _, writeErr := fmt.Fprint(c.Writer, token); writeErr != nil {
				// Log the error but continue streaming
				log.Printf("Error writing token: %v", writeErr)
			}
			flusher.Flush()
		})
	} else {
		err = h.processor.StreamQuery(ctx, req, func(token string) {
			if _, writeErr := fmt.Fprint(c.Writer, token); writeErr != nil {
				// Log the error but continue streaming
				log.Printf("Error writing token: %v", writeErr)
			}
			flusher.Flush()
		})
	}

	if err != nil {
		if _, writeErr := fmt.Fprintf(c.Writer, "\n\nError: %v", err); writeErr != nil {
			log.Printf("Error writing error message: %v", writeErr)
		}
		flusher.Flush()
	}
}
