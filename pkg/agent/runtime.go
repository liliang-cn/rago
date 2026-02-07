package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

		// 5. LLM Call (Streaming)
		var fullContent strings.Builder
		var toolCalls []domain.ToolCall

		err := r.svc.llmService.StreamWithTools(ctx, genMessages, tools, &domain.GenerationOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		}, func(chunk string, calls []domain.ToolCall) error {
			if chunk != "" {
				fullContent.WriteString(chunk)
				r.emit(EventTypePartial, chunk)
			}
			if len(calls) > 0 {
				toolCalls = calls
			}
			return nil
		})

		if err != nil {
			r.emit(EventTypeError, fmt.Sprintf("LLM error: %v", err))
			return
		}

		// 6. Handle Result
		if len(toolCalls) > 0 {
			// Add assistant's tool call message to history
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   fullContent.String(),
				ToolCalls: toolCalls,
			})

			// 7. Process Tool Calls (The "Side Effects")
			handoffOccurred := false
			
			for _, tc := range toolCalls {
				// Emit Tool Call Event
				r.emitToolCall(tc.Function.Name, tc.Function.Arguments)

				// Execute Tool logic
				res, err, isHandoff := r.executeToolOrHandoff(ctx, tc)
				
				if isHandoff {
					handoffOccurred = true
					// Update session state
					r.session.AgentID = r.currentAgent.ID()
					
					// Add tool result
					messages = append(messages, domain.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("Transferred to %s", r.currentAgent.Name()),
					})
					break
				}

				// Normal tool result
				content := ""
				if err != nil {
					content = fmt.Sprintf("Error: %v", err)
				} else {
					content = fmt.Sprintf("%v", res)
				}

				r.emitToolResult(tc.Function.Name, res, err)

				messages = append(messages, domain.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    content,
				})
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
	// 1. Agent-Local Tools
	if handler, ok := r.currentAgent.GetHandler(tc.Function.Name); ok {
		res, err := handler(ctx, tc.Function.Arguments)
		return res, err, false
	}

	// 2. MCP Tools
	if r.svc.isMCPTool(tc.Function.Name) {
		res, err := r.svc.mcpService.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
		return res, err, false
	}

	// 3. Skills
	if r.svc.isSkill(ctx, tc.Function.Name) && r.svc.skillsService != nil {
		res, err := r.svc.skillsService.Execute(ctx, &skills.ExecutionRequest{
			SkillID:   tc.Function.Name,
			Variables: tc.Function.Arguments,
		})
		if err != nil {
			return nil, err, false
		}
		return res.Output, nil, false
	}

	// 4. RAG
	if tc.Function.Name == "rag_query" && r.svc.ragProcessor != nil {
		q, _ := tc.Function.Arguments["query"].(string)
		resp, err := r.svc.ragProcessor.Query(ctx, domain.QueryRequest{Query: q})
		if err != nil {
			return nil, err, false
		}
		return resp.Answer, nil, false
	}

	// 5. Memory Tools
	if strings.HasPrefix(tc.Function.Name, "memory_") && r.svc.memoryService != nil {
		// Reuse existing logic via executeToolCalls or implement directly here
		// For brevity, simple implementation:
		if tc.Function.Name == "memory_save" {
			content, _ := tc.Function.Arguments["content"].(string)
			_ = r.svc.memoryService.Add(ctx, &domain.Memory{
				Type: domain.MemoryTypePreference, 
				Content: content,
				Importance: 0.8,
			})
			return "Saved", nil, false
		}
	}

	return nil, fmt.Errorf("unknown tool: %s", tc.Function.Name), false
}

func (r *Runtime) prepareContext(ctx context.Context, goal string) (string, string) {
	var ragCtx, memCtx string
	
	// RAG Retrieval
	if r.svc.ragProcessor != nil {
		if res, err := r.svc.performRAGQuery(ctx, goal); err == nil {
			ragCtx = res
		}
	}

	// Memory Retrieval
	if r.svc.memoryService != nil {
		memCtx, _, _ = r.svc.memoryService.RetrieveAndInject(ctx, goal, r.session.GetID())
	}

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
