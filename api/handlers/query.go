package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

type QueryHandler struct {
	client *client.Client
}

func NewQueryHandler(c *client.Client) *QueryHandler {
	return &QueryHandler{client: c}
}

func (h *QueryHandler) Handle(c *gin.Context) {
	var req core.SearchRequest
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

	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	resp, err := h.client.RAG().Search(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process query: " + err.Error(),
		})
		return
	}

	// Ensure sources is always an array
	if resp.Results == nil {
		resp.Results = []core.SearchResult{}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *QueryHandler) SearchOnly(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
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

	if req.Limit <= 0 {
		req.Limit = 5
	}

	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	searchReq := core.SearchRequest{
		Query: req.Query,
		Limit: req.Limit,
	}

	resp, err := h.client.RAG().Search(c.Request.Context(), searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search: " + err.Error(),
		})
		return
	}

	// Ensure we always return an array, even if empty
	results := resp.Results
	if results == nil {
		results = []core.SearchResult{}
	}

	c.JSON(http.StatusOK, results)
}
