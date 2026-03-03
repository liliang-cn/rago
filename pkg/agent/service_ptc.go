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
