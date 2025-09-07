package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
)

type ResetHandler struct {
	client *client.Client
}

func NewResetHandler(c *client.Client) *ResetHandler {
	return &ResetHandler{client: c}
}

func (h *ResetHandler) Handle(c *gin.Context) {
	var req struct {
		Confirm bool `json:"confirm"`
	}

	if err := c.ShouldBindJSON(&req); err == nil && !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Confirmation required: set confirm=true",
		})
		return
	}

	if h.client.RAG() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "RAG service not available",
		})
		return
	}

	err := h.client.RAG().Reset(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reset database: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database reset successfully",
	})
}
