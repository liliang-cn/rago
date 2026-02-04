package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/skills"
)

// ProgressEvent 进度事件
type ProgressEvent struct {
	Type    string // "thinking", "tool_call", "tool_result", "done"
	Round   int
	Message string
	Tool    string
}

// ProgressCallback 进度回调函数
type ProgressCallback func(ProgressEvent)

// Service is the main agent service that handles planning and execution
// This matches the interface expected by the CLI in cmd/rago-cli/agent/agent.go
type Service struct {
	llmService    domain.Generator
	mcpService    MCPToolExecutor
	ragProcessor  domain.Processor
	memoryService domain.MemoryService
	skillsService *skills.Service
	routerService *router.Service // Semantic Router for fast intent recognition
	planner       *Planner
	executor      *Executor
	store         *Store
	agent         *Agent
	cancelMu      sync.RWMutex
	cancelFunc    context.CancelFunc
	progressCb    ProgressCallback
}

// NewService creates a new agent service
// This matches the signature expected by the CLI:
// agent.NewService(llmService, mcpService, processor, agentDBPath, memoryService)
func NewService(
	llmService domain.Generator,
	mcpService MCPToolExecutor,
	ragProcessor domain.Processor,
	agentDBPath string,
	memoryService domain.MemoryService,
) (*Service, error) {
	// Initialize store
	store, err := NewStore(agentDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent store: %w", err)
	}

	// Collect available tools
	tools := collectAvailableTools(mcpService, ragProcessor, nil)

	// Create default agent
	agent := NewAgentWithConfig(
		"RAGO Agent",
		`You are RAGO, a helpful AI assistant with access to RAG (Retrieval Augmented Generation),
MCP tools, Skills, and various processing capabilities.

When given a goal:
1. Break it down into clear steps
2. Choose appropriate tools for each step
3. Execute steps in logical order
4. Provide clear results

Available tools include:
- RAG queries (rag_query, rag_ingest)
- MCP tools (external integrations)
- Skills (reusable skill packages)
- General LLM reasoning`,
		tools,
	)

	// Create planner (without router initially)
	planner := NewPlanner(llmService, tools)

	// Create executor
	executor := NewExecutor(llmService, nil, mcpService, ragProcessor, memoryService)

	return &Service{
		llmService:    llmService,
		mcpService:    mcpService,
		ragProcessor:  ragProcessor,
		memoryService: memoryService,
		planner:       planner,
		executor:      executor,
		store:         store,
		agent:         agent,
	}, nil
}

// SetRouter sets the semantic router for improved intent recognition
func (s *Service) SetRouter(routerService *router.Service) {
	s.routerService = routerService
	// Update planner with router
	if routerService != nil {
		s.planner.SetRouter(routerService)
	}
}

// SetSkillsService sets the skills service for agent integration
func (s *Service) SetSkillsService(skillsService *skills.Service) {
	s.skillsService = skillsService
	// Re-create agent with updated tools
	if skillsService != nil {
		tools := collectAvailableTools(s.mcpService, s.ragProcessor, skillsService)
		s.agent = NewAgentWithConfig(
			s.agent.Name(),
			s.agent.Instructions(),
			tools,
		)
		s.planner = NewPlanner(s.llmService, tools)
		// Restore router if it was set
		if s.routerService != nil {
			s.planner.SetRouter(s.routerService)
		}
		// Set skills service on executor for tool execution
		s.executor.SetSkillsService(skillsService)
	}
}

// SetProgressCallback sets the progress callback for execution events
func (s *Service) SetProgressCallback(cb ProgressCallback) {
	s.progressCb = cb
}

// emitProgress emits a progress event if callback is set
func (s *Service) emitProgress(eventType, message string, round int, tool string) {
	if s.progressCb != nil {
		s.progressCb(ProgressEvent{
			Type:    eventType,
			Round:   round,
			Message: message,
			Tool:    tool,
		})
	}
}

// Plan generates an execution plan for the given goal
// This matches the CLI expectation: agentService.Plan(ctx, goal)
func (s *Service) Plan(ctx context.Context, goal string) (*Plan, error) {
	session := NewSession(s.agent.ID())
	return s.planner.PlanWithFallback(ctx, goal, session)
}

// ExecutePlan executes the given plan
// This matches the CLI expectation: agentService.ExecutePlan(ctx, plan)
func (s *Service) ExecutePlan(ctx context.Context, plan *Plan) error {
	result, err := s.executor.ExecutePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Save the plan state
	if err := s.store.SavePlan(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("plan execution completed with errors: %s", result.Error)
	}

	return nil
}

// Run executes a goal using a fixed workflow:
// 1. Intent Recognition
// 2. Check Memory
// 3. RAG Query
// Run executes a goal - simple and dynamic:
// 1. Collect all available tools (MCP + Skills + RAG)
// 2. Intent Recognition
// 3. Let LLM match intent to tools and execute
func (s *Service) Run(ctx context.Context, goal string) (*ExecutionResult, error) {
	// Create cancellable context for this run
	runCtx, cancel := context.WithCancel(ctx)

	// Store cancel function for external cancellation
	s.cancelMu.Lock()
	s.cancelFunc = cancel
	s.cancelMu.Unlock()

	defer func() {
		s.cancelMu.Lock()
		s.cancelFunc = nil
		s.cancelMu.Unlock()
	}()

	session := NewSession(s.agent.ID())

	// Step 1: Collect ALL available tools
	availableTools := s.collectAllAvailableTools(runCtx)

	// Step 2: Intent Recognition (for context, not routing)
	intent, _ := s.recognizeIntent(runCtx, goal, session)

	// Step 3: Get memory context
	var memoryContext string
	var memoryMemories []*domain.MemoryWithScore
	if s.memoryService != nil {
		memoryContext, memoryMemories, _ = s.memoryService.RetrieveAndInject(runCtx, goal, session.GetID())
	}

	// Step 4: Let LLM decide and execute
	finalResult, err := s.executeWithLLM(runCtx, goal, intent, availableTools, memoryContext)
	if err != nil {
		return nil, err
	}

	// Step 5: Verify result with LLM (optional verification step)
	const maxVerifyRetries = 2
	currentResult := finalResult
	for verifyAttempt := 0; verifyAttempt <= maxVerifyRetries; verifyAttempt++ {
		verified, reason, correctedResult, err := s.verifyResult(runCtx, goal, currentResult)
		if err != nil {
			s.emitProgress("tool_result", fmt.Sprintf("⚠ Verification failed: %v", err), 0, "")
			if verifyAttempt == maxVerifyRetries {
				return s.finalizeExecution(runCtx, session, goal, intent, memoryMemories, "", currentResult)
			}
			continue
		}

		if verified {
			if correctedResult != nil {
				currentResult = correctedResult
			}
			break
		}

		s.emitProgress("tool_result", fmt.Sprintf("⚠ Verification: %s - Retrying...", reason), 0, "")
		if verifyAttempt < maxVerifyRetries {
			// Retry with correction prompt
			currentResult, err = s.executeWithLLM(runCtx, fmt.Sprintf("%s\n\nPrevious attempt was incomplete. Please ensure: %s", goal, reason), intent, availableTools, memoryContext)
			if err != nil {
				break
			}
		}
	}

	return s.finalizeExecution(runCtx, session, goal, intent, memoryMemories, "", currentResult)
}

// Cancel forcefully stops the current agent execution
func (s *Service) Cancel() bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if s.cancelFunc != nil {
		log.Printf("[Agent] Cancelling current execution...")
		s.cancelFunc()
		return true
	}
	return false
}

// collectAllAvailableTools collects tools from MCP, Skills, and RAG
func (s *Service) collectAllAvailableTools(ctx context.Context) []domain.ToolDefinition {
	var tools []domain.ToolDefinition

	// MCP tools
	if s.mcpService != nil {
		tools = append(tools, s.mcpService.ListTools()...)
	}

	// Skills tools
	if s.skillsService != nil {
		skillsList, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
		for _, sk := range skillsList {
			tools = append(tools, domain.ToolDefinition{
				Type:     "function",
				Function: domain.ToolFunction{
					Name:        sk.ID,
					Description: sk.Description,
					Parameters:  map[string]interface{}{},
				},
			})
		}
	}

	// RAG tools
	if s.ragProcessor != nil {
		tools = append(tools, domain.ToolDefinition{
			Type:     "function",
			Function: domain.ToolFunction{
				Name:        "rag_query",
				Description: "Search knowledge base for information",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
					},
				},
			},
		})
	}

	return tools
}

// buildSystemPrompt 构建包含系统上下文的system prompt
func (s *Service) buildSystemPrompt() string {
	systemCtx := s.buildSystemContext()

	var sb strings.Builder
	sb.WriteString("You are a helpful assistant with access to various tools. Use tools when helpful. After using tools, provide a clear summary of what was done.\n\n")
	sb.WriteString(systemCtx.FormatForPrompt())

	return sb.String()
}

// executeWithLLM lets LLM decide which tool to use and executes with multi-round support
func (s *Service) executeWithLLM(ctx context.Context, goal string, intent *IntentRecognitionResult, tools []domain.ToolDefinition, memoryContext string) (interface{}, error) {
	const maxRounds = 10 // Maximum rounds to prevent infinite loops

	// Build system message with context
	systemMsg := s.buildSystemPrompt()

	// Build initial messages
	messages := []domain.Message{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: goal},
	}

	// Add memory context if available
	if memoryContext != "" {
		messages[len(messages)-1].Content += "\n\nRelevant context from memory:\n" + memoryContext
	}

	// Multi-round conversation loop
	for round := 0; round < maxRounds; round++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution cancelled by user")
		default:
		}

		s.emitProgress("thinking", "思考中...", round+1, "")

		// Let LLM decide
		result, err := s.llmService.GenerateWithTools(ctx, messages, tools, &domain.GenerationOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		})

		if err != nil {
			return nil, fmt.Errorf("LLM execution failed: %w", err)
		}

		// If LLM made tool calls, execute them and continue the conversation
		if len(result.ToolCalls) > 0 {

			// Execute tools and collect results
			s.emitProgress("tool_call", fmt.Sprintf("Calling %d tool(s)", len(result.ToolCalls)), round+1, "")
			toolResults, err := s.executeToolCalls(ctx, result.ToolCalls)
			if err != nil {
				// Add error as assistant message and continue
				messages = append(messages, domain.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Tool execution failed: %v", err),
				})
				continue
			}

			// Add assistant's response (tool calls) to conversation history
			messages = append(messages, domain.Message{
				Role:    "assistant",
				Content: result.Content,
			})

			// Add tool results as a new user message for the LLM to process
			toolResultMsg := fmt.Sprintf("Tool execution results:\n%s", s.formatToolResults(toolResults))
			messages = append(messages, domain.Message{
				Role:    "user",
				Content: toolResultMsg,
			})

			continue
		}

		// No more tool calls - LLM is done
		return result.Content, nil
	}

	return nil, fmt.Errorf("exceeded maximum rounds (%d) - conversation may be stuck", maxRounds)
}

// verifyResult verifies the result with LLM
// Returns: (verified bool, reason string, correctedResult interface{}, err error)
func (s *Service) verifyResult(ctx context.Context, goal string, result interface{}) (bool, string, interface{}, error) {
	resultStr := formatResultForContent(result)

	// Build verification prompt
	verifyPrompt := fmt.Sprintf(`Original Goal: %s

Agent Result: %s

Please verify:
1. Does the result actually complete the original goal?
2. Is the result accurate and complete?
3. For file operations: was the actual content (not placeholder) written to the file?
4. For data queries: was real data retrieved and saved?

Respond with JSON:
{
  "verified": true/false,
  "reason": "brief explanation if not verified",
  "needs_retry": true/false
}

If the goal was fully accomplished with real data (not placeholders), return {"verified": true, "needs_retry": false}.`, goal, resultStr)

	verifyResp, err := s.llmService.Generate(ctx, verifyPrompt, &domain.GenerationOptions{
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
func (s *Service) executeToolCalls(ctx context.Context, toolCalls []domain.ToolCall) ([]ToolExecutionResult, error) {
	results := make([]ToolExecutionResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		// Format tool name for display
		toolName := tc.Function.Name
		toolDesc := toolName
		if strings.HasPrefix(toolName, "mcp_") {
			toolDesc = strings.TrimPrefix(toolName, "mcp_")
		}

		s.emitProgress("tool_call", fmt.Sprintf("→ %s", toolDesc), 0, toolName)

		var result interface{}
		var err error
		var toolType string

		// Route to appropriate executor
		if s.isMCPTool(tc.Function.Name) {
			result, err = s.mcpService.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
			toolType = "mcp"
		} else if s.isSkill(ctx, tc.Function.Name) && s.skillsService != nil {
			skillResult, skillErr := s.skillsService.Execute(ctx, &skills.ExecutionRequest{
				SkillID:     tc.Function.Name,
				Variables:   tc.Function.Arguments,
				Interactive: false,
			})
			if skillErr == nil {
				result = skillResult.Output
				err = skillErr
			}
			toolType = "skill"
		} else if tc.Function.Name == "rag_query" && s.ragProcessor != nil {
			query, _ := tc.Function.Arguments["query"].(string)
			resp, ragErr := s.ragProcessor.Query(ctx, domain.QueryRequest{Query: query})
			if ragErr == nil {
				result = resp.Answer
			}
			err = ragErr
			toolType = "rag"
		} else {
			err = fmt.Errorf("unknown tool: %s", tc.Function.Name)
		}

		if err != nil {
			return nil, fmt.Errorf("Tool %s (%s) failed: %w", tc.Function.Name, toolType, err)
		}

		// Emit tool result progress
		s.emitProgress("tool_result", fmt.Sprintf("✓ %s Done", toolDesc), 0, toolName)

		results = append(results, ToolExecutionResult{
			ToolName: tc.Function.Name,
			ToolType: toolType,
			Result:   result,
		})
	}

	return results, nil
}

// ToolExecutionResult represents the result of a single tool execution
type ToolExecutionResult struct {
	ToolName string      `json:"tool_name"`
	ToolType string      `json:"tool_type"`
	Result   interface{} `json:"result"`
}

// formatToolResults formats tool execution results for LLM consumption
func (s *Service) formatToolResults(results []ToolExecutionResult) string {
	var sb strings.Builder

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("Tool %d: %s (%s)\n", i+1, r.ToolName, r.ToolType))

		// Format result based on type
		switch v := r.Result.(type) {
		case string:
			if len(v) > 500 {
				sb.WriteString(fmt.Sprintf("Result: %s...\n", v[:500]))
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
	skills, _ := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
	for _, sk := range skills {
		if sk.ID == name {
			return true
		}
	}
	return false
}

// recognizeIntent performs intent recognition using planner
func (s *Service) recognizeIntent(ctx context.Context, goal string, session *Session) (*IntentRecognitionResult, error) {
	return s.planner.RecognizeIntent(ctx, goal, session)
}

// shouldUseRAG determines if RAG should be used based on intent
func (s *Service) shouldUseRAG(intent *IntentRecognitionResult) bool {
	// Use RAG for query, analysis, and general_qa intents
	return intent.IntentType == "rag_query" ||
		intent.IntentType == "analysis" ||
		intent.IntentType == "general_qa" ||
		intent.IntentType == "question"
}

// shouldUseSkills determines if skills should be used based on intent
func (s *Service) shouldUseSkills(intent *IntentRecognitionResult) bool {
	// Use skills for web_search, file operations, etc.
	return intent.IntentType == "web_search" ||
		intent.IntentType == "file_create" ||
		intent.IntentType == "file_read"
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

// executeSkills executes skills based on intent
func (s *Service) executeSkills(ctx context.Context, intent *IntentRecognitionResult, prompt string) (interface{}, error) {
	// Find relevant skill based on intent
	if s.skillsService == nil {
		return nil, fmt.Errorf("skills service not available")
	}

	// List available skills
	skillList, err := s.skillsService.ListSkills(ctx, skills.SkillFilter{})
	if err != nil {
		return nil, err
	}

	// Map intents to skill keyword patterns
	intentSkillPatterns := map[string][]string{
		"web_search":  {"search", "web", "query", "rag"},
		"rag_query":   {"query", "rag", "search"},
		"file_create": {"create", "write", "file"},
		"file_read":   {"read", "file", "open"},
	}

	// Find patterns for this intent
	patterns, hasPatterns := intentSkillPatterns[intent.IntentType]
	if !hasPatterns {
		// No specific patterns, try any skill
		for _, sk := range skillList {
			req := &skills.ExecutionRequest{
				SkillID:     sk.ID,
				Variables:   map[string]interface{}{"query": intent.Topic},
				Interactive: false,
			}
			result, err := s.skillsService.Execute(ctx, req)
			if err == nil && result.Success {
				return result.Output, nil
			}
		}
	}

	// Try to find a matching skill
	for _, sk := range skillList {
		skillIDLower := strings.ToLower(sk.ID)
		for _, pattern := range patterns {
			if strings.Contains(skillIDLower, pattern) {
				req := &skills.ExecutionRequest{
					SkillID:     sk.ID,
					Variables:   map[string]interface{}{"query": intent.Topic},
					Interactive: false,
				}
				result, err := s.skillsService.Execute(ctx, req)
				if err == nil && result.Success {
					return result.Output, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no suitable skill found for intent: %s", intent.IntentType)
}

// RunWithSession executes a goal with an existing session ID
func (s *Service) RunWithSession(ctx context.Context, goal, sessionID string) (*ExecutionResult, error) {
	// Load or create session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
	}

	// Generate plan
	plan, err := s.planner.PlanWithFallback(ctx, goal, session)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Save plan
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Execute plan
	result, err := s.executor.ExecutePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Save updated plan
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return result, nil
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(sessionID string) (*Session, error) {
	return s.store.GetSession(sessionID)
}

// GetPlan retrieves a plan by ID
func (s *Service) GetPlan(planID string) (*Plan, error) {
	return s.store.GetPlan(planID)
}

// ListSessions returns all sessions
func (s *Service) ListSessions(limit int) ([]*Session, error) {
	return s.store.ListSessions(limit)
}

// ListPlans returns plans for a session
func (s *Service) ListPlans(sessionID string, limit int) ([]*Plan, error) {
	return s.store.ListPlans(sessionID, limit)
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	return s.store.Close()
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

// executeWithDynamicToolSelection uses LLM's native function calling to decide which MCP tools to use
func (s *Service) executeWithDynamicToolSelection(ctx context.Context, goal string, intent *IntentRecognitionResult, availableTools []domain.ToolDefinition, memoryContext, ragResult string) (interface{}, error) {
	// Build messages - let LLM decide which tools to call
	messages := []domain.Message{
		{Role: "system", Content: "You are a helpful assistant with access to tools. Use tools when appropriate to help the user."},
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
		log.Printf("[Agent] Calling MCP tool: %s", tc.Function.Name)
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
func (s *Service) finalizeExecution(ctx context.Context, session *Session, goal string, intent *IntentRecognitionResult, memoryMemories []*domain.MemoryWithScore, ragResult string, finalResult interface{}) (*ExecutionResult, error) {
	// Store to memory after completion
	if s.memoryService != nil {
		_ = s.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:    session.GetID(),
			TaskGoal:     goal,
			TaskResult:  formatResultForContent(finalResult),
			ExecutionLog: fmt.Sprintf("Intent: %s\nMemory: %d items\nRAG: %d chars",
				intent.IntentType, len(memoryMemories), len(ragResult)),
		})
	}

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &ExecutionResult{
		PlanID:      uuid.New().String(),
		SessionID:   session.GetID(),
		Success:     true,
		StepsTotal:  1,
		StepsDone:   1,
		StepsFailed: 0,
		FinalResult: finalResult,
		Duration:    "completed",
	}, nil
}
