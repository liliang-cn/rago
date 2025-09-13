package agents

import "time"

// WorkflowSpec defines a simple workflow structure
type WorkflowSpec struct {
	Steps     []WorkflowStep         `json:"steps"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        StepType               `json:"type"`
	Tool        string                 `json:"tool,omitempty"`        // For legacy compatibility
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]string      `json:"outputs,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty"`
}

// StepType defines the type of workflow step
type StepType string

const (
	StepTypeTool      StepType = "tool"      // Execute an MCP tool
	StepTypeCondition StepType = "condition" // Conditional branching  
	StepTypeLoop      StepType = "loop"      // Loop execution
	StepTypeVariable  StepType = "variable"  // Variable assignment
)

// ExecutionResult represents the result of workflow execution
type ExecutionResult struct {
	ExecutionID  string                    `json:"execution_id"`
	AgentID      string                    `json:"agent_id,omitempty"`
	Status       ExecutionStatus           `json:"status"`
	StartTime    time.Time                 `json:"start_time"`
	EndTime      *time.Time                `json:"end_time,omitempty"`
	Duration     time.Duration             `json:"duration"`
	Results      map[string]interface{}    `json:"results,omitempty"`    // Deprecated
	Outputs      map[string]interface{}    `json:"outputs"`
	StepResults  []StepResult              `json:"step_results"`
	ErrorMessage string                    `json:"error_message,omitempty"`
}

// StepResult represents the result of a single step
type StepResult struct {
	StepID       string                 `json:"step_id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"` // Using string for simplicity
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// ExecutionStatus represents the status of execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// Plan represents a complete execution plan (used by both planner and executor)
type Plan struct {
	Request      string `json:"request"`
	Goal         string `json:"goal"`
	Steps        []Step `json:"steps"`
	OutputFormat string `json:"output_format"`
}

// Step represents a single step in the execution plan
type Step struct {
	StepNumber     int                    `json:"step_number"`
	Tool           string                 `json:"tool"`
	Arguments      map[string]interface{} `json:"arguments"`
	Description    string                 `json:"description"`
	ExpectedOutput string                 `json:"expected_output"`
	DependsOn      []int                  `json:"depends_on"`
}