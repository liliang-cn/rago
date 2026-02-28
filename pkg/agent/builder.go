package agent

import (
	"context"
	"fmt"
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

// ServiceBuilder provides a fluent API for creating agent services
type ServiceBuilder struct {
	name string
	cfg  *AgentConfig

	ragoCfg      *config.Config
	customDBPath string

	enablePTC    bool
	ptcConfig    *PTCConfig
	enableRAG    bool
	enableMCP    bool
	enableMemory bool
	enableRouter bool
	enableSkills bool
	skillsPaths  []string

	systemPrompt string
	debug        bool

	mcpOverride     *mcp.Service
	ragOverride     domain.Processor
	memoryOverride  domain.MemoryService
	routerOverride  *router.Service
	skillsOverride  *skills.Service
	progressCb      ProgressCallback
	routerThreshold float64
}

// NewBuilder creates a new ServiceBuilder with fluent API
func NewBuilder(name string) *ServiceBuilder {
	return &ServiceBuilder{
		name: name,
		cfg:  &AgentConfig{Name: name},
	}
}

// WithPTC enables PTC with default config
func (b *ServiceBuilder) WithPTC(opts ...PTCOption) *ServiceBuilder {
	b.enablePTC = true
	cfg := DefaultPTCConfig()
	cfg.Enabled = true
	for _, opt := range opts {
		opt(&cfg)
	}
	b.ptcConfig = &cfg
	return b
}

// WithPTCConfig enables PTC with custom config
func (b *ServiceBuilder) WithPTCConfig(cfg PTCConfig) *ServiceBuilder {
	b.enablePTC = true
	cfg.Enabled = true
	b.ptcConfig = &cfg
	return b
}

// WithRAG enables RAG processor
func (b *ServiceBuilder) WithRAG() *ServiceBuilder {
	b.enableRAG = true
	return b
}

// WithMCP enables MCP tools
func (b *ServiceBuilder) WithMCP() *ServiceBuilder {
	b.enableMCP = true
	return b
}

// WithMemory enables memory service
func (b *ServiceBuilder) WithMemory() *ServiceBuilder {
	b.enableMemory = true
	return b
}

// WithRouter enables semantic router
func (b *ServiceBuilder) WithRouter(threshold ...float64) *ServiceBuilder {
	b.enableRouter = true
	if len(threshold) > 0 {
		b.routerThreshold = threshold[0]
	}
	return b
}

// WithSkills enables skills service
func (b *ServiceBuilder) WithSkills(paths ...string) *ServiceBuilder {
	b.enableSkills = true
	b.skillsPaths = paths
	return b
}

// WithSystemPrompt sets custom system prompt
func (b *ServiceBuilder) WithSystemPrompt(prompt string) *ServiceBuilder {
	b.systemPrompt = prompt
	return b
}

// WithDebug enables debug mode
func (b *ServiceBuilder) WithDebug(debug bool) *ServiceBuilder {
	b.debug = debug
	return b
}

// WithProgressCallback sets progress callback
func (b *ServiceBuilder) WithProgressCallback(cb ProgressCallback) *ServiceBuilder {
	b.progressCb = cb
	return b
}

// WithConfig sets custom rago config
func (b *ServiceBuilder) WithConfig(cfg *config.Config) *ServiceBuilder {
	b.ragoCfg = cfg
	return b
}

// WithDBPath sets custom database path
func (b *ServiceBuilder) WithDBPath(path string) *ServiceBuilder {
	b.customDBPath = path
	return b
}

// WithMCPService overrides MCP service creation
func (b *ServiceBuilder) WithMCPService(mcpSvc *mcp.Service) *ServiceBuilder {
	b.mcpOverride = mcpSvc
	b.enableMCP = true
	return b
}

// WithRAGProcessor overrides RAG processor creation
func (b *ServiceBuilder) WithRAGProcessor(rag domain.Processor) *ServiceBuilder {
	b.ragOverride = rag
	b.enableRAG = true
	return b
}

// WithMemoryService overrides memory service creation
func (b *ServiceBuilder) WithMemoryService(mem domain.MemoryService) *ServiceBuilder {
	b.memoryOverride = mem
	b.enableMemory = true
	return b
}

// WithRouterService overrides router service creation
func (b *ServiceBuilder) WithRouterService(r *router.Service) *ServiceBuilder {
	b.routerOverride = r
	b.enableRouter = true
	return b
}

// WithSkillsService overrides skills service creation
func (b *ServiceBuilder) WithSkillsService(s *skills.Service) *ServiceBuilder {
	b.skillsOverride = s
	b.enableSkills = true
	return b
}

// Build creates the Service instance
func (b *ServiceBuilder) Build() (*Service, error) {
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
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	var mcpSvc *mcp.Service
	var mcpAdapter MCPToolExecutor
	if b.enableMCP {
		if b.mcpOverride != nil {
			mcpSvc = b.mcpOverride
		} else {
			mcpSvc, err = mcp.NewService(&ragoCfg.MCP, llmSvc)
			if err == nil {
				mcpSvc.StartServers(context.Background(), nil)
			}
		}
		if mcpSvc != nil {
			mcpAdapter = &mcpToolAdapter{service: mcpSvc}
		}
	}

	var memSvc domain.MemoryService
	if b.enableMemory {
		if b.memoryOverride != nil {
			memSvc = b.memoryOverride
		} else {
			memSvc, err = b.buildMemoryService(ragoCfg, embedSvc, llmSvc)
			if err != nil {
				return nil, fmt.Errorf("failed to create memory service: %w", err)
			}
		}
	}

	var routerSvc *router.Service
	if b.enableRouter {
		if b.routerOverride != nil {
			routerSvc = b.routerOverride
		} else {
			routerCfg := router.DefaultConfig()
			if b.routerThreshold > 0 {
				routerCfg.Threshold = b.routerThreshold
			}
			routerSvc, err = router.NewService(embedSvc, routerCfg)
			if err == nil {
				_ = routerSvc.RegisterDefaultIntents()
			}
		}
	}

	var ragProcessor domain.Processor
	if b.enableRAG {
		if b.ragOverride != nil {
			ragProcessor = b.ragOverride
		} else {
			ragProcessor, err = b.buildRAGProcessor(ragoCfg, embedSvc, llmSvc, memSvc)
			if err != nil {
				return nil, fmt.Errorf("failed to create RAG processor: %w", err)
			}
		}
	}

	var skillsSvc *skills.Service
	if b.enableSkills {
		if b.skillsOverride != nil {
			skillsSvc = b.skillsOverride
		} else {
			skillsSvc, err = b.buildSkillsService(ragoCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create skills service: %w", err)
			}
		}
	}

	agentDBPath := b.customDBPath
	if agentDBPath == "" {
		agentDBPath = ragoCfg.Sqvect.DBPath
	}

	svc, err := NewService(llmSvc, mcpAdapter, ragProcessor, agentDBPath, memSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent service: %w", err)
	}

	cfg := &AgentConfig{
		Name:            b.name,
		SystemPrompt:    b.systemPrompt,
		DBPath:          agentDBPath,
		EnableMCP:       b.enableMCP,
		EnableMemory:    b.enableMemory,
		EnableRAG:       b.enableRAG,
		EnableRouter:    b.enableRouter,
		EnableSkills:    b.enableSkills,
		EnablePTC:       b.enablePTC,
		Debug:           b.debug,
		RouterThreshold: b.routerThreshold,
		ProgressCb:      b.progressCb,
	}
	svc.config = cfg

	if b.systemPrompt != "" {
		svc.SetAgentInstructions(b.systemPrompt)
	}

	if b.progressCb != nil {
		svc.SetProgressCallback(b.progressCb)
	}

	if b.enablePTC && b.ptcConfig != nil {
		routerOpts := buildPTCRouterOptions(mcpAdapter, skillsSvc, ragProcessor)
		ptcRouter := ptc.NewRAGORouter(routerOpts...)

		ptcInteg, ptcErr := NewPTCIntegration(*b.ptcConfig, ptcRouter)
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

	return svc, nil
}

func (b *ServiceBuilder) buildMemoryService(ragoCfg *config.Config, embedSvc domain.Embedder, llmSvc domain.Generator) (domain.MemoryService, error) {
	var memStore domain.MemoryStore
	var shadowStore domain.MemoryStore
	var err error

	memPath := filepath.Join(ragoCfg.DataDir(), "memories")
	memStore, err = store.NewFileMemoryStore(memPath)
	if err != nil {
		return nil, err
	}

	sqlitePath := filepath.Join(ragoCfg.DataDir(), "rago.db")
	if sqliteStore, serr := store.NewMemoryStore(sqlitePath); serr == nil {
		_ = sqliteStore.InitSchema(context.Background())
		shadowStore = sqliteStore
	}

	memSvc := memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
	if shadowStore != nil {
		memSvc.SetShadowIndex(shadowStore)
	}

	return memSvc, nil
}

func (b *ServiceBuilder) buildRAGProcessor(ragoCfg *config.Config, embedSvc domain.Embedder, llmSvc domain.Generator, memSvc domain.MemoryService) (domain.Processor, error) {
	vectorStore, err := ragstore.NewVectorStore(ragstore.StoreConfig{
		Type: "sqlite",
		Parameters: map[string]interface{}{
			"db_path": ragoCfg.Sqvect.DBPath,
		},
	})
	if err != nil {
		return nil, err
	}

	docStore := ragstore.NewDocumentStoreFor(vectorStore)
	return ragprocessor.New(embedSvc, llmSvc, nil, vectorStore, docStore, ragoCfg, nil, memSvc), nil
}

func (b *ServiceBuilder) buildSkillsService(ragoCfg *config.Config) (*skills.Service, error) {
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

type PTCOption func(*PTCConfig)

func WithPTCMaxToolCalls(n int) PTCOption {
	return func(c *PTCConfig) { c.MaxToolCalls = n }
}

func WithPTCTimeout(d time.Duration) PTCOption {
	return func(c *PTCConfig) { c.Timeout = d }
}
