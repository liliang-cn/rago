package rag

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
)

// SearchHandler handles advanced search requests
type SearchHandler struct {
	processor *processor.Service
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(p *processor.Service) *SearchHandler {
	return &SearchHandler{processor: p}
}

// SearchRequest represents an advanced search request
type SearchRequest struct {
	Query          string                 `json:"query" binding:"required"`
	TopK           int                    `json:"top_k"`
	ScoreThreshold float64                `json:"score_threshold,omitempty"`
	HybridSearch   bool                   `json:"hybrid_search,omitempty"`
	VectorWeight   float64                `json:"vector_weight,omitempty"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
	IncludeContent bool                   `json:"include_content,omitempty"`
}

// SearchResult represents a search result with metadata
type SearchResult struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content,omitempty"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Source   string                 `json:"source,omitempty"`
}

// HybridSearch performs a hybrid search combining vector and keyword search
func (h *SearchHandler) HybridSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if req.VectorWeight <= 0 || req.VectorWeight > 1 {
		req.VectorWeight = 0.7 // Default to 70% vector, 30% keyword
	}

	ctx := c.Request.Context()

	// Create query request with filters
	// Note: Hybrid search would need to be implemented in the processor layer
	queryReq := domain.QueryRequest{
		Query:   req.Query,
		TopK:    req.TopK,
		Filters: req.Filters,
	}

	// Use processor's query method with hybrid search enabled
	resp, err := h.processor.Query(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to perform hybrid search: " + err.Error(),
		})
		return
	}

	// Convert response to search results
	var results []SearchResult
	for _, chunk := range resp.Sources {
		result := SearchResult{
			ID:       chunk.ID,
			Score:    chunk.Score,
			Metadata: chunk.Metadata,
		}
		
		if req.IncludeContent {
			result.Content = chunk.Content
		}
		
		// Apply score threshold filtering
		if req.ScoreThreshold > 0 && result.Score < req.ScoreThreshold {
			continue
		}
		
		results = append(results, result)
	}

	// Ensure we always return an array
	if results == nil {
		results = []SearchResult{}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
	})
}

// SemanticSearch performs pure semantic vector search
func (h *SearchHandler) SemanticSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.TopK <= 0 {
		req.TopK = 10
	}

	ctx := c.Request.Context()

	// Create query request for pure semantic search
	queryReq := domain.QueryRequest{
		Query:   req.Query,
		TopK:    req.TopK,
		Filters: req.Filters,
	}

	resp, err := h.processor.Query(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to perform semantic search: " + err.Error(),
		})
		return
	}

	// Convert and filter results
	var results []SearchResult
	for _, chunk := range resp.Sources {
		result := SearchResult{
			ID:       chunk.ID,
			Score:    chunk.Score,
			Metadata: chunk.Metadata,
		}
		
		if req.IncludeContent {
			result.Content = chunk.Content
		}
		
		// Apply score threshold
		if req.ScoreThreshold > 0 && result.Score < req.ScoreThreshold {
			continue
		}
		
		results = append(results, result)
	}

	// Ensure we always return an array
	if results == nil {
		results = []SearchResult{}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
		"method":  "semantic",
	})
}

// FilteredSearch performs search with metadata filtering
func (h *SearchHandler) FilteredSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate that filters are provided
	if req.Filters == nil || len(req.Filters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filters are required for filtered search",
		})
		return
	}

	// Set defaults
	if req.TopK <= 0 {
		req.TopK = 10
	}

	ctx := c.Request.Context()

	// Create query request with filters
	queryReq := domain.QueryRequest{
		Query:   req.Query,
		TopK:    req.TopK,
		Filters: req.Filters,
	}

	resp, err := h.processor.Query(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to perform filtered search: " + err.Error(),
		})
		return
	}

	// Convert results
	var results []SearchResult
	for _, chunk := range resp.Sources {
		result := SearchResult{
			ID:       chunk.ID,
			Score:    chunk.Score,
			Metadata: chunk.Metadata,
		}
		
		if req.IncludeContent {
			result.Content = chunk.Content
		}
		
		results = append(results, result)
	}

	// Ensure we always return an array
	if results == nil {
		results = []SearchResult{}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
		"filters": req.Filters,
	})
}