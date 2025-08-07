package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Ollama  OllamaConfig  `mapstructure:"ollama"`
	Sqvect  SqvectConfig  `mapstructure:"sqvect"`
	Chunker ChunkerConfig `mapstructure:"chunker"`
	UI      UIConfig      `mapstructure:"ui"`
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
	VectorDim int     `mapstructure:"vector_dim"`
	MaxConns  int     `mapstructure:"max_conns"`
	BatchSize int     `mapstructure:"batch_size"`
	TopK      int     `mapstructure:"top_k"`
	Threshold float64 `mapstructure:"threshold"`
}

type ChunkerConfig struct {
	ChunkSize int    `mapstructure:"chunk_size"`
	Overlap   int    `mapstructure:"overlap"`
	Method    string `mapstructure:"method"`
}

type UIConfig struct {
	Title       string `mapstructure:"title"`
	Theme       string `mapstructure:"theme"`
	MaxFileSize string `mapstructure:"max_file_size"`
}

func Load(configPath string) (*Config, error) {
	config := &Config{}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME/.rago")
	}

	setDefaults()
	bindEnvVars()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.enable_ui", false)
	viper.SetDefault("server.cors_origins", []string{"*"})

	viper.SetDefault("ollama.embedding_model", "nomic-embed-text")
	viper.SetDefault("ollama.llm_model", "qwen3")
	viper.SetDefault("ollama.base_url", "http://localhost:11434")
	viper.SetDefault("ollama.timeout", "30s")

	viper.SetDefault("sqvect.db_path", "./data/rag.db")
	viper.SetDefault("sqvect.vector_dim", 1536)
	viper.SetDefault("sqvect.max_conns", 10)
	viper.SetDefault("sqvect.batch_size", 100)
	viper.SetDefault("sqvect.top_k", 5)
	viper.SetDefault("sqvect.threshold", 0.0)

	viper.SetDefault("chunker.chunk_size", 300)
	viper.SetDefault("chunker.overlap", 50)
	viper.SetDefault("chunker.method", "sentence")

	viper.SetDefault("ui.title", "RAGO - 本地 RAG 系统")
	viper.SetDefault("ui.theme", "light")
	viper.SetDefault("ui.max_file_size", "10MB")
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

	if err := viper.BindEnv("chunker.chunk_size", "RAGO_CHUNKER_CHUNK_SIZE"); err != nil {
		log.Printf("Warning: failed to bind chunker.chunk_size env var: %v", err)
	}
	if err := viper.BindEnv("chunker.overlap", "RAGO_CHUNKER_OVERLAP"); err != nil {
		log.Printf("Warning: failed to bind chunker.overlap env var: %v", err)
	}
	if err := viper.BindEnv("chunker.method", "RAGO_CHUNKER_METHOD"); err != nil {
		log.Printf("Warning: failed to bind chunker.method env var: %v", err)
	}
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if c.Ollama.BaseURL == "" {
		return fmt.Errorf("ollama base URL cannot be empty")
	}

	if c.Ollama.EmbeddingModel == "" {
		return fmt.Errorf("embedding model cannot be empty")
	}

	if c.Ollama.LLMModel == "" {
		return fmt.Errorf("LLM model cannot be empty")
	}

	if c.Sqvect.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if c.Sqvect.VectorDim <= 0 {
		return fmt.Errorf("vector dimension must be positive: %d", c.Sqvect.VectorDim)
	}

	if c.Sqvect.MaxConns <= 0 {
		return fmt.Errorf("max connections must be positive: %d", c.Sqvect.MaxConns)
	}

	if c.Sqvect.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive: %d", c.Sqvect.BatchSize)
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
