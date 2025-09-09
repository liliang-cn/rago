package providers

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Factory implements the ProviderFactory interface
type Factory struct{}

// NewFactory creates a new provider factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateLLMProvider creates an LLM provider based on the configuration
func (f *Factory) CreateLLMProvider(ctx context.Context, config interface{}) (domain.LLMProvider, error) {
	switch cfg := config.(type) {
	case *domain.OllamaProviderConfig:
		return NewOllamaLLMProvider(cfg)
	case *domain.OpenAIProviderConfig:
		return NewOpenAILLMProvider(cfg)
	case *domain.LMStudioProviderConfig:
		return NewLMStudioLLMProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider config type: %T", config)
	}
}

// CreateEmbedderProvider creates an embedder provider based on the configuration
func (f *Factory) CreateEmbedderProvider(ctx context.Context, config interface{}) (domain.EmbedderProvider, error) {
	switch cfg := config.(type) {
	case *domain.OllamaProviderConfig:
		return NewOllamaEmbedderProvider(cfg)
	case *domain.OpenAIProviderConfig:
		return NewOpenAIEmbedderProvider(cfg)
	case *domain.LMStudioProviderConfig:
		return NewLMStudioEmbedderProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported embedder provider config type: %T", config)
	}
}

// DetermineProviderType determines the provider type from configuration
func DetermineProviderType(config *domain.ProviderConfig) (domain.ProviderType, error) {
	if config.Ollama != nil {
		return domain.ProviderOllama, nil
	}
	if config.OpenAI != nil {
		return domain.ProviderOpenAI, nil
	}
	if config.LMStudio != nil {
		return domain.ProviderLMStudio, nil
	}
	return "", fmt.Errorf("no valid provider configuration found")
}

// GetProviderConfig returns the appropriate provider configuration
func GetProviderConfig(config *domain.ProviderConfig) (interface{}, error) {
	providerType, err := DetermineProviderType(config)
	if err != nil {
		return nil, err
	}

	switch providerType {
	case domain.ProviderOllama:
		return config.Ollama, nil
	case domain.ProviderOpenAI:
		return config.OpenAI, nil
	case domain.ProviderLMStudio:
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetLLMProviderConfig returns the LLM provider configuration based on default settings
func GetLLMProviderConfig(config *domain.ProviderConfig, defaultProvider string) (interface{}, error) {
	switch defaultProvider {
	case "ollama":
		if config.Ollama == nil {
			return nil, fmt.Errorf("ollama provider configuration not found")
		}
		return config.Ollama, nil
	case "openai":
		if config.OpenAI == nil {
			return nil, fmt.Errorf("openai provider configuration not found")
		}
		return config.OpenAI, nil
	case "lmstudio":
		if config.LMStudio == nil {
			return nil, fmt.Errorf("lmstudio provider configuration not found")
		}
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported default LLM provider: %s", defaultProvider)
	}
}

// CreateLLMPool creates a pool of LLM providers from configuration
func (f *Factory) CreateLLMPool(ctx context.Context, providerConfigs map[string]interface{}, poolConfig LLMPoolConfig) (*LLMPool, error) {
	providers := make(map[string]domain.LLMProvider)
	
	for name, config := range providerConfigs {
		provider, err := f.CreateLLMProvider(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
		}
		providers[name] = provider
	}
	
	return NewLLMPool(providers, poolConfig)
}

// GetEmbedderProviderConfig returns the embedder provider configuration based on default settings
func GetEmbedderProviderConfig(config *domain.ProviderConfig, defaultProvider string) (interface{}, error) {
	switch defaultProvider {
	case "ollama":
		if config.Ollama == nil {
			return nil, fmt.Errorf("ollama provider configuration not found")
		}
		return config.Ollama, nil
	case "openai":
		if config.OpenAI == nil {
			return nil, fmt.Errorf("openai provider configuration not found")
		}
		return config.OpenAI, nil
	case "lmstudio":
		if config.LMStudio == nil {
			return nil, fmt.Errorf("lmstudio provider configuration not found")
		}
		return config.LMStudio, nil
	default:
		return nil, fmt.Errorf("unsupported default embedder provider: %s", defaultProvider)
	}
}
