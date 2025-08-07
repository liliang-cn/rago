package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/internal/processor"
)

type ResetHandler struct {
	processor *processor.Service
}

func NewResetHandler(p *processor.Service) *ResetHandler {
	return &ResetHandler{processor: p}
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

	err := h.processor.Reset(c.Request.Context())
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