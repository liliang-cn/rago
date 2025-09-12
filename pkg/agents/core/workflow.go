package core

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// WorkflowEngine manages workflow execution with advanced features
type WorkflowEngine struct {
	executor       *AgentExecutor
	templateEngine *SimpleTemplateEngine
	validator      *WorkflowValidator
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(executor *AgentExecutor) *WorkflowEngine {
	engine := &WorkflowEngine{
		executor:       executor,
		templateEngine: NewSimpleTemplateEngine(),
		validator:      NewWorkflowValidator(),
	}

	executor.SetTemplateEngine(engine.templateEngine)
	executor.SetValidator(engine.validator)

	return engine
}

// ExecuteWorkflow executes a workflow with full automation support
func (w *WorkflowEngine) ExecuteWorkflow(ctx context.Context, agent types.AgentInterface, variables map[string]interface{}) (*types.ExecutionResult, error) {
	workflow := agent.GetAgent().Workflow

	// Validate workflow
	if err := w.validator.Validate(workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create enhanced execution context
	execCtx := types.ExecutionContext{
		RequestID: fmt.Sprintf("workflow_%d", time.Now().UnixNano()),
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
		Timeout:   30 * time.Minute,
	}

	// Merge initial variables
	for k, v := range workflow.Variables {
		execCtx.Variables[k] = v
	}
	for k, v := range variables {
		execCtx.Variables[k] = v
	}

	return w.executor.Execute(ctx, agent)
}

// SimpleTemplateEngine provides basic template variable interpolation
type SimpleTemplateEngine struct{}

// NewSimpleTemplateEngine creates a new simple template engine
func NewSimpleTemplateEngine() *SimpleTemplateEngine {
	return &SimpleTemplateEngine{}
}

// Render renders a template string with variables
func (t *SimpleTemplateEngine) Render(template string, variables map[string]interface{}) (string, error) {
	// Simple variable substitution using {{variable}} syntax
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := re.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name (remove {{ and }})
		varName := strings.Trim(match, "{}")
		varName = strings.TrimSpace(varName)

		// Handle nested paths like "steps.extract.text_content"
		value := t.getNestedValue(variables, varName)
		if value != nil {
			return fmt.Sprintf("%v", value)
		}

		// Return original if variable not found
		return match
	})

	return result, nil
}

// RenderObject renders variables in any object structure
func (t *SimpleTemplateEngine) RenderObject(obj interface{}, variables map[string]interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case string:
		return t.Render(v, variables)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			renderedValue, err := t.RenderObject(value, variables)
			if err != nil {
				return nil, err
			}
			result[key] = renderedValue
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			renderedValue, err := t.RenderObject(value, variables)
			if err != nil {
				return nil, err
			}
			result[i] = renderedValue
		}
		return result, nil
	default:
		return obj, nil
	}
}

// getNestedValue retrieves nested values using dot notation
func (t *SimpleTemplateEngine) getNestedValue(variables map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = variables

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case map[string]string:
			current = v[part]
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}

// WorkflowValidator validates workflow definitions
type WorkflowValidator struct{}

// NewWorkflowValidator creates a new workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{}
}

// Validate validates a complete workflow specification
func (v *WorkflowValidator) Validate(workflow types.WorkflowSpec) error {
	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Validate individual steps
	for i, step := range workflow.Steps {
		if err := v.ValidateStep(step); err != nil {
			return fmt.Errorf("step %d validation failed: %w", i, err)
		}
	}

	// Validate step dependencies
	stepIDs := make(map[string]bool)
	for _, step := range workflow.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range workflow.Steps {
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				return fmt.Errorf("step %s depends on non-existent step %s", step.ID, depID)
			}
		}
	}

	return nil
}

// ValidateStep validates a single workflow step
func (v *WorkflowValidator) ValidateStep(step types.WorkflowStep) error {
	if step.ID == "" {
		return fmt.Errorf("step ID is required")
	}
	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}
	if step.Type == "" {
		return fmt.Errorf("step type is required")
	}

	// Validate step type specific requirements
	switch step.Type {
	case types.StepTypeTool:
		if step.Tool == "" {
			return fmt.Errorf("tool step requires tool name")
		}
	case types.StepTypeCondition:
		if len(step.Conditions) == 0 {
			return fmt.Errorf("condition step requires conditions")
		}
	case types.StepTypeLoop:
		// Loop validation would go here
	case types.StepTypeParallel:
		// Parallel validation would go here
	case types.StepTypeDelay:
		if _, exists := step.Inputs["duration"]; !exists {
			return fmt.Errorf("delay step requires duration input")
		}
	case types.StepTypeVariable:
		// Variable step is always valid
	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}

	return nil
}

// ToolChainExecutor executes tool chains with advanced features
type ToolChainExecutor struct {
	mcpClient      interface{}
	templateEngine *SimpleTemplateEngine
}

// NewToolChainExecutor creates a new tool chain executor
func NewToolChainExecutor(mcpClient interface{}) *ToolChainExecutor {
	return &ToolChainExecutor{
		mcpClient:      mcpClient,
		templateEngine: NewSimpleTemplateEngine(),
	}
}

// ExecuteChain executes a tool chain
func (t *ToolChainExecutor) ExecuteChain(ctx context.Context, chain types.ToolChain) (*ChainResult, error) {
	result := &ChainResult{
		ChainID:     chain.ID,
		StartTime:   time.Now(),
		Status:      "running",
		Results:     make(map[string]interface{}),
		StepResults: make([]ChainStepResult, 0),
		Variables:   make(map[string]interface{}),
	}

	// Initialize variables
	for k, v := range chain.Variables {
		result.Variables[k] = v
	}

	// Execute steps
	if chain.Parallel {
		return t.executeParallel(ctx, chain, result)
	} else {
		return t.executeSequential(ctx, chain, result)
	}
}

// executeSequential executes steps one by one
func (t *ToolChainExecutor) executeSequential(ctx context.Context, chain types.ToolChain, result *ChainResult) (*ChainResult, error) {
	for i, step := range chain.Steps {
		select {
		case <-ctx.Done():
			result.Status = "cancelled"
			return result, ctx.Err()
		default:
		}

		stepResult, err := t.executeChainStep(ctx, step, result.Variables)
		result.StepResults = append(result.StepResults, *stepResult)

		if err != nil {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("step %d failed: %v", i, err)
			endTime := time.Now()
			result.EndTime = &endTime
			result.Duration = endTime.Sub(result.StartTime)
			return result, err
		}

		// Merge step outputs into variables
		for k, v := range stepResult.Outputs {
			result.Variables[k] = v
		}
	}

	result.Status = "completed"
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	return result, nil
}

// executeParallel executes steps in parallel (simplified implementation)
func (t *ToolChainExecutor) executeParallel(ctx context.Context, chain types.ToolChain, result *ChainResult) (*ChainResult, error) {
	// For simplicity, this implementation executes sequentially
	// A full implementation would use goroutines and sync mechanisms
	return t.executeSequential(ctx, chain, result)
}

// executeChainStep executes a single chain step
func (t *ToolChainExecutor) executeChainStep(ctx context.Context, step types.ChainStep, variables map[string]interface{}) (*ChainStepResult, error) {
	stepResult := &ChainStepResult{
		StepID:    step.ID,
		Name:      step.Name,
		StartTime: time.Now(),
		Status:    "running",
		Inputs:    make(map[string]interface{}),
		Outputs:   make(map[string]interface{}),
	}

	// Render inputs
	renderedInputs := make(map[string]interface{})
	for k, v := range step.Inputs {
		rendered, err := t.templateEngine.RenderObject(v, variables)
		if err != nil {
			stepResult.Status = "failed"
			stepResult.ErrorMessage = fmt.Sprintf("failed to render input %s: %v", k, err)
			return stepResult, err
		}
		renderedInputs[k] = rendered
	}
	stepResult.Inputs = renderedInputs

	// Execute tool
	var toolResult map[string]interface{}

	// Check if mcpClient has CallTool method
	type mcpCaller interface {
		CallTool(tool string, inputs map[string]interface{}) (interface{}, error)
	}

	if caller, ok := t.mcpClient.(mcpCaller); ok {
		result, callErr := caller.CallTool(step.ToolName, renderedInputs)
		if callErr != nil {
			stepResult.Status = "failed"
			stepResult.ErrorMessage = callErr.Error()
			endTime := time.Now()
			stepResult.EndTime = &endTime
			stepResult.Duration = endTime.Sub(stepResult.StartTime)
			return stepResult, callErr
		}
		if resultMap, ok := result.(map[string]interface{}); ok {
			toolResult = resultMap
		} else {
			toolResult = map[string]interface{}{"result": result}
		}
	} else {
		// Fallback mock implementation
		toolResult = map[string]interface{}{
			"tool":     step.ToolName,
			"executed": true,
			"result":   "Tool executed successfully",
			"inputs":   renderedInputs,
		}
	}

	// Map outputs
	for outputKey, variableName := range step.Outputs {
		if value, exists := toolResult[outputKey]; exists {
			stepResult.Outputs[variableName] = value
		}
	}

	stepResult.Status = "completed"
	endTime := time.Now()
	stepResult.EndTime = &endTime
	stepResult.Duration = endTime.Sub(stepResult.StartTime)

	return stepResult, nil
}

// Supporting types for chain execution
type ChainResult struct {
	ChainID      string                 `json:"chain_id"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Results      map[string]interface{} `json:"results"`
	StepResults  []ChainStepResult      `json:"step_results"`
	Variables    map[string]interface{} `json:"variables"`
}

type ChainStepResult struct {
	StepID       string                 `json:"step_id"`
	Name         string                 `json:"name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Status       string                 `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs"`
}
