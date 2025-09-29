package chat

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// ChatHandler handles chat-related HTTP requests
type ChatHandler struct {
	processor    domain.RAGProcessor
	llmService   *llm.Service
	usageService *usage.Service
}

// NewChatHandler creates a new chat handler
func NewChatHandler(p domain.RAGProcessor, llm *llm.Service, usageService *usage.Service) *ChatHandler {
	return &ChatHandler{
		processor:    p,
		llmService:   llm,
		usageService: usageService,
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Message        string                    `json:"message" binding:"required"`
	ConversationID string                    `json:"conversation_id,omitempty"`
	Stream         bool                      `json:"stream"`
	Temperature    float64                   `json:"temperature"`
	MaxTokens      int                       `json:"max_tokens"`
	SystemPrompt   string                    `json:"system_prompt"`
	History        []domain.Message          `json:"history"`
	Options        *domain.GenerationOptions `json:"options"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response       string           `json:"response"`
	ConversationID string           `json:"conversation_id,omitempty"`
	History        []domain.Message `json:"history,omitempty"`
	Usage          *Usage           `json:"usage,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// Handle processes a chat request
func (h *ChatHandler) Handle(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate message
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Message is required",
		})
		return
	}

	// Handle streaming response
	if req.Stream {
		h.handleStream(c, req)
		return
	}

	// Handle regular response
	h.handleRegular(c, req)
}

// handleRegular handles non-streaming chat requests
func (h *ChatHandler) handleRegular(c *gin.Context, req ChatRequest) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	startTime := time.Now()

	// Prepare options
	opts := req.Options
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}
	if req.Temperature > 0 {
		opts.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		opts.MaxTokens = req.MaxTokens
	}

	// Handle conversation management
	var conversationID string
	if req.ConversationID != "" {
		// Use existing conversation
		conversationID = req.ConversationID
		if err := h.usageService.SetCurrentConversation(ctx, conversationID); err != nil {
			log.Printf("Warning: failed to set current conversation: %v", err)
		}
	} else {
		// Create new conversation
		conversation, err := h.usageService.StartConversation(ctx, "New Chat")
		if err != nil {
			log.Printf("Warning: failed to start conversation: %v", err)
		} else {
			conversationID = conversation.ID
		}
	}

	// Add user message to database
	userMessage, err := h.usageService.AddMessage(ctx, "user", req.Message)
	if err != nil {
		log.Printf("Warning: failed to add user message: %v", err)
	}

	// Build prompt with history if provided
	prompt := h.buildPromptWithHistory(req)

	// Generate response
	response, err := h.llmService.Generate(ctx, prompt, opts)
	if err != nil {
		// Track error
		if h.usageService != nil {
			h.usageService.TrackError(ctx, usage.CallTypeLLM, "unknown", "unknown", err.Error(), startTime)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate response: " + err.Error(),
		})
		return
	}

	// Add assistant message to database
	assistantMessage, err := h.usageService.AddMessage(ctx, "assistant", response)
	if err != nil {
		log.Printf("Warning: failed to add assistant message: %v", err)
	}

	// Track LLM call
	if h.usageService != nil {
		_, err := h.usageService.TrackLLMCall(ctx, "unknown", "unknown", prompt, response, startTime)
		if err != nil {
			log.Printf("Warning: failed to track LLM call: %v", err)
		}
	}

	// Build response with updated history
	chatResp := ChatResponse{
		Response:       response,
		ConversationID: conversationID,
	}

	// Include conversation history if requested
	if len(req.History) > 0 || conversationID != "" {
		history := append(req.History,
			domain.Message{Role: "user", Content: req.Message},
			domain.Message{Role: "assistant", Content: response},
		)
		chatResp.History = history
	}

	// Include usage information if available
	if userMessage != nil && assistantMessage != nil {
		chatResp.Usage = &Usage{
			PromptTokens:     userMessage.TokenCount,
			CompletionTokens: assistantMessage.TokenCount,
			TotalTokens:      userMessage.TokenCount + assistantMessage.TokenCount,
		}
	}

	c.JSON(http.StatusOK, chatResp)
}

// handleStream handles streaming chat requests
func (h *ChatHandler) handleStream(c *gin.Context, req ChatRequest) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	startTime := time.Now()

	// Prepare options
	opts := req.Options
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}
	if req.Temperature > 0 {
		opts.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		opts.MaxTokens = req.MaxTokens
	}

	// Handle conversation management
	var conversationID string
	if req.ConversationID != "" {
		// Use existing conversation
		conversationID = req.ConversationID
		if err := h.usageService.SetCurrentConversation(ctx, conversationID); err != nil {
			log.Printf("Warning: failed to set current conversation: %v", err)
		}
	} else {
		// Create new conversation
		conversation, err := h.usageService.StartConversation(ctx, "New Chat")
		if err != nil {
			log.Printf("Warning: failed to start conversation: %v", err)
		} else {
			conversationID = conversation.ID
		}
	}

	// Add user message to database
	_, err := h.usageService.AddMessage(ctx, "user", req.Message)
	if err != nil {
		log.Printf("Warning: failed to add user message: %v", err)
	}

	// Build prompt with history
	prompt := h.buildPromptWithHistory(req)

	// Stream response
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	var fullResponse string
	err = h.llmService.Stream(ctx, prompt, opts, func(chunk string) {
		fullResponse += chunk
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
		flusher.Flush()
	})

	if err != nil {
		// Track error
		if h.usageService != nil {
			h.usageService.TrackError(ctx, usage.CallTypeLLM, "unknown", "unknown", err.Error(), startTime)
		}
		log.Printf("Stream error: %v", err)
		_, _ = fmt.Fprintf(c.Writer, "data: [ERROR] %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Add assistant message to database
	_, err = h.usageService.AddMessage(ctx, "assistant", fullResponse)
	if err != nil {
		log.Printf("Warning: failed to add assistant message: %v", err)
	}

	// Track LLM call
	if h.usageService != nil {
		_, err := h.usageService.TrackLLMCall(ctx, "unknown", "unknown", prompt, fullResponse, startTime)
		if err != nil {
			log.Printf("Warning: failed to track LLM call: %v", err)
		}
	}

	// Send completion signal with conversation ID
	_, _ = fmt.Fprintf(c.Writer, "data: [DONE] conversation_id:%s\n\n", conversationID)
	flusher.Flush()
}

// buildPromptWithHistory builds a prompt including conversation history
func (h *ChatHandler) buildPromptWithHistory(req ChatRequest) string {
	if len(req.History) == 0 && req.SystemPrompt == "" {
		return req.Message
	}

	var prompt string

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("System: %s\n\n", req.SystemPrompt)
	}

	// Add conversation history
	for _, msg := range req.History {
		switch msg.Role {
		case "user":
			prompt += fmt.Sprintf("User: %s\n", msg.Content)
		case "assistant":
			prompt += fmt.Sprintf("Assistant: %s\n", msg.Content)
		case "system":
			prompt += fmt.Sprintf("System: %s\n", msg.Content)
		}
	}

	// Add current message
	prompt += fmt.Sprintf("User: %s\nAssistant: ", req.Message)

	return prompt
}

// Complete handles completion requests (similar to OpenAI's completion API)
func (h *ChatHandler) Complete(c *gin.Context) {
	var req struct {
		Prompt      string                    `json:"prompt" binding:"required"`
		MaxTokens   int                       `json:"max_tokens"`
		Temperature float64                   `json:"temperature"`
		Stream      bool                      `json:"stream"`
		Options     *domain.GenerationOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Convert to chat request
	chatReq := ChatRequest{
		Message:     req.Prompt,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Options:     req.Options,
	}

	if req.Stream {
		h.handleStream(c, chatReq)
	} else {
		h.handleRegular(c, chatReq)
	}
}
