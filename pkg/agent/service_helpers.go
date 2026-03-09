package agent

import (
	"context"
	"fmt"
	"os"
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
	// This includes built-in tools like delegate_to_subagent and task_complete
	addTools(s.toolRegistry.ListForLLM(ptcEnabled, sessionID))

	// Check if there are deferred tools - if so, add tool search tools
	hasDeferredTools := false
	for _, t := range s.toolRegistry.ListForLLM(ptcEnabled, sessionID) {
		if t.DeferLoading {
			hasDeferredTools = true
			break
		}
	}
	// Also check MCP and skills for deferred tools
	if !hasDeferredTools && s.mcpService != nil {
		// If MCP has many tools, consider them deferred
		if len(s.mcpService.ListTools()) > 5 {
			hasDeferredTools = true
		}
	}
	if !hasDeferredTools && s.skillsService != nil {
		skillsList, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
		if len(skillsList) > 5 {
			hasDeferredTools = true
		}
	}
	// Add tool search tools if there are deferred tools
	if hasDeferredTools && !ptcEnabled {
		for _, ts := range GetToolSearchTools() {
			toolsMap[ts.Function.Name] = ts
		}
	}

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
					// Set DeferLoading based on whether we're deferring
					t := tool
					if deferAllMCP {
						t.DeferLoading = true
					}
					addTools([]domain.ToolDefinition{t})
				}
			}
		} else {
			for _, tool := range allMCP {
				if containsStr(currentAgent.mcpTools, tool.Function.Name) {
					if !deferAllMCP || (activeMap != nil && activeMap[tool.Function.Name]) {
						// Set DeferLoading based on whether we're deferring
						t := tool
						if deferAllMCP {
							t.DeferLoading = true
						}
						addTools([]domain.ToolDefinition{t})
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
			// Skip if disabled or explicitly hidden from model invocation
			if !sk.Enabled || sk.DisableModelInvocation {
				continue
			}

			if allowedAll || containsStr(currentAgent.skills, sk.ID) {
				if !deferAllSkills || (activeMap != nil && activeMap[sk.ID]) {
					// Build variable schema from skill definition
					properties := make(map[string]interface{})
					required := make([]string, 0)
					for _, v := range sk.Variables {
						prop := map[string]interface{}{
							"type":        getSkillVarTypeString(v.Type),
							"description": v.Description,
						}
						if v.Default != nil {
							prop["default"] = v.Default
						}
						properties[v.Name] = prop
						if v.Required {
							required = append(required, v.Name)
						}
					}

					desc := sk.Description
					if desc == "" {
						desc = sk.Name
					}
					// Clarify that calling this skill returns its workflow instructions.
					desc = "Skill workflow: " + desc + ". Call this tool to receive step-by-step instructions for this task; you MUST then follow those instructions to complete the work."

					// Use "skill_" prefix to match RegisterAsMCPTools and isSkill check
					toolName := "skill_" + sk.ID
					// Set DeferLoading based on whether we're deferring skills
					deferLoading := deferAllSkills
					toolsMap[toolName] = domain.ToolDefinition{
						Type:         "function",
						DeferLoading: deferLoading,
						Function: domain.ToolFunction{
							Name:        toolName,
							Description: desc,
							Parameters: map[string]interface{}{
								"type":       "object",
								"properties": properties,
								"required":   required,
							},
						},
					}
				}
			}
		}
	}

	// PTC: expose execute_javascript as a direct LLM tool. Embed the dynamic
	// callTool() list so the model knows exactly what it can call.
	if s.ptcIntegration != nil {
		availableCallTools := s.ptcIntegration.GetAvailableCallTools(ctx)
		addTools(s.ptcIntegration.GetPTCTools(availableCallTools))
	}

	// 4. Convert map back to slice
	tools := make([]domain.ToolDefinition, 0, len(toolsMap))
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

// executeToolViaSubAgent runs a tool or skill call using a separate SubAgent goroutine
func (s *Service) executeToolViaSubAgent(ctx context.Context, currentAgent *Agent, session *Session, tc domain.ToolCall) (interface{}, error, bool) {
	// Create subagent config
	subCfg := SubAgentConfig{
		Agent:         currentAgent,
		ParentSession: session,
		Goal:          fmt.Sprintf("Execute tool: %s", tc.Function.Name),
		Service:       s,
		ToolCall:      &tc,
	}

	sa := NewSubAgent(subCfg)

	// Run subagent
	result, err := sa.Run(ctx)

	// Check if this was a handoff
	isHandoff := strings.HasPrefix(tc.Function.Name, "transfer_to_") && err == nil

	return result, err, isHandoff
}

// EmitDebugPrint prints formatted debug information to console if debug mode is enabled.
// This ensures consistent look across different execution paths (Execute, Run, RunStream).
func (s *Service) EmitDebugPrint(round int, debugType string, content string) {
	if !s.debug {
		return
	}

	sep := strings.Repeat("─", 60)
	label := strings.ToUpper(debugType)

	fmt.Fprintf(os.Stderr, "\n\033[2m%s\n🐛 DEBUG [Round %d] %s\n%s\n%s\n%s\033[0m\n",
		sep, round, label, sep, content, sep)
}

func truncateGoal(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getSkillVarTypeString(typ string) string {
	switch typ {
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}
