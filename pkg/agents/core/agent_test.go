package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseAgent(t *testing.T) {
	agent := &types.Agent{
		ID:     "test-agent-1",
		Name:   "Test Agent",
		Type:   types.AgentTypeResearch,
		Status: types.AgentStatusActive,
	}

	baseAgent := NewBaseAgent(agent)

	assert.NotNil(t, baseAgent)
	assert.Equal(t, agent, baseAgent.agent)
}

func TestBaseAgent_GetMethods(t *testing.T) {
	agent := &types.Agent{
		ID:     "test-agent-1",
		Name:   "Test Agent",
		Type:   types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
	}

	baseAgent := NewBaseAgent(agent)

	assert.Equal(t, "test-agent-1", baseAgent.GetID())
	assert.Equal(t, "Test Agent", baseAgent.GetName())
	assert.Equal(t, types.AgentTypeWorkflow, baseAgent.GetType())
	assert.Equal(t, types.AgentStatusActive, baseAgent.GetStatus())
	assert.Equal(t, agent, baseAgent.GetAgent())
}

func TestBaseAgent_UpdateStatus(t *testing.T) {
	agent := &types.Agent{
		ID:        "test-agent-1",
		Name:      "Test Agent",
		Status:    types.AgentStatusActive,
		UpdatedAt: time.Time{}, // Zero time
	}

	baseAgent := NewBaseAgent(agent)

	// Update status
	baseAgent.UpdateStatus(types.AgentStatusError)

	assert.Equal(t, types.AgentStatusError, baseAgent.GetStatus())
	assert.False(t, agent.UpdatedAt.IsZero())
}

func TestBaseAgent_Validate(t *testing.T) {
	tests := []struct {
		name        string
		agent       *types.Agent
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid agent",
			agent: &types.Agent{
				ID:   "valid-agent",
				Name: "Valid Agent",
				Type: types.AgentTypeResearch,
			},
			expectError: false,
		},
		{
			name:        "Nil agent",
			agent:       nil,
			expectError: true,
			errorMsg:    "agent is nil",
		},
		{
			name: "Empty ID",
			agent: &types.Agent{
				ID:   "",
				Name: "Test Agent",
				Type: types.AgentTypeResearch,
			},
			expectError: true,
			errorMsg:    "agent ID is required",
		},
		{
			name: "Empty Name",
			agent: &types.Agent{
				ID:   "test-agent",
				Name: "",
				Type: types.AgentTypeResearch,
			},
			expectError: true,
			errorMsg:    "agent name is required",
		},
		{
			name: "Empty Type",
			agent: &types.Agent{
				ID:   "test-agent",
				Name: "Test Agent",
				Type: "",
			},
			expectError: true,
			errorMsg:    "agent type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseAgent := NewBaseAgent(tt.agent)
			err := baseAgent.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBaseAgent_Execute(t *testing.T) {
	agent := &types.Agent{
		ID:     "test-agent-1",
		Name:   "Test Agent",
		Type:   types.AgentTypeResearch,
		Status: types.AgentStatusActive,
	}

	baseAgent := NewBaseAgent(agent)

	ctx := types.ExecutionContext{
		RequestID: "test-request-1",
		Variables: map[string]interface{}{
			"input": "test",
		},
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	result, err := baseAgent.Execute(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-request-1", result.ExecutionID)
	assert.Equal(t, "test-agent-1", result.AgentID)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	assert.NotNil(t, result.EndTime)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.NotNil(t, result.Results)
	assert.NotNil(t, result.Outputs)
	assert.NotNil(t, result.Logs)
	assert.NotNil(t, result.StepResults)
}

func TestBaseAgent_Execute_ValidationFailure(t *testing.T) {
	// Invalid agent (missing ID)
	agent := &types.Agent{
		ID:     "", // Invalid
		Name:   "Test Agent",
		Type:   types.AgentTypeResearch,
		Status: types.AgentStatusActive,
	}

	baseAgent := NewBaseAgent(agent)

	ctx := types.ExecutionContext{
		RequestID: "test-request-1",
		StartTime: time.Now(),
	}

	result, err := baseAgent.Execute(ctx)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusFailed, result.Status)
	assert.Contains(t, result.ErrorMessage, "Agent validation failed")
	assert.NotNil(t, result.EndTime)
}

func TestNewResearchAgent(t *testing.T) {
	agent := &types.Agent{
		ID:     "research-agent-1",
		Name:   "Research Agent",
		Status: types.AgentStatusActive,
	}

	researchAgent := NewResearchAgent(agent)

	assert.NotNil(t, researchAgent)
	assert.Equal(t, types.AgentTypeResearch, researchAgent.GetType())
	assert.Equal(t, "research-agent-1", researchAgent.GetID())
	assert.Equal(t, "Research Agent", researchAgent.GetName())
}

func TestResearchAgent_Execute(t *testing.T) {
	agent := &types.Agent{
		ID:     "research-agent-1",
		Name:   "Research Agent",
		Status: types.AgentStatusActive,
	}

	researchAgent := NewResearchAgent(agent)

	ctx := types.ExecutionContext{
		RequestID: "research-request-1",
		Variables: map[string]interface{}{
			"document": "test-document",
		},
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	result, err := researchAgent.Execute(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)

	// Check research-specific outputs
	assert.Equal(t, "document_analysis", result.Outputs["research_type"])
	assert.Equal(t, "Research completed successfully", result.Results["findings"])

	// Should have at least one log entry
	assert.Greater(t, len(result.Logs), 0)
}

func TestNewWorkflowAgent(t *testing.T) {
	agent := &types.Agent{
		ID:     "workflow-agent-1",
		Name:   "Workflow Agent",
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{ID: "step1", Name: "Step 1", Type: types.StepTypeTool},
				{ID: "step2", Name: "Step 2", Type: types.StepTypeVariable},
			},
		},
	}

	workflowAgent := NewWorkflowAgent(agent)

	assert.NotNil(t, workflowAgent)
	assert.Equal(t, types.AgentTypeWorkflow, workflowAgent.GetType())
	assert.Equal(t, "workflow-agent-1", workflowAgent.GetID())
	assert.Equal(t, "Workflow Agent", workflowAgent.GetName())
}

func TestWorkflowAgent_Execute(t *testing.T) {
	agent := &types.Agent{
		ID:     "workflow-agent-1",
		Name:   "Workflow Agent",
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{ID: "step1", Name: "Step 1", Type: types.StepTypeTool},
				{ID: "step2", Name: "Step 2", Type: types.StepTypeVariable},
			},
		},
	}

	workflowAgent := NewWorkflowAgent(agent)

	ctx := types.ExecutionContext{
		RequestID: "workflow-request-1",
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	result, err := workflowAgent.Execute(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)

	// Check workflow-specific outputs
	assert.Equal(t, "automation", result.Outputs["workflow_type"])
	assert.Equal(t, 2, result.Results["steps_executed"])
}

func TestNewMonitoringAgent(t *testing.T) {
	agent := &types.Agent{
		ID:     "monitoring-agent-1",
		Name:   "Monitoring Agent",
		Status: types.AgentStatusActive,
	}

	monitoringAgent := NewMonitoringAgent(agent)

	assert.NotNil(t, monitoringAgent)
	assert.Equal(t, types.AgentTypeMonitoring, monitoringAgent.GetType())
	assert.Equal(t, "monitoring-agent-1", monitoringAgent.GetID())
	assert.Equal(t, "Monitoring Agent", monitoringAgent.GetName())
}

func TestMonitoringAgent_Execute(t *testing.T) {
	agent := &types.Agent{
		ID:     "monitoring-agent-1",
		Name:   "Monitoring Agent",
		Status: types.AgentStatusActive,
	}

	monitoringAgent := NewMonitoringAgent(agent)

	ctx := types.ExecutionContext{
		RequestID: "monitoring-request-1",
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	result, err := monitoringAgent.Execute(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)

	// Check monitoring-specific outputs
	assert.Equal(t, "health_check", result.Outputs["monitoring_type"])
	assert.Equal(t, "all_systems_operational", result.Results["status"])
}

func TestAgentFactory_NewAgentFactory(t *testing.T) {
	factory := NewAgentFactory()
	assert.NotNil(t, factory)
}

func TestAgentFactory_CreateAgent(t *testing.T) {
	factory := NewAgentFactory()

	tests := []struct {
		name         string
		agentDef     *types.Agent
		expectedType types.AgentType
		expectedImpl string
	}{
		{
			name: "Create Research Agent",
			agentDef: &types.Agent{
				ID:     "research-1",
				Name:   "Research Agent",
				Type:   types.AgentTypeResearch,
				Status: types.AgentStatusActive,
			},
			expectedType: types.AgentTypeResearch,
			expectedImpl: "*core.ResearchAgent",
		},
		{
			name: "Create Workflow Agent",
			agentDef: &types.Agent{
				ID:     "workflow-1",
				Name:   "Workflow Agent",
				Type:   types.AgentTypeWorkflow,
				Status: types.AgentStatusActive,
			},
			expectedType: types.AgentTypeWorkflow,
			expectedImpl: "*core.WorkflowAgent",
		},
		{
			name: "Create Monitoring Agent",
			agentDef: &types.Agent{
				ID:     "monitoring-1",
				Name:   "Monitoring Agent",
				Type:   types.AgentTypeMonitoring,
				Status: types.AgentStatusActive,
			},
			expectedType: types.AgentTypeMonitoring,
			expectedImpl: "*core.MonitoringAgent",
		},
		{
			name: "Create Base Agent (unknown type)",
			agentDef: &types.Agent{
				ID:     "unknown-1",
				Name:   "Unknown Agent",
				Type:   "unknown",
				Status: types.AgentStatusActive,
			},
			expectedType: "unknown",
			expectedImpl: "*core.BaseAgent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := factory.CreateAgent(tt.agentDef)

			require.NoError(t, err)
			assert.NotNil(t, agent)
			assert.Equal(t, tt.expectedType, agent.GetType())
			assert.Equal(t, tt.agentDef.ID, agent.GetID())
			assert.Equal(t, tt.agentDef.Name, agent.GetName())

			// Verify implementation type
			assert.Contains(t, fmt.Sprintf("%T", agent), tt.expectedImpl)
		})
	}
}

func TestAgentInterface_Compliance(t *testing.T) {
	// Test that all agent implementations comply with the AgentInterface
	factory := NewAgentFactory()

	agentTypes := []types.AgentType{
		types.AgentTypeResearch,
		types.AgentTypeWorkflow,
		types.AgentTypeMonitoring,
	}

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			agentDef := &types.Agent{
				ID:     "test-" + string(agentType),
				Name:   "Test " + string(agentType) + " Agent",
				Type:   agentType,
				Status: types.AgentStatusActive,
			}

			agent, err := factory.CreateAgent(agentDef)
			require.NoError(t, err)

			// Test interface methods
			assert.NotEmpty(t, agent.GetID())
			assert.NotEmpty(t, agent.GetName())
			assert.NotEmpty(t, agent.GetType())
			assert.NotEmpty(t, agent.GetStatus())
			assert.NotNil(t, agent.GetAgent())
			assert.NoError(t, agent.Validate())

			// Test Execute method
			ctx := types.ExecutionContext{
				RequestID: "test-request",
				Variables: make(map[string]interface{}),
				StartTime: time.Now(),
				Timeout:   30 * time.Second,
			}

			result, err := agent.Execute(ctx)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
		})
	}
}

func TestBaseAgent_LogInfo(t *testing.T) {
	agent := &types.Agent{
		ID:     "test-agent-1",
		Name:   "Test Agent",
		Type:   types.AgentTypeResearch,
		Status: types.AgentStatusActive,
	}

	baseAgent := NewBaseAgent(agent)

	// Since logInfo is a private method on BaseAgent, we need to access it
	// For testing purposes, we'll call it indirectly through Execute which uses it
	ctx := types.ExecutionContext{
		RequestID: "test-request",
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
		Timeout:   30 * time.Second,
	}

	_, err := baseAgent.Execute(ctx)
	require.NoError(t, err)

	// The Execute method doesn't add logs in the base implementation,
	// but derived classes do. Let's test with a research agent
	researchAgent := NewResearchAgent(agent)
	researchResult, err := researchAgent.Execute(ctx)
	require.NoError(t, err)

	// Research agent should have added logs
	assert.Greater(t, len(researchResult.Logs), 0)

	// Verify log structure
	log := researchResult.Logs[0]
	assert.Equal(t, types.LogLevelInfo, log.Level)
	assert.NotEmpty(t, log.Message)
	assert.False(t, log.Timestamp.IsZero())
}

func TestAgent_ConcurrentExecution(t *testing.T) {
	agent := &types.Agent{
		ID:     "concurrent-test-agent",
		Name:   "Concurrent Test Agent",
		Type:   types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
	}

	workflowAgent := NewWorkflowAgent(agent)

	// Run multiple executions concurrently
	const numExecutions = 10
	results := make([]*types.ExecutionResult, numExecutions)
	errors := make([]error, numExecutions)

	// Use channels to coordinate goroutines
	done := make(chan bool, numExecutions)

	for i := 0; i < numExecutions; i++ {
		go func(index int) {
			defer func() { done <- true }()

			ctx := types.ExecutionContext{
				RequestID: fmt.Sprintf("concurrent-request-%d", index),
				Variables: map[string]interface{}{
					"index": index,
				},
				StartTime: time.Now(),
				Timeout:   30 * time.Second,
			}

			result, err := workflowAgent.Execute(ctx)
			results[index] = result
			errors[index] = err
		}(i)
	}

	// Wait for all executions to complete
	for i := 0; i < numExecutions; i++ {
		<-done
	}

	// Verify all executions succeeded
	for i := 0; i < numExecutions; i++ {
		assert.NoError(t, errors[i], "Execution %d should not error", i)
		assert.NotNil(t, results[i], "Result %d should not be nil", i)
		assert.Equal(t, types.ExecutionStatusCompleted, results[i].Status, "Execution %d should be completed", i)
		assert.Equal(t, fmt.Sprintf("concurrent-request-%d", i), results[i].ExecutionID, "Execution ID should match")
	}
}

// Helper function to compare agent types using string representation
func formatAgentType(agent types.AgentInterface) string {
	return fmt.Sprintf("%T", agent)
}
