package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

type IngestHandler struct {
	client *client.Client
}

func NewIngestHandler(c *client.Client) *IngestHandler {
	return &IngestHandler{client: c}
}

func (h *IngestHandler) Handle(c *gin.Context) {
	var req core.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	resp, err := h.client.RAG().IngestDocument(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ingest document: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
