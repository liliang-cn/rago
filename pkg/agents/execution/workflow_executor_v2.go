package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// WorkflowExecutorV2 handles workflow execution with async and parallel support
type WorkflowExecutorV2 struct {
	config      *core.Config
	llmProvider core.Generator
	memory      map[string]interface{}
	verbose     bool
	mu          sync.RWMutex
}

// NewWorkflowExecutorV2 creates a new workflow executor with async support
func NewWorkflowExecutorV2(cfg *core.Config, llm core.Generator) *WorkflowExecutorV2 {
	return &WorkflowExecutorV2{
		config:      cfg,
		llmProvider: llm,
		memory:      make(map[string]interface{}),
		verbose:     false,
	}
}

// SetVerbose enables verbose output
func (e *WorkflowExecutorV2) SetVerbose(v bool) {
	e.verbose = v
}

// Execute runs a workflow with async and parallel support
func (e *WorkflowExecutorV2) Execute(ctx context.Context, workflow *types.WorkflowSpec) (*types.ExecutionResult, error) {
	result := &types.ExecutionResult{
		ExecutionID: fmt.Sprintf("exec_%d", time.Now().Unix()),
		Status:      types.ExecutionStatusRunning,
		StartTime:   time.Now(),
		Outputs:     make(map[string]interface{}),
		StepResults: make([]types.StepResult, 0),
	}

	// Initialize variables
	variables := make(map[string]interface{})
	for k, v := range workflow.Variables {
		variables[k] = v
	}

	// Build dependency graph
	stepDeps := e.buildDependencyGraph(workflow)

	// Execute steps with dependency resolution
	completed := make(map[string]bool)
	results := make(map[string]interface{})
	var wg sync.WaitGroup
	errChan := make(chan error, len(workflow.Steps))
	resultChan := make(chan types.StepResult, len(workflow.Steps))

	for len(completed) < len(workflow.Steps) {
		// Find steps that can be executed (all deps satisfied)
		readySteps := e.findReadySteps(workflow.Steps, stepDeps, completed)

		if len(readySteps) == 0 && len(completed) < len(workflow.Steps) {
			return result, fmt.Errorf("circular dependency detected or no executable steps")
		}

		// Execute ready steps in parallel
		for _, step := range readySteps {
			wg.Add(1)
			go func(s types.WorkflowStep) {
				defer wg.Done()

				if e.verbose {
					fmt.Printf("\n⚡ Starting Step (async): %s\n", s.Name)
				}

				// Execute the step
				stepResult, err := e.executeStep(ctx, s, variables, results)
				if err != nil {
					errChan <- fmt.Errorf("step %s failed: %w", s.ID, err)
					return
				}

				// Store results
				e.mu.Lock()
				if s.Outputs != nil {
					for _, varName := range s.Outputs {
						results[varName] = stepResult.Outputs["result"]
						variables[varName] = stepResult.Outputs["result"]
					}
				}
				completed[s.ID] = true
				e.mu.Unlock()

				resultChan <- stepResult

				if e.verbose {
					fmt.Printf("   ✅ Completed (async): %s\n", s.Name)
				}
			}(step)
		}

		// Wait for this batch to complete
		wg.Wait()
	}

	close(errChan)
	close(resultChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			result.Status = types.ExecutionStatusFailed
			return result, err
		}
	}

	// Collect results
	for stepResult := range resultChan {
		result.StepResults = append(result.StepResults, stepResult)
	}

	// Copy final outputs
	for k, v := range results {
		result.Outputs[k] = v
	}

	result.Status = types.ExecutionStatusCompleted
	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	return result, nil
}

// executeStep executes a single workflow step
func (e *WorkflowExecutorV2) executeStep(ctx context.Context, step types.WorkflowStep, variables, results map[string]interface{}) (types.StepResult, error) {
	stepResult := types.StepResult{
		StepID:    step.ID,
		Status:    "running",
		StartTime: time.Now(),
		Outputs:   make(map[string]interface{}),
	}

	// Resolve variables in inputs
	e.mu.RLock()
	allVars := make(map[string]interface{})
	for k, v := range variables {
		allVars[k] = v
	}
	for k, v := range results {
		allVars[k] = v
	}
	e.mu.RUnlock()

	inputs := e.resolveVariables(step.Inputs, allVars)

	// Execute based on tool type
	var output interface{}
	var err error

	switch step.Tool {
	case "fetch":
		output, err = e.executeFetch(ctx, inputs)
	case "filesystem":
		output, err = e.executeFilesystem(ctx, inputs)
	case "memory":
		output, err = e.executeMemory(ctx, inputs)
	case "time":
		output, err = e.executeTime(ctx, inputs)
	case "sequential-thinking":
		output, err = e.executeSequentialThinking(ctx, inputs, allVars)
	default:
		err = fmt.Errorf("unknown tool: %s", step.Tool)
	}

	if err != nil {
		stepResult.Status = "failed"
		return stepResult, err
	}

	stepResult.Outputs["result"] = output
	stepResult.Status = "completed"
	now := time.Now()
	stepResult.EndTime = &now
	stepResult.Duration = now.Sub(stepResult.StartTime)

	return stepResult, nil
}

// buildDependencyGraph analyzes steps to find dependencies
func (e *WorkflowExecutorV2) buildDependencyGraph(workflow *types.WorkflowSpec) map[string][]string {
	deps := make(map[string][]string)

	// Build output to step mapping
	outputToStep := make(map[string]string)
	for _, step := range workflow.Steps {
		if step.Outputs != nil {
			for _, varName := range step.Outputs {
				outputToStep[varName] = step.ID
			}
		}
	}

	// Find dependencies based on variable usage
	for _, step := range workflow.Steps {
		deps[step.ID] = []string{}

		// Check if step uses variables from other steps
		inputStr := fmt.Sprintf("%v", step.Inputs)
		for varName, producerStep := range outputToStep {
			if strings.Contains(inputStr, "{{"+varName+"}}") ||
				strings.Contains(inputStr, "{{$"+varName+"}}") {
				deps[step.ID] = append(deps[step.ID], producerStep)
			}
		}

		// Also check explicit DependsOn field if it exists
		if len(step.DependsOn) > 0 {
			deps[step.ID] = append(deps[step.ID], step.DependsOn...)
		}
	}

	return deps
}

// findReadySteps finds steps that can be executed now
func (e *WorkflowExecutorV2) findReadySteps(steps []types.WorkflowStep, deps map[string][]string, completed map[string]bool) []types.WorkflowStep {
	var ready []types.WorkflowStep

	for _, step := range steps {
		if completed[step.ID] {
			continue
		}

		// Check if all dependencies are satisfied
		canExecute := true
		for _, dep := range deps[step.ID] {
			if !completed[dep] {
				canExecute = false
				break
			}
		}

		if canExecute {
			ready = append(ready, step)
		}
	}

	return ready
}

// executeFetch handles HTTP fetch operations
func (e *WorkflowExecutorV2) executeFetch(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	url, ok := inputs["url"].(string)
	if !ok {
		return nil, fmt.Errorf("fetch requires 'url' input")
	}

	method := "GET"
	if m, ok := inputs["method"].(string); ok {
		method = m
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	if headers, ok := inputs["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if vStr, ok := v.(string); ok {
				req.Header.Set(k, vStr)
			}
		}
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "RAGO-Workflow/1.0")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		return jsonData, nil
	}

	// Return as string if not JSON
	return string(body), nil
}

// Other execute methods remain similar but with proper async handling
func (e *WorkflowExecutorV2) executeFilesystem(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action, ok := inputs["action"].(string)
	if !ok {
		// Infer action
		if _, hasPath := inputs["path"]; hasPath {
			if _, hasData := inputs["data"]; hasData {
				action = "write"
			} else if _, hasContent := inputs["content"]; hasContent {
				action = "write"
			} else {
				action = "read"
			}
		} else {
			return nil, fmt.Errorf("filesystem requires 'action' input")
		}
	}

	switch action {
	case "read":
		path, ok := inputs["path"].(string)
		if !ok {
			return nil, fmt.Errorf("read requires 'path' input")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return string(data), nil

	case "write":
		path, ok := inputs["path"].(string)
		if !ok {
			return nil, fmt.Errorf("write requires 'path' input")
		}
		content := ""
		if c, ok := inputs["content"].(string); ok {
			content = c
		} else if c, ok := inputs["data"].(string); ok {
			content = c
		}
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return nil, err
		}
		return path, nil

	case "append":
		path, ok := inputs["path"].(string)
		if !ok {
			return nil, fmt.Errorf("append requires 'path' input")
		}
		content := ""
		if c, ok := inputs["content"].(string); ok {
			content = c
		}
		existing, _ := os.ReadFile(path)
		newContent := string(existing) + content
		err := os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return nil, err
		}
		return path, nil

	default:
		return nil, fmt.Errorf("unknown filesystem action: %s", action)
	}
}

func (e *WorkflowExecutorV2) executeMemory(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action, ok := inputs["action"].(string)
	if !ok {
		if key, ok := inputs["key"].(string); ok {
			if value, ok := inputs["value"]; ok {
				e.mu.Lock()
				e.memory[key] = value
				e.mu.Unlock()
				return value, nil
			}
		}
		return nil, fmt.Errorf("memory requires 'action' input")
	}

	key, _ := inputs["key"].(string)

	switch action {
	case "store":
		value := inputs["value"]
		e.mu.Lock()
		e.memory[key] = value
		e.mu.Unlock()
		return value, nil

	case "retrieve":
		e.mu.RLock()
		value, exists := e.memory[key]
		e.mu.RUnlock()
		if !exists {
			return nil, nil
		}
		return value, nil

	case "delete":
		e.mu.Lock()
		delete(e.memory, key)
		e.mu.Unlock()
		return "deleted", nil

	default:
		return nil, fmt.Errorf("unknown memory action: %s", action)
	}
}

func (e *WorkflowExecutorV2) executeTime(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action := "now"
	if a, ok := inputs["action"].(string); ok {
		action = a
	}

	switch action {
	case "now":
		format := "2006-01-02 15:04:05"
		if f, ok := inputs["format"].(string); ok {
			format = f
		}
		return time.Now().Format(format), nil

	default:
		return nil, fmt.Errorf("unknown time action: %s", action)
	}
}

func (e *WorkflowExecutorV2) executeSequentialThinking(ctx context.Context, inputs map[string]interface{}, variables map[string]interface{}) (interface{}, error) {
	// Build prompt
	prompt := ""
	if p, ok := inputs["prompt"].(string); ok {
		prompt = p
	} else if p, ok := inputs["task"].(string); ok {
		prompt = p
	}

	// Add context data
	if context, ok := inputs["context"].(string); ok {
		prompt = fmt.Sprintf("%s\n\nContext: %s", prompt, context)
	}
	if data, ok := inputs["data"].(string); ok {
		prompt = fmt.Sprintf("%s\n\nData: %s", prompt, data)
	}

	// Resolve variables in prompt
	prompt = e.resolveString(prompt, variables)

	if e.verbose {
		fmt.Printf("   🧠 Calling LLM with prompt: %s...\n", prompt[:minInt(50, len(prompt))])
	}

	// Call LLM
	opts := &core.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	response, err := e.llmProvider.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return response, nil
}

// resolveVariables replaces variable references in inputs
func (e *WorkflowExecutorV2) resolveVariables(inputs map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	resolved := make(map[string]interface{})
	for k, v := range inputs {
		if str, ok := v.(string); ok {
			resolved[k] = e.resolveString(str, variables)
		} else {
			resolved[k] = v
		}
	}
	return resolved
}

// resolveString replaces variable references in a string
func (e *WorkflowExecutorV2) resolveString(str string, variables map[string]interface{}) string {
	result := str
	for key, value := range variables {
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case map[string]interface{}:
			if jsonBytes, err := json.Marshal(v); err == nil {
				valueStr = string(jsonBytes)
			} else {
				valueStr = fmt.Sprintf("%v", v)
			}
		default:
			valueStr = fmt.Sprintf("%v", value)
		}

		patterns := []string{
			fmt.Sprintf("{{%s}}", key),
			fmt.Sprintf("{{outputs.%s}}", key),
			fmt.Sprintf("{{$%s}}", key),
			fmt.Sprintf("{{$outputs.%s}}", key),
		}

		for _, pattern := range patterns {
			result = strings.ReplaceAll(result, pattern, valueStr)
		}
	}
	return result
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
