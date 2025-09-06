package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWorkflowStorage implements WorkflowStorage for testing
type MockWorkflowStorage struct {
	workflows  map[string]*Workflow
	executions map[string]*Execution
	mu         sync.RWMutex
}

func NewMockWorkflowStorage() *MockWorkflowStorage {
	return &MockWorkflowStorage{
		workflows:  make(map[string]*Workflow),
		executions: make(map[string]*Execution),
	}
}

func (m *MockWorkflowStorage) SaveWorkflow(workflow *Workflow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *MockWorkflowStorage) LoadWorkflow(id string) (*Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if workflow, exists := m.workflows[id]; exists {
		return workflow, nil
	}
	return nil, ErrWorkflowNotFound
}

func (m *MockWorkflowStorage) ListWorkflows() ([]*Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	workflows := make([]*Workflow, 0, len(m.workflows))
	for _, w := range m.workflows {
		workflows = append(workflows, w)
	}
	return workflows, nil
}

func (m *MockWorkflowStorage) DeleteWorkflow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.workflows, id)
	return nil
}

func (m *MockWorkflowStorage) SaveExecution(execution *Execution) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[execution.ID] = execution
	return nil
}

func (m *MockWorkflowStorage) LoadExecution(id string) (*Execution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if execution, exists := m.executions[id]; exists {
		return execution, nil
	}
	return nil, ErrExecutionNotFound
}

func (m *MockWorkflowStorage) ListExecutions(workflowID string, limit int) ([]*Execution, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var executions []*Execution
	count := 0
	for _, exec := range m.executions {
		if exec.WorkflowID == workflowID {
			executions = append(executions, exec)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return executions, nil
}

func (m *MockWorkflowStorage) UpdateExecution(execution *Execution) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[execution.ID] = execution
	return nil
}

// MockStepExecutor implements StepExecutor for testing
type MockStepExecutor struct {
	executeFn func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error)
	results   []StepExecutionRecord
	mu        sync.Mutex
}

type StepExecutionRecord struct {
	StepID     string
	Parameters map[string]interface{}
	Context    *ExecutionContext
}

func NewMockStepExecutor() *MockStepExecutor {
	return &MockStepExecutor{
		results: make([]StepExecutionRecord, 0),
	}
}

func (m *MockStepExecutor) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
	m.mu.Lock()
	m.results = append(m.results, StepExecutionRecord{
		StepID:     step.ID,
		Parameters: step.Parameters,
		Context:    execCtx,
	})
	m.mu.Unlock()

	if m.executeFn != nil {
		return m.executeFn(ctx, step, execCtx)
	}

	// Default successful execution
	return &StepResult{
		StepID:    step.ID,
		Status:    ExecutionStatusSuccess,
		StartTime: time.Now(),
		EndTime:   timePtr(time.Now()),
		Output:    map[string]interface{}{"result": "success"},
		Retries:   0,
	}, nil
}

func (m *MockStepExecutor) ValidateStep(step *Step) error {
	return nil
}

func (m *MockStepExecutor) GetExecutionRecords() []StepExecutionRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]StepExecutionRecord{}, m.results...)
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func createTestWorkflow() *Workflow {
	return &Workflow{
		ID:          "test-workflow-1",
		Name:        "Test Workflow",
		Description: "A test workflow",
		Version:     "1.0",
		Steps: []Step{
			{
				ID:         "step-1",
				Name:       "First Step",
				Type:       StepTypeAction,
				Action:     "test-action",
				Parameters: map[string]interface{}{"key": "value"},
			},
			{
				ID:           "step-2",
				Name:         "Second Step",
				Type:         StepTypeAction,
				Action:       "test-action",
				Dependencies: []string{"step-1"},
				Parameters:   map[string]interface{}{"key": "value2"},
			},
		},
		Variables: map[string]interface{}{"var1": "value1"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestEngine(t *testing.T) (*Engine, *MockWorkflowStorage, *MockStepExecutor) {
	storage := NewMockWorkflowStorage()
	executor := NewMockStepExecutor()
	config := DefaultEngineConfig()
	config.MaxConcurrentWorkflows = 2
	config.MaxConcurrentSteps = 5
	// Use fast retry policy for tests
	config.RetryPolicy = &RetryPolicy{
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
		BackoffFactor: 1.0,
	}

	engine := NewEngine(config, storage, executor)
	require.NotNil(t, engine)

	return engine, storage, executor
}

func TestNewEngine(t *testing.T) {
	storage := NewMockWorkflowStorage()
	executor := NewMockStepExecutor()
	config := DefaultEngineConfig()

	engine := NewEngine(config, storage, executor)
	assert.NotNil(t, engine)
	assert.Equal(t, storage, engine.storage)
	assert.Equal(t, executor, engine.executor)
	assert.Equal(t, config, engine.config)
	assert.False(t, engine.running)
}

func TestEngineStartStop(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	// Test starting engine
	err := engine.Start()
	assert.NoError(t, err)
	assert.True(t, engine.running)

	// Test double start
	err = engine.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stopping engine
	err = engine.Stop()
	assert.NoError(t, err)
	assert.False(t, engine.running)

	// Test double stop
	err = engine.Stop()
	assert.NoError(t, err) // Should not error
}

func TestRegisterWorkflow(t *testing.T) {
	engine, storage, _ := createTestEngine(t)

	workflow := createTestWorkflow()

	err := engine.RegisterWorkflow(workflow)
	assert.NoError(t, err)

	// Verify workflow was stored
	stored, err := storage.LoadWorkflow(workflow.ID)
	assert.NoError(t, err)
	assert.Equal(t, workflow.ID, stored.ID)
	assert.Equal(t, workflow.Name, stored.Name)
}

func TestRegisterWorkflowValidation(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	tests := []struct {
		name     string
		workflow *Workflow
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid workflow",
			workflow: createTestWorkflow(),
			wantErr:  false,
		},
		{
			name: "workflow without steps",
			workflow: &Workflow{
				ID:          "test-workflow",
				Name:        "No Steps Workflow",
				Description: "Missing steps",
				Steps:       []Step{},
			},
			wantErr: true,
			errMsg:  "workflow must have at least one step",
		},
		{
			name: "workflow without name",
			workflow: &Workflow{
				ID:          "no-name-workflow",
				Description: "Missing name",
				Steps:       []Step{},
			},
			wantErr: true,
			errMsg:  "workflow name is required",
		},
		{
			name: "workflow with duplicate step IDs",
			workflow: &Workflow{
				ID:   "dup-steps-workflow",
				Name: "Duplicate Steps",
				Steps: []Step{
					{ID: "step-1", Name: "Step 1", Type: StepTypeAction},
					{ID: "step-1", Name: "Step 1 Duplicate", Type: StepTypeAction},
				},
			},
			wantErr: true,
			errMsg:  "duplicate step ID",
		},
		{
			name: "workflow with invalid dependency",
			workflow: &Workflow{
				ID:   "invalid-dep-workflow",
				Name: "Invalid Dependency",
				Steps: []Step{
					{
						ID:           "step-1",
						Name:         "Step 1",
						Type:         StepTypeAction,
						Dependencies: []string{"non-existent-step"},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.RegisterWorkflow(tt.workflow)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetWorkflow(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	workflow := createTestWorkflow()
	err := engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Get existing workflow
	retrieved, err := engine.GetWorkflow(workflow.ID)
	assert.NoError(t, err)
	assert.Equal(t, workflow.ID, retrieved.ID)
	assert.Equal(t, workflow.Name, retrieved.Name)

	// Get non-existent workflow
	_, err = engine.GetWorkflow("non-existent")
	assert.Error(t, err)
}

func TestListWorkflows(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	// Initially no workflows
	workflows := engine.ListWorkflows()
	assert.Empty(t, workflows)

	// Create some workflows
	for i := 1; i <= 3; i++ {
		workflow := createTestWorkflow()
		workflow.ID = fmt.Sprintf("workflow-%d", i)
		workflow.Name = fmt.Sprintf("Workflow %d", i)
		err := engine.RegisterWorkflow(workflow)
		require.NoError(t, err)
	}

	// List workflows
	workflows = engine.ListWorkflows()
	assert.Len(t, workflows, 3)
}

func TestDeleteWorkflow(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	workflow := createTestWorkflow()
	err := engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Delete workflow
	err = engine.DeleteWorkflow(workflow.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = engine.GetWorkflow(workflow.ID)
	assert.Error(t, err)

	// Delete non-existent workflow
	err = engine.DeleteWorkflow("non-existent")
	assert.NoError(t, err) // Should not error
}

func TestExecuteWorkflow(t *testing.T) {
	engine, storage, executor := createTestEngine(t)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	input := map[string]interface{}{"input_key": "input_value"}
	execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, input)
	assert.NoError(t, err)
	assert.NotNil(t, execution)
	executionID := execution.ID

	// Give some time for execution
	time.Sleep(100 * time.Millisecond)

	// Verify execution was created
	exec, err := storage.LoadExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, workflow.ID, exec.WorkflowID)
	assert.Equal(t, input, exec.Input)

	// Verify steps were executed
	records := executor.GetExecutionRecords()
	assert.Len(t, records, 2) // Both steps should have been executed
	assert.Equal(t, "step-1", records[0].StepID)
	assert.Equal(t, "step-2", records[1].StepID)
}

func TestExecuteWorkflowWithFailure(t *testing.T) {
	engine, storage, executor := createTestEngine(t)

	// Make step-2 fail
	executor.executeFn = func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
		if step.ID == "step-2" {
			return &StepResult{
				StepID:    step.ID,
				Status:    ExecutionStatusFailed,
				StartTime: time.Now(),
				EndTime:   timePtr(time.Now()),
				Error:     "simulated failure",
			}, fmt.Errorf("simulated failure")
		}
		return &StepResult{
			StepID:    step.ID,
			Status:    ExecutionStatusSuccess,
			StartTime: time.Now(),
			EndTime:   timePtr(time.Now()),
			Output:    map[string]interface{}{"result": "success"},
		}, nil
	}

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
	assert.NoError(t, err)
	executionID := execution.ID

	// Give time for execution
	time.Sleep(200 * time.Millisecond)

	// Verify execution failed
	exec, err := storage.LoadExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionStatusFailed, exec.Status)
	assert.Contains(t, exec.Error, "simulated failure")
}

func TestGetExecution(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
	require.NoError(t, err)
	executionID := execution.ID

	// Get execution
	exec2, err := engine.GetExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, executionID, exec2.ID)
	assert.Equal(t, workflow.ID, exec2.WorkflowID)

	// Get non-existent execution
	_, err = engine.GetExecution("non-existent")
	assert.Error(t, err)
}

func TestListExecutions(t *testing.T) {
	engine, _, _ := createTestEngine(t)

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow multiple times
	var executionIDs []string
	for i := 0; i < 3; i++ {
		execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
		require.NoError(t, err)
		executionIDs = append(executionIDs, execution.ID)
	}

	// List executions
	executions, err := engine.ListExecutions(workflow.ID, 10)
	assert.NoError(t, err)
	assert.Len(t, executions, 3)

	// Test with limit
	executions, err = engine.ListExecutions(workflow.ID, 2)
	assert.NoError(t, err)
	assert.Len(t, executions, 2)
}

func TestCancelExecution(t *testing.T) {
	engine, storage, executor := createTestEngine(t)

	// Make steps take some time
	executor.executeFn = func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
		select {
		case <-ctx.Done():
			return &StepResult{
				StepID:    step.ID,
				Status:    ExecutionStatusCancelled,
				StartTime: time.Now(),
				EndTime:   timePtr(time.Now()),
				Error:     "cancelled",
			}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return &StepResult{
				StepID:    step.ID,
				Status:    ExecutionStatusSuccess,
				StartTime: time.Now(),
				EndTime:   timePtr(time.Now()),
				Output:    map[string]interface{}{"result": "success"},
			}, nil
		}
	}

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
	require.NoError(t, err)
	executionID := execution.ID

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel execution
	err = engine.CancelExecution(executionID)
	assert.NoError(t, err)

	// Give time for cancellation to process
	time.Sleep(200 * time.Millisecond)

	// Verify execution was cancelled
	exec, err := storage.LoadExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionStatusCancelled, exec.Status)
}

func TestWorkflowParallelExecution(t *testing.T) {
	engine, _, executor := createTestEngine(t)

	executionOrder := make([]string, 0)
	var orderMutex sync.Mutex

	executor.executeFn = func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
		orderMutex.Lock()
		executionOrder = append(executionOrder, step.ID)
		orderMutex.Unlock()

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		return &StepResult{
			StepID:    step.ID,
			Status:    ExecutionStatusSuccess,
			StartTime: time.Now(),
			EndTime:   timePtr(time.Now()),
			Output:    map[string]interface{}{"result": "success"},
		}, nil
	}

	// Create workflow with parallel steps
	workflow := &Workflow{
		ID:   "parallel-workflow",
		Name: "Parallel Workflow",
		Steps: []Step{
			{ID: "step-1", Name: "Step 1", Type: StepTypeAction, Action: "action1"},
			{ID: "step-2", Name: "Step 2", Type: StepTypeAction, Action: "action2"},
			{ID: "step-3", Name: "Step 3", Type: StepTypeAction, Action: "action3"},
			{ID: "step-4", Name: "Step 4", Type: StepTypeAction, Action: "action4", 
				Dependencies: []string{"step-1", "step-2", "step-3"}}, // Depends on all previous
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	_, err = engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
	assert.NoError(t, err)

	// Give time for execution
	time.Sleep(200 * time.Millisecond)

	// Verify execution order - steps 1, 2, 3 should execute in parallel, then step 4
	orderMutex.Lock()
	defer orderMutex.Unlock()
	
	assert.Len(t, executionOrder, 4)
	
	// step-4 should be last
	assert.Equal(t, "step-4", executionOrder[3])
	
	// steps 1, 2, 3 should be in the first three positions (any order)
	firstThree := executionOrder[:3]
	assert.Contains(t, firstThree, "step-1")
	assert.Contains(t, firstThree, "step-2")
	assert.Contains(t, firstThree, "step-3")
}

func TestWorkflowWithRetries(t *testing.T) {
	engine, storage, executor := createTestEngine(t)

	attemptCount := 0
	executor.executeFn = func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
		attemptCount++
		if attemptCount < 3 { // Fail first 2 attempts
			return &StepResult{
				StepID:    step.ID,
				Status:    ExecutionStatusFailed,
				StartTime: time.Now(),
				EndTime:   timePtr(time.Now()),
				Error:     "simulated failure",
				Retries:   attemptCount - 1,
			}, fmt.Errorf("simulated failure")
		}
		// Succeed on 3rd attempt
		return &StepResult{
			StepID:    step.ID,
			Status:    ExecutionStatusSuccess,
			StartTime: time.Now(),
			EndTime:   timePtr(time.Now()),
			Output:    map[string]interface{}{"result": "success"},
			Retries:   attemptCount - 1,
		}, nil
	}

	// Create workflow with retry policy
	workflow := &Workflow{
		ID:   "retry-workflow",
		Name: "Retry Workflow",
		Steps: []Step{
			{
				ID:     "step-1",
				Name:   "Retry Step",
				Type:   StepTypeAction,
				Action: "failing-action",
				RetryPolicy: &RetryPolicy{
					MaxRetries:    3,
					RetryDelay:    10 * time.Millisecond,
					BackoffFactor: 1.0,
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute workflow
	execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
	require.NoError(t, err)
	executionID := execution.ID

	// Give time for execution with retries
	time.Sleep(500 * time.Millisecond)

	// Verify execution succeeded after retries
	exec, err := storage.LoadExecution(executionID)
	assert.NoError(t, err)
	assert.Equal(t, ExecutionStatusSuccess, exec.Status)
	assert.Equal(t, 3, attemptCount)
}

func TestConcurrentWorkflowExecution(t *testing.T) {
	engine, _, executor := createTestEngine(t)
	engine.config.MaxConcurrentWorkflows = 2

	executionCount := 0
	var countMutex sync.Mutex

	executor.executeFn = func(ctx context.Context, step *Step, execCtx *ExecutionContext) (*StepResult, error) {
		countMutex.Lock()
		executionCount++
		countMutex.Unlock()

		time.Sleep(50 * time.Millisecond)

		return &StepResult{
			StepID:    step.ID,
			Status:    ExecutionStatusSuccess,
			StartTime: time.Now(),
			EndTime:   timePtr(time.Now()),
			Output:    map[string]interface{}{"result": "success"},
		}, nil
	}

	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	workflow := createTestWorkflow()
	err = engine.RegisterWorkflow(workflow)
	require.NoError(t, err)

	// Execute multiple workflows concurrently
	var executionIDs []string
	for i := 0; i < 5; i++ {
		execution, err := engine.ExecuteWorkflow(context.Background(), workflow.ID, nil)
		require.NoError(t, err)
		executionIDs = append(executionIDs, execution.ID)
	}

	// Give time for executions
	time.Sleep(500 * time.Millisecond)

	// All executions should eventually complete
	countMutex.Lock()
	assert.Equal(t, 10, executionCount) // 5 workflows * 2 steps each
	countMutex.Unlock()

	// Verify all executions exist
	for _, id := range executionIDs {
		execution, err := engine.GetExecution(id)
		assert.NoError(t, err)
		assert.NotNil(t, execution)
	}
}