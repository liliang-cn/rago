package rag

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/processor"
)

type IngestHandler struct {
	processor *processor.Service
}

func NewIngestHandler(p *processor.Service) *IngestHandler {
	return &IngestHandler{processor: p}
}

func (h *IngestHandler) Handle(c *gin.Context) {
	var req domain.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	resp, err := h.processor.Ingest(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ingest document: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
