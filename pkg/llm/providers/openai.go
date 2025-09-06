package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// OpenAIProvider wraps the domain OpenAI provider for the LLM pillar
type OpenAIProvider struct {
	provider domain.LLMProvider
	config   core.ProviderConfig
	name     string
}

// NewOpenAIProvider creates a new OpenAI provider for the LLM pillar
func NewOpenAIProvider(name string, config core.ProviderConfig) (*OpenAIProvider, error) {
	// Convert core config to domain config
	domainConfig := &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderOpenAI,
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
		return nil, fmt.Errorf("failed to create OpenAI provider: %w", err)
	}
	
	return &OpenAIProvider{
		provider: provider,
		config:   config,
		name:     name,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OpenAIProvider) Type() string {
	return "openai"
}

// Model returns the configured model
func (p *OpenAIProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *OpenAIProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *OpenAIProvider) Health(ctx context.Context) error {
	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	return p.provider.Health(healthCtx)
}

// Generate generates text using the OpenAI provider
func (p *OpenAIProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Convert to domain format
	prompt, opts := p.convertRequest(req)
	
	// Call domain provider
	content, err := p.provider.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("OpenAI generation failed: %w", err)
	}
	
	// Create response
	return &GenerationResponse{
		Content:  content,
		Model:    p.config.Model,
		Provider: p.name,
		Usage: TokenUsage{
			// TODO: Extract actual token usage from OpenAI response
			TotalTokens: len(content) / 4, // Rough estimate
		},
		Metadata: make(map[string]interface{}),
	}, nil
}

// Stream generates streaming text using the OpenAI provider
func (p *OpenAIProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
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

// GenerateWithTools generates text with tool calling
func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// Convert requests to domain format for tool calling
	messages := p.convertMessages(req.Context)
	if req.Prompt != "" {
		messages = append(messages, domain.Message{
			Role:    "user",
			Content: req.Prompt,
		})
	}
	
	tools := p.convertTools(req.Tools)
	opts := &domain.GenerationOptions{
		Temperature: float64(req.Temperature),
		MaxTokens:   req.MaxTokens,
		ToolChoice:  req.ToolChoice,
	}
	
	// Call domain provider with tools
	result, err := p.provider.GenerateWithTools(ctx, messages, tools, opts)
	if err != nil {
		return nil, fmt.Errorf("OpenAI tool generation failed: %w", err)
	}
	
	// Convert result back to LLM pillar format
	toolCalls := make([]ToolCall, len(result.ToolCalls))
	for i, tc := range result.ToolCalls {
		toolCalls[i] = ToolCall{
			ID:         tc.ID,
			Name:       tc.Function.Name,
			Parameters: tc.Function.Arguments,
		}
	}
	
	return &ToolGenerationResponse{
		GenerationResponse: GenerationResponse{
			Content:  result.Content,
			Model:    p.config.Model,
			Provider: p.name,
			Usage: TokenUsage{
				// TODO: Extract actual token usage
				TotalTokens: len(result.Content) / 4,
			},
			Metadata: make(map[string]interface{}),
		},
		ToolCalls: toolCalls,
	}, nil
}

// StreamWithTools generates streaming text with tool calling
func (p *OpenAIProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
	// Convert to domain format
	messages := p.convertMessages(req.Context)
	if req.Prompt != "" {
		messages = append(messages, domain.Message{
			Role:    "user",
			Content: req.Prompt,
		})
	}
	
	tools := p.convertTools(req.Tools)
	opts := &domain.GenerationOptions{
		Temperature: float64(req.Temperature),
		MaxTokens:   req.MaxTokens,
		ToolChoice:  req.ToolChoice,
	}
	
	// Call domain provider with streaming tool support
	return p.provider.StreamWithTools(ctx, messages, tools, opts, func(chunk string, toolCalls []domain.ToolCall) error {
		// Convert tool calls
		llmToolCalls := make([]ToolCall, len(toolCalls))
		for i, tc := range toolCalls {
			llmToolCalls[i] = ToolCall{
				ID:         tc.ID,
				Name:       tc.Function.Name,
				Parameters: tc.Function.Arguments,
			}
		}
		
		toolChunk := &ToolStreamChunk{
			StreamChunk: StreamChunk{
				Content: chunk,
				Delta:   chunk,
			},
			ToolCalls: llmToolCalls,
		}
		
		callback(toolChunk)
		return nil
	})
}

// Capabilities returns the provider capabilities
func (p *OpenAIProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming:  true,
		SupportsToolCalls:  true, // OpenAI supports function calling
		SupportsBatch:      false, // TODO: Implement batch API
		MaxTokens:          getMaxTokensForModel(p.config.Model),
		MaxContextLength:   getContextLengthForModel(p.config.Model),
	}
}

// Metadata returns additional provider metadata
func (p *OpenAIProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type":   "openai",
		"base_url":        p.config.BaseURL,
		"model":           p.config.Model,
		"timeout":         p.config.Timeout,
		"supports_tools":  true,
		"supports_stream": true,
	}
}

// Close closes the provider and cleans up resources
func (p *OpenAIProvider) Close() error {
	// Domain providers don't typically have close methods
	// This is a no-op for now
	return nil
}

// === PRIVATE METHODS ===

// convertRequest converts LLM pillar request to domain format
func (p *OpenAIProvider) convertRequest(req *GenerationRequest) (string, *domain.GenerationOptions) {
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

// convertMessages converts LLM pillar messages to domain messages
func (p *OpenAIProvider) convertMessages(messages []Message) []domain.Message {
	domainMessages := make([]domain.Message, len(messages))
	for i, msg := range messages {
		domainMessages[i] = domain.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}
	return domainMessages
}

// convertTools converts LLM pillar tools to domain tools
func (p *OpenAIProvider) convertTools(tools []ToolInfo) []domain.ToolDefinition {
	domainTools := make([]domain.ToolDefinition, len(tools))
	for i, tool := range tools {
		domainTools[i] = domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return domainTools
}

// getMaxTokensForModel returns the maximum tokens for a given OpenAI model
func getMaxTokensForModel(model string) int {
	switch model {
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		return 128000
	case "gpt-4":
		return 8192
	case "gpt-3.5-turbo":
		return 4096
	case "gpt-3.5-turbo-16k":
		return 16384
	default:
		return 4096 // Default fallback
	}
}

// getContextLengthForModel returns the context length for a given OpenAI model
func getContextLengthForModel(model string) int {
	switch model {
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		return 128000
	case "gpt-4":
		return 8192
	case "gpt-3.5-turbo":
		return 4096
	case "gpt-3.5-turbo-16k":
		return 16384
	default:
		return 4096 // Default fallback
	}
}