package planner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGenerator implements domain.Generator for testing
type MockGenerator struct {
	response         string
	generationResult *domain.GenerationResult
	structuredResult *domain.StructuredResult
	error            error
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

func TestNewAgentPlanner(t *testing.T) {
	llm := &MockGenerator{}
	storageDir := "/tmp/test-planner"

	planner := NewAgentPlanner(llm, storageDir)

	assert.NotNil(t, planner)
	assert.Equal(t, llm, planner.llm)
	assert.Equal(t, storageDir, planner.storageDir)
	assert.Empty(t, planner.mcpTools)
	assert.False(t, planner.verbose)
}

func TestAgentPlanner_SetMCPTools(t *testing.T) {
	planner := NewAgentPlanner(&MockGenerator{}, "/tmp")

	tools := []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  map[string]interface{}{},
			},
		},
	}

	planner.SetMCPTools(tools)
	assert.Equal(t, tools, planner.mcpTools)
}

func TestAgentPlanner_SetVerbose(t *testing.T) {
	planner := NewAgentPlanner(&MockGenerator{}, "/tmp")

	planner.SetVerbose(true)
	assert.True(t, planner.verbose)

	planner.SetVerbose(false)
	assert.False(t, planner.verbose)
}

func TestTaskStatus_Constants(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPlanned, "planned"},
		{TaskStatusInProgress, "in_progress"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestPlanStatus_Constants(t *testing.T) {
	tests := []struct {
		status   PlanStatus
		expected string
	}{
		{PlanStatusDraft, "draft"},
		{PlanStatusReady, "ready"},
		{PlanStatusExecuting, "executing"},
		{PlanStatusCompleted, "completed"},
		{PlanStatusFailed, "failed"},
		{PlanStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestTaskPlan_Structure(t *testing.T) {
	now := time.Now()

	task := TaskPlan{
		ID:           "task-1",
		Name:         "Test Task",
		Description:  "A test task for validation",
		Status:       TaskStatusPlanned,
		Priority:     1,
		Dependencies: []string{"task-0"},
		Tools:        []string{"filesystem", "fetch"},
		Steps: []TaskStep{
			{
				ID:          "step-1",
				Action:      "read_file",
				Description: "Read a test file",
				Tool:        "filesystem",
				Parameters: map[string]interface{}{
					"path": "/test/file.txt",
				},
				Status: TaskStatusPlanned,
			},
		},
		StartedAt: &now,
		Outputs: map[string]interface{}{
			"result": "success",
		},
	}

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "Test Task", task.Name)
	assert.Equal(t, TaskStatusPlanned, task.Status)
	assert.Len(t, task.Dependencies, 1)
	assert.Len(t, task.Tools, 2)
	assert.Len(t, task.Steps, 1)
	assert.Equal(t, &now, task.StartedAt)
}

func TestAgentPlan_Structure(t *testing.T) {
	now := time.Now()

	plan := AgentPlan{
		ID:        "plan-1",
		AgentID:   "agent-1",
		Goal:      "Complete a test goal",
		CreatedAt: now,
		UpdatedAt: now,
		Status:    PlanStatusReady,
		Tasks: []TaskPlan{
			{
				ID:     "task-1",
				Name:   "First Task",
				Status: TaskStatusPlanned,
				Steps: []TaskStep{
					{ID: "step-1", Status: TaskStatusPlanned},
					{ID: "step-2", Status: TaskStatusPlanned},
				},
			},
		},
		Context:        map[string]interface{}{"key": "value"},
		Summary:        "A test plan",
		TotalSteps:     2,
		CompletedSteps: 0,
	}

	assert.Equal(t, "plan-1", plan.ID)
	assert.Equal(t, "Complete a test goal", plan.Goal)
	assert.Equal(t, PlanStatusReady, plan.Status)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, 2, plan.TotalSteps)
	assert.Equal(t, 0, plan.CompletedSteps)
}

func TestAgentPlanner_ExtractJSON(t *testing.T) {
	planner := NewAgentPlanner(&MockGenerator{}, "/tmp")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple JSON object",
			input:    `{"goal": "test", "tasks": []}`,
			expected: `{"goal": "test", "tasks": []}`,
		},
		{
			name:     "JSON with nested objects",
			input:    `{"outer": {"inner": "value"}, "array": [1, 2]}`,
			expected: `{"outer": {"inner": "value"}, "array": [1, 2]}`,
		},
		{
			name: "JSON in text with braces",
			input: `Here is the plan: {"goal": "test goal", "summary": "A test plan"}
			
			This should work well.`,
			expected: `{"goal": "test goal", "summary": "A test plan"}`,
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "No JSON",
			input:    "This is just plain text without JSON",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planner.extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgentPlanner_CreatePlan(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "planner-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Mock LLM response with valid JSON
	mockResponse := `{
		"goal": "Create a test file",
		"summary": "Simple plan to create and verify a test file",
		"tasks": [
			{
				"name": "Create File",
				"description": "Create a new test file",
				"priority": 1,
				"dependencies": [],
				"tools": ["filesystem"],
				"steps": [
					{
						"action": "write_file",
						"description": "Write content to test file",
						"tool": "filesystem",
						"parameters": {
							"path": "/tmp/test.txt",
							"content": "Hello World"
						}
					}
				]
			}
		]
	}`

	llm := &MockGenerator{response: mockResponse}
	planner := NewAgentPlanner(llm, tmpDir)

	ctx := context.Background()
	plan, err := planner.CreatePlan(ctx, "Create a test file")

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "Create a test file", plan.Goal)
	assert.Equal(t, PlanStatusReady, plan.Status)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, 1, plan.TotalSteps)

	// Verify task and step IDs were generated
	task := plan.Tasks[0]
	assert.Equal(t, "task_1", task.ID)
	assert.Equal(t, TaskStatusPlanned, task.Status)
	assert.Len(t, task.Steps, 1)
	assert.Equal(t, "step_1_1", task.Steps[0].ID)
	assert.Equal(t, TaskStatusPlanned, task.Steps[0].Status)

	// Verify plan was saved to filesystem
	planFile := filepath.Join(tmpDir, "plans", plan.ID, "plan.json")
	_, err = os.Stat(planFile)
	assert.NoError(t, err)
}

func TestAgentPlanner_SaveLoadPlan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-save-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	planner := NewAgentPlanner(&MockGenerator{}, tmpDir)

	// Create a test plan
	now := time.Now()
	plan := &AgentPlan{
		ID:        "test-plan-123",
		AgentID:   "agent-456",
		Goal:      "Test goal",
		CreatedAt: now,
		UpdatedAt: now,
		Status:    PlanStatusReady,
		Tasks: []TaskPlan{
			{
				ID:          "task-1",
				Name:        "Test Task",
				Description: "A test task",
				Status:      TaskStatusPlanned,
				Steps: []TaskStep{
					{
						ID:          "step-1",
						Action:      "test_action",
						Description: "Test step",
						Status:      TaskStatusPlanned,
					},
				},
			},
		},
		Context:        map[string]interface{}{"test": "value"},
		Summary:        "Test summary",
		TotalSteps:     1,
		CompletedSteps: 0,
	}

	// Test save
	err = planner.SavePlan(plan)
	require.NoError(t, err)

	// Test load
	loaded, err := planner.LoadPlan(plan.ID)
	require.NoError(t, err)

	assert.Equal(t, plan.ID, loaded.ID)
	assert.Equal(t, plan.Goal, loaded.Goal)
	assert.Equal(t, plan.Status, loaded.Status)
	assert.Len(t, loaded.Tasks, 1)
	assert.Equal(t, plan.Tasks[0].ID, loaded.Tasks[0].ID)
}

func TestAgentPlanner_UpdateTaskStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-update-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	planner := NewAgentPlanner(&MockGenerator{}, tmpDir)

	// Create and save initial plan
	plan := &AgentPlan{
		ID:     "update-test-plan",
		Status: PlanStatusReady,
		Tasks: []TaskPlan{
			{
				ID:     "task-1",
				Name:   "Test Task",
				Status: TaskStatusPlanned,
			},
		},
	}

	err = planner.SavePlan(plan)
	require.NoError(t, err)

	// Update task status
	err = planner.UpdateTaskStatus(plan.ID, "task-1", TaskStatusInProgress)
	require.NoError(t, err)

	// Load and verify
	updated, err := planner.LoadPlan(plan.ID)
	require.NoError(t, err)

	assert.Equal(t, TaskStatusInProgress, updated.Tasks[0].Status)
	assert.NotNil(t, updated.Tasks[0].StartedAt)
}

func TestAgentPlanner_UpdateStepStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-step-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	planner := NewAgentPlanner(&MockGenerator{}, tmpDir)

	// Create plan with steps
	plan := &AgentPlan{
		ID:     "step-test-plan",
		Status: PlanStatusExecuting,
		Tasks: []TaskPlan{
			{
				ID: "task-1",
				Steps: []TaskStep{
					{
						ID:     "step-1",
						Status: TaskStatusPlanned,
					},
					{
						ID:     "step-2",
						Status: TaskStatusPlanned,
					},
				},
			},
		},
		TotalSteps: 2,
	}

	err = planner.SavePlan(plan)
	require.NoError(t, err)

	// Complete first step
	err = planner.UpdateStepStatus(plan.ID, "task-1", "step-1", TaskStatusCompleted, "step output", "")
	require.NoError(t, err)

	// Load and verify
	updated, err := planner.LoadPlan(plan.ID)
	require.NoError(t, err)

	step1 := updated.Tasks[0].Steps[0]
	assert.Equal(t, TaskStatusCompleted, step1.Status)
	assert.Equal(t, "step output", step1.Output)
	assert.NotNil(t, step1.CompletedAt)
	assert.Equal(t, 1, updated.CompletedSteps)
}

func TestAgentPlanner_ListPlans(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-list-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	planner := NewAgentPlanner(&MockGenerator{}, tmpDir)

	// Initially no plans
	plans, err := planner.ListPlans()
	require.NoError(t, err)
	assert.Empty(t, plans)

	// Create some plans
	testPlans := []*AgentPlan{
		{ID: "plan-1", Goal: "Goal 1", Status: PlanStatusReady, TotalSteps: 1},
		{ID: "plan-2", Goal: "Goal 2", Status: PlanStatusCompleted, TotalSteps: 2},
	}

	for _, plan := range testPlans {
		err = planner.SavePlan(plan)
		require.NoError(t, err)
	}

	// List plans
	plans, err = planner.ListPlans()
	require.NoError(t, err)
	assert.Len(t, plans, 2)

	// Verify plan IDs are present
	planIDs := make(map[string]bool)
	for _, plan := range plans {
		planIDs[plan.ID] = true
	}
	assert.True(t, planIDs["plan-1"])
	assert.True(t, planIDs["plan-2"])
}

func TestAgentPlanner_UpdatePlanStatus(t *testing.T) {
	planner := NewAgentPlanner(&MockGenerator{}, "/tmp")

	tests := []struct {
		name           string
		tasks          []TaskPlan
		expectedStatus PlanStatus
	}{
		{
			name: "All completed",
			tasks: []TaskPlan{
				{Status: TaskStatusCompleted},
				{Status: TaskStatusCompleted},
			},
			expectedStatus: PlanStatusCompleted,
		},
		{
			name: "Some failed",
			tasks: []TaskPlan{
				{Status: TaskStatusCompleted},
				{Status: TaskStatusFailed},
			},
			expectedStatus: PlanStatusFailed,
		},
		{
			name: "Some in progress",
			tasks: []TaskPlan{
				{Status: TaskStatusCompleted},
				{Status: TaskStatusInProgress},
			},
			expectedStatus: PlanStatusExecuting,
		},
		{
			name: "All planned",
			tasks: []TaskPlan{
				{Status: TaskStatusPlanned},
				{Status: TaskStatusPlanned},
			},
			expectedStatus: PlanStatusReady, // No change from initial
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &AgentPlan{
				Status: PlanStatusReady,
				Tasks:  tt.tasks,
			}

			planner.updatePlanStatus(plan)
			assert.Equal(t, tt.expectedStatus, plan.Status)
		})
	}
}

func TestAgentPlanner_GeneratePlan_Error(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-error-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Mock LLM that returns an error
	llm := &MockGenerator{error: fmt.Errorf("LLM error")}
	planner := NewAgentPlanner(llm, tmpDir)

	ctx := context.Background()
	plan, err := planner.CreatePlan(ctx, "Test goal")

	require.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "failed to generate plan")
}

func TestAgentPlanner_GeneratePlan_InvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-json-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Mock LLM that returns invalid JSON
	llm := &MockGenerator{response: "This is not JSON at all"}
	planner := NewAgentPlanner(llm, tmpDir)

	ctx := context.Background()
	plan, err := planner.CreatePlan(ctx, "Test goal")

	require.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestAgentPlanner_LoadPlan_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planner-notfound-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	planner := NewAgentPlanner(&MockGenerator{}, tmpDir)

	plan, err := planner.LoadPlan("non-existent-plan")
	require.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "failed to read plan file")
}
