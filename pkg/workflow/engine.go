package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrWorkflowNotFound    = errors.New("workflow not found")
	ErrExecutionNotFound   = errors.New("execution not found")
	ErrEngineNotRunning    = errors.New("engine is not running")
	ErrEngineAlreadyRunning = errors.New("engine is already running")
	ErrInvalidWorkflow     = errors.New("invalid workflow")
	ErrExecutionFailed     = errors.New("execution failed")
	ErrCyclicDependency    = errors.New("cyclic dependency detected")
)

// Engine is the main workflow execution engine
type Engine struct {
	mu         sync.RWMutex
	workflows  map[string]*Workflow
	executions map[string]*Execution
	storage    WorkflowStorage
	executor   StepExecutor
	config     *EngineConfig
	
	// Runtime state
	running bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// EngineConfig holds configuration for the workflow engine
type EngineConfig struct {
	MaxConcurrentWorkflows int
	MaxConcurrentSteps     int
	DefaultTimeout         time.Duration
	EnableParallelism      bool
	RetryPolicy            *RetryPolicy
}

// DefaultEngineConfig returns default configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxConcurrentWorkflows: 10,
		MaxConcurrentSteps:     20,
		DefaultTimeout:         30 * time.Minute,
		EnableParallelism:      true,
		RetryPolicy: &RetryPolicy{
			MaxRetries:    3,
			RetryDelay:    5 * time.Second,
			BackoffFactor: 2.0,
		},
	}
}

// Workflow represents a workflow definition
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Steps       []Step                 `json:"steps"`
	Triggers    []Trigger              `json:"triggers"`
	Variables   map[string]interface{} `json:"variables"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Tags        []string               `json:"tags,omitempty"`
}

// Step represents a single step in a workflow
type Step struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         StepType               `json:"type"`
	Action       string                 `json:"action"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Conditions   []Condition            `json:"conditions,omitempty"`
	OnSuccess    []string               `json:"on_success,omitempty"`
	OnFailure    []string               `json:"on_failure,omitempty"`
	RetryPolicy  *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
}

// StepType defines the type of workflow step
type StepType string

const (
	StepTypeAction    StepType = "action"
	StepTypeCondition StepType = "condition"
	StepTypeLoop      StepType = "loop"
	StepTypeParallel  StepType = "parallel"
	StepTypeSubflow   StepType = "subflow"
	StepTypeWait      StepType = "wait"
)

// Condition represents a condition that must be met for a step to execute
type Condition struct {
	Type       ConditionType          `json:"type"`
	Field      string                 `json:"field"`
	Operator   string                 `json:"operator"`
	Value      interface{}            `json:"value"`
	Expression string                 `json:"expression,omitempty"`
}

// ConditionType defines types of conditions
type ConditionType string

const (
	ConditionTypeSimple     ConditionType = "simple"
	ConditionTypeExpression ConditionType = "expression"
	ConditionTypeScript     ConditionType = "script"
)

// Trigger defines when a workflow should be executed
type Trigger struct {
	Type     TriggerType            `json:"type"`
	Schedule string                 `json:"schedule,omitempty"`
	Event    string                 `json:"event,omitempty"`
	Webhook  string                 `json:"webhook,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// TriggerType defines types of workflow triggers
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeFile     TriggerType = "file"
)

// Execution represents a workflow execution instance
type Execution struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	Status     ExecutionStatus        `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Steps      map[string]*StepResult `json:"steps"`
	Error      string                 `json:"error,omitempty"`
	Context    *ExecutionContext      `json:"context"`
	mu         sync.RWMutex
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSuccess   ExecutionStatus = "success"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusPaused    ExecutionStatus = "paused"
)

// StepResult represents the result of a step execution
type StepResult struct {
	StepID    string                 `json:"step_id"`
	Status    ExecutionStatus        `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Retries   int                    `json:"retries"`
}

// ExecutionContext holds runtime context for workflow execution
type ExecutionContext struct {
	Variables map[string]interface{} `json:"variables"`
	Secrets   map[string]string      `json:"secrets,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	mu        sync.RWMutex
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// WorkflowStorage interface for workflow persistence
type WorkflowStorage interface {
	SaveWorkflow(workflow *Workflow) error
	LoadWorkflow(id string) (*Workflow, error)
	ListWorkflows() ([]*Workflow, error)
	DeleteWorkflow(id string) error
	
	SaveExecution(execution *Execution) error
	LoadExecution(id string) (*Execution, error)
	ListExecutions(workflowID string, limit int) ([]*Execution, error)
	UpdateExecution(execution *Execution) error
}

// StepExecutor interface for executing workflow steps
type StepExecutor interface {
	Execute(ctx context.Context, step *Step, context *ExecutionContext) (*StepResult, error)
}

// NewEngine creates a new workflow engine
func NewEngine(config *EngineConfig, storage WorkflowStorage, executor StepExecutor) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		workflows:  make(map[string]*Workflow),
		executions: make(map[string]*Execution),
		storage:    storage,
		executor:   executor,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Load existing workflows from storage
	if storage != nil {
		if workflows, err := storage.ListWorkflows(); err == nil {
			for _, wf := range workflows {
				engine.workflows[wf.ID] = wf
			}
		}
	}

	return engine
}

// Start starts the workflow engine
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return ErrEngineAlreadyRunning
	}

	e.running = true
	return nil
}

// Stop stops the workflow engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.running = false
	e.cancel()
	e.wg.Wait()

	return nil
}

// RegisterWorkflow registers a new workflow
func (e *Engine) RegisterWorkflow(workflow *Workflow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = time.Now()
	}
	workflow.UpdatedAt = time.Now()

	// Validate workflow
	if err := e.validateWorkflow(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Build DAG to check for cycles
	dag := e.buildDAG(workflow)
	if dag.HasCycle() {
		return ErrCyclicDependency
	}

	e.workflows[workflow.ID] = workflow

	// Persist to storage
	if e.storage != nil {
		if err := e.storage.SaveWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to save workflow: %w", err)
		}
	}

	return nil
}

// ExecuteWorkflow executes a workflow
func (e *Engine) ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (*Execution, error) {
	e.mu.RLock()
	workflow, exists := e.workflows[workflowID]
	e.mu.RUnlock()

	if !exists {
		return nil, ErrWorkflowNotFound
	}

	// Create execution instance
	execution := &Execution{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Status:     ExecutionStatusPending,
		StartTime:  time.Now(),
		Input:      input,
		Steps:      make(map[string]*StepResult),
		Context: &ExecutionContext{
			Variables: make(map[string]interface{}),
			Secrets:   make(map[string]string),
			Metadata:  make(map[string]interface{}),
		},
	}

	// Merge workflow variables with input
	for k, v := range workflow.Variables {
		execution.Context.Variables[k] = v
	}
	for k, v := range input {
		execution.Context.Variables[k] = v
	}

	// Store execution
	e.mu.Lock()
	e.executions[execution.ID] = execution
	e.mu.Unlock()

	// Save to storage
	if e.storage != nil {
		if err := e.storage.SaveExecution(execution); err != nil {
			return nil, fmt.Errorf("failed to save execution: %w", err)
		}
	}

	// Execute workflow asynchronously
	e.wg.Add(1)
	go e.executeWorkflowAsync(ctx, workflow, execution)

	return execution, nil
}

// executeWorkflowAsync executes a workflow asynchronously
func (e *Engine) executeWorkflowAsync(ctx context.Context, workflow *Workflow, execution *Execution) {
	defer e.wg.Done()

	// Update status to running
	execution.mu.Lock()
	execution.Status = ExecutionStatusRunning
	execution.mu.Unlock()

	// Build execution plan
	dag := e.buildDAG(workflow)
	plan := dag.GetExecutionPlan()

	// Execute steps according to plan
	success := true
	for _, level := range plan {
		if !e.config.EnableParallelism {
			// Sequential execution
			for _, stepID := range level {
				if err := e.executeStep(ctx, workflow, execution, stepID); err != nil {
					success = false
					execution.Error = err.Error()
					break
				}
			}
		} else {
			// Parallel execution
			if err := e.executeStepsParallel(ctx, workflow, execution, level); err != nil {
				success = false
				execution.Error = err.Error()
				break
			}
		}

		if !success {
			break
		}
	}

	// Update final status only if not already cancelled
	endTime := time.Now()
	execution.mu.Lock()
	execution.EndTime = &endTime
	// Don't overwrite cancelled status
	if execution.Status != ExecutionStatusCancelled {
		if success {
			execution.Status = ExecutionStatusSuccess
		} else {
			execution.Status = ExecutionStatusFailed
		}
	}
	execution.mu.Unlock()

	// Save final state
	if e.storage != nil {
		e.storage.UpdateExecution(execution)
	}
}

// executeStep executes a single workflow step
func (e *Engine) executeStep(ctx context.Context, workflow *Workflow, execution *Execution, stepID string) error {
	// Find step
	var step *Step
	for _, s := range workflow.Steps {
		if s.ID == stepID {
			step = &s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepID)
	}

	// Check conditions
	if !e.evaluateConditions(step.Conditions, execution.Context) {
		// Skip step if conditions not met
		return nil
	}

	// Create step result
	stepResult := &StepResult{
		StepID:    stepID,
		Status:    ExecutionStatusRunning,
		StartTime: time.Now(),
	}

	execution.mu.Lock()
	execution.Steps[stepID] = stepResult
	execution.mu.Unlock()

	// Execute with retry logic
	var err error
	retryPolicy := step.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = e.config.RetryPolicy
	}

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate retry delay
			delay := retryPolicy.RetryDelay
			for i := 1; i < attempt; i++ {
				delay = time.Duration(float64(delay) * retryPolicy.BackoffFactor)
			}
			time.Sleep(delay)
			stepResult.Retries = attempt
		}

		// Execute step
		result, execErr := e.executor.Execute(ctx, step, execution.Context)
		if execErr == nil {
			// Success
			endTime := time.Now()
			stepResult.EndTime = &endTime
			stepResult.Status = ExecutionStatusSuccess
			stepResult.Output = result.Output
			err = nil
			break
		}

		err = execErr
		stepResult.Error = err.Error()
	}

	if err != nil {
		stepResult.Status = ExecutionStatusFailed
		endTime := time.Now()
		stepResult.EndTime = &endTime
		
		// Handle failure actions
		if len(step.OnFailure) > 0 {
			for _, failStepID := range step.OnFailure {
				e.executeStep(ctx, workflow, execution, failStepID)
			}
		}
		
		return err
	}

	// Handle success actions
	if len(step.OnSuccess) > 0 {
		for _, successStepID := range step.OnSuccess {
			e.executeStep(ctx, workflow, execution, successStepID)
		}
	}

	return nil
}

// executeStepsParallel executes multiple steps in parallel
func (e *Engine) executeStepsParallel(ctx context.Context, workflow *Workflow, execution *Execution, stepIDs []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(stepIDs))

	for _, stepID := range stepIDs {
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()
			if err := e.executeStep(ctx, workflow, execution, sid); err != nil {
				errChan <- err
			}
		}(stepID)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// evaluateConditions evaluates step conditions
func (e *Engine) evaluateConditions(conditions []Condition, context *ExecutionContext) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, condition := range conditions {
		if !e.evaluateCondition(condition, context) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single condition
func (e *Engine) evaluateCondition(condition Condition, context *ExecutionContext) bool {
	context.mu.RLock()
	defer context.mu.RUnlock()

	switch condition.Type {
	case ConditionTypeSimple:
		value, exists := context.Variables[condition.Field]
		if !exists {
			return false
		}

		switch condition.Operator {
		case "==", "equals":
			return value == condition.Value
		case "!=", "not_equals":
			return value != condition.Value
		case ">", "greater_than":
			return compareValues(value, condition.Value) > 0
		case "<", "less_than":
			return compareValues(value, condition.Value) < 0
		case ">=", "greater_or_equal":
			return compareValues(value, condition.Value) >= 0
		case "<=", "less_or_equal":
			return compareValues(value, condition.Value) <= 0
		default:
			return false
		}

	case ConditionTypeExpression:
		// TODO: Implement expression evaluation
		return true

	default:
		return false
	}
}

// compareValues compares two values
func compareValues(a, b interface{}) int {
	// Simple comparison logic - can be extended
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// validateWorkflow validates a workflow definition
func (e *Engine) validateWorkflow(workflow *Workflow) error {
	if workflow.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Validate step IDs are unique
	stepIDs := make(map[string]bool)
	for _, step := range workflow.Steps {
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true
	}

	// Validate dependencies exist
	for _, step := range workflow.Steps {
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				return fmt.Errorf("step %s has invalid dependency: %s", step.ID, depID)
			}
		}
	}

	return nil
}

// buildDAG builds a DAG from workflow steps
func (e *Engine) buildDAG(workflow *Workflow) *DAG {
	dag := NewDAG()

	// Add all steps as nodes
	for _, step := range workflow.Steps {
		dag.AddNode(step.ID, &step)
	}

	// Add edges for dependencies
	for _, step := range workflow.Steps {
		for _, depID := range step.Dependencies {
			dag.AddEdge(depID, step.ID)
		}
	}

	return dag
}

// GetWorkflow retrieves a workflow by ID
func (e *Engine) GetWorkflow(id string) (*Workflow, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, exists := e.workflows[id]
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	return workflow, nil
}

// ListWorkflows lists all registered workflows
func (e *Engine) ListWorkflows() []*Workflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflows := make([]*Workflow, 0, len(e.workflows))
	for _, wf := range e.workflows {
		workflows = append(workflows, wf)
	}

	return workflows
}

// GetExecution retrieves an execution by ID
func (e *Engine) GetExecution(id string) (*Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execution, exists := e.executions[id]
	if !exists {
		// Try loading from storage
		if e.storage != nil {
			return e.storage.LoadExecution(id)
		}
		return nil, ErrExecutionNotFound
	}

	return execution, nil
}

// ListExecutions lists executions for a workflow
func (e *Engine) ListExecutions(workflowID string, limit int) ([]*Execution, error) {
	if e.storage != nil {
		return e.storage.ListExecutions(workflowID, limit)
	}

	// If no storage, return from in-memory executions
	e.mu.RLock()
	defer e.mu.RUnlock()

	var executions []*Execution
	count := 0
	for _, exec := range e.executions {
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

// DeleteWorkflow deletes a workflow
func (e *Engine) DeleteWorkflow(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove from memory
	delete(e.workflows, id)

	// Delete from storage if available
	if e.storage != nil {
		return e.storage.DeleteWorkflow(id)
	}

	return nil
}

// CancelExecution cancels a running execution
func (e *Engine) CancelExecution(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	execution, exists := e.executions[id]
	if !exists {
		return ErrExecutionNotFound
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()

	if execution.Status != ExecutionStatusRunning {
		return fmt.Errorf("execution not running")
	}

	execution.Status = ExecutionStatusCancelled
	endTime := time.Now()
	execution.EndTime = &endTime

	// Save cancellation
	if e.storage != nil {
		e.storage.UpdateExecution(execution)
	}

	return nil
}