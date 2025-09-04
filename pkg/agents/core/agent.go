package core

import (
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// BaseAgent provides a default implementation of AgentInterface
type BaseAgent struct {
	agent *types.Agent
}

// NewBaseAgent creates a new BaseAgent instance
func NewBaseAgent(agent *types.Agent) *BaseAgent {
	return &BaseAgent{agent: agent}
}

// GetID returns the agent ID
func (a *BaseAgent) GetID() string {
	return a.agent.ID
}

// GetName returns the agent name
func (a *BaseAgent) GetName() string {
	return a.agent.Name
}

// GetType returns the agent type
func (a *BaseAgent) GetType() types.AgentType {
	return a.agent.Type
}

// GetStatus returns the agent status
func (a *BaseAgent) GetStatus() types.AgentStatus {
	return a.agent.Status
}

// Execute executes the agent workflow
func (a *BaseAgent) Execute(ctx types.ExecutionContext) (*types.ExecutionResult, error) {
	// This is a base implementation - specific agent types should override
	result := &types.ExecutionResult{
		ExecutionID: ctx.RequestID,
		AgentID:     a.agent.ID,
		Status:      types.ExecutionStatusRunning,
		StartTime:   ctx.StartTime,
		Results:     make(map[string]interface{}),
		Outputs:     make(map[string]interface{}),
		Logs:        make([]types.ExecutionLog, 0),
		StepResults: make([]types.StepResult, 0),
	}

	// Basic validation
	if err := a.Validate(); err != nil {
		result.Status = types.ExecutionStatusFailed
		result.ErrorMessage = fmt.Sprintf("Agent validation failed: %v", err)
		endTime := time.Now()
		result.EndTime = &endTime
		result.Duration = endTime.Sub(result.StartTime)
		return result, err
	}

	// Mark as completed for base implementation
	result.Status = types.ExecutionStatusCompleted
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	return result, nil
}

// Validate validates the agent configuration
func (a *BaseAgent) Validate() error {
	if a.agent == nil {
		return fmt.Errorf("agent is nil")
	}
	if a.agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if a.agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.agent.Type == "" {
		return fmt.Errorf("agent type is required")
	}
	return nil
}

// GetAgent returns the underlying agent structure
func (a *BaseAgent) GetAgent() *types.Agent {
	return a.agent
}

// UpdateStatus updates the agent status
func (a *BaseAgent) UpdateStatus(status types.AgentStatus) {
	a.agent.Status = status
	a.agent.UpdatedAt = time.Now()
}

// ResearchAgent implements research-specific functionality
type ResearchAgent struct {
	*BaseAgent
}

// NewResearchAgent creates a new research agent
func NewResearchAgent(agent *types.Agent) *ResearchAgent {
	agent.Type = types.AgentTypeResearch
	return &ResearchAgent{
		BaseAgent: NewBaseAgent(agent),
	}
}

// Execute implements research-specific execution logic
func (r *ResearchAgent) Execute(ctx types.ExecutionContext) (*types.ExecutionResult, error) {
	result, err := r.BaseAgent.Execute(ctx)
	if err != nil {
		return result, err
	}

	// Add research-specific logic here
	r.logInfo(result, "Executing research workflow", nil)

	// Research agents focus on data gathering and analysis
	result.Outputs["research_type"] = "document_analysis"
	result.Results["findings"] = "Research completed successfully"

	return result, nil
}

// WorkflowAgent implements workflow automation functionality
type WorkflowAgent struct {
	*BaseAgent
}

// NewWorkflowAgent creates a new workflow agent
func NewWorkflowAgent(agent *types.Agent) *WorkflowAgent {
	agent.Type = types.AgentTypeWorkflow
	return &WorkflowAgent{
		BaseAgent: NewBaseAgent(agent),
	}
}

// Execute implements workflow-specific execution logic
func (w *WorkflowAgent) Execute(ctx types.ExecutionContext) (*types.ExecutionResult, error) {
	result, err := w.BaseAgent.Execute(ctx)
	if err != nil {
		return result, err
	}

	// Add workflow-specific logic here
	w.logInfo(result, "Executing workflow automation", nil)

	// Workflow agents focus on multi-step processes
	result.Outputs["workflow_type"] = "automation"
	result.Results["steps_executed"] = len(w.agent.Workflow.Steps)

	return result, nil
}

// MonitoringAgent implements monitoring functionality
type MonitoringAgent struct {
	*BaseAgent
}

// NewMonitoringAgent creates a new monitoring agent
func NewMonitoringAgent(agent *types.Agent) *MonitoringAgent {
	agent.Type = types.AgentTypeMonitoring
	return &MonitoringAgent{
		BaseAgent: NewBaseAgent(agent),
	}
}

// Execute implements monitoring-specific execution logic
func (m *MonitoringAgent) Execute(ctx types.ExecutionContext) (*types.ExecutionResult, error) {
	result, err := m.BaseAgent.Execute(ctx)
	if err != nil {
		return result, err
	}

	// Add monitoring-specific logic here
	m.logInfo(result, "Executing monitoring workflow", nil)

	// Monitoring agents focus on system health and alerts
	result.Outputs["monitoring_type"] = "health_check"
	result.Results["status"] = "all_systems_operational"

	return result, nil
}

// Helper method to add info log entries
func (a *BaseAgent) logInfo(result *types.ExecutionResult, message string, data interface{}) {
	result.Logs = append(result.Logs, types.ExecutionLog{
		Timestamp: time.Now(),
		Level:     types.LogLevelInfo,
		Message:   message,
		Data:      data,
	})
}

// AgentFactory creates agents based on type
type AgentFactory struct{}

// NewAgentFactory creates a new agent factory
func NewAgentFactory() *AgentFactory {
	return &AgentFactory{}
}

// CreateAgent creates an agent of the specified type
func (f *AgentFactory) CreateAgent(agentDef *types.Agent) (types.AgentInterface, error) {
	switch agentDef.Type {
	case types.AgentTypeResearch:
		return NewResearchAgent(agentDef), nil
	case types.AgentTypeWorkflow:
		return NewWorkflowAgent(agentDef), nil
	case types.AgentTypeMonitoring:
		return NewMonitoringAgent(agentDef), nil
	default:
		return NewBaseAgent(agentDef), nil
	}
}
