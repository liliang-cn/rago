package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/tools"
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig     `mapstructure:"server"`
	Providers ProvidersConfig  `mapstructure:"providers"`
	Sqvect    SqvectConfig     `mapstructure:"sqvect"`
	Keyword   KeywordConfig    `mapstructure:"keyword"`
	Chunker   ChunkerConfig    `mapstructure:"chunker"`
	Ingest    IngestConfig     `mapstructure:"ingest"`
	Tools     tools.ToolConfig `mapstructure:"tools"`
	MCP       mcp.Config       `mapstructure:"mcp"`
	RRF       RRFConfig        `mapstructure:"rrf"`
	Agents    AgentsConfig     `mapstructure:"agents"`
	Mode      ModeConfig       `mapstructure:"mode"`

	// Deprecated: Use Providers instead
	Ollama OllamaConfig `mapstructure:"ollama"`
}

type ProvidersConfig struct {
	// The default provider to use for LLM operations
	DefaultLLM string `mapstructure:"default_llm"`
	// The default provider to use for embedding operations
	DefaultEmbedder string `mapstructure:"default_embedder"`
	// Provider configurations
	ProviderConfigs domain.ProviderConfig `mapstructure:",squash"`
}

type IngestConfig struct {
	MetadataExtraction MetadataExtractionConfig `mapstructure:"metadata_extraction"`
}

type MetadataExtractionConfig struct {
	Enable   bool   `mapstructure:"enable"`
	LLMModel string `mapstructure:"llm_model"`
}

type ServerConfig struct {
	Port        int      `mapstructure:"port"`
	Host        string   `mapstructure:"host"`
	EnableUI    bool     `mapstructure:"enable_ui"`
	CORSOrigins []string `mapstructure:"cors_origins"`
}

type OllamaConfig struct {
	EmbeddingModel string        `mapstructure:"embedding_model"`
	LLMModel       string        `mapstructure:"llm_model"`
	BaseURL        string        `mapstructure:"base_url"`
	Timeout        time.Duration `mapstructure:"timeout"`
}

type SqvectConfig struct {
	DBPath    string  `mapstructure:"db_path"`
	MaxConns  int     `mapstructure:"max_conns"`
	BatchSize int     `mapstructure:"batch_size"`
	TopK      int     `mapstructure:"top_k"`
	Threshold float64 `mapstructure:"threshold"`
}

type KeywordConfig struct {
	IndexPath string `mapstructure:"index_path"`
}

type ChunkerConfig struct {
	ChunkSize int    `mapstructure:"chunk_size"`
	Overlap   int    `mapstructure:"overlap"`
	Method    string `mapstructure:"method"`
}

type RRFConfig struct {
	K                  int     `mapstructure:"k"`                   // RRF constant (default: 10)
	RelevanceThreshold float64 `mapstructure:"relevance_threshold"` // Threshold for considering context relevant (default: 0.05)
}

// AgentsConfig holds agent system configuration
type AgentsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// ModeConfig defines operational modes for RAGO
type ModeConfig struct {
	// RAGOnly disables MCP and agent features, running pure RAG mode
	RAGOnly bool `mapstructure:"rag_only"`
	// DisableMCP specifically disables MCP even if tools are configured
	DisableMCP bool `mapstructure:"disable_mcp"`
}

func Load(configPath string) (*Config, error) {
	config := &Config{}

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Try multiple locations in order of preference
		var configFound bool
		homeDir, _ := os.UserHomeDir()

		// Priority order:
		// 1. ./rago.toml (current directory)
		// 2. ./.rago/rago.toml (current directory .rago folder)
		// 3. ~/.rago/rago.toml (user home directory)

		configPaths := []string{
			"rago.toml",                         // Current directory
			filepath.Join(".rago", "rago.toml"), // Current .rago directory
		}

		// Add home directory path if available
		if homeDir != "" {
			configPaths = append(configPaths, filepath.Join(homeDir, ".rago", "rago.toml"))
		}

		// Try each path in order
		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				viper.SetConfigFile(path)
				configFound = true
				break
			}
		}

		// If no config found, use default path (will use built-in defaults)
		if !configFound {
			if homeDir != "" {
				viper.SetConfigFile(filepath.Join(homeDir, ".rago", "rago.toml"))
			} else {
				viper.SetConfigFile("rago.toml")
			}
		}
	}

	setDefaults()
	bindEnvVars()

	// Try to read config file, but don't fail if it doesn't exist
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is OK - we'll use defaults
		// Only return error for actual read/parse errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Check if it's any kind of "file not found" error
			errStr := err.Error()
			if !strings.Contains(errStr, "no such file") &&
				!strings.Contains(errStr, "cannot find the file") &&
				!strings.Contains(errStr, "not found") {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
		// Config file not found - that's OK, we'll use defaults
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Auto-configure metadata extraction LLM model if not set
	if config.Ingest.MetadataExtraction.Enable && config.Ingest.MetadataExtraction.LLMModel == "" {
		config.Ingest.MetadataExtraction.LLMModel = config.getDefaultLLMModel()
	}

	// Initialize MCP config if not properly loaded
	if config.MCP.DefaultTimeout == 0 {
		defaultMCP := mcp.DefaultConfig()
		if config.MCP.LogLevel == "" {
			config.MCP = defaultMCP
		} else {
			// Preserve any values that were set
			if config.MCP.DefaultTimeout == 0 {
				config.MCP.DefaultTimeout = defaultMCP.DefaultTimeout
			}
			if config.MCP.MaxConcurrentRequests == 0 {
				config.MCP.MaxConcurrentRequests = defaultMCP.MaxConcurrentRequests
			}
			if config.MCP.HealthCheckInterval == 0 {
				config.MCP.HealthCheckInterval = defaultMCP.HealthCheckInterval
			}
		}
	}

	// Load MCP servers from external JSON file if specified
	if err := config.MCP.LoadServersFromJSON(); err != nil {
		return nil, fmt.Errorf("failed to load MCP servers from JSON: %w", err)
	}

	// Expand home directory paths
	config.expandPaths()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 7127)
	viper.SetDefault("server.host", "0.0.0.0") // 支持局域网访问
	viper.SetDefault("server.enable_ui", false)
	viper.SetDefault("server.cors_origins", []string{"*"})

	// Provider defaults
	viper.SetDefault("providers.default_llm", "ollama")
	viper.SetDefault("providers.default_embedder", "ollama")

	// Ollama provider defaults (backward compatibility)
	viper.SetDefault("providers.ollama.type", "ollama")
	viper.SetDefault("providers.ollama.embedding_model", "nomic-embed-text")
	viper.SetDefault("providers.ollama.llm_model", "qwen3")
	viper.SetDefault("providers.ollama.base_url", "http://localhost:11434")
	viper.SetDefault("providers.ollama.timeout", "30s")

	// Deprecated: Keep for backward compatibility
	viper.SetDefault("ollama.embedding_model", "nomic-embed-text")
	viper.SetDefault("ollama.llm_model", "qwen3")
	viper.SetDefault("ollama.base_url", "http://localhost:11434")
	viper.SetDefault("ollama.timeout", "30s")

	viper.SetDefault("sqvect.db_path", "~/.rago/rag.db")
	viper.SetDefault("sqvect.max_conns", 10)
	viper.SetDefault("sqvect.batch_size", 100)
	viper.SetDefault("sqvect.top_k", 5)
	viper.SetDefault("sqvect.threshold", 0.0)

	viper.SetDefault("keyword.index_path", "~/.rago/keyword.bleve")

	viper.SetDefault("rrf.k", 10)
	viper.SetDefault("rrf.relevance_threshold", 0.05)

	viper.SetDefault("chunker.chunk_size", 500)
	viper.SetDefault("chunker.overlap", 50)
	viper.SetDefault("chunker.method", "sentence")

	viper.SetDefault("ingest.metadata_extraction.enable", false)
	// Note: llm_model will be auto-configured to use default LLM if not set

	// Tools configuration defaults
	toolConfig := tools.DefaultToolConfig()
	viper.SetDefault("tools.enabled", toolConfig.Enabled)
	viper.SetDefault("tools.max_concurrent_calls", toolConfig.MaxConcurrency)
	viper.SetDefault("tools.call_timeout", toolConfig.CallTimeout)
	viper.SetDefault("tools.security_level", toolConfig.SecurityLevel)
	viper.SetDefault("tools.enabled_tools", toolConfig.EnabledTools)
	viper.SetDefault("tools.log_level", toolConfig.LogLevel)
	viper.SetDefault("tools.rate_limit.calls_per_minute", toolConfig.RateLimit.CallsPerMinute)
	viper.SetDefault("tools.rate_limit.calls_per_hour", toolConfig.RateLimit.CallsPerHour)
	viper.SetDefault("tools.rate_limit.burst_size", toolConfig.RateLimit.BurstSize)
	// Built-in tools have been removed - use MCP servers instead

	// Plugin configuration defaults
	viper.SetDefault("tools.plugins.enabled", toolConfig.Plugins.Enabled)
	viper.SetDefault("tools.plugins.plugin_paths", toolConfig.Plugins.PluginPaths)
	viper.SetDefault("tools.plugins.auto_load", toolConfig.Plugins.AutoLoad)

	// MCP configuration defaults
	mcpConfig := mcp.DefaultConfig()
	viper.SetDefault("mcp.enabled", mcpConfig.Enabled)
	viper.SetDefault("mcp.log_level", mcpConfig.LogLevel)
	viper.SetDefault("mcp.default_timeout", mcpConfig.DefaultTimeout)
	viper.SetDefault("mcp.max_concurrent_requests", mcpConfig.MaxConcurrentRequests)
	viper.SetDefault("mcp.health_check_interval", mcpConfig.HealthCheckInterval)
	viper.SetDefault("mcp.servers_config_path", mcpConfig.ServersConfigPath)
	viper.SetDefault("mcp.servers", mcpConfig.Servers)
}

func bindEnvVars() {
	viper.SetEnvPrefix("RAGO")
	viper.AutomaticEnv()

	if err := viper.BindEnv("server.port", "RAGO_SERVER_PORT"); err != nil {
		log.Printf("Warning: failed to bind server.port env var: %v", err)
	}
	if err := viper.BindEnv("server.host", "RAGO_SERVER_HOST"); err != nil {
		log.Printf("Warning: failed to bind server.host env var: %v", err)
	}
	if err := viper.BindEnv("server.enable_ui", "RAGO_SERVER_ENABLE_UI"); err != nil {
		log.Printf("Warning: failed to bind server.enable_ui env var: %v", err)
	}

	// Provider environment variables
	if err := viper.BindEnv("providers.default_llm", "RAGO_PROVIDERS_DEFAULT_LLM"); err != nil {
		log.Printf("Warning: failed to bind providers.default_llm env var: %v", err)
	}
	if err := viper.BindEnv("providers.default_embedder", "RAGO_PROVIDERS_DEFAULT_EMBEDDER"); err != nil {
		log.Printf("Warning: failed to bind providers.default_embedder env var: %v", err)
	}

	// Ollama provider environment variables
	if err := viper.BindEnv("providers.ollama.embedding_model", "RAGO_OLLAMA_EMBEDDING_MODEL"); err != nil {
		log.Printf("Warning: failed to bind providers.ollama.embedding_model env var: %v", err)
	}
	if err := viper.BindEnv("providers.ollama.llm_model", "RAGO_OLLAMA_LLM_MODEL"); err != nil {
		log.Printf("Warning: failed to bind providers.ollama.llm_model env var: %v", err)
	}
	if err := viper.BindEnv("providers.ollama.base_url", "RAGO_OLLAMA_BASE_URL"); err != nil {
		log.Printf("Warning: failed to bind providers.ollama.base_url env var: %v", err)
	}
	if err := viper.BindEnv("providers.ollama.timeout", "RAGO_OLLAMA_TIMEOUT"); err != nil {
		log.Printf("Warning: failed to bind providers.ollama.timeout env var: %v", err)
	}

	// OpenAI provider environment variables
	if err := viper.BindEnv("providers.openai.api_key", "RAGO_OPENAI_API_KEY"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.api_key env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.base_url", "RAGO_OPENAI_BASE_URL"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.base_url env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.embedding_model", "RAGO_OPENAI_EMBEDDING_MODEL"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.embedding_model env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.llm_model", "RAGO_OPENAI_LLM_MODEL"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.llm_model env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.organization", "RAGO_OPENAI_ORGANIZATION"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.organization env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.project", "RAGO_OPENAI_PROJECT"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.project env var: %v", err)
	}
	if err := viper.BindEnv("providers.openai.timeout", "RAGO_OPENAI_TIMEOUT"); err != nil {
		log.Printf("Warning: failed to bind providers.openai.timeout env var: %v", err)
	}

	// Deprecated: Keep for backward compatibility
	if err := viper.BindEnv("ollama.embedding_model", "RAGO_OLLAMA_EMBEDDING_MODEL"); err != nil {
		log.Printf("Warning: failed to bind ollama.embedding_model env var: %v", err)
	}
	if err := viper.BindEnv("ollama.llm_model", "RAGO_OLLAMA_LLM_MODEL"); err != nil {
		log.Printf("Warning: failed to bind ollama.llm_model env var: %v", err)
	}
	if err := viper.BindEnv("ollama.base_url", "RAGO_OLLAMA_BASE_URL"); err != nil {
		log.Printf("Warning: failed to bind ollama.base_url env var: %v", err)
	}
	if err := viper.BindEnv("ollama.timeout", "RAGO_OLLAMA_TIMEOUT"); err != nil {
		log.Printf("Warning: failed to bind ollama.timeout env var: %v", err)
	}

	if err := viper.BindEnv("sqvect.db_path", "RAGO_SQVECT_DB_PATH"); err != nil {
		log.Printf("Warning: failed to bind sqvect.db_path env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.vector_dim", "RAGO_SQVECT_VECTOR_DIM"); err != nil {
		log.Printf("Warning: failed to bind sqvect.vector_dim env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.max_conns", "RAGO_SQVECT_MAX_CONNS"); err != nil {
		log.Printf("Warning: failed to bind sqvect.max_conns env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.batch_size", "RAGO_SQVECT_BATCH_SIZE"); err != nil {
		log.Printf("Warning: failed to bind sqvect.batch_size env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.top_k", "RAGO_SQVECT_TOP_K"); err != nil {
		log.Printf("Warning: failed to bind sqvect.top_k env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.threshold", "RAGO_SQVECT_THRESHOLD"); err != nil {
		log.Printf("Warning: failed to bind sqvect.threshold env var: %v", err)
	}

	if err := viper.BindEnv("keyword.index_path", "RAGO_KEYWORD_INDEX_PATH"); err != nil {
		log.Printf("Warning: failed to bind keyword.index_path env var: %v", err)
	}

	if err := viper.BindEnv("rrf.k", "RAGO_RRF_K"); err != nil {
		log.Printf("Warning: failed to bind rrf.k env var: %v", err)
	}
	if err := viper.BindEnv("rrf.relevance_threshold", "RAGO_RRF_RELEVANCE_THRESHOLD"); err != nil {
		log.Printf("Warning: failed to bind rrf.relevance_threshold env var: %v", err)
	}

	if err := viper.BindEnv("chunker.chunk_size", "RAGO_CHUNKER_CHUNK_SIZE"); err != nil {
		log.Printf("Warning: failed to bind chunker.chunk_size env var: %v", err)
	}
	if err := viper.BindEnv("chunker.overlap", "RAGO_CHUNKER_OVERLAP"); err != nil {
		log.Printf("Warning: failed to bind chunker.overlap env var: %v", err)
	}
	if err := viper.BindEnv("chunker.method", "RAGO_CHUNKER_METHOD"); err != nil {
		log.Printf("Warning: failed to bind chunker.method env var: %v", err)
	}

	if err := viper.BindEnv("ingest.metadata_extraction.enable", "RAGO_INGEST_METADATA_EXTRACTION_ENABLE"); err != nil {
		log.Printf("Warning: failed to bind ingest.metadata_extraction.enable env var: %v", err)
	}
	if err := viper.BindEnv("ingest.metadata_extraction.llm_model", "RAGO_INGEST_METADATA_EXTRACTION_LLM_MODEL"); err != nil {
		log.Printf("Warning: failed to bind ingest.metadata_extraction.llm_model env var: %v", err)
	}

	// Tools environment variables
	if err := viper.BindEnv("tools.enabled", "RAGO_TOOLS_ENABLED"); err != nil {
		log.Printf("Warning: failed to bind tools.enabled env var: %v", err)
	}
	if err := viper.BindEnv("tools.max_concurrent_calls", "RAGO_TOOLS_MAX_CONCURRENT_CALLS"); err != nil {
		log.Printf("Warning: failed to bind tools.max_concurrent_calls env var: %v", err)
	}
	if err := viper.BindEnv("tools.call_timeout", "RAGO_TOOLS_CALL_TIMEOUT"); err != nil {
		log.Printf("Warning: failed to bind tools.call_timeout env var: %v", err)
	}
	if err := viper.BindEnv("tools.security_level", "RAGO_TOOLS_SECURITY_LEVEL"); err != nil {
		log.Printf("Warning: failed to bind tools.security_level env var: %v", err)
	}
	if err := viper.BindEnv("tools.log_level", "RAGO_TOOLS_LOG_LEVEL"); err != nil {
		log.Printf("Warning: failed to bind tools.log_level env var: %v", err)
	}

	// Plugin environment variables
	if err := viper.BindEnv("tools.plugins.enabled", "RAGO_TOOLS_PLUGINS_ENABLED"); err != nil {
		log.Printf("Warning: failed to bind tools.plugins.enabled env var: %v", err)
	}
	if err := viper.BindEnv("tools.plugins.auto_load", "RAGO_TOOLS_PLUGINS_AUTO_LOAD"); err != nil {
		log.Printf("Warning: failed to bind tools.plugins.auto_load env var: %v", err)
	}

	// MCP environment variables
	if err := viper.BindEnv("mcp.enabled", "RAGO_MCP_ENABLED"); err != nil {
		log.Printf("Warning: failed to bind mcp.enabled env var: %v", err)
	}
	if err := viper.BindEnv("mcp.log_level", "RAGO_MCP_LOG_LEVEL"); err != nil {
		log.Printf("Warning: failed to bind mcp.log_level env var: %v", err)
	}
	if err := viper.BindEnv("mcp.default_timeout", "RAGO_MCP_DEFAULT_TIMEOUT"); err != nil {
		log.Printf("Warning: failed to bind mcp.default_timeout env var: %v", err)
	}
	if err := viper.BindEnv("mcp.max_concurrent_requests", "RAGO_MCP_MAX_CONCURRENT_REQUESTS"); err != nil {
		log.Printf("Warning: failed to bind mcp.max_concurrent_requests env var: %v", err)
	}
	if err := viper.BindEnv("mcp.health_check_interval", "RAGO_MCP_HEALTH_CHECK_INTERVAL"); err != nil {
		log.Printf("Warning: failed to bind mcp.health_check_interval env var: %v", err)
	}
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	// Validate provider configurations only if they exist
	if c.Providers.DefaultLLM != "" || c.Providers.DefaultEmbedder != "" ||
		c.Providers.ProviderConfigs.Ollama != nil || c.Providers.ProviderConfigs.OpenAI != nil {
		if err := c.validateProviderConfig(); err != nil {
			return fmt.Errorf("invalid provider configuration: %w", err)
		}
	}

	// Backward compatibility: validate deprecated ollama config if new config is not provided
	if c.Providers.ProviderConfigs.Ollama == nil && c.Providers.ProviderConfigs.OpenAI == nil {
		if err := c.validateLegacyOllamaConfig(); err != nil {
			return fmt.Errorf("invalid ollama configuration: %w", err)
		}
	}

	if c.Sqvect.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if c.Keyword.IndexPath == "" {
		return fmt.Errorf("keyword index path cannot be empty")
	}

	if c.Sqvect.TopK <= 0 {
		return fmt.Errorf("topK must be positive: %d", c.Sqvect.TopK)
	}

	if c.Sqvect.Threshold < 0 {
		return fmt.Errorf("threshold must be non-negative: %f", c.Sqvect.Threshold)
	}

	if c.Chunker.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive: %d", c.Chunker.ChunkSize)
	}

	if c.Chunker.Overlap < 0 || c.Chunker.Overlap >= c.Chunker.ChunkSize {
		return fmt.Errorf("overlap must be between 0 and chunk size: %d", c.Chunker.Overlap)
	}

	validMethods := map[string]bool{"sentence": true, "paragraph": true, "token": true}
	if !validMethods[c.Chunker.Method] {
		return fmt.Errorf("invalid chunker method: %s", c.Chunker.Method)
	}

	if c.Ingest.MetadataExtraction.Enable && c.Ingest.MetadataExtraction.LLMModel == "" {
		return fmt.Errorf("llm_model for metadata extraction should be auto-configured but is still empty")
	}

	// Validate tools configuration
	if err := c.validateToolsConfig(); err != nil {
		return fmt.Errorf("invalid tools configuration: %w", err)
	}

	// Validate MCP configuration
	if err := c.validateMCPConfig(); err != nil {
		return fmt.Errorf("invalid MCP configuration: %w", err)
	}

	// Validate RRF configuration
	if err := c.validateRRFConfig(); err != nil {
		return fmt.Errorf("invalid RRF configuration: %w", err)
	}

	return nil
}

// getDefaultLLMModel returns the appropriate LLM model based on configuration
func (c *Config) getDefaultLLMModel() string {
	// Use new provider system if available
	if c.Providers.ProviderConfigs.Ollama != nil {
		return c.Providers.ProviderConfigs.Ollama.LLMModel
	}
	if c.Providers.ProviderConfigs.OpenAI != nil {
		return c.Providers.ProviderConfigs.OpenAI.LLMModel
	}
	if c.Providers.ProviderConfigs.LMStudio != nil {
		return c.Providers.ProviderConfigs.LMStudio.LLMModel
	}

	// Fallback to legacy ollama config
	if c.Ollama.LLMModel != "" {
		return c.Ollama.LLMModel
	}

	// Final fallback
	return "qwen3"
}

// validateProviderConfig validates the new provider configuration
func (c *Config) validateProviderConfig() error {
	// Validate default provider settings
	if c.Providers.DefaultLLM == "" {
		return fmt.Errorf("default_llm cannot be empty")
	}
	if c.Providers.DefaultEmbedder == "" {
		return fmt.Errorf("default_embedder cannot be empty")
	}

	validProviders := map[string]bool{"ollama": true, "openai": true, "lmstudio": true}
	if !validProviders[c.Providers.DefaultLLM] {
		return fmt.Errorf("invalid default_llm provider: %s (supported: ollama, openai, lmstudio)", c.Providers.DefaultLLM)
	}
	if !validProviders[c.Providers.DefaultEmbedder] {
		return fmt.Errorf("invalid default_embedder provider: %s (supported: ollama, openai, lmstudio)", c.Providers.DefaultEmbedder)
	}

	// Validate individual provider configurations
	if c.Providers.ProviderConfigs.Ollama != nil {
		if err := c.validateOllamaProviderConfig(c.Providers.ProviderConfigs.Ollama); err != nil {
			return fmt.Errorf("invalid ollama provider config: %w", err)
		}
	}

	if c.Providers.ProviderConfigs.OpenAI != nil {
		if err := c.validateOpenAIProviderConfig(c.Providers.ProviderConfigs.OpenAI); err != nil {
			return fmt.Errorf("invalid openai provider config: %w", err)
		}
	}

	if c.Providers.ProviderConfigs.LMStudio != nil {
		if err := c.validateLMStudioProviderConfig(c.Providers.ProviderConfigs.LMStudio); err != nil {
			return fmt.Errorf("invalid lmstudio provider config: %w", err)
		}
	}

	// Ensure the default providers have corresponding configurations
	if c.Providers.DefaultLLM == "ollama" || c.Providers.DefaultEmbedder == "ollama" {
		if c.Providers.ProviderConfigs.Ollama == nil {
			return fmt.Errorf("ollama provider configuration is required when using ollama as default provider")
		}
	}
	if c.Providers.DefaultLLM == "openai" || c.Providers.DefaultEmbedder == "openai" {
		if c.Providers.ProviderConfigs.OpenAI == nil {
			return fmt.Errorf("openai provider configuration is required when using openai as default provider")
		}
	}
	if c.Providers.DefaultLLM == "lmstudio" || c.Providers.DefaultEmbedder == "lmstudio" {
		if c.Providers.ProviderConfigs.LMStudio == nil {
			return fmt.Errorf("lmstudio provider configuration is required when using lmstudio as default provider")
		}
	}

	return nil
}

// validateOllamaProviderConfig validates Ollama provider configuration
func (c *Config) validateOllamaProviderConfig(config *domain.OllamaProviderConfig) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base_url cannot be empty")
	}
	if config.EmbeddingModel == "" {
		return fmt.Errorf("embedding_model cannot be empty")
	}
	if config.LLMModel == "" {
		return fmt.Errorf("llm_model cannot be empty")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive: %v", config.Timeout)
	}
	return nil
}

// validateOpenAIProviderConfig validates OpenAI provider configuration
func (c *Config) validateOpenAIProviderConfig(config *domain.OpenAIProviderConfig) error {
	if config.APIKey == "" {
		return fmt.Errorf("api_key cannot be empty")
	}
	if config.EmbeddingModel == "" {
		return fmt.Errorf("embedding_model cannot be empty")
	}
	if config.LLMModel == "" {
		return fmt.Errorf("llm_model cannot be empty")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive: %v", config.Timeout)
	}
	return nil
}

// validateLMStudioProviderConfig validates LM Studio provider configuration
func (c *Config) validateLMStudioProviderConfig(config *domain.LMStudioProviderConfig) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base_url cannot be empty")
	}
	if config.LLMModel == "" {
		return fmt.Errorf("llm_model cannot be empty")
	}
	if config.EmbeddingModel == "" {
		return fmt.Errorf("embedding_model cannot be empty")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive: %v", config.Timeout)
	}
	return nil
}

// validateLegacyOllamaConfig validates the deprecated ollama configuration
func (c *Config) validateLegacyOllamaConfig() error {
	if c.Ollama.BaseURL == "" {
		return fmt.Errorf("ollama base URL cannot be empty")
	}
	if c.Ollama.EmbeddingModel == "" {
		return fmt.Errorf("embedding model cannot be empty")
	}
	if c.Ollama.LLMModel == "" {
		return fmt.Errorf("LLM model cannot be empty")
	}
	return nil
}

// validateToolsConfig validates the tools configuration
func (c *Config) validateToolsConfig() error {
	if c.Tools.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrent_calls must be non-negative: %d", c.Tools.MaxConcurrency)
	}

	if c.Tools.CallTimeout < 0 {
		return fmt.Errorf("call_timeout must be non-negative: %v", c.Tools.CallTimeout)
	}

	validSecurityLevels := map[string]bool{
		"strict":     true,
		"normal":     true,
		"permissive": true,
	}
	if !validSecurityLevels[c.Tools.SecurityLevel] {
		return fmt.Errorf("invalid security_level: %s (must be strict, normal, or permissive)", c.Tools.SecurityLevel)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if c.Tools.LogLevel != "" && !validLogLevels[c.Tools.LogLevel] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.Tools.LogLevel)
	}

	if c.Tools.RateLimit.CallsPerMinute < 0 {
		return fmt.Errorf("rate_limit.calls_per_minute must be non-negative: %d", c.Tools.RateLimit.CallsPerMinute)
	}

	if c.Tools.RateLimit.CallsPerHour < 0 {
		return fmt.Errorf("rate_limit.calls_per_hour must be non-negative: %d", c.Tools.RateLimit.CallsPerHour)
	}

	if c.Tools.RateLimit.BurstSize < 0 {
		return fmt.Errorf("rate_limit.burst_size must be non-negative: %d", c.Tools.RateLimit.BurstSize)
	}

	return nil
}

func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func GetEnvOrDefaultBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// validateMCPConfig validates the MCP configuration
func (c *Config) validateMCPConfig() error {
	if !c.MCP.Enabled {
		return nil // Skip validation if MCP is disabled
	}

	if c.MCP.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive: %v", c.MCP.DefaultTimeout)
	}

	if c.MCP.MaxConcurrentRequests < 0 {
		return fmt.Errorf("max_concurrent_requests must be non-negative: %d", c.MCP.MaxConcurrentRequests)
	}

	if c.MCP.HealthCheckInterval <= 0 {
		return fmt.Errorf("health_check_interval must be positive: %v", c.MCP.HealthCheckInterval)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if c.MCP.LogLevel != "" && !validLogLevels[c.MCP.LogLevel] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.MCP.LogLevel)
	}

	// Validate server configurations
	for i, server := range c.MCP.Servers {
		if err := c.validateMCPServerConfig(&server, i); err != nil {
			return fmt.Errorf("invalid MCP server config at index %d: %w", i, err)
		}
	}

	return nil
}

// validateMCPServerConfig validates individual MCP server configuration
func (c *Config) validateMCPServerConfig(server *mcp.ServerConfig, index int) error {
	if server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if len(server.Command) == 0 {
		return fmt.Errorf("server command cannot be empty")
	}

	if server.MaxRestarts < 0 {
		return fmt.Errorf("max_restarts must be non-negative: %d", server.MaxRestarts)
	}

	if server.RestartDelay < 0 {
		return fmt.Errorf("restart_delay must be non-negative: %v", server.RestartDelay)
	}

	// Check for duplicate server names
	for j, other := range c.MCP.Servers {
		if j != index && other.Name == server.Name {
			return fmt.Errorf("duplicate server name: %s", server.Name)
		}
	}

	return nil
}

// validateRRFConfig validates RRF configuration
func (c *Config) validateRRFConfig() error {
	if c.RRF.K <= 0 {
		return fmt.Errorf("RRF k value must be positive: %d", c.RRF.K)
	}

	if c.RRF.RelevanceThreshold < 0 {
		return fmt.Errorf("relevance threshold must be non-negative: %f", c.RRF.RelevanceThreshold)
	}

	if c.RRF.RelevanceThreshold > 1.0 {
		return fmt.Errorf("relevance threshold must be <= 1.0: %f", c.RRF.RelevanceThreshold)
	}

	return nil
}

// expandPaths expands ~ to home directory in file paths
func (c *Config) expandPaths() {
	c.Sqvect.DBPath = expandHomePath(c.Sqvect.DBPath)
	c.Keyword.IndexPath = expandHomePath(c.Keyword.IndexPath)

	// Ensure directories exist for default paths
	ensureParentDir(c.Sqvect.DBPath)
	ensureParentDir(c.Keyword.IndexPath)
}

// expandHomePath expands ~ to home directory
func expandHomePath(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to original path if can't get home directory
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}

	return path
}

// ensureParentDir creates the parent directory if it doesn't exist
func ensureParentDir(filePath string) {
	if filePath == "" {
		return
	}

	dir := filepath.Dir(filePath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Log the error but don't fail configuration loading
			log.Printf("Warning: failed to create directory %s: %v", dir, err)
		}
	}
}
