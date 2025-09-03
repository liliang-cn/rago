package types

import (
	"time"
)

// WorkflowSpec defines the structure of a workflow
type WorkflowSpec struct {
	Steps       []WorkflowStep        `json:"steps" yaml:"steps"`
	Triggers    []Trigger             `json:"triggers" yaml:"triggers"`
	Variables   map[string]interface{} `json:"variables" yaml:"variables"`
	ErrorPolicy ErrorPolicy           `json:"error_policy" yaml:"error_policy"`
	Timeout     time.Duration         `json:"timeout" yaml:"timeout"`
	Metadata    WorkflowMetadata      `json:"metadata" yaml:"metadata"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Type        StepType               `json:"type" yaml:"type"`
	Tool        string                 `json:"tool" yaml:"tool"`
	Inputs      map[string]interface{} `json:"inputs" yaml:"inputs"`
	Outputs     map[string]string      `json:"outputs" yaml:"outputs"` // Variable mappings
	Conditions  []Condition            `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Retry       RetryPolicy            `json:"retry" yaml:"retry"`
	Timeout     time.Duration          `json:"timeout" yaml:"timeout"`
	DependsOn   []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

// StepType defines the type of workflow step
type StepType string

const (
	StepTypeTool       StepType = "tool"       // Execute an MCP tool
	StepTypeCondition  StepType = "condition"  // Conditional branching
	StepTypeLoop       StepType = "loop"       // Loop execution
	StepTypeParallel   StepType = "parallel"   // Parallel execution
	StepTypeDelay      StepType = "delay"      // Time delay
	StepTypeVariable   StepType = "variable"   // Variable assignment
)

// Trigger defines when a workflow should be executed
type Trigger struct {
	ID         string                 `json:"id" yaml:"id"`
	Type       TriggerType            `json:"type" yaml:"type"`
	Name       string                 `json:"name" yaml:"name"`
	Conditions []Condition            `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Schedule   string                 `json:"schedule,omitempty" yaml:"schedule,omitempty"` // Cron expression
	Config     map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// TriggerType defines different trigger mechanisms
type TriggerType string

const (
	TriggerTypeManual    TriggerType = "manual"     // User-triggered
	TriggerTypeSchedule  TriggerType = "schedule"   // Time-based (cron)
	TriggerTypeEvent     TriggerType = "event"      // Event-driven
	TriggerTypeWebhook   TriggerType = "webhook"    // Webhook trigger
	TriggerTypeFileWatch TriggerType = "file_watch" // File system events
	TriggerTypeAPI       TriggerType = "api"        // API endpoint trigger
)

// Condition represents a conditional expression
type Condition struct {
	Field    string      `json:"field" yaml:"field"`
	Operator string      `json:"operator" yaml:"operator"` // eq, ne, gt, lt, contains, exists
	Value    interface{} `json:"value" yaml:"value"`
	LogicOp  string      `json:"logic_op,omitempty" yaml:"logic_op,omitempty"` // and, or
}

// ErrorPolicy defines how errors should be handled
type ErrorPolicy struct {
	Strategy    ErrorStrategy `json:"strategy" yaml:"strategy"`
	MaxRetries  int           `json:"max_retries" yaml:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay" yaml:"retry_delay"`
	ContinueOn  []string      `json:"continue_on,omitempty" yaml:"continue_on,omitempty"` // Error types to continue on
	NotifyOn    []string      `json:"notify_on,omitempty" yaml:"notify_on,omitempty"`     // Error types to notify on
}

// ErrorStrategy defines error handling strategies
type ErrorStrategy string

const (
	ErrorStrategyFail     ErrorStrategy = "fail"     // Stop execution on error
	ErrorStrategyContinue ErrorStrategy = "continue" // Continue with next step
	ErrorStrategyRetry    ErrorStrategy = "retry"    // Retry failed step
	ErrorStrategySkip     ErrorStrategy = "skip"     // Skip failed step
)

// RetryPolicy defines retry behavior for individual steps
type RetryPolicy struct {
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	MaxRetries  int           `json:"max_retries" yaml:"max_retries"`
	Delay       time.Duration `json:"delay" yaml:"delay"`
	BackoffType BackoffType   `json:"backoff_type" yaml:"backoff_type"`
	MaxDelay    time.Duration `json:"max_delay" yaml:"max_delay"`
}

// BackoffType defines retry backoff strategies
type BackoffType string

const (
	BackoffFixed       BackoffType = "fixed"       // Fixed delay
	BackoffExponential BackoffType = "exponential" // Exponential backoff
	BackoffLinear      BackoffType = "linear"      // Linear backoff
)

// WorkflowMetadata contains workflow metadata
type WorkflowMetadata struct {
	Author      string            `json:"author,omitempty" yaml:"author,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Category    string            `json:"category,omitempty" yaml:"category,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// ToolChain represents a sequence of tool executions
type ToolChain struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Steps     []ChainStep       `json:"steps" yaml:"steps"`
	Variables map[string]interface{} `json:"variables" yaml:"variables"`
	Parallel  bool              `json:"parallel" yaml:"parallel"`
}

// ChainStep represents a step in a tool chain
type ChainStep struct {
	ID         string                 `json:"id" yaml:"id"`
	Name       string                 `json:"name" yaml:"name"`
	ToolName   string                 `json:"tool_name" yaml:"tool_name"`
	Inputs     map[string]interface{} `json:"inputs" yaml:"inputs"`     // Template expressions
	Outputs    map[string]string      `json:"outputs" yaml:"outputs"`   // Variable mappings
	Conditions []Condition            `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Retry      RetryPolicy            `json:"retry" yaml:"retry"`
	Timeout    time.Duration          `json:"timeout" yaml:"timeout"`
}

// LoopDefinition defines loop behavior
type LoopDefinition struct {
	ID          string      `json:"id" yaml:"id"`
	Type        LoopType    `json:"type" yaml:"type"`
	Condition   Condition   `json:"condition,omitempty" yaml:"condition,omitempty"`
	Iterator    string      `json:"iterator,omitempty" yaml:"iterator,omitempty"`
	MaxIterations int       `json:"max_iterations" yaml:"max_iterations"`
	Steps       []string    `json:"steps" yaml:"steps"` // Step IDs to loop over
}

// LoopType defines different loop types
type LoopType string

const (
	LoopTypeWhile   LoopType = "while"   // While condition is true
	LoopTypeFor     LoopType = "for"     // For each item in iterator
	LoopTypeCount   LoopType = "count"   // Fixed number of iterations
)

// ParallelGroup defines parallel execution
type ParallelGroup struct {
	ID      string   `json:"id" yaml:"id"`
	Name    string   `json:"name" yaml:"name"`
	Steps   []string `json:"steps" yaml:"steps"` // Step IDs to execute in parallel
	WaitAll bool     `json:"wait_all" yaml:"wait_all"` // Wait for all or first completion
}

// TemplateEngine handles variable interpolation
type TemplateEngine interface {
	Render(template string, variables map[string]interface{}) (string, error)
	RenderObject(obj interface{}, variables map[string]interface{}) (interface{}, error)
}

// WorkflowValidator validates workflow definitions
type WorkflowValidator interface {
	Validate(workflow WorkflowSpec) error
	ValidateStep(step WorkflowStep) error
}