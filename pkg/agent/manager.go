package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/pool"
	"github.com/liliang-cn/agent-go/pkg/services"
)

// AgentManager handles the lifecycle, discovery, and execution routing for dynamic agents.
type AgentManager struct {
	store         *Store
	runningAgents map[string]context.CancelFunc // Tracks running agents if they are background loopers
	services      map[string]*Service           // Cached instantiated agent services
	mu            sync.RWMutex
}

// SeedDefaultAgents seeds some default specialized agents if none exist.
func (m *AgentManager) SeedDefaultAgents() error {
	agents, err := m.store.ListAgentModels()
	if err != nil {
		return err
	}
	if len(agents) > 0 {
		return nil // Already seeded
	}

	defaults := []*AgentModel{
		{
			ID:           "agent-coder-001",
			Name:         "Coder",
			Description:  "An expert programmer. Can write, review, and refactor code.",
			Instructions: "You are an expert programmer. Provide clean, correct, and well-documented code.",
			Status:       AgentStatusStopped,
			EnablePTC:    true,
			EnableMCP:    true,
		},
		{
			ID:           "agent-researcher-001",
			Name:         "Researcher",
			Description:  "An expert researcher. Can search for information and summarize complex topics.",
			Instructions: "You are an expert researcher. Be thorough, factual, and cite your sources.",
			Status:       AgentStatusStopped,
			EnableRAG:    true,
			EnableMCP:    true,
		},
	}

	for _, a := range defaults {
		if err := m.store.SaveAgentModel(a); err != nil {
			return err
		}
	}
	return nil
}

// NewAgentManager creates a new multi-agent manager based on a store.
func NewAgentManager(s *Store) *AgentManager {
	return &AgentManager{
		store:         s,
		runningAgents: make(map[string]context.CancelFunc),
		services:      make(map[string]*Service),
	}
}

// StartAgent marks an agent as running in the database and prepares its service.
func (m *AgentManager) StartAgent(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", name, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Optionally start a background task queue poller here for async mode.
	// For now, we mark it as running and cache the builder.

	err = m.store.UpdateAgentStatus(model.ID, AgentStatusRunning)
	if err != nil {
		return err
	}
	model.Status = AgentStatusRunning

	return nil
}

// StopAgent marks an agent as stopped.
func (m *AgentManager) StopAgent(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, exists := m.runningAgents[name]; exists {
		cancel()
		delete(m.runningAgents, name)
	}
	delete(m.services, name)

	return m.store.UpdateAgentStatus(model.ID, AgentStatusStopped)
}

// ListRunningAgents returns a list of actively running agents.
func (m *AgentManager) ListRunningAgents() ([]*AgentModel, error) {
	all, err := m.store.ListAgentModels()
	if err != nil {
		return nil, err
	}
	var running []*AgentModel
	for _, a := range all {
		if a.Status == AgentStatusRunning {
			running = append(running, a)
		}
	}
	return running, nil
}

// DiscoverAgents returns all registered agents regardless of status.
func (m *AgentManager) DiscoverAgents() ([]*AgentModel, error) {
	return m.store.ListAgentModels()
}

// CreateAgent persists a new dynamic agent configuration.
func (m *AgentManager) CreateAgent(_ context.Context, model *AgentModel) (*AgentModel, error) {
	if model == nil {
		return nil, fmt.Errorf("agent model is required")
	}

	now := time.Now()
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	if model.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if model.Description == "" {
		model.Description = model.Name
	}
	if model.Instructions == "" {
		model.Instructions = model.Description
	}
	if model.Status == "" {
		model.Status = AgentStatusStopped
	}
	if model.RequiredLLMCapability < 0 {
		model.RequiredLLMCapability = 0
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	model.UpdatedAt = now

	if err := m.store.SaveAgentModel(model); err != nil {
		return nil, err
	}

	return m.store.GetAgentModel(model.ID)
}

// GetAgentByName retrieves a persisted agent model by name.
func (m *AgentManager) GetAgentByName(name string) (*AgentModel, error) {
	return m.store.GetAgentModelByName(name)
}

// getOrBuildService returns a cached service or builds a new one for the agent model.
func (m *AgentManager) getOrBuildService(name string) (*Service, error) {
	m.mu.RLock()
	svc, exists := m.services[name]
	m.mu.RUnlock()

	if exists {
		return svc, nil
	}

	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return nil, err
	}

	if model.Status != AgentStatusRunning {
		return nil, fmt.Errorf("agent '%s' is not running", name)
	}

	builder := New(model.Name).
		WithSystemPrompt(model.Instructions)

	if agentgoCfg, cfgErr := config.Load(""); cfgErr == nil {
		builder.WithConfig(agentgoCfg)

		globalPool := services.GetGlobalPoolService()
		if initErr := globalPool.Initialize(context.Background(), agentgoCfg); initErr == nil {
			if llmSvc, llmErr := globalPool.GetLLMServiceWithHint(pool.SelectionHint{
				PreferredProvider: model.Model,
				PreferredModel:    model.Model,
				MinCapability:     model.RequiredLLMCapability,
			}); llmErr == nil {
				builder.WithLLM(llmSvc)
			}
		}
	}

	if model.EnableRAG {
		builder.WithRAG()
	}
	if model.EnableMemory {
		builder.WithMemory()
	}
	if model.EnablePTC {
		builder.WithPTC()
	}
	if model.EnableMCP {
		builder.WithMCP()
	}

	// If the model specifies an LLM model string, this logic would require pool support to select specifically.
	// For now, relies on the default or global pool inside Build().

	if len(model.Skills) > 0 {
		builder.WithSkills()
	}

	newSvc, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// Apply tool filters to the agent
	if len(model.MCPTools) > 0 {
		newSvc.agent.SetAllowedMCPTools(model.MCPTools)
	} else {
		newSvc.agent.SetAllowedMCPTools([]string{}) // none allowed if empty
	}

	if len(model.Skills) > 0 {
		newSvc.agent.SetAllowedSkills(model.Skills)
	} else {
		newSvc.agent.SetAllowedSkills([]string{}) // none allowed if empty
	}

	if model.Model != "" {
		newSvc.agent.SetModel(model.Model)
	}

	m.mu.Lock()
	m.services[name] = newSvc
	m.mu.Unlock()

	return newSvc, nil
}

func (m *AgentManager) ensureAgentRunning(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return err
	}
	if model.Status == AgentStatusRunning {
		return nil
	}
	return m.StartAgent(ctx, name)
}

func extractDispatchText(res *ExecutionResult) string {
	if res == nil {
		return ""
	}

	if res.PTCResult != nil && res.PTCResult.Type != PTCResultTypeText {
		text := strings.TrimSpace(res.PTCResult.FormatForLLM())
		if isMeaningfulDispatchText(text) {
			return text
		}
	}

	textCandidates := []string{
		res.Text(),
	}

	if s, ok := res.Metadata["dispatch_result"].(string); ok {
		textCandidates = append(textCandidates, s)
	}
	if s, ok := res.Metadata["final_text"].(string); ok {
		textCandidates = append(textCandidates, s)
	}

	for _, candidate := range textCandidates {
		candidate = strings.TrimSpace(candidate)
		if isMeaningfulDispatchText(candidate) {
			return candidate
		}
	}

	if res.FinalResult != nil {
		if bz, err := json.Marshal(res.FinalResult); err == nil {
			candidate := strings.TrimSpace(string(bz))
			if candidate != "" && candidate != "null" {
				return candidate
			}
		}
	}

	for _, candidate := range textCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

func isMeaningfulDispatchText(text string) bool {
	if text == "" {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(text))
	generic := map[string]struct{}{
		"task complete":                {},
		"the task has been completed.": {},
		"the task has been completed":  {},
		"the task has been completed. the information has been saved to memory.": {},
		"the information has been saved to memory.":                              {},
		"done": {},
	}

	_, isGeneric := generic[normalized]
	return !isGeneric
}

// DispatchTask synchronous delegation: runs the task on the target agent service directly.
func (m *AgentManager) DispatchTask(ctx context.Context, agentName string, instruction string) (string, error) {
	if err := m.ensureAgentRunning(ctx, agentName); err != nil {
		return "", fmt.Errorf("cannot start agent %s: %w", agentName, err)
	}

	svc, err := m.getOrBuildService(agentName)
	if err != nil {
		return "", fmt.Errorf("cannot dispatch to agent %s: %w", agentName, err)
	}

	// For dispatch, we create a temporary sub-agent flow or run directly
	// Let's run a single session Run
	res, err := svc.Run(ctx, instruction, WithMaxTurns(10))
	if err != nil {
		return "", err
	}

	if text := extractDispatchText(res); text != "" {
		return text, nil
	}

	bz, _ := json.Marshal(res.FinalResult)
	return string(bz), nil
}

// RegisterCommanderTools adds the multi-agent management tools to the Frontdesk (Commander) Agent.
func (m *AgentManager) RegisterCommanderTools(commander *Service) {
	// 1. discover_agents
	discoverDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "discover_agents",
			Description: "Discover all available specialized agents in the system and their descriptions.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
	commander.toolRegistry.Register(discoverDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		agents, err := m.DiscoverAgents()
		if err != nil {
			return nil, err
		}
		var result []map[string]interface{}
		for _, a := range agents {
			result = append(result, map[string]interface{}{
				"name":        a.Name,
				"description": a.Description,
				"status":      a.Status,
			})
		}
		return result, nil
	}, CategoryCustom)

	// 2. start_agent
	startDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "start_agent",
			Description: "Start/wake up a specific agent by name.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the agent to start.",
					},
				},
				"required": []string{"name"},
			},
		},
	}
	commander.toolRegistry.Register(startDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		err := m.StartAgent(ctx, name)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Agent '%s' started successfully.", name), nil
	}, CategoryCustom)

	// 3. stop_agent
	stopDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "stop_agent",
			Description: "Stop a currently running agent.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the agent to stop.",
					},
				},
				"required": []string{"name"},
			},
		},
	}
	commander.toolRegistry.Register(stopDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		err := m.StopAgent(ctx, name)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Agent '%s' stopped successfully.", name), nil
	}, CategoryCustom)

	// 4. delegate_task
	delegateDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "delegate_task",
			Description: "Delegate a specific task to a running agent.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the running agent.",
					},
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "The full prompt/instruction for the task.",
					},
				},
				"required": []string{"agent_name", "instruction"},
			},
		},
	}
	commander.toolRegistry.Register(delegateDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		agentName, _ := args["agent_name"].(string)
		instruction, _ := args["instruction"].(string)
		return m.DispatchTask(ctx, agentName, instruction)
	}, CategoryCustom)
}
