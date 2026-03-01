package agent

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
	ragprocessor "github.com/liliang-cn/rago/v2/pkg/rag/processor"
	ragstore "github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// ============================================================
// Config structures for JSON/YAML style initialization
// ============================================================

// Config holds all agent configuration
type Config struct {
	Name         string `json:"name"`
	DBPath       string `json:"db_path,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	Debug        bool   `json:"debug,omitempty"`

	RAG    *RAGConfig    `json:"rag,omitempty"`
	MCP    *MCPConfig    `json:"mcp,omitempty"`
	Memory *MemoryConfig `json:"memory,omitempty"`
	Router *RouterConfig `json:"router,omitempty"`
	Skills *SkillsConfig `json:"skills,omitempty"`
	PTC    *PTCConfig    `json:"ptc,omitempty"`

	ProgressCallback ProgressCallback `json:"-"`
}

// RAGConfig holds RAG configuration
type RAGConfig struct {
	Enabled    bool   `json:"enabled"`
	ChunkSize  int    `json:"chunk_size,omitempty"`
	Overlap    int    `json:"overlap,omitempty"`
	DBPath     string `json:"db_path,omitempty"`
	Collection string `json:"collection,omitempty"`
}

// MCPConfig holds MCP configuration
type MCPConfig struct {
	Enabled     bool     `json:"enabled"`
	ConfigPaths []string `json:"config_paths,omitempty"`
}

// MemoryConfig holds Memory configuration
type MemoryConfig struct {
	Enabled   bool   `json:"enabled"`
	DBPath    string `json:"db_path,omitempty"`
	StoreType string `json:"store_type,omitempty"` // "file", "vector", "hybrid"
}

// RouterConfig holds Router configuration
type RouterConfig struct {
	Enabled   bool    `json:"enabled"`
	Threshold float64 `json:"threshold,omitempty"`
}

// SkillsConfig holds Skills configuration
type SkillsConfig struct {
	Enabled bool     `json:"enabled"`
	Paths   []string `json:"paths,omitempty"`
}

// PTCConfig is defined in ptc_integration.go

// ============================================================
// Builder - chainable configuration without explicit Build()
// ============================================================

// Builder allows chainable agent configuration.
// Assign to (*Service, error) to build - no explicit Build() needed!
type Builder struct {
	name         string
	ragoCfg      *config.Config
	dbPath       string
	systemPrompt string
	debug        bool
	progressCb   ProgressCallback

	enableRAG       bool
	ragCfg          RAGConfig
	enableMCP       bool
	mcpCfgPaths     []string
	enableMemory    bool
	memoryCfg       MemoryConfig
	enableRouter    bool
	routerThreshold float64
	enableSkills    bool
	skillsPaths     []string
	enablePTC       bool
	ptcCfg          *PTCConfig

	// cached result
	svc *Service
	err error
}

// New creates a new agent builder for chainable configuration.
// No Build() needed - just assign to (*Service, error)!
//
// Example:
//
//	// Simple agent
//	svc, err := agent.New("my-agent")
//
//	// Chainable configuration
//	svc, err := agent.New("my-agent").WithRAG().WithMemory().WithMCP()
func New(name string) *Builder {
	return &Builder{name: name}
}

// WithRAG enables RAG processor
func (b *Builder) WithRAG(opts ...RAGOption) *Builder {
	b.enableRAG = true
	cfg := RAGConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	b.ragCfg = cfg
	return b
}

// WithMCP enables MCP tools
func (b *Builder) WithMCP(opts ...MCPOption) *Builder {
	b.enableMCP = true
	cfg := MCPConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	b.mcpCfgPaths = cfg.ConfigPaths
	return b
}

// WithMemory enables memory service
func (b *Builder) WithMemory(opts ...MemoryOption) *Builder {
	b.enableMemory = true
	cfg := MemoryConfig{StoreType: "file"} // default
	for _, opt := range opts {
		opt(&cfg)
	}
	b.memoryCfg = cfg
	return b
}

// WithRouter enables semantic router
func (b *Builder) WithRouter(opts ...RouterOption) *Builder {
	b.enableRouter = true
	cfg := RouterConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	b.routerThreshold = cfg.Threshold
	return b
}

// WithSkills enables skills service
func (b *Builder) WithSkills(opts ...SkillsOption) *Builder {
	b.enableSkills = true
	cfg := SkillsConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	b.skillsPaths = cfg.Paths
	return b
}

// WithPTC enables PTC
func (b *Builder) WithPTC(opts ...PTCOption) *Builder {
	b.enablePTC = true
	cfg := &PTCConfig{Enabled: true, MaxToolCalls: 20, Timeout: 600 * time.Second}
	for _, opt := range opts {
		opt(cfg)
	}
	b.ptcCfg = cfg
	return b
}

// WithDBPath sets database path
func (b *Builder) WithDBPath(path string) *Builder {
	b.dbPath = path
	return b
}

// WithSystemPrompt sets system prompt
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithDebug enables debug mode
func (b *Builder) WithDebug(debug bool) *Builder {
	b.debug = debug
	return b
}

// WithProgressCallback sets progress callback
func (b *Builder) WithProgressCallback(cb ProgressCallback) *Builder {
	b.progressCb = cb
	return b
}

// WithConfig sets rago config
func (b *Builder) WithConfig(cfg *config.Config) *Builder {
	b.ragoCfg = cfg
	return b
}

// Build constructs the Service. Called automatically on assignment.
func (b *Builder) Build() (*Service, error) {
	if b.svc != nil || b.err != nil {
		return b.svc, b.err
	}
	b.svc, b.err = b.build()
	return b.svc, b.err
}

// Unpack allows direct assignment to (*Service, error)
func (b *Builder) Unpack() (*Service, error) {
	return b.Build()
}

// Get builds and returns the Service (alias for Build)
func (b *Builder) Get() (*Service, error) {
	return b.Build()
}

func (b *Builder) build() (*Service, error) {
	if b.name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	ragoCfg := b.ragoCfg
	var err error
	if ragoCfg == nil {
		ragoCfg, err = config.Load("")
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(context.Background(), ragoCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize pool: %w", err)
	}

	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	embedSvc, err := globalPool.GetEmbeddingService(context.Background())
	if err != nil {
		log.Printf("[INFO] Embedding service not available: %v", err)
	}

	// Build MCP
	var mcpSvc *mcp.Service
	var mcpAdapter MCPToolExecutor
	if b.enableMCP {
		mcpCfg := &ragoCfg.MCP
		if len(b.mcpCfgPaths) > 0 {
			loadedCfg, loadErr := config.LoadMCPConfig(b.mcpCfgPaths...)
			if loadErr != nil {
				return nil, fmt.Errorf("failed to load MCP config: %w", loadErr)
			}
			mcpCfg = loadedCfg
		}
		mcpSvc, err = mcp.NewService(mcpCfg, llmSvc)
		if err != nil {
			log.Printf("[WARN] Failed to create MCP service: %v", err)
		} else {
			if startErr := mcpSvc.StartServers(context.Background(), nil); startErr != nil {
				log.Printf("[WARN] Failed to start MCP servers: %v", startErr)
			} else {
				log.Printf("[INFO] MCP servers started successfully")
			}
			mcpAdapter = &mcpToolAdapter{service: mcpSvc}
		}
	}

	// Build Memory
	var memSvc domain.MemoryService
	if b.enableMemory {
		memSvc, err = b.buildMemoryService(ragoCfg, embedSvc, llmSvc)
		if err != nil {
			return nil, fmt.Errorf("failed to create memory service: %w", err)
		}
	}

	// Build RAG
	var ragProcessor domain.Processor
	if b.enableRAG {
		if embedSvc == nil {
			log.Printf("[WARN] RAG requires embedding model, but none available. RAG disabled.")
		} else {
			ragProcessor, err = b.buildRAGProcessor(ragoCfg, embedSvc, llmSvc, memSvc)
			if err != nil {
				return nil, fmt.Errorf("failed to create RAG processor: %w", err)
			}
		}
	}

	// Build Router
	var routerSvc *router.Service
	if b.enableRouter {
		routerCfg := router.DefaultConfig()
		if b.routerThreshold > 0 {
			routerCfg.Threshold = b.routerThreshold
		}
		routerSvc, err = router.NewService(embedSvc, routerCfg)
		if err == nil {
			_ = routerSvc.RegisterDefaultIntents()
		}
	}

	// Build Skills
	var skillsSvc *skills.Service
	if b.enableSkills {
		skillsSvc, err = b.buildSkillsService(ragoCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create skills service: %w", err)
		}
	}

	// DB Path
	dbPath := b.dbPath
	if dbPath == "" {
		dbPath = filepath.Join(ragoCfg.DataDir(), "agent.db")
	}

	// Create service
	svc, err := NewService(llmSvc, mcpAdapter, ragProcessor, dbPath, memSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Store model metadata for Info()
	if len(ragoCfg.LLMPool.Providers) > 0 {
		p := ragoCfg.LLMPool.Providers[0]
		svc.SetModelInfo(p.ModelName, p.BaseURL)
	}

	// Apply config
	svc.SetDebug(b.debug)
	if b.systemPrompt != "" {
		svc.SetAgentInstructions(b.systemPrompt)
	}
	if b.progressCb != nil {
		svc.SetProgressCallback(b.progressCb)
	}

	// PTC
	if b.enablePTC && b.ptcCfg != nil {
		routerOpts := buildPTCRouterOptions(mcpAdapter, skillsSvc, ragProcessor)
		ptcRouter := ptc.NewRAGORouter(routerOpts...)
		ptcInteg, ptcErr := NewPTCIntegration(*b.ptcCfg, ptcRouter)
		if ptcErr != nil {
			return nil, fmt.Errorf("failed to create PTC integration: %w", ptcErr)
		}
		svc.SetPTC(ptcInteg)
	}

	if routerSvc != nil {
		svc.SetRouter(routerSvc)
	}
	if skillsSvc != nil {
		svc.SetSkillsService(skillsSvc)
	}
	if mcpSvc != nil {
		svc.SetMCPService(mcpSvc)
	}

	return svc, nil
}

func (b *Builder) buildMemoryService(ragoCfg *config.Config, embedSvc domain.Embedder, llmSvc domain.Generator) (domain.MemoryService, error) {
	var memStore domain.MemoryStore
	var shadowStore domain.MemoryStore
	var err error

	memPath := b.memoryCfg.DBPath
	if memPath == "" {
		memPath = filepath.Join(ragoCfg.DataDir(), "memories")
	}

	storeType := b.memoryCfg.StoreType
	if storeType == "" {
		storeType = "file"
	}

	// Warn if vector/hybrid requested but no embedding model available
	if (storeType == "vector" || storeType == "hybrid") && embedSvc == nil {
		log.Printf("[WARN] Memory store type '%s' requires embedding model, but none available. Falling back to 'file'.", storeType)
		storeType = "file"
	}

	switch storeType {
	case "file":
		memStore, err = store.NewFileMemoryStore(memPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file memory store: %w", err)
		}
	case "vector":
		sqlitePath := filepath.Join(ragoCfg.DataDir(), "rago.db")
		memStore, err = store.NewMemoryStore(sqlitePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector memory store: %w", err)
		}
		if err := memStore.InitSchema(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to init memory schema: %w", err)
		}
	case "hybrid":
		memStore, err = store.NewFileMemoryStore(memPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file memory store: %w", err)
		}
		sqlitePath := filepath.Join(ragoCfg.DataDir(), "rago.db")
		if sqliteStore, serr := store.NewMemoryStore(sqlitePath); serr == nil {
			_ = sqliteStore.InitSchema(context.Background())
			shadowStore = sqliteStore
		}
	default:
		return nil, fmt.Errorf("unsupported memory store type: %s", storeType)
	}

	memSvc := memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
	if shadowStore != nil {
		memSvc.SetShadowIndex(shadowStore)
	}
	return memSvc, nil
}

func (b *Builder) buildRAGProcessor(ragoCfg *config.Config, embedSvc domain.Embedder, llmSvc domain.Generator, memSvc domain.MemoryService) (domain.Processor, error) {
	vectorStore, err := ragstore.NewVectorStore(ragstore.StoreConfig{
		Type:       "sqlite",
		Parameters: map[string]interface{}{"db_path": ragoCfg.Sqvect.DBPath},
	})
	if err != nil {
		return nil, err
	}
	docStore := ragstore.NewDocumentStoreFor(vectorStore)
	return ragprocessor.New(embedSvc, llmSvc, nil, vectorStore, docStore, ragoCfg, nil, memSvc), nil
}

func (b *Builder) buildSkillsService(ragoCfg *config.Config) (*skills.Service, error) {
	skillsCfg := skills.DefaultConfig()
	paths := b.skillsPaths
	if len(paths) == 0 {
		paths = []string{ragoCfg.SkillsDir()}
	}
	skillsCfg.Paths = paths
	skillsCfg.DBPath = ragoCfg.Sqvect.DBPath
	svc, err := skills.NewService(skillsCfg)
	if err != nil {
		return nil, err
	}
	_ = svc.LoadAll(context.Background())
	return svc, nil
}

// ============================================================
// Option types for nested configuration
// ============================================================

// RAGOption modifies RAGConfig
type RAGOption func(*RAGConfig)

// WithRAGChunkSize sets RAG chunk size
func WithRAGChunkSize(size int) RAGOption { return func(c *RAGConfig) { c.ChunkSize = size } }

// WithRAGOverlap sets RAG overlap
func WithRAGOverlap(overlap int) RAGOption { return func(c *RAGConfig) { c.Overlap = overlap } }

// WithRAGDBPath sets RAG database path
func WithRAGDBPath(path string) RAGOption { return func(c *RAGConfig) { c.DBPath = path } }

// MCPOption modifies MCPConfig
type MCPOption func(*MCPConfig)

// WithMCPConfigPaths sets MCP config file paths
func WithMCPConfigPaths(paths ...string) MCPOption {
	return func(c *MCPConfig) { c.ConfigPaths = paths }
}

// MemoryOption modifies MemoryConfig
type MemoryOption func(*MemoryConfig)

// WithMemoryDBPath sets memory database path
func WithMemoryDBPath(path string) MemoryOption {
	return func(c *MemoryConfig) { c.DBPath = path }
}

// WithMemoryStoreType sets memory store type: "file", "vector", or "hybrid"
func WithMemoryStoreType(storeType string) MemoryOption {
	return func(c *MemoryConfig) { c.StoreType = storeType }
}

// RouterOption modifies RouterConfig
type RouterOption func(*RouterConfig)

// WithRouterThreshold sets router threshold
func WithRouterThreshold(threshold float64) RouterOption {
	return func(c *RouterConfig) { c.Threshold = threshold }
}

// SkillsOption modifies SkillsConfig
type SkillsOption func(*SkillsConfig)

// WithSkillsPaths sets skills paths
func WithSkillsPaths(paths ...string) SkillsOption {
	return func(c *SkillsConfig) { c.Paths = paths }
}

// PTCOption modifies PTCConfig
type PTCOption func(*PTCConfig)

// WithPTCMaxToolCalls sets max tool calls
func WithPTCMaxToolCalls(n int) PTCOption { return func(c *PTCConfig) { c.MaxToolCalls = n } }

// WithPTCTimeout sets PTC timeout
func WithPTCTimeout(d time.Duration) PTCOption { return func(c *PTCConfig) { c.Timeout = d } }

// NewWithConfig creates a new agent service from a Config structure.
// This is the second way to create an agent (alongside the builder pattern).
//
// Example:
//
//	svc, err := agent.NewWithConfig(&agent.Config{
//	    Name: "my-agent",
//	    MCP:  &agent.MCPConfig{Enabled: true},
//	    RAG:  &agent.RAGConfig{Enabled: true, DBPath: "/path/to/db"},
//	})
func NewWithConfig(cfg *Config) (*Service, error) {
	if cfg == nil || cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	b := &Builder{
		name:         cfg.Name,
		dbPath:       cfg.DBPath,
		systemPrompt: cfg.SystemPrompt,
		debug:        cfg.Debug,
	}

	if cfg.RAG != nil && cfg.RAG.Enabled {
		b.enableRAG = true
		b.ragCfg = *cfg.RAG
	}
	if cfg.MCP != nil && cfg.MCP.Enabled {
		b.enableMCP = true
		b.mcpCfgPaths = cfg.MCP.ConfigPaths
	}
	if cfg.Memory != nil && cfg.Memory.Enabled {
		b.enableMemory = true
		b.memoryCfg = *cfg.Memory
	}
	if cfg.Router != nil && cfg.Router.Enabled {
		b.enableRouter = true
		b.routerThreshold = cfg.Router.Threshold
	}
	if cfg.Skills != nil && cfg.Skills.Enabled {
		b.enableSkills = true
		b.skillsPaths = cfg.Skills.Paths
	}
	if cfg.PTC != nil && cfg.PTC.Enabled {
		b.enablePTC = true
		b.ptcCfg = cfg.PTC
	}
	if cfg.ProgressCallback != nil {
		b.progressCb = cfg.ProgressCallback
	}

	return b.Build()
}
