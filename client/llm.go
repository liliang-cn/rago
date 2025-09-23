package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// LLMWrapper wraps the domain.Generator to provide additional methods
type LLMWrapper struct {
	generator domain.Generator
}

// NewLLMWrapper creates a new LLM wrapper
func NewLLMWrapper(gen domain.Generator) *LLMWrapper {
	return &LLMWrapper{generator: gen}
}

// Generate generates text from a prompt (simple convenience method)
func (l *LLMWrapper) Generate(prompt string) (string, error) {
	ctx := context.Background()
	if l.generator == nil {
		return "", fmt.Errorf("LLM not initialized")
	}
	return l.generator.Generate(ctx, prompt, nil)
}

// GenerateWithOptions generates text with specific options
func (l *LLMWrapper) GenerateWithOptions(ctx context.Context, prompt string, opts *GenerateOptions) (string, error) {
	if l.generator == nil {
		return "", fmt.Errorf("LLM not initialized")
	}

	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	return l.generator.Generate(ctx, prompt, genOpts)
}

// StreamWithOptions streams text generation with specific options
func (l *LLMWrapper) StreamWithOptions(ctx context.Context, prompt string, callback func(string), opts *GenerateOptions) error {
	if l.generator == nil {
		return fmt.Errorf("LLM not initialized")
	}

	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	return l.generator.Stream(ctx, prompt, genOpts, callback)
}

// ChatWithOptions performs chat completion with specific options
func (l *LLMWrapper) ChatWithOptions(ctx context.Context, messages []ChatMessage, opts *GenerateOptions) (string, error) {
	if l.generator == nil {
		return "", fmt.Errorf("LLM not initialized")
	}

	// Convert ChatMessage to domain.Message
	domainMessages := make([]domain.Message, len(messages))
	for i, msg := range messages {
		domainMessages[i] = domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	// Use GenerateWithTools with nil tools for chat
	result, err := l.generator.GenerateWithTools(ctx, domainMessages, nil, genOpts)
	if err != nil {
		return "", err
	}

	return result.Content, nil
}

// ChatStreamWithOptions performs streaming chat completion with specific options
func (l *LLMWrapper) ChatStreamWithOptions(ctx context.Context, messages []ChatMessage, callback func(string), opts *GenerateOptions) error {
	if l.generator == nil {
		return fmt.Errorf("LLM not initialized")
	}

	// Convert ChatMessage to domain.Message
	domainMessages := make([]domain.Message, len(messages))
	for i, msg := range messages {
		domainMessages[i] = domain.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	genOpts := &domain.GenerationOptions{
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	// Use StreamWithTools with nil tools for chat
	return l.generator.StreamWithTools(ctx, domainMessages, nil, genOpts, func(chunk string, toolCalls []domain.ToolCall) error {
		if chunk != "" {
			callback(chunk)
		}
		return nil
	})
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMGenerateRequest defines the request for direct LLM generation
// Note: If using Ollama provider with HideBuiltinThinkTag enabled,
// think tags (<think>...</think>) will be automatically removed from responses
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
func (c *BaseClient) LLMGenerate(ctx context.Context, req LLMGenerateRequest) (LLMGenerateResponse, error) {
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
func (c *BaseClient) LLMGenerateStream(ctx context.Context, req LLMGenerateRequest, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	return c.llm.Stream(ctx, req.Prompt, opts, callback)
}

// LLMChatRequest defines the request for a direct LLM chat
type LLMChatRequest struct {
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// LLMChat performs a direct multi-turn chat using the configured LLM
func (c *BaseClient) LLMChat(ctx context.Context, req LLMChatRequest) (LLMGenerateResponse, error) {
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
func (c *BaseClient) LLMChatStream(ctx context.Context, req LLMChatRequest, callback func(string)) error {
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
func (c *BaseClient) LLMGenerateStructured(ctx context.Context, req LLMStructuredRequest) (LLMStructuredResponse, error) {
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
