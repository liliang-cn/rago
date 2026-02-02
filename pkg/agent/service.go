package agent

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Service is the main agent service that handles planning and execution
// This matches the interface expected by the CLI in cmd/rago-cli/agent/agent.go
type Service struct {
	llmService   domain.Generator
	mcpService   MCPToolExecutor
	ragProcessor domain.Processor
	planner      *Planner
	executor     *Executor
	store        *Store
	agent        *Agent
}

// NewService creates a new agent service
// This matches the signature expected by the CLI:
// agent.NewService(llmService, mcpService, processor, agentDBPath)
func NewService(
	llmService domain.Generator,
	mcpService MCPToolExecutor,
	ragProcessor domain.Processor,
	agentDBPath string,
) (*Service, error) {
	// Initialize store
	store, err := NewStore(agentDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent store: %w", err)
	}

	// Collect available tools
	tools := collectAvailableTools(mcpService, ragProcessor)

	// Create default agent
	agent := NewAgentWithConfig(
		"RAGO Agent",
		`You are RAGO, a helpful AI assistant with access to RAG (Retrieval Augmented Generation),
MCP tools, and various processing capabilities.

When given a goal:
1. Break it down into clear steps
2. Choose appropriate tools for each step
3. Execute steps in logical order
4. Provide clear results

Available tools include RAG queries, MCP tools, and general LLM reasoning.`,
		tools,
	)

	// Create planner
	planner := NewPlanner(llmService, tools)

	// Create executor
	executor := NewExecutor(llmService, nil, mcpService, ragProcessor)

	return &Service{
		llmService:   llmService,
		mcpService:   mcpService,
		ragProcessor: ragProcessor,
		planner:      planner,
		executor:     executor,
		store:        store,
		agent:        agent,
	}, nil
}

// Plan generates an execution plan for the given goal
// This matches the CLI expectation: agentService.Plan(ctx, goal)
func (s *Service) Plan(ctx context.Context, goal string) (*Plan, error) {
	session := NewSession(s.agent.ID())
	return s.planner.PlanWithFallback(ctx, goal, session)
}

// ExecutePlan executes the given plan
// This matches the CLI expectation: agentService.ExecutePlan(ctx, plan)
func (s *Service) ExecutePlan(ctx context.Context, plan *Plan) error {
	result, err := s.executor.ExecutePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Save the plan state
	if err := s.store.SavePlan(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("plan execution completed with errors: %s", result.Error)
	}

	return nil
}

// Run executes a goal from planning to completion
func (s *Service) Run(ctx context.Context, goal string) (*ExecutionResult, error) {
	// Create session
	session := NewSession(s.agent.ID())

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
	result, err := s.executor.ExecutePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
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

// RunWithSession executes a goal with an existing session ID
func (s *Service) RunWithSession(ctx context.Context, goal, sessionID string) (*ExecutionResult, error) {
	// Load or create session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
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
	result, err := s.executor.ExecutePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
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

// Close closes the service and releases resources
func (s *Service) Close() error {
	return s.store.Close()
}

// collectAvailableTools collects tools from all available sources
func collectAvailableTools(mcpService MCPToolExecutor, ragProcessor domain.Processor) []domain.ToolDefinition {
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
