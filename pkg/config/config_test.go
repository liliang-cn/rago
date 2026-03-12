package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/agent-go/pkg/mcp"
	"github.com/liliang-cn/agent-go/pkg/pool"
)

func validConfig(home string) *Config {
	return &Config{
		Home: home,
		Server: ServerConfig{
			Port: 7127,
			Host: "127.0.0.1",
		},
		RAG: RAGConfig{
			Storage: CortexdbConfig{
				DBPath:    filepath.Join(home, "data", "agentgo.db"),
				TopK:      5,
				Threshold: 0.1,
				IndexType: "hnsw",
			},
			Chunker: ChunkerConfig{
				ChunkSize: 500,
				Overlap:   50,
				Method:    "sentence",
			},
		},
		MCP: defaultMCPConfig(),
		Skills: SkillsConfig{
			Paths: []string{"custom-skills"},
		},
		Memory: MemoryConfig{
			StoreType:  "file",
			MemoryPath: filepath.Join(home, "data", "memories"),
		},
		Cache: CacheConfig{
			StoreType:         "file",
			Path:              filepath.Join(home, "data", "cache"),
			MaxSize:           1000,
			EnableQueryCache:  true,
			EnableVectorCache: true,
			EnableLLMCache:    true,
			EnableChunkCache:  true,
			QueryCacheTTL:     15 * time.Minute,
			VectorCacheTTL:    24 * time.Hour,
			LLMCacheTTL:       time.Hour,
			ChunkCacheTTL:     24 * time.Hour,
		},
	}
}

func defaultMCPConfig() mcp.Config {
	return mcp.DefaultConfig()
}

func TestConfigValidate(t *testing.T) {
	cfg := validConfig(t.TempDir())
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestConfigValidateFailures(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Config)
		want string
	}{
		{"bad port", func(c *Config) { c.Server.Port = 0 }, "invalid server port"},
		{"empty host", func(c *Config) { c.Server.Host = "" }, "server host cannot be empty"},
		{"empty db path", func(c *Config) { c.RAG.Storage.DBPath = "" }, "database path cannot be empty"},
		{"bad topk", func(c *Config) { c.RAG.Storage.TopK = 0 }, "topK must be positive"},
		{"bad threshold", func(c *Config) { c.RAG.Storage.Threshold = -1 }, "threshold must be non-negative"},
		{"bad index type", func(c *Config) { c.RAG.Storage.IndexType = "bad" }, "invalid index_type"},
		{"bad chunk size", func(c *Config) { c.RAG.Chunker.ChunkSize = 0 }, "chunk size must be positive"},
		{"bad overlap", func(c *Config) { c.RAG.Chunker.Overlap = 500 }, "overlap must be between 0 and chunk size"},
		{"bad method", func(c *Config) { c.RAG.Chunker.Method = "bad" }, "invalid chunker method"},
		{"bad mcp timeout", func(c *Config) { c.MCP.Enabled = true; c.MCP.DefaultTimeout = 0 }, "default_timeout must be positive"},
		{"bad mcp concurrency", func(c *Config) { c.MCP.Enabled = true; c.MCP.MaxConcurrentRequests = -1 }, "max_concurrent_requests must be non-negative"},
		{"bad mcp health", func(c *Config) { c.MCP.Enabled = true; c.MCP.HealthCheckInterval = 0 }, "health_check_interval must be positive"},
		{"bad mcp log level", func(c *Config) { c.MCP.Enabled = true; c.MCP.LogLevel = "trace" }, "invalid log_level"},
		{"empty mcp server file", func(c *Config) { c.MCP.Enabled = true; c.MCP.Servers = []string{""} }, "empty server config file path"},
		{"bad cache store", func(c *Config) { c.Cache.StoreType = "bad" }, "invalid store_type"},
		{"bad cache max size", func(c *Config) { c.Cache.MaxSize = 0 }, "max_size must be positive"},
		{"bad cache ttl", func(c *Config) { c.Cache.QueryCacheTTL = 0 }, "query_ttl must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(t.TempDir())
			tt.mut(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in %v", tt.want, err)
			}
		})
	}
}

func TestResolveDatabasePath(t *testing.T) {
	home := t.TempDir()

	t.Run("file memory defaults", func(t *testing.T) {
		cfg := validConfig(home)
		cfg.RAG.Storage.DBPath = ""
		cfg.Memory.MemoryPath = ""
		cfg.Memory.StoreType = "file"

		cfg.resolveDatabasePath()

		if got := cfg.RAG.Storage.DBPath; got != filepath.Join(cfg.DataDir(), "agentgo.db") {
			t.Fatalf("unexpected db path: %s", got)
		}
		if got := cfg.Memory.MemoryPath; got != filepath.Join(cfg.DataDir(), "memories") {
			t.Fatalf("unexpected memory path: %s", got)
		}
		if got := cfg.Cache.Path; got != filepath.Join(cfg.DataDir(), "cache") {
			t.Fatalf("unexpected cache path: %s", got)
		}
	})

	t.Run("vector memory reuses rag db", func(t *testing.T) {
		cfg := validConfig(home)
		cfg.RAG.Storage.DBPath = filepath.Join(home, "data", "rag.db")
		cfg.Memory.MemoryPath = ""
		cfg.Memory.StoreType = "vector"

		cfg.resolveDatabasePath()

		if cfg.Memory.MemoryPath != cfg.RAG.Storage.DBPath {
			t.Fatalf("expected vector memory path to reuse db path, got %s", cfg.Memory.MemoryPath)
		}
	})
}

func TestResolveMCPServerPaths(t *testing.T) {
	home := t.TempDir()
	cfg := validConfig(home)
	unified := filepath.Join(home, "mcpServers.json")
	cfg.MCP.Servers = []string{"./mcpServers.json", "/tmp/custom.json", unified}

	cfg.resolveMCPServerPaths()

	if len(cfg.MCP.Servers) != 2 {
		t.Fatalf("expected 2 server paths, got %v", cfg.MCP.Servers)
	}
	foundUnified := false
	for _, path := range cfg.MCP.Servers {
		if path == unified {
			foundUnified = true
		}
		if path == "./mcpServers.json" {
			t.Fatalf("expected legacy path to be removed, got %v", cfg.MCP.Servers)
		}
	}
	if !foundUnified {
		t.Fatalf("expected unified path to be present, got %v", cfg.MCP.Servers)
	}
}

func TestSkillsPaths(t *testing.T) {
	home := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	cfg := validConfig(home)
	cfg.Skills.Paths = []string{"relative-skills", filepath.Join(home, "skills")}

	paths := cfg.SkillsPaths()

	if paths[0] != filepath.Join(home, "relative-skills") {
		t.Fatalf("expected relative configured path to resolve under home, got %s", paths[0])
	}

	expected := map[string]bool{
		filepath.Join(home, "relative-skills"):         false,
		filepath.Join(home, "skills"):                  false,
		".skills":                                      false,
		filepath.Join(".agentgo", "skills"):            false,
		filepath.Join(userHome, ".agents", "skills"): false,
	}
	for _, p := range paths {
		if _, ok := expected[p]; ok {
			expected[p] = true
		}
	}
	for p, seen := range expected {
		if !seen {
			t.Fatalf("expected path %s in skills paths, got %v", p, paths)
		}
	}
}

func TestExpandAndEnsurePaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg := validConfig("~/agentgo-home")
	cfg.RAG.Storage.DBPath = "~/agentgo-home/data/agentgo.db"
	cfg.Memory.MemoryPath = "~/agentgo-home/data/memories"
	cfg.Memory.StoreType = "file"
	cfg.Cache.Path = "~/agentgo-home/data/cache"
	cfg.Cache.StoreType = "file"

	cfg.expandPaths()

	if !strings.HasPrefix(cfg.Home, homeDir) {
		t.Fatalf("expected expanded home path, got %s", cfg.Home)
	}
	if _, err := os.Stat(filepath.Dir(cfg.RAG.Storage.DBPath)); err != nil {
		t.Fatalf("expected rag parent dir to exist: %v", err)
	}
	if _, err := os.Stat(cfg.Memory.MemoryPath); err != nil {
		t.Fatalf("expected memory dir to exist: %v", err)
	}
	if _, err := os.Stat(cfg.Cache.Path); err != nil {
		t.Fatalf("expected cache dir to exist: %v", err)
	}
}

func TestUnmarshalProvidersAliases(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"name":                    "primary",
			"base_url":                "http://localhost:11434/v1",
			"key":                     "test",
			"model_name":              "gpt-test",
			"max_concurrent_requests": 7,
			"capability_rating":       5,
		},
	}

	var providers []pool.Provider
	if err := unmarshalProviders(raw, &providers); err != nil {
		t.Fatalf("unmarshalProviders failed: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].MaxConcurrency != 7 {
		t.Fatalf("expected max concurrency alias to map, got %d", providers[0].MaxConcurrency)
	}
	if providers[0].Capability != 5 {
		t.Fatalf("expected capability alias to map, got %d", providers[0].Capability)
	}

	data, err := json.Marshal(providers[0])
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"max_concurrency":7`) {
		t.Fatalf("expected canonical json field, got %s", string(data))
	}
}

func TestEnvFallbackHelpers(t *testing.T) {
	t.Setenv("CFG_STRING", "value")
	t.Setenv("CFG_INT", "42")
	t.Setenv("CFG_BOOL", "true")
	t.Setenv("CFG_BAD_INT", "oops")
	t.Setenv("CFG_BAD_BOOL", "oops")

	if got := GetEnvOrDefault("CFG_STRING", "default"); got != "value" {
		t.Fatalf("unexpected string env value: %s", got)
	}
	if got := GetEnvOrDefault("CFG_MISSING", "default"); got != "default" {
		t.Fatalf("unexpected default string: %s", got)
	}
	if got := GetEnvOrDefaultInt("CFG_INT", 1); got != 42 {
		t.Fatalf("unexpected int env value: %d", got)
	}
	if got := GetEnvOrDefaultInt("CFG_BAD_INT", 7); got != 7 {
		t.Fatalf("unexpected bad int fallback: %d", got)
	}
	if got := GetEnvOrDefaultBool("CFG_BOOL", false); !got {
		t.Fatal("expected true bool env value")
	}
	if got := GetEnvOrDefaultBool("CFG_BAD_BOOL", true); !got {
		t.Fatal("expected bad bool to fall back to default")
	}
}

func TestLoadMCPConfigMissingFileIsTolerated(t *testing.T) {
	cfg, err := LoadMCPConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("expected missing file to be tolerated, got %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("expected returned mcp config to remain enabled")
	}
}
