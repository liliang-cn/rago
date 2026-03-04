package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/prompt"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"golang.org/x/sync/errgroup"
)

// executeWithLLM lets LLM decide which tool to use and executes with multi-round support
func (s *Service) executeWithLLM(ctx context.Context, goal string, intent *IntentRecognitionResult, session *Session, memoryContext string, ragContext string, cfg *RunConfig) (interface{}, error) {
	maxRounds := cfg.MaxTurns
	if maxRounds <= 0 {
		maxRounds = 20
	}

	// Determine starting agent
	currentAgent := s.agent
	if session != nil && session.AgentID != "" && s.registry != nil {
		if a, ok := s.registry.GetAgent(session.AgentID); ok {
			currentAgent = a
		}
	}

	prevToolCalls := make(map[string]int)
	messages := s.buildConversationMessages(goal, ragContext, memoryContext)

	if cfg.StoreHistory && s.historyStore != nil {
		s.historyStore.RecordMessage(ctx, session.GetID(), currentAgent.ID(), goal, messages[0], 0)
	}

	toolCallCount := 0

	for round := 0; round < maxRounds; round++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution cancelled by user")
		default:
		}

		s.emitProgress("thinking", fmt.Sprintf("[%s] Thinking...", currentAgent.Name()), round+1, "")

		result, err := s.runOneLLMTurn(ctx, currentAgent, messages, cfg, round)
		if err != nil {
			return nil, err
		}

		if len(result.ToolCalls) > 0 {
			// Check for handoff first
			if newAgent, updated := s.applyHandoff(ctx, &messages, currentAgent, result, session, round); updated {
				currentAgent = newAgent
				continue
			}

			// Detect duplicate tool calls
			if s.isDuplicateToolCall(result.ToolCalls, prevToolCalls) {
				return "The task has been completed. The information has been saved to memory.", nil
			}

			// Execute tool calls and append results to messages
			s.emitProgress("tool_call", fmt.Sprintf("Calling %d tool(s)", len(result.ToolCalls)), round+1, "")
			toolResults, err := s.executeToolCalls(ctx, currentAgent, result.ToolCalls)
			if err != nil {
				messages = append(messages, domain.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Tool execution failed: %v", err),
				})
				continue
			}

			messages = s.appendToolRoundToMessages(messages, result, toolResults)
			s.recordToolResults(ctx, session, currentAgent, goal, toolResults, cfg, round)
			toolCallCount += len(toolResults)
			continue
		}

		// No tool calls — done
		if cfg.StoreHistory && s.historyStore != nil {
			s.historyStore.CompleteSession(ctx, session.GetID(), currentAgent.ID(), goal, round+1, toolCallCount, true, 0)
		}
		return result.Content, nil
	}

	return s.handleMaxTurnsExceeded(ctx, session, currentAgent, goal, maxRounds, toolCallCount, messages, cfg)
}

// buildConversationMessages constructs the initial user message, enriched with RAG and memory context.
func (s *Service) buildConversationMessages(goal, ragContext, memoryContext string) []domain.Message {
	content := goal
	if ragContext != "" {
		content += "\n\n--- Relevant documents from knowledge base ---\n" + ragContext + "\n--- End of documents ---"
	}
	if memoryContext != "" {
		content += "\n\nRelevant context from memory:\n" + memoryContext
	}
	return []domain.Message{{Role: "user", Content: content}}
}

// runOneLLMTurn builds the prompt for this round and calls the LLM once.
func (s *Service) runOneLLMTurn(ctx context.Context, currentAgent *Agent, messages []domain.Message, cfg *RunConfig, round int) (*domain.GenerationResult, error) {
	tools := s.collectAllAvailableTools(ctx, currentAgent)
	systemMsg := s.buildSystemPrompt(ctx, currentAgent)
	genMessages := append([]domain.Message{{Role: "system", Content: systemMsg}}, messages...)

	if s.debug || cfg.Debug {
		s.logDebugPrompt(genMessages, round)
	}

	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.3
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	result, err := s.llmService.GenerateWithTools(ctx, genMessages, tools, &domain.GenerationOptions{
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("LLM generation returned nil result")
	}

	if (s.debug || cfg.Debug) && err == nil {
		s.logDebugResponse(result, round)
	}
	return result, nil
}

// applyHandoff checks if any tool call is a handoff, applies it, and returns (newAgent, true) if so.
func (s *Service) applyHandoff(ctx context.Context, messages *[]domain.Message, currentAgent *Agent, result *domain.GenerationResult, session *Session, round int) (*Agent, bool) {
	for _, tc := range result.ToolCalls {
		if !strings.HasPrefix(tc.Function.Name, "transfer_to_") {
			continue
		}
		for _, h := range currentAgent.Handoffs() {
			if h.ToolName() != tc.Function.Name {
				continue
			}
			targetAgent := h.TargetAgent()
			reason := tc.Function.Arguments["reason"]
			s.emitProgress("tool_call", fmt.Sprintf("Transferring to %s", targetAgent.Name()), round+1, "handoff")

			if session != nil {
				session.AgentID = targetAgent.ID()
			}
			*messages = append(*messages,
				domain.Message{
					Role:             "assistant",
					Content:          result.Content,
					ReasoningContent: result.ReasoningContent,
					ToolCalls:        result.ToolCalls,
				},
				domain.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Transferred to %s. Reason: %v", targetAgent.Name(), reason),
				},
			)
			return targetAgent, true
		}
	}
	return currentAgent, false
}

// isDuplicateToolCall returns true if any call in toolCalls has been seen before.
func (s *Service) isDuplicateToolCall(toolCalls []domain.ToolCall, seen map[string]int) bool {
	for _, tc := range toolCalls {
		key := fmt.Sprintf("%s:%v", tc.Function.Name, tc.Function.Arguments)
		seen[key]++
		if seen[key] > 1 {
			log.Printf("[Agent] Duplicate tool call detected: %s, stopping", key)
			return true
		}
	}
	return false
}

// toolResultToString converts a tool execution result to a string suitable for
// the LLM's "tool" role message. Strings are returned as-is; maps and slices
// are JSON-encoded so the LLM receives well-structured output rather than Go's
// fmt.Sprintf("%v") representation (e.g. "map[key:value]").
func toolResultToString(result interface{}) string {
	switch v := result.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
}

// appendToolRoundToMessages appends the assistant message and tool result messages.
func (s *Service) appendToolRoundToMessages(messages []domain.Message, result *domain.GenerationResult, toolResults []ToolExecutionResult) []domain.Message {
	messages = append(messages, domain.Message{
		Role:             "assistant",
		Content:          result.Content,
		ReasoningContent: result.ReasoningContent,
		ToolCalls:        result.ToolCalls,
	})
	for _, tr := range toolResults {
		resStr := toolResultToString(tr.Result)
		messages = append(messages, domain.Message{
			Role:       "tool",
			Content:    resStr,
			ToolCallID: tr.ToolCallID,
		})
	}
	return messages
}

// recordToolResults writes tool results to history store if enabled.
func (s *Service) recordToolResults(ctx context.Context, session *Session, agent *Agent, goal string, toolResults []ToolExecutionResult, cfg *RunConfig, round int) {
	if !cfg.StoreHistory || s.historyStore == nil {
		return
	}
	for _, tr := range toolResults {
		success := true
		var errMsg string
		if errMap, ok := tr.Result.(map[string]interface{}); ok {
			if errVal, exists := errMap["error"]; exists && errVal != nil {
				success = false
				errMsg = fmt.Sprintf("%v", errVal)
			}
		}
		s.historyStore.RecordToolResult(ctx, session.GetID(), agent.ID(), goal,
			tr.ToolName, tr.ToolCallID, nil, tr.Result, success, errMsg, round+1)
	}
}

// handleMaxTurnsExceeded handles the case where max turns is reached.
func (s *Service) handleMaxTurnsExceeded(ctx context.Context, session *Session, agent *Agent, goal string, maxRounds, toolCallCount int, messages []domain.Message, cfg *RunConfig) (interface{}, error) {
	errExceeded := NewMaxTurnsExceeded(maxRounds, maxRounds, goal)
	if handler, ok := cfg.ErrorHandlers["max_turns"]; ok {
		handlerResult := handler(ErrorHandlerInput{
			Kind:         "max_turns",
			Round:        maxRounds,
			MaxTurns:     maxRounds,
			MessageCount: len(messages),
			Goal:         goal,
		})
		if cfg.StoreHistory && s.historyStore != nil {
			s.historyStore.CompleteSession(ctx, session.GetID(), agent.ID(), goal, maxRounds, toolCallCount, handlerResult.FinalOutput != nil, 0)
		}
		if handlerResult.FinalOutput != nil {
			return handlerResult.FinalOutput, nil
		}
		if handlerResult.Error != nil {
			return nil, handlerResult.Error
		}
	}
	if cfg.StoreHistory && s.historyStore != nil {
		s.historyStore.CompleteSession(ctx, session.GetID(), agent.ID(), goal, maxRounds, toolCallCount, false, 0)
	}
	return nil, errExceeded
}

// logDebugPrompt logs the full prompt for debugging.
func (s *Service) logDebugPrompt(genMessages []domain.Message, round int) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Printf("DEBUG: [ROUND %d] LLM FULL PROMPT\n", round+1)
	fmt.Println(strings.Repeat("-", 40))
	for _, m := range genMessages {
		fmt.Printf("[%s]:\n%s\n", strings.ToUpper(m.Role), m.Content)
		if len(m.ToolCalls) > 0 {
			fmt.Printf("  (ToolCalls: %d)\n", len(m.ToolCalls))
		}
	}
	fmt.Println(strings.Repeat("=", 40) + "\n")
}

// logDebugResponse logs the raw LLM response for debugging.
func (s *Service) logDebugResponse(result *domain.GenerationResult, round int) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Printf("DEBUG: [ROUND %d] LLM RAW RESPONSE\n", round+1)
	fmt.Println(strings.Repeat("-", 40))
	if result.ReasoningContent != "" {
		fmt.Printf("REASONING: %s\n", result.ReasoningContent)
	}
	fmt.Printf("CONTENT: %s\n", result.Content)
	if len(result.ToolCalls) > 0 {
		fmt.Println("TOOL CALLS:")
		for _, tc := range result.ToolCalls {
			fmt.Printf("  - %s(%v)\n", tc.Function.Name, tc.Function.Arguments)
		}
	}
	fmt.Println(strings.Repeat("=", 40) + "\n")
}

// verifyResult verifies the result with LLM
// Returns: (verified bool, reason string, correctedResult interface{}, err error)
func (s *Service) verifyResult(ctx context.Context, goal string, result interface{}) (bool, string, interface{}, error) {
	resultStr := formatResultForContent(result)

	data := map[string]interface{}{
		"Goal":   goal,
		"Result": resultStr,
	}

	rendered, err := s.promptManager.Render(prompt.AgentVerification, data)
	if err != nil {
		return true, "Render failed, assume verified", result, nil
	}

	verifyResp, err := s.llmService.Generate(ctx, rendered, &domain.GenerationOptions{
		Temperature: 0.1,
		MaxTokens:   300,
	})
	if err != nil {
		return true, "", result, nil // Return original on error, assume verified
	}

	// Try to parse as JSON verification
	var verifyRespJSON struct {
		Verified   bool   `json:"verified"`
		Reason     string `json:"reason"`
		NeedsRetry bool   `json:"needs_retry"`
	}

	// Simple JSON extraction
	if err := extractJSON(verifyResp, &verifyRespJSON); err == nil {
		if verifyRespJSON.Verified {
			return true, "Verified", result, nil
		}
		return false, verifyRespJSON.Reason, nil, fmt.Errorf("verification failed: %s", verifyRespJSON.Reason)
	}

	// If parsing failed, assume verified to avoid infinite loops
	return true, "Parse OK, assume verified", result, nil
}

// extractJSON extracts JSON from LLM response (handles markdown code blocks)
func extractJSON(resp string, target interface{}) error {
	// Try direct parse first
	if err := json.Unmarshal([]byte(resp), target); err == nil {
		return nil
	}

	// Try to find JSON in markdown code blocks
	if strings.Contains(resp, "```json") {
		start := strings.Index(resp, "```json")
		if start >= 0 {
			jsonStart := start + 7
			end := strings.Index(resp[jsonStart:], "```")
			if end >= 0 {
				jsonStr := strings.TrimSpace(resp[jsonStart : jsonStart+end])
				return json.Unmarshal([]byte(jsonStr), target)
			}
		}
	}

	// Try to find plain code block
	if strings.Contains(resp, "```") {
		start := strings.Index(resp, "```")
		if start >= 0 {
			jsonStart := start + 3
			end := strings.Index(resp[jsonStart:], "```")
			if end >= 0 {
				jsonStr := strings.TrimSpace(resp[jsonStart : jsonStart+end])
				return json.Unmarshal([]byte(jsonStr), target)
			}
		}
	}

	return fmt.Errorf("no JSON found in response")
}

// executeToolCalls executes the tool calls decided by LLM and returns all results
// executeToolCalls executes the tool calls decided by LLM and returns all results
func (s *Service) executeToolCalls(ctx context.Context, currentAgent *Agent, toolCalls []domain.ToolCall) ([]ToolExecutionResult, error) {
	results := make([]ToolExecutionResult, len(toolCalls))

	// Create an errgroup to run tools in parallel
	g, groupCtx := errgroup.WithContext(ctx)

	for i, tc := range toolCalls {
		// Capture index and tool call for the goroutine
		idx, toolCall := i, tc

		g.Go(func() error {
			// Format tool name for display
			toolName := toolCall.Function.Name
			toolDesc := toolName
			if strings.HasPrefix(toolName, "mcp_") {
				toolDesc = strings.TrimPrefix(toolName, "mcp_")
			}

			s.emitProgress("tool_call", fmt.Sprintf("→ %s", toolDesc), 0, toolName)

			// --- DEBUG: LOG TOOL CALL ---
			if s.debug {
				fmt.Printf("\n🛠️  DEBUG TOOL CALL: %s\n", toolName)
				fmt.Printf("   Arguments: %v\n", toolCall.Function.Arguments)
			}

			s.logger.Info("Executing Tool",
				slog.String("tool", toolName),
				slog.Any("arguments", toolCall.Function.Arguments))

			var result interface{}
			var err error
			var toolType string

			// 0. Priority: Agent-local handler (multi-agent override scenarios).
			if handler, ok := currentAgent.GetHandler(toolName); ok {
				if s.debug {
					fmt.Println("   Type: Local Handler")
				}
				result, err = handler(groupCtx, toolCall.Function.Arguments)
				toolType = "local"
			} else if s.isMCPTool(toolName) {
				// 1. MCP tools — dynamic (managed externally via mcpService).
				if s.debug {
					fmt.Printf("   Type: MCP Tool\n")
				}
				result, err = s.mcpService.CallTool(groupCtx, toolName, toolCall.Function.Arguments)
				toolType = "mcp"
			} else if s.isSkill(groupCtx, toolName) && s.skillsService != nil {
				// 2. Skills — dynamic (managed via skillsService).
				if s.debug {
					fmt.Printf("   Type: Skill (%s)\n", toolName)
				}
				skillID := strings.TrimPrefix(toolName, "skill_")
				skillResult, skillErr := s.skillsService.Execute(groupCtx, &skills.ExecutionRequest{
					SkillID:     skillID,
					Variables:   toolCall.Function.Arguments,
					Interactive: false,
				})
				if skillErr == nil {
					result = skillResult.Output
				}
				err = skillErr
				toolType = "skill"
			} else if toolName == "execute_javascript" && s.ptcIntegration != nil {
				// 3. PTC: execute JavaScript in the goja sandbox.
				result, err = s.ptcIntegration.ExecuteJavascriptTool(groupCtx, toolCall.Function.Arguments)
				toolType = "ptc"
			} else if toolName == "delegate_to_subagent" {
				// 4. SubAgent delegation (needs reference to currentAgent).
				result, err = s.executeSubAgentDelegation(groupCtx, currentAgent, toolCall.Function.Arguments)
				toolType = "subagent"
			} else if s.toolRegistry.Has(toolName) {
				// 5. Unified ToolRegistry — custom tools, RAG, Memory.
				result, err = s.toolRegistry.Call(groupCtx, toolName, toolCall.Function.Arguments)
				toolType = s.toolRegistry.CategoryOf(toolName)
			} else {
				err = fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
			}

			if err != nil {
				s.logger.Error("Tool execution failed",
					slog.String("tool", toolName),
					slog.Any("error", err))

				if s.debug {
					fmt.Printf("   ❌ ERROR: %v\n", err)
					fmt.Println(strings.Repeat("-", 20))
				}

				return fmt.Errorf("Tool %s (%s) failed: %w", toolCall.Function.Name, toolType, err)
			}

			s.logger.Info("Tool Result",
				slog.String("tool", toolName),
				slog.Any("result", result))

			// --- DEBUG: LOG TOOL SUCCESS ---
			if s.debug {
				fmt.Printf("   ✅ RESULT: %v\n", result)
				fmt.Println(strings.Repeat("-", 20))
			}

			// Emit tool result progress
			s.emitProgress("tool_result", fmt.Sprintf("✓ %s Done", toolDesc), 0, toolName)

			results[idx] = ToolExecutionResult{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Function.Name,
				ToolType:   toolType,
				Result:     result,
			}
			return nil
		})
	}

	// Wait for all tools to finish
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// ToolExecutionResult represents the result of a single tool execution
type ToolExecutionResult struct {
	ToolCallID string      `json:"tool_call_id"`
	ToolName   string      `json:"tool_name"`
	ToolType   string      `json:"tool_type"`
	Result     interface{} `json:"result"`
}

// formatToolResults formats tool execution results for LLM consumption
func (s *Service) formatToolResults(results []ToolExecutionResult) string {
	var sb strings.Builder

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("Tool %d: %s (%s)\n", i+1, r.ToolName, r.ToolType))

		// Format result based on type
		switch v := r.Result.(type) {
		case string:
			if len(v) > 5000 {
				sb.WriteString(fmt.Sprintf("Result: %s...\n", v[:5000]))
			} else {
				sb.WriteString(fmt.Sprintf("Result: %s\n", v))
			}
		case []interface{}:
			// Handle array results (e.g., search results)
			for j, item := range v {
				sb.WriteString(fmt.Sprintf("  [%d] %v\n", j+1, item))
			}
		default:
			sb.WriteString(fmt.Sprintf("Result: %v\n", r.Result))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// executeWithDynamicToolSelection uses LLM's native function calling to decide which MCP tools to use
func (s *Service) executeWithDynamicToolSelection(ctx context.Context, goal string, intent *IntentRecognitionResult, availableTools []domain.ToolDefinition, memoryContext, ragResult string) (interface{}, error) {
	systemPrompt, err := s.promptManager.Render(prompt.AgentDynamicToolSelection, nil)
	if err != nil {
		systemPrompt = "You are a helpful assistant with access to tools. Use tools when appropriate to help the user."
	}

	// Build messages - let LLM decide which tools to call
	messages := []domain.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: goal},
	}

	// Add context if available
	if memoryContext != "" || ragResult != "" {
		contextMsg := "\n\nRelevant context:\n"
		if memoryContext != "" {
			contextMsg += memoryContext + "\n"
		}
		if ragResult != "" {
			contextMsg += ragResult + "\n"
		}
		messages[len(messages)-1].Content += contextMsg
	}

	// Use GenerateWithTools - let LLM natively decide which tools to call
	result, err := s.llmService.GenerateWithTools(ctx, messages, availableTools, &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   1000,
	})

	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// If LLM made tool calls, execute them
	if len(result.ToolCalls) > 0 {
		return s.executeLLMToolCalls(ctx, result.ToolCalls, goal, memoryContext, ragResult)
	}

	// No tool calls needed, return the text response
	return result.Content, nil
}

// executeLLMToolCalls executes tool calls decided by LLM
func (s *Service) executeLLMToolCalls(ctx context.Context, toolCalls []domain.ToolCall, goal, memoryContext, ragResult string) (interface{}, error) {
	var results []interface{}

	for _, tc := range toolCalls {
		log.Printf("[Agent] Calling tool: %s", tc.Function.Name)

		// Route through ToolRegistry first (covers custom, RAG, Memory tools).
		if s.toolRegistry.Has(tc.Function.Name) {
			result, err := s.toolRegistry.Call(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				results = append(results, fmt.Sprintf("Tool %s failed: %v", tc.Function.Name, err))
			} else {
				results = append(results, result)
			}
			continue
		}

		// MCP tools — handled by mcpService.
		result, err := s.mcpService.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("tool call failed: %w", err)
		}
		results = append(results, result)
	}

	// If results were obtained, format them
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// finalizeExecution finalizes the execution result
func (s *Service) finalizeExecution(ctx context.Context, session *Session, goal string, intent *IntentRecognitionResult, memoryMemories []*domain.MemoryWithScore, memoryLogic string, ragResult string, finalResult interface{}) (*ExecutionResult, error) {
	// Store to memory after completion
	if s.memoryService != nil {
		// Auto-store for explicit memory request patterns
		goalLower := strings.ToLower(goal)
		if strings.HasPrefix(goalLower, "remember:") ||
			strings.HasPrefix(goalLower, "save to memory") ||
			strings.HasPrefix(goalLower, "my favorite") ||
			strings.HasPrefix(goalLower, "i prefer") ||
			strings.Contains(goalLower, "preference is") {

			if !s.hasRunMemorySaved() {
				// Direct storage for explicit memory requests
				content := goal
				if strings.HasPrefix(goalLower, "remember:") {
					content = strings.TrimSpace(goal[len("remember:"):])
				} else if strings.HasPrefix(goalLower, "save to memory") {
					content = strings.TrimSpace(goal[len("save to memory"):])
				}

				if err := s.memoryService.Add(ctx, &domain.Memory{
					Type:       domain.MemoryTypePreference,
					Content:    content,
					Importance: 0.8,
					Metadata: map[string]interface{}{
						"source": "user_direct",
					},
				}); err != nil {
					s.logger.Warn("failed to store preference memory", slog.String("error", err.Error()))
				} else {
					log.Printf("[Agent] Stored to memory: %s", content)
				}
			}
		}

		// LLM-based extraction for complex memories
		if err := s.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:  session.GetID(),
			TaskGoal:   goal,
			TaskResult: formatResultForContent(finalResult),
			ExecutionLog: fmt.Sprintf("Intent: %s\nMemory: %d items\nRAG: %d chars",
				intent.IntentType, len(memoryMemories), len(ragResult)),
		}); err != nil {
			s.logger.Warn("failed to store memory", slog.String("error", err.Error()))
		}
	}

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	res := &ExecutionResult{
		PlanID:      uuid.New().String(),
		SessionID:   session.GetID(),
		Success:     true,
		StepsTotal:  1,
		StepsDone:   1,
		StepsFailed: 0,
		FinalResult: finalResult,
		Memories:    memoryMemories,
		MemoryLogic: memoryLogic,
		Duration:    "completed",
	}

	// Collect RAG sources
	s.ragSourcesMu.RLock()
	if len(s.ragSources) > 0 {
		res.Sources = append([]domain.Chunk{}, s.ragSources...)
	}
	s.ragSourcesMu.RUnlock()

	// Clear sources for next run
	s.ragSourcesMu.Lock()
	s.ragSources = nil
	s.ragSourcesMu.Unlock()

	// Emit PostExecution Hook on per-service registry
	s.hooks.Emit(HookEventPostExecution, HookData{
		SessionID: session.GetID(),
		AgentID:   session.AgentID,
		Goal:      goal,
		Result:    finalResult,
		Metadata: map[string]interface{}{
			"intent": intent.IntentType,
		},
	})

	return res, nil
}

// performRAGQuery performs a RAG query to get relevant documents
func (s *Service) performRAGQuery(ctx context.Context, query string) (string, error) {
	if s.ragProcessor == nil {
		return "", nil
	}

	// Use the RAG processor to query
	request := domain.QueryRequest{
		Query:        query,
		TopK:         5, // Get top 5 results
		Temperature:  0.3,
		ShowThinking: false,
		ShowSources:  true,
	}

	results, err := s.ragProcessor.Query(ctx, request)
	if err != nil {
		return "", err
	}

	// Format results as context
	if results.Answer == "" && len(results.Sources) == 0 {
		return "", nil
	}

	// Collect sources for final result (deduplicated)
	s.addRAGSources(results.Sources)

	var context strings.Builder
	context.WriteString("## Relevant Documents\n\n")

	// Add answer if available
	if results.Answer != "" {
		context.WriteString(fmt.Sprintf("**Answer:** %s\n\n", results.Answer))
	}

	// Add sources
	for i, source := range results.Sources {
		context.WriteString(fmt.Sprintf("### Document %d\n", i+1))
		if source.DocumentID != "" {
			context.WriteString(fmt.Sprintf("**Source:** %s\n", source.DocumentID))
		}
		if source.Score > 0 {
			context.WriteString(fmt.Sprintf("**Score:** %.2f\n", source.Score))
		}
		if source.Content != "" {
			context.WriteString(fmt.Sprintf("**Content:** %s\n", source.Content))
		}
		context.WriteString("\n---\n\n")
	}

	return context.String(), nil
}

// countDocuments counts the number of documents in RAG context
func countDocuments(ragContext string) int {
	if ragContext == "" {
		return 0
	}
	// Count "### Document" occurrences
	count := strings.Count(ragContext, "### Document")
	return count
}

// executeSubAgentDelegation handles the delegate_to_subagent tool call.
// It creates a SubAgent with the specified configuration, runs it, and returns the result.
func (s *Service) executeSubAgentDelegation(ctx context.Context, currentAgent *Agent, args map[string]interface{}) (interface{}, error) {
	goal, _ := args["goal"].(string)
	if goal == "" {
		return nil, fmt.Errorf("delegate_to_subagent: 'goal' argument is required")
	}

	maxTurns := 5
	if mt, ok := args["max_turns"].(float64); ok {
		maxTurns = int(mt)
	} else if mt, ok := args["max_turns"].(int); ok {
		maxTurns = mt
	}

	timeoutSeconds := 60
	if ts, ok := args["timeout_seconds"].(float64); ok {
		timeoutSeconds = int(ts)
	} else if ts, ok := args["timeout_seconds"].(int); ok {
		timeoutSeconds = ts
	}

	var allowlist, denylist []string
	if al, ok := args["tools_allowlist"].([]interface{}); ok {
		for _, v := range al {
			if s, ok := v.(string); ok {
				allowlist = append(allowlist, s)
			}
		}
	}
	if dl, ok := args["tools_denylist"].([]interface{}); ok {
		for _, v := range dl {
			if s, ok := v.(string); ok {
				denylist = append(denylist, s)
			}
		}
	}

	var contextData map[string]interface{}
	if ctxData, ok := args["context"].(map[string]interface{}); ok {
		contextData = ctxData
	}

	s.emitProgress("tool_call", fmt.Sprintf("→ Delegating to SubAgent: %s", truncateGoal(goal, 50)), 0, "delegate_to_subagent")

	subAgent := s.CreateSubAgent(currentAgent, goal,
		WithSubAgentMaxTurns(maxTurns),
		WithSubAgentTimeout(time.Duration(timeoutSeconds)*time.Second),
		WithSubAgentToolAllowlist(allowlist),
		WithSubAgentToolDenylist(denylist),
		WithSubAgentContext(contextData),
	)

	result, err := subAgent.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("sub-agent execution failed: %w", err)
	}

	s.emitProgress("tool_result", "✓ SubAgent completed", 0, "delegate_to_subagent")

	return map[string]interface{}{
		"subagent_id":   subAgent.ID(),
		"subagent_name": subAgent.Name(),
		"state":         string(subAgent.GetState()),
		"turns_used":    subAgent.GetCurrentTurn(),
		"duration_ms":   subAgent.GetDuration().Milliseconds(),
		"result":        result,
	}, nil
}
