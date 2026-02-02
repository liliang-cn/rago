package agent

import (
	"context"
	"fmt"
)

// HandoffExecutor handles agent-to-agent delegation
type HandoffExecutor struct {
	agents   map[string]*Agent
	handoffs map[string]*Handoff
}

// NewHandoffExecutor creates a new handoff executor
func NewHandoffExecutor() *HandoffExecutor {
	return &HandoffExecutor{
		agents:   make(map[string]*Agent),
		handoffs: make(map[string]*Handoff),
	}
}

// RegisterAgent registers an agent for potential handoffs
func (he *HandoffExecutor) RegisterAgent(agent *Agent) {
	he.agents[agent.ID()] = agent
}

// RegisterHandoff registers a handoff configuration
func (he *HandoffExecutor) RegisterHandoff(handoff *Handoff) {
	he.handoffs[handoff.ID()] = handoff
	if handoff.TargetAgent() != nil {
		he.RegisterAgent(handoff.TargetAgent())
	}
}

// GetHandoffByToolName finds a handoff by its tool name
func (he *HandoffExecutor) GetHandoffByToolName(toolName string) (*Handoff, bool) {
	for _, h := range he.handoffs {
		if h.ToolName() == toolName {
			return h, true
		}
	}
	return nil, false
}

// GetAgent retrieves an agent by ID
func (he *HandoffExecutor) GetAgent(id string) (*Agent, bool) {
	agent, ok := he.agents[id]
	return agent, ok
}

// ListHandoffs returns all registered handoffs
func (he *HandoffExecutor) ListHandoffs() []*Handoff {
	handoffs := make([]*Handoff, 0, len(he.handoffs))
	for _, h := range he.handoffs {
		handoffs = append(handoffs, h)
	}
	return handoffs
}

// ExecuteHandoff executes a handoff to another agent
func (he *HandoffExecutor) ExecuteHandoff(
	ctx context.Context,
	handoff *Handoff,
	data HandoffInputData,
	session *Session,
) (*ExecutionResult, error) {
	// Check if handoff is enabled
	if !handoff.IsEnabled(ctx) {
		return nil, fmt.Errorf("handoff %s is not enabled", handoff.ID())
	}

	// Execute on_handoff callback
	if err := handoff.OnHandoff(ctx, data); err != nil {
		return nil, fmt.Errorf("on_handoff callback failed: %w", err)
	}

	// Apply input filter
	filteredData, err := handoff.ApplyInputFilter(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("input filter failed: %w", err)
	}

	// Get target agent
	targetAgent := handoff.TargetAgent()
	if targetAgent == nil {
		return nil, fmt.Errorf("target agent not found for handoff %s", handoff.ID())
	}

	// Create or update session for the target agent
	targetSession := session
	if handoff.NestHandoffHistory() {
		// Add handoff marker to conversation
		session.AddHandoffMessage(HandoffMessage{
			Role:    "system",
			Content: fmt.Sprintf("<HANDOFF from=%s to=%s reason=%s>", data.SourceAgent, targetAgent.Name(), data.Goal),
		})
	}

	// Note: The actual agent execution would be handled by the AgentRunner
	// This executor just sets up the handoff context
	return &ExecutionResult{
		SessionID: targetSession.GetID(),
		Success:   true,
		Metadata: map[string]interface{}{
			"handoff_id":    handoff.ID(),
			"target_agent":  targetAgent.Name(),
			"source_agent":  data.SourceAgent,
			"handed_off_at": filteredData,
		},
	}, nil
}

// HandoffFromAgents creates handoffs from a list of agents
func (he *HandoffExecutor) HandoffFromAgents(agents []*Agent, opts ...HandoffOption) {
	for _, agent := range agents {
		handoff := NewHandoff(agent, opts...)
		he.RegisterHandoff(handoff)
	}
}

// HandoffChain creates a chain of handoffs (sequential delegation)
type HandoffChain struct {
	handoffs []*Handoff
}

// NewHandoffChain creates a new handoff chain
func NewHandoffChain(handoffs ...*Handoff) *HandoffChain {
	return &HandoffChain{
		handoffs: handoffs,
	}
}

// Add adds a handoff to the chain
func (hc *HandoffChain) Add(handoff *Handoff) *HandoffChain {
	hc.handoffs = append(hc.handoffs, handoff)
	return hc
}

// ToToolDefinitions converts all handoffs in the chain to tool definitions
func (hc *HandoffChain) ToToolDefinitions() []ToolDef {
	tools := make([]ToolDef, len(hc.handoffs))
	for i, h := range hc.handoffs {
		tools[i] = h.ToToolDefinition()
	}
	return tools
}

// HandoffFilters provides common input filter implementations
type HandoffFilters struct{}

// RemoveToolCalls removes all tool calls from the input data
func (hf HandoffFilters) RemoveToolCalls(ctx context.Context, data HandoffInputData) (HandoffInputData, error) {
	filtered := HandoffInputData{
		Goal:        data.Goal,
		Context:     data.Context,
		SourceAgent: data.SourceAgent,
		TargetAgent: data.TargetAgent,
	}
	// Filter messages to remove tool calls
	for _, msg := range data.Messages {
		filtered.Messages = append(filtered.Messages, HandoffMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCalls: nil, // Remove tool calls
			Metadata:  msg.Metadata,
		})
	}
	return filtered, nil
}

// RemoveSystemMessages removes system messages from input
func (hf HandoffFilters) RemoveSystemMessages(ctx context.Context, data HandoffInputData) (HandoffInputData, error) {
	filtered := data
	filtered.Messages = []HandoffMessage{}
	for _, msg := range data.Messages {
		if msg.Role != "system" {
			filtered.Messages = append(filtered.Messages, msg)
		}
	}
	return filtered, nil
}

// SummarizeHistory summarizes the conversation history
func (hf HandoffFilters) SummarizeHistory(ctx context.Context, data HandoffInputData, summary string) (HandoffInputData, error) {
	filtered := data
	filtered.Messages = []HandoffMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf("<CONVERSATION HISTORY>\n%s\n</CONVERSATION HISTORY>", summary),
		},
	}
	return filtered, nil
}

// RemoveAllTools is a convenience function that removes all tool calls
var RemoveAllTools = HandoffFilters{}.RemoveToolCalls

// RemoveSystem is a convenience function that removes system messages
var RemoveSystem = HandoffFilters{}.RemoveSystemMessages

// HandoffPrompt provides recommended prompt prefixes for agents with handoffs
type HandoffPrompt struct{}

// RecommendedPrefix returns the recommended prefix for agent instructions
func (hp HandoffPrompt) RecommendedPrefix() string {
	return `You have the ability to hand off tasks to other specialized agents.
When a task would be better handled by another agent, use the appropriate handoff tool.
`
}

// PromptWithHandoffInstructions adds handoff instructions to a prompt
func PromptWithHandoffInstructions(basePrompt string, handoffs []*Handoff) string {
	if len(handoffs) == 0 {
		return basePrompt
	}

	handoffInfo := "\n\nAvailable handoffs:\n"
	for _, h := range handoffs {
		handoffInfo += fmt.Sprintf("- %s: %s\n", h.ToolName(), h.ToolDescription())
	}

	return basePrompt + HandoffPrompt{}.RecommendedPrefix() + handoffInfo
}
