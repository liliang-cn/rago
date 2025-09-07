package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// AgentGenerator generates agents and workflows using LLM with structured output
type AgentGenerator struct {
	llm     core.Generator
	verbose bool
}

// NewAgentGenerator creates a new agent generator
func NewAgentGenerator(llm core.Generator) *AgentGenerator {
	return &AgentGenerator{
		llm: llm,
	}
}

// SetVerbose enables verbose logging
func (g *AgentGenerator) SetVerbose(verbose bool) {
	g.verbose = verbose
}

// GenerateWorkflow generates a workflow from a natural language request
func (g *AgentGenerator) GenerateWorkflow(ctx context.Context, request string) (*types.WorkflowSpec, error) {
	prompt := g.buildWorkflowPrompt(request)

	opts := &core.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	// Define the expected schema
	var workflowSchema types.WorkflowSpec

	// Use GenerateStructured for type-safe JSON generation
	result, err := g.llm.GenerateStructured(ctx, prompt, &workflowSchema, opts)
	if err != nil {
		if g.verbose {
			fmt.Printf("⚠️  Structured generation failed: %v\n", err)
		}
		// Fallback to unstructured generation
		return g.generateWorkflowUnstructured(ctx, request, opts)
	}

	if g.verbose {
		fmt.Printf("✅ Structured generation successful (valid: %v)\n", result.Valid)
	}

	// Extract the workflow
	workflow, ok := result.Data.(*types.WorkflowSpec)
	if !ok {
		// Try unmarshaling from raw JSON
		var parsedWorkflow types.WorkflowSpec
		if err := json.Unmarshal([]byte(result.Raw), &parsedWorkflow); err != nil {
			return nil, fmt.Errorf("failed to extract workflow: %w", err)
		}
		workflow = &parsedWorkflow
	}

	// Validate and enhance workflow
	if err := g.validateWorkflow(workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// GenerateAgent generates a complete agent definition from a description
func (g *AgentGenerator) GenerateAgent(ctx context.Context, description string, agentType types.AgentType) (*types.Agent, error) {
	prompt := g.buildAgentPrompt(description, agentType)

	opts := &core.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	// Define schema for agent generation
	type AgentSchema struct {
		Name                    string             `json:"name"`
		Description             string             `json:"description"`
		MaxConcurrentExecutions int                `json:"max_concurrent_executions"`
		DefaultTimeoutMinutes   int                `json:"default_timeout_minutes"`
		EnableMetrics           bool               `json:"enable_metrics"`
		AutonomyLevel           string             `json:"autonomy_level"`
		Workflow                types.WorkflowSpec `json:"workflow"`
	}

	var schema AgentSchema

	result, err := g.llm.GenerateStructured(ctx, prompt, &schema, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent: %w", err)
	}

	// Extract generated agent data
	agentData, ok := result.Data.(*AgentSchema)
	if !ok {
		var parsed AgentSchema
		if err := json.Unmarshal([]byte(result.Raw), &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse agent data: %w", err)
		}
		agentData = &parsed
	}

	// Map autonomy level string to enum
	autonomyLevel := types.AutonomyManual
	switch strings.ToLower(agentData.AutonomyLevel) {
	case "scheduled":
		autonomyLevel = types.AutonomyScheduled
	case "reactive":
		autonomyLevel = types.AutonomyReactive
	case "proactive":
		autonomyLevel = types.AutonomyProactive
	case "adaptive":
		autonomyLevel = types.AutonomyAdaptive
	}

	// Create agent from generated data
	agent := &types.Agent{
		Name:        agentData.Name,
		Description: agentData.Description,
		Type:        agentType,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: agentData.MaxConcurrentExecutions,
			DefaultTimeout:          time.Duration(agentData.DefaultTimeoutMinutes) * time.Minute,
			EnableMetrics:           agentData.EnableMetrics,
			AutonomyLevel:           autonomyLevel,
		},
		Workflow:  agentData.Workflow,
		Status:    types.AgentStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return agent, nil
}

// GenerateToolCall generates parameters for a tool call based on context
func (g *AgentGenerator) GenerateToolCall(ctx context.Context, tool string, context string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Generate parameters for the MCP tool '%s' based on this context:
%s

Return only the JSON parameters object that matches the tool's expected schema.`, tool, context)

	opts := &core.GenerationOptions{
		Temperature: 0.5, // Lower temperature for more consistent parameter generation
		MaxTokens:   1000,
	}

	// Generic map for tool parameters
	var params map[string]interface{}

	result, err := g.llm.GenerateStructured(ctx, prompt, &params, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tool parameters: %w", err)
	}

	// Extract parameters
	if paramsData, ok := result.Data.(*map[string]interface{}); ok {
		return *paramsData, nil
	}

	// Fallback to parsing raw JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.Raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse tool parameters: %w", err)
	}

	return parsed, nil
}

// buildWorkflowPrompt creates the prompt for workflow generation
func (g *AgentGenerator) buildWorkflowPrompt(request string) string {
	return fmt.Sprintf(`You are a workflow generator for RAGO. Generate valid workflow JSON based on user requests.

Available MCP tools:
- filesystem: File operations (read, write, list, execute, move, copy, delete, mkdir)
- fetch: HTTP/HTTPS requests for APIs and websites  
- memory: Temporary storage (store, retrieve, delete, append)
- time: Date/time operations (now, format, parse)
- sequential-thinking: LLM analysis and reasoning

Data flow between steps:
- Use "outputs" to store step results in variables
- Use {{variableName}} in "inputs" to reference variables
- For sequential-thinking, pass data via "data": "{{variableName}}"

User request: "%s"

Generate a workflow that accomplishes this request. Include all necessary steps.`, request)
}

// buildAgentPrompt creates the prompt for agent generation
func (g *AgentGenerator) buildAgentPrompt(description string, agentType types.AgentType) string {
	typeDescription := ""
	switch agentType {
	case types.AgentTypeResearch:
		typeDescription = "Research agents focus on gathering, analyzing, and synthesizing information."
	case types.AgentTypeWorkflow:
		typeDescription = "Workflow agents automate multi-step processes and coordinate tasks."
	case types.AgentTypeMonitoring:
		typeDescription = "Monitoring agents track systems, detect issues, and send alerts."
	default:
		typeDescription = "General purpose agents perform various tasks."
	}

	return fmt.Sprintf(`Create a %s agent based on this description:
%s

%s

Generate a complete agent definition including:
- name: A concise, descriptive name
- description: Clear explanation of what the agent does
- max_concurrent_executions: Max parallel runs (usually 1-5)
- default_timeout_minutes: Timeout in minutes (usually 5-30)
- enable_metrics: Whether to track metrics (true/false)
- autonomy_level: One of "manual", "scheduled", "reactive", "proactive", or "adaptive"
- workflow: Complete workflow with steps using MCP tools

Available MCP tools: filesystem, fetch, memory, time, sequential-thinking

Return as JSON matching the expected schema.`, agentType, description, typeDescription)
}

// validateWorkflow validates and enhances a generated workflow
func (g *AgentGenerator) validateWorkflow(workflow *types.WorkflowSpec) error {
	if workflow == nil {
		return fmt.Errorf("workflow is nil")
	}

	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow has no steps")
	}

	// Initialize variables if needed
	if workflow.Variables == nil {
		workflow.Variables = make(map[string]interface{})
	}

	// Validate each step
	for i, step := range workflow.Steps {
		if step.ID == "" {
			step.ID = fmt.Sprintf("step%d", i+1)
		}
		if step.Name == "" {
			step.Name = fmt.Sprintf("Step %d", i+1)
		}
		if step.Type == "" {
			step.Type = "tool"
		}
	}

	return nil
}

// generateWorkflowUnstructured fallback for when structured generation fails
func (g *AgentGenerator) generateWorkflowUnstructured(ctx context.Context, request string, opts *core.GenerationOptions) (*types.WorkflowSpec, error) {
	prompt := fmt.Sprintf(`Generate ONLY a valid JSON workflow for this request: %s

Return ONLY the JSON structure with steps array, no explanation.`, request)

	response, err := g.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	// Extract JSON from response
	jsonStr := extractJSON(response.Content)

	var workflow types.WorkflowSpec
	if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	if err := g.validateWorkflow(&workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// extractJSON extracts JSON content from a text response
func extractJSON(text string) string {
	// Try to find JSON between code blocks
	if start := strings.Index(text, "```json"); start != -1 {
		start += 7
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Try to find JSON starting with { or [
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		// Find the matching closing bracket
		depth := 0
		inString := false
		escape := false

		for i, ch := range text {
			if escape {
				escape = false
				continue
			}

			if ch == '\\' {
				escape = true
				continue
			}

			if ch == '"' && !escape {
				inString = !inString
				continue
			}

			if !inString {
				switch ch {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
					if depth == 0 {
						return text[:i+1]
					}
				}
			}
		}
	}

	return text
}
