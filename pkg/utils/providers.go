package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/pkg/config"
	"github.com/liliang-cn/rago/pkg/domain"
	"github.com/liliang-cn/rago/pkg/embedder"
	"github.com/liliang-cn/rago/pkg/llm"
	"github.com/liliang-cn/rago/pkg/providers"
)

// InitializeProviders is a helper function to initialize services using the provider system
func InitializeProviders(ctx context.Context, cfg *config.Config) (domain.Embedder, domain.Generator, domain.MetadataExtractor, error) {
	factory := providers.NewFactory()

	// Initialize embedder service using providers
	embedService, err := InitializeEmbedder(ctx, cfg, factory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize LLM service using providers  
	llmService, err := InitializeLLM(ctx, cfg, factory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create LLM service: %w", err)
	}

	// Create processor service 
	// For the metadata extractor, we need to use the LLM service if it implements the interface
	var metadataExtractor domain.MetadataExtractor
	if extractor, ok := llmService.(domain.MetadataExtractor); ok {
		metadataExtractor = extractor
	} else {
		// Fallback: use embedder service (it has a placeholder implementation)
		metadataExtractor = embedService.(domain.MetadataExtractor)
	}

	return embedService, llmService, metadataExtractor, nil
}

// InitializeEmbedder initializes the embedder service using the provider system
func InitializeEmbedder(ctx context.Context, cfg *config.Config, factory *providers.Factory) (domain.Embedder, error) {
	// Check if new provider configuration exists
	if cfg.Providers.ProviderConfigs.Ollama != nil || cfg.Providers.ProviderConfigs.OpenAI != nil || cfg.Providers.ProviderConfigs.LMStudio != nil {
		// Use new provider system
		providerConfig, err := providers.GetEmbedderProviderConfig(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultEmbedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder provider config: %w", err)
		}

		provider, err := factory.CreateEmbedderProvider(ctx, providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedder provider: %w", err)
		}

		return embedder.NewService(provider), nil
	}

	// Fallback to legacy Ollama configuration for backward compatibility
	// Create legacy Ollama provider using the new provider system
	legacyConfig := &domain.OllamaProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderOllama,
			Timeout: cfg.Ollama.Timeout,
		},
		BaseURL:        cfg.Ollama.BaseURL,
		LLMModel:       cfg.Ollama.LLMModel,
		EmbeddingModel: cfg.Ollama.EmbeddingModel,
	}
	
	provider, err := factory.CreateEmbedderProvider(ctx, legacyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create legacy Ollama embedder provider: %w", err)
	}

	return embedder.NewService(provider), nil
}

// InitializeLLM initializes the LLM service using the provider system
func InitializeLLM(ctx context.Context, cfg *config.Config, factory *providers.Factory) (domain.Generator, error) {
	// Check if new provider configuration exists
	if cfg.Providers.ProviderConfigs.Ollama != nil || cfg.Providers.ProviderConfigs.OpenAI != nil || cfg.Providers.ProviderConfigs.LMStudio != nil {
		// Use new provider system
		providerConfig, err := providers.GetLLMProviderConfig(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultLLM)
		if err != nil {
			return nil, fmt.Errorf("failed to get LLM provider config: %w", err)
		}

		provider, err := factory.CreateLLMProvider(ctx, providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM provider: %w", err)
		}

		return llm.NewService(provider), nil
	}

	// Fallback to legacy Ollama configuration for backward compatibility
	// Create legacy Ollama provider using the new provider system
	legacyConfig := &domain.OllamaProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderOllama,
			Timeout: cfg.Ollama.Timeout,
		},
		BaseURL:        cfg.Ollama.BaseURL,
		LLMModel:       cfg.Ollama.LLMModel,
		EmbeddingModel: cfg.Ollama.EmbeddingModel,
	}
	
	provider, err := factory.CreateLLMProvider(ctx, legacyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create legacy Ollama LLM provider: %w", err)
	}

	return llm.NewService(provider), nil
}

// CheckProviderHealth checks the health of provider services
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

// ComposePrompt creates a RAG prompt from chunks and user query
func ComposePrompt(chunks []domain.Chunk, query string) string {
	if len(chunks) == 0 {
		return fmt.Sprintf("Please answer the following question:\n\n%s", query)
	}
	
	var contextParts []string
	for i, chunk := range chunks {
		contextParts = append(contextParts, fmt.Sprintf("[Document Fragment %d]\n%s", i+1, chunk.Content))
	}
	
	context := strings.Join(contextParts, "\n\n")
	prompt := fmt.Sprintf(`Based on the following document content, please answer the user's question. If the documents do not contain relevant information, please indicate that you cannot find an answer from the provided documents.

Document Content:
%s

User Question: %s

Please provide a detailed and accurate answer based on the document content:`, context, query)
	
	return prompt
}