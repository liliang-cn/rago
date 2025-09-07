package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
)

// HealthHandler provides comprehensive health check for all components using V3 client
type HealthHandler struct {
	client *client.Client
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(c *client.Client) *HealthHandler {
	return &HealthHandler{
		client: c,
	}
}


func (h *HealthHandler) Handle(c *gin.Context) {
	// Use the V3 client's built-in health check
	health := h.client.Health()

	// Convert to HTTP-friendly format
	response := map[string]interface{}{
		"status":    string(health.Overall),
		"pillars":   health.Pillars,
		"providers": health.Providers,
		"servers":   health.Servers,
		"timestamp": health.LastCheck.UTC().Format("2006-01-02T15:04:05Z"),
		"version":   "3.0.0",
	}

	// Add details if available
	if health.Details != nil {
		response["details"] = health.Details
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if health.Overall == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Overall == "degraded" {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, response)
}
