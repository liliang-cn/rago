package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// Planner creates execution plans from natural language requests
type Planner struct {
	config     *config.Config
	llm        domain.Generator
	mcpManager *mcp.Manager
	verbose    bool
	storage    *PlanStorage
}

// NewPlanner creates a new planner
func NewPlanner(cfg *config.Config, llm domain.Generator, mcpManager *mcp.Manager) *Planner {
	// Get configured data path with default
	dataPath := ".rago/data"
	if cfg.Agents != nil && cfg.Agents.DataPath != "" {
		dataPath = cfg.Agents.DataPath
	}
	
	// Ensure data directory exists
	os.MkdirAll(dataPath, 0755)
	
	// Initialize database storage in configured data path
	dbPath := filepath.Join(dataPath, "plans.db")
	storage, err := NewPlanStorage(dbPath)
	if err != nil {
		// Database is required now
		panic(fmt.Sprintf("Failed to initialize plan database: %v", err))
	}
	
	return &Planner{
		config:     cfg,
		llm:        llm,
		mcpManager: mcpManager,
		verbose:    false,
		storage:    storage,
	}
}

// SetVerbose enables verbose output
func (p *Planner) SetVerbose(v bool) {
	p.verbose = v
}

// Plan creates an execution plan and saves it to database
func (p *Planner) Plan(ctx context.Context, request string) (string, error) {
	if p.verbose {
		fmt.Printf("📝 Planning request: %s\n", request)
	}

	// Get available MCP tools
	availableTools := p.getAvailableTools(ctx)
	
	// Ask LLM to create execution plan
	plan, err := p.createPlan(ctx, request, availableTools)
	if err != nil {
		return "", fmt.Errorf("failed to create plan: %w", err)
	}

	// Generate unique task ID
	taskID := uuid.New().String()
	
	// Save to database
	if err := p.storage.SavePlan(taskID, plan, ""); err != nil {
		return "", fmt.Errorf("failed to save plan to database: %w", err)
	}
	
	if p.verbose {
		fmt.Printf("💾 Plan saved to database with ID: %s\n", taskID)
		fmt.Printf("📋 Generated %d steps\n", len(plan.Steps))
	}
	
	return taskID, nil
}

// createPlan asks LLM to create an execution plan
func (p *Planner) createPlan(ctx context.Context, request string, availableTools string) (*Plan, error) {
	prompt := fmt.Sprintf(`You are an AI assistant that plans tool execution.

USER REQUEST: %s

AVAILABLE MCP TOOLS:
%s

Create an execution plan using the available MCP tools. Return ONLY valid JSON in this exact format:
{
  "request": "original user request",
  "goal": "what we're trying to accomplish",
  "steps": [
    {
      "step_number": 1,
      "tool": "exact_tool_name_from_list", 
      "arguments": {"param_name": "value"},
      "description": "what this step does",
      "expected_output": "what we expect to get from this step",
      "depends_on": []
    }
  ],
  "output_format": "expected format of final output"
}

CRITICAL RULES:
- Use EXACT tool names from the list above (e.g., "mcp_filesystem_write_file" NOT "write_file")
- Use EXACT parameter names shown after "Parameters:" for each tool
- Common parameter mappings:
  * For filesystem tools: use "path" NOT "file_path" or "directory_path"
  * For write_file: use {"path": "...", "content": "..."}
  * For create_directory: use {"path": "..."}
  * For list_directory: use {"path": "..."}
- Return ONLY valid JSON (no explanations, no thinking, no markdown)
- All strings must be quoted with double quotes
- No trailing commas
- Include step numbers starting from 1
- Use depends_on array to specify which previous steps this step depends on (e.g., [1, 2])`, request, availableTools)

	opts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	response, err := p.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	if p.verbose {
		fmt.Printf("🔍 LLM Response: %s\n", response)
	}

	// Extract and parse JSON
	jsonStr := p.extractJSON(response)
	
	if p.verbose {
		fmt.Printf("🔍 Extracted JSON: %s\n", jsonStr)
	}
	
	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan: %w", err)
	}

	return &plan, nil
}

// getAvailableTools returns a description of all available MCP tools with parameter details
func (p *Planner) getAvailableTools(ctx context.Context) string {
	if p.mcpManager == nil {
		return "No MCP tools available"
	}
	
	// Use the new MCP package method
	return p.mcpManager.GetToolsDescription(ctx)
}

// extractJSON extracts JSON from LLM response
func (p *Planner) extractJSON(content string) string {
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