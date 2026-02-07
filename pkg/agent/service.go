package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	ragprocessor "github.com/liliang-cn/rago/v2/pkg/rag/processor"
	ragstore "github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/liliang-cn/rago/v2/pkg/store"
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
	llmService       domain.Generator
	mcpService       MCPToolExecutor
	ragProcessor     domain.Processor
	memoryService    domain.MemoryService
	skillsService    *skills.Service
	routerService    *router.Service // Semantic Router for fast intent recognition
	planner          *Planner
	executor         *Executor
	store            *Store
	agent            *Agent
	cancelMu         sync.RWMutex
	cancelFunc       context.CancelFunc
	progressCb       ProgressCallback
	currentSessionID string // Auto-generated UUID for Chat() method
	sessionMu        sync.RWMutex
	memorySaveMu     sync.RWMutex
	memorySavedInRun bool

	// Public access to underlying services
	LLM    domain.Generator
	MCP    MCPToolExecutor
	RAG    domain.Processor
	Memory domain.MemoryService
	Router *router.Service
	Skills *skills.Service
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
		// Public fields
		LLM:    llmService,
		MCP:    mcpService,
		RAG:    ragProcessor,
		Memory: memoryService,
	}, nil
}

// SetRouter sets the semantic router for improved intent recognition
func (s *Service) SetRouter(routerService *router.Service) {
	s.routerService = routerService
	s.Router = routerService
	s.planner.SetRouter(routerService)
}

// SetSkillsService sets the skills service for agent integration
func (s *Service) SetSkillsService(skillsService *skills.Service) {
	s.skillsService = skillsService
	s.Skills = skillsService
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
	plan, err := s.planner.PlanWithFallback(ctx, goal, session)
	if err != nil {
		return nil, err
	}
	// Save plan to database
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}
	return plan, nil
}

// RevisePlan revises an existing plan based on user instructions
// The user can modify the plan through natural language chat
func (s *Service) RevisePlan(ctx context.Context, plan *Plan, instruction string) (*Plan, error) {
	// Build prompt for plan revision
	var prompt strings.Builder
	prompt.WriteString("You are revising an existing execution plan based on user feedback.\n\n")
	prompt.WriteString("=== Original Plan ===\n")
	prompt.WriteString(fmt.Sprintf("Goal: %s\n", plan.Goal))
	prompt.WriteString(fmt.Sprintf("Status: %s\n", plan.Status))
	prompt.WriteString(fmt.Sprintf("Current Steps (%d):\n", len(plan.Steps)))
	for i, step := range plan.Steps {
		prompt.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, step.Tool, step.Description))
	}
	prompt.WriteString("\n=== User Instruction ===\n")
	prompt.WriteString(instruction)
	prompt.WriteString("\n\n=== Task ===\n")
	prompt.WriteString("Generate a revised plan based on the user's instruction. ")
	prompt.WriteString("Return JSON with:\n")
	prompt.WriteString("- reasoning: explanation of changes\n")
	prompt.WriteString("- steps: array of steps, each with tool, description, arguments\n")
	prompt.WriteString("Keep the same step structure. Only include steps that need to be done.")

	// Call LLM to get revised plan
	response, err := s.llmService.Generate(ctx, prompt.String(), &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the response
	var revisedPlan struct {
		Reasoning string `json:"reasoning"`
		Steps     []struct {
			Tool        string                 `json:"tool"`
			Description string                 `json:"description"`
			Arguments   map[string]interface{} `json:"arguments"`
		} `json:"steps"`
	}

	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no valid JSON in LLM response")
	}
	jsonStr := response[jsonStart : jsonEnd+1]

	if err := json.Unmarshal([]byte(jsonStr), &revisedPlan); err != nil {
		return nil, fmt.Errorf("failed to parse revised plan: %w", err)
	}

	// Create new plan with revisions
	newPlan := &Plan{
		ID:        uuid.New().String(),
		SessionID: plan.SessionID,
		Goal:      plan.Goal,
		Status:    PlanStatusPending,
		Reasoning: revisedPlan.Reasoning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert steps
	for i, step := range revisedPlan.Steps {
		newPlan.Steps = append(newPlan.Steps, Step{
			ID:          fmt.Sprintf("step-%d", i+1),
			Tool:        step.Tool,
			Description: step.Description,
			Arguments:   step.Arguments,
			Status:      StepStatusPending,
		})
	}

	// Save revised plan
	if err := s.store.SavePlan(newPlan); err != nil {
		return nil, fmt.Errorf("failed to save revised plan: %w", err)
	}

	return newPlan, nil
}

// ExecutePlan executes the given plan
// This matches the CLI expectation: agentService.ExecutePlan(ctx, plan)
func (s *Service) ExecutePlan(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	result, err := s.executor.ExecutePlan(ctx, plan, nil)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Save the plan state
	if err := s.store.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	if !result.Success {
		return result, fmt.Errorf("plan execution completed with errors: %s", result.Error)
	}

	return result, nil
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
	s.resetRunMemorySaved()

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

	// Step 3: RAG Query - get relevant documents from knowledge base
	var ragContext string
	if s.ragProcessor != nil {
		s.emitProgress("thinking", "🔍 Searching knowledge base...", 0, "")
		var err error
		ragContext, err = s.performRAGQuery(runCtx, goal)
		if err != nil {
			// Log but don't fail - RAG is optional
			log.Printf("[Agent] RAG query failed: %v", err)
		} else if ragContext != "" {
			s.emitProgress("tool_result", fmt.Sprintf("✓ Found %d relevant documents", countDocuments(ragContext)), 0, "")
		}
	}

	// Step 4: Get memory context
	var memoryContext string
	var memoryMemories []*domain.MemoryWithScore
	if s.memoryService != nil {
		memoryContext, memoryMemories, _ = s.memoryService.RetrieveAndInject(runCtx, goal, session.GetID())
	}

	// Step 5: Let LLM decide and execute (with RAG context)
	finalResult, err := s.executeWithLLM(runCtx, goal, intent, availableTools, memoryContext, ragContext)
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
			// Retry with correction prompt (reuse the same RAG context)
			currentResult, err = s.executeWithLLM(runCtx, fmt.Sprintf("%s\n\nPrevious attempt was incomplete. Please ensure: %s", goal, reason), intent, availableTools, memoryContext, ragContext)
			if err != nil {
				break
			}
		}
	}

	// Add messages to session before saving
	session.AddMessage(domain.Message{
		Role:    "user",
		Content: goal,
	})
	if currentResult != nil {
		session.AddMessage(domain.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("%v", currentResult),
		})
	}

	// Create a simple plan to track this execution
	now := time.Now()
	plan := &Plan{
		ID:        uuid.New().String(),
		SessionID: session.GetID(),
		Goal:      goal,
		Status:    StatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
		Steps: []Step{
			{
				ID:          uuid.New().String(),
				Description: goal,
				Tool:        "llm",
				Status:      StepCompleted,
				Result:      currentResult,
			},
		},
	}
	if err := s.store.SavePlan(plan); err != nil {
		log.Printf("[Agent] Failed to save plan: %v", err)
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
				Type: "function",
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
			Type: "function",
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
		})	}

	// Add Memory tools
	if s.memoryService != nil {
		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_save",
				Description: "Save information to long-term memory for future reference",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The information to remember",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Type of memory (fact, preference, skill, pattern)",
							"enum":        []string{"fact", "preference", "skill", "pattern", "context"},
						},
					},
					"required": []string{"content"},
				},
			},
		})

		tools = append(tools, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_recall",
				Description: "Recall information from long-term memory",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The query to search memory for",
						},
					},
					"required": []string{"query"},
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
	sb.WriteString(`You are a helpful assistant with access to various tools.

IMPORTANT - Tool Response Guidelines:
- After using tools, provide a clear text response to summarize what was done
- For memory/save operations: respond with a brief confirmation like "I've saved that to memory" and STOP - do not call memory_save again
- For memory/recall operations: report what you found and respond to the user's question
- NEVER repeat the same tool call with the same arguments. If you already have the information, provide the final answer.
- If a tool succeeds, move to the next step or provide a final answer

`)
	sb.WriteString(systemCtx.FormatForPrompt())

	return sb.String()
}

// executeWithLLM lets LLM decide which tool to use and executes with multi-round support
func (s *Service) executeWithLLM(ctx context.Context, goal string, intent *IntentRecognitionResult, tools []domain.ToolDefinition, memoryContext string, ragContext string) (interface{}, error) {
	const maxRounds = 10 // Maximum rounds to prevent infinite loops

	// Track tool calls to detect duplicates
	prevToolCalls := make(map[string]int)

	// Build system message with context
	systemMsg := s.buildSystemPrompt()

	// Build initial messages
	messages := []domain.Message{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: goal},
	}

	// Add RAG context if available
	if ragContext != "" {
		messages[len(messages)-1].Content += "\n\n--- Relevant documents from knowledge base ---\n" + ragContext + "\n--- End of documents ---"
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

			// Check for duplicate tool calls (same tool, same arguments)
			for _, tc := range result.ToolCalls {
				// Create a simple key for the tool call
				callKey := fmt.Sprintf("%s:%v", tc.Function.Name, tc.Function.Arguments)
				prevToolCalls[callKey]++
				if prevToolCalls[callKey] > 1 {
					// Duplicate call detected - force stop
					log.Printf("[Agent] Duplicate tool call detected: %s, stopping", callKey)
					return "The task has been completed. The information has been saved to memory.", nil
				}
			}

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
				Role:      "assistant",
				Content:   result.Content,
				ReasoningContent: result.ReasoningContent,
				ToolCalls: result.ToolCalls,
			})

			// Add tool results using Role: tool
			for _, tr := range toolResults {
				resStr := fmt.Sprintf("%v", tr.Result)
				if s, ok := tr.Result.(string); ok {
					resStr = s
				}
				messages = append(messages, domain.Message{
					Role:       "tool",
					Content:    resStr,
					ToolCallID: tr.ToolCallID,
				})
			}

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
		} else if tc.Function.Name == "rag_ingest" && s.ragProcessor != nil {
			content, _ := tc.Function.Arguments["content"].(string)
			filePath, _ := tc.Function.Arguments["file_path"].(string)
			_, err = s.ragProcessor.Ingest(ctx, domain.IngestRequest{
				Content:  content,
				FilePath: filePath,
			})
			if err == nil {
				result = "Successfully ingested document"
			}
			toolType = "rag"
		} else if tc.Function.Name == "memory_save" && s.memoryService != nil {
			s.markRunMemorySaved()
			content, _ := tc.Function.Arguments["content"].(string)
			memType := "preference"
			if t, ok := tc.Function.Arguments["type"].(string); ok {
				memType = t
			}
			err = s.memoryService.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
				Metadata: map[string]interface{}{
					"source": "tool_call",
				},
			})
			if err == nil {
				result = fmt.Sprintf("Saved to memory: %s", content)
			}
			toolType = "memory"
		} else if tc.Function.Name == "memory_recall" && s.memoryService != nil {
			query, _ := tc.Function.Arguments["query"].(string)
			memories, memErr := s.memoryService.Search(ctx, query, 5)
			if memErr == nil {
				if len(memories) == 0 {
					// Fallback: list all recent memories
					allMems, _, listErr := s.memoryService.List(ctx, 10, 0)
					if listErr == nil && len(allMems) > 0 {
						var memResults []string
						for _, m := range allMems {
							memResults = append(memResults, fmt.Sprintf("- [%s] %s", m.Type, m.Content))
						}
						result = fmt.Sprintf("Recent memories:\n%s", strings.Join(memResults, "\n"))
					} else {
						result = "No relevant memories found"
					}
				} else {
					var memResults []string
					for _, m := range memories {
						memResults = append(memResults, fmt.Sprintf("- [%s: %.2f] %s", m.Type, m.Score, m.Content))
					}
					result = fmt.Sprintf("Found %d memories:\n%s", len(memories), strings.Join(memResults, "\n"))
				}
			}
			err = memErr
			toolType = "memory"
		} else {
			err = fmt.Errorf("unknown tool: %s", tc.Function.Name)
		}

		if err != nil {
			return nil, fmt.Errorf("Tool %s (%s) failed: %w", tc.Function.Name, toolType, err)
		}

		// Emit tool result progress
		s.emitProgress("tool_result", fmt.Sprintf("✓ %s Done", toolDesc), 0, toolName)

		results = append(results, ToolExecutionResult{
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			ToolType:   toolType,
			Result:     result,
		})
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
	s.resetRunMemorySaved()

	// Load or create session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
	}

	// Add user message to session
	userMsg := domain.Message{
		Role:    "user",
		Content: goal,
	}
	session.AddMessage(userMsg)

	// Step 1: Retrieve memory context before planning
	if s.memoryService != nil {
		ragolog.Debug("[RunWithSession] Retrieving memory context for goal: %q, sessionID: %q", goal, sessionID)
		memoryContext, memoryMemories, _ := s.memoryService.RetrieveAndInject(ctx, goal, sessionID)
		ragolog.Debug("[RunWithSession] Memory context retrieved - context length: %d, memories: %d", len(memoryContext), len(memoryMemories))
		if memoryContext != "" {
			// Inject memory context into the session messages
			session.AddMessage(domain.Message{
				Role:    "system",
				Content: memoryContext,
			})
		}
	} else {
		ragolog.Debug("[RunWithSession] Memory service is nil, skipping memory retrieval")
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
	result, err := s.executor.ExecutePlan(ctx, plan, session)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Add assistant response to session
	if result.FinalResult != nil {
		assistantContent := fmt.Sprintf("%v", result.FinalResult)
		assistantMsg := domain.Message{
			Role:    "assistant",
			Content: assistantContent,
		}
		session.AddMessage(assistantMsg)
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

// Chat sends a message with auto-generated session UUID.
// This is the simplest API for conversational AI with memory.
//
// Example:
//
//	svc, _ := agent.New(&agent.AgentConfig{Name: "assistant"})
//	result, _ := svc.Chat(ctx, "My name is Alice")
//	result, _ = svc.Chat(ctx, "What's my name?") // Will remember "Alice"
func (s *Service) Chat(ctx context.Context, message string) (*ExecutionResult, error) {
	s.sessionMu.Lock()
	if s.currentSessionID == "" {
		s.currentSessionID = uuid.New().String()
	}
	sessionID := s.currentSessionID
	s.sessionMu.Unlock()

	return s.RunWithSession(ctx, message, sessionID)
}

// CurrentSessionID returns the current session UUID used by Chat()
func (s *Service) CurrentSessionID() string {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.currentSessionID
}

// SetSessionID sets a specific session ID for Chat() to use
func (s *Service) SetSessionID(sessionID string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.currentSessionID = sessionID
}

// ResetSession clears the current session and starts a new one with a new UUID
func (s *Service) ResetSession() {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.currentSessionID = uuid.New().String()
}

// CompactSession summarizes a session into key points
func (s *Service) CompactSession(ctx context.Context, sessionID string) (string, error) {
	// Load session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
	}

	messages := session.GetMessages()
	if len(messages) == 0 {
		return "", nil
	}

	// Check if llmService supports Compact
	llmSvc, ok := s.llmService.(interface {
		Compact(ctx context.Context, messages []domain.Message) (string, error)
	})
	if !ok {
		return "", fmt.Errorf("underlying LLM service does not support Compact")
	}

	// Generate summary
	summary, err := llmSvc.Compact(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to compact session: %w", err)
	}

	// Update session
	session.SetSummary(summary)
	if err := s.store.SaveSession(session); err != nil {
		return "", fmt.Errorf("failed to save session summary: %w", err)
	}

	return summary, nil
}

// Execute executes a plan by ID and returns the result
func (s *Service) Execute(ctx context.Context, planID string) (*ExecutionResult, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	return s.ExecutePlan(ctx, plan)
}

// SaveToFile saves content to a file
func (s *Service) SaveToFile(content, filePath string) error {
	// Create directory if needed
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("[Agent] ✅ Saved to %s\n", filePath)
	return nil
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	return s.store.Close()
}

// ========================================
// Simplified API
// ========================================

// AgentConfig holds configuration for New()
type AgentConfig struct {
	Name         string
	SystemPrompt string
	DBPath       string
	MemoryDBPath string
	EnableMCP    bool
	EnableMemory bool
	EnableRAG    bool
	EnableRouter bool
	EnableSkills bool
	IntentPaths  []string
	RouterThreshold float64
	ProgressCb   ProgressCallback
}

// New creates an agent service with simplified configuration.
// It automatically initializes LLM, Embedding, MCP, Memory, and Router services from rago.toml.
//
// Example:
//
//	svc, err := agent.New(&agent.AgentConfig{
//	    Name: "my-agent",
//	    EnableMCP: true,
//	    EnableMemory: true,
//	})
//	result, _ := svc.Run(ctx, "Hello!")
func New(cfg *AgentConfig) (*Service, error) {
	if cfg == nil || cfg.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	// Load config
	ragoCfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize global pool
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(context.Background(), ragoCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize pool: %w", err)
	}

	// Get LLM service
	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	// Get Embedding service
	embedSvc, err := globalPool.GetEmbeddingService(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service
	var mcpSvc *mcp.Service
	var mcpAdapter MCPToolExecutor
	if cfg.EnableMCP {
		mcpSvc, err = mcp.NewService(&ragoCfg.MCP, llmSvc)
		if err == nil {
			mcpSvc.StartServers(context.Background(), nil)
			mcpAdapter = &mcpToolAdapter{service: mcpSvc}
		}
	}

	// Memory service
	var memSvc domain.MemoryService
	memDBPath := cfg.MemoryDBPath
	if memDBPath == "" {
		memDBPath = filepath.Join(ragoCfg.DataDir(), "memory.db")
	}
	if cfg.EnableMemory {
		memStore, err := store.NewMemoryStore(memDBPath)
		if err == nil {
			_ = memStore.InitSchema(context.Background())
			memSvc = memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
		}
	}

	// Router service
	var routerSvc *router.Service
	if cfg.EnableRouter {
		routerCfg := router.DefaultConfig()
		if cfg.RouterThreshold > 0 {
			routerCfg.Threshold = cfg.RouterThreshold
		}
		routerSvc, err = router.NewService(embedSvc, routerCfg)
		if err == nil {
			_ = routerSvc.RegisterDefaultIntents()
		}
	}

	// RAG processor
	var ragProcessor domain.Processor
	if cfg.EnableRAG {
		// Create RAG stores
		ragStorePath := filepath.Join(ragoCfg.DataDir(), cfg.Name+"_rag.db")
		vectorStore, _ := ragstore.NewVectorStore(ragstore.StoreConfig{
			Type: "sqlite",
			Parameters: map[string]interface{}{
				"path": ragStorePath,
			},
		})
		if vectorStore != nil {
			docStore := ragstore.NewDocumentStoreFor(vectorStore)
			// Create RAG processor with proper stores
			ragProcessor = ragprocessor.New(
				embedSvc,
				llmSvc,
				nil,         // chunker - will use default
				vectorStore, // vector store
				docStore,    // document store
				ragoCfg,
				nil,    // llmService for metadata extraction (optional)
				memSvc, // memory service
			)
		}
	}

	// Skills service
	var skillsSvc *skills.Service
	if cfg.EnableSkills {
		skillsCfg := skills.DefaultConfig()
		skillsCfg.Paths = []string{ragoCfg.SkillsDir()}
		skillsCfg.DBPath = filepath.Join(ragoCfg.DataDir(), "skills.db")
		skillsSvc, err = skills.NewService(skillsCfg)
		if err == nil {
			_ = skillsSvc.LoadAll(context.Background())
		}
	}

	// Agent DB path
	agentDBPath := cfg.DBPath
	if agentDBPath == "" {
		agentDBPath = filepath.Join(ragoCfg.DataDir(), cfg.Name+".db")
	}

	// Create agent service
	svc, err := NewService(llmSvc, mcpAdapter, ragProcessor, agentDBPath, memSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Set skills service
	if skillsSvc != nil {
		svc.SetSkillsService(skillsSvc)
	}

	// Set router
	if routerSvc != nil {
		svc.SetRouter(routerSvc)

		// Load custom intents if provided or from default path
		intentPaths := cfg.IntentPaths
		if len(intentPaths) == 0 {
			intentPaths = []string{ragoCfg.IntentsDir()}
		}
		_ = svc.Router.LoadIntentsFromPaths(intentPaths)
	}

	// Set progress callback
	if cfg.ProgressCb != nil {
		svc.SetProgressCallback(cfg.ProgressCb)
	}

	return svc, nil
}

// mcpToolAdapter wraps mcp.Service to implement MCPToolExecutor
type mcpToolAdapter struct {
	service *mcp.Service
}

func (a *mcpToolAdapter) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	result, err := a.service.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("MCP tool error: %s", result.Error)
	}
	return result.Data, nil
}

func (a *mcpToolAdapter) ListTools() []domain.ToolDefinition {
	tools := a.service.GetAvailableTools(context.Background())
	result := make([]domain.ToolDefinition, 0, len(tools))

	for _, t := range tools {
		params := t.InputSchema
		if params == nil {
			params = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"args": map[string]interface{}{
						"description": "arguments",
						"type":        "object",
					},
				},
			}
		}
		result = append(result, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return result
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
		log.Printf("[Agent] Calling tool: %s", tc.Function.Name)

		// Handle memory tools
		if tc.Function.Name == "memory_save" {
			s.markRunMemorySaved()
			content, _ := tc.Function.Arguments["content"].(string)
			memType := "preference"
			if t, ok := tc.Function.Arguments["type"].(string); ok {
				memType = t
			}
			err := s.memoryService.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
				Metadata: map[string]interface{}{
					"source": "tool_call",
				},
			})
			if err != nil {
				results = append(results, fmt.Sprintf("Failed to save memory: %v", err))
			} else {
				results = append(results, fmt.Sprintf("Saved to memory: %s", content))
			}
			continue
		}

		if tc.Function.Name == "memory_recall" {
			query, _ := tc.Function.Arguments["query"].(string)
			memories, err := s.memoryService.Search(ctx, query, 5)
			if err != nil {
				results = append(results, fmt.Sprintf("Memory search failed: %v", err))
			} else if len(memories) == 0 {
				results = append(results, "No relevant memories found")
			} else {
				var memResults []string
				for _, m := range memories {
					memResults = append(memResults, fmt.Sprintf("- [%s: %.2f] %s", m.Type, m.Score, m.Content))
				}
				results = append(results, fmt.Sprintf("Found %d memories:\n%s", len(memories), strings.Join(memResults, "\n")))
			}
			continue
		}

		// Handle MCP tools
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

				_ = s.memoryService.Add(ctx, &domain.Memory{
					Type:       domain.MemoryTypePreference,
					Content:    content,
					Importance: 0.8,
					Metadata: map[string]interface{}{
						"source": "user_direct",
					},
				})
				log.Printf("[Agent] Stored to memory: %s", content)
			}
		}

		// LLM-based extraction for complex memories
		_ = s.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:  session.GetID(),
			TaskGoal:   goal,
			TaskResult: formatResultForContent(finalResult),
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
