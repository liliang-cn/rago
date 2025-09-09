package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/embedder"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/providers"
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

	// No provider configured
	return nil, fmt.Errorf("no embedder provider configured")
}

// InitializeLLM initializes the LLM service using the provider system
func InitializeLLM(ctx context.Context, cfg *config.Config, factory *providers.Factory) (domain.Generator, error) {
	// Check if new provider configuration exists
	if cfg.Providers.ProviderConfigs.Ollama != nil || cfg.Providers.ProviderConfigs.OpenAI != nil || cfg.Providers.ProviderConfigs.LMStudio != nil {
		// Check if LLM pool is enabled
		if cfg.Providers.LLMPool != nil && cfg.Providers.LLMPool.Enabled {
			poolConfig, err := convertLLMPoolConfig(cfg.Providers.LLMPool)
			if err != nil {
				return nil, fmt.Errorf("failed to convert pool config: %w", err)
			}

			// Build provider configs map for the pool
			providerConfigs := make(map[string]interface{})
			for _, name := range cfg.Providers.LLMPool.Providers {
				var providerConfig interface{}
				switch strings.ToLower(name) {
				case "ollama":
					if cfg.Providers.ProviderConfigs.Ollama == nil {
						return nil, fmt.Errorf("ollama provider configuration not found for pool")
					}
					providerConfig = cfg.Providers.ProviderConfigs.Ollama
				case "openai":
					if cfg.Providers.ProviderConfigs.OpenAI == nil {
						return nil, fmt.Errorf("openai provider configuration not found for pool")
					}
					providerConfig = cfg.Providers.ProviderConfigs.OpenAI
				case "lmstudio":
					if cfg.Providers.ProviderConfigs.LMStudio == nil {
						return nil, fmt.Errorf("lmstudio provider configuration not found for pool")
					}
					providerConfig = cfg.Providers.ProviderConfigs.LMStudio
				default:
					return nil, fmt.Errorf("unsupported provider in pool: %s", name)
				}
				providerConfigs[name] = providerConfig
			}

			pool, err := factory.CreateLLMPool(ctx, providerConfigs, poolConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create LLM pool: %w", err)
			}

			return llm.NewService(pool), nil
		}

		// Use single provider
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

	// No provider configured
	return nil, fmt.Errorf("no LLM provider configured")
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

// convertLLMPoolConfig converts config.LLMPoolConfig to providers.LLMPoolConfig
func convertLLMPoolConfig(cfg *config.LLMPoolConfig) (providers.LLMPoolConfig, error) {
	poolConfig := providers.LLMPoolConfig{
		MaxRetries: cfg.MaxRetries,
	}

	// Convert strategy
	switch strings.ToLower(cfg.Strategy) {
	case "round_robin":
		poolConfig.Strategy = providers.RoundRobinStrategy
	case "random":
		poolConfig.Strategy = providers.RandomStrategy
	case "least_load":
		poolConfig.Strategy = providers.LeastLoadStrategy
	case "failover":
		poolConfig.Strategy = providers.FailoverStrategy
	default:
		poolConfig.Strategy = providers.RoundRobinStrategy // Default
	}

	// Parse health check interval
	if cfg.HealthCheckInterval != "" {
		duration, err := time.ParseDuration(cfg.HealthCheckInterval)
		if err != nil {
			return poolConfig, fmt.Errorf("invalid health check interval: %w", err)
		}
		poolConfig.HealthCheckInterval = duration
	}

	// Parse retry delay
	if cfg.RetryDelay != "" {
		duration, err := time.ParseDuration(cfg.RetryDelay)
		if err != nil {
			return poolConfig, fmt.Errorf("invalid retry delay: %w", err)
		}
		poolConfig.RetryDelay = duration
	}

	return poolConfig, nil
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
