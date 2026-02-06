package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/pool"
	"github.com/spf13/viper"
)

type Config struct {
	Home         string          `mapstructure:"home"`
	Server       ServerConfig    `mapstructure:"server"`
	LLMPool      pool.PoolConfig `mapstructure:"llm_pool"`
	EmbeddingPool pool.PoolConfig `mapstructure:"embedding_pool"`
	Sqvect       SqvectConfig    `mapstructure:"sqvect"`
	Chunker      ChunkerConfig   `mapstructure:"chunker"`
	Ingest       IngestConfig    `mapstructure:"ingest"`
	MCP          mcp.Config      `mapstructure:"mcp"`
	VectorStore  *VectorStoreConfig `mapstructure:"vector_store"`
}

// VectorStoreConfig configures the vector storage backend
type VectorStoreConfig struct {
	Type       string                 `mapstructure:"type"`
	Parameters map[string]interface{} `mapstructure:"parameters"`
}

type IngestConfig struct {
	MetadataExtraction MetadataExtractionConfig `mapstructure:"metadata_extraction"`
}

type MetadataExtractionConfig struct {
	Enable bool `mapstructure:"enable"`
}

type ServerConfig struct {
	Port        int      `mapstructure:"port"`
	Host        string   `mapstructure:"host"`
	EnableUI    bool     `mapstructure:"enable_ui"`
	CORSOrigins []string `mapstructure:"cors_origins"`
}

type SqvectConfig struct {
	DBPath    string  `mapstructure:"db_path"`
	MaxConns  int     `mapstructure:"max_conns"`
	BatchSize int     `mapstructure:"batch_size"`
	TopK      int     `mapstructure:"top_k"`
	Threshold float64 `mapstructure:"threshold"`
	IndexType string  `mapstructure:"index_type"`
}

type ChunkerConfig struct {
	ChunkSize int    `mapstructure:"chunk_size"`
	Overlap   int    `mapstructure:"overlap"`
	Method    string `mapstructure:"method"`
}

func Load(configPath string) (*Config, error) {
	config := &Config{}

	// 1. Determine the source of truth for Home
	home := os.Getenv("RAGO_HOME")
	if home == "" {
		home = "~/.rago"
	}
	home = expandHomePath(home)

	// 2. Set config file path
	if configPath != "" {
		absPath, _ := filepath.Abs(configPath)
		viper.SetConfigFile(absPath)
		// If user provides a config file, its directory becomes the Home
		home = filepath.Dir(absPath)
	} else {
		// Check order:
		// 1. ./rago.toml
		// 2. ~/.rago/rago.toml
		// 3. ~/.rago/config/rago.toml
		if _, err := os.Stat("rago.toml"); err == nil {
			abs, _ := filepath.Abs("rago.toml")
			viper.SetConfigFile(abs)
			home = filepath.Dir(abs)
		} else {
			p1 := filepath.Join(home, "rago.toml")
			p2 := filepath.Join(home, "config", "rago.toml")
			if _, err := os.Stat(p1); err == nil {
				viper.SetConfigFile(p1)
			} else if _, err := os.Stat(p2); err == nil {
				viper.SetConfigFile(p2)
			} else {
				// Fallback to default path
				viper.SetConfigFile(p1)
			}
		}
	}

	setDefaults()
	bindEnvVars()

	// 3. Read config
	if err := viper.ReadInConfig(); err != nil {
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		// If default config doesn't exist, we continue with defaults
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 4. Finalize Home
	if config.Home == "" {
		config.Home = home
	}
	config.Home = expandHomePath(config.Home)

	// 手动处理provider数组
	if viper.IsSet("llm_pool.providers") {
		var llmPool struct {
			Enabled   bool
			Strategy  string
			Providers []interface{}
		}
		viper.UnmarshalKey("llm_pool", &llmPool)
		config.LLMPool.Enabled = llmPool.Enabled
		config.LLMPool.Strategy = pool.SelectionStrategy(llmPool.Strategy)
		unmarshalProviders(llmPool.Providers, &config.LLMPool.Providers)
	}
	if viper.IsSet("embedding_pool.providers") {
		var embeddingPool struct {
			Enabled   bool
			Strategy  string
			Providers []interface{}
		}
		viper.UnmarshalKey("embedding_pool", &embeddingPool)
		config.EmbeddingPool.Enabled = embeddingPool.Enabled
		config.EmbeddingPool.Strategy = pool.SelectionStrategy(embeddingPool.Strategy)
		unmarshalProviders(embeddingPool.Providers, &config.EmbeddingPool.Providers)
	}

	// Initialize MCP
	if config.MCP.DefaultTimeout == 0 {
		defaultMCP := mcp.DefaultConfig()
		config.MCP.LogLevel = defaultMCP.LogLevel
		config.MCP.DefaultTimeout = defaultMCP.DefaultTimeout
		config.MCP.MaxConcurrentRequests = defaultMCP.MaxConcurrentRequests
		config.MCP.HealthCheckInterval = defaultMCP.HealthCheckInterval
		config.MCP.Enabled = defaultMCP.Enabled
	}

	// Unify all paths under Home
	config.resolveDatabasePath()
	config.resolveMCPServerPaths()
	config.expandPaths()

	// Load MCP servers
	if err := config.MCP.LoadServersFromJSON(); err != nil {
		if config.MCP.Enabled && len(config.MCP.Servers) > 0 {
			log.Printf("Warning: failed to load MCP servers: %v", err)
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func (c *Config) resolveMCPServerPaths() {
	// MCP server config is Home/mcpServers.json
	unifiedPath := filepath.Join(c.Home, "mcpServers.json")

	// Remove legacy ./mcpServers.json path (current directory) if present
	filtered := make([]string, 0, len(c.MCP.Servers))
	for _, s := range c.MCP.Servers {
		if s != "./mcpServers.json" {
			filtered = append(filtered, s)
		}
	}
	c.MCP.Servers = filtered

	// Add unified path to front if not already present
	found := false
	for _, s := range c.MCP.Servers {
		if s == unifiedPath {
			found = true
			break
		}
	}
	if !found {
		c.MCP.Servers = append([]string{unifiedPath}, c.MCP.Servers...)
	}
}

func setDefaults() {
	viper.SetDefault("server.port", 7127)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.enable_ui", false)
	viper.SetDefault("server.cors_origins", []string{"*"})

	viper.SetDefault("llm_pool.enabled", true)
	viper.SetDefault("llm_pool.strategy", "round_robin")
	viper.SetDefault("embedding_pool.enabled", true)
	viper.SetDefault("embedding_pool.strategy", "round_robin")

	viper.SetDefault("sqvect.max_conns", 10)
	viper.SetDefault("sqvect.batch_size", 100)
	viper.SetDefault("sqvect.top_k", 5)
	viper.SetDefault("sqvect.threshold", 0.0)
	viper.SetDefault("sqvect.index_type", "hnsw")

	viper.SetDefault("chunker.chunk_size", 500)
	viper.SetDefault("chunker.overlap", 50)
	viper.SetDefault("chunker.method", "sentence")

	viper.SetDefault("ingest.metadata_extraction.enable", false)

	mcpConfig := mcp.DefaultConfig()
	viper.SetDefault("mcp.enabled", mcpConfig.Enabled)
	viper.SetDefault("mcp.log_level", mcpConfig.LogLevel)
	viper.SetDefault("mcp.default_timeout", mcpConfig.DefaultTimeout)
	viper.SetDefault("mcp.max_concurrent_requests", mcpConfig.MaxConcurrentRequests)
	viper.SetDefault("mcp.health_check_interval", mcpConfig.HealthCheckInterval)
	viper.SetDefault("mcp.servers", []string{})
}

func bindEnvVars() {
	viper.SetEnvPrefix("RAGO")
	viper.AutomaticEnv()

	if err := viper.BindEnv("home", "RAGO_HOME"); err != nil {
		log.Printf("Warning: failed to bind home env var: %v", err)
	}
	if err := viper.BindEnv("server.port", "RAGO_SERVER_PORT"); err != nil {
		log.Printf("Warning: failed to bind server.port env var: %v", err)
	}
	if err := viper.BindEnv("server.host", "RAGO_SERVER_HOST"); err != nil {
		log.Printf("Warning: failed to bind server.host env var: %v", err)
	}
	if err := viper.BindEnv("sqvect.db_path", "RAGO_SQVECT_DB_PATH"); err != nil {
		log.Printf("Warning: failed to bind sqvect.db_path env var: %v", err)
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

// DataDir returns the path to the data directory
func (c *Config) DataDir() string {
	return filepath.Join(c.Home, "data")
}

// SkillsDir returns the path to the skills directory
func (c *Config) SkillsDir() string {
	return filepath.Join(c.Home, "skills")
}

// IntentsDir returns the path to the intents directory
func (c *Config) IntentsDir() string {
	return filepath.Join(c.Home, "intents")
}

// WorkspaceDir returns the path to the workspace directory
func (c *Config) WorkspaceDir() string {
	return filepath.Join(c.Home, "workspace")
}

// MCPServersPath returns the path to the MCP servers configuration file
func (c *Config) MCPServersPath() string {
	return filepath.Join(c.Home, "mcpServers.json")
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if c.Sqvect.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if c.Sqvect.TopK <= 0 {
		return fmt.Errorf("topK must be positive: %d", c.Sqvect.TopK)
	}

	if c.Sqvect.Threshold < 0 {
		return fmt.Errorf("threshold must be non-negative: %f", c.Sqvect.Threshold)
	}

	validIndexTypes := map[string]bool{"hnsw": true, "ivf": true, "flat": true, "": true}
	if !validIndexTypes[strings.ToLower(c.Sqvect.IndexType)] {
		return fmt.Errorf("invalid index_type: %s (supported: hnsw, ivf, flat)", c.Sqvect.IndexType)
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

	// Validate MCP configuration
	if err := c.validateMCPConfig(); err != nil {
		return fmt.Errorf("invalid MCP configuration: %w", err)
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

func (c *Config) validateMCPConfig() error {
	if !c.MCP.Enabled {
		return nil
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

	for i, serverFile := range c.MCP.Servers {
		if serverFile == "" {
			return fmt.Errorf("empty server config file path at index %d", i)
		}
	}

	return nil
}

func (c *Config) resolveDatabasePath() {
	if c.Sqvect.DBPath != "" {
		return
	}

	c.Sqvect.DBPath = filepath.Join(c.DataDir(), "rag.db")
}

func (c *Config) expandPaths() {
	c.Home = expandHomePath(c.Home)
	c.Sqvect.DBPath = expandHomePath(c.Sqvect.DBPath)
	ensureParentDir(c.Sqvect.DBPath)
}

func expandHomePath(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}

	return path
}

func ensureParentDir(filePath string) {
	if filePath == "" {
		return
	}

	dir := filepath.Dir(filePath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: failed to create directory %s: %v", dir, err)
		}
	}
}

// unmarshalProviders 将viper读取的provider数组解析为Provider结构体
func unmarshalProviders(raw interface{}, target *[]pool.Provider) error {
	if raw == nil {
		return nil
	}

	// 转换为JSON再解析（绕过mapstructure的限制）
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal providers: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal providers: %w", err)
	}

	return nil
}
