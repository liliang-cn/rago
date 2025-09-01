package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// LLMGenerateRequest defines the request for direct LLM generation
type LLMGenerateRequest struct {
	Prompt      string  // The prompt to generate from
	Temperature float64 // Generation temperature (0.0-1.0)
	MaxTokens   int     // Maximum tokens to generate
}

// LLMGenerateResponse defines the response from a direct LLM generation
type LLMGenerateResponse struct {
	Content string
}

// LLMGenerate performs a direct generation using the configured LLM
func (c *Client) LLMGenerate(ctx context.Context, req LLMGenerateRequest) (LLMGenerateResponse, error) {
	if c.llm == nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	content, err := c.llm.Generate(ctx, req.Prompt, opts)
	if err != nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM generation failed: %w", err)
	}

	return LLMGenerateResponse{Content: content}, nil
}

// LLMGenerateStream performs a direct streaming generation using the configured LLM
func (c *Client) LLMGenerateStream(ctx context.Context, req LLMGenerateRequest, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	return c.llm.Stream(ctx, req.Prompt, opts, callback)
}

// ChatMessage defines a single message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// LLMChatRequest defines the request for a direct LLM chat
type LLMChatRequest struct {
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// LLMChat performs a direct multi-turn chat using the configured LLM
func (c *Client) LLMChat(ctx context.Context, req LLMChatRequest) (LLMGenerateResponse, error) {
	if c.llm == nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM service not initialized")
	}

	// Convert messages to internal domain format
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{Role: msg.Role, Content: msg.Content}
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Use GenerateWithTools with no tools for chat
	result, err := c.llm.GenerateWithTools(ctx, domainMessages, nil, opts)
	if err != nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM chat failed: %w", err)
	}

	return LLMGenerateResponse{Content: result.Content}, nil
}

// LLMChatStream performs a direct streaming chat using the configured LLM
func (c *Client) LLMChatStream(ctx context.Context, req LLMChatRequest, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	// Convert messages to internal domain format
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{Role: msg.Role, Content: msg.Content}
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Use StreamWithTools with no tools for chat streaming
	return c.llm.StreamWithTools(ctx, domainMessages, nil, opts, func(chunk string, toolCalls []domain.ToolCall) error {
		callback(chunk)
		return nil
	})
}

// LLMStructuredRequest defines the request for structured LLM generation
type LLMStructuredRequest struct {
	Prompt      string      // The prompt to generate from
	Schema      interface{} // Target struct to parse JSON into
	Temperature float64     // Generation temperature (0.0-1.0)
	MaxTokens   int         // Maximum tokens to generate
}

// LLMStructuredResponse defines the response from structured LLM generation
type LLMStructuredResponse struct {
	Data  interface{} // Parsed structured data
	Raw   string      // Raw JSON string
	Valid bool        // Whether response passed schema validation
}

// LLMGenerateStructured performs structured JSON generation using the configured LLM
func (c *Client) LLMGenerateStructured(ctx context.Context, req LLMStructuredRequest) (LLMStructuredResponse, error) {
	if c.llm == nil {
		return LLMStructuredResponse{}, fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	result, err := c.llm.GenerateStructured(ctx, req.Prompt, req.Schema, opts)
	if err != nil {
		return LLMStructuredResponse{}, fmt.Errorf("LLM structured generation failed: %w", err)
	}

	return LLMStructuredResponse{
		Data:  result.Data,
		Raw:   result.Raw,
		Valid: result.Valid,
	}, nil
}