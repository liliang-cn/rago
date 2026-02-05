// Package main shows how to use the rago agent library in your Go program
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/services"
)

func main() {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Initialize global pool (required for LLM/Embedding services)
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(ctx, cfg); err != nil {
		log.Fatal(err)
	}

	// 3. Get LLM and Embedding services
	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		log.Fatal(err)
	}

	_, err = globalPool.GetEmbeddingService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Create MCP service (optional, for tool integration)
	mcpSvc, err := mcp.NewService(&cfg.MCP, llmSvc)
	if err != nil {
		log.Printf("Warning: MCP service failed: %v", err)
		mcpSvc = nil
	} else {
		if err := mcpSvc.StartServers(ctx, nil); err != nil {
			log.Printf("Warning: MCP servers failed: %v", err)
		}
	}

	// 5. Create agent service
	agentDBPath := "/tmp/rago_agent_example.db"

	// MCP adapter for tool execution
	var mcpAdapter agent.MCPToolExecutor
	if mcpSvc != nil {
		mcpAdapter = &mcpToolWrapper{svc: mcpSvc}
	}

	agentSvc, err := agent.NewService(
		llmSvc,      // LLM service
		mcpAdapter,  // MCP tool executor (can be nil)
		nil,         // RAG processor (can be nil)
		agentDBPath, // Agent database path
		nil,         // Memory service (can be nil)
	)
	if err != nil {
		log.Fatal(err)
	}
	defer agentSvc.Close()

	// Optional: Set progress callback
	agentSvc.SetProgressCallback(func(event agent.ProgressEvent) {
		log.Printf("[%s] Round %d: %s", event.Type, event.Round, event.Message)
	})

	// 6. Example 1: Simple run
	fmt.Println("\n--- Example 1: Simple Run ---")
	result, err := agentSvc.Run(ctx, "What is 2+2?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result.FinalResult)

	// 7. Example 2: Run with session (conversation history)
	fmt.Println("\n--- Example 2: Session-based Run ---")
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	// First message
	result2, _ := agentSvc.RunWithSession(ctx, "My name is Alice", sessionID)
	fmt.Printf("Alice: %v\n", result2.FinalResult)

	// Second message (remembers context)
	result3, _ := agentSvc.RunWithSession(ctx, "What's my name?", sessionID)
	fmt.Printf("Memory: %v\n", result3.FinalResult)

	// 8. Example 3: Plan only (no execution)
	fmt.Println("\n--- Example 3: Plan Only ---")
	plan, err := agentSvc.Plan(ctx, "Write a Go function that calculates fibonacci")
	if err != nil {
		log.Printf("Plan error: %v", err)
	} else {
		fmt.Printf("Plan ID: %s\n", plan.ID)
		fmt.Printf("Goal: %s\n", plan.Goal)
		fmt.Println("Steps:")
		for _, step := range plan.Steps {
			fmt.Printf("  - [%s] %s (tool: %s)\n", step.ID, step.Description, step.Tool)
		}
	}

	// 9. Example 4: List sessions
	fmt.Println("\n--- Example 4: List Sessions ---")
	sessions, err := agentSvc.ListSessions(10)
	if err == nil {
		fmt.Printf("Found %d sessions\n", len(sessions))
		for _, s := range sessions {
			fmt.Printf("  - %s: %d messages\n", s.ID, len(s.Messages))
		}
	}

	// 10. Example 5: Get session details
	if len(sessions) > 0 {
		fmt.Println("\n--- Example 5: Get Session Details ---")
		session, err := agentSvc.GetSession(sessions[0].ID)
		if err == nil {
			fmt.Printf("Session: %s\n", session.ID)
			fmt.Printf("Agent: %s\n", session.AgentID)
			fmt.Printf("Messages: %d\n", len(session.Messages))
		}
	}

	fmt.Println("\n--- Examples Complete ---")
}

// mcpToolWrapper wraps mcp.Service to implement agent.MCPToolExecutor
type mcpToolWrapper struct {
	svc *mcp.Service
}

func (w *mcpToolWrapper) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	result, err := w.svc.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("MCP tool error: %s", result.Error)
	}
	return result.Data, nil
}

func (w *mcpToolWrapper) ListTools() []domain.ToolDefinition {
	tools := w.svc.GetAvailableTools(context.Background())
	result := make([]domain.ToolDefinition, 0, len(tools))

	for _, t := range tools {
		var parameters map[string]interface{}
		if t.InputSchema != nil && len(t.InputSchema) > 0 {
			parameters = t.InputSchema
		} else {
			parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments",
					},
				},
			}
		}

		result = append(result, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  parameters,
			},
		})
	}
	return result
}
