package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// LMStudioProvider wraps the domain LMStudio provider for the LLM pillar
type LMStudioProvider struct {
	provider domain.LLMProvider
	config   core.ProviderConfig
	name     string
}

// NewLMStudioProvider creates a new LMStudio provider for the LLM pillar
func NewLMStudioProvider(name string, config core.ProviderConfig) (*LMStudioProvider, error) {
	// Convert core config to domain config
	domainConfig := &domain.LMStudioProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderLMStudio,
			Timeout: config.Timeout,
		},
		BaseURL:        config.BaseURL,
		APIKey:         config.APIKey,
		LLMModel:       config.Model,
		EmbeddingModel: config.Model, // Use same model for embedding by default
	}
	
	// Create domain provider
	factory := providers.NewFactory()
	provider, err := factory.CreateLLMProvider(context.Background(), domainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LMStudio provider: %w", err)
	}
	
	return &LMStudioProvider{
		provider: provider,
		config:   config,
		name:     name,
	}, nil
}

// Name returns the provider name
func (p *LMStudioProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *LMStudioProvider) Type() string {
	return "lmstudio"
}

// Model returns the configured model
func (p *LMStudioProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *LMStudioProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *LMStudioProvider) Health(ctx context.Context) error {
	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	return p.provider.Health(healthCtx)
}

// Generate generates text using the LMStudio provider
func (p *LMStudioProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Convert to domain format
	prompt, opts := p.convertRequest(req)
	
	// Call domain provider
	content, err := p.provider.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("LMStudio generation failed: %w", err)
	}
	
	// Create response
	return &GenerationResponse{
		Content:  content,
		Model:    p.config.Model,
		Provider: p.name,
		Usage: TokenUsage{
			// TODO: Extract token usage if available from LMStudio
			TotalTokens: len(content) / 4, // Rough estimate
		},
		Metadata: make(map[string]interface{}),
	}, nil
}

// Stream generates streaming text using the LMStudio provider
func (p *LMStudioProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
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
func (p *LMStudioProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// LMStudio might support tools in the future, for now delegate to regular generation
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
func (p *LMStudioProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
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
func (p *LMStudioProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming:  true,
		SupportsToolCalls:  false, // LMStudio doesn't support tool calling yet
		SupportsBatch:      false,
		MaxTokens:          4096, // Default for most local models
		MaxContextLength:   4096, // Depends on the loaded model
	}
}

// Metadata returns additional provider metadata
func (p *LMStudioProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type":   "lmstudio",
		"base_url":        p.config.BaseURL,
		"model":           p.config.Model,
		"timeout":         p.config.Timeout,
		"supports_tools":  false,
		"supports_stream": true,
		"local":           true, // LMStudio is typically local
	}
}

// Close closes the provider and cleans up resources
func (p *LMStudioProvider) Close() error {
	// Domain providers don't typically have close methods
	// This is a no-op for now
	return nil
}

// === PRIVATE METHODS ===

// convertRequest converts LLM pillar request to domain format
func (p *LMStudioProvider) convertRequest(req *GenerationRequest) (string, *domain.GenerationOptions) {
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