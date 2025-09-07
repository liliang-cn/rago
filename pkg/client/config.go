// Package client - config.go
// This file provides configuration management and backward compatibility 
// for the unified RAGO client supporting the four-pillar architecture.

package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	// TODO: Legacy config conversion - to be refactored
	// "github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Config represents the unified RAGO configuration supporting all four pillars.
// This extends the core.Config with client-specific settings and convenience methods.
type Config struct {
	core.Config
	
	// Client-specific settings
	ClientName    string `toml:"client_name"`
	ClientVersion string `toml:"client_version"`
	
	// ConfigPath stores the path to the loaded configuration file
	ConfigPath string `toml:"-"`
	
	// Backward compatibility settings
	LegacyMode bool `toml:"legacy_mode"`
}

// rawConfig is used for TOML unmarshaling to handle both legacy and new formats
type rawConfig struct {
	// Core configuration
	DataDir  string `toml:"data_dir"`
	LogLevel string `toml:"log_level"`
	
	// Client settings
	ClientName    string `toml:"client_name"`
	ClientVersion string `toml:"client_version"`
	LegacyMode    bool   `toml:"legacy_mode"`
	
	// Mode configuration
	Mode core.ModeConfig `toml:"mode"`
	
	// V3 Four-Pillar configurations (preferred format)
	LLM    *rawLLMConfig     `toml:"llm"`     // LLM pillar config
	RAG    core.RAGConfig    `toml:"rag"`     // RAG pillar config
	MCP    core.MCPConfig    `toml:"mcp"`     // MCP pillar config
	Agents core.AgentsConfig `toml:"agents"`  // Agent pillar config
	
	// Legacy providers section (backward compatibility)
	Providers *rawProvidersConfig `toml:"providers"`
	
	// Legacy top-level configs (deprecated, use [llm] section)
	LoadBalancing *core.LoadBalancingConfig `toml:"load_balancing"`
	HealthCheck   *core.HealthCheckConfig   `toml:"health_check"`
}

// rawLLMConfig represents the V3 LLM pillar configuration
type rawLLMConfig struct {
	DefaultProvider string                   `toml:"default_provider"`
	Providers       rawProvidersConfig       `toml:"providers"`
	LoadBalancing   core.LoadBalancingConfig `toml:"load_balancing"`
	HealthCheck     core.HealthCheckConfig   `toml:"health_check"`
}

// rawProvidersConfig handles both legacy map format and new array format
type rawProvidersConfig struct {
	// Common fields
	DefaultLLM     string `toml:"default_llm"`
	DefaultEmbedder string `toml:"default_embedder"`
	
	// New array-based configuration
	List []core.ProviderConfig `toml:"list"`
	
	// Legacy direct provider configurations
	Ollama   *core.ProviderConfig `toml:"ollama"`
	OpenAI   *core.ProviderConfig `toml:"openai"`
	LMStudio *core.ProviderConfig `toml:"lmstudio"`
	
	// Legacy nested configurations (for older formats)
	ProviderConfigs map[string]core.ProviderConfig `toml:"-"` // Populated manually
}

// ModeConfig extensions for client-specific mode checking
type ClientModeConfig struct {
	core.ModeConfig
}

// DisableLLM returns true if LLM pillar should be disabled
func (m *ClientModeConfig) DisableLLM() bool {
	return m.LLMOnly == false && (m.RAGOnly || len(m.String()) == 0)
}

// DisableRAG returns true if RAG pillar should be disabled  
func (m *ClientModeConfig) DisableRAG() bool {
	return m.RAGOnly == false && m.LLMOnly
}

// String provides a string representation of the active mode
func (m *ClientModeConfig) String() string {
	if m.LLMOnly {
		return "llm-only"
	}
	if m.RAGOnly {
		return "rag-only"  
	}
	if m.DisableMCP && m.DisableAgent {
		return "llm-rag"
	}
	if m.DisableMCP {
		return "no-mcp"
	}
	if m.DisableAgent {
		return "no-agents"
	}
	return "full"
}

// Legacy conversion functions removed - V3 uses pure TOML configuration
// All configuration now loads directly from TOML using the new V3 format

// getDefaultConfig provides a complete default configuration
func getDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".rago")
	
	return &Config{
		Config: core.Config{
			DataDir:  dataDir,
			LogLevel: "info",
			LLM:      getDefaultLLMConfig(),
			RAG:      getDefaultRAGConfig(dataDir),
			MCP:      getDefaultMCPConfig(),
			Agents:   getDefaultAgentsConfig(),
			Mode: core.ModeConfig{
				RAGOnly:      false,
				LLMOnly:      false,
				DisableMCP:   false,
				DisableAgent: false,
			},
		},
		ClientName:    "rago-client",
		ClientVersion: "3.0.0",
		LegacyMode:    false,
	}
}

// getDefaultLLMConfig provides default LLM configuration
func getDefaultLLMConfig() core.LLMConfig {
	return core.LLMConfig{
		DefaultProvider: "ollama-default",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:                "weighted_round_robin",
			MaxRetries:              3,
			RetryDelay:              1 * time.Second,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   30 * time.Second,
			HealthCheck:             true,
			CheckInterval:           30 * time.Second,
		},
		Providers: core.ProvidersConfig{
			List: []core.ProviderConfig{
				{
					Name:    "ollama-default",
					Type:    "ollama", 
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
					Weight:  10,
					Enabled: true,
					Timeout: 30 * time.Second,
					Parameters: map[string]interface{}{
						"temperature": 0.7,
						"max_tokens":  4000,
					},
				},
			},
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
	}
}

// getDefaultRAGConfig provides default RAG configuration  
func getDefaultRAGConfig(dataDir string) core.RAGConfig {
	return core.RAGConfig{
		StorageBackend: "hybrid",
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:     "fixed",
			ChunkSize:    1000,
			ChunkOverlap: 200,
			MinChunkSize: 50,
		},
		VectorStore: core.VectorStoreConfig{
			Backend:    "sqvect",
			Dimensions: 0,
			Metric:     "cosine", 
			IndexType:  "flat",
		},
		KeywordStore: core.KeywordStoreConfig{
			Backend:   "bleve",
			Analyzer:  "standard",
			Languages: []string{"en"},
			Stemming:  true,
		},
		Search: core.SearchConfig{
			DefaultLimit:     10,
			MaxLimit:         100,
			DefaultThreshold: 0.7,
			HybridWeights: struct {
				Vector  float32 `toml:"vector"`
				Keyword float32 `toml:"keyword"`
			}{
				Vector:  0.7,
				Keyword: 0.3,
			},
		},
		Embedding: core.EmbeddingConfig{
			Provider:   "ollama-default",
			Model:      "nomic-embed-text",
			Dimensions: 768,
			BatchSize:  32,
		},
	}
}

// getDefaultMCPConfig provides default MCP configuration
func getDefaultMCPConfig() core.MCPConfig {
	return core.MCPConfig{
		ServersPath: "mcpServers.json",
		Servers:     []core.ServerConfig{},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 60 * time.Second,
			Timeout:  30 * time.Second,
			Retries:  3,
		},
		ToolExecution: core.ToolExecutionConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 30 * time.Second,
			EnableCache:    true,
			CacheTTL:       5 * time.Minute,
		},
		HealthCheckInterval: 60 * time.Second,
		CacheSize:          100,
		CacheTTL:           5 * time.Minute,
	}
}

// getDefaultAgentsConfig provides default agents configuration
func getDefaultAgentsConfig() core.AgentsConfig {
	return core.AgentsConfig{
		WorkflowEngine: core.WorkflowEngineConfig{
			MaxSteps:       50,
			StepTimeout:    30 * time.Second,
			StateBackend:   "memory",
		},
		Scheduling: core.SchedulingConfig{
			Backend:       "memory",
			MaxConcurrent: 3,
			QueueSize:     100,
		},
		StateStorage: core.StateStorageConfig{
			Backend:    "memory",
			Persistent: false,
			TTL:        1 * time.Hour,
		},
		ReasoningChains: core.ReasoningChainsConfig{
			MaxSteps:      25,
			MaxMemorySize: 1000,
			StepTimeout:   10 * time.Second,
		},
	}
}

// Validate validates the configuration and returns any errors
// With minimal configuration philosophy: only validate what's absolutely essential
func (c *Config) Validate() error {
	// Data directory will use default if not specified - no validation needed
	
	// Check what pillars are intended to be enabled based on mode
	llmIntended := !c.Mode.RAGOnly
	
	// With minimal config philosophy, we don't enforce pillar requirements
	// Users can enable/disable pillars as needed, defaults will handle the rest
	
	// Only validate LLM providers if LLM is intended and providers are explicitly configured
	if llmIntended && len(c.LLM.Providers.List) > 0 {
		// Validate provider configurations only if explicitly provided
		for _, provider := range c.LLM.Providers.List {
			if provider.Name == "" {
				return fmt.Errorf("provider name cannot be empty")
			}
			if provider.Type == "" {
				return fmt.Errorf("provider '%s' type cannot be empty", provider.Name)
			}
			if provider.Type != "ollama" && provider.Type != "openai" && provider.Type != "lmstudio" {
				return fmt.Errorf("provider '%s' has unsupported type '%s'", provider.Name, provider.Type)
			}
			if provider.Weight < 0 {
				return fmt.Errorf("provider '%s' weight cannot be negative", provider.Name)
			}
		}
		
		// If providers are configured but no default is set, use the first one
		if c.LLM.DefaultProvider == "" && len(c.LLM.Providers.List) > 0 {
			c.LLM.DefaultProvider = c.LLM.Providers.List[0].Name
		}
	}
	
	// MCP servers validation - only validate if explicitly configured
	// MCP servers are configured via mcpServers.json file, not in TOML
	// So no validation needed here
	
	return nil
}

// Save saves the configuration to a TOML file
func (c *Config) Save(path string) error {
	// Convert config to raw format for TOML marshaling
	raw := convertConfigToRaw(c)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	// Write TOML header
	file.WriteString("# RAGO V3 Configuration\n")
	file.WriteString("# Generated automatically - edit with care\n\n")
	
	// Encode to TOML
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(raw); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}
	
	return nil
}

// convertConfigToRaw converts Config to rawConfig for TOML marshaling
func convertConfigToRaw(c *Config) rawConfig {
	raw := rawConfig{
		DataDir:       c.DataDir,
		LogLevel:      c.LogLevel,
		ClientName:    c.ClientName,
		ClientVersion: c.ClientVersion,
		LegacyMode:    c.LegacyMode,
		Mode:          c.Mode,
		RAG:           c.RAG,
		MCP:           c.MCP,
		Agents:        c.Agents,
		LoadBalancing: &c.LLM.LoadBalancing,  // Fix: take address
		HealthCheck:   &c.LLM.HealthCheck,    // Fix: take address
	}
	
	// Convert providers to raw format - prefer array format
	raw.Providers = &rawProvidersConfig{  // Fix: pointer type
		DefaultLLM:  c.LLM.DefaultProvider,
		List:        c.LLM.Providers.List,
	}
	
	// Convert legacy providers to individual sections if no array providers exist
	if len(c.LLM.Providers.List) == 0 && len(c.LLM.Providers.Legacy) > 0 {
		for _, provider := range c.LLM.Providers.Legacy {
			switch strings.ToLower(provider.Type) {
			case "ollama":
				raw.Providers.Ollama = &provider
			case "openai":
				raw.Providers.OpenAI = &provider
			case "lmstudio":
				raw.Providers.LMStudio = &provider
			}
		}
	}
	
	return raw
}

// ResolveConfigPath resolves the configuration file path based on the search order:
// 1. If a specific path is provided, use it
// 2. Otherwise search in order: ./rago.toml → ./.rago/rago.toml → ~/.rago/rago.toml
func ResolveConfigPath(providedPath string) (string, error) {
	// If a specific path was provided, use it directly
	if providedPath != "" {
		if _, err := os.Stat(providedPath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("specified config file not found: %s", providedPath)
			}
			return "", fmt.Errorf("error accessing config file %s: %w", providedPath, err)
		}
		return providedPath, nil
	}
	
	// Search for config in priority order
	searchPaths := []string{
		"./rago.toml",           // 1. Current directory (highest priority)
		"./.rago/rago.toml",     // 2. .rago subdirectory in current directory
	}
	
	// 3. User home directory (lowest priority)
	if homeDir, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(homeDir, ".rago", "rago.toml"))
	}
	
	// Return the first existing config file
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// No config file found in any location
	return "", fmt.Errorf("no configuration file found in any of the following locations: %v", searchPaths)
}

// ResolveDataDir resolves the data directory based on the config location
// Uses the same priority as config: ./data → ./.rago/data → ~/.rago/data
func ResolveDataDir(configPath string) string {
	// If config path is provided, derive data dir from config location
	if configPath != "" {
		dir := filepath.Dir(configPath)
		base := filepath.Base(configPath)
		
		// If config is in a .rago directory, use .rago/data
		if filepath.Base(dir) == ".rago" {
			return filepath.Join(dir, "data")
		}
		
		// If config is rago.toml in current directory, use ./data
		if base == "rago.toml" {
			return filepath.Join(dir, "data")
		}
	}
	
	// Check for existing data directories in priority order
	dataDirs := []string{
		"./data",           // 1. Current directory data
		"./.rago/data",     // 2. Project .rago directory
	}
	
	// 3. User home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		dataDirs = append(dataDirs, filepath.Join(homeDir, ".rago", "data"))
	}
	
	// Return the first existing data directory
	for _, dir := range dataDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	
	// Default to ./data if no existing directory found
	return "./data"
}

// Load loads configuration from a TOML file
// If path is empty, it uses ResolveConfigPath to find the config file
func Load(path string) (*Config, error) {
	// Resolve the config path if not explicitly provided
	configPath, err := ResolveConfigPath(path)
	if err != nil {
		return nil, err
	}
	
	// Read the TOML file
	var raw rawConfig
	if _, err := toml.DecodeFile(configPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config at %s: %w", configPath, err)
	}
	
	// Convert raw config to unified config
	config, err := convertRawConfig(&raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	
	// Store the loaded config path for reference
	config.ConfigPath = configPath
	
	// If DataDir is not explicitly set in config, resolve it based on config location
	if config.DataDir == "" || raw.DataDir == "" {
		config.DataDir = ResolveDataDir(configPath)
	}
	
	// Set default values if not specified
	fillDefaults(config)
	
	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return config, nil
}

// LoadWithDefaults loads configuration from file or returns defaults if file doesn't exist
func LoadWithDefaults(path string) (*Config, error) {
	// Try to resolve and load config
	configPath, err := ResolveConfigPath(path)
	if err != nil {
		// No config file found, return default config
		config := getDefaultConfig()
		return config, nil
	}
	
	// Config file found, load it
	return Load(configPath)
}

// convertRawConfig converts rawConfig to unified Config
func convertRawConfig(raw *rawConfig) (*Config, error) {
	config := &Config{
		Config: core.Config{
			DataDir:  raw.DataDir,
			LogLevel: raw.LogLevel,
			Mode:     raw.Mode,
			RAG:      raw.RAG,
			MCP:      raw.MCP,
			Agents:   raw.Agents,
		},
		ClientName:    raw.ClientName,
		ClientVersion: raw.ClientVersion,
		LegacyMode:    raw.LegacyMode,
	}
	
	// Handle V3 format with [llm] section (preferred)
	if raw.LLM != nil {
		// V3 format - use LLM section directly
		providersConfig, err := convertRawProviders(&raw.LLM.Providers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert LLM providers config: %w", err)
		}
		
		config.LLM = core.LLMConfig{
			DefaultProvider: raw.LLM.DefaultProvider,
			Providers:       *providersConfig,
			LoadBalancing:   raw.LLM.LoadBalancing,
			HealthCheck:     raw.LLM.HealthCheck,
		}
	} else if raw.Providers != nil {
		// Legacy format - use providers section
		providersConfig, err := convertRawProviders(raw.Providers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert providers config: %w", err)
		}
		
		// Use legacy load balancing and health check if provided
		loadBalancing := core.LoadBalancingConfig{}
		if raw.LoadBalancing != nil {
			loadBalancing = *raw.LoadBalancing
		}
		
		healthCheck := core.HealthCheckConfig{}
		if raw.HealthCheck != nil {
			healthCheck = *raw.HealthCheck
		}
		
		config.LLM = core.LLMConfig{
			DefaultProvider: raw.Providers.DefaultLLM,
			Providers:       *providersConfig,
			LoadBalancing:   loadBalancing,
			HealthCheck:     healthCheck,
		}
	} else {
		// No LLM configuration provided - use defaults
		config.LLM = getDefaultLLMConfig()
	}
	
	return config, nil
}

// convertRawProviders handles both legacy and new provider formats
func convertRawProviders(raw *rawProvidersConfig) (*core.ProvidersConfig, error) {
	config := &core.ProvidersConfig{
		List:   []core.ProviderConfig{},
		Legacy: make(map[string]core.ProviderConfig),
	}
	
	// Add providers from new array-based configuration
	for _, provider := range raw.List {
		// Ensure required fields are set
		if provider.Name == "" {
			return nil, fmt.Errorf("provider name cannot be empty in array configuration")
		}
		
		// Set defaults if not specified
		if provider.Weight == 0 {
			provider.Weight = 1
		}
		
		config.List = append(config.List, provider)
	}
	
	// Convert legacy direct provider configurations
	if raw.Ollama != nil {
		provider := *raw.Ollama
		if provider.Name == "" {
			provider.Name = "ollama"
		}
		if provider.Type == "" {
			provider.Type = "ollama"
		}
		if provider.Weight == 0 {
			provider.Weight = 10
		}
		if !provider.Enabled {
			provider.Enabled = true
		}
		config.Legacy[provider.Name] = provider
	}
	
	if raw.OpenAI != nil {
		provider := *raw.OpenAI
		if provider.Name == "" {
			provider.Name = "openai"
		}
		if provider.Type == "" {
			provider.Type = "openai"
		}
		if provider.Weight == 0 {
			provider.Weight = 10
		}
		if !provider.Enabled {
			provider.Enabled = true
		}
		config.Legacy[provider.Name] = provider
	}
	
	if raw.LMStudio != nil {
		provider := *raw.LMStudio
		if provider.Name == "" {
			provider.Name = "lmstudio"
		}
		if provider.Type == "" {
			provider.Type = "lmstudio"
		}
		if provider.Weight == 0 {
			provider.Weight = 10
		}
		if !provider.Enabled {
			provider.Enabled = true
		}
		config.Legacy[provider.Name] = provider
	}
	
	return config, nil
}

// fillDefaults sets default values for unspecified configuration fields
func fillDefaults(config *Config) {
	// DataDir should already be set by ResolveDataDir in Load function
	// Only set if still empty (shouldn't happen in normal flow)
	if config.DataDir == "" {
		config.DataDir = "./data"
	}
	
	// Set default log level
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	
	// Set default client info
	if config.ClientName == "" {
		config.ClientName = "rago-client"
	}
	if config.ClientVersion == "" {
		config.ClientVersion = "3.0.0"
	}
	
	// Auto-enable providers that don't have explicit enabled setting
	for i := range config.LLM.Providers.List {
		// If enabled is not explicitly set (false by default in Go), set it to true
		if !config.LLM.Providers.List[i].Enabled && config.LLM.Providers.List[i].Name != "" {
			config.LLM.Providers.List[i].Enabled = true
		}
	}
	
	// Set default LLM provider if not specified
	allProviders := config.LLM.Providers.GetEnabledProviders()
	if config.LLM.DefaultProvider == "" && len(allProviders) > 0 {
		config.LLM.DefaultProvider = allProviders[0].Name
	}
	
	// Fill default load balancing configuration
	if config.LLM.LoadBalancing.Strategy == "" {
		config.LLM.LoadBalancing.Strategy = "weighted_round_robin"
	}
	if config.LLM.LoadBalancing.MaxRetries == 0 {
		config.LLM.LoadBalancing.MaxRetries = 3
	}
	if config.LLM.LoadBalancing.RetryDelay == 0 {
		config.LLM.LoadBalancing.RetryDelay = 1 * time.Second
	}
	if config.LLM.LoadBalancing.CircuitBreakerThreshold == 0 {
		config.LLM.LoadBalancing.CircuitBreakerThreshold = 5
	}
	if config.LLM.LoadBalancing.CircuitBreakerTimeout == 0 {
		config.LLM.LoadBalancing.CircuitBreakerTimeout = 30 * time.Second
	}
	if config.LLM.LoadBalancing.CheckInterval == 0 {
		config.LLM.LoadBalancing.CheckInterval = 30 * time.Second
	}
	
	// Fill default health check configuration
	if config.LLM.HealthCheck.Interval == 0 {
		config.LLM.HealthCheck.Interval = 30 * time.Second
	}
	if config.LLM.HealthCheck.Timeout == 0 {
		config.LLM.HealthCheck.Timeout = 10 * time.Second
	}
	if config.LLM.HealthCheck.Retries == 0 {
		config.LLM.HealthCheck.Retries = 3
	}
	
	// Fill default RAG configuration
	if config.RAG.StorageBackend == "" {
		config.RAG.StorageBackend = "hybrid"
	}
	if config.RAG.ChunkingStrategy.Strategy == "" {
		config.RAG.ChunkingStrategy.Strategy = "fixed"
		config.RAG.ChunkingStrategy.ChunkSize = 1000
		config.RAG.ChunkingStrategy.ChunkOverlap = 200
		config.RAG.ChunkingStrategy.MinChunkSize = 50
	}
	if config.RAG.VectorStore.Backend == "" {
		config.RAG.VectorStore.Backend = "sqvect"
		config.RAG.VectorStore.Metric = "cosine"
		config.RAG.VectorStore.IndexType = "flat"
	}
	if config.RAG.KeywordStore.Backend == "" {
		config.RAG.KeywordStore.Backend = "bleve"
		config.RAG.KeywordStore.Analyzer = "standard"
		config.RAG.KeywordStore.Stemming = true
	}
	
	// Fill default MCP configuration
	// MCP is now optional - only set servers path if explicitly configured
	// This prevents MCP from trying to load non-existent server configs
	
	// Fill default Agents configuration
	if config.Agents.WorkflowEngine.StateBackend == "" {
		config.Agents.WorkflowEngine.StateBackend = "memory"
		config.Agents.WorkflowEngine.MaxSteps = 50
		config.Agents.WorkflowEngine.StepTimeout = 30 * time.Second
	}
	if config.Agents.Scheduling.Backend == "" {
		config.Agents.Scheduling.Backend = "memory"
		config.Agents.Scheduling.MaxConcurrent = 3
		config.Agents.Scheduling.QueueSize = 100
	}
}