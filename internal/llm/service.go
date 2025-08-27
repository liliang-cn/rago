package llm

import (
	"context"

	"github.com/liliang-cn/rago/internal/domain"
)

// Service wraps a provider-based LLM
type Service struct {
	provider domain.LLMProvider
}

// NewService creates a new LLM service with a provider
func NewService(provider domain.LLMProvider) *Service {
	return &Service{
		provider: provider,
	}
}

// Generate generates text using the configured provider
func (s *Service) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	return s.provider.Generate(ctx, prompt, opts)
}

// Stream generates text with streaming using the configured provider
func (s *Service) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	return s.provider.Stream(ctx, prompt, opts, callback)
}

// GenerateWithTools generates text with tool calling support
func (s *Service) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return s.provider.GenerateWithTools(ctx, messages, tools, opts)
}

// StreamWithTools generates text with tool calling support in streaming mode
func (s *Service) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	return s.provider.StreamWithTools(ctx, messages, tools, opts, callback)
}

// ExtractMetadata extracts metadata from content
func (s *Service) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	return s.provider.ExtractMetadata(ctx, content, model)
}

// Health checks the health of the underlying provider
func (s *Service) Health(ctx context.Context) error {
	return s.provider.Health(ctx)
}

// ProviderType returns the provider type being used
func (s *Service) ProviderType() domain.ProviderType {
	return s.provider.ProviderType()
}