package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	ragprocessor "github.com/liliang-cn/rago/v2/pkg/rag/processor"
	ragstore "github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/prompt"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/liliang-cn/rago/v2/pkg/store"
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
	config           *AgentConfig
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

	// Hook system for lifecycle events
	hooks *HookRegistry

	// Public access to underlying services
	LLM     domain.Generator
	MCP     MCPToolExecutor
	RAG     domain.Processor
	Memory  domain.MemoryService
	Router  *router.Service
	Skills  *skills.Service
	Prompts *prompt.Manager
}

// NewService creates a new agent service
// This matches the signature expected by the CLI:
// agent.NewService(llmService, mcpService, processor, agentDBPath, memoryService)
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

	// Create default agent
	agent := NewAgentWithConfig(
		"RAGO Agent",
		`You are RAGO, a helpful AI assistant with access to RAG (Retrieval Augmented Generation),
MCP tools, Skills, and various processing capabilities.

When given a goal:
1. Break it down into clear steps
2. Choose appropriate tools for each step
3. Execute steps in logical order
4. Provide clear results

Available tools include:
- RAG queries (rag_query, rag_ingest)
- MCP tools (external integrations)
- Skills (reusable skill packages)
- General LLM reasoning`,
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
		config:        &AgentConfig{}, // Default empty config
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
		hooks:         GlobalHookRegistry(),
		// Public fields
		LLM:     llmService,
		MCP:     mcpService,
		RAG:     ragProcessor,
		Memory:  memoryService,
		Prompts: promptMgr,
	}, nil
}

// SetConfig updates the agent's runtime configuration
func (s *Service) SetConfig(cfg *AgentConfig) {
	if cfg != nil {
		s.config = cfg
	}
}

// SetRouter sets the semantic router for improved intent recognition
func (s *Service) SetRouter(routerService *router.Service) {
	s.routerService = routerService
	s.Router = routerService
	s.planner.SetRouter(routerService)
}

// RegisterAgent registers a new agent with the service
func (s *Service) RegisterAgent(agent *Agent) {
	if s.registry != nil {
		s.registry.Register(agent)
	}
}

// SetSkillsService sets the skills service for agent integration
func (s *Service) SetSkillsService(skillsService *skills.Service) {
	s.skillsService = skillsService
	s.Skills = skillsService
	// Re-create agent with updated tools
	if skillsService != nil {
		tools := collectAvailableTools(s.mcpService, s.ragProcessor, skillsService)
		s.agent = NewAgentWithConfig(
			s.agent.Name(),
			s.agent.Instructions(),
			tools,
		)
		s.planner = NewPlanner(s.llmService, tools)
		// Restore router if it was set
		if s.routerService != nil {
			s.planner.SetRouter(s.routerService)
		}
		// Set skills service on executor for tool execution
		s.executor.SetSkillsService(skillsService)
	}
}

// SetProgressCallback sets the progress callback for execution events
func (s *Service) SetProgressCallback(cb ProgressCallback) {
	s.progressCb = cb
}

// SetAgentInstructions sets the instructions for the default agent
func (s *Service) SetAgentInstructions(instructions string) {
	if s.agent != nil {
		s.agent.SetInstructions(instructions)
	}
}

// RegisterHook registers a hook for lifecycle events
// Returns the hook ID for later unregistration
func (s *Service) RegisterHook(event HookEvent, handler HookHandler, opts ...HookOption) string {
	return s.hooks.Register(event, handler, opts...)
}

// UnregisterHook removes a hook by ID
func (s *Service) UnregisterHook(hookID string) bool {
	return s.hooks.Unregister(hookID)
}

// GetHooks returns the hook registry for advanced usage
func (s *Service) GetHooks() *HookRegistry {
	return s.hooks
}

// CreateSubAgent creates a sub-agent wrapper for isolated execution
func (s *Service) CreateSubAgent(agent *Agent, goal string, opts ...SubAgentOption) *SubAgent {
	cfg := SubAgentConfig{
		Agent:   agent,
		Goal:    goal,
		Service: s,
	}
	return NewSubAgent(cfg, opts...)
}

// emitProgress emits a progress event if callback is set
func (s *Service) emitProgress(eventType, message string, round int, tool string) {
	if s.progressCb != nil {
		s.progressCb(ProgressEvent{
			Type:    eventType,
			Round:   round,
			Message: message,
			Tool:    tool,
		})
	}
}

// AddFunctionSkill adds a function-based skill dynamically
func (s *Service) AddFunctionSkill(id, name, description string, fn func(ctx context.Context, vars map[string]interface{}) (string, error)) error {
	if s.skillsService == nil {
		return fmt.Errorf("skills service not initialized")
	}
	s.skillsService.RegisterFunction(id, name, description, fn)
	return nil
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

// Run executes a goal using a fixed workflow:
// 1. Intent Recognition
// 2. Check Memory
// 3. RAG Query
// Run executes a goal - simple and dynamic:
// 1. Collect all available tools (MCP + Skills + RAG)
// 2. Intent Recognition
// 3. Let LLM match intent to tools and execute
func (s *Service) Run(ctx context.Context, goal string) (*ExecutionResult, error) {
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

	session := NewSession(s.agent.ID())

	// Parallel Context Collection
	var (
		intent         *IntentRecognitionResult
		ragContext     string
		memoryContext  string
		memoryMemories []*domain.MemoryWithScore
	)

	g, groupCtx := errgroup.WithContext(runCtx)

	// 1. Intent Recognition
	g.Go(func() error {
		var err error
		intent, err = s.recognizeIntent(groupCtx, goal, session)
		return err
	})

	// 2. RAG Retrieval
	if s.ragProcessor != nil {
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
			memoryContext, memoryMemories, err = s.memoryService.RetrieveAndInject(groupCtx, goal, session.GetID())
			return err
		})
	}

	// Wait for all context collection to finish
	if err := g.Wait(); err != nil {
		s.logger.Warn("Context collection partial failure", slog.Any("error", err))
	}

	// Step 5: Let LLM decide and execute (with gathered context)
	finalResult, err := s.executeWithLLM(runCtx, goal, intent, session, memoryContext, ragContext)
	if err != nil {
		return nil, err
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

	return s.finalizeExecution(runCtx, session, goal, intent, memoryMemories, "", currentResult)
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

// collectAllAvailableTools collects tools from MCP, Skills, RAG, and Agent Handoffs
func (s *Service) collectAllAvailableTools(ctx context.Context, currentAgent *Agent) []domain.ToolDefinition {
	toolsMap := make(map[string]domain.ToolDefinition)

	// Helper to add tools with deduplication
	addTools := func(defs []domain.ToolDefinition) {
		for _, tool := range defs {
			toolsMap[tool.Function.Name] = tool
		}
	}

	// Agent Handoffs and Tools
	if currentAgent != nil {
		for _, handoff := range currentAgent.Handoffs() {
			tool := handoff.ToToolDefinition().ToDomainTool()
			toolsMap[tool.Function.Name] = tool
		}
		addTools(currentAgent.Tools())
	}

	// MCP tools
	if s.mcpService != nil {
		allMCP := s.mcpService.ListTools()
		if isAllAllowed(currentAgent.mcpTools) {
			addTools(allMCP)
		} else {
			for _, tool := range allMCP {
				if containsStr(currentAgent.mcpTools, tool.Function.Name) {
					addTools([]domain.ToolDefinition{tool})
				}
			}
		}
	}

	// Skills tools
	if s.skillsService != nil {
		skillsList, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
		allowedAll := isAllAllowed(currentAgent.skills)
		for _, sk := range skillsList {
			if allowedAll || containsStr(currentAgent.skills, sk.ID) {
				toolsMap[sk.ID] = domain.ToolDefinition{
					Type: "function",
					Function: domain.ToolFunction{
						Name:        sk.ID,
						Description: sk.Description,
						Parameters:  map[string]interface{}{},
					},
				}
			}
		}
	}

	// RAG tools
	if s.ragProcessor != nil {
		addTools([]domain.ToolDefinition{
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "rag_query",
					Description: "Search knowledge base for information",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
							},
						},
					},
				},
			},
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "rag_ingest",
					Description: "Ingest a document into the RAG system",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"content": map[string]interface{}{
								"type":        "string",
								"description": "The document content",
							},
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "Path to the document file",
							},
						},
					},
				},
			},
		})
	}

	// Add Memory tools
	if s.memoryService != nil {
		addTools([]domain.ToolDefinition{
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "memory_save",
					Description: "Save information to long-term memory for future reference",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"content": map[string]interface{}{
								"type":        "string",
								"description": "The information to remember",
							},
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Type of memory (fact, preference, skill, pattern, context)",
								"enum":        []string{"fact", "preference", "skill", "pattern", "context"},
							},
						},
						"required": []string{"content"},
					},
				},
			},
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "memory_update",
					Description: "Update an existing memory entry by its ID",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]interface{}{
								"type":        "string",
								"description": "The ID of the memory to update (e.g., 'mem_123')",
							},
							"content": map[string]interface{}{
								"type":        "string",
								"description": "The new content for the memory",
							},
						},
						"required": []string{"id", "content"},
					},
				},
			},
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "memory_delete",
					Description: "Permanently remove a memory entry by its ID",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]interface{}{
								"type":        "string",
								"description": "The ID of the memory to delete",
							},
						},
						"required": []string{"id"},
					},
				},
			},
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "memory_recall",
					Description: "Recall information from long-term memory",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "The query to search memory for",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		})
	}

	// Convert map to slice
	var tools []domain.ToolDefinition
	for _, tool := range toolsMap {
		tools = append(tools, tool)
	}

	return tools
}

// buildSystemPrompt 构建包含系统上下文的system prompt
func (s *Service) buildSystemPrompt(agent *Agent) string {
	systemCtx := s.buildSystemContext()

	data := map[string]interface{}{
		"AgentInstructions": agent.Instructions(),
		"SystemContext":     systemCtx.FormatForPrompt(),
	}

	rendered, err := s.promptManager.Render(prompt.AgentSystemPrompt, data)
	if err != nil {
		// Fallback
		return agent.Instructions() + "\n\n" + systemCtx.FormatForPrompt()
	}

	return rendered
}

// executeWithLLM lets LLM decide which tool to use and executes with multi-round support
func (s *Service) executeWithLLM(ctx context.Context, goal string, intent *IntentRecognitionResult, session *Session, memoryContext string, ragContext string) (interface{}, error) {
	const maxRounds = 20 // Maximum rounds to prevent infinite loops

	// Determine starting agent
	currentAgent := s.agent
	if session != nil && session.AgentID != "" && s.registry != nil {
		if a, ok := s.registry.GetAgent(session.AgentID); ok {
			currentAgent = a
		}
	}

	// Track tool calls to detect duplicates
	prevToolCalls := make(map[string]int)

	// Build initial user message
	messages := []domain.Message{
		{Role: "user", Content: goal},
	}

	// Add RAG context if available
	if ragContext != "" {
		messages[len(messages)-1].Content += "\n\n--- Relevant documents from knowledge base ---\n" + ragContext + "\n--- End of documents ---"
	}

	// Add memory context if available
	if memoryContext != "" {
		messages[len(messages)-1].Content += "\n\nRelevant context from memory:\n" + memoryContext
	}

	// Multi-round conversation loop
	for round := 0; round < maxRounds; round++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution cancelled by user")
		default:
		}

		s.emitProgress("thinking", fmt.Sprintf("[%s] Thinking...", currentAgent.Name()), round+1, "")

		// Collect tools for current agent
		tools := s.collectAllAvailableTools(ctx, currentAgent)

		// Build system message for current agent
		systemMsg := s.buildSystemPrompt(currentAgent)

		// Prepare messages for this generation (System + History)
		genMessages := append([]domain.Message{{Role: "system", Content: systemMsg}}, messages...)

		// --- DEBUG: LOG FULL PROMPT ---
		if s.config.Debug {
			fmt.Println("\n" + strings.Repeat("=", 40))
			fmt.Printf("DEBUG: [ROUND %d] LLM FULL PROMPT\n", round+1)
			fmt.Println(strings.Repeat("-", 40))
			for _, m := range genMessages {
				fmt.Printf("[%s]:\n%s\n", strings.ToUpper(m.Role), m.Content)
				if len(m.ToolCalls) > 0 {
					fmt.Printf("  (ToolCalls: %d)\n", len(m.ToolCalls))
				}
			}
			fmt.Println(strings.Repeat("=", 40) + "\n")
		}

		// Let LLM decide
		result, err := s.llmService.GenerateWithTools(ctx, genMessages, tools, &domain.GenerationOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}
		if result == nil {
			return nil, fmt.Errorf("LLM generation returned nil result")
		}

		// --- DEBUG: LOG RAW RESPONSE ---
		if s.config.Debug && err == nil {
			fmt.Println("\n" + strings.Repeat("=", 40))
			fmt.Printf("DEBUG: [ROUND %d] LLM RAW RESPONSE\n", round+1)
			fmt.Println(strings.Repeat("-", 40))
			if result.ReasoningContent != "" {
				fmt.Printf("REASONING: %s\n", result.ReasoningContent)
			}
			fmt.Printf("CONTENT: %s\n", result.Content)
			if len(result.ToolCalls) > 0 {
				fmt.Println("TOOL CALLS:")
				for _, tc := range result.ToolCalls {
					fmt.Printf("  - %s(%v)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
			fmt.Println(strings.Repeat("=", 40) + "\n")
		}

		// If LLM made tool calls, execute them and continue the conversation
		if len(result.ToolCalls) > 0 {

			// Check for Handoffs FIRST
			handoffOccurred := false
			for _, tc := range result.ToolCalls {
				if strings.HasPrefix(tc.Function.Name, "transfer_to_") {
					// Find target agent in current agent's handoffs
					for _, h := range currentAgent.Handoffs() {
						if h.ToolName() == tc.Function.Name {
							// Perform Handoff
							targetAgent := h.TargetAgent()
							reason := tc.Function.Arguments["reason"]

							s.emitProgress("tool_call", fmt.Sprintf("Transferring to %s", targetAgent.Name()), round+1, "handoff")

							// Update state
							currentAgent = targetAgent
							if session != nil {
								session.AgentID = targetAgent.ID()
							}

							// Add transfer message to history
							messages = append(messages, domain.Message{
								Role:             "assistant",
								Content:          result.Content,
								ReasoningContent: result.ReasoningContent,
								ToolCalls:        result.ToolCalls,
							})
							messages = append(messages, domain.Message{
								Role:       "tool",
								ToolCallID: tc.ID,
								Content:    fmt.Sprintf("Transferred to %s. Reason: %v", targetAgent.Name(), reason),
							})

							handoffOccurred = true
							break
						}
					}
				}
				if handoffOccurred {
					break
				}
			}

			if handoffOccurred {
				continue // Start next round with new agent
			}

			// Check for duplicate tool calls (same tool, same arguments)
			for _, tc := range result.ToolCalls {
				// Create a simple key for the tool call
				callKey := fmt.Sprintf("%s:%v", tc.Function.Name, tc.Function.Arguments)
				prevToolCalls[callKey]++
				if prevToolCalls[callKey] > 1 {
					// Duplicate call detected - force stop
					log.Printf("[Agent] Duplicate tool call detected: %s, stopping", callKey)
					return "The task has been completed. The information has been saved to memory.", nil
				}
			}

			// Execute tools and collect results
			s.emitProgress("tool_call", fmt.Sprintf("Calling %d tool(s)", len(result.ToolCalls)), round+1, "")
			toolResults, err := s.executeToolCalls(ctx, currentAgent, result.ToolCalls)
			if err != nil {
				// Add error as assistant message and continue
				messages = append(messages, domain.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Tool execution failed: %v", err),
				})
				continue
			}

			// Add assistant's response (tool calls) to conversation history
			messages = append(messages, domain.Message{
				Role:             "assistant",
				Content:          result.Content,
				ReasoningContent: result.ReasoningContent,
				ToolCalls:        result.ToolCalls,
			})

			// Add tool results using Role: tool
			for _, tr := range toolResults {
				resStr := fmt.Sprintf("%v", tr.Result)
				if s, ok := tr.Result.(string); ok {
					resStr = s
				}
				messages = append(messages, domain.Message{
					Role:       "tool",
					Content:    resStr,
					ToolCallID: tr.ToolCallID,
				})
			}

			continue
		}

		// No more tool calls - LLM is done
		return result.Content, nil
	}

	return nil, fmt.Errorf("exceeded maximum rounds (%d) - conversation may be stuck", maxRounds)
}

// verifyResult verifies the result with LLM
// Returns: (verified bool, reason string, correctedResult interface{}, err error)
func (s *Service) verifyResult(ctx context.Context, goal string, result interface{}) (bool, string, interface{}, error) {
	resultStr := formatResultForContent(result)

	data := map[string]interface{}{
		"Goal":   goal,
		"Result": resultStr,
	}

	rendered, err := s.promptManager.Render(prompt.AgentVerification, data)
	if err != nil {
		return true, "Render failed, assume verified", result, nil
	}

	verifyResp, err := s.llmService.Generate(ctx, rendered, &domain.GenerationOptions{
		Temperature: 0.1,
		MaxTokens:   300,
	})
	if err != nil {
		return true, "", result, nil // Return original on error, assume verified
	}

	// Try to parse as JSON verification
	var verifyRespJSON struct {
		Verified   bool   `json:"verified"`
		Reason     string `json:"reason"`
		NeedsRetry bool   `json:"needs_retry"`
	}

	// Simple JSON extraction
	if err := extractJSON(verifyResp, &verifyRespJSON); err == nil {
		if verifyRespJSON.Verified {
			return true, "Verified", result, nil
		}
		return false, verifyRespJSON.Reason, nil, fmt.Errorf("verification failed: %s", verifyRespJSON.Reason)
	}

	// If parsing failed, assume verified to avoid infinite loops
	return true, "Parse OK, assume verified", result, nil
}

// extractJSON extracts JSON from LLM response (handles markdown code blocks)
func extractJSON(resp string, target interface{}) error {
	// Try direct parse first
	if err := json.Unmarshal([]byte(resp), target); err == nil {
		return nil
	}

	// Try to find JSON in markdown code blocks
	if strings.Contains(resp, "```json") {
		start := strings.Index(resp, "```json")
		if start >= 0 {
			jsonStart := start + 7
			end := strings.Index(resp[jsonStart:], "```")
			if end >= 0 {
				jsonStr := strings.TrimSpace(resp[jsonStart : jsonStart+end])
				return json.Unmarshal([]byte(jsonStr), target)
			}
		}
	}

	// Try to find plain code block
	if strings.Contains(resp, "```") {
		start := strings.Index(resp, "```")
		if start >= 0 {
			jsonStart := start + 3
			end := strings.Index(resp[jsonStart:], "```")
			if end >= 0 {
				jsonStr := strings.TrimSpace(resp[jsonStart : jsonStart+end])
				return json.Unmarshal([]byte(jsonStr), target)
			}
		}
	}

	return fmt.Errorf("no JSON found in response")
}

// executeToolCalls executes the tool calls decided by LLM and returns all results
// executeToolCalls executes the tool calls decided by LLM and returns all results
func (s *Service) executeToolCalls(ctx context.Context, currentAgent *Agent, toolCalls []domain.ToolCall) ([]ToolExecutionResult, error) {
	results := make([]ToolExecutionResult, len(toolCalls))
	
	// Create an errgroup to run tools in parallel
	g, groupCtx := errgroup.WithContext(ctx)

	for i, tc := range toolCalls {
		// Capture index and tool call for the goroutine
		idx, toolCall := i, tc

		g.Go(func() error {
			// Format tool name for display
			toolName := toolCall.Function.Name
			toolDesc := toolName
			if strings.HasPrefix(toolName, "mcp_") {
				toolDesc = strings.TrimPrefix(toolName, "mcp_")
			}

			s.emitProgress("tool_call", fmt.Sprintf("→ %s", toolDesc), 0, toolName)

			// --- DEBUG: LOG TOOL CALL ---
			if s.config != nil && s.config.Debug {
				fmt.Printf("\n🛠️  DEBUG TOOL CALL: %s\n", toolName)
				fmt.Printf("   Arguments: %v\n", toolCall.Function.Arguments)
			}

			s.logger.Info("Executing Tool",
				slog.String("tool", toolName),
				slog.Any("arguments", toolCall.Function.Arguments))

			var result interface{}
			var err error
			var toolType string

					// 0. Priority: Agent-local Tools
					if handler, ok := currentAgent.GetHandler(toolName); ok {
						if s.config != nil && s.config.Debug { fmt.Println("   Type: Local Handler") }
						result, err = handler(groupCtx, toolCall.Function.Arguments)
						toolType = "local"
					} else if s.isMCPTool(toolName) {
						// 1. MCP tools
						if s.config != nil && s.config.Debug { fmt.Printf("   Type: MCP Tool\n") }
						result, err = s.mcpService.CallTool(groupCtx, toolName, toolCall.Function.Arguments)
						toolType = "mcp"
					} else if s.isSkill(groupCtx, toolName) && s.skillsService != nil {
						// 2. Skills
						if s.config != nil && s.config.Debug { fmt.Printf("   Type: Skill (%s)\n", toolName) }
			
				skillResult, skillErr := s.skillsService.Execute(groupCtx, &skills.ExecutionRequest{
					SkillID:     toolName,
					Variables:   toolCall.Function.Arguments,
					Interactive: false,
				})
				if skillErr == nil {
					result = skillResult.Output
					err = skillErr
				}
				toolType = "skill"
			} else if toolName == "rag_query" && s.ragProcessor != nil {
				query, _ := toolCall.Function.Arguments["query"].(string)
				resp, ragErr := s.ragProcessor.Query(groupCtx, domain.QueryRequest{Query: query})
				if ragErr == nil {
					result = resp.Answer
					// Collect sources for final result
					if len(resp.Sources) > 0 {
						s.ragSourcesMu.Lock()
						s.ragSources = append(s.ragSources, resp.Sources...)
						s.ragSourcesMu.Unlock()
					}
				}
				err = ragErr
				toolType = "rag"
			} else if toolName == "rag_ingest" && s.ragProcessor != nil {
				content, _ := toolCall.Function.Arguments["content"].(string)
				filePath, _ := toolCall.Function.Arguments["file_path"].(string)
				_, err = s.ragProcessor.Ingest(groupCtx, domain.IngestRequest{
					Content:  content,
					FilePath: filePath,
				})
				if err == nil {
					result = "Successfully ingested document"
				}
				toolType = "rag"
			} else if toolName == "memory_save" && s.memoryService != nil {
				s.markRunMemorySaved()
				content, _ := toolCall.Function.Arguments["content"].(string)
				memType := "preference"
				if t, ok := toolCall.Function.Arguments["type"].(string); ok {
					memType = t
				}
				err = s.memoryService.Add(groupCtx, &domain.Memory{
					Type:       domain.MemoryType(memType),
					Content:    content,
					Importance: 0.8,
					Metadata: map[string]interface{}{
						"source": "tool_call",
					},
				})
				if err == nil {
					result = fmt.Sprintf("Saved to memory: %s", content)
				}
				toolType = "memory"
			} else if toolName == "memory_update" && s.memoryService != nil {
				id, _ := toolCall.Function.Arguments["id"].(string)
				content, _ := toolCall.Function.Arguments["content"].(string)
				err = s.memoryService.Update(groupCtx, id, content)
				if err == nil {
					result = fmt.Sprintf("Memory %s updated successfully.", id)
				}
				toolType = "memory"
			} else if toolName == "memory_delete" && s.memoryService != nil {
				id, _ := toolCall.Function.Arguments["id"].(string)
				err = s.memoryService.Delete(groupCtx, id)
				if err == nil {
					result = fmt.Sprintf("Memory %s deleted successfully.", id)
				}
				toolType = "memory"
			} else if toolName == "memory_recall" && s.memoryService != nil {
				query, _ := toolCall.Function.Arguments["query"].(string)
				memories, memErr := s.memoryService.Search(groupCtx, query, 5)
				if memErr == nil {
					if len(memories) == 0 {
						// Fallback: list all recent memories
						allMems, _, listErr := s.memoryService.List(groupCtx, 10, 0)
						if listErr == nil && len(allMems) > 0 {
							var memResults []string
							for _, m := range allMems {
								memResults = append(memResults, fmt.Sprintf("- [%s] %s", m.Type, m.Content))
							}
							result = fmt.Sprintf("Recent memories:\n%s", strings.Join(memResults, "\n"))
						} else {
							result = "No relevant memories found"
						}
					} else {
						var memResults []string
						for _, m := range memories {
							memResults = append(memResults, fmt.Sprintf("- [%s: %.2f] %s", m.Type, m.Score, m.Content))
						}
						result = fmt.Sprintf("Found %d memories:\n%s", len(memories), strings.Join(memResults, "\n"))
					}
				}
				err = memErr
				toolType = "memory"
			} else {
				err = fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
			}

			if err != nil {
				s.logger.Error("Tool execution failed",
					slog.String("tool", toolName),
					slog.Any("error", err))
				
				if s.config != nil && s.config.Debug {
					fmt.Printf("   ❌ ERROR: %v\n", err)
					fmt.Println(strings.Repeat("-", 20))
				}
				
				return fmt.Errorf("Tool %s (%s) failed: %w", toolCall.Function.Name, toolType, err)
			}

			s.logger.Info("Tool Result",
				slog.String("tool", toolName),
				slog.Any("result", result))

			// --- DEBUG: LOG TOOL SUCCESS ---
			if s.config != nil && s.config.Debug {
				fmt.Printf("   ✅ RESULT: %v\n", result)
				fmt.Println(strings.Repeat("-", 20))
			}

			// Emit tool result progress
			s.emitProgress("tool_result", fmt.Sprintf("✓ %s Done", toolDesc), 0, toolName)

			results[idx] = ToolExecutionResult{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Function.Name,
				ToolType:   toolType,
				Result:     result,
			}
			return nil
		})
	}

	// Wait for all tools to finish
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// ToolExecutionResult represents the result of a single tool execution
type ToolExecutionResult struct {
	ToolCallID string      `json:"tool_call_id"`
	ToolName   string      `json:"tool_name"`
	ToolType   string      `json:"tool_type"`
	Result     interface{} `json:"result"`
}

// formatToolResults formats tool execution results for LLM consumption
func (s *Service) formatToolResults(results []ToolExecutionResult) string {
	var sb strings.Builder

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("Tool %d: %s (%s)\n", i+1, r.ToolName, r.ToolType))

		// Format result based on type
		switch v := r.Result.(type) {
		case string:
			if len(v) > 5000 {
				sb.WriteString(fmt.Sprintf("Result: %s...\n", v[:5000]))
			} else {
				sb.WriteString(fmt.Sprintf("Result: %s\n", v))
			}
		case []interface{}:
			// Handle array results (e.g., search results)
			for j, item := range v {
				sb.WriteString(fmt.Sprintf("  [%d] %v\n", j+1, item))
			}
		default:
			sb.WriteString(fmt.Sprintf("Result: %v\n", r.Result))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Service) resetRunMemorySaved() {
	s.memorySaveMu.Lock()
	s.memorySavedInRun = false
	s.memorySaveMu.Unlock()
}

func (s *Service) markRunMemorySaved() {
	s.memorySaveMu.Lock()
	s.memorySavedInRun = true
	s.memorySaveMu.Unlock()
}

func (s *Service) hasRunMemorySaved() bool {
	s.memorySaveMu.RLock()
	defer s.memorySaveMu.RUnlock()
	return s.memorySavedInRun
}

// containsStr checks if a string slice contains a string
func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// isMCPTool checks if a tool name is from MCP
func (s *Service) isMCPTool(name string) bool {
	if s.mcpService == nil {
		return false
	}
	for _, tool := range s.mcpService.ListTools() {
		if tool.Function.Name == name {
			return true
		}
	}
	return false
}

// isSkill checks if a tool name is a skill
func (s *Service) isSkill(ctx context.Context, name string) bool {
	if s.skillsService == nil {
		return false
	}
	skills, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
	for _, sk := range skills {
		if sk.ID == name {
			return true
		}
	}
	return false
}

// recognizeIntent performs intent recognition using planner
func (s *Service) recognizeIntent(ctx context.Context, goal string, session *Session) (*IntentRecognitionResult, error) {
	return s.planner.RecognizeIntent(ctx, goal, session)
}

// shouldUseRAG determines if RAG should be used based on intent
func (s *Service) shouldUseRAG(intent *IntentRecognitionResult) bool {
	// Use RAG for query, analysis, and general_qa intents
	return intent.IntentType == "rag_query" ||
		intent.IntentType == "analysis" ||
		intent.IntentType == "general_qa" ||
		intent.IntentType == "question"
}

// shouldUseSkills determines if skills should be used based on intent
func (s *Service) shouldUseSkills(intent *IntentRecognitionResult) bool {
	// Use skills for web_search, file operations, etc.
	return intent.IntentType == "web_search" ||
		intent.IntentType == "file_create" ||
		intent.IntentType == "file_read"
}

// buildEnrichedPrompt builds a prompt enriched with memory and RAG results
func (s *Service) buildEnrichedPrompt(goal, memoryContext, ragResult string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User Question: %s\n\n", goal))

	if memoryContext != "" {
		prompt.WriteString("--- Relevant Memory ---\n")
		prompt.WriteString(memoryContext)
		prompt.WriteString("\n\n")
	}

	if ragResult != "" {
		prompt.WriteString("--- Knowledge Base Results ---\n")
		prompt.WriteString(ragResult)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please answer the user's question based on the memory and knowledge base information above.")
	prompt.WriteString(" If there's no relevant information, say so honestly.")

	return prompt.String()
}

// executeSkills executes skills based on intent
func (s *Service) executeSkills(ctx context.Context, intent *IntentRecognitionResult, prompt string) (interface{}, error) {
	// Find relevant skill based on intent
	if s.skillsService == nil {
		return nil, fmt.Errorf("skills service not available")
	}

	// List available skills
	skillList, err := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
	if err != nil {
		return nil, err
	}

	// Map intents to skill keyword patterns
	intentSkillPatterns := map[string][]string{
		"web_search":  {"search", "web", "query", "rag"},
		"rag_query":   {"query", "rag", "search"},
		"file_create": {"create", "write", "file"},
		"file_read":   {"read", "file", "open"},
	}

	// Find patterns for this intent
	patterns, hasPatterns := intentSkillPatterns[intent.IntentType]
	if !hasPatterns {
		// No specific patterns, try any skill
		for _, sk := range skillList {
			req := &skills.ExecutionRequest{
				SkillID:     sk.ID,
				Variables:   map[string]interface{}{"query": intent.Topic},
				Interactive: false,
			}
			result, err := s.skillsService.Execute(ctx, req)
			if err == nil && result.Success {
				return result.Output, nil
			}
		}
	}

	// Try to find a matching skill
	for _, sk := range skillList {
		skillIDLower := strings.ToLower(sk.ID)
		for _, pattern := range patterns {
			if strings.Contains(skillIDLower, pattern) {
				req := &skills.ExecutionRequest{
					SkillID:     sk.ID,
					Variables:   map[string]interface{}{"query": intent.Topic},
					Interactive: false,
				}
				result, err := s.skillsService.Execute(ctx, req)
				if err == nil && result.Success {
					return result.Output, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no suitable skill found for intent: %s", intent.IntentType)
}

// RunWithSession executes a goal with an existing session ID
func (s *Service) RunWithSession(ctx context.Context, goal, sessionID string) (*ExecutionResult, error) {
	s.resetRunMemorySaved()

	// Load or create session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
	}

	// Add user message to session
	userMsg := domain.Message{
		Role:    "user",
		Content: goal,
	}
	session.AddMessage(userMsg)

	// Step 1: Retrieve memory context before planning
	if s.memoryService != nil {
		ragolog.Debug("[RunWithSession] Retrieving memory context for goal: %q, sessionID: %q", goal, sessionID)
		memoryContext, memoryMemories, _ := s.memoryService.RetrieveAndInject(ctx, goal, sessionID)
		ragolog.Debug("[RunWithSession] Memory context retrieved - context length: %d, memories: %d", len(memoryContext), len(memoryMemories))
		if memoryContext != "" {
			// Inject memory context into the session messages
			session.AddMessage(domain.Message{
				Role:    "system",
				Content: memoryContext,
			})
		}
	} else {
		ragolog.Debug("[RunWithSession] Memory service is nil, skipping memory retrieval")
	}

	// Generate plan
	plan, err := s.planner.PlanWithFallback(ctx, goal, session)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Save plan
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Execute plan
	result, err := s.executor.ExecutePlan(ctx, plan, session)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Add assistant response to session
	if result.FinalResult != nil {
		assistantContent := fmt.Sprintf("%v", result.FinalResult)
		assistantMsg := domain.Message{
			Role:    "assistant",
			Content: assistantContent,
		}
		session.AddMessage(assistantMsg)
	}

	// Save updated plan
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return result, nil
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(sessionID string) (*Session, error) {
	return s.store.GetSession(sessionID)
}

// GetPlan retrieves a plan by ID
func (s *Service) GetPlan(planID string) (*Plan, error) {
	return s.store.GetPlan(planID)
}

// ListSessions returns all sessions
func (s *Service) ListSessions(limit int) ([]*Session, error) {
	return s.store.ListSessions(limit)
}

// ListPlans returns plans for a session
func (s *Service) ListPlans(sessionID string, limit int) ([]*Plan, error) {
	return s.store.ListPlans(sessionID, limit)
}

// Chat sends a message with auto-generated session UUID.
// This is the simplest API for conversational AI with memory.
//
// Example:
//
//	svc, _ := agent.New(&agent.AgentConfig{Name: "assistant"})
//	result, _ := svc.Chat(ctx, "My name is Alice")
//	result, _ = svc.Chat(ctx, "What's my name?") // Will remember "Alice"
func (s *Service) Chat(ctx context.Context, message string) (*ExecutionResult, error) {
	s.sessionMu.Lock()
	if s.currentSessionID == "" {
		s.currentSessionID = uuid.New().String()
	}
	sessionID := s.currentSessionID
	s.sessionMu.Unlock()

	return s.RunWithSession(ctx, message, sessionID)
}

// CurrentSessionID returns the current session UUID used by Chat()
func (s *Service) CurrentSessionID() string {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.currentSessionID
}

// SetSessionID sets a specific session ID for Chat() to use
func (s *Service) SetSessionID(sessionID string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.currentSessionID = sessionID
}

// ResetSession clears the current session and starts a new one with a new UUID
func (s *Service) ResetSession() {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.currentSessionID = uuid.New().String()
}

// ConfigureMemory sets the memory bank personality for the current session
func (s *Service) ConfigureMemory(ctx context.Context, config *domain.MemoryBankConfig) error {
	if s.memoryService == nil {
		return fmt.Errorf("memory service not enabled")
	}
	return s.memoryService.ConfigureBank(ctx, s.CurrentSessionID(), config)
}

// ReflectMemory triggers memory consolidation and returns current system observations
func (s *Service) ReflectMemory(ctx context.Context) (string, error) {
	if s.memoryService == nil {
		return "", fmt.Errorf("memory service not enabled")
	}
	return s.memoryService.Reflect(ctx, s.CurrentSessionID())
}

// CompactSession summarizes a session into key points
func (s *Service) CompactSession(ctx context.Context, sessionID string) (string, error) {
	// Load session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
	}

	messages := session.GetMessages()
	if len(messages) == 0 {
		return "", nil
	}

	// Check if llmService supports Compact
	llmSvc, ok := s.llmService.(interface {
		Compact(ctx context.Context, messages []domain.Message) (string, error)
	})
	if !ok {
		return "", fmt.Errorf("underlying LLM service does not support Compact")
	}

	// Generate summary
	summary, err := llmSvc.Compact(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to compact session: %w", err)
	}

	// Update session
	session.SetSummary(summary)
	if err := s.store.SaveSession(session); err != nil {
		return "", fmt.Errorf("failed to save session summary: %w", err)
	}

	return summary, nil
}

// Execute executes a plan by ID and returns the result
func (s *Service) Execute(ctx context.Context, planID string) (*ExecutionResult, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	return s.ExecutePlan(ctx, plan)
}

// RunRealtime starts a bidirectional realtime session with the agent's capabilities.
func (s *Service) RunRealtime(ctx context.Context, opts *domain.GenerationOptions) (domain.RealtimeSession, error) {
	// 1. Check if provider supports realtime
	realtimeGen, ok := s.llmService.(domain.RealtimeGenerator)
	if !ok {
		return nil, fmt.Errorf("current LLM provider does not support realtime interactions")
	}

	// 2. Collect tools for the current agent
	tools := s.collectAllAvailableTools(ctx, s.agent)

	// 3. Create session
	session, err := realtimeGen.NewSession(ctx, tools, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create realtime session: %w", err)
	}

	s.logger.Info("Realtime session started", slog.Int("tools_count", len(tools)))
	return session, nil
}

// SaveToFile saves content to a file
func (s *Service) SaveToFile(content, filePath string) error {
	// Create directory if needed
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("[Agent] ✅ Saved to %s\n", filePath)
	return nil
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	return s.store.Close()
}

// ========================================
// Simplified API
// ========================================

// AgentConfig holds configuration for New()
type AgentConfig struct {
	Name            string
	SystemPrompt    string
	DBPath          string
	MemoryDBPath    string
	MemoryStoreType string // "sqlite" (default) or "file"
	EnableMCP       bool
	EnableMemory    bool
	EnableRAG       bool
	EnableRouter    bool
	EnableSkills    bool
	EnableAutoMemory bool // Automatically save important info to memory
	Debug           bool // Enable verbose debugging output
	IntentPaths     []string
	RouterThreshold float64
	ProgressCb      ProgressCallback
}

// New creates an agent service with simplified configuration.
// It automatically initializes LLM, Embedding, MCP, Memory, and Router services from rago.toml.
//
// Example:
//
//	svc, err := agent.New(&agent.AgentConfig{
//	    Name: "my-agent",
//	    EnableMCP: true,
//	    EnableMemory: true,
//	})
//	result, _ := svc.Run(ctx, "Hello!")
func New(cfg *AgentConfig) (*Service, error) {
	if cfg == nil || cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	// Load config
	ragoCfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize global pool
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(context.Background(), ragoCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize pool: %w", err)
	}

	// Get LLM service
	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	// Get Embedding service
	embedSvc, err := globalPool.GetEmbeddingService(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service
	var mcpSvc *mcp.Service
	var mcpAdapter MCPToolExecutor
	if cfg.EnableMCP {
		mcpSvc, err = mcp.NewService(&ragoCfg.MCP, llmSvc)
		if err == nil {
			mcpSvc.StartServers(context.Background(), nil)
			mcpAdapter = &mcpToolAdapter{service: mcpSvc}
		}
	}

	// Memory service
	var memSvc domain.MemoryService
	if cfg.EnableMemory {
		var memStore domain.MemoryStore
		var shadowStore domain.MemoryStore // New: optional shadow index
		var err error

		if cfg.MemoryStoreType == "file" {
			memPath := cfg.MemoryDBPath
			if memPath == "" {
				memPath = filepath.Join(ragoCfg.DataDir(), "memories")
			}
			memStore, err = store.NewFileMemoryStore(memPath)
			
			// Initialize Shadow Index (Vector accelerator)
			sqlitePath := filepath.Join(ragoCfg.DataDir(), "rago.db")
			if sqliteStore, serr := store.NewMemoryStore(sqlitePath); serr == nil {
				_ = sqliteStore.InitSchema(context.Background())
				shadowStore = sqliteStore
			}
		} else {
			// (Old logic for direct SQLite storage...)
			memDBPath := cfg.MemoryDBPath
			if memDBPath == "" {
				memDBPath = ragoCfg.Sqvect.DBPath
			}
			sqliteStore, serr := store.NewMemoryStore(memDBPath)
			if serr == nil {
				_ = sqliteStore.InitSchema(context.Background())
				memStore = sqliteStore
			}
			err = serr
		}

		if err == nil && memStore != nil {
			memSvcInstance := memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
			// Inject shadow index if available
			if shadowStore != nil {
				memSvcInstance.SetShadowIndex(shadowStore)
			}
			memSvc = memSvcInstance
		}
	}

	// Router service
	var routerSvc *router.Service
	if cfg.EnableRouter {
		routerCfg := router.DefaultConfig()
		if cfg.RouterThreshold > 0 {
			routerCfg.Threshold = cfg.RouterThreshold
		}
		routerSvc, err = router.NewService(embedSvc, routerCfg)
		if err == nil {
			_ = routerSvc.RegisterDefaultIntents()
		}
	}

	// RAG processor
	var ragProcessor domain.Processor
	if cfg.EnableRAG {
		// Create RAG stores using the unified DB path
		vectorStore, err := ragstore.NewVectorStore(ragstore.StoreConfig{
			Type: "sqlite",
			Parameters: map[string]interface{}{
				"db_path": ragoCfg.Sqvect.DBPath,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create vector store: %w", err)
		}
		if vectorStore != nil {
			docStore := ragstore.NewDocumentStoreFor(vectorStore)
			// Create RAG processor with proper stores
			ragProcessor = ragprocessor.New(
				embedSvc,
				llmSvc,
				nil,         // chunker - will use default
				vectorStore, // vector store
				docStore,    // document store
				ragoCfg,
				nil,    // llmService for metadata extraction (optional)
				memSvc, // memory service
			)
		}
	}

	// Skills service
	var skillsSvc *skills.Service
	if cfg.EnableSkills {
		skillsCfg := skills.DefaultConfig()
		skillsCfg.Paths = []string{ragoCfg.SkillsDir()}
		// Use the unified DB path
		skillsCfg.DBPath = ragoCfg.Sqvect.DBPath
		skillsSvc, err = skills.NewService(skillsCfg)
		if err == nil {
			_ = skillsSvc.LoadAll(context.Background())
		}
	}

	// Agent DB path - use the unified DB path
	agentDBPath := cfg.DBPath
	if agentDBPath == "" {
		agentDBPath = ragoCfg.Sqvect.DBPath
	}

	// Create agent service
	svc, err := NewService(llmSvc, mcpAdapter, ragProcessor, agentDBPath, memSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	svc.config = cfg

	// Load prompts from default directory (~/.rago/prompts)
	promptDir := filepath.Join(ragoCfg.Home, "prompts")
	_ = svc.promptManager.LoadFromDir(promptDir)

	// Set skills service
	if skillsSvc != nil {
		svc.SetSkillsService(skillsSvc)
	}

	// Set router
	if routerSvc != nil {
		svc.SetRouter(routerSvc)

		// Load custom intents if provided or from default path
		intentPaths := cfg.IntentPaths
		if len(intentPaths) == 0 {
			intentPaths = []string{ragoCfg.IntentsDir()}
		}
		_ = svc.Router.LoadIntentsFromPaths(intentPaths)
	}

	// Set progress callback
	if cfg.ProgressCb != nil {
		svc.SetProgressCallback(cfg.ProgressCb)
	}

	// Set custom system prompt if provided
	if cfg.SystemPrompt != "" {
		svc.SetAgentInstructions(cfg.SystemPrompt)
	}

	// Register auto-memory hook if enabled
	if cfg.EnableAutoMemory {
		svc.RegisterAutoMemoryHook()
	}

	return svc, nil
}

// mcpToolAdapter wraps mcp.Service to implement MCPToolExecutor
type mcpToolAdapter struct {
	service *mcp.Service
}

func (a *mcpToolAdapter) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	result, err := a.service.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("MCP tool error: %s", result.Error)
	}
	return result.Data, nil
}

func (a *mcpToolAdapter) ListTools() []domain.ToolDefinition {
	tools := a.service.GetAvailableTools(context.Background())
	result := make([]domain.ToolDefinition, 0, len(tools))

	for _, t := range tools {
		params := t.InputSchema
		if params == nil {
			params = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"args": map[string]interface{}{
						"description": "arguments",
						"type":        "object",
					},
				},
			}
		}
		result = append(result, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return result
}

func (a *mcpToolAdapter) AddServer(ctx context.Context, name string, command string, args []string) error {
	return a.service.AddDynamicServer(ctx, name, command, args)
}

// AddMCPServer dynamically adds and starts an MCP server
func (s *Service) AddMCPServer(ctx context.Context, name string, command string, args []string) error {
	if s.mcpService == nil {
		return fmt.Errorf("MCP service not initialized")
	}
	return s.mcpService.AddServer(ctx, name, command, args)
}

// collectAvailableTools collects tools from all available sources
func collectAvailableTools(mcpService MCPToolExecutor, ragProcessor domain.Processor, skillsService *skills.Service) []domain.ToolDefinition {
	tools := []domain.ToolDefinition{}

	// Add RAG tools
	if ragProcessor != nil {
		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_query",
				Description: "Query the RAG system to retrieve relevant document chunks",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query",
						},
						"top_k": map[string]interface{}{
							"type":        "integer",
							"description": "Number of results to return",
							"default":     5,
						},
					},
					"required": []string{"query"},
				},
			},
		})

		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_ingest",
				Description: "Ingest a document into the RAG system",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The document content",
						},
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the document file",
						},
					},
				},
			},
		})
	}

	// Add Skills tools
	if skillsService != nil {
		skillTools, err := skillsService.RegisterAsMCPTools()
		if err == nil {
			tools = append(tools, skillTools...)
		}
	}

	// Add MCP tools
	if mcpService != nil {
		mcpTools := mcpService.ListTools()
		tools = append(tools, mcpTools...)
	}

	// Add general LLM tool
	tools = append(tools, domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "llm",
			Description: "General LLM reasoning and text generation",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The prompt for the LLM",
					},
					"temperature": map[string]interface{}{
						"type":        "number",
						"description": "Temperature for generation (0-1)",
						"default":     0.7,
					},
				},
				"required": []string{"prompt"},
			},
		},
	})

	return tools
}

// executeWithDynamicToolSelection uses LLM's native function calling to decide which MCP tools to use
func (s *Service) executeWithDynamicToolSelection(ctx context.Context, goal string, intent *IntentRecognitionResult, availableTools []domain.ToolDefinition, memoryContext, ragResult string) (interface{}, error) {
	systemPrompt, err := s.promptManager.Render(prompt.AgentDynamicToolSelection, nil)
	if err != nil {
		systemPrompt = "You are a helpful assistant with access to tools. Use tools when appropriate to help the user."
	}

	// Build messages - let LLM decide which tools to call
	messages := []domain.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: goal},
	}

	// Add context if available
	if memoryContext != "" || ragResult != "" {
		contextMsg := "\n\nRelevant context:\n"
		if memoryContext != "" {
			contextMsg += memoryContext + "\n"
		}
		if ragResult != "" {
			contextMsg += ragResult + "\n"
		}
		messages[len(messages)-1].Content += contextMsg
	}

	// Use GenerateWithTools - let LLM natively decide which tools to call
	result, err := s.llmService.GenerateWithTools(ctx, messages, availableTools, &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   1000,
	})

	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// If LLM made tool calls, execute them
	if len(result.ToolCalls) > 0 {
		return s.executeLLMToolCalls(ctx, result.ToolCalls, goal, memoryContext, ragResult)
	}

	// No tool calls needed, return the text response
	return result.Content, nil
}

// executeLLMToolCalls executes tool calls decided by LLM
func (s *Service) executeLLMToolCalls(ctx context.Context, toolCalls []domain.ToolCall, goal, memoryContext, ragResult string) (interface{}, error) {
	var results []interface{}

	for _, tc := range toolCalls {
		log.Printf("[Agent] Calling tool: %s", tc.Function.Name)

		// Handle memory tools
		if tc.Function.Name == "memory_save" {
			s.markRunMemorySaved()
			content, _ := tc.Function.Arguments["content"].(string)
			memType := "preference"
			if t, ok := tc.Function.Arguments["type"].(string); ok {
				memType = t
			}
			err := s.memoryService.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
				Metadata: map[string]interface{}{
					"source": "tool_call",
				},
			})
			if err != nil {
				results = append(results, fmt.Sprintf("Failed to save memory: %v", err))
			} else {
				results = append(results, fmt.Sprintf("Saved to memory: %s", content))
			}
			continue
		}

		if tc.Function.Name == "memory_recall" {
			query, _ := tc.Function.Arguments["query"].(string)
			memories, err := s.memoryService.Search(ctx, query, 5)
			if err != nil {
				results = append(results, fmt.Sprintf("Memory search failed: %v", err))
			} else if len(memories) == 0 {
				results = append(results, "No relevant memories found")
			} else {
				var memResults []string
				for _, m := range memories {
					memResults = append(memResults, fmt.Sprintf("- [%s: %.2f] %s", m.Type, m.Score, m.Content))
				}
				results = append(results, fmt.Sprintf("Found %d memories:\n%s", len(memories), strings.Join(memResults, "\n")))
			}
			continue
		}

		// Handle MCP tools
		result, err := s.mcpService.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("tool call failed: %w", err)
		}
		results = append(results, result)
	}

	// If results were obtained, format them
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// finalizeExecution finalizes the execution result
func (s *Service) finalizeExecution(ctx context.Context, session *Session, goal string, intent *IntentRecognitionResult, memoryMemories []*domain.MemoryWithScore, ragResult string, finalResult interface{}) (*ExecutionResult, error) {
	// Store to memory after completion
	if s.memoryService != nil {
		// Auto-store for explicit memory request patterns
		goalLower := strings.ToLower(goal)
		if strings.HasPrefix(goalLower, "remember:") ||
			strings.HasPrefix(goalLower, "save to memory") ||
			strings.HasPrefix(goalLower, "my favorite") ||
			strings.HasPrefix(goalLower, "i prefer") ||
			strings.Contains(goalLower, "preference is") {

			if !s.hasRunMemorySaved() {
				// Direct storage for explicit memory requests
				content := goal
				if strings.HasPrefix(goalLower, "remember:") {
					content = strings.TrimSpace(goal[len("remember:"):])
				} else if strings.HasPrefix(goalLower, "save to memory") {
					content = strings.TrimSpace(goal[len("save to memory"):])
				}

				_ = s.memoryService.Add(ctx, &domain.Memory{
					Type:       domain.MemoryTypePreference,
					Content:    content,
					Importance: 0.8,
					Metadata: map[string]interface{}{
						"source": "user_direct",
					},
				})
				log.Printf("[Agent] Stored to memory: %s", content)
			}
		}

		// LLM-based extraction for complex memories
		_ = s.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:  session.GetID(),
			TaskGoal:   goal,
			TaskResult: formatResultForContent(finalResult),
			ExecutionLog: fmt.Sprintf("Intent: %s\nMemory: %d items\nRAG: %d chars",
				intent.IntentType, len(memoryMemories), len(ragResult)),
		})
	}

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	res := &ExecutionResult{
		PlanID:      uuid.New().String(),
		SessionID:   session.GetID(),
		Success:     true,
		StepsTotal:  1,
		StepsDone:   1,
		StepsFailed: 0,
		FinalResult: finalResult,
		Duration:    "completed",
	}

	// Collect RAG sources
	s.ragSourcesMu.RLock()
	if len(s.ragSources) > 0 {
		res.Sources = append([]domain.Chunk{}, s.ragSources...)
	}
	s.ragSourcesMu.RUnlock()

	// Clear sources for next run
	s.ragSourcesMu.Lock()
	s.ragSources = nil
	s.ragSourcesMu.Unlock()

	// Emit PostExecution Hook
	GlobalHookRegistry().Emit(HookEventPostExecution, HookData{
		SessionID: session.GetID(),
		AgentID:   session.AgentID,
		Goal:      goal,
		Result:    finalResult,
		Metadata: map[string]interface{}{
			"intent": intent.IntentType,
		},
	})

	return res, nil
}

// performRAGQuery performs a RAG query to get relevant documents
func (s *Service) performRAGQuery(ctx context.Context, query string) (string, error) {
	if s.ragProcessor == nil {
		return "", nil
	}

	// Use the RAG processor to query
	request := domain.QueryRequest{
		Query:        query,
		TopK:         5, // Get top 5 results
		Temperature:  0.3,
		ShowThinking: false,
		ShowSources:  true,
	}

	results, err := s.ragProcessor.Query(ctx, request)
	if err != nil {
		return "", err
	}

	// Format results as context
	if results.Answer == "" && len(results.Sources) == 0 {
		return "", nil
	}

	// Collect sources for final result
	if len(results.Sources) > 0 {
		s.ragSourcesMu.Lock()
		s.ragSources = append(s.ragSources, results.Sources...)
		s.ragSourcesMu.Unlock()
	}

	var context strings.Builder
	context.WriteString("## Relevant Documents\n\n")

	// Add answer if available
	if results.Answer != "" {
		context.WriteString(fmt.Sprintf("**Answer:** %s\n\n", results.Answer))
	}

	// Add sources
	for i, source := range results.Sources {
		context.WriteString(fmt.Sprintf("### Document %d\n", i+1))
		if source.DocumentID != "" {
			context.WriteString(fmt.Sprintf("**Source:** %s\n", source.DocumentID))
		}
		if source.Score > 0 {
			context.WriteString(fmt.Sprintf("**Score:** %.2f\n", source.Score))
		}
		if source.Content != "" {
			context.WriteString(fmt.Sprintf("**Content:** %s\n", source.Content))
		}
		context.WriteString("\n---\n\n")
	}

	return context.String(), nil
}

// countDocuments counts the number of documents in RAG context
func countDocuments(ragContext string) int {
	if ragContext == "" {
		return 0
	}
	// Count "### Document" occurrences
	count := strings.Count(ragContext, "### Document")
	return count
}
