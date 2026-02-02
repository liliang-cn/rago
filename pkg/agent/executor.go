package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// ToolExecutor executes tool calls
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error)
}

// Executor executes agent plans step by step
type Executor struct {
	llmService    domain.Generator
	toolExecutor  ToolExecutor
	mcpService    MCPToolExecutor
	ragProcessor  domain.Processor
}

// MCPToolExecutor defines the interface for MCP tool execution
type MCPToolExecutor interface {
	CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error)
	ListTools() []domain.ToolDefinition
}

// NewExecutor creates a new executor
func NewExecutor(
	llmService domain.Generator,
	toolExecutor ToolExecutor,
	mcpService MCPToolExecutor,
	ragProcessor domain.Processor,
) *Executor {
	return &Executor{
		llmService:   llmService,
		toolExecutor: toolExecutor,
		mcpService:   mcpService,
		ragProcessor: ragProcessor,
	}
}

// ExecutePlan executes a plan step by step
func (e *Executor) ExecutePlan(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	startTime := time.Now()
	plan.Status = PlanStatusRunning

	stepsTotal := len(plan.Steps)
	stepsDone := 0
	stepsFailed := 0

	var finalResult interface{}
	var firstError string

	// Execute steps in order, respecting dependencies
	for i := range plan.Steps {
		step := &plan.Steps[i]

		// Check if dependencies are satisfied
		if !e.dependenciesSatisfied(step, plan.Steps) {
			step.Status = StepStatusFailed
			step.Error = "Dependencies not satisfied"
			stepsFailed++
			continue
		}

		// Execute the step
		result, err := e.ExecuteStep(ctx, step, plan)
		if err != nil {
			step.Status = StepStatusFailed
			step.Error = err.Error()
			stepsFailed++
			if firstError == "" {
				firstError = err.Error()
			}
		} else {
			step.Status = StepStatusCompleted
			step.Result = result
			stepsDone++
			// Last step's result becomes the final result
			if i == len(plan.Steps)-1 {
				finalResult = result
			}
		}

		step.CompletedAt = &[]time.Time{time.Now()}[0]
		plan.UpdatedAt = time.Now()
	}

	// Update plan status
	if stepsFailed == 0 {
		plan.Status = PlanStatusCompleted
	} else if stepsDone > 0 {
		plan.Status = PlanStatusCompleted // Partial success
	} else {
		plan.Status = PlanStatusFailed
	}

	duration := time.Since(startTime)

	return &ExecutionResult{
		PlanID:      plan.ID,
		SessionID:   plan.SessionID,
		Success:     stepsFailed == 0,
		StepsTotal:  stepsTotal,
		StepsDone:   stepsDone,
		StepsFailed: stepsFailed,
		FinalResult: finalResult,
		Error:       firstError,
		Duration:    duration.String(),
	}, nil
}

// ExecuteStep executes a single step
func (e *Executor) ExecuteStep(ctx context.Context, step *Step, plan *Plan) (interface{}, error) {
	step.Status = StepStatusRunning
	startTime := time.Now()
	step.StartedAt = &startTime

	// Preprocess arguments - handle placeholders like {{PREVIOUS_OUTPUT}}
	args := e.preprocessArguments(step, plan)

	// Route to appropriate executor based on tool name
	result, err := e.executeTool(ctx, step.Tool, args, plan)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// preprocessArguments handles special placeholders in arguments
func (e *Executor) preprocessArguments(step *Step, plan *Plan) map[string]interface{} {
	args := step.Arguments
	if args == nil {
		args = make(map[string]interface{})
	}

	// Find the current step index
	var stepIndex int = -1
	for i, s := range plan.Steps {
		if s.ID == step.ID {
			stepIndex = i
			break
		}
	}

	// Handle {{PREVIOUS_OUTPUT}} or similar placeholders
	for key, value := range args {
		if strValue, ok := value.(string); ok {
			if strings.Contains(strValue, "{{PREVIOUS_OUTPUT}}") || strings.Contains(strValue, "{{previous_output}}") {
				// Find the previous completed step
				if stepIndex > 0 {
					prevStep := &plan.Steps[stepIndex-1]
					if prevStep.Status == StepStatusCompleted && prevStep.Result != nil {
						// Convert previous result to string
						prevOutput := formatResultForContent(prevStep.Result)
						args[key] = strings.ReplaceAll(strValue, "{{PREVIOUS_OUTPUT}}", prevOutput)
						args[key] = strings.ReplaceAll(args[key].(string), "{{previous_output}}", prevOutput)
					}
				}
			}
		}
	}

	// Special handling for filesystem write_file: if content is missing or placeholder, use previous output
	if isFileWriteTool(step.Tool) {
		if _, hasContent := args["content"]; !hasContent {
			// Try to get content from previous step
			if stepIndex > 0 {
				prevStep := &plan.Steps[stepIndex-1]
				if prevStep.Status == StepStatusCompleted && prevStep.Result != nil {
					args["content"] = formatResultForContent(prevStep.Result)
				}
			}
		}
	}

	return args
}

// formatResultForContent formats a step result for use as file content
func formatResultForContent(result interface{}) string {
	if result == nil {
		return ""
	}

	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// For complex types, try to format as JSON
		jsonBytes, err := json.Marshal(result)
		if err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", result)
	}
}

// isFileWriteTool checks if the tool name indicates a file write operation
func isFileWriteTool(toolName string) bool {
	return strings.Contains(strings.ToLower(toolName), "write_file") ||
		strings.Contains(strings.ToLower(toolName), "create_file") ||
		strings.Contains(strings.ToLower(toolName), "filesystem") && strings.Contains(strings.ToLower(toolName), "write")
}

// executeTool routes the tool call to the appropriate executor
func (e *Executor) executeTool(ctx context.Context, toolName string, args map[string]interface{}, plan *Plan) (interface{}, error) {
	// Normalize args to ensure we have a valid map
	if args == nil {
		args = make(map[string]interface{})
	}

	// Try RAG processor first (for rag_query, rag_ingest, etc.)
	if e.ragProcessor != nil {
		if result, err := e.tryRAGTool(ctx, toolName, args); err == nil {
			return result, nil
		}
	}

	// Try MCP tools
	if e.mcpService != nil {
		if result, err := e.tryMCPTool(ctx, toolName, args); err == nil {
			return result, nil
		}
	}

	// Try custom tool executor
	if e.toolExecutor != nil {
		if result, err := e.toolExecutor.ExecuteTool(ctx, toolName, args); err == nil {
			return result, nil
		}
	}

	// Fall back to LLM for general reasoning
	if toolName == "llm" || toolName == "generate" {
		return e.executeLLM(ctx, args, plan)
	}

	return nil, fmt.Errorf("unknown tool: %s", toolName)
}

// tryRAGTool attempts to execute a RAG-related tool
func (e *Executor) tryRAGTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "rag_query", "query":
		query, ok := args["query"].(string)
		if !ok {
			return nil, fmt.Errorf("missing 'query' argument")
		}
		req := domain.QueryRequest{
			Query:        query,
			TopK:         5,
			Temperature:  0.7,
			MaxTokens:    1000,
			Stream:       false,
			ShowThinking: false,
			ShowSources:  true,
		}
		if topK, ok := args["top_k"].(float64); ok {
			req.TopK = int(topK)
		}
		return e.ragProcessor.Query(ctx, req)

	case "rag_ingest", "ingest":
		content, _ := args["content"].(string)
		filePath, _ := args["file_path"].(string)
		if content == "" && filePath == "" {
			return nil, fmt.Errorf("missing 'content' or 'file_path' argument")
		}
		req := domain.IngestRequest{
			Content:   content,
			FilePath:  filePath,
			ChunkSize: 1000,
			Overlap:   200,
		}
		return e.ragProcessor.Ingest(ctx, req)

	default:
		return nil, fmt.Errorf("not a RAG tool: %s", toolName)
	}
}

// tryMCPTool attempts to execute an MCP tool

// executeLLM executes a general LLM call
func (e *Executor) executeLLM(ctx context.Context, args map[string]interface{}, plan *Plan) (interface{}, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		// Fall back to using the goal or step description
		if plan != nil && plan.Goal != "" {
			prompt = plan.Goal
		}
		if prompt == "" {
			return nil, fmt.Errorf("missing 'prompt' argument")
		}
	}

	opts := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	if temp, ok := args["temperature"].(float64); ok {
		opts.Temperature = temp
	}
	if maxTokens, ok := args["max_tokens"].(float64); ok {
		opts.MaxTokens = int(maxTokens)
	}

	result, err := e.llmService.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return result, nil
}

// dependenciesSatisfied checks if all dependencies for a step are satisfied
func (e *Executor) dependenciesSatisfied(step *Step, steps []Step) bool {
	for _, depID := range step.DependsOn {
		found := false
		for _, s := range steps {
			if s.ID == depID && s.Status == StepStatusCompleted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// FormatResult formats a step result for display
func FormatResult(result interface{}) string {
	if result == nil {
		return "(empty)"
	}

	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	default:
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", result)
		}
		return string(jsonBytes)
	}
}

// tryMCPTool attempts to execute an MCP tool
func (e *Executor) tryMCPTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Call MCP service
	result, err := e.mcpService.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}

	// The result is *mcp.ToolResult with fields: Success (bool), Data (interface{}), Error (string)
	// Handle via type assertion since we can't import mcp package directly
	if result == nil {
		return nil, fmt.Errorf("MCP tool returned nil result")
	}

	// Try to convert to map for generic access
	if resultMap, ok := result.(map[string]interface{}); ok {
		if success, ok := resultMap["success"].(bool); ok && !success {
			errMsg, _ := resultMap["error"].(string)
			return nil, fmt.Errorf("MCP tool error: %s", errMsg)
		}
		data, ok := resultMap["data"]
		if ok {
			return data, nil
		}
		return result, nil
	}

	// If it's not a map, just return the result as-is
	return result, nil
}
