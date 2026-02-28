package agent

import "fmt"

// AgentError is the base type for agent-related errors
type AgentError struct {
	Message string
	Cause   error
}

func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AgentError) Unwrap() error {
	return e.Cause
}

// MaxTurnsExceeded is raised when the agent exceeds the maximum number of turns
type MaxTurnsExceeded struct {
	MaxTurns     int
	CurrentRound int
	Goal         string
}

func (e *MaxTurnsExceeded) Error() string {
	return fmt.Sprintf("agent exceeded maximum turns: completed %d rounds out of %d allowed while working on: %s",
		e.CurrentRound, e.MaxTurns, e.Goal)
}

// ModelBehaviorError indicates unexpected or invalid LLM outputs
type ModelBehaviorError struct {
	Message string
	Details string
}

func (e *ModelBehaviorError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("model behavior error: %s (%s)", e.Message, e.Details)
	}
	return fmt.Sprintf("model behavior error: %s", e.Message)
}

// ToolExecutionError indicates a tool execution failure
type ToolExecutionError struct {
	ToolName string
	Message  string
	Cause    error
}

func (e *ToolExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' execution failed: %s: %v", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s' execution failed: %s", e.ToolName, e.Message)
}

func (e *ToolExecutionError) Unwrap() error {
	return e.Cause
}

// NewMaxTurnsExceeded creates a MaxTurnsExceeded error
func NewMaxTurnsExceeded(maxTurns, currentRound int, goal string) *MaxTurnsExceeded {
	return &MaxTurnsExceeded{
		MaxTurns:     maxTurns,
		CurrentRound: currentRound,
		Goal:         goal,
	}
}
