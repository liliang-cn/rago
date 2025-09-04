package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// WorkflowExecutor handles the execution of workflow steps
type WorkflowExecutor struct {
	config      *config.Config
	llmProvider domain.Generator
	mcpClients  map[string]*MCPClient
	memory      map[string]interface{}
	verbose     bool
}

// MCPClient represents a connection to an MCP server
type MCPClient struct {
	// Note: This type is currently unused but kept for future MCP integration
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(cfg *config.Config, llm domain.Generator) *WorkflowExecutor {
	return &WorkflowExecutor{
		config:      cfg,
		llmProvider: llm,
		mcpClients:  make(map[string]*MCPClient),
		memory:      make(map[string]interface{}),
		verbose:     false,
	}
}

// SetVerbose enables verbose output
func (e *WorkflowExecutor) SetVerbose(v bool) {
	e.verbose = v
}

// Execute runs a complete workflow
func (e *WorkflowExecutor) Execute(ctx context.Context, workflow *types.WorkflowSpec) (*types.ExecutionResult, error) {
	result := &types.ExecutionResult{
		ExecutionID: fmt.Sprintf("exec_%d", time.Now().Unix()),
		Status:      types.ExecutionStatusRunning,
		StartTime:   time.Now(),
		Outputs:     make(map[string]interface{}),
		StepResults: make([]types.StepResult, 0),
	}

	// Initialize variables from workflow
	variables := make(map[string]interface{})
	for k, v := range workflow.Variables {
		variables[k] = v
	}

	// Execute each step
	for _, step := range workflow.Steps {
		if e.verbose {
			fmt.Printf("\n⚙️  Executing Step: %s\n", step.Name)
		}

		stepResult := types.StepResult{
			StepID:    step.ID,
			Status:    "running",
			StartTime: time.Now(),
			Outputs:   make(map[string]interface{}),
		}

		// Replace variables in inputs
		inputs := e.resolveVariables(step.Inputs, variables)

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
			output, err = e.executeSequentialThinking(ctx, inputs, variables)
		default:
			err = fmt.Errorf("unknown tool: %s", step.Tool)
		}

		if err != nil {
			stepResult.Status = "failed"
			result.StepResults = append(result.StepResults, stepResult)
			result.Status = types.ExecutionStatusFailed
			return result, err
		}

		// Store outputs in variables
		if step.Outputs != nil {
			for outKey, varName := range step.Outputs {
				variables[varName] = output
				result.Outputs[varName] = output
				stepResult.Outputs[outKey] = output
				if e.verbose {
					// Show a preview of stored data
					preview := fmt.Sprintf("%v", output)
					if len(preview) > 100 {
						preview = preview[:100] + "..."
					}
					fmt.Printf("   💾 Stored %s = %s\n", varName, preview)
				}
			}
		}

		stepResult.Status = "completed"
		now := time.Now()
		stepResult.EndTime = &now
		stepResult.Duration = now.Sub(stepResult.StartTime)
		result.StepResults = append(result.StepResults, stepResult)

		if e.verbose {
			fmt.Printf("   ✅ Completed: %s\n", step.Name)
		}
	}

	result.Status = types.ExecutionStatusCompleted
	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	return result, nil
}

// executeFetch handles HTTP fetch operations
func (e *WorkflowExecutor) executeFetch(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
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

	// Add headers if provided
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

// executeFilesystem handles file operations
func (e *WorkflowExecutor) executeFilesystem(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action, ok := inputs["action"].(string)
	if !ok {
		// Try to infer action from other parameters
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
		// Read existing content
		existing, _ := os.ReadFile(path)
		// Append new content
		newContent := string(existing) + content
		err := os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return nil, err
		}
		return path, nil

	case "list":
		path := "./"
		if p, ok := inputs["path"].(string); ok {
			path = p
		}
		// Simple implementation - just return the path for now
		return fmt.Sprintf("Files in %s", path), nil

	default:
		return nil, fmt.Errorf("unknown filesystem action: %s", action)
	}
}

// executeMemory handles in-memory storage
func (e *WorkflowExecutor) executeMemory(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action, ok := inputs["action"].(string)
	if !ok {
		// If no action, try to infer it
		if key, ok := inputs["key"].(string); ok {
			if value, ok := inputs["value"]; ok {
				// Store value directly and return
				e.memory[key] = value
				return value, nil
			} else {
				// Default to retrieve action
				action = "retrieve"
			}
		} else {
			return nil, fmt.Errorf("memory requires 'action' or 'key' input")
		}
	}

	key, _ := inputs["key"].(string)

	switch action {
	case "store":
		value := inputs["value"]
		e.memory[key] = value
		return value, nil

	case "retrieve":
		value, exists := e.memory[key]
		if !exists {
			return nil, nil
		}
		return value, nil

	case "delete":
		delete(e.memory, key)
		return "deleted", nil

	case "append":
		existing, _ := e.memory[key].(string)
		newValue, _ := inputs["value"].(string)
		combined := existing + newValue
		e.memory[key] = combined
		return combined, nil

	default:
		return nil, fmt.Errorf("unknown memory action: %s", action)
	}
}

// executeTime handles time operations
func (e *WorkflowExecutor) executeTime(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	action := "now"
	if a, ok := inputs["action"].(string); ok {
		action = a
	}

	switch action {
	case "now":
		format := "2006-01-02 15:04:05"
		if f, ok := inputs["format"].(string); ok {
			// Convert common time formats to Go format
			format = convertTimeFormat(f)
		}
		return time.Now().Format(format), nil

	default:
		return nil, fmt.Errorf("unknown time action: %s", action)
	}
}

// executeSequentialThinking handles LLM calls
func (e *WorkflowExecutor) executeSequentialThinking(ctx context.Context, inputs map[string]interface{}, variables map[string]interface{}) (interface{}, error) {
	// Build the prompt
	prompt := ""
	if p, ok := inputs["prompt"].(string); ok {
		prompt = p
	} else if p, ok := inputs["task"].(string); ok {
		prompt = p
	}

	// Resolve variables in prompt first
	prompt = e.resolveString(prompt, variables)

	// Add context data (also resolve variables in these)
	if context, ok := inputs["context"].(string); ok {
		contextResolved := e.resolveString(context, variables)
		prompt = fmt.Sprintf("%s\n\nContext: %s", prompt, contextResolved)
	}
	if data, ok := inputs["data"].(string); ok {
		dataResolved := e.resolveString(data, variables)
		prompt = fmt.Sprintf("%s\n\nData: %s", prompt, dataResolved)
	}

	// Add all other inputs as context
	for k, v := range inputs {
		if k != "prompt" && k != "task" && k != "context" && k != "data" {
			if vStr, ok := v.(string); ok {
				v = e.resolveString(vStr, variables)
			}
			prompt = fmt.Sprintf("%s\n\n%s: %v", prompt, k, v)
		}
	}

	if e.verbose {
		fmt.Printf("   🧠 Calling LLM with prompt: %s...\n", prompt[:min(50, len(prompt))])
	}

	// Call the LLM
	opts := &domain.GenerationOptions{
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
func (e *WorkflowExecutor) resolveVariables(inputs map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
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
func (e *WorkflowExecutor) resolveString(str string, variables map[string]interface{}) string {
	result := str
	for key, value := range variables {
		// Convert value to string, handling different types
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case map[string]interface{}:
			// For complex objects, try to marshal as JSON
			if jsonBytes, err := json.Marshal(v); err == nil {
				valueStr = string(jsonBytes)
			} else {
				valueStr = fmt.Sprintf("%v", v)
			}
		default:
			valueStr = fmt.Sprintf("%v", value)
		}

		// Try multiple variable formats
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertTimeFormat converts common time format strings to Go's time format
func convertTimeFormat(format string) string {
	// Handle common format strings
	switch format {
	case "HH:mm:ss":
		return "15:04:05"
	case "HH:mm":
		return "15:04"
	case "YYYY-MM-DD":
		return "2006-01-02"
	case "DD/MM/YYYY":
		return "02/01/2006"
	case "MM/DD/YYYY":
		return "01/02/2006"
	case "YYYY-MM-DD HH:mm:ss":
		return "2006-01-02 15:04:05"
	default:
		// Return as-is if it's already in Go format or unknown
		return format
	}
}
