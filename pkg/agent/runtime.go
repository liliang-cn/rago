package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/skills"
)

// Runtime orchestrates the event loop for agent execution
type Runtime struct {
	svc          *Service
	eventChan    chan *Event
	currentAgent *Agent
	session      *Session
}

// NewRuntime creates a new runtime instance
func NewRuntime(svc *Service, session *Session) *Runtime {
	// Determine initial agent
	currentAgent := svc.agent
	if session.AgentID != "" && svc.registry != nil {
		if a, ok := svc.registry.GetAgent(session.AgentID); ok {
			currentAgent = a
		}
	}

	return &Runtime{
		svc:          svc,
		eventChan:    make(chan *Event, 100), // Buffer events
		currentAgent: currentAgent,
		session:      session,
	}
}

// RunStream starts the event loop and returns a read-only channel of events
func (r *Runtime) RunStream(ctx context.Context, goal string) <-chan *Event {
	go r.loop(ctx, goal)
	return r.eventChan
}

// loop is the core event loop
func (r *Runtime) loop(ctx context.Context, goal string) {
	defer close(r.eventChan)

	r.emit(EventTypeStart, fmt.Sprintf("Starting task: %s", goal))

	// 1. Prepare context (Memory & RAG)
	memoryContext, ragContext := r.prepareContext(ctx, goal)

	// 2. Build initial messages
	messages := []domain.Message{
		{Role: "user", Content: goal},
	}
	if ragContext != "" {
		messages[len(messages)-1].Content += "\n\n--- Knowledge Base ---\n" + ragContext
	}
	if memoryContext != "" {
		messages[len(messages)-1].Content += "\n\n--- Memory ---\n" + memoryContext
	}

	const maxRounds = 20
	for round := 0; round < maxRounds; round++ {
		// Check cancellation
		if ctx.Err() != nil {
			r.emit(EventTypeError, "Execution cancelled")
			return
		}

		r.emit(EventTypeThinking, "Thinking...")

		// 3. Collect tools for CURRENT agent
		tools := r.svc.collectAllAvailableTools(ctx, r.currentAgent)

		// 4. Build System Prompt for CURRENT agent
		systemMsg := r.svc.buildSystemPrompt(r.currentAgent)
		genMessages := append([]domain.Message{{Role: "system", Content: systemMsg}}, messages...)

		// --- DEBUG: LOG FULL PROMPT ---
		if r.svc.config != nil && r.svc.config.Debug {
			fmt.Println("\n" + strings.Repeat("=", 40))
			fmt.Printf("DEBUG [Runtime Round %d] LLM FULL PROMPT\n", round+1)
			fmt.Println(strings.Repeat("-", 40))
			for _, m := range genMessages {
				fmt.Printf("[%s]:\n%s\n", strings.ToUpper(m.Role), m.Content)
			}
			fmt.Println(strings.Repeat("=", 40) + "\n")
		}

		// 5. LLM Call (Streaming)
		var fullContent strings.Builder
		var toolCalls []domain.ToolCall
		toolCallDetected := false

		err := r.svc.llmService.StreamWithTools(ctx, genMessages, tools, &domain.GenerationOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		}, func(delta *domain.GenerationResult) error {
			// 1. Handle Reasoning (The "Thinking" Stream)
			if delta.ReasoningContent != "" {
				r.emit(EventTypeThinking, delta.ReasoningContent)
			}

			// 2. Handle Content (The "Output" Stream)
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				r.emit(EventTypePartial, delta.Content)
			}

			// 3. Handle Tool Calls detection
			if len(delta.ToolCalls) > 0 {
				if !toolCallDetected {
					r.emit(EventTypeThinking, "Planning tool usage...")
					toolCallDetected = true
				}
				toolCalls = delta.ToolCalls
			}
			return nil
		})

		if err != nil {
			r.emit(EventTypeError, fmt.Sprintf("LLM error: %v", err))
			return
		}

		// --- DEBUG: LOG LLM RESPONSE ---
		if r.svc.config != nil && r.svc.config.Debug {
			fmt.Println("\n" + strings.Repeat("=", 40))
			fmt.Printf("DEBUG [Runtime Round %d] LLM RESPONSE\n", round+1)
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("CONTENT: %s\n", fullContent.String())
			if len(toolCalls) > 0 {
				fmt.Println("TOOL CALLS:")
				for _, tc := range toolCalls {
					fmt.Printf("  - %s(%v)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
			fmt.Println(strings.Repeat("=", 40) + "\n")
		}

		// 6. Handle Result
		if len(toolCalls) > 0 {
			// Add assistant's tool call message to history
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   fullContent.String(),
				ToolCalls: toolCalls,
			})

			// 7. Process Tool Calls (Parallel Execution)
			handoffOccurred := false
			
			// Use errgroup for parallel tool execution
			g, groupCtx := errgroup.WithContext(ctx)
			toolResults := make([]struct {
				Content    string
				IsHandoff  bool
				ToolCallID string
				ToolName   string
				Result     interface{}
				Error      error
			}, len(toolCalls))

			for i, tc := range toolCalls {
				idx, toolCall := i, tc
				
				// Handle Handoff immediately (sequential) as it changes state
				if strings.HasPrefix(toolCall.Function.Name, "transfer_to_") {
					res, err, isHandoff := r.executeToolOrHandoff(ctx, toolCall)
					toolResults[idx].Content = fmt.Sprintf("%v", res)
					if err != nil { toolResults[idx].Content = fmt.Sprintf("Error: %v", err) }
					toolResults[idx].IsHandoff = isHandoff
					toolResults[idx].ToolCallID = toolCall.ID
					toolResults[idx].ToolName = toolCall.Function.Name
					toolResults[idx].Result = res
					toolResults[idx].Error = err
					if isHandoff { handoffOccurred = true }
					continue
				}

				// Parallel execute independent tools
				g.Go(func() error {
					r.emitToolCall(toolCall.Function.Name, toolCall.Function.Arguments)
					res, err, isHandoff := r.executeToolOrHandoff(groupCtx, toolCall)
					
					content := ""
					if err != nil {
						content = fmt.Sprintf("Error: %v", err)
					} else {
						content = fmt.Sprintf("%v", res)
					}

					toolResults[idx].Content = content
					toolResults[idx].IsHandoff = isHandoff
					toolResults[idx].ToolCallID = toolCall.ID
					toolResults[idx].ToolName = toolCall.Function.Name
					toolResults[idx].Result = res
					toolResults[idx].Error = err
					
					r.emitToolResult(toolCall.Function.Name, res, err)
					return nil
				})
			}

			_ = g.Wait()

			// Collect all results into messages
			for _, tr := range toolResults {
				if tr.ToolCallID == "" { continue } // Skip if not handled (shouldn't happen)
				
				if tr.IsHandoff {
					r.session.AgentID = r.currentAgent.ID()
					messages = append(messages, domain.Message{
						Role:       "tool",
						ToolCallID: tr.ToolCallID,
						Content:    fmt.Sprintf("Transferred to %s", r.currentAgent.Name()),
					})
				} else {
					messages = append(messages, domain.Message{
						Role:       "tool",
						ToolCallID: tr.ToolCallID,
						Content:    tr.Content,
					})
				}
			}

			if handoffOccurred {
				continue
			}

		} else {
			// Final Answer
			r.emit(EventTypeComplete, fullContent.String())
			
			// Auto-save to memory ASYNC to prevent lag at the end
			go r.saveToMemory(context.Background(), goal, fullContent.String())
			return
		}
	}
}

// executeToolOrHandoff executes a tool call and handles agent switching
func (r *Runtime) executeToolOrHandoff(ctx context.Context, tc domain.ToolCall) (interface{}, error, bool) {
	toolName := tc.Function.Name

	// --- DEBUG: LOG TOOL START ---
	if r.svc.config != nil && r.svc.config.Debug {
		fmt.Printf("\n🛠️  DEBUG RUNTIME TOOL CALL: %s\n", toolName)
		fmt.Printf("   Arguments: %v\n", tc.Function.Arguments)
	}

	// === PRE-TOOL HOOK ===
	hookData := HookData{
		ToolName:  tc.Function.Name,
		ToolArgs:  tc.Function.Arguments,
		SessionID: r.session.GetID(),
		AgentID:   r.currentAgent.ID(),
	}

	if r.svc.hooks != nil {
		modifiedData, err := r.svc.hooks.EmitWithResult(ctx, HookEventPreToolUse, hookData)
		if err != nil {
			// Hook blocked execution
			return nil, err, false
		}
		// Use modified args if hook changed them
		if modifiedData.ToolArgs != nil {
			tc.Function.Arguments = modifiedData.ToolArgs
		}
	}

	// Check for Handoff
	if strings.HasPrefix(tc.Function.Name, "transfer_to_") {
		for _, h := range r.currentAgent.Handoffs() {
			if h.ToolName() == tc.Function.Name {
				targetAgent := h.TargetAgent()
				reason := tc.Function.Arguments["reason"]

				r.emit(EventTypeHandoff, fmt.Sprintf("Transferring to %s: %v", targetAgent.Name(), reason))

				// SWITCH AGENT
				r.currentAgent = targetAgent
				return nil, nil, true
			}
		}
	}

	// Normal Tool Execution
	var result interface{}
	var execErr error

	// 1. Agent-Local Tools
	if handler, ok := r.currentAgent.GetHandler(tc.Function.Name); ok {
		result, execErr = handler(ctx, tc.Function.Arguments)
	} else if r.svc.isMCPTool(tc.Function.Name) {
		// 2. MCP Tools
		result, execErr = r.svc.mcpService.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
	} else if r.svc.isSkill(ctx, tc.Function.Name) && r.svc.skillsService != nil {
		// 3. Skills
		res, err := r.svc.skillsService.Execute(ctx, &skills.ExecutionRequest{
			SkillID:   tc.Function.Name,
			Variables: tc.Function.Arguments,
		})
		if err != nil {
			execErr = err
		} else {
			result = res.Output
		}
	} else if toolName == "rag_query" && r.svc.ragProcessor != nil {
		// 4. RAG Query
		q, _ := tc.Function.Arguments["query"].(string)
		resp, err := r.svc.ragProcessor.Query(ctx, domain.QueryRequest{Query: q})
		if err != nil {
			execErr = err
		} else {
			result = resp.Answer
			// Include sources if available
			if len(resp.Sources) > 0 {
				var sourcesBuf strings.Builder
				sourcesBuf.WriteString("\n\n**Sources:**\n")
				for i, src := range resp.Sources {
					sourcesBuf.WriteString(fmt.Sprintf("%d. %s\n", i+1, src.Content))
					if len(sourcesBuf.String()) > 2000 {
						break // Limit sources length
					}
				}
				result = resp.Answer + sourcesBuf.String()
			}
		}
	} else if toolName == "rag_ingest" && r.svc.ragProcessor != nil {
		// 4.1 RAG Ingest
		content, _ := tc.Function.Arguments["content"].(string)
		filePath, _ := tc.Function.Arguments["file_path"].(string)
		_, execErr = r.svc.ragProcessor.Ingest(ctx, domain.IngestRequest{
			Content:  content,
			FilePath: filePath,
		})
		if execErr == nil {
			result = "Successfully ingested document"
		}
	} else if toolName == "rag_delete" && r.svc.ragProcessor != nil {
		// 4.2 RAG Delete
		docID, _ := tc.Function.Arguments["document_id"].(string)
		execErr = r.svc.ragProcessor.DeleteDocument(ctx, docID)
		if execErr == nil {
			result = fmt.Sprintf("Successfully deleted document %s", docID)
		}
	} else if strings.HasPrefix(toolName, "memory_") && r.svc.memoryService != nil {
		// 5. Memory Tools
		if toolName == "memory_save" {
			content, _ := tc.Function.Arguments["content"].(string)
			memType, _ := tc.Function.Arguments["type"].(string)
			if memType == "" { memType = string(domain.MemoryTypeFact) }
			execErr = r.svc.memoryService.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
			})
			if execErr == nil { result = "Saved" }
		} else if toolName == "memory_update" {
			id, _ := tc.Function.Arguments["id"].(string)
			content, _ := tc.Function.Arguments["content"].(string)
			execErr = r.svc.memoryService.Update(ctx, id, content)
			if execErr == nil { result = "Updated" }
		} else if toolName == "memory_delete" {
			id, _ := tc.Function.Arguments["id"].(string)
			execErr = r.svc.memoryService.Delete(ctx, id)
			if execErr == nil { result = "Deleted" }
		} else if toolName == "memory_recall" {
			query, _ := tc.Function.Arguments["query"].(string)
			mContext, _, err := r.svc.memoryService.RetrieveAndInject(ctx, query, "")
			result = mContext
			execErr = err
		}
	} else {
		execErr = fmt.Errorf("unknown tool: %s", toolName)
	}

	// === POST-TOOL HOOK ===
	hookData.ToolResult = result
	hookData.ToolError = execErr
	if r.svc.hooks != nil {
		r.svc.hooks.Emit(HookEventPostToolUse, hookData)
	}

	// --- DEBUG: LOG TOOL RESULT ---
	if r.svc.config != nil && r.svc.config.Debug {
		if execErr != nil {
			fmt.Printf("   ❌ ERROR: %v\n", execErr)
		} else {
			fmt.Printf("   ✅ RESULT: %v\n", result)
		}
		fmt.Println(strings.Repeat("-", 20))
	}

	return result, execErr, false
}

func (r *Runtime) prepareContext(ctx context.Context, goal string) (string, string) {
	var ragCtx, memCtx string
	
	g, groupCtx := errgroup.WithContext(ctx)

	// RAG Retrieval
	if r.svc.ragProcessor != nil {
		g.Go(func() error {
			if res, err := r.svc.performRAGQuery(groupCtx, goal); err == nil {
				ragCtx = res
			}
			return nil
		})
	}

	// Memory Retrieval
	if r.svc.memoryService != nil {
		g.Go(func() error {
			memCtx, _, _ = r.svc.memoryService.RetrieveAndInject(groupCtx, goal, r.session.GetID())
			return nil
		})
	}

	_ = g.Wait()
	return memCtx, ragCtx
}

func (r *Runtime) saveToMemory(ctx context.Context, goal, result string) {
	if r.svc.memoryService != nil {
		_ = r.svc.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:  r.session.GetID(),
			TaskGoal:   goal,
			TaskResult: result,
		})
	}
}

// Helpers to emit events
func (r *Runtime) emit(t EventType, content string) {
	r.eventChan <- &Event{
		ID:        uuid.New().String(),
		Type:      t,
		AgentName: r.currentAgent.Name(),
		AgentID:   r.currentAgent.ID(),
		Content:   content,
		Timestamp: time.Now(),
	}
}

func (r *Runtime) emitToolCall(name string, args map[string]interface{}) {
	r.eventChan <- &Event{
		ID:        uuid.New().String(),
		Type:      EventTypeToolCall,
		AgentName: r.currentAgent.Name(),
		ToolName:  name,
		ToolArgs:  args,
		Timestamp: time.Now(),
	}
}

func (r *Runtime) emitToolResult(name string, res interface{}, err error) {
	evt := &Event{
		ID:         uuid.New().String(),
		Type:       EventTypeToolResult,
		AgentName:  r.currentAgent.Name(),
		ToolName:   name,
		ToolResult: res,
		Timestamp:  time.Now(),
	}
	if err != nil {
		// You might want a specific error event or just include error in content
		evt.Content = err.Error()
	}
	r.eventChan <- evt
}
