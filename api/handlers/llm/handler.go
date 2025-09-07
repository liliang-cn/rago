package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Handler handles LLM-related HTTP requests
type Handler struct {
	client *client.Client
}

// NewHandler creates a new LLM handler
func NewHandler(client *client.Client) *Handler {
	return &Handler{
		client: client,
	}
}

// Generate handles text generation requests
// @Summary Generate text using LLM
// @Description Generate text based on a prompt using the configured LLM provider
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body core.GenerationRequest true "Generation request"
// @Success 200 {object} core.GenerationResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/generate [post]
func (h *Handler) Generate(c *gin.Context) {
	var req core.GenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.LLM().Generate(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Stream handles streaming text generation requests
// @Summary Stream text generation
// @Description Stream text generation responses using Server-Sent Events
// @Tags LLM
// @Accept json
// @Produce text/event-stream
// @Param request body core.GenerationRequest true "Generation request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/stream [post]
func (h *Handler) Stream(c *gin.Context) {
	var req core.GenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create a channel for streaming
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Start streaming
	err := h.client.LLM().Stream(c.Request.Context(), req, func(chunk core.StreamChunk) error {
		// Format as SSE
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}

	// Send done event
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ListProviders lists all configured LLM providers
// @Summary List LLM providers
// @Description Get a list of all configured LLM providers
// @Tags LLM
// @Produce json
// @Success 200 {array} core.ProviderInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/providers [get]
func (h *Handler) ListProviders(c *gin.Context) {
	providers := h.client.LLM().ListProviders()
	c.JSON(http.StatusOK, providers)
}

// AddProvider adds a new LLM provider
// @Summary Add LLM provider
// @Description Add a new LLM provider configuration
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body AddProviderRequest true "Provider configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/providers [post]
func (h *Handler) AddProvider(c *gin.Context) {
	var req AddProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.client.LLM().AddProvider(req.Name, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider added successfully"})
}

// RemoveProvider removes an LLM provider
// @Summary Remove LLM provider
// @Description Remove an LLM provider by name
// @Tags LLM
// @Param name path string true "Provider name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/providers/{name} [delete]
func (h *Handler) RemoveProvider(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider name required"})
		return
	}

	err := h.client.LLM().RemoveProvider(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider removed successfully"})
}

// GetProviderHealth gets health status of all providers
// @Summary Get provider health
// @Description Get health status of all LLM providers
// @Tags LLM
// @Produce json
// @Success 200 {object} map[string]core.HealthStatus
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/providers/health [get]
func (h *Handler) GetProviderHealth(c *gin.Context) {
	health := h.client.LLM().GetProviderHealth()
	c.JSON(http.StatusOK, health)
}

// GenerateBatch handles batch generation requests
// @Summary Batch generate text
// @Description Generate text for multiple prompts in batch
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body BatchGenerationRequest true "Batch generation request"
// @Success 200 {array} core.GenerationResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/batch [post]
func (h *Handler) GenerateBatch(c *gin.Context) {
	var req BatchGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	responses, err := h.client.LLM().GenerateBatch(c.Request.Context(), req.Requests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, responses)
}

// GenerateWithTools handles generation with tool calling
// @Summary Generate with tools
// @Description Generate text with tool calling capabilities
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body core.ToolGenerationRequest true "Tool generation request"
// @Success 200 {object} core.ToolGenerationResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/tools/generate [post]
func (h *Handler) GenerateWithTools(c *gin.Context) {
	var req core.ToolGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.LLM().GenerateWithTools(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StreamWithTools handles streaming generation with tool calling
// @Summary Stream with tools
// @Description Stream text generation with tool calling capabilities
// @Tags LLM
// @Accept json
// @Produce text/event-stream
// @Param request body core.ToolGenerationRequest true "Tool generation request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/llm/tools/stream [post]
func (h *Handler) StreamWithTools(c *gin.Context) {
	var req core.ToolGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Start streaming with tools
	err := h.client.LLM().StreamWithTools(c.Request.Context(), req, func(chunk core.ToolStreamChunk) error {
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// StreamNDJSON handles streaming with newline-delimited JSON
func (h *Handler) StreamNDJSON(c *gin.Context) {
	var req core.GenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set NDJSON headers
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	encoder := json.NewEncoder(w)

	// Start streaming
	err := h.client.LLM().Stream(c.Request.Context(), req, func(chunk core.StreamChunk) error {
		if err := encoder.Encode(chunk); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		encoder.Encode(map[string]string{"error": err.Error()})
		flusher.Flush()
	}
}

// Request types
type AddProviderRequest struct {
	Name   string               `json:"name" binding:"required"`
	Config core.ProviderConfig  `json:"config" binding:"required"`
}

type BatchGenerationRequest struct {
	Requests []core.GenerationRequest `json:"requests" binding:"required,min=1"`
}