package agent

import (
	"context"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/skills"
)

// addRAGSources adds sources with deduplication by ID
func (s *Service) addRAGSources(sources []domain.Chunk) {
	if len(sources) == 0 {
		return
	}
	s.ragSourcesMu.Lock()
	defer s.ragSourcesMu.Unlock()

	// Build map of existing IDs
	existing := make(map[string]bool)
	for _, src := range s.ragSources {
		existing[src.ID] = true
	}

	// Add only new sources
	for _, src := range sources {
		if !existing[src.ID] {
			s.ragSources = append(s.ragSources, src)
			existing[src.ID] = true
		}
	}
}

// collectAllAvailableTools collects tools from MCP, Skills, RAG, and Agent Handoffs.
// When PTC is enabled, RAG/MCP/Skills are NOT exposed as direct function-call tools —
// the LLM must call them through execute_javascript + callTool(), mirroring Anthropic's
// allowed_callers: ["code_execution"] behaviour where direct model invocation is removed.
func (s *Service) collectAllAvailableTools(ctx context.Context, currentAgent *Agent) []domain.ToolDefinition {
	toolsMap := make(map[string]domain.ToolDefinition)
	ptcEnabled := s.isPTCEnabled()
	sessionID := s.CurrentSessionID()

	// Helper to add tools with deduplication
	addTools := func(defs []domain.ToolDefinition) {
		for _, d := range defs {
			toolsMap[d.Function.Name] = d
		}
	}

	// 1. Add static tools and active deferred tools from Registry
	addTools(s.toolRegistry.ListForLLM(ptcEnabled, sessionID))

	// Agent Handoffs — always visible so the LLM can route between agents.
	if currentAgent != nil {
		for _, handoff := range currentAgent.Handoffs() {
			tool := handoff.ToToolDefinition().ToDomainTool()
			toolsMap[tool.Function.Name] = tool
		}
		// Per-agent custom tools (e.g. tools added directly to an Agent in multi-agent
		// scenarios) — hidden when PTC is enabled.
		if !ptcEnabled {
			for _, def := range currentAgent.Tools() {
				// Skip if already in registry (AddTool registers in both places).
				if !s.toolRegistry.Has(def.Function.Name) {
					toolsMap[def.Function.Name] = def
				}
			}
		}
	}

	// MCP tools — dynamic (servers may change at runtime); hidden in PTC mode.
	if s.mcpService != nil && !ptcEnabled {
		allMCP := s.mcpService.ListTools()
		activeMap := s.toolRegistry.sessionActivated[sessionID]
		deferAllMCP := len(allMCP) > 5 // Automatically defer if there are many tools

		if currentAgent == nil || isAllAllowed(currentAgent.mcpTools) {
			for _, tool := range allMCP {
				if !deferAllMCP || (activeMap != nil && activeMap[tool.Function.Name]) {
					addTools([]domain.ToolDefinition{tool})
				}
			}
		} else {
			for _, tool := range allMCP {
				if containsStr(currentAgent.mcpTools, tool.Function.Name) {
					if !deferAllMCP || (activeMap != nil && activeMap[tool.Function.Name]) {
						addTools([]domain.ToolDefinition{tool})
					}
				}
			}
		}
	}

	// Skills tools — dynamic; hidden in PTC mode.
	if s.skillsService != nil && !ptcEnabled {
		skillsList, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
		activeMap := s.toolRegistry.sessionActivated[sessionID]
		deferAllSkills := len(skillsList) > 5

		allowedAll := currentAgent == nil || isAllAllowed(currentAgent.skills)
		for _, sk := range skillsList {
			if allowedAll || containsStr(currentAgent.skills, sk.ID) {
				if !deferAllSkills || (activeMap != nil && activeMap[sk.ID]) {
					toolsMap[sk.ID] = domain.ToolDefinition{
						Type: "function",
						Function: domain.ToolFunction{
							Name:        sk.ID,
							Description: sk.Description,
							Parameters:  map[string]interface{}{},
						},
					}
				}
			}
		}
	}

	// SubAgent delegation — hidden in PTC mode.
	if !ptcEnabled {
		toolsMap["delegate_to_subagent"] = domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "delegate_to_subagent",
				Description: "Delegate a specific task to a sub-agent. The sub-agent will execute the task with a subset of available tools and return the result. Use this for focused, isolated tasks.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"goal": map[string]interface{}{
							"type":        "string",
							"description": "The specific task/goal for the sub-agent to accomplish",
						},
						"tools_allowlist": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Optional list of tool names the sub-agent is allowed to use. If not specified, all tools are available.",
						},
						"tools_denylist": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Optional list of tool names the sub-agent is NOT allowed to use",
						},
						"max_turns": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of turns for the sub-agent (default: 5)",
						},
						"timeout_seconds": map[string]interface{}{
							"type":        "integer",
							"description": "Timeout in seconds for the sub-agent execution (default: 60)",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Optional additional context to pass to the sub-agent",
						},
					},
					"required": []string{"goal"},
				},
			},
		}
	}

	// PTC: expose execute_javascript as a direct LLM tool. Embed the dynamic
	// callTool() list so the model knows exactly what it can call.
	if s.ptcIntegration != nil {
		availableCallTools := s.ptcIntegration.GetAvailableCallTools(ctx)
		addTools(s.ptcIntegration.GetPTCTools(availableCallTools))
	}

	// Convert map to slice
	var tools []domain.ToolDefinition
	for _, tool := range toolsMap {
		tools = append(tools, tool)
	}

	return tools
}

func (s *Service) resetRunMemorySaved() {
	s.memorySaveMu.Lock()
	s.memorySavedInRun = false
	s.memorySaveMu.Unlock()
}

func (s *Service) markRunMemorySaved() {
	s.memorySaveMu.Lock()
	s.memorySavedInRun = true
	s.memorySaveMu.Unlock()
}

func (s *Service) hasRunMemorySaved() bool {
	s.memorySaveMu.RLock()
	defer s.memorySaveMu.RUnlock()
	return s.memorySavedInRun
}

// containsStr checks if a string slice contains a string
func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// isMCPTool checks if a tool name is from MCP
func (s *Service) isMCPTool(name string) bool {
	if s.mcpService == nil {
		return false
	}
	for _, tool := range s.mcpService.ListTools() {
		if tool.Function.Name == name {
			return true
		}
	}
	return false
}

// isSkill checks if a tool name is a skill
func (s *Service) isSkill(ctx context.Context, name string) bool {
	if s.skillsService == nil {
		return false
	}
	// Remove "skill_" prefix if present
	skillID := strings.TrimPrefix(name, "skill_")
	skills, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
	for _, sk := range skills {
		if sk.ID == skillID {
			return true
		}
	}
	return false
}

// collectAvailableTools collects tools from all available sources
func collectAvailableTools(mcpService MCPToolExecutor, ragProcessor domain.Processor, skillsService *skills.Service) []domain.ToolDefinition {
	tools := []domain.ToolDefinition{}

	// Add RAG tools
	if ragProcessor != nil {
		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_query",
				Description: "Query the RAG system to retrieve relevant document chunks",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query",
						},
						"top_k": map[string]interface{}{
							"type":        "integer",
							"description": "Number of results to return",
							"default":     5,
						},
					},
					"required": []string{"query"},
				},
			},
		})

		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_ingest",
				Description: "Ingest a document into the RAG system",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The document content",
						},
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the document file",
						},
					},
				},
			},
		})
	}

	// Add Skills tools
	if skillsService != nil {
		skillTools, err := skillsService.RegisterAsMCPTools()
		if err == nil {
			tools = append(tools, skillTools...)
		}
	}

	// Add MCP tools
	if mcpService != nil {
		mcpTools := mcpService.ListTools()
		tools = append(tools, mcpTools...)
	}

	// Add general LLM tool
	tools = append(tools, domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "llm",
			Description: "General LLM reasoning and text generation",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The prompt for the LLM",
					},
					"temperature": map[string]interface{}{
						"type":        "number",
						"description": "Temperature for generation (0-1)",
						"default":     0.7,
					},
				},
				"required": []string{"prompt"},
			},
		},
	})

	return tools
}

func truncateGoal(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
