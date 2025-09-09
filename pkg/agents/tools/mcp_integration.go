package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// MCPToolExecutor integrates with MCP services to execute tools
type MCPToolExecutor struct {
	mcpClient MCPClientInterface
}

// MCPClientInterface defines the interface for MCP client interactions
type MCPClientInterface interface {
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error)
	GetServerStatus(ctx context.Context, serverName string) (MCPServerStatus, error)
}

// MCPTool represents an MCP tool
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Server      string                 `json:"server"`
}

// MCPServerStatus represents MCP server status
type MCPServerStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
}

// MCPExecutionResult represents the result of MCP tool execution
type MCPExecutionResult struct {
	Tool     string                 `json:"tool"`
	Success  bool                   `json:"success"`
	Result   interface{}            `json:"result"`
	Error    string                 `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMCPToolExecutor creates a new MCP tool executor
func NewMCPToolExecutor(mcpClient MCPClientInterface) *MCPToolExecutor {
	return &MCPToolExecutor{
		mcpClient: mcpClient,
	}
}

// ExecuteTool executes an MCP tool with the given inputs
func (m *MCPToolExecutor) ExecuteTool(ctx context.Context, toolName string, inputs map[string]interface{}) (*MCPExecutionResult, error) {
	startTime := time.Now()

	result := &MCPExecutionResult{
		Tool:     toolName,
		Success:  false,
		Metadata: make(map[string]interface{}),
	}

	// Execute the MCP tool
	toolResult, err := m.mcpClient.CallTool(ctx, toolName, inputs)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.Result = toolResult

	// Add execution metadata
	result.Metadata["execution_time"] = result.Duration.String()
	result.Metadata["inputs"] = inputs

	return result, nil
}

// ListAvailableTools returns all available MCP tools
func (m *MCPToolExecutor) ListAvailableTools(ctx context.Context) ([]MCPTool, error) {
	return m.mcpClient.ListTools(ctx)
}

// ValidateToolInputs validates inputs against the tool's schema
func (m *MCPToolExecutor) ValidateToolInputs(ctx context.Context, toolName string, inputs map[string]interface{}) error {
	tools, err := m.ListAvailableTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Find the tool
	var tool *MCPTool
	for _, t := range tools {
		if t.Name == toolName {
			tool = &t
			break
		}
	}

	if tool == nil {
		return fmt.Errorf("tool %s not found", toolName)
	}

	// Basic validation - in a real implementation, you'd validate against the JSON schema
	if tool.InputSchema != nil {
		// Simplified validation - check required fields exist
		if _, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
			if required, ok := tool.InputSchema["required"].([]interface{}); ok {
				for _, req := range required {
					if reqField, ok := req.(string); ok {
						if _, exists := inputs[reqField]; !exists {
							return fmt.Errorf("required field %s is missing", reqField)
						}
					}
				}
			}
		}
	}

	return nil
}

// MCPWorkflowStepExecutor executes workflow steps that use MCP tools
type MCPWorkflowStepExecutor struct {
	toolExecutor   *MCPToolExecutor
	templateEngine types.TemplateEngine
}

// NewMCPWorkflowStepExecutor creates a new MCP workflow step executor
func NewMCPWorkflowStepExecutor(mcpClient MCPClientInterface, templateEngine types.TemplateEngine) *MCPWorkflowStepExecutor {
	return &MCPWorkflowStepExecutor{
		toolExecutor:   NewMCPToolExecutor(mcpClient),
		templateEngine: templateEngine,
	}
}

// ExecuteStep executes a workflow step using MCP tools
func (m *MCPWorkflowStepExecutor) ExecuteStep(ctx context.Context, step types.WorkflowStep, variables map[string]interface{}) (*types.StepResult, error) {
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
		if m.templateEngine != nil {
			rendered, err := m.templateEngine.RenderObject(value, variables)
			if err != nil {
				return m.failStep(stepResult, fmt.Sprintf("failed to render input %s: %v", key, err))
			}
			renderedInputs[key] = rendered
		} else {
			renderedInputs[key] = value
		}
	}
	stepResult.Inputs = renderedInputs

	// Validate inputs
	if err := m.toolExecutor.ValidateToolInputs(ctx, step.Tool, renderedInputs); err != nil {
		return m.failStep(stepResult, fmt.Sprintf("input validation failed: %v", err))
	}

	// Execute MCP tool
	mcpResult, err := m.toolExecutor.ExecuteTool(ctx, step.Tool, renderedInputs)
	if err != nil {
		return m.failStep(stepResult, fmt.Sprintf("MCP tool execution failed: %v", err))
	}

	// Process results
	if !mcpResult.Success {
		return m.failStep(stepResult, fmt.Sprintf("MCP tool failed: %s", mcpResult.Error))
	}

	// Map outputs based on step configuration
	if mcpResult.Result != nil {
		m.mapStepOutputs(step, mcpResult.Result, stepResult)
	}

	// Mark step as completed
	stepResult.Status = types.ExecutionStatusCompleted
	endTime := time.Now()
	stepResult.EndTime = &endTime
	stepResult.Duration = endTime.Sub(stepResult.StartTime)

	return stepResult, nil
}

// mapStepOutputs maps MCP tool results to step outputs
func (m *MCPWorkflowStepExecutor) mapStepOutputs(step types.WorkflowStep, mcpResult interface{}, stepResult *types.StepResult) {
	// If MCP result is a map, map specific fields
	if resultMap, ok := mcpResult.(map[string]interface{}); ok {
		for outputKey, variableName := range step.Outputs {
			if value, exists := resultMap[outputKey]; exists {
				stepResult.Outputs[variableName] = value
			}
		}
	} else {
		// If result is not a map, use the entire result for the first output
		if len(step.Outputs) > 0 {
			for _, variableName := range step.Outputs {
				stepResult.Outputs[variableName] = mcpResult
				break // Only use the first output mapping
			}
		}
	}
}

// failStep marks a step as failed and returns the result
func (m *MCPWorkflowStepExecutor) failStep(stepResult *types.StepResult, errorMessage string) (*types.StepResult, error) {
	stepResult.Status = types.ExecutionStatusFailed
	stepResult.ErrorMessage = errorMessage
	endTime := time.Now()
	stepResult.EndTime = &endTime
	stepResult.Duration = endTime.Sub(stepResult.StartTime)
	return stepResult, fmt.Errorf("%s", errorMessage)
}

// MockMCPClient provides a mock implementation for testing
type MockMCPClient struct {
	tools   []MCPTool
	results map[string]interface{}
}

// NewMockMCPClient creates a new mock MCP client
func NewMockMCPClient() *MockMCPClient {
	return &MockMCPClient{
		tools: []MCPTool{
			{
				Name:        "sqlite_query",
				Description: "Execute SQLite queries",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "SQL query to execute",
						},
						"database": map[string]interface{}{
							"type":        "string",
							"description": "Database file path",
						},
					},
					"required": []interface{}{"query"},
				},
				Server: "sqlite-server",
			},
			{
				Name:        "file_read",
				Description: "Read file contents",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path to read",
						},
					},
					"required": []interface{}{"path"},
				},
				Server: "filesystem-server",
			},
			{
				Name:        "web_search",
				Description: "Search the web",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Number of results",
							"default":     10,
						},
					},
					"required": []interface{}{"query"},
				},
				Server: "web-search-server",
			},
		},
		results: make(map[string]interface{}),
	}
}

// ListTools returns available tools
func (m *MockMCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	return m.tools, nil
}

// CallTool executes a tool
func (m *MockMCPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	// Check if a custom result has been set for this tool
	if result, exists := m.results[name]; exists {
		return result, nil
	}

	// Simulate tool execution with mock results
	switch name {
	case "sqlite_query":
		return map[string]interface{}{
			"result": []map[string]interface{}{
				{"id": 1, "name": "Example Row 1"},
				{"id": 2, "name": "Example Row 2"},
			},
			"rowCount": 2,
		}, nil
	case "file_read":
		return map[string]interface{}{
			"content":  "Mock file content",
			"size":     17,
			"encoding": "utf-8",
		}, nil
	case "web_search":
		return map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":   "Example Search Result 1",
					"url":     "https://example.com/1",
					"snippet": "This is a mock search result",
				},
				{
					"title":   "Example Search Result 2",
					"url":     "https://example.com/2",
					"snippet": "Another mock search result",
				},
			},
			"totalResults": 2,
		}, nil
	default:
		return map[string]interface{}{
			"message":   fmt.Sprintf("Mock execution of %s", name),
			"arguments": arguments,
		}, nil
	}
}

// GetServerStatus returns server status
func (m *MockMCPClient) GetServerStatus(ctx context.Context, serverName string) (MCPServerStatus, error) {
	return MCPServerStatus{
		Name:      serverName,
		Status:    "running",
		Connected: true,
	}, nil
}

// SetMockResult sets a mock result for a specific tool
func (m *MockMCPClient) SetMockResult(toolName string, result interface{}) {
	m.results[toolName] = result
}

// ConvertToMCPResult converts agent execution results to a format compatible with MCP tools
func ConvertToMCPResult(result *types.ExecutionResult) map[string]interface{} {
	mcpResult := map[string]interface{}{
		"execution_id": result.ExecutionID,
		"agent_id":     result.AgentID,
		"status":       string(result.Status),
		"start_time":   result.StartTime,
		"duration":     result.Duration.String(),
		"results":      result.Results,
		"outputs":      result.Outputs,
	}

	if result.EndTime != nil {
		mcpResult["end_time"] = *result.EndTime
	}

	if result.ErrorMessage != "" {
		mcpResult["error"] = result.ErrorMessage
	}

	// Add step results
	stepResults := make([]map[string]interface{}, len(result.StepResults))
	for i, step := range result.StepResults {
		stepResults[i] = map[string]interface{}{
			"step_id":  step.StepID,
			"name":     step.Name,
			"status":   string(step.Status),
			"duration": step.Duration.String(),
			"inputs":   step.Inputs,
			"outputs":  step.Outputs,
		}

		if step.ErrorMessage != "" {
			stepResults[i]["error"] = step.ErrorMessage
		}
	}
	mcpResult["step_results"] = stepResults

	return mcpResult
}
