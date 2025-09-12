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

	t.Run("CreateFromMapConfig", func(t *testing.T) {
		// Test creating Ollama provider from map config
		mapConfig := map[string]interface{}{
			"type":            "ollama",
			"base_url":        "http://localhost:11434",
			"llm_model":       "qwen3",
			"embedding_model": "nomic-embed-text",
			"timeout":         "30s",
		}

		provider, err := factory.CreateLLMProvider(ctx, mapConfig)
		if err != nil {
			t.Fatalf("Failed to create LLM provider from map config: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected non-nil provider")
		}
		if provider.ProviderType() != domain.ProviderOllama {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderOllama, provider.ProviderType())
		}

		// Test creating embedder from map config
		embedder, err := factory.CreateEmbedderProvider(ctx, mapConfig)
		if err != nil {
			t.Fatalf("Failed to create embedder provider from map config: %v", err)
		}
		if embedder == nil {
			t.Fatal("Expected non-nil embedder")
		}
		if embedder.ProviderType() != domain.ProviderOllama {
			t.Errorf("Expected embedder type %s, got %s", domain.ProviderOllama, embedder.ProviderType())
		}
	})
}

func TestDetectProviderType(t *testing.T) {
	tests := []struct {
		name        string
		config      interface{}
		expected    domain.ProviderType
		expectError bool
	}{
		{
			name: "Valid ollama type",
			config: map[string]interface{}{
				"type": "ollama",
			},
			expected:    domain.ProviderOllama,
			expectError: false,
		},
		{
			name: "Valid openai type",
			config: map[string]interface{}{
				"type": "openai",
			},
			expected:    domain.ProviderOpenAI,
			expectError: false,
		},
		{
			name: "Valid lmstudio type",
			config: map[string]interface{}{
				"type": "lmstudio",
			},
			expected:    domain.ProviderLMStudio,
			expectError: false,
		},
		{
			name: "Missing type field",
			config: map[string]interface{}{
				"base_url": "http://localhost",
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "Invalid type value",
			config: map[string]interface{}{
				"type": "invalid",
			},
			expected:    "",
			expectError: true,
		},
		{
			name:        "Non-map config",
			config:      "not a map",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectProviderType(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
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

func TestFactory_CreateLMStudioProviders(t *testing.T) {
	factory := NewFactory()
	ctx := context.Background()

	t.Run("CreateLMStudioLLMProvider", func(t *testing.T) {
		config := &domain.LMStudioProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderLMStudio,
			},
			BaseURL:        "http://localhost:1234/v1",
			APIKey:         "test-key",
			EmbeddingModel: "embedding-model",
			LLMModel:       "llm-model",
		}

		provider, err := factory.CreateLLMProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create LMStudio LLM provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderLMStudio {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderLMStudio, provider.ProviderType())
		}
	})

	t.Run("CreateLMStudioEmbedderProvider", func(t *testing.T) {
		config := &domain.LMStudioProviderConfig{
			BaseProviderConfig: domain.BaseProviderConfig{
				Type: domain.ProviderLMStudio,
			},
			BaseURL:        "http://localhost:1234/v1",
			APIKey:         "test-key",
			EmbeddingModel: "embedding-model",
			LLMModel:       "llm-model",
		}

		provider, err := factory.CreateEmbedderProvider(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create LMStudio embedder provider: %v", err)
		}

		if provider.ProviderType() != domain.ProviderLMStudio {
			t.Errorf("Expected provider type %s, got %s", domain.ProviderLMStudio, provider.ProviderType())
		}
	})
}

func TestFactory_Validation(t *testing.T) {
	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory returned nil")
	}
}

func TestDetermineProviderType_AllCases(t *testing.T) {
	testCases := []struct {
		name        string
		config      *domain.ProviderConfig
		expected    domain.ProviderType
		expectError bool
	}{
		{
			name: "LMStudio config",
			config: &domain.ProviderConfig{
				LMStudio: &domain.LMStudioProviderConfig{},
			},
			expected:    domain.ProviderLMStudio,
			expectError: false,
		},
		{
			name: "Multiple configs - Ollama priority",
			config: &domain.ProviderConfig{
				Ollama: &domain.OllamaProviderConfig{},
				OpenAI: &domain.OpenAIProviderConfig{},
			},
			expected:    domain.ProviderOllama,
			expectError: false,
		},
		{
			name:        "Empty config",
			config:      &domain.ProviderConfig{},
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetermineProviderType(tc.config)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestGetProviderConfig_AllProviders(t *testing.T) {
	testCases := []struct {
		name        string
		config      *domain.ProviderConfig
		expectError bool
	}{
		{
			name: "LMStudio provider config",
			config: &domain.ProviderConfig{
				LMStudio: &domain.LMStudioProviderConfig{
					BaseURL: "http://localhost:1234/v1",
					APIKey:  "test-key",
				},
			},
			expectError: false,
		},
		{
			name:        "Empty config",
			config:      &domain.ProviderConfig{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetProviderConfig(tc.config)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectError && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestGetLLMProviderConfig_LMStudio(t *testing.T) {
	config := &domain.ProviderConfig{
		LMStudio: &domain.LMStudioProviderConfig{
			BaseURL: "http://localhost:1234/v1",
		},
	}

	result, err := GetLLMProviderConfig(config, "lmstudio")
	if err != nil {
		t.Fatalf("Failed to get LMStudio provider config: %v", err)
	}

	if result != config.LMStudio {
		t.Error("Expected same config instance")
	}

	// Test missing LMStudio config
	emptyConfig := &domain.ProviderConfig{}
	_, err = GetLLMProviderConfig(emptyConfig, "lmstudio")
	if err == nil {
		t.Error("Expected error for missing lmstudio config")
	}
}

func TestGetEmbedderProviderConfig_LMStudio(t *testing.T) {
	config := &domain.ProviderConfig{
		LMStudio: &domain.LMStudioProviderConfig{
			BaseURL: "http://localhost:1234/v1",
		},
	}

	result, err := GetEmbedderProviderConfig(config, "lmstudio")
	if err != nil {
		t.Fatalf("Failed to get LMStudio embedder config: %v", err)
	}

	if result != config.LMStudio {
		t.Error("Expected same config instance")
	}

	// Test missing LMStudio config
	emptyConfig := &domain.ProviderConfig{}
	_, err = GetEmbedderProviderConfig(emptyConfig, "lmstudio")
	if err == nil {
		t.Error("Expected error for missing lmstudio config")
	}
}
