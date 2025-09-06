package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// OllamaProvider wraps the domain Ollama provider for the LLM pillar
type OllamaProvider struct {
	provider domain.LLMProvider
	config   core.ProviderConfig
	name     string
}

// NewOllamaProvider creates a new Ollama provider for the LLM pillar
func NewOllamaProvider(name string, config core.ProviderConfig) (*OllamaProvider, error) {
	// Convert core config to domain config
	domainConfig := &domain.OllamaProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderOllama,
			Timeout: config.Timeout,
		},
		BaseURL:        config.BaseURL,
		LLMModel:       config.Model,
		EmbeddingModel: config.Model, // Use same model for embedding by default
	}
	
	// Create domain provider
	factory := providers.NewFactory()
	provider, err := factory.CreateLLMProvider(context.Background(), domainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama provider: %w", err)
	}
	
	return &OllamaProvider{
		provider: provider,
		config:   config,
		name:     name,
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OllamaProvider) Type() string {
	return "ollama"
}

// Model returns the configured model
func (p *OllamaProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *OllamaProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *OllamaProvider) Health(ctx context.Context) error {
	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	return p.provider.Health(healthCtx)
}

// Generate generates text using the Ollama provider
func (p *OllamaProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Convert to domain format
	prompt, opts := p.convertRequest(req)
	
	// Call domain provider
	content, err := p.provider.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("Ollama generation failed: %w", err)
	}
	
	// Create response
	return &GenerationResponse{
		Content:  content,
		Model:    p.config.Model,
		Provider: p.name,
		Usage: TokenUsage{
			// TODO: Extract token usage if available from Ollama
			TotalTokens: len(content) / 4, // Rough estimate
		},
		Metadata: make(map[string]interface{}),
	}, nil
}

// Stream generates streaming text using the Ollama provider
func (p *OllamaProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
	// Convert to domain format
	prompt, opts := p.convertRequest(req)
	
	// Call domain provider with streaming
	return p.provider.Stream(ctx, prompt, opts, func(chunk string) {
		streamChunk := &StreamChunk{
			Content: chunk,
			Delta:   chunk,
		}
		callback(streamChunk)
	})
}

// GenerateWithTools generates text with tool calling (placeholder implementation)
func (p *OllamaProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// For now, delegate to regular generation
	genReq := &GenerationRequest{
		Prompt:      req.Prompt,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     req.Context,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	
	response, err := p.Generate(ctx, genReq)
	if err != nil {
		return nil, err
	}
	
	return &ToolGenerationResponse{
		GenerationResponse: *response,
		ToolCalls:         []ToolCall{}, // No tool calls for now
	}, nil
}

// StreamWithTools generates streaming text with tool calling (placeholder implementation)
func (p *OllamaProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
	// For now, delegate to regular streaming
	genReq := &GenerationRequest{
		Prompt:      req.Prompt,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     req.Context,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	
	return p.Stream(ctx, genReq, func(chunk *StreamChunk) {
		toolChunk := &ToolStreamChunk{
			StreamChunk: *chunk,
			ToolCalls:   []ToolCall{}, // No tool calls for now
		}
		callback(toolChunk)
	})
}

// Capabilities returns the provider capabilities
func (p *OllamaProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming:  true,
		SupportsToolCalls: false, // TODO: Implement when Ollama supports it
		SupportsBatch:     false,
		MaxTokens:         4096, // Default for most Ollama models
		MaxContextLength:  4096,
	}
}

// Metadata returns additional provider metadata
func (p *OllamaProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type": "ollama",
		"base_url":      p.config.BaseURL,
		"model":         p.config.Model,
		"timeout":       p.config.Timeout,
	}
}

// Close closes the provider and cleans up resources
func (p *OllamaProvider) Close() error {
	// Domain providers don't typically have close methods
	// This is a no-op for now
	return nil
}

// === PRIVATE METHODS ===

// convertRequest converts LLM pillar request to domain format
func (p *OllamaProvider) convertRequest(req *GenerationRequest) (string, *domain.GenerationOptions) {
	prompt := req.Prompt
	
	// Build prompt from context if available
	if len(req.Context) > 0 {
		for _, msg := range req.Context {
			prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
		if req.Prompt != "" {
			prompt += fmt.Sprintf("user: %s\n", req.Prompt)
		}
	}
	
	opts := &domain.GenerationOptions{
		Temperature: float64(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}
	
	return prompt, opts
}