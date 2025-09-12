package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPToolExecutor(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	assert.NotNil(t, executor)
	assert.Equal(t, mockClient, executor.mcpClient)
}

func TestMCPToolExecutor_ExecuteTool_Success(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()
	toolName := "sqlite_query"
	inputs := map[string]interface{}{
		"query":    "SELECT * FROM users",
		"database": "test.db",
	}

	result, err := executor.ExecuteTool(ctx, toolName, inputs)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, toolName, result.Tool)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Result)
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.NotNil(t, result.Metadata)

	// Verify metadata
	assert.Contains(t, result.Metadata, "execution_time")
	assert.Contains(t, result.Metadata, "inputs")
	assert.Equal(t, inputs, result.Metadata["inputs"])
}

func TestMCPToolExecutor_ExecuteTool_Success_AllTools(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()

	// Test all supported tools
	tests := []struct {
		toolName string
		inputs   map[string]interface{}
	}{
		{
			toolName: "sqlite_query",
			inputs: map[string]interface{}{
				"query": "SELECT * FROM users",
			},
		},
		{
			toolName: "file_read",
			inputs: map[string]interface{}{
				"path": "/test/file.txt",
			},
		},
		{
			toolName: "web_search",
			inputs: map[string]interface{}{
				"query": "test search",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result, err := executor.ExecuteTool(ctx, tt.toolName, tt.inputs)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.toolName, result.Tool)
			assert.True(t, result.Success)
			assert.NotNil(t, result.Result)
			assert.Empty(t, result.Error)
			assert.Greater(t, result.Duration, time.Duration(0))
		})
	}
}

func TestMCPToolExecutor_ListAvailableTools(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()
	tools, err := executor.ListAvailableTools(ctx)

	require.NoError(t, err)
	assert.Len(t, tools, 3) // sqlite_query, file_read, web_search

	// Verify tool structure
	expectedTools := map[string]bool{
		"sqlite_query": false,
		"file_read":    false,
		"web_search":   false,
	}

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotEmpty(t, tool.Server)

		if _, exists := expectedTools[tool.Name]; exists {
			expectedTools[tool.Name] = true
		}
	}

	// Verify all expected tools were found
	for toolName, found := range expectedTools {
		assert.True(t, found, "Tool %s not found", toolName)
	}
}

func TestMCPToolExecutor_ValidateToolInputs_Success(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()

	tests := []struct {
		name     string
		toolName string
		inputs   map[string]interface{}
	}{
		{
			name:     "sqlite_query with required fields",
			toolName: "sqlite_query",
			inputs: map[string]interface{}{
				"query":    "SELECT * FROM users",
				"database": "test.db",
			},
		},
		{
			name:     "file_read with required fields",
			toolName: "file_read",
			inputs: map[string]interface{}{
				"path": "/path/to/file.txt",
			},
		},
		{
			name:     "web_search with required fields",
			toolName: "web_search",
			inputs: map[string]interface{}{
				"query": "test search",
				"limit": 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateToolInputs(ctx, tt.toolName, tt.inputs)
			assert.NoError(t, err)
		})
	}
}

func TestMCPToolExecutor_ValidateToolInputs_Failures(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()

	tests := []struct {
		name        string
		toolName    string
		inputs      map[string]interface{}
		expectedErr string
	}{
		{
			name:        "Non-existent tool",
			toolName:    "non_existent_tool",
			inputs:      map[string]interface{}{},
			expectedErr: "tool non_existent_tool not found",
		},
		{
			name:        "Missing required field",
			toolName:    "sqlite_query",
			inputs:      map[string]interface{}{}, // Missing required "query" field
			expectedErr: "required field query is missing",
		},
		{
			name:        "Missing required field for file_read",
			toolName:    "file_read",
			inputs:      map[string]interface{}{}, // Missing required "path" field
			expectedErr: "required field path is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateToolInputs(ctx, tt.toolName, tt.inputs)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestNewMCPWorkflowStepExecutor(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.toolExecutor)
	assert.Equal(t, templateEngine, executor.templateEngine)
}

func TestMCPWorkflowStepExecutor_ExecuteStep_Success(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		ID:   "test_step",
		Name: "Test Step",
		Type: types.StepTypeTool,
		Tool: "sqlite_query",
		Inputs: map[string]interface{}{
			"query":    "{{sql_query}}",
			"database": "{{db_path}}",
		},
		Outputs: map[string]string{
			"result":   "query_result",
			"rowCount": "row_count",
		},
	}

	variables := map[string]interface{}{
		"sql_query": "SELECT * FROM users",
		"db_path":   "test.db",
	}

	// Set up template engine to render variables
	templateEngine.SetResponse("{{sql_query}}", "SELECT * FROM users")
	templateEngine.SetResponse("{{db_path}}", "test.db")

	ctx := context.Background()
	result, err := executor.ExecuteStep(ctx, step, variables)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test_step", result.StepID)
	assert.Equal(t, "Test Step", result.Name)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	assert.NotNil(t, result.EndTime)
	assert.Greater(t, result.Duration, time.Duration(0))

	// Check that inputs were rendered
	assert.Equal(t, "SELECT * FROM users", result.Inputs["query"])
	assert.Equal(t, "test.db", result.Inputs["database"])

	// Check outputs mapping
	assert.NotNil(t, result.Outputs)
}

func TestMCPWorkflowStepExecutor_ExecuteStep_TemplateError(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		ID:   "template_error_step",
		Name: "Template Error Step",
		Type: types.StepTypeTool,
		Tool: "sqlite_query",
		Inputs: map[string]interface{}{
			"query": "{{invalid_template}}",
		},
	}

	variables := map[string]interface{}{}

	// Set up template engine to return error
	templateEngine.SetError("{{invalid_template}}", fmt.Errorf("template rendering failed"))

	ctx := context.Background()
	result, err := executor.ExecuteStep(ctx, step, variables)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusFailed, result.Status)
	assert.Contains(t, result.ErrorMessage, "failed to render input")
}

func TestMCPWorkflowStepExecutor_ExecuteStep_ValidationError(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		ID:   "validation_error_step",
		Name: "Validation Error Step",
		Type: types.StepTypeTool,
		Tool: "sqlite_query",
		Inputs: map[string]interface{}{
			// Missing required "query" field
			"database": "test.db",
		},
	}

	variables := map[string]interface{}{}

	ctx := context.Background()
	result, err := executor.ExecuteStep(ctx, step, variables)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusFailed, result.Status)
	assert.Contains(t, result.ErrorMessage, "input validation failed")
}

func TestMCPWorkflowStepExecutor_ExecuteStep_UnknownTool(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		ID:   "unknown_tool_step",
		Name: "Unknown Tool Step",
		Type: types.StepTypeTool,
		Tool: "unknown_tool",
		Inputs: map[string]interface{}{
			"param": "value",
		},
	}

	variables := map[string]interface{}{}

	ctx := context.Background()
	result, err := executor.ExecuteStep(ctx, step, variables)

	// Should fail during validation since unknown tools are not in the tools list
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusFailed, result.Status)
	assert.Contains(t, result.ErrorMessage, "input validation failed")
}

func TestMCPTool_Structure(t *testing.T) {
	tool := MCPTool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Server: "test-server",
	}

	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "A test tool", tool.Description)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "test-server", tool.Server)
}

func TestMCPServerStatus_Structure(t *testing.T) {
	status := MCPServerStatus{
		Name:      "test-server",
		Status:    "running",
		Connected: true,
	}

	assert.Equal(t, "test-server", status.Name)
	assert.Equal(t, "running", status.Status)
	assert.True(t, status.Connected)
}

func TestMCPExecutionResult_Structure(t *testing.T) {
	result := MCPExecutionResult{
		Tool:     "test_tool",
		Success:  true,
		Result:   map[string]interface{}{"output": "value"},
		Error:    "",
		Duration: 100 * time.Millisecond,
		Metadata: map[string]interface{}{"version": "1.0"},
	}

	assert.Equal(t, "test_tool", result.Tool)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Result)
	assert.Empty(t, result.Error)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
	assert.NotNil(t, result.Metadata)
}

func TestMockMCPClient_CallTool(t *testing.T) {
	mockClient := NewMockMCPClient()

	tests := []struct {
		name     string
		toolName string
		inputs   map[string]interface{}
	}{
		{
			name:     "sqlite_query",
			toolName: "sqlite_query",
			inputs:   map[string]interface{}{"query": "SELECT * FROM users"},
		},
		{
			name:     "file_read",
			toolName: "file_read",
			inputs:   map[string]interface{}{"path": "/test/file.txt"},
		},
		{
			name:     "web_search",
			toolName: "web_search",
			inputs:   map[string]interface{}{"query": "test search"},
		},
		{
			name:     "unknown_tool",
			toolName: "unknown_tool",
			inputs:   map[string]interface{}{"param": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := mockClient.CallTool(ctx, tt.toolName, tt.inputs)

			require.NoError(t, err)
			assert.NotNil(t, result)

			// Verify result structure
			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok)

			// Different tools return different structures
			switch tt.toolName {
			case "sqlite_query":
				assert.Contains(t, resultMap, "result")
				assert.Contains(t, resultMap, "rowCount")
			case "file_read":
				assert.Contains(t, resultMap, "content")
				assert.Contains(t, resultMap, "size")
			case "web_search":
				assert.Contains(t, resultMap, "results")
				assert.Contains(t, resultMap, "totalResults")
			default:
				assert.Contains(t, resultMap, "message")
				assert.Contains(t, resultMap, "arguments")
			}
		})
	}
}

func TestMockMCPClient_GetServerStatus(t *testing.T) {
	mockClient := NewMockMCPClient()
	ctx := context.Background()

	status, err := mockClient.GetServerStatus(ctx, "test-server")

	require.NoError(t, err)
	assert.Equal(t, "test-server", status.Name)
	assert.Equal(t, "running", status.Status)
	assert.True(t, status.Connected)
}

func TestMockMCPClient_SetMockResult(t *testing.T) {
	mockClient := NewMockMCPClient()

	customResult := map[string]interface{}{
		"custom": "result",
		"data":   123,
	}

	mockClient.SetMockResult("custom_tool", customResult)

	ctx := context.Background()
	result, err := mockClient.CallTool(ctx, "custom_tool", nil)

	require.NoError(t, err)
	assert.Equal(t, customResult, result)
}

func TestConvertToMCPResult(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Second)

	executionResult := &types.ExecutionResult{
		ExecutionID:  "exec-123",
		AgentID:      "agent-456",
		Status:       types.ExecutionStatusCompleted,
		StartTime:    now,
		EndTime:      &endTime,
		Duration:     5 * time.Second,
		Results:      map[string]interface{}{"success": true},
		Outputs:      map[string]interface{}{"output": "result"},
		ErrorMessage: "",
		Logs: []types.ExecutionLog{
			{
				Timestamp: now,
				Level:     types.LogLevelInfo,
				Message:   "Test log",
			},
		},
		StepResults: []types.StepResult{
			{
				StepID:       "step-1",
				Name:         "Test Step",
				Status:       types.ExecutionStatusCompleted,
				StartTime:    now,
				EndTime:      &endTime,
				Duration:     5 * time.Second,
				Inputs:       map[string]interface{}{"input": "value"},
				Outputs:      map[string]interface{}{"output": "result"},
				ErrorMessage: "",
			},
		},
	}

	mcpResult := ConvertToMCPResult(executionResult)

	// Verify basic fields
	assert.Equal(t, "exec-123", mcpResult["execution_id"])
	assert.Equal(t, "agent-456", mcpResult["agent_id"])
	assert.Equal(t, "completed", mcpResult["status"])
	assert.Equal(t, now, mcpResult["start_time"])
	assert.Equal(t, endTime, mcpResult["end_time"])
	assert.Equal(t, "5s", mcpResult["duration"])
	assert.Equal(t, executionResult.Results, mcpResult["results"])
	assert.Equal(t, executionResult.Outputs, mcpResult["outputs"])

	// Should not have error field since ErrorMessage is empty
	_, hasError := mcpResult["error"]
	assert.False(t, hasError)

	// Verify step results
	stepResults, ok := mcpResult["step_results"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, stepResults, 1)

	stepResult := stepResults[0]
	assert.Equal(t, "step-1", stepResult["step_id"])
	assert.Equal(t, "Test Step", stepResult["name"])
	assert.Equal(t, "completed", stepResult["status"])
	assert.Equal(t, "5s", stepResult["duration"])
	assert.Equal(t, executionResult.StepResults[0].Inputs, stepResult["inputs"])
	assert.Equal(t, executionResult.StepResults[0].Outputs, stepResult["outputs"])
}

func TestConvertToMCPResult_WithError(t *testing.T) {
	executionResult := &types.ExecutionResult{
		ExecutionID:  "exec-error",
		AgentID:      "agent-error",
		Status:       types.ExecutionStatusFailed,
		StartTime:    time.Now(),
		Duration:     2 * time.Second,
		Results:      map[string]interface{}{},
		Outputs:      map[string]interface{}{},
		ErrorMessage: "Something went wrong",
		StepResults: []types.StepResult{
			{
				StepID:       "failing-step",
				Name:         "Failing Step",
				Status:       types.ExecutionStatusFailed,
				StartTime:    time.Now(),
				Duration:     1 * time.Second,
				ErrorMessage: "Step failed",
			},
		},
	}

	mcpResult := ConvertToMCPResult(executionResult)

	assert.Equal(t, "failed", mcpResult["status"])
	assert.Equal(t, "Something went wrong", mcpResult["error"])

	stepResults, ok := mcpResult["step_results"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, stepResults, 1)

	stepResult := stepResults[0]
	assert.Equal(t, "failed", stepResult["status"])
	assert.Equal(t, "Step failed", stepResult["error"])
}

// Mock template engine for testing
type MockTemplateEngine struct {
	responses map[string]string
	errors    map[string]error
}

func (m *MockTemplateEngine) SetResponse(template, response string) {
	if m.responses == nil {
		m.responses = make(map[string]string)
	}
	m.responses[template] = response
}

func (m *MockTemplateEngine) SetError(template string, err error) {
	if m.errors == nil {
		m.errors = make(map[string]error)
	}
	m.errors[template] = err
}

func (m *MockTemplateEngine) Render(template string, variables map[string]interface{}) (string, error) {
	if err, exists := m.errors[template]; exists {
		return "", err
	}
	if response, exists := m.responses[template]; exists {
		return response, nil
	}
	return template, nil
}

func (m *MockTemplateEngine) RenderObject(obj interface{}, variables map[string]interface{}) (interface{}, error) {
	if str, ok := obj.(string); ok {
		return m.Render(str, variables)
	}
	return obj, nil
}

func TestMCPWorkflowStepExecutor_MapStepOutputs(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		Outputs: map[string]string{
			"result":  "mapped_result",
			"count":   "mapped_count",
			"missing": "mapped_missing", // This key won't exist in result
		},
	}

	stepResult := &types.StepResult{
		Outputs: make(map[string]interface{}),
	}

	// Test with map result
	mcpResult := map[string]interface{}{
		"result": "success",
		"count":  42,
		"extra":  "not_mapped",
	}

	executor.mapStepOutputs(step, mcpResult, stepResult)

	assert.Equal(t, "success", stepResult.Outputs["mapped_result"])
	assert.Equal(t, 42, stepResult.Outputs["mapped_count"])
	_, hasMissing := stepResult.Outputs["mapped_missing"]
	assert.False(t, hasMissing)
}

func TestMCPWorkflowStepExecutor_MapStepOutputs_NonMapResult(t *testing.T) {
	mockClient := NewMockMCPClient()
	templateEngine := &MockTemplateEngine{}
	executor := NewMCPWorkflowStepExecutor(mockClient, templateEngine)

	step := types.WorkflowStep{
		Outputs: map[string]string{
			"result": "mapped_result",
			"other":  "mapped_other", // Only first should be used
		},
	}

	stepResult := &types.StepResult{
		Outputs: make(map[string]interface{}),
	}

	// Test with non-map result
	mcpResult := "simple string result"

	executor.mapStepOutputs(step, mcpResult, stepResult)

	// Only the first output should be mapped
	assert.Equal(t, "simple string result", stepResult.Outputs["mapped_result"])
	_, hasOther := stepResult.Outputs["mapped_other"]
	assert.False(t, hasOther)
}

func TestMCPWorkflowStepExecutor_NoTemplateEngine(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPWorkflowStepExecutor(mockClient, nil) // No template engine

	step := types.WorkflowStep{
		ID:   "no_template_step",
		Name: "No Template Step",
		Type: types.StepTypeTool,
		Tool: "sqlite_query",
		Inputs: map[string]interface{}{
			"query": "SELECT * FROM users", // No template variables
		},
		Outputs: map[string]string{
			"result": "query_result",
		},
	}

	variables := map[string]interface{}{}

	ctx := context.Background()
	result, err := executor.ExecuteStep(ctx, step, variables)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)

	// Inputs should remain unchanged since no template engine
	assert.Equal(t, "SELECT * FROM users", result.Inputs["query"])
}

func TestMCPToolExecutor_IntegrationFlow(t *testing.T) {
	mockClient := NewMockMCPClient()
	executor := NewMCPToolExecutor(mockClient)

	ctx := context.Background()

	// 1. List available tools
	tools, err := executor.ListAvailableTools(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(tools), 0)

	// 2. Validate inputs for a known tool
	inputs := map[string]interface{}{
		"query": "SELECT * FROM users",
	}
	err = executor.ValidateToolInputs(ctx, "sqlite_query", inputs)
	require.NoError(t, err)

	// 3. Execute the tool
	result, err := executor.ExecuteTool(ctx, "sqlite_query", inputs)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, result.Result)
}
