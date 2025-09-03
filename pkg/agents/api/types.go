package api

import (
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// Request types

// CreateAgentRequest represents a request to create a new agent
type CreateAgentRequest struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        types.AgentType     `json:"type"`
	Config      types.AgentConfig   `json:"config"`
	Workflow    types.WorkflowSpec  `json:"workflow"`
}

// UpdateAgentRequest represents a request to update an agent
type UpdateAgentRequest struct {
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	Status      types.AgentStatus    `json:"status,omitempty"`
	Config      *types.AgentConfig   `json:"config,omitempty"`
	Workflow    *types.WorkflowSpec  `json:"workflow,omitempty"`
}

// ExecuteAgentRequest represents a request to execute an agent
type ExecuteAgentRequest struct {
	Variables map[string]interface{} `json:"variables,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"` // Timeout in seconds
	UserID    string                 `json:"user_id,omitempty"`
}

// Response types

// CreateAgentResponse represents the response after creating an agent
type CreateAgentResponse struct {
	Agent   *types.Agent `json:"agent"`
	Message string       `json:"message"`
}

// ListAgentsResponse represents the response for listing agents
type ListAgentsResponse struct {
	Agents []*types.Agent `json:"agents"`
	Count  int            `json:"count"`
}

// ListExecutionsResponse represents the response for listing executions
type ListExecutionsResponse struct {
	Executions []*types.ExecutionResult `json:"executions"`
	Count      int                      `json:"count"`
}

// WorkflowTemplate represents a workflow template
type WorkflowTemplate struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    string             `json:"category"`
	Author      string             `json:"author,omitempty"`
	Version     string             `json:"version,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Workflow    types.WorkflowSpec `json:"workflow"`
}

// WorkflowTemplatesResponse represents the response for workflow templates
type WorkflowTemplatesResponse struct {
	Templates []WorkflowTemplate `json:"templates"`
	Count     int                `json:"count"`
}

// ValidateWorkflowResponse represents the response for workflow validation
type ValidateWorkflowResponse struct {
	Valid        bool   `json:"valid"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// AgentsStatusResponse represents the overall status of agents
type AgentsStatusResponse struct {
	TotalAgents int            `json:"total_agents"`
	StatusCount map[string]int `json:"status_count"`
	TypeCount   map[string]int `json:"type_count"`
	Timestamp   time.Time      `json:"timestamp"`
}

// ActiveExecutionInfo represents information about an active execution
type ActiveExecutionInfo struct {
	ExecutionID string        `json:"execution_id"`
	AgentID     string        `json:"agent_id"`
	AgentName   string        `json:"agent_name"`
	Status      string        `json:"status"`
	StartTime   time.Time     `json:"start_time"`
	CurrentStep string        `json:"current_step"`
	Duration    time.Duration `json:"duration"`
}

// ActiveExecutionsResponse represents the response for active executions
type ActiveExecutionsResponse struct {
	Executions []ActiveExecutionInfo `json:"executions"`
	Count      int                   `json:"count"`
}

// Error response types

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Pagination types

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int `json:"page,omitempty" form:"page"`
	PageSize int `json:"page_size,omitempty" form:"page_size"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// Filtering types

// AgentFilter represents filtering options for agents
type AgentFilter struct {
	Type   types.AgentType   `json:"type,omitempty" form:"type"`
	Status types.AgentStatus `json:"status,omitempty" form:"status"`
	Name   string            `json:"name,omitempty" form:"name"`
	Tag    string            `json:"tag,omitempty" form:"tag"`
}

// ExecutionFilter represents filtering options for executions
type ExecutionFilter struct {
	AgentID    string              `json:"agent_id,omitempty" form:"agent_id"`
	Status     types.ExecutionStatus `json:"status,omitempty" form:"status"`
	StartTime  *time.Time           `json:"start_time,omitempty" form:"start_time"`
	EndTime    *time.Time           `json:"end_time,omitempty" form:"end_time"`
}

// Sorting types

// SortOption represents sorting options
type SortOption struct {
	Field     string `json:"field" form:"sort_by"`
	Direction string `json:"direction" form:"sort_dir"` // "asc" or "desc"
}

// Batch operation types

// BatchAgentOperation represents a batch operation on agents
type BatchAgentOperation struct {
	Operation string   `json:"operation"` // "delete", "activate", "deactivate"
	AgentIDs  []string `json:"agent_ids"`
}

// BatchAgentResponse represents the response for batch operations
type BatchAgentResponse struct {
	Success     []string `json:"success"`
	Failed      []string `json:"failed"`
	Errors      []string `json:"errors,omitempty"`
	TotalCount  int      `json:"total_count"`
	SuccessCount int     `json:"success_count"`
	FailedCount int      `json:"failed_count"`
}

// Metrics types

// AgentMetrics represents metrics for an agent
type AgentMetrics struct {
	AgentID           string        `json:"agent_id"`
	TotalExecutions   int           `json:"total_executions"`
	SuccessfulExecutions int        `json:"successful_executions"`
	FailedExecutions  int           `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	LastExecutionTime *time.Time     `json:"last_execution_time,omitempty"`
	SuccessRate       float64       `json:"success_rate"`
}

// SystemMetrics represents overall system metrics
type SystemMetrics struct {
	TotalAgents        int           `json:"total_agents"`
	ActiveAgents       int           `json:"active_agents"`
	TotalExecutions    int           `json:"total_executions"`
	ActiveExecutions   int           `json:"active_executions"`
	ExecutionsLastHour int           `json:"executions_last_hour"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	SystemUptime       time.Duration `json:"system_uptime"`
	Timestamp          time.Time     `json:"timestamp"`
}

// Export/Import types

// AgentExport represents exported agent data
type AgentExport struct {
	Agents    []*types.Agent           `json:"agents"`
	Executions []*types.ExecutionResult `json:"executions,omitempty"`
	Templates []WorkflowTemplate        `json:"templates,omitempty"`
	Metadata  ExportMetadata           `json:"metadata"`
}

// ExportMetadata represents metadata for exports
type ExportMetadata struct {
	ExportTime    time.Time `json:"export_time"`
	Version       string    `json:"version"`
	TotalAgents   int       `json:"total_agents"`
	TotalExecutions int      `json:"total_executions"`
}

// WebSocket message types

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// WSExecutionUpdate represents an execution update via WebSocket
type WSExecutionUpdate struct {
	ExecutionID  string                 `json:"execution_id"`
	AgentID      string                 `json:"agent_id"`
	Status       types.ExecutionStatus   `json:"status"`
	CurrentStep  string                 `json:"current_step"`
	Progress     float64                `json:"progress"`
	Results      map[string]interface{} `json:"results,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// WSAgentStatus represents agent status update via WebSocket
type WSAgentStatus struct {
	AgentID   string            `json:"agent_id"`
	Status    types.AgentStatus `json:"status"`
	Message   string            `json:"message,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Configuration types

// APIConfig represents configuration for the agents API
type APIConfig struct {
	Host                  string        `json:"host" yaml:"host"`
	Port                  int           `json:"port" yaml:"port"`
	EnableCORS           bool          `json:"enable_cors" yaml:"enable_cors"`
	EnableMetrics        bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableWebSocket      bool          `json:"enable_websocket" yaml:"enable_websocket"`
	RateLimitEnabled     bool          `json:"rate_limit_enabled" yaml:"rate_limit_enabled"`
	RateLimitRPS         int           `json:"rate_limit_rps" yaml:"rate_limit_rps"`
	MaxRequestSize       int64         `json:"max_request_size" yaml:"max_request_size"`
	ReadTimeout          time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout         time.Duration `json:"write_timeout" yaml:"write_timeout"`
	ShutdownTimeout      time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}