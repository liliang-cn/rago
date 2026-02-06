package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Agent represents an autonomous agent with instructions and tools
// Inspired by OpenAI Agents SDK: Agent = LLM + instructions + tools
type Agent struct {
	id           string
	name         string
	instructions string
	tools        []domain.ToolDefinition
	model        string
	temperature  float64
}

// NewAgent creates a new Agent with default settings
func NewAgent(name string) *Agent {
	return &Agent{
		id:           uuid.New().String(),
		name:         name,
		instructions: "You are a helpful assistant.",
		tools:        []domain.ToolDefinition{},
		model:        "",
		temperature:  0.7,
	}
}

// NewAgentWithConfig creates a new Agent with custom configuration
func NewAgentWithConfig(name, instructions string, tools []domain.ToolDefinition) *Agent {
	return &Agent{
		id:           uuid.New().String(),
		name:         name,
		instructions: instructions,
		tools:        tools,
		model:        "",
		temperature:  0.7,
	}
}

// ID returns the agent's unique ID
func (a *Agent) ID() string {
	return a.id
}

// Name returns the agent's name
func (a *Agent) Name() string {
	return a.name
}

// Instructions returns the agent's instructions
func (a *Agent) Instructions() string {
	return a.instructions
}

// Tools returns the agent's available tools
func (a *Agent) Tools() []domain.ToolDefinition {
	return a.tools
}

// Model returns the model name
func (a *Agent) Model() string {
	return a.model
}

// Temperature returns the temperature setting
func (a *Agent) Temperature() float64 {
	return a.temperature
}

// SetInstructions sets the agent's instructions
func (a *Agent) SetInstructions(instructions string) {
	a.instructions = instructions
}

// SetTools sets the agent's available tools
func (a *Agent) SetTools(tools []domain.ToolDefinition) {
	a.tools = tools
}

// SetModel sets the model name
func (a *Agent) SetModel(model string) {
	a.model = model
}

// SetTemperature sets the temperature
func (a *Agent) SetTemperature(temp float64) {
	a.temperature = temp
}

// GetToolNames returns the names of available tools
func (a *Agent) GetToolNames() []string {
	names := make([]string, len(a.tools))
	for i, tool := range a.tools {
		names[i] = tool.Function.Name
	}
	return names
}

// HasTool checks if the agent has a specific tool
func (a *Agent) HasTool(toolName string) bool {
	for _, tool := range a.tools {
		if tool.Function.Name == toolName {
			return true
		}
	}
	return false
}

// AgentRunner runs an agent with a session
type AgentRunner struct {
	agent      *Agent
	planner    *Planner
	executor   *Executor
	sessionMgr *SessionManager
	store      *Store
}

// NewAgentRunner creates a new agent runner
func NewAgentRunner(
	agent *Agent,
	planner *Planner,
	executor *Executor,
	store *Store,
) *AgentRunner {
	return &AgentRunner{
		agent:      agent,
		planner:    planner,
		executor:   executor,
		sessionMgr: NewSessionManager(),
		store:      store,
	}
}

// Run runs the agent with a goal
func (r *AgentRunner) Run(ctx context.Context, goal string) (*ExecutionResult, error) {
	// Create or get session
	session := r.sessionMgr.CreateSession(r.agent.id)

	// Generate plan
	plan, err := r.planner.Plan(ctx, goal, session)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Save plan
	if err := r.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Execute plan
	result, err := r.executor.ExecutePlan(ctx, plan, session)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Save updated plan
	if err := r.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Save session
	if err := r.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return result, nil
}

// RunWithSession runs the agent with a goal and existing session
func (r *AgentRunner) RunWithSession(ctx context.Context, goal string, sessionID string) (*ExecutionResult, error) {
	// Get or create session
	session, ok := r.sessionMgr.GetSession(sessionID)
	if !ok {
		// Try to load from store
		loadedSession, err := r.store.GetSession(sessionID)
		if err != nil {
			// Create new session
			session = NewSessionWithID(sessionID, r.agent.id)
		} else {
			session = loadedSession
		}
		r.sessionMgr.sessions[sessionID] = session
	}

	// Generate plan
	plan, err := r.planner.Plan(ctx, goal, session)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Save plan
	if err := r.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Execute plan
	result, err := r.executor.ExecutePlan(ctx, plan, session)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Save updated plan
	if err := r.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Save session
	if err := r.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return result, nil
}
