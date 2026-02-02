package agent

import (
	"time"
)

// Step status constants
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"

	// Convenience aliases for UI compatibility
	StepPending   = StepStatusPending
	StepRunning   = StepStatusRunning
	StepCompleted = StepStatusCompleted
	StepFailed    = StepStatusFailed
	StepSkipped   = StepStatusSkipped
)

// Plan status constants
const (
	PlanStatusPending   = "pending"
	PlanStatusRunning   = "running"
	PlanStatusCompleted = "completed"
	PlanStatusFailed    = "failed"

	// Convenience aliases for UI compatibility
	StatusPending   = PlanStatusPending
	StatusRunning   = PlanStatusRunning
	StatusCompleted = PlanStatusCompleted
	StatusFailed    = PlanStatusFailed
)

// Step represents a single step in an agent's execution plan
type Step struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Tool        string                 `json:"tool"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
	Status      string                 `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty"` // IDs of steps this step depends on
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
}

// Plan represents an agent's execution plan for a goal
type Plan struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	SessionID string    `json:"session_id"`
	Steps     []Step    `json:"steps"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Reasoning string    `json:"reasoning,omitempty"` // LLM's reasoning for the plan
}

// ExecutionResult represents the result of an agent execution
type ExecutionResult struct {
	PlanID      string                 `json:"plan_id"`
	SessionID   string                 `json:"session_id"`
	Success     bool                   `json:"success"`
	StepsTotal  int                    `json:"steps_total"`
	StepsDone   int                    `json:"steps_done"`
	StepsFailed int                    `json:"steps_failed"`
	FinalResult interface{}            `json:"final_result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    string                 `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
