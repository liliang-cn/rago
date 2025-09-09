package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/tools"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Port: 7127,
					Host: "localhost",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath:    "./data/test.db",
					MaxConns:  10,
					BatchSize: 100,
					TopK:      5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
				Tools: tools.DefaultToolConfig(),
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server: ServerConfig{
					Port: 0,
					Host: "localhost",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath: "./data/test.db",
					TopK:   5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server: ServerConfig{
					Port: 70000,
					Host: "localhost",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath: "./data/test.db",
					TopK:   5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
			},
			wantErr: true,
		},
		{
			name: "empty host",
			config: &Config{
				Server: ServerConfig{
					Port: 7127,
					Host: "",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath: "./data/test.db",
					TopK:   5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid chunker method",
			config: &Config{
				Server: ServerConfig{
					Port: 7127,
					Host: "localhost",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath: "./data/test.db",
					TopK:   5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var exists",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "from_env",
			expected:     "from_env",
		},
		{
			name:         "env var does not exist",
			key:          "NON_EXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.key, tt.envValue); err != nil {
					t.Fatalf("Failed to set environment variable: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.key); err != nil {
						t.Logf("Warning: failed to unset environment variable: %v", err)
					}
				}()
			}

			result := GetEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvOrDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvOrDefaultInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid int env var",
			key:          "TEST_INT_VAR",
			defaultValue: 42,
			envValue:     "100",
			expected:     100,
		},
		{
			name:         "invalid int env var",
			key:          "TEST_INVALID_INT_VAR",
			defaultValue: 42,
			envValue:     "not_an_int",
			expected:     42,
		},
		{
			name:         "env var does not exist",
			key:          "NON_EXISTENT_INT_VAR",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.key, tt.envValue); err != nil {
					t.Fatalf("Failed to set environment variable: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.key); err != nil {
						t.Logf("Warning: failed to unset environment variable: %v", err)
					}
				}()
			}

			result := GetEnvOrDefaultInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvOrDefaultInt() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfigValues(t *testing.T) {
	// Test that Load with empty config path returns reasonable defaults
	config, err := Load("")
	if err != nil {
		// If no config file found, that's expected
		// The Load function should still return a config with viper defaults
		t.Skip("No config file found, skipping default config test")
	}
	
	if config == nil {
		t.Fatal("Load returned nil config")
	}
}

func TestLoad_ConfigNotFound(t *testing.T) {
	// Test with non-existent config file
	_, err := Load("non_existent_config.toml")
	if err == nil {
		t.Error("Expected error when loading non-existent config")
	}
}

func TestLoad_EmptyConfigPath(t *testing.T) {
	// This will try to find config in standard locations
	// Should not panic and should return a config with defaults
	config, err := Load("")
	if err != nil && !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "mcpServers.json") {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Even if config file is not found, we should get a config with defaults
	if config == nil && err == nil {
		t.Error("Expected either config or error")
	}
}

func TestConfig_ValidateChunkerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ChunkerConfig
		wantErr bool
	}{
		{
			name: "valid sentence method",
			config: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "sentence",
			},
			wantErr: false,
		},
		{
			name: "valid paragraph method",
			config: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   50,
				Method:    "paragraph",
			},
			wantErr: false,
		},
		{
			name: "zero chunk size",
			config: ChunkerConfig{
				ChunkSize: 0,
				Overlap:   50,
				Method:    "sentence",
			},
			wantErr: true,
		},
		{
			name: "negative overlap",
			config: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   -1,
				Method:    "sentence",
			},
			wantErr: true,
		},
		{
			name: "overlap >= chunk size",
			config: ChunkerConfig{
				ChunkSize: 300,
				Overlap:   300,
				Method:    "sentence",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server: ServerConfig{
					Port: 7127,
					Host: "localhost",
				},
				Ollama: OllamaConfig{
					BaseURL:        "http://localhost:11434",
					EmbeddingModel: "nomic-embed-text",
					LLMModel:       "qwen3",
				},
				Sqvect: SqvectConfig{
					DBPath: "./data/test.db",
					TopK:   5,
				},
				Tools: tools.ToolConfig{
					Enabled:       false,
					SecurityLevel: "normal",
				},
				Chunker: tt.config,
			}
			
			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvOrDefaultBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true env var",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false env var",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "1 env var",
			key:          "TEST_BOOL_1",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "0 env var",
			key:          "TEST_BOOL_0",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "invalid env var",
			key:          "TEST_BOOL_INVALID",
			defaultValue: true,
			envValue:     "invalid",
			expected:     true,
		},
		{
			name:         "empty env var",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.key, tt.envValue); err != nil {
					t.Fatalf("Failed to set environment variable: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(tt.key); err != nil {
						t.Logf("Warning: failed to unset environment variable: %v", err)
					}
				}()
			}

			result := GetEnvOrDefaultBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvOrDefaultBool() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestProvidersConfig(t *testing.T) {
	config := &Config{
		Providers: ProvidersConfig{
			DefaultLLM:      "ollama",
			DefaultEmbedder: "openai",
			ProviderConfigs: domain.ProviderConfig{
				Ollama: &domain.OllamaProviderConfig{
					BaseProviderConfig: domain.BaseProviderConfig{
						Type: domain.ProviderOllama,
					},
					BaseURL: "http://localhost:11434",
				},
			},
		},
	}
	
	if config.Providers.DefaultLLM != "ollama" {
		t.Error("DefaultLLM not set correctly")
	}
	
	if config.Providers.DefaultEmbedder != "openai" {
		t.Error("DefaultEmbedder not set correctly")
	}
	
	if config.Providers.ProviderConfigs.Ollama == nil {
		t.Error("Ollama provider config not set")
	}
}

func TestIngestConfig(t *testing.T) {
	config := &Config{
		Ingest: IngestConfig{
			MetadataExtraction: MetadataExtractionConfig{
				Enable:   true,
				LLMModel: "gpt-4",
			},
		},
	}
	
	if !config.Ingest.MetadataExtraction.Enable {
		t.Error("Metadata extraction not enabled")
	}
	
	if config.Ingest.MetadataExtraction.LLMModel != "gpt-4" {
		t.Error("LLM model not set correctly")
	}
}

func TestOllamaConfig_Timeout(t *testing.T) {
	config := &Config{
		Ollama: OllamaConfig{
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			LLMModel:       "qwen3",
			Timeout:        30 * time.Second,
		},
	}
	
	if config.Ollama.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Ollama.Timeout)
	}
}
