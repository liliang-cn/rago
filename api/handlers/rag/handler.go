package rag

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Handler handles RAG-related HTTP requests
type Handler struct {
	client *client.Client
}

// NewHandler creates a new RAG handler
func NewHandler(client *client.Client) *Handler {
	return &Handler{
		client: client,
	}
}

// IngestDocument handles document ingestion requests
// @Summary Ingest a document
// @Description Ingest a document into the RAG system for indexing and retrieval
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body core.IngestRequest true "Ingest request"
// @Success 200 {object} core.IngestResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/ingest [post]
func (h *Handler) IngestDocument(c *gin.Context) {
	var req core.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RAG().IngestDocument(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// IngestBatch handles batch document ingestion
// @Summary Ingest multiple documents
// @Description Ingest multiple documents in a single batch operation
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body BatchIngestRequest true "Batch ingest request"
// @Success 200 {object} core.BatchIngestResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/ingest/batch [post]
func (h *Handler) IngestBatch(c *gin.Context) {
	var req BatchIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RAG().IngestBatch(c.Request.Context(), req.Requests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListDocuments lists all documents in the RAG system
// @Summary List documents
// @Description Get a list of all documents in the RAG system with optional filtering
// @Tags RAG
// @Produce json
// @Param source query string false "Filter by source"
// @Param type query string false "Filter by document type"
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} core.Document
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/documents [get]
func (h *Handler) ListDocuments(c *gin.Context) {
	filter := core.DocumentFilter{
		Source: c.Query("source"),
		Type:   c.Query("type"),
	}

	// Parse limit and offset
	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := fmt.Sscanf(limit, "%d", &l); err == nil {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := fmt.Sscanf(offset, "%d", &o); err == nil {
			filter.Offset = o
		}
	}

	docs, err := h.client.RAG().ListDocuments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// DeleteDocument deletes a document from the RAG system
// @Summary Delete a document
// @Description Delete a document from the RAG system by ID
// @Tags RAG
// @Param id path string true "Document ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/documents/{id} [delete]
func (h *Handler) DeleteDocument(c *gin.Context) {
	docID := c.Param("id")
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document ID required"})
		return
	}

	err := h.client.RAG().DeleteDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document deleted successfully"})
}

// Search performs a search query
// @Summary Search documents
// @Description Search for relevant documents using vector similarity
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body core.SearchRequest true "Search request"
// @Success 200 {object} core.SearchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/search [post]
func (h *Handler) Search(c *gin.Context) {
	var req core.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RAG().Search(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HybridSearch performs a hybrid search
// @Summary Hybrid search
// @Description Search using both vector similarity and keyword matching
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body core.HybridSearchRequest true "Hybrid search request"
// @Success 200 {object} core.HybridSearchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/search/hybrid [post]
func (h *Handler) HybridSearch(c *gin.Context) {
	var req core.HybridSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RAG().HybridSearch(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStats returns RAG system statistics
// @Summary Get RAG statistics
// @Description Get statistics about the RAG system including document count and index size
// @Tags RAG
// @Produce json
// @Success 200 {object} core.RAGStats
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.client.RAG().GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Optimize optimizes the RAG indices
// @Summary Optimize RAG indices
// @Description Optimize vector and keyword indices for better performance
// @Tags RAG
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/optimize [post]
func (h *Handler) Optimize(c *gin.Context) {
	err := h.client.RAG().Optimize(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "optimization completed successfully"})
}

// Reset resets the RAG system
// @Summary Reset RAG system
// @Description Reset the RAG system, clearing all documents and indices
// @Tags RAG
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/rag/reset [post]
func (h *Handler) Reset(c *gin.Context) {
	// Add confirmation check for safety
	confirm := c.Query("confirm")
	if confirm != "true" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "confirmation required",
			"message": "Add ?confirm=true to reset the RAG system",
		})
		return
	}

	err := h.client.RAG().Reset(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "RAG system reset successfully"})
}

// Request types
type BatchIngestRequest struct {
	Requests []core.IngestRequest `json:"requests" binding:"required,min=1"`
}