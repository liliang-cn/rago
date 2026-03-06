package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/skills"
	"golang.org/x/sync/errgroup"
)

// Runtime orchestrates the event loop for agent execution
type Runtime struct {
	svc          *Service
	eventChan    chan *Event
	currentAgent *Agent
	session      *Session
	sources      []domain.Chunk // Collect RAG sources during execution
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
	defer func() {
		fmt.Printf("[AGENT] Runtime loop finished\n")
		close(r.eventChan)
	}()

	fmt.Printf("[AGENT] Runtime loop started for goal: %s\n", goal)

	r.emit(EventTypeStart, fmt.Sprintf("Starting task: %s", goal))

	// 1. Prepare context (Memory & RAG)
	fmt.Printf("[AGENT] Preparing context...\n")
	memoryContext, ragContext := r.prepareContext(ctx, goal)
	fmt.Printf("[AGENT] Context prepared, memory=%d chars, rag=%d chars\n", len(memoryContext), len(ragContext))

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
		systemMsg := r.svc.buildSystemPrompt(ctx, r.currentAgent)
		genMessages := append([]domain.Message{{Role: "system", Content: systemMsg}}, messages...)

		// --- DEBUG: LOG FULL PROMPT ---
		if r.svc.debug {
			var promptBuilder strings.Builder
			for _, m := range genMessages {
				fmt.Fprintf(&promptBuilder, "[%s]:\n%s\n\n", strings.ToUpper(m.Role), m.Content)
			}
			r.emitDebug(round+1, "prompt", promptBuilder.String())
		}

		// 5. LLM Call (Streaming)
		var fullContent strings.Builder
		var toolCalls []domain.ToolCall
		toolCallDetected := false

		fmt.Printf("[AGENT] Round %d: Calling LLM with %d tools...\n", round, len(tools))
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
		if r.svc.debug {
			var respBuilder strings.Builder
			fmt.Fprintf(&respBuilder, "CONTENT: %s\n", fullContent.String())
			if len(toolCalls) > 0 {
				fmt.Fprintf(&respBuilder, "TOOL CALLS:\n")
				for _, tc := range toolCalls {
					fmt.Fprintf(&respBuilder, "  - %s(%v)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
			r.emitDebug(round+1, "response", respBuilder.String())
		}

		// 6. Handle Result
		if len(toolCalls) > 0 {
			// PTC fix: some models (e.g. gpt-5.2) emit valid JS as text content
			// and then issue a broken execute_javascript tool call with garbage code.
			// When PTC is active and the text stream contains valid JS, override the
			// tool call's code with the sanitised text-stream code.
			if r.svc.isPTCEnabled() {
				content := fullContent.String()
				isCode := r.svc.ptcIntegration.IsCodeResponse(content)
				if r.svc.debug {
					fmt.Printf("DEBUG [PTC Override] IsCodeResponse=%v contentLen=%d\n", isCode, len(content))
				}
				if isCode {
					extracted := r.svc.ptcIntegration.ExtractCode(content)
					if r.svc.debug {
						if len(extracted) > 100 {
							fmt.Printf("DEBUG [PTC Override] ExtractCode len=%d first100=%q\n", len(extracted), extracted[:100])
						} else {
							fmt.Printf("DEBUG [PTC Override] ExtractCode len=%d content=%q\n", len(extracted), extracted)
						}
					}
					extracted = sanitiseJSCode(extracted)
					if r.svc.debug {
						if len(extracted) > 100 {
							fmt.Printf("DEBUG [PTC Override] After sanitise len=%d first100=%q\n", len(extracted), extracted[:100])
						} else {
							fmt.Printf("DEBUG [PTC Override] After sanitise len=%d content=%q\n", len(extracted), extracted)
						}
					}
					if extracted != "" {
						for i, tc := range toolCalls {
							if tc.Function.Name == "execute_javascript" {
								if toolCalls[i].Function.Arguments == nil {
									toolCalls[i].Function.Arguments = make(map[string]interface{})
								}
								toolCalls[i].Function.Arguments["code"] = extracted
								if r.svc.debug {
									fmt.Printf("DEBUG [PTC Override] Replaced code for tool call %d\n", i)
								}
							}
						}
					}
				}
			}

			// Ensure every tool call has an ID before building the assistant message —
			// some OpenAI-compatible providers omit the id field, which causes
			// tool results to be silently dropped when matched back.
			for i := range toolCalls {
				if toolCalls[i].ID == "" {
					toolCalls[i].ID = fmt.Sprintf("call_%s_%d", toolCalls[i].Function.Name, i)
				}
			}

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
					toolResults[idx].Content = toolResultToString(res)
					if err != nil {
						toolResults[idx].Content = fmt.Sprintf("Error: %v", err)
					}
					toolResults[idx].IsHandoff = isHandoff
					toolResults[idx].ToolCallID = toolCall.ID
					toolResults[idx].ToolName = toolCall.Function.Name
					toolResults[idx].Result = res
					toolResults[idx].Error = err
					if isHandoff {
						handoffOccurred = true
					}
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
						content = toolResultToString(res)
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
				if tr.ToolCallID == "" {
					continue
				} // Skip if not handled (shouldn't happen)

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
			// PTC fallback: when PTC is active and the LLM wrote JS code as a
			// text/markdown response instead of using the execute_javascript
			// function-call, intercept it, execute it, and inject the result
			// back so the LLM can produce a grounded final answer.
			if r.svc.isPTCEnabled() {
				content := fullContent.String()
				if r.svc.ptcIntegration.IsCodeResponse(content) {
					code := r.svc.ptcIntegration.ExtractCode(content)
					if code != "" {
						r.emitToolCall("execute_javascript", map[string]interface{}{"code": code})
						execResult, execErr := r.svc.ptcIntegration.ExecuteJavascriptTool(ctx, map[string]interface{}{"code": code})
						r.emitToolResult("execute_javascript", execResult, execErr)

						// Append assistant's code message + execution result so
						// the LLM can synthesise a final answer in the next round.
						messages = append(messages, domain.Message{
							Role:    "assistant",
							Content: content,
						})
						resultMsg := execResult
						if execErr != nil {
							resultMsg = fmt.Sprintf("execute_javascript error: %v", execErr)
						}
						messages = append(messages, domain.Message{
							Role:    "user",
							Content: fmt.Sprintf("execute_javascript result:\n%s\n\nBased on these results, please provide the final answer.", resultMsg),
						})
						continue // next round → LLM synthesises answer
					}
				}
			}

			// Final Answer - merge sources from runtime and service
			allSources := r.sources
			r.svc.ragSourcesMu.RLock()
			if len(r.svc.ragSources) > 0 {
				allSources = append(allSources, r.svc.ragSources...)
			}
			r.svc.ragSourcesMu.RUnlock()

			r.eventChan <- &Event{
				ID:        uuid.New().String(),
				Type:      EventTypeComplete,
				AgentName: r.currentAgent.Name(),
				AgentID:   r.currentAgent.ID(),
				Content:   fullContent.String(),
				Sources:   allSources, // Include all collected RAG sources
				Timestamp: time.Now(),
			}

			// Clear service sources for next run
			r.svc.ragSourcesMu.Lock()
			r.svc.ragSources = nil
			r.svc.ragSourcesMu.Unlock()

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
	if r.svc.debug {
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
		skillID := strings.TrimPrefix(tc.Function.Name, "skill_")
		res, err := r.svc.skillsService.Execute(ctx, &skills.ExecutionRequest{
			SkillID:   skillID,
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
			// Collect sources for final result
			if len(resp.Sources) > 0 {
				r.sources = append(r.sources, resp.Sources...)
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
			if memType == "" {
				memType = string(domain.MemoryTypeFact)
			}
			execErr = r.svc.memoryService.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
			})
			if execErr == nil {
				result = "Saved"
			}
		} else if toolName == "memory_update" {
			id, _ := tc.Function.Arguments["id"].(string)
			content, _ := tc.Function.Arguments["content"].(string)
			execErr = r.svc.memoryService.Update(ctx, id, content)
			if execErr == nil {
				result = "Updated"
			}
		} else if toolName == "memory_delete" {
			id, _ := tc.Function.Arguments["id"].(string)
			execErr = r.svc.memoryService.Delete(ctx, id)
			if execErr == nil {
				result = "Deleted"
			}
		} else if toolName == "memory_recall" {
			query, _ := tc.Function.Arguments["query"].(string)
			mContext, _, err := r.svc.memoryService.RetrieveAndInject(ctx, query, "")
			result = mContext
			execErr = err
		}
	} else if toolName == "execute_javascript" && r.svc.ptcIntegration != nil {
		// 6. PTC: Execute JavaScript in sandbox
		result, execErr = r.svc.ptcIntegration.ExecuteJavascriptTool(ctx, tc.Function.Arguments)
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
	if r.svc.debug {
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

	// RAG Retrieval — skip when PTC is enabled: the LLM will call rag_query
	// explicitly via execute_javascript / callTool, so pre-injecting the
	// answer would short-circuit the tool and make it unreachable.
	if r.svc.ragProcessor != nil && !r.svc.isPTCEnabled() {
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
			var err error
			memCtx, _, err = r.svc.memoryService.RetrieveAndInject(groupCtx, goal, r.session.GetID())
			if err != nil {
				r.svc.logger.Warn("memory retrieval failed", slog.String("error", err.Error()))
			}
			return nil
		})
	}

	_ = g.Wait()
	return memCtx, ragCtx
}

func (r *Runtime) saveToMemory(ctx context.Context, goal, result string) {
	if r.svc.memoryService != nil {
		if err := r.svc.memoryService.StoreIfWorthwhile(ctx, &domain.MemoryStoreRequest{
			SessionID:  r.session.GetID(),
			TaskGoal:   goal,
			TaskResult: result,
		}); err != nil {
			r.svc.logger.Warn("failed to store memory after run", slog.String("error", err.Error()))
		}
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

func (r *Runtime) emitDebug(round int, debugType string, content string) {
	r.eventChan <- &Event{
		ID:        uuid.New().String(),
		Type:      EventTypeDebug,
		AgentName: r.currentAgent.Name(),
		Round:     round,
		DebugType: debugType,
		Content:   content,
		Timestamp: time.Now(),
	}
}
