package rag

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
)

// DocumentInfo represents enhanced document information with metadata
type DocumentInfo struct {
	ID       string                 `json:"id"`
	Path     string                 `json:"path,omitempty"`
	Source   string                 `json:"source,omitempty"`
	Summary  string                 `json:"summary,omitempty"`
	Keywords []string               `json:"keywords,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Created  string                 `json:"created,omitempty"`
}

// ListWithInfo handles requests for documents with enhanced metadata
func (h *DocumentsHandler) ListWithInfo(c *gin.Context) {
	// Get basic documents first
	documents, err := h.processor.ListDocuments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list documents: " + err.Error(),
		})
		return
	}

	// Convert to enhanced info format
	var docsWithInfo []DocumentInfo
	for _, doc := range documents {
		info := DocumentInfo{
			ID:       doc.ID,
			Path:     doc.Path,
			Metadata: doc.Metadata,
		}

		// Extract summary if available in metadata
		if doc.Metadata != nil {
			if summary, ok := doc.Metadata["summary"].(string); ok {
				info.Summary = summary
			}
			if keywords, ok := doc.Metadata["keywords"].([]interface{}); ok {
				for _, kw := range keywords {
					if kwStr, ok := kw.(string); ok {
						info.Keywords = append(info.Keywords, kwStr)
					}
				}
			}
		}

		// Add created timestamp if available
		if !doc.Created.IsZero() {
			info.Created = doc.Created.Format("2006-01-02 15:04:05")
		}

		docsWithInfo = append(docsWithInfo, info)
	}

	// Ensure we always return an array
	if docsWithInfo == nil {
		docsWithInfo = []DocumentInfo{}
	}

	handlers.SendListResponse(c, docsWithInfo, len(docsWithInfo))
}

// GetDocumentInfo retrieves detailed information about a specific document
func (h *DocumentsHandler) GetDocumentInfo(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Document ID is required",
		})
		return
	}

	// Get all documents and find the specific one
	documents, err := h.processor.ListDocuments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve document: " + err.Error(),
		})
		return
	}

	for _, doc := range documents {
		if doc.ID == documentID {
			info := DocumentInfo{
				ID:       doc.ID,
				Path:     doc.Path,
				Metadata: doc.Metadata,
			}

			// Extract enhanced metadata
			if doc.Metadata != nil {
				if summary, ok := doc.Metadata["summary"].(string); ok {
					info.Summary = summary
				}
				if keywords, ok := doc.Metadata["keywords"].([]interface{}); ok {
					for _, kw := range keywords {
						if kwStr, ok := kw.(string); ok {
							info.Keywords = append(info.Keywords, kwStr)
						}
					}
				}
			}

			if !doc.Created.IsZero() {
				info.Created = doc.Created.Format("2006-01-02 15:04:05")
			}

			c.JSON(http.StatusOK, info)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "Document not found",
	})
}