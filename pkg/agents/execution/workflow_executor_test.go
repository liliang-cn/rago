package execution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGenerator implements domain.Generator for testing
type MockGenerator struct {
	response         string
	generationResult *domain.GenerationResult
	structuredResult *domain.StructuredResult
	error           error
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
	if m.structuredResult != nil {
		return m.structuredResult, nil
	}
	return &domain.StructuredResult{Valid: true, Data: schema, Raw: "{}"}, nil
}

func TestNewWorkflowExecutor(t *testing.T) {
	cfg := &config.Config{}
	llm := &MockGenerator{response: "test response"}
	
	executor := NewWorkflowExecutor(cfg, llm)
	
	assert.NotNil(t, executor)
	assert.Equal(t, cfg, executor.config)
	assert.Equal(t, llm, executor.llmProvider)
	assert.NotNil(t, executor.mcpClients)
	assert.NotNil(t, executor.memory)
	assert.False(t, executor.verbose)
}

func TestWorkflowExecutor_SetVerbose(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	executor.SetVerbose(true)
	assert.True(t, executor.verbose)
	
	executor.SetVerbose(false)
	assert.False(t, executor.verbose)
}

func TestWorkflowExecutor_Execute_EmptyWorkflow(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	workflow := &types.WorkflowSpec{
		Steps:     []types.WorkflowStep{},
		Variables: map[string]interface{}{"test": "value"},
	}
	
	ctx := context.Background()
	result, err := executor.Execute(ctx, workflow)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	assert.Empty(t, result.StepResults)
}

func TestWorkflowExecutor_ExecuteFetch(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test response", "status": "ok"}`))
	}))
	defer server.Close()
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
	}
	
	ctx := context.Background()
	result, err := executor.executeFetch(ctx, inputs)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Should return parsed JSON
	jsonResult, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test response", jsonResult["message"])
}

func TestWorkflowExecutor_ExecuteFetch_TextResponse(t *testing.T) {
	// Create test server that returns plain text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"url": server.URL,
	}
	
	ctx := context.Background()
	result, err := executor.executeFetch(ctx, inputs)
	
	require.NoError(t, err)
	assert.Equal(t, "plain text response", result)
}

func TestWorkflowExecutor_ExecuteFetch_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom header
		if r.Header.Get("X-Custom-Header") == "test-value" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("header received"))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"url": server.URL,
		"headers": map[string]interface{}{
			"X-Custom-Header": "test-value",
		},
	}
	
	ctx := context.Background()
	result, err := executor.executeFetch(ctx, inputs)
	
	require.NoError(t, err)
	assert.Equal(t, "header received", result)
}

func TestWorkflowExecutor_ExecuteFilesystem_Read(t *testing.T) {
	// Create temporary file
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "test_read.txt")
	testContent := "test file content"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"action": "read",
		"path":   testFile,
	}
	
	ctx := context.Background()
	result, err := executor.executeFilesystem(ctx, inputs)
	
	require.NoError(t, err)
	assert.Equal(t, testContent, result)
}

func TestWorkflowExecutor_ExecuteFilesystem_Write(t *testing.T) {
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "test_write.txt")
	testContent := "written content"
	
	defer os.Remove(testFile)
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"action":  "write",
		"path":    testFile,
		"content": testContent,
	}
	
	ctx := context.Background()
	result, err := executor.executeFilesystem(ctx, inputs)
	
	require.NoError(t, err)
	assert.Equal(t, testFile, result)
	
	// Verify file was written
	written, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(written))
}

func TestWorkflowExecutor_ExecuteFilesystem_Append(t *testing.T) {
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "test_append.txt")
	initialContent := "initial"
	appendContent := " appended"
	
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)
	
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"action":  "append",
		"path":    testFile,
		"content": appendContent,
	}
	
	ctx := context.Background()
	result, err := executor.executeFilesystem(ctx, inputs)
	
	require.NoError(t, err)
	assert.Equal(t, testFile, result)
	
	// Verify content was appended
	final, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, initialContent+appendContent, string(final))
}

func TestWorkflowExecutor_ExecuteMemory(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	ctx := context.Background()
	
	// Test store
	inputs := map[string]interface{}{
		"action": "store",
		"key":    "test_key",
		"value":  "test_value",
	}
	
	result, err := executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "test_value", result)
	
	// Test retrieve
	inputs = map[string]interface{}{
		"action": "retrieve",
		"key":    "test_key",
	}
	
	result, err = executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "test_value", result)
	
	// Test append
	inputs = map[string]interface{}{
		"action": "append",
		"key":    "test_key",
		"value":  " appended",
	}
	
	result, err = executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "test_value appended", result)
	
	// Test delete
	inputs = map[string]interface{}{
		"action": "delete",
		"key":    "test_key",
	}
	
	result, err = executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "deleted", result)
	
	// Verify deleted
	inputs = map[string]interface{}{
		"action": "retrieve",
		"key":    "test_key",
	}
	
	result, err = executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestWorkflowExecutor_ExecuteMemory_ShorthandSyntax(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	ctx := context.Background()
	
	// Test shorthand store (no explicit action)
	inputs := map[string]interface{}{
		"key":   "shorthand_key",
		"value": "shorthand_value",
	}
	
	result, err := executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "shorthand_value", result)
	
	// Test shorthand retrieve (no explicit action)
	inputs = map[string]interface{}{
		"key": "shorthand_key",
	}
	
	result, err = executor.executeMemory(ctx, inputs)
	require.NoError(t, err)
	assert.Equal(t, "shorthand_value", result)
}

func TestWorkflowExecutor_ExecuteTime(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	ctx := context.Background()
	
	// Test default "now" action
	inputs := map[string]interface{}{}
	
	result, err := executor.executeTime(ctx, inputs)
	require.NoError(t, err)
	
	// Should return current time as string
	timeStr, ok := result.(string)
	require.True(t, ok)
	assert.NotEmpty(t, timeStr)
	
	// Test with custom format
	inputs = map[string]interface{}{
		"format": "HH:mm:ss",
	}
	
	result, err = executor.executeTime(ctx, inputs)
	require.NoError(t, err)
	
	timeStr, ok = result.(string)
	require.True(t, ok)
	// Should match HH:mm:ss format (15:04:05 in Go)
	assert.Regexp(t, `\d{2}:\d{2}:\d{2}`, timeStr)
}

func TestWorkflowExecutor_ExecuteSequentialThinking(t *testing.T) {
	mockLLM := &MockGenerator{response: "LLM response to the prompt"}
	executor := NewWorkflowExecutor(&config.Config{}, mockLLM)
	ctx := context.Background()
	
	inputs := map[string]interface{}{
		"prompt": "Analyze this data",
		"data":   "{{test_data}}",
	}
	
	variables := map[string]interface{}{
		"test_data": "sample data for analysis",
	}
	
	result, err := executor.executeSequentialThinking(ctx, inputs, variables)
	require.NoError(t, err)
	assert.Equal(t, "LLM response to the prompt", result)
}

func TestWorkflowExecutor_ResolveVariables(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	inputs := map[string]interface{}{
		"text_with_var":   "Hello {{name}}",
		"number":          42,
		"nested_var":      "Value: {{outputs.result}}",
		"multiple_vars":   "{{greeting}} {{name}}!",
	}
	
	variables := map[string]interface{}{
		"name":   "World",
		"result": "success",
		"greeting": "Hi",
	}
	
	resolved := executor.resolveVariables(inputs, variables)
	
	assert.Equal(t, "Hello World", resolved["text_with_var"])
	assert.Equal(t, 42, resolved["number"])
	assert.Equal(t, "Value: success", resolved["nested_var"])
	assert.Equal(t, "Hi World!", resolved["multiple_vars"])
}

func TestWorkflowExecutor_ResolveString(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	variables := map[string]interface{}{
		"name":        "Alice",
		"count":       5,
		"active":      true,
		"data":        map[string]interface{}{"key": "value"},
	}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello {{name}}", "Hello Alice"},
		{"Count: {{count}}", "Count: 5"},
		{"Active: {{active}}", "Active: true"},
		{"{{outputs.name}}", "Alice"},
		{"{{$name}}", "Alice"},
		{"Data: {{data}}", "Data: {\"key\":\"value\"}"},
		{"No variables", "No variables"},
		{"Multiple {{name}} and {{count}}", "Multiple Alice and 5"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := executor.resolveString(tt.input, variables)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowExecutor_Execute_FullWorkflow(t *testing.T) {
	mockLLM := &MockGenerator{response: "Thinking complete"}
	executor := NewWorkflowExecutor(&config.Config{}, mockLLM)
	
	// Create a temporary file for testing
	tmpFile := filepath.Join(os.TempDir(), "workflow_test.txt")
	defer os.Remove(tmpFile)
	
	workflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "step1",
				Name: "Store in memory",
				Tool: "memory",
				Inputs: map[string]interface{}{
					"action": "store",
					"key":    "data",
					"value":  "test content",
				},
				Outputs: map[string]string{
					"value": "stored_data",
				},
			},
			{
				ID:   "step2",
				Name: "Write to file",
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    tmpFile,
					"content": "{{stored_data}}",
				},
				Outputs: map[string]string{
					"path": "file_path",
				},
			},
			{
				ID:   "step3",
				Name: "Think about it",
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"prompt": "Analyze the file",
					"data":   "{{stored_data}}",
				},
				Outputs: map[string]string{
					"analysis": "thinking_result",
				},
			},
		},
		Variables: map[string]interface{}{
			"initial": "start",
		},
	}
	
	ctx := context.Background()
	result, err := executor.Execute(ctx, workflow)
	
	require.NoError(t, err)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	assert.Len(t, result.StepResults, 3)
	
	// Check all steps completed
	for i, stepResult := range result.StepResults {
		assert.Equal(t, types.ExecutionStatusCompleted, stepResult.Status, "Step %d should be completed", i+1)
	}
	
	// Check outputs
	assert.Equal(t, "test content", result.Outputs["stored_data"])
	assert.Equal(t, tmpFile, result.Outputs["file_path"])
	assert.Equal(t, "Thinking complete", result.Outputs["thinking_result"])
	
	// Verify file was created
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestWorkflowExecutor_Execute_StepError(t *testing.T) {
	executor := NewWorkflowExecutor(&config.Config{}, &MockGenerator{})
	
	workflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "failing_step",
				Name: "Failing Step",
				Tool: "unknown_tool", // This should cause an error
			},
		},
	}
	
	ctx := context.Background()
	result, err := executor.Execute(ctx, workflow)
	
	require.Error(t, err)
	assert.Equal(t, types.ExecutionStatusFailed, result.Status)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestConvertTimeFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HH:mm:ss", "15:04:05"},
		{"HH:mm", "15:04"},
		{"YYYY-MM-DD", "2006-01-02"},
		{"DD/MM/YYYY", "02/01/2006"},
		{"MM/DD/YYYY", "01/02/2006"},
		{"YYYY-MM-DD HH:mm:ss", "2006-01-02 15:04:05"},
		{"unknown format", "unknown format"},
		{"2006-01-02", "2006-01-02"}, // Already Go format
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertTimeFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
	}
	
	for _, tt := range tests {
		result := min(tt.a, tt.b)
		assert.Equal(t, tt.expected, result)
	}
}