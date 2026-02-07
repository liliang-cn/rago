package agent

import (
	"time"
)

// EventType defines the type of event in the runtime loop
type EventType string

const (
	// Workflow Events
	EventTypeStart       EventType = "workflow_start"
	EventTypeComplete    EventType = "workflow_complete"
	EventTypeError       EventType = "workflow_error"

	// Thinking & Streaming
	EventTypeThinking    EventType = "thinking"     // Agent is processing
	EventTypePartial     EventType = "partial"      // Streaming text output
	
	// Tool Execution
	EventTypeToolCall    EventType = "tool_call"    // Agent requests tool execution
	EventTypeToolResult  EventType = "tool_result"  // Runner returns tool result
	
	// State Management
	EventTypeStateUpdate EventType = "state_update" // Request to update session state
	
	// Handoff
	EventTypeHandoff     EventType = "handoff"      // Transferring to another agent
)

// Event represents a discrete occurrence in the agent execution loop
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	AgentID   string                 `json:"agent_id,omitempty"`
	AgentName string                 `json:"agent_name,omitempty"`
	Content   string                 `json:"content,omitempty"` // For text/thinking
	
	// Tool data
	ToolName  string                 `json:"tool_name,omitempty"`
	ToolArgs  map[string]interface{} `json:"tool_args,omitempty"`
	ToolResult interface{}           `json:"tool_result,omitempty"`
	
	// State data
	StateDelta map[string]interface{} `json:"state_delta,omitempty"`
	
	Timestamp time.Time              `json:"timestamp"`
}

// NewEvent creates a basic event
func NewEvent(evtType EventType, agent *Agent) *Event {
	agentName := "System"
	agentID := "system"
	if agent != nil {
		agentName = agent.Name()
		agentID = agent.ID()
	}
	
	return &Event{
		Type:      evtType,
		AgentName: agentName,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}
}
