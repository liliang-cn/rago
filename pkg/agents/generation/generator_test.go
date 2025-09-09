package generation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGenerator implements domain.Generator for testing
type MockGenerator struct {
	response           string
	structuredResponse *domain.StructuredResult
	generationResult   *domain.GenerationResult
	error             error
}

func (m *MockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if m.error != nil {
		return "", m.error
	}
	return m.response, nil
}

func (m *MockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if m.error != nil {
		return m.error
	}
	if callback != nil {
		callback(m.response)
	}
	return nil
}

func (m *MockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if m.error != nil {
		return nil, m.error
	}
	if m.generationResult != nil {
		return m.generationResult, nil
	}
	return &domain.GenerationResult{
		Content:   m.response,
		ToolCalls: []domain.ToolCall{},
		Finished:  true,
	}, nil
}

func (m *MockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if m.error != nil {
		return m.error
	}
	if callback != nil {
		return callback(m.response, []domain.ToolCall{})
	}
	return nil
}

func (m *MockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if m.error != nil {
		return nil, m.error
	}
	if m.structuredResponse != nil {
		return m.structuredResponse, nil
	}
	// Default fallback if no structured response is set - just return the schema pointer
	return &domain.StructuredResult{Valid: true, Data: schema, Raw: "{}"}, nil
}

func TestNewAgentGenerator(t *testing.T) {
	mockGen := &MockGenerator{}
	generator := NewAgentGenerator(mockGen)

	assert.NotNil(t, generator)
	assert.Equal(t, mockGen, generator.llm)
	assert.False(t, generator.verbose)
}

func TestAgentGenerator_SetVerbose(t *testing.T) {
	generator := NewAgentGenerator(&MockGenerator{})

	generator.SetVerbose(true)
	assert.True(t, generator.verbose)

	generator.SetVerbose(false)
	assert.False(t, generator.verbose)
}

func TestAgentGenerator_GenerateWorkflow_Structured(t *testing.T) {
	mockWorkflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "step1",
				Name: "Test Step",
				Type: types.StepTypeTool,
				Tool: "test_tool",
			},
		},
		Variables: map[string]interface{}{
			"var1": "value1",
		},
	}

	mockGen := &MockGenerator{
		structuredResponse: &domain.StructuredResult{
			Valid: true,
			Data:  mockWorkflow,
			Raw:   `{"steps":[{"id":"step1","name":"Test Step","type":"tool","tool":"test_tool"}]}`,
		},
	}

	generator := NewAgentGenerator(mockGen)
	
	ctx := context.Background()
	result, err := generator.GenerateWorkflow(ctx, "Create a test workflow")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Steps, 1)
	assert.Equal(t, "step1", result.Steps[0].ID)
	assert.Equal(t, "Test Step", result.Steps[0].Name)
}

func TestAgentGenerator_GenerateAgent(t *testing.T) {
	// This test uses the default behavior where the mock returns the schema parameter 
	// with zero values, so we test that the agent is created with default values
	mockGen := &MockGenerator{} // Uses default behavior

	generator := NewAgentGenerator(mockGen)
	
	ctx := context.Background()
	agent, err := generator.GenerateAgent(ctx, "Create a research agent", types.AgentTypeResearch)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, "", agent.Name) // Empty name from zero-value schema
	assert.Equal(t, types.AgentTypeResearch, agent.Type) // Type is set by the test parameter
	assert.Equal(t, types.AutonomyManual, agent.Config.AutonomyLevel) // Default autonomy level
	assert.Equal(t, 0, agent.Config.MaxConcurrentExecutions) // Zero value
	assert.Equal(t, time.Duration(0), agent.Config.DefaultTimeout) // Zero value
	assert.False(t, agent.Config.EnableMetrics) // Zero value
	assert.Len(t, agent.Workflow.Steps, 0) // Empty workflow from zero-value schema
}

func TestAgentGenerator_GenerateToolCall(t *testing.T) {
	mockParams := map[string]interface{}{
		"path":   "/test/file.txt",
		"action": "read",
	}

	mockGen := &MockGenerator{
		structuredResponse: &domain.StructuredResult{
			Valid: true,
			Data:  &mockParams,
			Raw:   `{"path":"/test/file.txt","action":"read"}`,
		},
	}

	generator := NewAgentGenerator(mockGen)
	
	ctx := context.Background()
	params, err := generator.GenerateToolCall(ctx, "filesystem", "Read a test file")

	require.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, "/test/file.txt", params["path"])
	assert.Equal(t, "read", params["action"])
}

func TestAgentGenerator_BuildWorkflowPrompt(t *testing.T) {
	generator := NewAgentGenerator(&MockGenerator{})
	
	prompt := generator.buildWorkflowPrompt("Create a file reading workflow")
	
	assert.Contains(t, prompt, "workflow generator")
	assert.Contains(t, prompt, "filesystem")
	assert.Contains(t, prompt, "Create a file reading workflow")
	assert.Contains(t, prompt, "MCP tools")
}

func TestAgentGenerator_BuildAgentPrompt(t *testing.T) {
	generator := NewAgentGenerator(&MockGenerator{})
	
	tests := []struct {
		agentType   types.AgentType
		description string
		expected    string
	}{
		{
			agentType:   types.AgentTypeResearch,
			description: "Research agent for data analysis",
			expected:    "Research agents focus on gathering",
		},
		{
			agentType:   types.AgentTypeWorkflow,
			description: "Workflow automation agent",
			expected:    "Workflow agents automate",
		},
		{
			agentType:   types.AgentTypeMonitoring,
			description: "System monitoring agent",
			expected:    "Monitoring agents track",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			prompt := generator.buildAgentPrompt(tt.description, tt.agentType)
			
			assert.Contains(t, prompt, string(tt.agentType))
			assert.Contains(t, prompt, tt.description)
			assert.Contains(t, prompt, tt.expected)
			assert.Contains(t, prompt, "autonomy_level")
		})
	}
}

func TestAgentGenerator_ValidateWorkflow(t *testing.T) {
	generator := NewAgentGenerator(&MockGenerator{})
	
	tests := []struct {
		name        string
		workflow    *types.WorkflowSpec
		expectError bool
	}{
		{
			name:        "Nil workflow",
			workflow:    nil,
			expectError: true,
		},
		{
			name: "Empty steps",
			workflow: &types.WorkflowSpec{
				Steps: []types.WorkflowStep{},
			},
			expectError: true,
		},
		{
			name: "Valid workflow",
			workflow: &types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:   "step1",
						Name: "Test Step",
						Type: types.StepTypeTool,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Workflow with missing IDs",
			workflow: &types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						Name: "Unnamed Step",
						Type: types.StepTypeTool,
					},
				},
			},
			expectError: false, // Should fix missing ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.validateWorkflow(tt.workflow)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.workflow != nil && len(tt.workflow.Steps) > 0 {
					// Check that missing fields are filled in
					step := tt.workflow.Steps[0]
					assert.NotEmpty(t, step.ID)
					assert.NotEmpty(t, step.Name)
					assert.NotEmpty(t, step.Type)
				}
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON in code block",
			input:    "Here is the JSON:\n```json\n{\"key\": \"value\"}\n```\nEnd of response",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "Raw JSON object",
			input:    "{\"name\": \"test\", \"value\": 123}",
			expected: "{\"name\": \"test\", \"value\": 123}",
		},
		{
			name:     "JSON with nested objects",
			input:    "{\"outer\": {\"inner\": \"value\"}}",
			expected: "{\"outer\": {\"inner\": \"value\"}}",
		},
		{
			name:     "JSON with array",
			input:    "[{\"item1\": \"value1\"}, {\"item2\": \"value2\"}]",
			expected: "[{\"item1\": \"value1\"}, {\"item2\": \"value2\"}]",
		},
		{
			name:     "JSON with escaped quotes",
			input:    "{\"message\": \"Hello \\\"world\\\"\"}",
			expected: "{\"message\": \"Hello \\\"world\\\"\"}",
		},
		{
			name:     "Plain text",
			input:    "This is not JSON",
			expected: "This is not JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// FailingMockGenerator that fails structured generation but succeeds with regular generation
type FailingMockGenerator struct {
	response string
}

func (m *FailingMockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	return m.response, nil
}

func (m *FailingMockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if callback != nil {
		callback(m.response)
	}
	return nil
}

func (m *FailingMockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return &domain.GenerationResult{
		Content:   m.response,
		ToolCalls: []domain.ToolCall{},
		Finished:  true,
	}, nil
}

func (m *FailingMockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if callback != nil {
		return callback(m.response, []domain.ToolCall{})
	}
	return nil
}

func (m *FailingMockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	return nil, fmt.Errorf("structured generation failed")
}

func TestAgentGenerator_GenerateWorkflow_FallbackToUnstructured(t *testing.T) {
	// Mock that fails structured generation but succeeds with regular generation
	mockGen := &FailingMockGenerator{
		response: `{
			"steps": [
				{
					"id": "fallback_step",
					"name": "Fallback Step",
					"type": "tool",
					"tool": "filesystem"
				}
			]
		}`,
	}

	generator := NewAgentGenerator(mockGen)
	generator.SetVerbose(true) // Enable verbose for better testing
	
	ctx := context.Background()
	result, err := generator.GenerateWorkflow(ctx, "Create a fallback workflow")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Steps, 1)
	assert.Equal(t, "fallback_step", result.Steps[0].ID)
}

func TestAgentGenerator_AutonomyLevelMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected types.AutonomyLevel
	}{
		{"manual", types.AutonomyManual},
		{"scheduled", types.AutonomyScheduled},
		{"reactive", types.AutonomyReactive},
		{"proactive", types.AutonomyProactive},
		{"adaptive", types.AutonomyAdaptive},
		{"unknown", types.AutonomyManual}, // default
		{"", types.AutonomyManual},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			agentSchema := struct {
				Name                    string             `json:"name"`
				Description             string             `json:"description"`
				MaxConcurrentExecutions int                `json:"max_concurrent_executions"`
				DefaultTimeoutMinutes   int                `json:"default_timeout_minutes"`
				EnableMetrics           bool               `json:"enable_metrics"`
				AutonomyLevel           string             `json:"autonomy_level"`
				Workflow                types.WorkflowSpec `json:"workflow"`
			}{
				Name:                    "Test Agent",
				Description:             "Test description",
				MaxConcurrentExecutions: 1,
				DefaultTimeoutMinutes:   5,
				EnableMetrics:           true,
				AutonomyLevel:           tt.input,
				Workflow:                types.WorkflowSpec{Steps: []types.WorkflowStep{{ID: "step1", Name: "Step 1", Type: "tool"}}},
			}

			mockGen := &MockGenerator{
				structuredResponse: &domain.StructuredResult{
					Valid: true,
					Data:  &agentSchema,
					Raw:   `{"name":"Test Agent","autonomy_level":"` + tt.input + `"}`,
				},
			}

			generator := NewAgentGenerator(mockGen)
			
			ctx := context.Background()
			agent, err := generator.GenerateAgent(ctx, "Test", types.AgentTypeResearch)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, agent.Config.AutonomyLevel)
		})
	}
}

func TestAgentGenerator_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		mockGen *MockGenerator
		method  string
	}{
		{
			name: "GenerateWorkflow error",
			mockGen: &MockGenerator{
				error: assert.AnError,
			},
			method: "workflow",
		},
		{
			name: "GenerateAgent error",
			mockGen: &MockGenerator{
				error: assert.AnError,
			},
			method: "agent",
		},
		{
			name: "GenerateToolCall error",
			mockGen: &MockGenerator{
				error: assert.AnError,
			},
			method: "toolcall",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewAgentGenerator(tt.mockGen)
			ctx := context.Background()

			switch tt.method {
			case "workflow":
				result, err := generator.GenerateWorkflow(ctx, "test")
				assert.Error(t, err)
				assert.Nil(t, result)

			case "agent":
				result, err := generator.GenerateAgent(ctx, "test", types.AgentTypeResearch)
				assert.Error(t, err)
				assert.Nil(t, result)

			case "toolcall":
				result, err := generator.GenerateToolCall(ctx, "test", "test")
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}