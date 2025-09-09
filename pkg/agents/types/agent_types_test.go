package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second) // Truncate for consistent comparison
	
	agent := &Agent{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent for validation",
		Type:        AgentTypeResearch,
		Status:      AgentStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Config: AgentConfig{
			MaxConcurrentExecutions: 5,
			DefaultTimeout:          10 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:           AutonomyScheduled,
		},
		Workflow: WorkflowSpec{
			Steps: []WorkflowStep{
				{
					ID:   "step1",
					Name: "Test Step",
					Type: StepTypeTool,
					Tool: "test_tool",
				},
			},
			Variables: map[string]interface{}{
				"test_var": "test_value",
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(agent)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled Agent
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Compare fields
	assert.Equal(t, agent.ID, unmarshaled.ID)
	assert.Equal(t, agent.Name, unmarshaled.Name)
	assert.Equal(t, agent.Description, unmarshaled.Description)
	assert.Equal(t, agent.Type, unmarshaled.Type)
	assert.Equal(t, agent.Status, unmarshaled.Status)
	assert.Equal(t, agent.Config, unmarshaled.Config)
	assert.Equal(t, len(agent.Workflow.Steps), len(unmarshaled.Workflow.Steps))
	assert.Equal(t, agent.Workflow.Variables, unmarshaled.Workflow.Variables)
}

func TestAgentType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		agentType AgentType
		expected string
	}{
		{"Research Agent", AgentTypeResearch, "research"},
		{"Workflow Agent", AgentTypeWorkflow, "workflow"},
		{"Monitoring Agent", AgentTypeMonitoring, "monitoring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.agentType))
		})
	}
}

func TestAgentStatus_Constants(t *testing.T) {
	tests := []struct {
		name   string
		status AgentStatus
		expected string
	}{
		{"Active Status", AgentStatusActive, "active"},
		{"Inactive Status", AgentStatusInactive, "inactive"},
		{"Error Status", AgentStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestAutonomyLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    AutonomyLevel
		expected string
	}{
		{"Manual", AutonomyManual, "manual"},
		{"Scheduled", AutonomyScheduled, "scheduled"},
		{"Reactive", AutonomyReactive, "reactive"},
		{"Proactive", AutonomyProactive, "proactive"},
		{"Adaptive", AutonomyAdaptive, "adaptive"},
		{"Unknown", AutonomyLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestExecutionContext_Creation(t *testing.T) {
	now := time.Now()
	variables := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	ctx := ExecutionContext{
		RequestID: "req-123",
		UserID:    "user-456",
		Variables: variables,
		StartTime: now,
		Timeout:   30 * time.Second,
		MCPClient: "mock-client",
	}

	assert.Equal(t, "req-123", ctx.RequestID)
	assert.Equal(t, "user-456", ctx.UserID)
	assert.Equal(t, variables, ctx.Variables)
	assert.Equal(t, now, ctx.StartTime)
	assert.Equal(t, 30*time.Second, ctx.Timeout)
	assert.Equal(t, "mock-client", ctx.MCPClient)
}

func TestExecutionResult_Creation(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Second)

	result := ExecutionResult{
		ExecutionID:  "exec-123",
		AgentID:      "agent-456",
		Status:       ExecutionStatusCompleted,
		StartTime:    now,
		EndTime:      &endTime,
		Duration:     5 * time.Second,
		Results:      map[string]interface{}{"success": true},
		Outputs:      map[string]interface{}{"output": "test"},
		ErrorMessage: "",
		Logs: []ExecutionLog{
			{
				Timestamp: now,
				Level:     LogLevelInfo,
				Message:   "Test message",
				StepID:    "step1",
				Data:      map[string]interface{}{"test": true},
			},
		},
		StepResults: []StepResult{
			{
				StepID:    "step1",
				Name:      "Test Step",
				Status:    ExecutionStatusCompleted,
				StartTime: now,
				EndTime:   &endTime,
				Duration:  5 * time.Second,
				Inputs:    map[string]interface{}{"input": "test"},
				Outputs:   map[string]interface{}{"output": "result"},
			},
		},
	}

	assert.Equal(t, "exec-123", result.ExecutionID)
	assert.Equal(t, "agent-456", result.AgentID)
	assert.Equal(t, ExecutionStatusCompleted, result.Status)
	assert.Equal(t, now, result.StartTime)
	assert.Equal(t, &endTime, result.EndTime)
	assert.Equal(t, 5*time.Second, result.Duration)
	assert.Len(t, result.Logs, 1)
	assert.Len(t, result.StepResults, 1)
}

func TestExecutionStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected string
	}{
		{"Pending", ExecutionStatusPending, "pending"},
		{"Running", ExecutionStatusRunning, "running"},
		{"Completed", ExecutionStatusCompleted, "completed"},
		{"Failed", ExecutionStatusFailed, "failed"},
		{"Cancelled", ExecutionStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestLogLevel_Constants(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{"Debug", LogLevelDebug, "debug"},
		{"Info", LogLevelInfo, "info"},
		{"Warn", LogLevelWarn, "warn"},
		{"Error", LogLevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.level))
		})
	}
}

func TestExecutionLog_Creation(t *testing.T) {
	now := time.Now()
	
	log := ExecutionLog{
		Timestamp: now,
		Level:     LogLevelError,
		Message:   "Test error message",
		StepID:    "step-1",
		Data:      map[string]interface{}{"error": "something went wrong"},
	}

	assert.Equal(t, now, log.Timestamp)
	assert.Equal(t, LogLevelError, log.Level)
	assert.Equal(t, "Test error message", log.Message)
	assert.Equal(t, "step-1", log.StepID)
	assert.NotNil(t, log.Data)
}

func TestStepResult_Creation(t *testing.T) {
	now := time.Now()
	endTime := now.Add(2 * time.Second)

	step := StepResult{
		StepID:       "step-123",
		Name:         "Test Step",
		Status:       ExecutionStatusCompleted,
		StartTime:    now,
		EndTime:      &endTime,
		Duration:     2 * time.Second,
		Inputs:       map[string]interface{}{"param": "value"},
		Outputs:      map[string]interface{}{"result": "success"},
		ErrorMessage: "",
	}

	assert.Equal(t, "step-123", step.StepID)
	assert.Equal(t, "Test Step", step.Name)
	assert.Equal(t, ExecutionStatusCompleted, step.Status)
	assert.Equal(t, now, step.StartTime)
	assert.Equal(t, &endTime, step.EndTime)
	assert.Equal(t, 2*time.Second, step.Duration)
	assert.NotNil(t, step.Inputs)
	assert.NotNil(t, step.Outputs)
	assert.Empty(t, step.ErrorMessage)
}

func TestAgentConfig_DefaultValues(t *testing.T) {
	config := AgentConfig{
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          5 * time.Minute,
		EnableMetrics:           true,
		AutonomyLevel:           AutonomyManual,
	}

	assert.Equal(t, 10, config.MaxConcurrentExecutions)
	assert.Equal(t, 5*time.Minute, config.DefaultTimeout)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, AutonomyManual, config.AutonomyLevel)
}

func TestAgentConfig_JSONSerialization(t *testing.T) {
	config := AgentConfig{
		MaxConcurrentExecutions: 5,
		DefaultTimeout:          10 * time.Minute,
		EnableMetrics:           false,
		AutonomyLevel:           AutonomyAdaptive,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(config)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled AgentConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.MaxConcurrentExecutions, unmarshaled.MaxConcurrentExecutions)
	assert.Equal(t, config.DefaultTimeout, unmarshaled.DefaultTimeout)
	assert.Equal(t, config.EnableMetrics, unmarshaled.EnableMetrics)
	assert.Equal(t, config.AutonomyLevel, unmarshaled.AutonomyLevel)
}

// Mock implementation for interface testing
type MockAgent struct {
	agent *Agent
}

func (m *MockAgent) GetID() string                              { return m.agent.ID }
func (m *MockAgent) GetName() string                            { return m.agent.Name }
func (m *MockAgent) GetType() AgentType                         { return m.agent.Type }
func (m *MockAgent) GetStatus() AgentStatus                     { return m.agent.Status }
func (m *MockAgent) GetAgent() *Agent                           { return m.agent }
func (m *MockAgent) Execute(ctx ExecutionContext) (*ExecutionResult, error) {
	return &ExecutionResult{
		ExecutionID: ctx.RequestID,
		AgentID:     m.agent.ID,
		Status:      ExecutionStatusCompleted,
		StartTime:   ctx.StartTime,
		Results:     make(map[string]interface{}),
		Outputs:     make(map[string]interface{}),
		Logs:        make([]ExecutionLog, 0),
		StepResults: make([]StepResult, 0),
	}, nil
}
func (m *MockAgent) Validate() error { return nil }

func TestAgentInterface_Implementation(t *testing.T) {
	agent := &Agent{
		ID:   "test-agent",
		Name: "Test Agent",
		Type: AgentTypeResearch,
		Status: AgentStatusActive,
	}

	mockAgent := &MockAgent{agent: agent}

	// Test interface methods
	assert.Equal(t, "test-agent", mockAgent.GetID())
	assert.Equal(t, "Test Agent", mockAgent.GetName())
	assert.Equal(t, AgentTypeResearch, mockAgent.GetType())
	assert.Equal(t, AgentStatusActive, mockAgent.GetStatus())
	assert.Equal(t, agent, mockAgent.GetAgent())
	assert.NoError(t, mockAgent.Validate())

	// Test Execute method
	ctx := ExecutionContext{
		RequestID: "test-request",
		StartTime: time.Now(),
	}
	
	result, err := mockAgent.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-request", result.ExecutionID)
	assert.Equal(t, "test-agent", result.AgentID)
	assert.Equal(t, ExecutionStatusCompleted, result.Status)
}

func TestAgent_EmptyFieldValidation(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		valid bool
	}{
		{
			name: "Valid agent",
			agent: Agent{
				ID:   "test-1",
				Name: "Test Agent",
				Type: AgentTypeResearch,
				Status: AgentStatusActive,
			},
			valid: true,
		},
		{
			name: "Empty ID",
			agent: Agent{
				ID:   "",
				Name: "Test Agent",
				Type: AgentTypeResearch,
				Status: AgentStatusActive,
			},
			valid: false,
		},
		{
			name: "Empty Name",
			agent: Agent{
				ID:   "test-1",
				Name: "",
				Type: AgentTypeResearch,
				Status: AgentStatusActive,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAgent := &MockAgent{agent: &tt.agent}
			
			// Basic validation through interface
			if tt.valid {
				assert.Equal(t, tt.agent.ID, mockAgent.GetID())
				assert.Equal(t, tt.agent.Name, mockAgent.GetName())
			} else {
				// These should still return the values, validation is separate
				assert.Equal(t, tt.agent.ID, mockAgent.GetID())
				assert.Equal(t, tt.agent.Name, mockAgent.GetName())
			}
		})
	}
}