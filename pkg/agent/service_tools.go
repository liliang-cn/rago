package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/skills"
)

// SearchAndExecute searches for tools matching the query and optionally executes them.
func (s *Service) SearchAndExecute(ctx context.Context, query string, instruction string) (interface{}, error) {
	query = strings.ToLower(query)
	keywords := strings.Fields(query)

	matches := s.toolRegistry.SearchDeferredTools(query)

	// Search MCP tools if available
	if s.mcpService != nil {
		for _, t := range s.mcpService.ListTools() {
			name := strings.ToLower(t.Function.Name)
			desc := strings.ToLower(t.Function.Description)

			matched := false
			for _, kw := range keywords {
				if strings.Contains(name, kw) || strings.Contains(desc, kw) {
					matched = true
					break
				}
			}
			if matched {
				t.DeferLoading = true
				matches = append(matches, t)
			}
		}
	}

	// Search Skills if available
	if s.skillsService != nil {
		if skillsList, err := s.skillsService.ListSkills(ctx, skills.SkillFilter{}); err == nil {
			for _, sk := range skillsList {
				name := strings.ToLower(sk.ID)
				desc := strings.ToLower(sk.Description)

				matched := false
				for _, kw := range keywords {
					if strings.Contains(name, kw) || strings.Contains(desc, kw) {
						matched = true
						break
					}
				}
				if matched {
					def := domain.ToolDefinition{
						Type: "function",
						Function: domain.ToolFunction{
							Name:        sk.ID,
							Description: sk.Description,
							Parameters:  map[string]interface{}{},
						},
						DeferLoading: true,
					}
					matches = append(matches, def)
				}
			}
		}
	}

	// Activate matching tools for current session
	sessionID := s.CurrentSessionID()
	for _, m := range matches {
		s.toolRegistry.ActivateForSession(sessionID, m.Function.Name)
	}

	// Automatic execution if instruction provided
	if instruction != "" {
		if s.llmService == nil {
			return nil, fmt.Errorf("LLM service not available for automatic execution")
		}

		messages := []domain.Message{
			{Role: "system", Content: "You are an intelligent executor. You MUST strictly call the appropriate tool to fulfill the user's instruction. Do not provide conversational filler, ONLY call the tool."},
			{Role: "user", Content: instruction},
		}

		opts := &domain.GenerationOptions{
			Temperature: 0.1,
			ToolChoice:  "auto",
		}

		result, err := s.llmService.GenerateWithTools(ctx, messages, matches, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to map instruction to tool: %w", err)
		}

		if len(result.ToolCalls) == 0 {
			return "Found tools: " + fmt.Sprintf("%v", getToolNames(matches)) + ". But could not map instruction to any of them.", nil
		}

		execResults, err := s.executeToolCalls(ctx, s.agent, result.ToolCalls)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		var finalResults []string
		for _, r := range execResults {
			finalResults = append(finalResults, fmt.Sprintf("Tool '%s' executed successfully. Result: %v", r.ToolName, r.Result))
		}
		return strings.Join(finalResults, "\n") + "\n\nPlease provide a final summary of these results to the user.", nil
	}

	// Just return found tools metadata
	if len(matches) == 0 {
		return "No tools found matching the query.", nil
	}

	var result []map[string]interface{}
	for _, m := range matches {
		result = append(result, map[string]interface{}{
			"name":        m.Function.Name,
			"description": m.Function.Description,
			"parameters":  m.Function.Parameters,
		})
	}
	return result, nil
}

func getToolNames(defs []domain.ToolDefinition) []string {
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Function.Name
	}
	return names
}
