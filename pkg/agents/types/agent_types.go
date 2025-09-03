package types

import (
	"time"
)

// Agent represents the core agent structure
type Agent struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Type        AgentType         `json:"type" yaml:"type"`
	Config      AgentConfig       `json:"config" yaml:"config"`
	Workflow    WorkflowSpec      `json:"workflow" yaml:"workflow"`
	Status      AgentStatus       `json:"status" yaml:"status"`
	CreatedAt   time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" yaml:"updated_at"`
}

// AgentType defines the category of agent
type AgentType string

const (
	AgentTypeResearch  AgentType = "research"
	AgentTypeWorkflow  AgentType = "workflow"
	AgentTypeMonitoring AgentType = "monitoring"
)

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusInactive AgentStatus = "inactive"
	AgentStatusError    AgentStatus = "error"
)

// AgentConfig holds agent configuration
type AgentConfig struct {
	MaxConcurrentExecutions int           `json:"max_concurrent_executions" yaml:"max_concurrent_executions"`
	DefaultTimeout          time.Duration `json:"default_timeout" yaml:"default_timeout"`
	EnableMetrics           bool          `json:"enable_metrics" yaml:"enable_metrics"`
	AutonomyLevel          AutonomyLevel  `json:"autonomy_level" yaml:"autonomy_level"`
}

// AutonomyLevel defines how autonomous an agent can be
type AutonomyLevel int

const (
	AutonomyManual     AutonomyLevel = iota // User-triggered only
	AutonomyScheduled                       // Time-based execution
	AutonomyReactive                        // Event-driven execution
	AutonomyProactive                       // Goal-seeking behavior
	AutonomyAdaptive                        // Learning and optimization
)

// String returns the string representation of AutonomyLevel
func (a AutonomyLevel) String() string {
	switch a {
	case AutonomyManual:
		return "manual"
	case AutonomyScheduled:
		return "scheduled"
	case AutonomyReactive:
		return "reactive"
	case AutonomyProactive:
		return "proactive"
	case AutonomyAdaptive:
		return "adaptive"
	default:
		return "unknown"
	}
}

// AgentInterface defines the contract that all agents must implement
type AgentInterface interface {
	GetID() string
	GetName() string
	GetType() AgentType
	GetStatus() AgentStatus
	GetAgent() *Agent
	Execute(ctx ExecutionContext) (*ExecutionResult, error)
	Validate() error
}

// ExecutionContext provides runtime context for agent execution
type ExecutionContext struct {
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id,omitempty"`
	Variables   map[string]interface{} `json:"variables"`
	StartTime   time.Time              `json:"start_time"`
	Timeout     time.Duration          `json:"timeout"`
	MCPClient   interface{}            `json:"-"` // MCP service interface
}

// ExecutionResult holds the results of agent execution
type ExecutionResult struct {
	ExecutionID  string                 `json:"execution_id"`
	AgentID      string                 `json:"agent_id"`
	Status       ExecutionStatus        `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Results      map[string]interface{} `json:"results"`
	Outputs      map[string]interface{} `json:"outputs"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Logs         []ExecutionLog         `json:"logs"`
	StepResults  []StepResult           `json:"step_results"`
}

// ExecutionStatus represents the state of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionLog represents a log entry during execution
type ExecutionLog struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`
	StepID    string      `json:"step_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// LogLevel defines logging levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// StepResult holds the result of a single workflow step
type StepResult struct {
	StepID       string                 `json:"step_id"`
	Name         string                 `json:"name"`
	Status       ExecutionStatus        `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}