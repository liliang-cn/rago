package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

type DocumentsHandler struct {
	client *client.Client
}

func NewDocumentsHandler(c *client.Client) *DocumentsHandler {
	return &DocumentsHandler{client: c}
}

func (h *DocumentsHandler) List(c *gin.Context) {
	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	documents, err := h.client.RAG().ListDocuments(c.Request.Context(), core.DocumentFilter{
		Limit:  100, // Default limit
		Offset: 0,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list documents: " + err.Error(),
		})
		return
	}

	// Ensure we always return an array, even if empty
	if documents == nil {
		documents = []core.Document{}
	}

	c.JSON(http.StatusOK, documents)
}

func (h *DocumentsHandler) Delete(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Document ID is required",
		})
		return
	}

	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	err := h.client.RAG().DeleteDocument(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete document: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"document_id": documentID,
		"message":     "Document deleted successfully",
	})
}
