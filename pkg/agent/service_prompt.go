package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/prompt"
)

// buildSystemPrompt constructs the system prompt for the current agent.
// ctx is required when PTC is enabled so available callTool() names can be listed dynamically.
func (s *Service) buildSystemPrompt(ctx context.Context, agent *Agent) string {
	systemCtx := s.buildSystemContext()

	data := map[string]interface{}{
		"AgentInstructions": agent.Instructions(),
		"SystemContext":     systemCtx.FormatForPrompt(),
	}

	rendered, err := s.promptManager.Render(prompt.AgentSystemPrompt, data)
	if err != nil {
		// Fallback
		rendered = agent.Instructions() + "\n\n" + systemCtx.FormatForPrompt()
	}

	// Append PTC instructions when enabled so the LLM knows how to use execute_javascript.
	// Dynamically list what is callable via callTool() so the model doesn't have to guess.
	if s.ptcIntegration != nil {
		availableCallTools := s.ptcAvailableCallTools(ctx)
		if ptcPrompt := s.ptcIntegration.GetPTCSystemPrompt(availableCallTools); ptcPrompt != "" {
			rendered += "\n\n" + ptcPrompt
		}
	}

	if summary := s.buildToolCatalogSummary(ctx); summary != "" {
		rendered += "\n\n" + summary
	}

	if note := s.buildWebSearchPromptNote(); note != "" {
		rendered += "\n\n" + note
	}

	return rendered
}

// buildEnrichedPrompt builds a prompt enriched with memory and RAG results
func (s *Service) buildEnrichedPrompt(goal, memoryContext, ragResult string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User Question: %s\n\n", goal))

	if memoryContext != "" {
		prompt.WriteString("--- Relevant Memory ---\n")
		prompt.WriteString(memoryContext)
		prompt.WriteString("\n\n")
	}

	if ragResult != "" {
		prompt.WriteString("--- Knowledge Base Results ---\n")
		prompt.WriteString(ragResult)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please answer the user's question based on the memory and knowledge base information above.")
	prompt.WriteString(" If there's no relevant information, say so honestly.")

	return prompt.String()
}

// buildPTCSystemPrompt builds the system prompt with PTC instructions
func (s *Service) buildPTCSystemPrompt(ctx context.Context) string {
	var sb strings.Builder

	// Base agent instructions
	if s.agent != nil {
		sb.WriteString(s.agent.Instructions())
		sb.WriteString("\n\n")
	}

	// PTC instructions with dynamic tool list
	if s.ptcIntegration != nil && s.ptcIntegration.config.Enabled {
		availableCallTools := s.ptcAvailableCallTools(ctx)
		sb.WriteString(s.ptcIntegration.GetPTCSystemPrompt(availableCallTools))
	}

	if summary := s.buildToolCatalogSummary(ctx); summary != "" {
		sb.WriteString("\n")
		sb.WriteString(summary)
	}

	if note := s.buildWebSearchPromptNote(); note != "" {
		sb.WriteString("\n\n")
		sb.WriteString(note)
	}

	return sb.String()
}
