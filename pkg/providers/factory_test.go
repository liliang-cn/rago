package providers

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func TestFactoryCreateProviders(t *testing.T) {
	factory := NewFactory()
	ctx := context.Background()

	t.Run("CreateOllamaLLMProvider", func(t *testing.T) {
		config := &domain.OllamaProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderOllama,
			},
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			LLMModel:       "qwen3",
		}

		provider, err := factory.CreateLLMProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create Ollama LLM provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderOllama {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderOllama, provider.ProviderType())
		}
	})

	t.Run("CreateOllamaEmbedderProvider", func(t *testing.T) {
		config := &domain.OllamaProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderOllama,
			},
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			LLMModel:       "qwen3",
		}

		provider, err := factory.CreateEmbedderProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create Ollama embedder provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderOllama {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderOllama, provider.ProviderType())
		}
	})

	t.Run("CreateOpenAILLMProvider", func(t *testing.T) {
		config := &domain.OpenAIProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderOpenAI,
			},
			BaseURL:        "https://api.openai.com/v1",
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-3-small",
			LLMModel:       "gpt-4",
		}

		provider, err := factory.CreateLLMProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create OpenAI LLM provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderOpenAI {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderOpenAI, provider.ProviderType())
		}
	})

	t.Run("CreateOpenAIEmbedderProvider", func(t *testing.T) {
		config := &domain.OpenAIProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderOpenAI,
			},
			BaseURL:        "https://api.openai.com/v1",
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-3-small",
			LLMModel:       "gpt-4",
		}

		provider, err := factory.CreateEmbedderProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create OpenAI embedder provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderOpenAI {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderOpenAI, provider.ProviderType())
		}
	})

	t.Run("UnsupportedProviderType", func(t *testing.T) {
		_, err := factory.CreateLLMProvider(ctx, "unsupported-config")
		if err == nil {
			t.Error("Expected error for unsupported provider config type")
		}

		_, err = factory.CreateEmbedderProvider(ctx, "unsupported-config")
		if err == nil {
			t.Error("Expected error for unsupported provider config type")
		}
	})
}

func TestProviderConfigHelpers(t *testing.T) {
	t.Run("DetermineProviderType", func(t *testing.T) {
		// Test Ollama config
		ollamaConfig := &domain.ProviderConfig{
			Ollama: &domain.OllamaProviderConfig{},
		}
		providerType, err := DetermineProviderType(ollamaConfig)
		if err != nil {
			t.Fatalf("Failed to determine provider type for Ollama: %v", err)
		}
		if providerType != domain.ProviderOllama {
			t.Errorf("Expected %s, got %s", domain.ProviderOllama, providerType)
		}

		// Test OpenAI config
		openaiConfig := &domain.ProviderConfig{
			OpenAI: &domain.OpenAIProviderConfig{},
		}
		providerType, err = DetermineProviderType(openaiConfig)
		if err != nil {
			t.Fatalf("Failed to determine provider type for OpenAI: %v", err)
		}
		if providerType != domain.ProviderOpenAI {
			t.Errorf("Expected %s, got %s", domain.ProviderOpenAI, providerType)
		}

		// Test empty config
		emptyConfig := &domain.ProviderConfig{}
		_, err = DetermineProviderType(emptyConfig)
		if err == nil {
			t.Error("Expected error for empty provider config")
		}
	})

	t.Run("GetProviderConfig", func(t *testing.T) {
		ollamaProviderConfig := &domain.OllamaProviderConfig{
			BaseURL: "http://localhost:11434",
		}
		config := &domain.ProviderConfig{
			Ollama: ollamaProviderConfig,
		}

		result, err := GetProviderConfig(config)
		if err != nil {
			t.Fatalf("Failed to get provider config: %v", err)
		}

		if result != ollamaProviderConfig {
			t.Error("Expected same config instance")
		}
	})

	t.Run("GetLLMProviderConfig", func(t *testing.T) {
		ollamaProviderConfig := &domain.OllamaProviderConfig{
			BaseURL: "http://localhost:11434",
		}
		config := &domain.ProviderConfig{
			Ollama: ollamaProviderConfig,
		}

		result, err := GetLLMProviderConfig(config, "ollama")
		if err != nil {
			t.Fatalf("Failed to get LLM provider config: %v", err)
		}

		if result != ollamaProviderConfig {
			t.Error("Expected same config instance")
		}

		// Test unsupported provider
		_, err = GetLLMProviderConfig(config, "unsupported")
		if err == nil {
			t.Error("Expected error for unsupported provider")
		}
	})

	t.Run("GetEmbedderProviderConfig", func(t *testing.T) {
		openaiProviderConfig := &domain.OpenAIProviderConfig{
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
		}
		config := &domain.ProviderConfig{
			OpenAI: openaiProviderConfig,
		}

		result, err := GetEmbedderProviderConfig(config, "openai")
		if err != nil {
			t.Fatalf("Failed to get embedder provider config: %v", err)
		}

		if result != openaiProviderConfig {
			t.Error("Expected same config instance")
		}

		// Test missing config
		_, err = GetEmbedderProviderConfig(config, "ollama")
		if err == nil {
			t.Error("Expected error for missing ollama config")
		}
	})
}