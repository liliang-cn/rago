package config

import (
	"testing"
	"time"

	"github.com/liliang-cn/rago/pkg/domain"
	"github.com/liliang-cn/rago/pkg/tools"
)

func TestProviderConfigValidation(t *testing.T) {
	t.Run("ValidOllamaProviderConfig", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Port: 7127,
				Host: "localhost",
			},
			Providers: ProvidersConfig{
				DefaultLLM:      "ollama",
				DefaultEmbedder: "ollama",
				ProviderConfigs: domain.ProviderConfig{
					Ollama: &domain.OllamaProviderConfig{
						BaseProviderConfig: domain.BaseProviderConfig{
							Type:    domain.ProviderOllama,
							Timeout: 30 * time.Second,
						},
						BaseURL:        "http://localhost:11434",
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
			Keyword: KeywordConfig{
				IndexPath: "./test.bleve",
			},
			Chunker: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
			Tools: tools.ToolConfig{
				Enabled:       false,
				SecurityLevel: "normal",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Valid Ollama provider config should not fail validation: %v", err)
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
			Keyword: KeywordConfig{
				IndexPath: "./test.bleve",
			},
			Chunker: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
			Tools: tools.ToolConfig{
				Enabled:       false,
				SecurityLevel: "normal",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Valid OpenAI provider config should not fail validation: %v", err)
		}
	})

	t.Run("BackwardCompatibilityMode", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Port: 7127,
				Host: "localhost",
			},
			// No Providers config, should fall back to legacy Ollama config
			Ollama: OllamaConfig{
				BaseURL:        "http://localhost:11434",
				EmbeddingModel: "nomic-embed-text",
				LLMModel:       "qwen3",
				Timeout:        30 * time.Second,
			},
			Sqvect: SqvectConfig{
				DBPath:    "./test.db",
				MaxConns:  10,
				BatchSize: 100,
				TopK:      5,
			},
			Keyword: KeywordConfig{
				IndexPath: "./test.bleve",
			},
			Chunker: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
			Tools: tools.ToolConfig{
				Enabled:       false,
				SecurityLevel: "normal",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Backward compatibility mode should not fail validation: %v", err)
		}
	})
}