package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflowEngine(t *testing.T) {
	storage := NewMockAgentStorage()
	mcpClient := NewMockMCPClient()
	executor := NewAgentExecutor(mcpClient, storage)

	engine := NewWorkflowEngine(executor)

	assert.NotNil(t, engine)
	assert.Equal(t, executor, engine.executor)
	assert.NotNil(t, engine.templateEngine)
	assert.NotNil(t, engine.validator)
}

func TestWorkflowEngine_ExecuteWorkflow(t *testing.T) {
	storage := NewMockAgentStorage()
	mcpClient := NewMockMCPClient()
	executor := NewAgentExecutor(mcpClient, storage)
	engine := NewWorkflowEngine(executor)

	agent := &types.Agent{
		ID:     "workflow-engine-test",
		Name:   "Workflow Engine Test",
		Type:   types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "step1",
					Name: "First Step",
					Type: types.StepTypeTool,
					Tool: "test_tool",
					Inputs: map[string]interface{}{
						"param1": "{{global_var}}",
					},
					Outputs: map[string]string{
						"result": "step1_result",
					},
				},
				{
					ID:   "step2",
					Name: "Second Step",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"derived_value": "{{step1_result}}_modified",
					},
				},
			},
			Variables: map[string]interface{}{
				"global_var": "global_value",
			},
		},
	}

	baseAgent := NewBaseAgent(agent)
	ctx := context.Background()

	variables := map[string]interface{}{
		"input_var": "input_value",
	}

	// Set up mock response
	mcpClient.SetResponse("test_tool", map[string]interface{}{
		"result": "tool_output",
		"status": "success",
	})

	result, err := engine.ExecuteWorkflow(ctx, baseAgent, variables)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	assert.Len(t, result.StepResults, 2)

	// Verify global variables were merged
	assert.Equal(t, true, result.Results["workflow_completed"])
	assert.Equal(t, 2, result.Results["steps_executed"])
}

func TestWorkflowEngine_ExecuteWorkflow_ValidationFailure(t *testing.T) {
	storage := NewMockAgentStorage()
	mcpClient := NewMockMCPClient()
	executor := NewAgentExecutor(mcpClient, storage)
	engine := NewWorkflowEngine(executor)

	// Agent with invalid workflow (no steps)
	agent := &types.Agent{
		ID:     "invalid-workflow-agent",
		Name:   "Invalid Workflow Agent",
		Type:   types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{}, // Empty steps - invalid
		},
	}

	baseAgent := NewBaseAgent(agent)
	ctx := context.Background()

	result, err := engine.ExecuteWorkflow(ctx, baseAgent, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "workflow validation failed")
}

func TestSimpleTemplateEngine_Render(t *testing.T) {
	engine := NewSimpleTemplateEngine()

	variables := map[string]interface{}{
		"name":   "John",
		"age":    30,
		"active": true,
		"nested": map[string]interface{}{
			"city": "New York",
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple variable",
			template: "Hello {{name}}",
			expected: "Hello John",
		},
		{
			name:     "Multiple variables",
			template: "{{name}} is {{age}} years old",
			expected: "John is 30 years old",
		},
		{
			name:     "Boolean variable",
			template: "Active: {{active}}",
			expected: "Active: true",
		},
		{
			name:     "Nested variable",
			template: "Lives in {{nested.city}}",
			expected: "Lives in New York",
		},
		{
			name:     "Non-existent variable",
			template: "Missing: {{missing}}",
			expected: "Missing: {{missing}}",
		},
		{
			name:     "No variables",
			template: "Plain text",
			expected: "Plain text",
		},
		{
			name:     "Multiple same variable",
			template: "{{name}} and {{name}}",
			expected: "John and John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Render(tt.template, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleTemplateEngine_RenderObject(t *testing.T) {
	engine := NewSimpleTemplateEngine()

	variables := map[string]interface{}{
		"name": "Alice",
		"id":   123,
	}

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "String template",
			input:    "Hello {{name}}",
			expected: "Hello Alice",
		},
		{
			name: "Map with templates",
			input: map[string]interface{}{
				"greeting": "Hi {{name}}",
				"user_id":  "{{id}}",
				"static":   "unchanged",
			},
			expected: map[string]interface{}{
				"greeting": "Hi Alice",
				"user_id":  "123",
				"static":   "unchanged",
			},
		},
		{
			name: "Array with templates",
			input: []interface{}{
				"{{name}}",
				"{{id}}",
				"static",
			},
			expected: []interface{}{
				"Alice",
				"123",
				"static",
			},
		},
		{
			name:     "Non-string, non-collection",
			input:    42,
			expected: 42,
		},
		{
			name: "Nested structure",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "{{name}}",
					"id":   "{{id}}",
				},
				"meta": []interface{}{
					"{{name}}_meta",
					42,
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"id":   "123",
				},
				"meta": []interface{}{
					"Alice_meta",
					42,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderObject(tt.input, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleTemplateEngine_GetNestedValue(t *testing.T) {
	engine := NewSimpleTemplateEngine()

	variables := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			"settings": map[string]string{
				"theme":    "dark",
				"language": "en",
			},
		},
		"simple": "value",
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{
			name:     "Simple path",
			path:     "simple",
			expected: "value",
		},
		{
			name:     "Nested path",
			path:     "user.profile.name",
			expected: "John",
		},
		{
			name:     "Nested path with different map type",
			path:     "user.settings.theme",
			expected: "dark",
		},
		{
			name:     "Non-existent path",
			path:     "user.missing.field",
			expected: nil,
		},
		{
			name:     "Non-existent root",
			path:     "missing",
			expected: nil,
		},
		{
			name:     "Path into non-map",
			path:     "simple.field",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.getNestedValue(variables, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowValidator_Validate(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name        string
		workflow    types.WorkflowSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid workflow",
			workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:   "step1",
						Name: "Step 1",
						Type: types.StepTypeTool,
						Tool: "test_tool",
					},
					{
						ID:   "step2",
						Name: "Step 2",
						Type: types.StepTypeVariable,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty workflow",
			workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{},
			},
			expectError: true,
			errorMsg:    "workflow must have at least one step",
		},
		{
			name: "Step with invalid dependency",
			workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:        "step1",
						Name:      "Step 1",
						Type:      types.StepTypeTool,
						Tool:      "test_tool",
						DependsOn: []string{"non_existent_step"},
					},
				},
			},
			expectError: true,
			errorMsg:    "depends on non-existent step",
		},
		{
			name: "Valid workflow with dependencies",
			workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:   "step1",
						Name: "Step 1",
						Type: types.StepTypeTool,
						Tool: "test_tool",
					},
					{
						ID:        "step2",
						Name:      "Step 2",
						Type:      types.StepTypeVariable,
						DependsOn: []string{"step1"},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.workflow)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWorkflowValidator_ValidateStep(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name        string
		step        types.WorkflowStep
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid tool step",
			step: types.WorkflowStep{
				ID:   "tool_step",
				Name: "Tool Step",
				Type: types.StepTypeTool,
				Tool: "test_tool",
			},
			expectError: false,
		},
		{
			name: "Valid variable step",
			step: types.WorkflowStep{
				ID:   "var_step",
				Name: "Variable Step",
				Type: types.StepTypeVariable,
			},
			expectError: false,
		},
		{
			name: "Valid delay step",
			step: types.WorkflowStep{
				ID:   "delay_step",
				Name: "Delay Step",
				Type: types.StepTypeDelay,
				Inputs: map[string]interface{}{
					"duration": "5s",
				},
			},
			expectError: false,
		},
		{
			name: "Valid condition step",
			step: types.WorkflowStep{
				ID:   "condition_step",
				Name: "Condition Step",
				Type: types.StepTypeCondition,
				Conditions: []types.Condition{
					{
						Field:    "status",
						Operator: "eq",
						Value:    "active",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Step without ID",
			step: types.WorkflowStep{
				Name: "Step",
				Type: types.StepTypeTool,
				Tool: "test_tool",
			},
			expectError: true,
			errorMsg:    "step ID is required",
		},
		{
			name: "Step without name",
			step: types.WorkflowStep{
				ID:   "step1",
				Type: types.StepTypeTool,
				Tool: "test_tool",
			},
			expectError: true,
			errorMsg:    "step name is required",
		},
		{
			name: "Step without type",
			step: types.WorkflowStep{
				ID:   "step1",
				Name: "Step 1",
				Tool: "test_tool",
			},
			expectError: true,
			errorMsg:    "step type is required",
		},
		{
			name: "Tool step without tool name",
			step: types.WorkflowStep{
				ID:   "tool_step",
				Name: "Tool Step",
				Type: types.StepTypeTool,
			},
			expectError: true,
			errorMsg:    "tool step requires tool name",
		},
		{
			name: "Condition step without conditions",
			step: types.WorkflowStep{
				ID:   "condition_step",
				Name: "Condition Step",
				Type: types.StepTypeCondition,
			},
			expectError: true,
			errorMsg:    "condition step requires conditions",
		},
		{
			name: "Delay step without duration",
			step: types.WorkflowStep{
				ID:   "delay_step",
				Name: "Delay Step",
				Type: types.StepTypeDelay,
			},
			expectError: true,
			errorMsg:    "delay step requires duration input",
		},
		{
			name: "Unsupported step type",
			step: types.WorkflowStep{
				ID:   "unsupported_step",
				Name: "Unsupported Step",
				Type: "unsupported",
			},
			expectError: true,
			errorMsg:    "unsupported step type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStep(tt.step)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewToolChainExecutor(t *testing.T) {
	mcpClient := NewMockMCPClient()
	executor := NewToolChainExecutor(mcpClient)

	assert.NotNil(t, executor)
	assert.Equal(t, mcpClient, executor.mcpClient)
	assert.NotNil(t, executor.templateEngine)
}

func TestToolChainExecutor_ExecuteChain_Sequential(t *testing.T) {
	mcpClient := NewMockMCPClient()
	executor := NewToolChainExecutor(mcpClient)

	chain := types.ToolChain{
		ID:   "test_chain",
		Name: "Test Chain",
		Steps: []types.ChainStep{
			{
				ID:       "step1",
				Name:     "First Step",
				ToolName: "tool1",
				Inputs: map[string]interface{}{
					"param": "value1",
				},
				Outputs: map[string]string{
					"executed": "step1_done",
				},
			},
			{
				ID:       "step2",
				Name:     "Second Step",
				ToolName: "tool2",
				Inputs: map[string]interface{}{
					"param": "value2",
				},
				Outputs: map[string]string{
					"executed": "step2_done",
				},
			},
		},
		Variables: map[string]interface{}{
			"chain_var": "chain_value",
		},
		Parallel: false,
	}

	ctx := context.Background()
	result, err := executor.ExecuteChain(ctx, chain)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test_chain", result.ChainID)
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.StepResults, 2)

	// Verify steps executed in order
	step1Result := result.StepResults[0]
	assert.Equal(t, "step1", step1Result.StepID)
	assert.Equal(t, "completed", step1Result.Status)

	step2Result := result.StepResults[1]
	assert.Equal(t, "step2", step2Result.StepID)
	assert.Equal(t, "completed", step2Result.Status)
}

func TestToolChainExecutor_ExecuteChain_Parallel(t *testing.T) {
	mcpClient := NewMockMCPClient()
	executor := NewToolChainExecutor(mcpClient)

	chain := types.ToolChain{
		ID:   "parallel_chain",
		Name: "Parallel Chain",
		Steps: []types.ChainStep{
			{
				ID:       "parallel_step1",
				Name:     "Parallel Step 1",
				ToolName: "tool1",
			},
			{
				ID:       "parallel_step2",
				Name:     "Parallel Step 2",
				ToolName: "tool2",
			},
		},
		Parallel: true,
	}

	ctx := context.Background()
	result, err := executor.ExecuteChain(ctx, chain)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "parallel_chain", result.ChainID)
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.StepResults, 2)
}

func TestToolChainExecutor_ExecuteChain_WithError(t *testing.T) {
	mcpClient := NewMockMCPClient()
	executor := NewToolChainExecutor(mcpClient)

	// Set up error for one of the tools
	mcpClient.SetError("failing_tool", fmt.Errorf("tool execution failed"))

	chain := types.ToolChain{
		ID:   "failing_chain",
		Name: "Failing Chain",
		Steps: []types.ChainStep{
			{
				ID:       "good_step",
				Name:     "Good Step",
				ToolName: "good_tool",
			},
			{
				ID:       "bad_step",
				Name:     "Bad Step",
				ToolName: "failing_tool",
			},
		},
		Parallel: false,
	}

	ctx := context.Background()
	result, err := executor.ExecuteChain(ctx, chain)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.ErrorMessage, "step 1 failed")
	assert.Len(t, result.StepResults, 2)

	// First step should succeed
	step1 := result.StepResults[0]
	assert.Equal(t, "completed", step1.Status)

	// Second step should fail
	step2 := result.StepResults[1]
	assert.Equal(t, "failed", step2.Status)
	assert.NotEmpty(t, step2.ErrorMessage)
}

func TestChainResult_Structure(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Second)

	result := ChainResult{
		ChainID:   "test_chain",
		StartTime: now,
		EndTime:   &endTime,
		Duration:  5 * time.Second,
		Status:    "completed",
		Results:   map[string]interface{}{"success": true},
		StepResults: []ChainStepResult{
			{
				StepID:    "step1",
				Name:      "Test Step",
				StartTime: now,
				EndTime:   &endTime,
				Duration:  5 * time.Second,
				Status:    "completed",
				Inputs:    map[string]interface{}{"input": "value"},
				Outputs:   map[string]interface{}{"output": "result"},
			},
		},
		Variables: map[string]interface{}{"var": "value"},
	}

	assert.Equal(t, "test_chain", result.ChainID)
	assert.Equal(t, now, result.StartTime)
	assert.Equal(t, &endTime, result.EndTime)
	assert.Equal(t, 5*time.Second, result.Duration)
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.StepResults, 1)
	assert.NotNil(t, result.Results)
	assert.NotNil(t, result.Variables)
}

func TestChainStepResult_Structure(t *testing.T) {
	now := time.Now()
	endTime := now.Add(2 * time.Second)

	stepResult := ChainStepResult{
		StepID:       "test_step",
		Name:         "Test Step",
		StartTime:    now,
		EndTime:      &endTime,
		Duration:     2 * time.Second,
		Status:       "completed",
		Inputs:       map[string]interface{}{"input": "test"},
		Outputs:      map[string]interface{}{"output": "result"},
		ErrorMessage: "",
	}

	assert.Equal(t, "test_step", stepResult.StepID)
	assert.Equal(t, "Test Step", stepResult.Name)
	assert.Equal(t, now, stepResult.StartTime)
	assert.Equal(t, &endTime, stepResult.EndTime)
	assert.Equal(t, 2*time.Second, stepResult.Duration)
	assert.Equal(t, "completed", stepResult.Status)
	assert.NotNil(t, stepResult.Inputs)
	assert.NotNil(t, stepResult.Outputs)
	assert.Empty(t, stepResult.ErrorMessage)
}

func TestToolChainExecutor_ExecuteChain_Cancellation(t *testing.T) {
	mcpClient := NewMockMCPClient()
	executor := NewToolChainExecutor(mcpClient)

	chain := types.ToolChain{
		ID:   "cancellable_chain",
		Name: "Cancellable Chain",
		Steps: []types.ChainStep{
			{
				ID:       "step1",
				Name:     "Step 1",
				ToolName: "tool1",
			},
		},
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := executor.ExecuteChain(ctx, chain)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "cancelled", result.Status)
	assert.Equal(t, context.Canceled, err)
}
