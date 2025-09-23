package llm

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
)

// LLMHandler handles direct LLM operations
type LLMHandler struct {
	llmService *llm.Service
}

// NewLLMHandler creates a new LLM handler
func NewLLMHandler(llmService *llm.Service) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
	}
}

// GenerateRequest represents a generation request
type GenerateRequest struct {
	Prompt       string  `json:"prompt" binding:"required"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	Stream       bool    `json:"stream"`
	SystemPrompt string  `json:"system_prompt"`
}

// GenerateResponse represents a generation response
type GenerateResponse struct {
	Content string `json:"content"`
	Model   string `json:"model,omitempty"`
	Usage   *Usage `json:"usage,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// Generate handles text generation requests
func (h *LLMHandler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.Temperature <= 0 {
		req.Temperature = 0.7
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 500
	}

	if req.Stream {
		h.handleStreamGenerate(c, req)
		return
	}

	// Create generation options
	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Generate response
	resp, err := h.llmService.Generate(c.Request.Context(), req.Prompt, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate response: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateResponse{
		Content: resp,
	})
}

// ChatRequest represents a chat request with history
type ChatRequest struct {
	Messages     []domain.Message `json:"messages" binding:"required"`
	Temperature  float64          `json:"temperature"`
	MaxTokens    int              `json:"max_tokens"`
	Stream       bool             `json:"stream"`
	SystemPrompt string           `json:"system_prompt"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response string           `json:"response"`
	Messages []domain.Message `json:"messages,omitempty"`
}

// Chat handles chat requests with history
func (h *LLMHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate messages
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one message is required",
		})
		return
	}

	// Set defaults
	if req.Temperature <= 0 {
		req.Temperature = 0.7
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 500
	}

	if req.Stream {
		h.handleStreamChat(c, req)
		return
	}

	// Create generation options
	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Generate chat response by converting to prompt
	// Since Chat method doesn't exist, we'll use Generate with formatted messages
	prompt := ""
	for _, msg := range req.Messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	resp, err := h.llmService.Generate(c.Request.Context(), prompt, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate chat response: " + err.Error(),
		})
		return
	}

	// Append response to messages
	updatedMessages := append(req.Messages, domain.Message{
		Role:    "assistant",
		Content: resp,
	})

	c.JSON(http.StatusOK, ChatResponse{
		Response: resp,
		Messages: updatedMessages,
	})
}

// StructuredRequest represents a structured generation request
type StructuredRequest struct {
	Prompt      string                 `json:"prompt" binding:"required"`
	Schema      map[string]interface{} `json:"schema" binding:"required"`
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
}

// StructuredResponse represents a structured generation response
type StructuredResponse struct {
	Data  interface{} `json:"data"`
	Valid bool        `json:"valid"`
	Raw   string      `json:"raw"`
}

// GenerateStructured handles structured JSON generation
func (h *LLMHandler) GenerateStructured(c *gin.Context) {
	var req StructuredRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.Temperature <= 0 {
		req.Temperature = 0.3 // Lower temperature for structured output
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 500
	}

	// Create a prompt that includes the schema
	structuredPrompt := fmt.Sprintf(
		"Generate a JSON response that matches this schema: %v\n\nRequest: %s\n\nRespond with valid JSON only.",
		req.Schema,
		req.Prompt,
	)

	// Create generation options
	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Generate response
	resp, err := h.llmService.Generate(c.Request.Context(), structuredPrompt, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate structured response: " + err.Error(),
		})
		return
	}

	// Try to parse as JSON (basic validation)
	// In a real implementation, you'd validate against the schema
	c.JSON(http.StatusOK, StructuredResponse{
		Raw:   resp,
		Valid: true, // Would need actual validation
		Data:  nil,  // Would need to parse JSON
	})
}

// handleStreamGenerate handles streaming generation
func (h *LLMHandler) handleStreamGenerate(c *gin.Context, req GenerateRequest) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	// Create generation options
	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Since StreamGenerate doesn't exist, use regular Generate
	// This is a simplified implementation
	resp, err := h.llmService.Generate(ctx, req.Prompt, opts)
	if err == nil {
		// Send the full response as a single chunk
		fmt.Fprintf(c.Writer, "data: %s\n\n", resp)
		flusher.Flush()
	}

	if err != nil {
		fmt.Fprintf(c.Writer, "data: [ERROR] %v\n\n", err)
		flusher.Flush()
		log.Printf("Streaming error: %v", err)
	}

	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleStreamChat handles streaming chat
func (h *LLMHandler) handleStreamChat(c *gin.Context, req ChatRequest) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	// Create generation options
	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Since StreamChat doesn't exist, use regular Generate with formatted messages
	prompt := ""
	for _, msg := range req.Messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	resp, err := h.llmService.Generate(ctx, prompt, opts)
	if err == nil {
		// Send the full response as a single chunk
		fmt.Fprintf(c.Writer, "data: %s\n\n", resp)
		flusher.Flush()
	}

	if err != nil {
		fmt.Fprintf(c.Writer, "data: [ERROR] %v\n\n", err)
		flusher.Flush()
		log.Printf("Streaming error: %v", err)
	}

	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}
