package config

import (
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func TestProviderConfigValidation(t *testing.T) {
	t.Run("ValidOpenAICompatibleProviderConfig", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Port: 7127,
				Host: "localhost",
			},
			Providers: ProvidersConfig{
				DefaultLLM:      "openai",
				DefaultEmbedder: "openai",
				ProviderConfigs: domain.ProviderConfig{
					OpenAI: &domain.OpenAIProviderConfig{
						BaseProviderConfig: domain.BaseProviderConfig{
							Type:    domain.ProviderOpenAI,
							Timeout: 30 * time.Second,
						},
						BaseURL:        "http://localhost:11434",
						APIKey:         "ollama", // Ollama doesn't need real API key
						EmbeddingModel: "nomic-embed-text",
						LLMModel:       "qwen3",
					},
				},
			},
			Sqvect: SqvectConfig{
				DBPath:    "./test.db",
				MaxConns:  10,
				BatchSize: 100,
				TopK:      5,
			},
			Chunker: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Valid OpenAI-compatible provider config should not fail validation: %v", err)
		}
	})

	t.Run("ValidOpenAIProviderConfig", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Port: 7127,
				Host: "localhost",
			},
			Providers: ProvidersConfig{
				DefaultLLM:      "openai",
				DefaultEmbedder: "openai",
				ProviderConfigs: domain.ProviderConfig{
					OpenAI: &domain.OpenAIProviderConfig{
						BaseProviderConfig: domain.BaseProviderConfig{
							Type:    domain.ProviderOpenAI,
							Timeout: 60 * time.Second,
						},
						BaseURL:        "https://api.openai.com/v1",
						APIKey:         "sk-test-key",
						EmbeddingModel: "text-embedding-3-small",
						LLMModel:       "gpt-4",
					},
				},
			},
			Sqvect: SqvectConfig{
				DBPath:    "./test.db",
				MaxConns:  10,
				BatchSize: 100,
				TopK:      5,
			},
			Chunker: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Valid OpenAI provider config should not fail validation: %v", err)
		}
	})
}
