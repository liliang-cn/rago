package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Keyword: KeywordConfig{
					IndexPath: "./data/keyword.bleve",
				},
				Chunker: ChunkerConfig{
					ChunkSize: 300,
					Overlap:   50,
					Method:    "sentence",
				},
				RRF: RRFConfig{
					K:                  10,
					RelevanceThreshold: 0.05,
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
				RRF: RRFConfig{
					K:                  10,
					RelevanceThreshold: 0.05,
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
				RRF: RRFConfig{
					K:                  10,
					RelevanceThreshold: 0.05,
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
				RRF: RRFConfig{
					K:                  10,
					RelevanceThreshold: 0.05,
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
				RRF: RRFConfig{
					K:                  10,
					RelevanceThreshold: 0.05,
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

// Test comprehensive provider configuration validation
func TestConfig_ValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ollama provider config",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "ollama",
					DefaultEmbedder: "ollama",
					ProviderConfigs: domain.ProviderConfig{
						Ollama: &domain.OllamaProviderConfig{
							BaseProviderConfig: domain.BaseProviderConfig{
								Type:    "ollama",
								Timeout: 30 * time.Second,
							},
							BaseURL:        "http://localhost:11434",
							EmbeddingModel: "nomic-embed-text",
							LLMModel:       "qwen3",
						},
					},
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: false,
		},
		{
			name: "valid openai provider config",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "openai",
					DefaultEmbedder: "openai",
					ProviderConfigs: domain.ProviderConfig{
						OpenAI: &domain.OpenAIProviderConfig{
							BaseProviderConfig: domain.BaseProviderConfig{
								Type:    "openai",
								Timeout: 30 * time.Second,
							},
							APIKey:         "sk-test-key",
							EmbeddingModel: "text-embedding-ada-002",
							LLMModel:       "gpt-3.5-turbo",
						},
					},
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: false,
		},
		{
			name: "invalid default provider - not supported",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "unsupported",
					DefaultEmbedder: "ollama",
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "invalid default_llm provider",
		},
		{
			name: "missing provider config for default",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "openai",
					DefaultEmbedder: "openai",
					// Missing OpenAI config
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "openai provider configuration is required",
		},
		{
			name: "invalid ollama config - empty base url",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "ollama",
					DefaultEmbedder: "ollama",
					ProviderConfigs: domain.ProviderConfig{
						Ollama: &domain.OllamaProviderConfig{
							BaseProviderConfig: domain.BaseProviderConfig{
								Type:    "ollama",
								Timeout: 30 * time.Second,
							},
							BaseURL:        "", // Invalid
							EmbeddingModel: "nomic-embed-text",
							LLMModel:       "qwen3",
						},
					},
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "base_url cannot be empty",
		},
		{
			name: "invalid openai config - empty api key",
			config: &Config{
				Server: ServerConfig{Port: 7127, Host: "localhost"},
				Providers: ProvidersConfig{
					DefaultLLM:      "openai",
					DefaultEmbedder: "openai",
					ProviderConfigs: domain.ProviderConfig{
						OpenAI: &domain.OpenAIProviderConfig{
							BaseProviderConfig: domain.BaseProviderConfig{
								Type:    "openai",
								Timeout: 30 * time.Second,
							},
							APIKey:         "", // Invalid
							EmbeddingModel: "text-embedding-ada-002",
							LLMModel:       "gpt-3.5-turbo",
						},
					},
				},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "api_key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test MCP configuration validation
func TestConfig_ValidateMCPConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid MCP config",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
				MCP: mcp.Config{
					Enabled:                true,
					LogLevel:              "info",
					DefaultTimeout:        30 * time.Second,
					MaxConcurrentRequests: 10,
					HealthCheckInterval:   5 * time.Minute,
					Servers: []mcp.ServerConfig{
						{
							Name:         "filesystem",
							Command:      []string{"npx", "@modelcontextprotocol/server-filesystem"},
							MaxRestarts:  3,
							RestartDelay: 5 * time.Second,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disabled MCP config - should skip validation",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
				MCP: mcp.Config{
					Enabled: false,
					// Other invalid values should be ignored
				},
			},
			wantErr: false,
		},
		{
			name: "invalid MCP config - negative timeout",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
				MCP: mcp.Config{
					Enabled:        true,
					DefaultTimeout: -1 * time.Second, // Invalid
				},
			},
			wantErr: true,
			errMsg:  "default_timeout must be positive",
		},
		{
			name: "invalid MCP server config - empty name",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
				MCP: mcp.Config{
					Enabled:                true,
					DefaultTimeout:        30 * time.Second,
					MaxConcurrentRequests: 10,
					HealthCheckInterval:   5 * time.Minute,
					Servers: []mcp.ServerConfig{
						{
							Name:    "", // Invalid
							Command: []string{"test"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "server name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test tools configuration validation
func TestConfig_ValidateToolsConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tools config",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools: tools.ToolConfig{
					Enabled:         true,
					MaxConcurrency:  10,
					CallTimeout:     30 * time.Second,
					SecurityLevel:   "normal",
					LogLevel:        "info",
					EnabledTools:    []string{"filesystem", "calculator"},
					RateLimit: tools.RateLimitConfig{
						CallsPerMinute: 100,
						CallsPerHour:   1000,
						BurstSize:      10,
					},
				},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: false,
		},
		{
			name: "invalid security level",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools: tools.ToolConfig{
					SecurityLevel: "invalid", // Invalid
				},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "invalid security_level",
		},
		{
			name: "negative max concurrency",
			config: &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     RRFConfig{K: 10, RelevanceThreshold: 0.05},
				Tools: tools.ToolConfig{
					MaxConcurrency: -1, // Invalid
				},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			},
			wantErr: true,
			errMsg:  "max_concurrent_calls must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test config loading from different sources
func TestConfig_Load(t *testing.T) {
	t.Run("load from non-existent file uses defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent.toml")
		
		config, err := Load(configPath)
		require.NoError(t, err)
		assert.NotNil(t, config)
		
		// Check some defaults
		assert.Equal(t, 7127, config.Server.Port)
		assert.Equal(t, "0.0.0.0", config.Server.Host)
		assert.Equal(t, "ollama", config.Providers.DefaultLLM)
	})

	t.Run("load with defaults when no config specified", func(t *testing.T) {
		config, err := Load("")
		require.NoError(t, err)
		assert.NotNil(t, config)
		
		// Verify defaults are applied
		assert.Equal(t, 7127, config.Server.Port)
		assert.Equal(t, "sentence", config.Chunker.Method)
		assert.Equal(t, 500, config.Chunker.ChunkSize)
		assert.Equal(t, 10, config.RRF.K)
	})

	t.Run("environment variables override defaults", func(t *testing.T) {
		// Set environment variables
		os.Setenv("RAGO_SERVER_PORT", "8080")
		os.Setenv("RAGO_CHUNKER_CHUNK_SIZE", "1000")
		defer func() {
			os.Unsetenv("RAGO_SERVER_PORT")
			os.Unsetenv("RAGO_CHUNKER_CHUNK_SIZE")
		}()

		config, err := Load("")
		require.NoError(t, err)
		
		assert.Equal(t, 8080, config.Server.Port)
		assert.Equal(t, 1000, config.Chunker.ChunkSize)
	})
}

// Test path expansion
func TestConfig_ExpandPaths(t *testing.T) {
	t.Run("expand tilde paths", func(t *testing.T) {
		config := &Config{
			Sqvect: SqvectConfig{
				DBPath: "~/test/data.db",
			},
			Keyword: KeywordConfig{
				IndexPath: "~/test/index.bleve",
			},
		}

		config.expandPaths()

		// Paths should no longer contain tilde
		assert.NotContains(t, config.Sqvect.DBPath, "~")
		assert.NotContains(t, config.Keyword.IndexPath, "~")
		
		// Should contain expanded home directory
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			assert.Contains(t, config.Sqvect.DBPath, homeDir)
			assert.Contains(t, config.Keyword.IndexPath, homeDir)
		}
	})

	t.Run("leave absolute paths unchanged", func(t *testing.T) {
		config := &Config{
			Sqvect: SqvectConfig{
				DBPath: "/absolute/path/data.db",
			},
			Keyword: KeywordConfig{
				IndexPath: "/absolute/path/index.bleve",
			},
		}

		originalDBPath := config.Sqvect.DBPath
		originalIndexPath := config.Keyword.IndexPath

		config.expandPaths()

		assert.Equal(t, originalDBPath, config.Sqvect.DBPath)
		assert.Equal(t, originalIndexPath, config.Keyword.IndexPath)
	})
}

// Test RRF configuration validation
func TestConfig_ValidateRRFConfig(t *testing.T) {
	tests := []struct {
		name    string
		rrf     RRFConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid RRF config",
			rrf:  RRFConfig{K: 10, RelevanceThreshold: 0.05},
			wantErr: false,
		},
		{
			name: "invalid K - zero",
			rrf:  RRFConfig{K: 0, RelevanceThreshold: 0.05},
			wantErr: true,
			errMsg: "RRF k value must be positive",
		},
		{
			name: "invalid threshold - negative",
			rrf:  RRFConfig{K: 10, RelevanceThreshold: -0.1},
			wantErr: true,
			errMsg: "relevance threshold must be non-negative",
		},
		{
			name: "invalid threshold - too high",
			rrf:  RRFConfig{K: 10, RelevanceThreshold: 1.5},
			wantErr: true,
			errMsg: "relevance threshold must be <= 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Server:  ServerConfig{Port: 7127, Host: "localhost"},
				Sqvect:  SqvectConfig{DBPath: "./test.db", TopK: 5},
				Keyword: KeywordConfig{IndexPath: "./test.bleve"},
				Chunker: ChunkerConfig{ChunkSize: 300, Overlap: 50, Method: "sentence"},
				RRF:     tt.rrf,
				Tools:   tools.ToolConfig{SecurityLevel: "normal"},
				Ollama:  OllamaConfig{BaseURL: "http://localhost:11434", EmbeddingModel: "nomic-embed-text", LLMModel: "qwen3"},
			}

			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
