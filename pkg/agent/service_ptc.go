package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
	"github.com/liliang-cn/rago/v2/pkg/skills"
)

// ChatWithPTC sends a message with PTC (Parallel Tool Calling) support.
// This method uses JavaScript code execution for tool orchestration.
func (s *Service) ChatWithPTC(ctx context.Context, message string) (*PTCChatResult, error) {
	// Check if PTC is available
	if s.ptcIntegration == nil || !s.ptcIntegration.config.Enabled {
		// Fall back to normal chat
		result, err := s.Chat(ctx, message)
		if err != nil {
			return nil, err
		}
		return &PTCChatResult{
			ExecutionResult: result,
			PTCUsed:         false,
		}, nil
	}

	// Get current session
	s.sessionMu.Lock()
	if s.currentSessionID == "" {
		s.currentSessionID = uuid.New().String()
	}
	sessionID := s.currentSessionID
	s.sessionMu.Unlock()

	// Load or create session
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		session = NewSessionWithID(sessionID, s.agent.ID())
	}

	// Build PTC-aware user message
	ptcPrompt := message + `

IMPORTANT: Respond with JavaScript code in <code> tags.
Your code will be executed in a secure sandbox.
Use console.log() for output and return the final result with a top-level return statement.
DO NOT wrap code in function main(){...}main().
Example format:
` + "<code>\nconst data = callTool('some_tool', { arg: 'value' });\nconsole.log(\"Processing:\", data);\nreturn { result: data };\n</code>"

	// Add user message to session
	userMsg := domain.Message{
		Role:    "user",
		Content: ptcPrompt,
	}
	session.AddMessage(userMsg)

	// Build messages for LLM
	messages := session.GetMessages()

	// Build PTC tools
	availableCallTools := s.ptcIntegration.GetAvailableCallTools(ctx)
	ptcTools := s.ptcIntegration.GetPTCTools(availableCallTools)

	// Call LLM with PTC tools
	opts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	var fullContent strings.Builder
	var toolCalls []domain.ToolCall

	err = s.llmService.StreamWithTools(ctx, messages, ptcTools, opts, func(delta *domain.GenerationResult) error {
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			s.emitProgress("partial", delta.Content, 0, "")
		}
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				// Simplified merging logic for PTC: find by index or ID
				found := false
				for j, existing := range toolCalls {
					if (existing.ID != "" && existing.ID == tc.ID) || (existing.ID == "" && existing.Function.Name == tc.Function.Name) {
						if existing.Function.Arguments == nil {
							existing.Function.Arguments = make(map[string]interface{})
						}
						for k, v := range tc.Function.Arguments {
							existing.Function.Arguments[k] = v
						}
						toolCalls[j] = existing
						found = true
						break
					}
				}
				if !found {
					toolCalls = append(toolCalls, tc)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("LLM streaming failed: %w", err)
	}

	content := fullContent.String()

	if os.Getenv("DEBUG") != "" {
		fmt.Printf("\nDEBUG [ChatWithPTC] raw content: %q\n", content)
		if len(toolCalls) > 0 {
			fmt.Printf("DEBUG [ChatWithPTC] tool calls: %v\n", toolCalls)
		}
	}

	// 1. Try to get code from tool calls first (preferred for structured responses)
	// Some LLMs return thinking text in content + code in execute_javascript tool call
	if len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			if tc.Function.Name == "execute_javascript" {
				if code, ok := tc.Function.Arguments["code"].(string); ok {
					// Re-wrap in <code> to ensure ProcessLLMResponse handles it
					content = "<code>" + code + "</code>"
					break
				}
			}
		}
	}

	// Process LLM response through PTC
	ptcResult, err := s.ptcIntegration.ProcessLLMResponse(ctx, content, nil)
	if err != nil {
		return nil, fmt.Errorf("PTC processing failed: %w", err)
	}

	// Add assistant response to session (without the PTC prompt instructions)
	assistantMsg := domain.Message{
		Role:    "assistant",
		Content: content,
	}
	session.AddMessage(assistantMsg)

	// Save session
	if err := s.store.SaveSession(session); err != nil {
		ragolog.Warn("failed to save session: %v", err)
	}

	return &PTCChatResult{
		PTCResult:   ptcResult,
		PTCUsed:     ptcResult.Code != "" || ptcResult.Type == PTCResultTypeExecuted || ptcResult.Type == PTCResultTypeCode,
		LLMResponse: content,
		SessionID:   sessionID,
	}, nil
}

// PTCChatResult contains the result of a PTC-aware chat
type PTCChatResult struct {
	ExecutionResult *ExecutionResult `json:"execution_result,omitempty"`
	PTCResult       *PTCResult       `json:"ptc_result,omitempty"`
	PTCUsed         bool             `json:"ptc_used"`
	LLMResponse     string           `json:"llm_response"`
	SessionID       string           `json:"session_id"`
}

// registerModuleTools populates the service's ToolRegistry with handlers for
// RAG and Memory module tools. This must be called after NewService() and before
// PTC setup (so SyncToPTCRouter picks them up).
func registerModuleTools(svc *Service, ragProc domain.Processor, memSvc domain.MemoryService) {
	if ragProc != nil {
		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_query",
				Description: "Search the knowledge base for information",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string", "description": "Search query"},
						"top_k": map[string]interface{}{"type": "integer", "description": "Number of results (default 5)"},
					},
					"required": []string{"query"},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			query, _ := args["query"].(string)
			if query == "" {
				return nil, fmt.Errorf("rag_query: 'query' argument is required")
			}
			topK := 5
			if tk, ok := args["top_k"].(float64); ok {
				topK = int(tk)
			} else if tk, ok := args["top_k"].(int); ok {
				topK = tk
			}
			resp, err := ragProc.Query(ctx, domain.QueryRequest{Query: query, TopK: topK})
			if err != nil {
				return nil, err
			}
			svc.addRAGSources(resp.Sources)
			return map[string]interface{}{"answer": resp.Answer, "sources": len(resp.Sources)}, nil
		}, CategoryRAG)

		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "rag_ingest",
				Description: "Ingest a document into the RAG knowledge base",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content":   map[string]interface{}{"type": "string", "description": "Document content"},
						"file_path": map[string]interface{}{"type": "string", "description": "Path to document file"},
					},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			content, _ := args["content"].(string)
			filePath, _ := args["file_path"].(string)
			if content == "" && filePath == "" {
				return nil, fmt.Errorf("rag_ingest: 'content' or 'file_path' is required")
			}
			_, err := ragProc.Ingest(ctx, domain.IngestRequest{Content: content, FilePath: filePath})
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "ingested"}, nil
		}, CategoryRAG)
	}

	if memSvc != nil {
		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_save",
				Description: "Save information to long-term memory for future reference",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{"type": "string", "description": "The information to remember"},
						"type":    map[string]interface{}{"type": "string", "description": "Type: fact, preference, skill, pattern, context"},
					},
					"required": []string{"content"},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			content, _ := args["content"].(string)
			if content == "" {
				return nil, fmt.Errorf("memory_save: 'content' argument is required")
			}
			memType := "fact"
			if t, ok := args["type"].(string); ok && t != "" {
				memType = t
			}
			svc.markRunMemorySaved()
			err := memSvc.Add(ctx, &domain.Memory{
				Type:       domain.MemoryType(memType),
				Content:    content,
				Importance: 0.8,
				Metadata:   map[string]interface{}{"source": "tool_call"},
			})
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "saved", "content": content}, nil
		}, CategoryMemory)

		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_recall",
				Description: "Recall information from long-term memory",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string", "description": "Query to search memory for"},
					},
					"required": []string{"query"},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			query, _ := args["query"].(string)
			if query == "" {
				return nil, fmt.Errorf("memory_recall: 'query' argument is required")
			}
			memories, err := memSvc.Search(ctx, query, 5)
			if err != nil {
				return nil, err
			}
			if len(memories) == 0 {
				allMems, _, listErr := memSvc.List(ctx, 10, 0)
				if listErr == nil && len(allMems) > 0 {
					var out []string
					for _, m := range allMems {
						out = append(out, fmt.Sprintf("- [%s] %s", m.Type, m.Content))
					}
					return map[string]interface{}{"memories": strings.Join(out, "\n")}, nil
				}
				return map[string]interface{}{"memories": ""}, nil
			}
			var out []string
			for _, m := range memories {
				out = append(out, fmt.Sprintf("- [%s: %.2f] %s", m.Type, m.Score, m.Content))
			}
			return map[string]interface{}{"memories": strings.Join(out, "\n"), "count": len(memories)}, nil
		}, CategoryMemory)

		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_update",
				Description: "Update an existing memory entry by its ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":      map[string]interface{}{"type": "string", "description": "Memory ID to update"},
						"content": map[string]interface{}{"type": "string", "description": "New content"},
					},
					"required": []string{"id", "content"},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			id, _ := args["id"].(string)
			content, _ := args["content"].(string)
			if id == "" || content == "" {
				return nil, fmt.Errorf("memory_update: 'id' and 'content' are required")
			}
			if err := memSvc.Update(ctx, id, content); err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "updated", "id": id}, nil
		}, CategoryMemory)

		svc.toolRegistry.Register(domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "memory_delete",
				Description: "Permanently remove a memory entry by its ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{"type": "string", "description": "Memory ID to delete"},
					},
					"required": []string{"id"},
				},
			},
		}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			id, _ := args["id"].(string)
			if id == "" {
				return nil, fmt.Errorf("memory_delete: 'id' argument is required")
			}
			if err := memSvc.Delete(ctx, id); err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "deleted", "id": id}, nil
		}, CategoryMemory)
	}
}

// buildPTCRouterOptions constructs ptc.RouterOption list for dynamic providers only.
// Static tools (RAG, Memory, custom) are registered via ToolRegistry.SyncToPTCRouter.
func buildPTCRouterOptions(mcpSvc MCPToolExecutor, skillsSvc *skills.Service) []ptc.RouterOption {
	var opts []ptc.RouterOption

	if mcpSvc != nil {
		opts = append(opts, ptc.WithMCPService(mcpSvc))
		mcpInfos := domainToolsToPTCInfos(mcpSvc.ListTools(), CategoryMCP)
		if len(mcpInfos) > 0 {
			opts = append(opts, ptc.WithMCPToolInfos(mcpInfos))
		}
	}

	if skillsSvc != nil {
		opts = append(opts, ptc.WithSkillsService(skillsSvc))
		skillList, _ := skillsSvc.ListSkills(context.Background(), skills.SkillFilter{})
		skillInfos := make([]ptc.ToolInfo, 0, len(skillList))
		for _, sk := range skillList {
			skillInfos = append(skillInfos, ptc.ToolInfo{
				Name:        sk.ID,
				Description: sk.Description,
				Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
				Category:    CategorySkill,
			})
		}
		if len(skillInfos) > 0 {
			opts = append(opts, ptc.WithSkillToolInfos(skillInfos))
		}
	}

	return opts
}

// domainToolsToPTCInfos converts domain.ToolDefinition slice to ptc.ToolInfo slice.
func domainToolsToPTCInfos(defs []domain.ToolDefinition, category string) []ptc.ToolInfo {
	infos := make([]ptc.ToolInfo, 0, len(defs))
	for _, d := range defs {
		params := d.Function.Parameters
		if params == nil {
			params = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		infos = append(infos, ptc.ToolInfo{
			Name:        d.Function.Name,
			Description: d.Function.Description,
			Parameters:  params,
			Category:    category,
		})
	}
	return infos
}
