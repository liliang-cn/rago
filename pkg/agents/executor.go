package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// Executor executes workflows using MCP tools and LLM planning
type Executor struct {
	config        *config.Config
	llm           domain.Generator
	mcpManager    *mcp.Manager
	verbose       bool
	storage       *PlanStorage
	workspacePath string
}

// NewExecutor creates a new workflow executor
func NewExecutor(cfg *config.Config, llm domain.Generator, mcpManager *mcp.Manager) *Executor {
	// Get configured data path with default
	dataPath := ".rago/data"
	if cfg.Agents != nil && cfg.Agents.DataPath != "" {
		dataPath = cfg.Agents.DataPath
	}
	
	// Get configured workspace path with default
	workspacePath := ".rago/workspace"
	if cfg.Agents != nil && cfg.Agents.WorkspacePath != "" {
		workspacePath = cfg.Agents.WorkspacePath
	}
	
	// Ensure directories exist
	os.MkdirAll(dataPath, 0755)
	os.MkdirAll(workspacePath, 0755)
	
	// Initialize database storage in configured data path
	dbPath := filepath.Join(dataPath, "plans.db")
	storage, err := NewPlanStorage(dbPath)
	if err != nil {
		// Database is required now
		panic(fmt.Sprintf("Failed to initialize plan database: %v", err))
	}
	
	return &Executor{
		config:        cfg,
		llm:           llm,
		mcpManager:    mcpManager,
		verbose:       false,
		storage:       storage,
		workspacePath: workspacePath,
	}
}

// SetVerbose enables verbose output
func (e *Executor) SetVerbose(v bool) {
	e.verbose = v
}

// Execute runs a workflow by asking LLM to plan and execute MCP tools
func (e *Executor) Execute(ctx context.Context, request string) (map[string]interface{}, error) {
	if e.verbose {
		fmt.Printf("🤖 Executing request: %s\n", request)
	}

	// Get available MCP tools
	availableTools := e.getAvailableTools(ctx)
	
	// Ask LLM to create execution plan
	plan, err := e.planExecution(ctx, request, availableTools)
	if err != nil {
		return nil, fmt.Errorf("failed to plan execution: %w", err)
	}

	if e.verbose {
		fmt.Printf("📋 Generated %d steps\n", len(plan.Steps))
	}

	// Execute the plan
	results := make(map[string]interface{})
	
	for i, step := range plan.Steps {
		if e.verbose {
			fmt.Printf("⚡ Step %d/%d: %s\n", i+1, len(plan.Steps), step.Description)
		}

		result, err := e.executeStep(ctx, step, results)
		if err != nil {
			return nil, fmt.Errorf("step %d failed: %w", i+1, err)
		}

		// Store result for next steps
		results[fmt.Sprintf("step_%d", i+1)] = result
		results["last_result"] = result
	}

	return results, nil
}

// ExecutePlan executes a saved plan by ID
func (e *Executor) ExecutePlan(ctx context.Context, planID string) (map[string]interface{}, error) {
	// Get plan from database
	plan, err := e.storage.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan from database: %w", err)
	}
	
	// Record execution start in database
	execID, err := e.storage.RecordExecution(planID)
	if err != nil && e.verbose {
		fmt.Printf("⚠️  Warning: Failed to record execution start: %v\n", err)
	}

	if e.verbose {
		fmt.Printf("📋 Executing plan: %s\n", plan.Goal)
		fmt.Printf("📝 Total steps: %d\n", len(plan.Steps))
	}

	// Execute each step
	results := make(map[string]interface{})
	
	for _, step := range plan.Steps {
		if e.verbose {
			fmt.Printf("⚡ Step %d/%d: %s\n", step.StepNumber, len(plan.Steps), step.Description)
		}

		// Convert Step to ExecutionStep for compatibility
		execStep := ExecutionStep{
			Tool:        step.Tool,
			Arguments:   step.Arguments,
			Description: step.Description,
		}

		result, err := e.executeStep(ctx, execStep, results)
		if err != nil {
			if execID > 0 {
				e.storage.CompleteExecution(execID, results, err)
			}
			return nil, fmt.Errorf("step %d failed: %w", step.StepNumber, err)
		}

		// Store result for next steps
		results[fmt.Sprintf("step_%d", step.StepNumber)] = result
		results["last_result"] = result
		
		// Also store by tool name for easier reference
		results[step.Tool] = result
	}

	// Record successful execution in database
	if execID > 0 {
		if err := e.storage.CompleteExecution(execID, results, nil); err != nil && e.verbose {
			fmt.Printf("⚠️  Warning: Failed to record execution completion: %v\n", err)
		}
	}

	return results, nil
}

// ExecutionPlan represents a plan for executing a request
type ExecutionPlan struct {
	Steps []ExecutionStep `json:"steps"`
	Goal  string          `json:"goal"`
}

// ExecutionStep represents a single step in execution
type ExecutionStep struct {
	Tool        string                 `json:"tool"`
	Arguments   map[string]interface{} `json:"arguments"`
	Description string                 `json:"description"`
}

// planExecution asks LLM to create an execution plan
func (e *Executor) planExecution(ctx context.Context, request string, availableTools string) (*ExecutionPlan, error) {
	prompt := fmt.Sprintf(`You are an AI assistant that plans tool execution.

USER REQUEST: %s

AVAILABLE MCP TOOLS:
%s

Create an execution plan using the available MCP tools. Return ONLY valid JSON in this exact format:
{
  "goal": "what we're trying to accomplish",
  "steps": [
    {
      "tool": "tool_name", 
      "arguments": {"arg1": "value1", "arg2": "value2"},
      "description": "what this step does"
    }
  ]
}

IMPORTANT:
- Use only the tools listed above
- Return ONLY valid JSON (no explanations, no thinking, no markdown)
- All strings must be quoted with double quotes
- No trailing commas
- When referencing data from previous steps, use template variables like "${step_1_result.length}" or "${search_result.count}"
- The system will automatically substitute these variables with actual data from previous step results
- Test your JSON before responding`, request, availableTools)

	opts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	response, err := e.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	if e.verbose {
		fmt.Printf("🔍 LLM Response: %s\n", response)
	}

	// Extract and parse JSON
	jsonStr := e.extractJSON(response)
	
	if e.verbose {
		fmt.Printf("🔍 Extracted JSON: %s\n", jsonStr)
	}
	
	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan: %w", err)
	}

	return &plan, nil
}

// executeStep executes a single step using MCP tools
func (e *Executor) executeStep(ctx context.Context, step ExecutionStep, previousResults map[string]interface{}) (interface{}, error) {
	if e.mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not available")
	}

	// Use the MCP manager to find and call the tool
	toolInfo, _, err := e.mcpManager.FindToolProvider(ctx, step.Tool)
	if err != nil {
		return nil, err
	}

	if e.verbose {
		fmt.Printf("   Using %s from server %s\n", toolInfo.ActualName, toolInfo.ServerName)
	}

	// Call the tool through the MCP manager
	result, err := e.mcpManager.CallTool(ctx, step.Tool, step.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool %s failed: %w", step.Tool, err)
	}

	return result, nil
}


// getAvailableTools returns a description of all available MCP tools
func (e *Executor) getAvailableTools(ctx context.Context) string {
	if e.mcpManager == nil {
		return "No MCP tools available"
	}
	
	// Use the new MCP package method
	return e.mcpManager.GetToolsDescription(ctx)
}

// extractJSON extracts JSON from LLM response
func (e *Executor) extractJSON(content string) string {
	// Remove thinking tags if present
	content = strings.ReplaceAll(content, "<think>", "")
	content = strings.ReplaceAll(content, "</think>", "")
	
	// Find JSON content between ```json and ``` 
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// Try to find JSON object directly - find first { and last }
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		jsonStr := content[start : end+1]
		// Validate it's actually JSON by trying to find complete object
		braceCount := 0
		for i, char := range jsonStr {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					return jsonStr[:i+1]
				}
			}
		}
		return jsonStr
	}

	return content
}