package config

import (
	"os"
	"testing"
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
					VectorDim: 768,
					MaxConns:  10,
					BatchSize: 100,
					TopK:      5,
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
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
