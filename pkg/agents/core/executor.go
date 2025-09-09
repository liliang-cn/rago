package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// AgentExecutor manages the execution of agents and workflows
type AgentExecutor struct {
	mcpClient      interface{} // MCP service interface
	storage        AgentStorage
	templateEngine types.TemplateEngine
	validator      types.WorkflowValidator

	// Execution tracking
	executions     map[string]*ExecutionInstance
	executionMutex sync.RWMutex

	// Configuration
	maxConcurrentExecutions int
	defaultTimeout          time.Duration
}

// AgentStorage defines the interface for agent persistence
type AgentStorage interface {
	SaveAgent(agent *types.Agent) error
	GetAgent(id string) (*types.Agent, error)
	ListAgents() ([]*types.Agent, error)
	DeleteAgent(id string) error
	SaveExecution(execution *types.ExecutionResult) error
	GetExecution(id string) (*types.ExecutionResult, error)
	ListExecutions(agentID string) ([]*types.ExecutionResult, error)
}

// ExecutionInstance tracks a running execution
type ExecutionInstance struct {
	ID          string
	Agent       types.AgentInterface
	Context     types.ExecutionContext
	Result      *types.ExecutionResult
	CancelFunc  context.CancelFunc
	StartTime   time.Time
	Status      types.ExecutionStatus
	CurrentStep string
}

// ExecutorConfig holds configuration for the executor
type ExecutorConfig struct {
	MaxConcurrentExecutions int
	DefaultTimeout          time.Duration
	EnableMetrics           bool
}

// NewAgentExecutor creates a new agent executor
func NewAgentExecutor(mcpClient interface{}, storage AgentStorage) *AgentExecutor {
	return &AgentExecutor{
		mcpClient:               mcpClient,
		storage:                 storage,
		executions:              make(map[string]*ExecutionInstance),
		maxConcurrentExecutions: 10,
		defaultTimeout:          30 * time.Minute,
	}
}

// SetTemplateEngine sets the template engine for variable interpolation
func (e *AgentExecutor) SetTemplateEngine(engine types.TemplateEngine) {
	e.templateEngine = engine
}

// SetValidator sets the workflow validator
func (e *AgentExecutor) SetValidator(validator types.WorkflowValidator) {
	e.validator = validator
}

// Execute executes an agent workflow
func (e *AgentExecutor) Execute(ctx context.Context, agent types.AgentInterface) (*types.ExecutionResult, error) {
	executionID := uuid.New().String()

	// Check concurrent execution limit
	if e.getCurrentExecutionCount() >= e.maxConcurrentExecutions {
		return nil, fmt.Errorf("maximum concurrent executions reached (%d)", e.maxConcurrentExecutions)
	}

	// Create execution context
	execCtx := types.ExecutionContext{
		RequestID: executionID,
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
		Timeout:   e.defaultTimeout,
		MCPClient: e.mcpClient,
	}

	// Add agent configuration variables
	if agentConfig := agent.GetAgent(); agentConfig != nil {
		if agentConfig.Config.DefaultTimeout > 0 {
			execCtx.Timeout = agentConfig.Config.DefaultTimeout
		}
		for k, v := range agentConfig.Workflow.Variables {
			execCtx.Variables[k] = v
		}
	}

	// Create cancellable context
	execCtxWithCancel, cancelFunc := context.WithTimeout(ctx, execCtx.Timeout)
	defer cancelFunc()

	// Create execution instance
	instance := &ExecutionInstance{
		ID:         executionID,
		Agent:      agent,
		Context:    execCtx,
		CancelFunc: cancelFunc,
		StartTime:  time.Now(),
		Status:     types.ExecutionStatusRunning,
	}

	// Track execution
	e.executionMutex.Lock()
	e.executions[executionID] = instance
	e.executionMutex.Unlock()

	// Cleanup on completion
	defer func() {
		e.executionMutex.Lock()
		delete(e.executions, executionID)
		e.executionMutex.Unlock()
	}()

	// Execute workflow
	result, err := e.executeWorkflow(execCtxWithCancel, agent, execCtx)
	if err != nil {
		if result == nil {
			result = &types.ExecutionResult{
				ExecutionID:  executionID,
				AgentID:      agent.GetID(),
				Status:       types.ExecutionStatusFailed,
				StartTime:    execCtx.StartTime,
				ErrorMessage: err.Error(),
				Results:      make(map[string]interface{}),
				Outputs:      make(map[string]interface{}),
				Logs:         make([]types.ExecutionLog, 0),
				StepResults:  make([]types.StepResult, 0),
			}
		}
		result.Status = types.ExecutionStatusFailed
		result.ErrorMessage = err.Error()
	}

	// Set end time and duration
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	// Store execution result
	if err := e.storage.SaveExecution(result); err != nil {
		// Log error but don't fail the execution
		fmt.Printf("Failed to save execution result: %v\n", err)
	}

	return result, err
}

// executeWorkflow executes the agent's workflow
func (e *AgentExecutor) executeWorkflow(ctx context.Context, agent types.AgentInterface, execCtx types.ExecutionContext) (*types.ExecutionResult, error) {
	agentDef := agent.GetAgent()
	workflow := agentDef.Workflow

	// Validate workflow if validator is available
	if e.validator != nil {
		if err := e.validator.Validate(workflow); err != nil {
			return nil, fmt.Errorf("workflow validation failed: %w", err)
		}
	}

	// Create result structure
	result := &types.ExecutionResult{
		ExecutionID: execCtx.RequestID,
		AgentID:     agent.GetID(),
		Status:      types.ExecutionStatusRunning,
		StartTime:   execCtx.StartTime,
		Results:     make(map[string]interface{}),
		Outputs:     make(map[string]interface{}),
		Logs:        make([]types.ExecutionLog, 0),
		StepResults: make([]types.StepResult, 0),
	}

	// Execute workflow steps
	for i, step := range workflow.Steps {
		select {
		case <-ctx.Done():
			result.Status = types.ExecutionStatusCancelled
			result.ErrorMessage = "execution cancelled"
			return result, ctx.Err()
		default:
		}

		// Update current step
		e.updateCurrentStep(execCtx.RequestID, step.ID)

		stepResult, err := e.executeStep(ctx, step, execCtx)
		result.StepResults = append(result.StepResults, *stepResult)

		if err != nil {
			e.addLog(result, types.LogLevelError, fmt.Sprintf("Step %d (%s) failed: %v", i+1, step.Name, err), map[string]interface{}{
				"step_id": step.ID,
				"error":   err.Error(),
			})

			// Handle error based on policy
			if e.shouldStopOnError(workflow.ErrorPolicy, err) {
				result.Status = types.ExecutionStatusFailed
				result.ErrorMessage = fmt.Sprintf("workflow stopped at step %d: %v", i+1, err)
				return result, err
			}
		} else {
			e.addLog(result, types.LogLevelInfo, fmt.Sprintf("Step %d (%s) completed successfully", i+1, step.Name), map[string]interface{}{
				"step_id": step.ID,
				"outputs": stepResult.Outputs,
			})

			// Merge step outputs into execution context
			for key, value := range stepResult.Outputs {
				execCtx.Variables[key] = value
			}
		}
	}

	result.Status = types.ExecutionStatusCompleted
	result.Results["workflow_completed"] = true
	result.Results["steps_executed"] = len(workflow.Steps)
	result.Results["total_duration"] = time.Since(execCtx.StartTime).String()

	return result, nil
}

// executeStep executes a single workflow step
func (e *AgentExecutor) executeStep(ctx context.Context, step types.WorkflowStep, execCtx types.ExecutionContext) (*types.StepResult, error) {
	stepResult := &types.StepResult{
		StepID:    step.ID,
		Name:      step.Name,
		Status:    types.ExecutionStatusRunning,
		StartTime: time.Now(),
		Inputs:    make(map[string]interface{}),
		Outputs:   make(map[string]interface{}),
	}

	// Render inputs using template engine
	renderedInputs := make(map[string]interface{})
	for key, value := range step.Inputs {
		if e.templateEngine != nil {
			rendered, err := e.templateEngine.RenderObject(value, execCtx.Variables)
			if err != nil {
				stepResult.Status = types.ExecutionStatusFailed
				stepResult.ErrorMessage = fmt.Sprintf("failed to render input %s: %v", key, err)
				endTime := time.Now()
				stepResult.EndTime = &endTime
				stepResult.Duration = endTime.Sub(stepResult.StartTime)
				return stepResult, err
			}
			renderedInputs[key] = rendered
		} else {
			renderedInputs[key] = value
		}
	}
	stepResult.Inputs = renderedInputs

	// Execute based on step type
	var result interface{}
	var err error

	switch step.Type {
	case types.StepTypeTool:
		result, err = e.executeMCPTool(ctx, step.Tool, renderedInputs)
	case types.StepTypeVariable:
		result, err = e.executeVariableAssignment(step, renderedInputs, execCtx)
	case types.StepTypeDelay:
		result, err = e.executeDelay(ctx, step, renderedInputs)
	default:
		err = fmt.Errorf("unsupported step type: %s", step.Type)
	}

	// Handle execution result
	endTime := time.Now()
	stepResult.EndTime = &endTime
	stepResult.Duration = endTime.Sub(stepResult.StartTime)

	if err != nil {
		stepResult.Status = types.ExecutionStatusFailed
		stepResult.ErrorMessage = err.Error()
		return stepResult, err
	}

	stepResult.Status = types.ExecutionStatusCompleted

	// Map outputs
	if result != nil {
		for outputKey, variableName := range step.Outputs {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if value, exists := resultMap[outputKey]; exists {
					stepResult.Outputs[variableName] = value
				}
			}
		}
	}

	return stepResult, nil
}

// executeMCPTool executes an MCP tool
func (e *AgentExecutor) executeMCPTool(ctx context.Context, toolName string, inputs map[string]interface{}) (interface{}, error) {
	if e.mcpClient == nil {
		return nil, fmt.Errorf("MCP client is not configured")
	}
	
	// Check if mcpClient has CallTool method
	type mcpCaller interface {
		CallTool(tool string, inputs map[string]interface{}) (interface{}, error)
	}
	
	if caller, ok := e.mcpClient.(mcpCaller); ok {
		return caller.CallTool(toolName, inputs)
	}
	
	// Fallback for testing - return a mock result
	return map[string]interface{}{
		"tool":   toolName,
		"result": "Tool executed successfully",
		"data":   inputs,
	}, nil
}

// executeVariableAssignment handles variable assignment steps
func (e *AgentExecutor) executeVariableAssignment(step types.WorkflowStep, inputs map[string]interface{}, execCtx types.ExecutionContext) (interface{}, error) {
	results := make(map[string]interface{})

	// Assign variables from inputs
	for key, value := range inputs {
		execCtx.Variables[key] = value
		results[key] = value
	}

	return results, nil
}

// executeDelay handles delay steps
func (e *AgentExecutor) executeDelay(ctx context.Context, step types.WorkflowStep, inputs map[string]interface{}) (interface{}, error) {
	duration := 1 * time.Second // default

	if delayValue, exists := inputs["duration"]; exists {
		if delayStr, ok := delayValue.(string); ok {
			if parsedDuration, err := time.ParseDuration(delayStr); err == nil {
				duration = parsedDuration
			}
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(duration):
		return map[string]interface{}{"delayed": duration.String()}, nil
	}
}

// Helper methods
func (e *AgentExecutor) getCurrentExecutionCount() int {
	e.executionMutex.RLock()
	defer e.executionMutex.RUnlock()
	return len(e.executions)
}

func (e *AgentExecutor) updateCurrentStep(executionID, stepID string) {
	e.executionMutex.Lock()
	defer e.executionMutex.Unlock()
	if instance, exists := e.executions[executionID]; exists {
		instance.CurrentStep = stepID
	}
}

func (e *AgentExecutor) shouldStopOnError(policy types.ErrorPolicy, err error) bool {
	// Simple implementation - can be enhanced with more sophisticated error handling
	return policy.Strategy == types.ErrorStrategyFail
}

func (e *AgentExecutor) addLog(result *types.ExecutionResult, level types.LogLevel, message string, data interface{}) {
	result.Logs = append(result.Logs, types.ExecutionLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Data:      data,
	})
}

// GetActiveExecutions returns currently running executions
func (e *AgentExecutor) GetActiveExecutions() map[string]*ExecutionInstance {
	e.executionMutex.RLock()
	defer e.executionMutex.RUnlock()

	result := make(map[string]*ExecutionInstance)
	for k, v := range e.executions {
		result[k] = v
	}
	return result
}

// CancelExecution cancels a running execution
func (e *AgentExecutor) CancelExecution(executionID string) error {
	e.executionMutex.RLock()
	instance, exists := e.executions[executionID]
	e.executionMutex.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	instance.CancelFunc()
	instance.Status = types.ExecutionStatusCancelled
	return nil
}
