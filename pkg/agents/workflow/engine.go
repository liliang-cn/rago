// Package workflow implements the workflow execution engine for the Agent pillar.
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Engine manages workflow execution with state persistence and error recovery.
type Engine struct {
	mu         sync.RWMutex
	config     EngineConfig
	workflows  map[string]*Definition
	executions map[string]*ExecutionContext
	storage    StateStorage
	
	// Runtime state
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// EngineConfig holds configuration for the workflow engine.
type EngineConfig struct {
	MaxConcurrentWorkflows int
	MaxConcurrentSteps     int
	DefaultTimeout         time.Duration
	EnableParallelism      bool
	StateStoragePath       string
}

// Definition represents a workflow definition.
type Definition struct {
	ID          string
	Name        string
	Description string
	Steps       []Step
	Inputs      []core.WorkflowInput
	Outputs     []core.WorkflowOutput
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Step represents a workflow step.
type Step struct {
	ID           string
	Name         string
	Type         string
	Parameters   map[string]interface{}
	Dependencies []string
	Condition    string
}

// ExecutionContext tracks workflow execution state.
type ExecutionContext struct {
	ID         string
	WorkflowID string
	Inputs     map[string]interface{}
	Context    map[string]interface{}
	Status     ExecutionStatus
	StartedAt  time.Time
	UpdatedAt  time.Time
	
	// Step tracking
	StepStates map[string]*StepState
	Variables  map[string]interface{}
	
	// Error recovery
	RetryCount int
	LastError  error
}

// ExecutionStatus represents the workflow execution status.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// StepState tracks individual step state.
type StepState struct {
	StepID      string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt time.Time
	Output      interface{}
	Error       error
	RetryCount  int
}

// ExecutionOptions provides options for workflow execution.
type ExecutionOptions struct {
	LLMService interface{}
	RAGService interface{}
	MCPService interface{}
	Reasoning  interface{}
}

// ExecutionResult holds the workflow execution result.
type ExecutionResult struct {
	WorkflowName string
	Status       string
	Outputs      map[string]interface{}
	Steps        []StepExecutionResult
	Duration     time.Duration
	StartedAt    time.Time
	CompletedAt  time.Time
}

// StepExecutionResult holds individual step execution result.
type StepExecutionResult struct {
	ID       string
	Status   string
	Output   interface{}
	Error    string
	Duration time.Duration
}

// NewEngine creates a new workflow engine.
func NewEngine(config EngineConfig) (*Engine, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize state storage
	storage, err := NewStateStorage(config.StateStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state storage: %w", err)
	}
	
	engine := &Engine{
		config:     config,
		workflows:  make(map[string]*Definition),
		executions: make(map[string]*ExecutionContext),
		storage:    storage,
		running:    true,
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Start background workers
	engine.wg.Add(1)
	go engine.runStateManager()
	
	return engine, nil
}

// RegisterWorkflow registers a workflow definition.
func (e *Engine) RegisterWorkflow(def *Definition) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.workflows[def.ID]; exists {
		return fmt.Errorf("workflow %s already registered", def.ID)
	}
	
	e.workflows[def.ID] = def
	
	// Persist workflow definition
	if err := e.storage.SaveWorkflow(def); err != nil {
		return fmt.Errorf("failed to persist workflow: %w", err)
	}
	
	return nil
}

// UnregisterWorkflow unregisters a workflow definition.
func (e *Engine) UnregisterWorkflow(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.workflows[id]; !exists {
		return fmt.Errorf("workflow %s not found", id)
	}
	
	delete(e.workflows, id)
	
	// Remove from storage
	if err := e.storage.DeleteWorkflow(id); err != nil {
		return fmt.Errorf("failed to delete workflow from storage: %w", err)
	}
	
	return nil
}

// ExecuteWorkflow executes a workflow with the given context.
func (e *Engine) ExecuteWorkflow(ctx context.Context, def *Definition, execCtx *ExecutionContext, opts ExecutionOptions) (*ExecutionResult, error) {
	e.mu.Lock()
	e.executions[execCtx.ID] = execCtx
	e.mu.Unlock()
	
	defer func() {
		e.mu.Lock()
		delete(e.executions, execCtx.ID)
		e.mu.Unlock()
	}()
	
	// Initialize step states
	execCtx.StepStates = make(map[string]*StepState)
	for _, step := range def.Steps {
		execCtx.StepStates[step.ID] = &StepState{
			StepID: step.ID,
			Status: StatusPending,
		}
	}
	
	// Save initial state
	if err := e.storage.SaveExecutionState(execCtx); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}
	
	// Execute workflow steps
	executor := &stepExecutor{
		engine:   e,
		def:      def,
		execCtx:  execCtx,
		opts:     opts,
	}
	
	result, err := executor.Execute(ctx)
	if err != nil {
		execCtx.Status = StatusFailed
		execCtx.LastError = err
		e.storage.SaveExecutionState(execCtx)
		return nil, err
	}
	
	execCtx.Status = StatusCompleted
	e.storage.SaveExecutionState(execCtx)
	
	return result, nil
}

// RecoverExecution recovers a workflow execution from persistent state.
func (e *Engine) RecoverExecution(execID string, opts ExecutionOptions) (*ExecutionResult, error) {
	// Load execution state
	execCtx, err := e.storage.LoadExecutionState(execID)
	if err != nil {
		return nil, fmt.Errorf("failed to load execution state: %w", err)
	}
	
	// Load workflow definition
	def, err := e.storage.LoadWorkflow(execCtx.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow definition: %w", err)
	}
	
	// Resume execution
	return e.ExecuteWorkflow(context.Background(), def, execCtx, opts)
}

// Stop stops the workflow engine.
func (e *Engine) Stop() error {
	e.running = false
	e.cancel()
	e.wg.Wait()
	
	// Close storage
	if e.storage != nil {
		return e.storage.Close()
	}
	
	return nil
}

// runStateManager manages workflow state persistence.
func (e *Engine) runStateManager() {
	defer e.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.persistActiveStates()
		}
	}
}

// persistActiveStates persists all active execution states.
func (e *Engine) persistActiveStates() {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	for _, execCtx := range e.executions {
		if execCtx.Status == StatusRunning {
			if err := e.storage.SaveExecutionState(execCtx); err != nil {
				fmt.Printf("Failed to persist execution state %s: %v\n", execCtx.ID, err)
			}
		}
	}
}

// stepExecutor handles the execution of workflow steps.
type stepExecutor struct {
	engine  *Engine
	def     *Definition
	execCtx *ExecutionContext
	opts    ExecutionOptions
}

// Execute executes all workflow steps.
func (se *stepExecutor) Execute(ctx context.Context) (*ExecutionResult, error) {
	// Build execution order based on dependencies
	order, err := se.buildExecutionOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to build execution order: %w", err)
	}
	
	// Execute steps in order
	stepResults := make([]StepExecutionResult, 0)
	
	for _, stepID := range order {
		step := se.findStep(stepID)
		if step == nil {
			return nil, fmt.Errorf("step %s not found", stepID)
		}
		
		// Check if step should be executed based on condition
		if step.Condition != "" {
			shouldExecute, err := se.evaluateCondition(step.Condition)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate condition for step %s: %w", stepID, err)
			}
			if !shouldExecute {
				continue
			}
		}
		
		// Execute step
		result, err := se.executeStep(ctx, step)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", stepID, err)
		}
		
		stepResults = append(stepResults, result)
		
		// Update variables with step output
		if result.Output != nil {
			se.execCtx.Variables[stepID] = result.Output
		}
	}
	
	// Build final result
	return &ExecutionResult{
		WorkflowName: se.def.Name,
		Status:       string(StatusCompleted),
		Outputs:      se.execCtx.Variables,
		Steps:        stepResults,
		Duration:     time.Since(se.execCtx.StartedAt),
		StartedAt:    se.execCtx.StartedAt,
		CompletedAt:  time.Now(),
	}, nil
}

// buildExecutionOrder builds the execution order based on dependencies.
func (se *stepExecutor) buildExecutionOrder() ([]string, error) {
	// Topological sort of steps based on dependencies
	visited := make(map[string]bool)
	order := make([]string, 0)
	
	var visit func(stepID string) error
	visit = func(stepID string) error {
		if visited[stepID] {
			return nil
		}
		
		step := se.findStep(stepID)
		if step == nil {
			return fmt.Errorf("step %s not found", stepID)
		}
		
		// Visit dependencies first
		for _, dep := range step.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		
		visited[stepID] = true
		order = append(order, stepID)
		return nil
	}
	
	// Visit all steps
	for _, step := range se.def.Steps {
		if err := visit(step.ID); err != nil {
			return nil, err
		}
	}
	
	return order, nil
}

// executeStep executes a single workflow step.
func (se *stepExecutor) executeStep(ctx context.Context, step *Step) (StepExecutionResult, error) {
	state := se.execCtx.StepStates[step.ID]
	state.Status = StatusRunning
	state.StartedAt = time.Now()
	
	// Execute based on step type
	var output interface{}
	var err error
	
	switch step.Type {
	case "llm":
		output, err = se.executeLLMStep(ctx, step)
	case "rag":
		output, err = se.executeRAGStep(ctx, step)
	case "mcp":
		output, err = se.executeMCPStep(ctx, step)
	case "agent":
		output, err = se.executeAgentStep(ctx, step)
	default:
		err = fmt.Errorf("unsupported step type: %s", step.Type)
	}
	
	state.CompletedAt = time.Now()
	
	if err != nil {
		state.Status = StatusFailed
		state.Error = err
		return StepExecutionResult{
			ID:       step.ID,
			Status:   string(StatusFailed),
			Error:    err.Error(),
			Duration: state.CompletedAt.Sub(state.StartedAt),
		}, err
	}
	
	state.Status = StatusCompleted
	state.Output = output
	
	return StepExecutionResult{
		ID:       step.ID,
		Status:   string(StatusCompleted),
		Output:   output,
		Duration: state.CompletedAt.Sub(state.StartedAt),
	}, nil
}

// executeLLMStep executes an LLM step.
func (se *stepExecutor) executeLLMStep(ctx context.Context, step *Step) (interface{}, error) {
	// Extract prompt from parameters
	prompt, ok := step.Parameters["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt parameter required for LLM step")
	}
	
	// Substitute variables in prompt
	prompt = se.substituteVariables(prompt)
	
	// Call LLM service if available
	if se.opts.LLMService != nil {
		// This would call the actual LLM service
		// For now, return a placeholder
		return fmt.Sprintf("LLM response for: %s", prompt), nil
	}
	
	return "LLM service not available", nil
}

// executeRAGStep executes a RAG step.
func (se *stepExecutor) executeRAGStep(ctx context.Context, step *Step) (interface{}, error) {
	// Extract query from parameters
	query, ok := step.Parameters["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter required for RAG step")
	}
	
	// Substitute variables in query
	query = se.substituteVariables(query)
	
	// Call RAG service if available
	if se.opts.RAGService != nil {
		// This would call the actual RAG service
		// For now, return a placeholder
		return fmt.Sprintf("RAG results for: %s", query), nil
	}
	
	return "RAG service not available", nil
}

// executeMCPStep executes an MCP step.
func (se *stepExecutor) executeMCPStep(ctx context.Context, step *Step) (interface{}, error) {
	// Extract tool and parameters
	tool, ok := step.Parameters["tool"].(string)
	if !ok {
		return nil, fmt.Errorf("tool parameter required for MCP step")
	}
	
	// Call MCP service if available
	if se.opts.MCPService != nil {
		// This would call the actual MCP service
		// For now, return a placeholder
		return fmt.Sprintf("MCP tool %s executed", tool), nil
	}
	
	return "MCP service not available", nil
}

// executeAgentStep executes an agent step.
func (se *stepExecutor) executeAgentStep(ctx context.Context, step *Step) (interface{}, error) {
	// Extract agent name and task
	agentName, ok := step.Parameters["agent"].(string)
	if !ok {
		return nil, fmt.Errorf("agent parameter required for agent step")
	}
	
	task, ok := step.Parameters["task"].(string)
	if !ok {
		return nil, fmt.Errorf("task parameter required for agent step")
	}
	
	// Substitute variables in task
	task = se.substituteVariables(task)
	
	// This would call an agent recursively
	// For now, return a placeholder
	return fmt.Sprintf("Agent %s executed task: %s", agentName, task), nil
}

// findStep finds a step by ID.
func (se *stepExecutor) findStep(id string) *Step {
	for i := range se.def.Steps {
		if se.def.Steps[i].ID == id {
			return &se.def.Steps[i]
		}
	}
	return nil
}

// evaluateCondition evaluates a step condition.
func (se *stepExecutor) evaluateCondition(condition string) (bool, error) {
	// Simple condition evaluation
	// In a real implementation, this would use a proper expression evaluator
	if condition == "true" {
		return true, nil
	}
	if condition == "false" {
		return false, nil
	}
	
	// Check if it's a variable reference
	if val, ok := se.execCtx.Variables[condition]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal, nil
		}
	}
	
	// Default to true for now
	return true, nil
}

// substituteVariables substitutes variables in a string.
func (se *stepExecutor) substituteVariables(str string) string {
	// Simple variable substitution
	// In a real implementation, this would use proper templating
	result := str
	for key, val := range se.execCtx.Variables {
		placeholder := fmt.Sprintf("${%s}", key)
		result = replaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}
	return result
}

// replaceAll replaces all occurrences of old with new in s.
func replaceAll(s, old, new string) string {
	// Simple string replacement
	// In Go 1.12+, we would use strings.ReplaceAll
	for {
		i := stringIndex(s, old)
		if i == -1 {
			break
		}
		s = s[:i] + new + s[i+len(old):]
	}
	return s
}

// stringIndex returns the index of substr in s, or -1 if not found.
func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}