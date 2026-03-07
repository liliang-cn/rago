package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/mcp"
	"github.com/liliang-cn/agent-go/pkg/pool"
	"github.com/spf13/viper"
)

type Config struct {
	Home   string       `mapstructure:"home"`
	Debug  bool         `mapstructure:"debug"`
	Server ServerConfig `mapstructure:"server"`
	LLM    LLMConfig    `mapstructure:"llm"`
	RAG    RAGConfig    `mapstructure:"rag"`
	MCP    mcp.Config   `mapstructure:"mcp"`
	Skills SkillsConfig `mapstructure:"skills"`
	Memory MemoryConfig `mapstructure:"memory"`
}

type LLMConfig struct {
	Enabled   bool                  `mapstructure:"enabled"`
	Strategy  pool.SelectionStrategy `mapstructure:"strategy"`
	Providers []pool.Provider        `mapstructure:"providers"`
}

type RAGConfig struct {
	Enabled   bool                `mapstructure:"enabled"`
	Embedding EmbeddingPoolConfig `mapstructure:"embedding"`
	Storage   CortexdbConfig      `mapstructure:"storage"`
	Chunker   ChunkerConfig       `mapstructure:"chunker"`
	Graph     GraphRAGConfig      `mapstructure:"graph"`
}

type EmbeddingPoolConfig struct {
	Enabled   bool                  `mapstructure:"enabled"`
	Strategy  pool.SelectionStrategy `mapstructure:"strategy"`
	Providers []pool.Provider        `mapstructure:"providers"`
}

// SkillsConfig configures skills paths and behavior
type SkillsConfig struct {
	Enabled               bool     `mapstructure:"enabled"`
	Paths                 []string `mapstructure:"paths"`
	AutoLoad              bool     `mapstructure:"auto_load"`
	AllowCommandInjection bool     `mapstructure:"allow_command_injection"`
	RequireConfirmation   bool     `mapstructure:"require_confirmation"`
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

type CortexdbConfig struct {
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

// MemoryConfig configures the memory system
type MemoryConfig struct {
	StoreType   string                  `mapstructure:"store_type"` // "file", "vector", "hybrid"
	MemoryPath  string                  `mapstructure:"memory_path"`
	MinScore    float64                 `mapstructure:"min_score"`
	MaxMemories int                     `mapstructure:"max_memories"`
	Scoring     MemoryScoringConfig     `mapstructure:"scoring"`
	NoiseFilter MemoryNoiseFilterConfig `mapstructure:"noise_filter"`
	Adaptive    MemoryAdaptiveConfig    `mapstructure:"adaptive"`
	Hybrid      MemoryHybridConfig      `mapstructure:"hybrid"`
}

// MemoryScoringConfig configures memory scoring
type MemoryScoringConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	RecencyWeight     float64 `mapstructure:"recency_weight"`
	HalfLifeDays      float64 `mapstructure:"half_life_days"`
	EnableRecency     bool    `mapstructure:"enable_recency"`
	ImportanceWeight  float64 `mapstructure:"importance_weight"`
	MinImportance     float64 `mapstructure:"min_importance"`
	EnableImportance  bool    `mapstructure:"enable_importance"`
	LengthNormWeight  float64 `mapstructure:"length_norm_weight"`
	AnchorLength      int     `mapstructure:"anchor_length"`
	EnableLengthNorm  bool    `mapstructure:"enable_length_norm"`
	AccessBoostWeight float64 `mapstructure:"access_boost_weight"`
	EnableAccessBoost bool    `mapstructure:"enable_access_boost"`
}

// MemoryNoiseFilterConfig configures noise filtering
type MemoryNoiseFilterConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	MinContentLength int  `mapstructure:"min_content_length"`
	FilterRefusals   bool `mapstructure:"filter_refusals"`
	FilterMeta       bool `mapstructure:"filter_meta"`
	FilterDuplicates bool `mapstructure:"filter_duplicates"`
}

// MemoryAdaptiveConfig configures adaptive retrieval
type MemoryAdaptiveConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	MinQueryLength int  `mapstructure:"min_query_length"`
}

// MemoryHybridConfig configures hybrid search (BM25 + Vector)
type MemoryHybridConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	RRF_K        float64 `mapstructure:"rrf_k"` // RRF fusion parameter
	VectorWeight float64 `mapstructure:"vector_weight"`
	BM25Weight   float64 `mapstructure:"bm25_weight"`
}

// GraphRAGConfig configures GraphRAG (Knowledge Graph + RAG)
type GraphRAGConfig struct {
	Enabled                  bool     `mapstructure:"enabled"`
	EntityTypes              []string `mapstructure:"entity_types"`
	MaxConcurrentExtractions int      `mapstructure:"max_concurrent_extractions"`
	MinEntityLength          int      `mapstructure:"min_entity_length"`
	CommunityDetection       bool     `mapstructure:"community_detection"`
	CommunityAlgorithm       string   `mapstructure:"community_algorithm"` // "louvain", "leiden"
	GraphQueryTopK           int      `mapstructure:"graph_query_topk"`
	VectorWeight             float64  `mapstructure:"vector_weight"`
	GraphWeight              float64  `mapstructure:"graph_weight"`
}

func Load(configPath string) (*Config, error) {
	config := &Config{}

	// 1. Determine the source of truth for Home
	home := os.Getenv("AgentGo_HOME")
	if home == "" {
		home = "~/.agentgo"
	}
	home = expandHomePath(home)

	// 2. Set config file path
	if configPath != "" {
		absPath, _ := filepath.Abs(configPath)
		viper.SetConfigFile(absPath)
		// If user provides a config file, its directory becomes the Home
		home = filepath.Dir(absPath)
	} else {
		if _, err := os.Stat("agentgo.toml"); err == nil {
			abs, _ := filepath.Abs("agentgo.toml")
			viper.SetConfigFile(abs)
			home = filepath.Dir(abs)
		} else {
			p1 := filepath.Join(home, "agentgo.toml")
			p2 := filepath.Join(home, "config", "agentgo.toml")
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
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 4. Finalize Home
	if config.Home == "" {
		config.Home = home
	}
	config.Home = expandHomePath(config.Home)

	// Manual handle provider arrays
	if viper.IsSet("llm.providers") {
		var llm struct {
			Enabled   bool
			Strategy  string
			Providers []interface{}
		}
		viper.UnmarshalKey("llm", &llm)
		config.LLM.Enabled = llm.Enabled
		config.LLM.Strategy = pool.SelectionStrategy(llm.Strategy)
		unmarshalProviders(llm.Providers, &config.LLM.Providers)
		fmt.Printf("[DEBUG] Loaded %d LLM providers\n", len(config.LLM.Providers))
	}
	if viper.IsSet("rag.embedding.providers") {
		var embedding struct {
			Enabled   bool
			Strategy  string
			Providers []interface{}
		}
		viper.UnmarshalKey("rag.embedding", &embedding)
		config.RAG.Embedding.Enabled = embedding.Enabled
		config.RAG.Embedding.Strategy = pool.SelectionStrategy(embedding.Strategy)
		unmarshalProviders(embedding.Providers, &config.RAG.Embedding.Providers)
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
	config.MCP.LoadedServers = mcp.GetBuiltInServers(config.MCP.FilesystemDirs)

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

	viper.SetDefault("llm.enabled", true)
	viper.SetDefault("llm.strategy", "round_robin")
	viper.SetDefault("rag.enabled", true)
	viper.SetDefault("rag.embedding.enabled", false)
	viper.SetDefault("rag.embedding.strategy", "round_robin")

	viper.SetDefault("rag.storage.max_conns", 10)
	viper.SetDefault("rag.storage.batch_size", 100)
	viper.SetDefault("rag.storage.top_k", 5)
	viper.SetDefault("rag.storage.threshold", 0.0)
	viper.SetDefault("rag.storage.index_type", "hnsw")

	viper.SetDefault("rag.chunker.chunk_size", 500)
	viper.SetDefault("rag.chunker.overlap", 50)
	viper.SetDefault("rag.chunker.method", "sentence")

	mcpConfig := mcp.DefaultConfig()
	viper.SetDefault("mcp.enabled", mcpConfig.Enabled)
	viper.SetDefault("mcp.log_level", mcpConfig.LogLevel)
	viper.SetDefault("mcp.default_timeout", mcpConfig.DefaultTimeout)
	viper.SetDefault("mcp.max_concurrent_requests", mcpConfig.MaxConcurrentRequests)
	viper.SetDefault("mcp.health_check_interval", mcpConfig.HealthCheckInterval)
	viper.SetDefault("mcp.servers", []string{})

	viper.SetDefault("skills.enabled", true)
	viper.SetDefault("skills.paths", []string{})
	viper.SetDefault("skills.auto_load", true)
	viper.SetDefault("skills.allow_command_injection", false)
	viper.SetDefault("skills.require_confirmation", true)

	// Memory defaults
	viper.SetDefault("memory.store_type", "file")
	viper.SetDefault("memory.min_score", 0.01)
	viper.SetDefault("memory.max_memories", 5)

	// Memory scoring defaults
	viper.SetDefault("memory.scoring.enabled", true)
	viper.SetDefault("memory.scoring.recency_weight", 0.3)
	viper.SetDefault("memory.scoring.half_life_days", 30.0)
	viper.SetDefault("memory.scoring.enable_recency", true)
	viper.SetDefault("memory.scoring.importance_weight", 0.3)
	viper.SetDefault("memory.scoring.min_importance", 0.7)
	viper.SetDefault("memory.scoring.enable_importance", true)
	viper.SetDefault("memory.scoring.length_norm_weight", 0.1)
	viper.SetDefault("memory.scoring.anchor_length", 100)
	viper.SetDefault("memory.scoring.enable_length_norm", true)
	viper.SetDefault("memory.scoring.access_boost_weight", 0.1)
	viper.SetDefault("memory.scoring.enable_access_boost", true)

	// Memory noise filter defaults
	viper.SetDefault("memory.noise_filter.enabled", true)
	viper.SetDefault("memory.noise_filter.min_content_length", 20)
	viper.SetDefault("memory.noise_filter.filter_refusals", true)
	viper.SetDefault("memory.noise_filter.filter_meta", true)
	viper.SetDefault("memory.noise_filter.filter_duplicates", true)

	// Memory adaptive retrieval defaults
	viper.SetDefault("memory.adaptive.enabled", true)
	viper.SetDefault("memory.adaptive.min_query_length", 5)

	// Memory hybrid search defaults
	viper.SetDefault("memory.hybrid.enabled", false)
	viper.SetDefault("memory.hybrid.rrf_k", 60.0)
	viper.SetDefault("memory.hybrid.vector_weight", 0.7)
	viper.SetDefault("memory.hybrid.bm25_weight", 0.3)

	// GraphRAG defaults
	viper.SetDefault("rag.graph.enabled", false)
	viper.SetDefault("rag.graph.entity_types", []string{"person", "organization", "location", "concept", "event", "product"})
	viper.SetDefault("rag.graph.max_concurrent_extractions", 3)
	viper.SetDefault("rag.graph.min_entity_length", 2)
	viper.SetDefault("rag.graph.community_detection", true)
	viper.SetDefault("rag.graph.community_algorithm", "louvain")
	viper.SetDefault("rag.graph.graph_query_topk", 10)
	viper.SetDefault("rag.graph.vector_weight", 0.7)
	viper.SetDefault("rag.graph.graph_weight", 0.3)
}

func bindEnvVars() {
	viper.SetEnvPrefix("AgentGo")
	viper.AutomaticEnv()

	// Direct binding for common DEBUG env var
	viper.BindEnv("debug", "DEBUG")

	viper.BindEnv("home", "AgentGo_HOME")
	viper.BindEnv("server.port", "AgentGo_SERVER_PORT")
	viper.BindEnv("server.host", "AgentGo_SERVER_HOST")
	viper.BindEnv("rag.storage.db_path", "AgentGo_RAG_STORAGE_DB_PATH")
	viper.BindEnv("rag.chunker.chunk_size", "AgentGo_RAG_CHUNKER_CHUNK_SIZE")
	viper.BindEnv("rag.chunker.overlap", "AgentGo_RAG_CHUNKER_OVERLAP")
	viper.BindEnv("rag.chunker.method", "AgentGo_RAG_CHUNKER_METHOD")
	viper.BindEnv("mcp.enabled", "AgentGo_MCP_ENABLED")
	viper.BindEnv("mcp.log_level", "AgentGo_MCP_LOG_LEVEL")
	viper.BindEnv("mcp.default_timeout", "AgentGo_MCP_DEFAULT_TIMEOUT")
	viper.BindEnv("mcp.max_concurrent_requests", "AgentGo_MCP_MAX_CONCURRENT_REQUESTS")
	viper.BindEnv("mcp.health_check_interval", "AgentGo_MCP_HEALTH_CHECK_INTERVAL")
	viper.BindEnv("skills.enabled", "AgentGo_SKILLS_ENABLED")
	viper.BindEnv("skills.auto_load", "AgentGo_SKILLS_AUTO_LOAD")
	viper.BindEnv("skills.allow_command_injection", "AgentGo_SKILLS_ALLOW_COMMAND_INJECTION")
	viper.BindEnv("skills.require_confirmation", "AgentGo_SKILLS_REQUIRE_CONFIRMATION")
	viper.BindEnv("memory.min_score", "AgentGo_MEMORY_MIN_SCORE")
	viper.BindEnv("memory.max_memories", "AgentGo_MEMORY_MAX_MEMORIES")
	viper.BindEnv("memory.scoring.enabled", "AgentGo_MEMORY_SCORING_ENABLED")
	viper.BindEnv("memory.noise_filter.enabled", "AgentGo_MEMORY_NOISE_FILTER_ENABLED")
	viper.BindEnv("memory.adaptive.enabled", "AgentGo_MEMORY_ADAPTIVE_ENABLED")
	viper.BindEnv("memory.hybrid.enabled", "AgentGo_MEMORY_HYBRID_ENABLED")
}

// DataDir returns the path to the data directory
func (c *Config) DataDir() string {
	return filepath.Join(c.Home, "data")
}

// SkillsDir returns the path to the skills directory
func (c *Config) SkillsDir() string {
	return filepath.Join(c.Home, "skills")
}

// SkillsPaths returns all skills search paths
// Returns configured paths + default paths (project local, user global)
func (c *Config) SkillsPaths() []string {
	paths := make([]string, 0)

	// 1. Add configured paths first (highest priority)
	for _, p := range c.Skills.Paths {
		expanded := expandHomePath(p)
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(c.Home, expanded)
		}
		paths = append(paths, expanded)
	}

	// 2. Add default paths
	defaults := []string{
		".skills",                           // Project root
		filepath.Join(".agentgo", "skills"), // Project .agentgo
		c.SkillsDir(),                       // Home/skills
	}
	for _, p := range defaults {
		expanded := expandHomePath(p)
		// Avoid duplicates
		found := false
		for _, existing := range paths {
			if existing == expanded {
				found = true
				break
			}
		}
		if !found {
			paths = append(paths, expanded)
		}
	}

	return paths
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

	if c.RAG.Storage.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if c.RAG.Storage.TopK <= 0 {
		return fmt.Errorf("topK must be positive: %d", c.RAG.Storage.TopK)
	}

	if c.RAG.Storage.Threshold < 0 {
		return fmt.Errorf("threshold must be non-negative: %f", c.RAG.Storage.Threshold)
	}

	validIndexTypes := map[string]bool{"hnsw": true, "ivf": true, "flat": true, "": true}
	if !validIndexTypes[strings.ToLower(c.RAG.Storage.IndexType)] {
		return fmt.Errorf("invalid index_type: %s (supported: hnsw, ivf, flat)", c.RAG.Storage.IndexType)
	}

	if c.RAG.Chunker.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive: %d", c.RAG.Chunker.ChunkSize)
	}

	if c.RAG.Chunker.Overlap < 0 || c.RAG.Chunker.Overlap >= c.RAG.Chunker.ChunkSize {
		return fmt.Errorf("overlap must be between 0 and chunk size: %d", c.RAG.Chunker.Overlap)
	}

	validMethods := map[string]bool{"sentence": true, "paragraph": true, "token": true}
	if !validMethods[c.RAG.Chunker.Method] {
		return fmt.Errorf("invalid chunker method: %s", c.RAG.Chunker.Method)
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
	if c.RAG.Storage.DBPath == "" {
		c.RAG.Storage.DBPath = filepath.Join(c.DataDir(), "agentgo.db")
	}

	if c.Memory.MemoryPath == "" {
		if c.Memory.StoreType == "vector" {
			c.Memory.MemoryPath = c.RAG.Storage.DBPath
		} else {
			c.Memory.MemoryPath = filepath.Join(c.DataDir(), "memories")
		}
	}
}

func (c *Config) expandPaths() {
	c.Home = expandHomePath(c.Home)
	c.RAG.Storage.DBPath = expandHomePath(c.RAG.Storage.DBPath)
	c.Memory.MemoryPath = expandHomePath(c.Memory.MemoryPath)
	ensureParentDir(c.RAG.Storage.DBPath)
	if c.Memory.StoreType != "vector" {
		os.MkdirAll(c.Memory.MemoryPath, 0755)
	} else {
		ensureParentDir(c.Memory.MemoryPath)
	}
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

// LoadMCPConfig loads MCP configuration from specific paths (supports multiple)
func LoadMCPConfig(paths ...string) (*mcp.Config, error) {
	cfg := mcp.DefaultConfig()
	cfg.Enabled = true
	cfg.Servers = paths
	if err := cfg.LoadServersFromJSON(); err != nil {
		return nil, fmt.Errorf("failed to load MCP servers: %w", err)
	}
	return &cfg, nil
}
