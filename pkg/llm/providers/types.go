package providers

import (
	"context"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Provider defines the interface for LLM pillar providers
type Provider interface {
	// Identity methods
	Name() string
	Type() string
	Model() string
	Config() core.ProviderConfig
	
	// Health and status
	Health(ctx context.Context) error
	Capabilities() ProviderCapabilities
	Metadata() map[string]interface{}
	
	// Generation operations
	Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error)
	Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error
	
	// Tool operations
	GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error)
	StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error
	
	// Lifecycle
	Close() error
}

// ProviderCapabilities describes what a provider supports
type ProviderCapabilities struct {
	SupportsStreaming  bool `json:"supports_streaming"`
	SupportsToolCalls  bool `json:"supports_tool_calls"`
	SupportsBatch      bool `json:"supports_batch"`
	MaxTokens          int  `json:"max_tokens"`
	MaxContextLength   int  `json:"max_context_length"`
}

// GenerationRequest represents a request for text generation within the LLM pillar
type GenerationRequest struct {
	Prompt      string                 `json:"prompt"`
	Model       string                 `json:"model,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Context     []Message              `json:"context,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
}

// GenerationResponse represents a response from text generation
type GenerationResponse struct {
	Content     string                 `json:"content"`
	Model       string                 `json:"model"`
	Provider    string                 `json:"provider"`
	Usage       TokenUsage             `json:"usage"`
	Metadata    map[string]interface{} `json:"metadata"`
	Duration    time.Duration          `json:"duration,omitempty"`
}

// ToolGenerationRequest represents a request with tool calling capability
type ToolGenerationRequest struct {
	GenerationRequest
	Tools         []ToolInfo `json:"tools"`
	ToolChoice    string     `json:"tool_choice,omitempty"`
	MaxToolCalls  int        `json:"max_tool_calls,omitempty"`
}

// ToolGenerationResponse represents response with potential tool calls
type ToolGenerationResponse struct {
	GenerationResponse
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamChunk represents a chunk of streamed content
type StreamChunk struct {
	Content   string        `json:"content"`
	Delta     string        `json:"delta"`
	Finished  bool          `json:"finished"`
	Usage     TokenUsage    `json:"usage,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// ToolStreamChunk represents a chunk of streamed content with tools
type ToolStreamChunk struct {
	StreamChunk
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role       string `json:"role"`    // "user", "assistant", "system", "tool"
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolInfo describes an available tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call in generation
type ToolCall struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// StreamCallback is called for each chunk in streaming generation
type StreamCallback func(chunk *StreamChunk)

// ToolStreamCallback is called for each chunk in streaming tool generation
type ToolStreamCallback func(chunk *ToolStreamChunk)

// ProviderFactory creates provider instances
type ProviderFactory interface {
	CreateProvider(providerType string, name string, config core.ProviderConfig) (Provider, error)
	SupportedTypes() []string
}

// DefaultProviderFactory implements the provider factory
type DefaultProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() ProviderFactory {
	return &DefaultProviderFactory{}
}

// CreateProvider creates a provider instance
func (f *DefaultProviderFactory) CreateProvider(providerType string, name string, config core.ProviderConfig) (Provider, error) {
	switch providerType {
	case "ollama":
		return NewOllamaProvider(name, config)
	case "openai":
		return NewOpenAIProvider(name, config)
	case "lmstudio":
		return NewLMStudioProvider(name, config)
	default:
		return nil, core.ErrProviderNotFound
	}
}

// SupportedTypes returns the list of supported provider types
func (f *DefaultProviderFactory) SupportedTypes() []string {
	return []string{"ollama", "openai", "lmstudio"}
}