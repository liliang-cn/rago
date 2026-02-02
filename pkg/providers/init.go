package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/embedder"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// InitializeProviders is a helper function to initialize services using provider system
func InitializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	factory := NewFactory()

	// Initialize embedder service using providers
	embedService, err := InitializeEmbedder(ctx, cfg, factory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize LLM provider using providers
	llmProvider, err := InitializeLLM(ctx, cfg, factory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Create processor service
	// For metadata extractor, we need to use LLM provider if it implements interface
	var metadataExtractor domain.MetadataExtractor
	if extractor, ok := llmProvider.(domain.MetadataExtractor); ok {
		metadataExtractor = extractor
	} else {
		// Fallback: use embedder service (it has a placeholder implementation)
		metadataExtractor = embedService.(domain.MetadataExtractor)
	}

	// Wrap LLM provider with usage tracking
	usageService, err := usage.NewService(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create usage service: %w", err)
	}

	trackedProvider := NewTrackedLLMProvider(llmProvider.(domain.LLMProvider), usageService)

	return embedService, llm.NewService(trackedProvider), metadataExtractor, nil
}

// InitializeEmbedder initializes embedder service using provider system
func InitializeEmbedder(ctx context.Context, cfg *config.Config, factory *Factory) (domain.Embedder, error) {
	// Try to get embedder config (with custom providers support)
	providerConfig, err := GetEmbedderProviderConfigWithCustom(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultEmbedder, cfg.Providers.Providers)
	if err != nil {
		return nil, err
	}

	provider, err := factory.CreateEmbedderProvider(ctx, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder provider: %w", err)
	}

	return embedder.NewService(provider), nil
}

// InitializeLLM initializes LLM service using provider system
func InitializeLLM(ctx context.Context, cfg *config.Config, factory *Factory) (domain.Generator, error) {
	// Try to get LLM config (with custom providers support)
	providerConfig, err := GetLLMProviderConfigWithCustom(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultLLM, cfg.Providers.Providers)
	if err != nil {
		return nil, err
	}

	provider, err := factory.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return provider, nil
}

// CheckProviderHealth checks health of provider services
func CheckProviderHealth(ctx context.Context, embedService domain.Embedder, llmService domain.Generator) error {
	// Check embedder health - need to check if it has Health method
	if healthChecker, ok := embedService.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			return fmt.Errorf("embedder health check failed: %w", err)
		}
	}

	// Check LLM health - need to check if it has Health method
	if healthChecker, ok := llmService.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			return fmt.Errorf("LLM health check failed: %w", err)
		}
	}

	return nil
}

// ComposePrompt creates a RAG prompt from document chunks and user query
func ComposePrompt(chunks []domain.Chunk, query string) string {
	if len(chunks) == 0 {
		return fmt.Sprintf("Please answer the following question:\n\n%s", query)
	}

	var promptBuilder strings.Builder

	promptBuilder.WriteString("Based on the following document content, please answer the user's question. If the documents do not contain relevant information, please indicate that you cannot find an answer from the provided documents.\n\n")
	promptBuilder.WriteString("Document Content:\n")

	for i, chunk := range chunks {
		promptBuilder.WriteString(fmt.Sprintf("[Document Fragment %d]\n%s\n\n", i+1, chunk.Content))
	}

	promptBuilder.WriteString(fmt.Sprintf("User Question: %s\n\n", query))
	promptBuilder.WriteString("Please provide a detailed and accurate answer based on the document content:")

	return promptBuilder.String()
}