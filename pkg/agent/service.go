package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	memorypkg "github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/prompt"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"golang.org/x/sync/errgroup"
)

// ProgressEvent 进度事件
type ProgressEvent struct {
	Type    string // "thinking", "tool_call", "tool_result", "done"
	Round   int
	Message string
	Tool    string
}

// ProgressCallback 进度回调函数
type ProgressCallback func(ProgressEvent)

// Service is the main agent service that handles planning and execution
// This matches the interface expected by the CLI in cmd/rago-cli/agent/agent.go
type Service struct {
	debug            bool
	llmService       domain.Generator
	mcpService       MCPToolExecutor
	ragProcessor     domain.Processor
	memoryService    domain.MemoryService
	skillsService    *skills.Service
	routerService    *router.Service // Semantic Router for fast intent recognition
	promptManager    *prompt.Manager // Central prompt management
	planner          *Planner
	executor         *Executor
	store            *Store
	agent            *Agent
	registry         *Registry
	logger           *slog.Logger
	cancelMu         sync.RWMutex
	cancelFunc       context.CancelFunc
	progressCb       ProgressCallback
	currentSessionID string // Auto-generated UUID for Chat() method
	sessionMu        sync.RWMutex
	memorySaveMu     sync.RWMutex
	memorySavedInRun bool
	ragSourcesMu     sync.RWMutex
	ragSources       []domain.Chunk // Collect RAG sources during execution

	// Model metadata for Info()
	modelName string
	baseURL   string

	// Hook system for lifecycle events
	hooks *HookRegistry

	// toolRegistry is the unified registry for custom, RAG, and Memory tools.
	// All modules register here so that both LLM listing and PTC callTool()
	// dispatch go through a single source of truth.
	toolRegistry *ToolRegistry

	// PTC (Programmatic Tool Calling) integration
	ptcIntegration *PTCIntegration

	// Execution history storage
	historyStore *HistoryStore

	// Public access to underlying services
	LLM     domain.Generator
	MCP     *mcp.Service // Full access to MCP service (Chat, StartServers, etc.)
	RAG     domain.Processor
	Memory  domain.MemoryService
	Router  *router.Service
	Skills  *skills.Service
	Prompts *prompt.Manager
	PTC     *PTCIntegration
}

// Ensure Service implements ptc.SearchProvider
var _ ptc.SearchProvider = (*Service)(nil)

// NewService creates a new agent service with the given dependencies.
//
// Deprecated: Prefer agent.New("name").WithRAG().WithMemory().Build() for
// a more ergonomic and composable construction. NewService is kept for
// internal use by the CLI and advanced callers that need fine-grained control.
func NewService(
	llmService domain.Generator,
	mcpService MCPToolExecutor,
	ragProcessor domain.Processor,
	agentDBPath string,
	memoryService domain.MemoryService,
) (*Service, error) {
	// Initialize store
	store, err := NewStore(agentDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent store: %w", err)
	}

	// Initialize prompt manager
	promptMgr := prompt.NewManager()

	// Collect available tools
	tools := collectAvailableTools(mcpService, ragProcessor, nil)

	// Build default agent instructions based on enabled features
	var instructions strings.Builder
	instructions.WriteString("You are RAGO, a helpful AI assistant")
	instructions.WriteString(" with access to")

	var features []string
	if ragProcessor != nil {
		features = append(features, " RAG (Retrieval Augmented Generation)")
	}
	features = append(features, " MCP tools, Skills, and various processing capabilities")
	instructions.WriteString(strings.Join(features, ","))
	instructions.WriteString(".\n\nWhen given a goal:\n1. Break it down into clear steps\n2. Choose appropriate tools for each step\n3. Execute steps in logical order\n4. Provide clear results\n\nAvailable tools include:")

	var toolList []string
	if ragProcessor != nil {
		toolList = append(toolList, "\n- RAG queries (rag_query, rag_ingest)")
	}
	toolList = append(toolList, "\n- MCP tools (external integrations)", "\n- Skills (reusable skill packages)", "\n- General LLM reasoning")
	instructions.WriteString(strings.Join(toolList, ""))

	// Create default agent
	agent := NewAgentWithConfig(
		"RAGO Agent",
		instructions.String(),
		tools,
	)

	// Initialize registry and register default agent
	registry := NewRegistry()
	registry.Register(agent)

	// Create planner (without router initially)
	planner := NewPlanner(llmService, tools)
	planner.SetPromptManager(promptMgr)

	// Inject prompt manager into memory service if it supports it
	if memoryService != nil {
		if s, ok := memoryService.(interface{ SetPromptManager(*prompt.Manager) }); ok {
			s.SetPromptManager(promptMgr)
		}
	}

	// Create executor
	executor := NewExecutor(llmService, nil, mcpService, ragProcessor, memoryService)

	// Initialize logger
	logger := ragolog.WithModule("agent.service")

	return &Service{
		llmService:    llmService,
		mcpService:    mcpService,
		ragProcessor:  ragProcessor,
		memoryService: memoryService,
		promptManager: promptMgr,
		planner:       planner,
		executor:      executor,
		store:         store,
		agent:         agent,
		registry:      registry,
		logger:        logger,
		hooks:         NewHookRegistry(),
		toolRegistry:  NewToolRegistry(),
		// Public fields - MCP is set separately via SetMCPService
		LLM:     llmService,
		MCP:     nil, // Set via SetMCPService for full access
		RAG:     ragProcessor,
		Memory:  memoryService,
		Prompts: promptMgr,
	}, nil
}

// Plan generates an execution plan for the given goal
// This matches the CLI expectation: agentService.Plan(ctx, goal)
func (s *Service) Plan(ctx context.Context, goal string) (*Plan, error) {
	session := NewSession(s.agent.ID())
	plan, err := s.planner.PlanWithFallback(ctx, goal, session)
	if err != nil {
		return nil, err
	}
	// Save plan to database
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}
	return plan, nil
}

// RevisePlan revises an existing plan based on user feedback.
// The user can modify the plan through natural language chat
func (s *Service) RevisePlan(ctx context.Context, plan *Plan, instruction string) (*Plan, error) {
	data := map[string]interface{}{
		"Goal":        plan.Goal,
		"Status":      plan.Status,
		"Steps":       plan.Steps,
		"Instruction": instruction,
	}

	rendered, err := s.promptManager.Render(prompt.AgentRevisePlan, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render revision prompt: %w", err)
	}

	// Call LLM to get revised plan
	response, err := s.llmService.Generate(ctx, rendered, &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the response
	var revisedPlan struct {
		Reasoning string `json:"reasoning"`
		Steps     []struct {
			Tool        string                 `json:"tool"`
			Description string                 `json:"description"`
			Arguments   map[string]interface{} `json:"arguments"`
		} `json:"steps"`
	}

	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no valid JSON in LLM response")
	}
	jsonStr := response[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &revisedPlan); err != nil {
		return nil, fmt.Errorf("failed to parse revised plan: %w", err)
	}

	// Create new plan with revisions
	newPlan := &Plan{
		ID:        uuid.New().String(),
		SessionID: plan.SessionID,
		Goal:      plan.Goal,
		Status:    PlanStatusPending,
		Reasoning: revisedPlan.Reasoning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert steps
	for i, step := range revisedPlan.Steps {
		newPlan.Steps = append(newPlan.Steps, Step{
			ID:          fmt.Sprintf("step-%d", i+1),
			Tool:        step.Tool,
			Description: step.Description,
			Arguments:   step.Arguments,
			Status:      StepStatusPending,
		})
	}

	// Save revised plan
	if err := s.store.SavePlan(newPlan); err != nil {
		return nil, fmt.Errorf("failed to save revised plan: %w", err)
	}

	return newPlan, nil
}

// ExecutePlan executes the given plan
// This matches the CLI expectation: agentService.ExecutePlan(ctx, plan)
func (s *Service) ExecutePlan(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	result, err := s.executor.ExecutePlan(ctx, plan, nil)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Save the plan state
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	if !result.Success {
		return result, fmt.Errorf("plan execution completed with errors: %s", result.Error)
	}

	return result, nil
}

// RunStream executes a goal and returns a stream of events
// This is the preferred method for reactive applications.
func (s *Service) RunStream(ctx context.Context, goal string) (<-chan *Event, error) {
	// Create or get session (auto-generated ID if not set)
	sessionID := s.CurrentSessionID()
	if sessionID == "" {
		s.ResetSession()
		sessionID = s.CurrentSessionID()
	}

	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
	}

	// Create Runtime
	runtime := NewRuntime(s, session)

	return runtime.RunStream(ctx, goal), nil
}

// Run executes a goal with optional configuration.
// Usage:
//
// // Simple
// result, err := svc.Run(ctx, "goal")
//
// // With options
// result, err := svc.Run(ctx, "goal",
//
//	agent.WithMaxTurns(10),
//	agent.WithSessionID("session-123"),
//	agent.WithStoreHistory(true),
//
// )
func (s *Service) Run(ctx context.Context, goal string, opts ...RunOption) (*ExecutionResult, error) {
	cfg := DefaultRunConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return s.runWithConfig(ctx, goal, cfg)
}

// runWithConfig is the internal implementation
func (s *Service) runWithConfig(ctx context.Context, goal string, cfg *RunConfig) (*ExecutionResult, error) {
	if cfg == nil {
		cfg = DefaultRunConfig()
	}
	s.resetRunMemorySaved()

	// Create cancellable context for this run
	runCtx, cancel := context.WithCancel(ctx)

	// Store cancel function for external cancellation
	s.cancelMu.Lock()
	s.cancelFunc = cancel
	s.cancelMu.Unlock()

	defer func() {
		s.cancelMu.Lock()
		s.cancelFunc = nil
		s.cancelMu.Unlock()
	}()

	// Load or create session based on SessionID
	var session *Session
	if cfg.SessionID != "" {
		var err error
		session, err = s.store.GetSession(cfg.SessionID)
		if err != nil {
			session = NewSessionWithID(cfg.SessionID, s.agent.ID())
		}
	} else {
		session = NewSession(s.agent.ID())
	}

	// Parallel Context Collection
	var (
		intent         *IntentRecognitionResult
		ragContext     string
		memoryContext  string
		memoryMemories []*domain.MemoryWithScore
		memoryLogic    string
	)

	g, groupCtx := errgroup.WithContext(runCtx)

	// 1. Intent Recognition
	g.Go(func() error {
		var err error
		intent, err = s.recognizeIntent(groupCtx, goal, session)
		return err
	})

	// 2. RAG Retrieval — skip when PTC is enabled (same reason as runtime.go:
	// the LLM must call rag_query explicitly via execute_javascript/callTool).
	if s.ragProcessor != nil && !s.isPTCEnabled() {
		g.Go(func() error {
			s.emitProgress("thinking", "🔍 Searching knowledge base...", 0, "")
			var err error
			ragContext, err = s.performRAGQuery(groupCtx, goal)
			if err == nil && ragContext != "" {
				s.emitProgress("tool_result", fmt.Sprintf("✓ Found %d relevant documents", countDocuments(ragContext)), 0, "")
			}
			return nil // Don't fail the whole run if RAG fails
		})
	}

	// 3. Memory Recall
	if s.memoryService != nil {
		g.Go(func() error {
			var err error
			memoryContext, memoryMemories, memoryLogic, err = s.memoryService.RetrieveAndInjectWithLogic(groupCtx, goal, session.GetID())
			return err
		})
	}

	// Wait for all context collection to finish
	if err := g.Wait(); err != nil {
		s.logger.Warn("Context collection partial failure", slog.Any("error", err))
	}

	// Execute: PTC is just a transport mode — branch internally, same public API.
	var finalResult interface{}
	var ptcRes *PTCResult

	if s.isPTCEnabled() {
		var err error
		finalResult, ptcRes, err = s.runPTCExecution(runCtx, goal, session, cfg)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		finalResult, err = s.executeWithLLM(runCtx, goal, intent, session, memoryContext, ragContext, cfg)
		if err != nil {
			return nil, err
		}
	}

	// Skip verification for faster response
	currentResult := finalResult

	// Add messages to session before saving
	session.AddMessage(domain.Message{
		Role:    "user",
		Content: goal,
	})
	if currentResult != nil {
		session.AddMessage(domain.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("%v", currentResult),
		})
	}

	// Create a simple plan to track this execution
	now := time.Now()
	plan := &Plan{
		ID:        uuid.New().String(),
		SessionID: session.GetID(),
		Goal:      goal,
		Status:    StatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
		Steps: []Step{
			{
				ID:          uuid.New().String(),
				Description: goal,
				Tool:        "llm",
				Status:      StepCompleted,
				Result:      currentResult,
			},
		},
	}
	if err := s.store.SavePlan(plan); err != nil {
		log.Printf("[Agent] Failed to save plan: %v", err)
	}

	result, err := s.finalizeExecution(runCtx, session, goal, intent, memoryMemories, memoryLogic, "", currentResult)
	if err != nil {
		return nil, err
	}
	if ptcRes != nil {
		result.PTCResult = ptcRes
	}
	return result, nil
}

// Cancel forcefully stops the current agent execution
func (s *Service) Cancel() bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if s.cancelFunc != nil {
		log.Printf("[Agent] Cancelling current execution...")
		s.cancelFunc()
		return true
	}
	return false
}

// ─── Cognitive Memory APIs ───────────────────────────────────────────────────

// TriggerReflection manually triggers memory consolidation for a session.
// The LLM analyses accumulated facts and generates higher-level observations.
// Returns a summary of what was consolidated, or an error.
func (s *Service) TriggerReflection(ctx context.Context, sessionID string) (string, error) {
	if s.memoryService == nil {
		return "", fmt.Errorf("memory service not configured")
	}
	return s.memoryService.Reflect(ctx, sessionID)
}

// ExplainMemory returns the full evolution graph for a memory, tracing how
// raw facts were consolidated into observations. Requires a file-based memory
// service (FileMemoryStore path).
func (s *Service) ExplainMemory(ctx context.Context, memoryID string) (*memorypkg.MemoryEvolutionNode, error) {
	svc, ok := s.memoryService.(*memorypkg.Service)
	if !ok {
		return nil, fmt.Errorf("ExplainMemory requires a *memory.Service (file-based store)")
	}
	return svc.GetEvolution(ctx, memoryID)
}

// SetAgentDirective stores a mission statement and hard directives as high-priority
// preference memories. These are injected into every prompt with the highest priority,
// overriding any conflicting context.
func (s *Service) SetAgentDirective(ctx context.Context, sessionID string, mission string, directives []string) error {
	if s.memoryService == nil {
		return fmt.Errorf("memory service not configured")
	}
	now := time.Now()
	if mission != "" {
		if err := s.memoryService.Add(ctx, &domain.Memory{
			Type:       domain.MemoryTypePreference,
			Content:    "Agent mission: " + mission,
			Importance: 1.0,
			SourceType: domain.MemorySourceUserInput,
			SessionID:  sessionID,
			CreatedAt:  now,
		}); err != nil {
			return fmt.Errorf("storing mission: %w", err)
		}
	}
	for _, d := range directives {
		if err := s.memoryService.Add(ctx, &domain.Memory{
			Type:       domain.MemoryTypePreference,
			Content:    "Directive: " + d,
			Importance: 1.0,
			SourceType: domain.MemorySourceUserInput,
			SessionID:  sessionID,
			CreatedAt:  now,
		}); err != nil {
			return fmt.Errorf("storing directive %q: %w", d, err)
		}
	}
	return nil
}
