package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/internal/processor"
)

type DocumentsHandler struct {
	processor *processor.Service
}

func NewDocumentsHandler(p *processor.Service) *DocumentsHandler {
	return &DocumentsHandler{processor: p}
}

func (h *DocumentsHandler) List(c *gin.Context) {
	documents, err := h.processor.ListDocuments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list documents: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"count":     len(documents),
	})
}

func (h *DocumentsHandler) Delete(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Document ID is required",
		})
		return
	}

	err := h.processor.DeleteDocument(c.Request.Context(), documentID)
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