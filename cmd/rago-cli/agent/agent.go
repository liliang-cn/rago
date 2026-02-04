package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/spf13/cobra"
)

var (
	Cfg            *config.Config
	Verbose        bool
	skillsService  *skills.Service
	skillsInitOnce sync.Once
	skillsInitErr  error
)

// SetSharedVariables sets the shared variables from the root command
func SetSharedVariables(c *config.Config, v bool) {
	Cfg = c
	Verbose = v
}

// AgentCmd is the main agent command
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Autonomous agent operations",
	Long:  `Run autonomous agent tasks, planning, and execution.`,
}

// runCmd runs an agent task
var runCmd = &cobra.Command{
	Use:   "run [goal]",
	Short: "Run an agent task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := args[0]
		ctx := context.Background()

		// Always use simple text output
		return runSimple(ctx, goal)
	},
}

// planCmd creates a plan without executing
var planCmd = &cobra.Command{
	Use:   "plan [goal]",
	Short: "Create an agent plan (without execution)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := args[0]
		ctx := context.Background()

		ragClient, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer ragClient.Close()
		defer agentService.Close()

		// Use library's PlanAgent method
		plan, err := ragClient.PlanAgent(ctx, goal)
		if err != nil {
			return fmt.Errorf("planning failed: %w", err)
		}

		// Print plan
		fmt.Printf("📋 Plan ID: %s\n", plan.ID)
		fmt.Printf("Goal: %s\n\n", plan.Goal)
		fmt.Println("Steps:")
		for _, step := range plan.Steps {
			fmt.Printf("  [%s] %s\n  └─ Tool: %s\n", step.ID, step.Description, step.Tool)
		}

		return nil
	},
}

// sessionCmd manages agent sessions
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage agent sessions",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent sessions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		ragClient, _, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer ragClient.Close()

		sessions, err := ragClient.ListAgentSessions(20)
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found")
			return nil
		}

		fmt.Println("Agent Sessions:")
		for _, s := range sessions {
			fmt.Printf("  [%s] %s - %d messages\n", s.ID, s.CreatedAt.Format("2006-01-02 15:04"), len(s.Messages))
		}

		return nil
	},
}

var sessionGetCmd = &cobra.Command{
	Use:   "get [session-id]",
	Short: "Get session details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		ctx := context.Background()
		ragClient, _, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer ragClient.Close()

		session, err := ragClient.GetAgentSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}

		fmt.Printf("Session ID: %s\n", session.ID)
		fmt.Printf("Agent ID: %s\n", session.AgentID)
		fmt.Printf("Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Messages: %d\n", len(session.Messages))

		return nil
	},
}

func init() {
	AgentCmd.AddCommand(runCmd)
	AgentCmd.AddCommand(planCmd)
	AgentCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionGetCmd)
}

// mcpToolAdapter adapts mcp.Service to agent.MCPToolExecutor
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
		// Use the full InputSchema if available, otherwise use minimal schema
		var parameters map[string]interface{}
		if t.InputSchema != nil && len(t.InputSchema) > 0 {
			// Use the actual schema from the MCP tool
			parameters = t.InputSchema
		} else {
			// Fallback to minimal schema
			parameters = map[string]interface{}{
				"type": "object",
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

// initializeSkills initializes the skills service
func initializeSkills(ctx context.Context, ragClient *rag.Client) error {
	skillsInitOnce.Do(func() {
		// Initialize skills service
		cfg := skills.DefaultConfig()
		cfg.AutoLoad = true

		// Create in-memory store for skills persistence
		skillStore := skills.NewMemoryStore()

		var err error
		skillsService, err = skills.NewService(cfg)
		if err != nil {
			skillsInitErr = fmt.Errorf("failed to create skills service: %w", err)
			return
		}
		skillsService.SetStore(skillStore)

		// Set RAG processor from ragClient
		if ragClient != nil && ragClient.GetProcessor() != nil {
			skillsService.SetRAGService(ragClient.GetProcessor())
		}

		// Auto-load skills
		if err := skillsService.LoadAll(ctx); err != nil {
			skillsInitErr = fmt.Errorf("failed to load skills: %w", err)
			return
		}
	})
	return skillsInitErr
}

// initAgentServices initializes RAG client and agent service
func initAgentServices(ctx context.Context) (*rag.Client, *agent.Service, error) {
	globalLLM := services.GetGlobalLLMService()
	llmService, err := globalLLM.GetLLMService()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LLM service: %w", err)
	}

	embedService, err := globalLLM.GetEmbeddingService(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get embedder service: %w", err)
	}

	var metadataExtractor domain.MetadataExtractor
	if ext, ok := llmService.(domain.MetadataExtractor); ok {
		metadataExtractor = ext
	} else if ext, ok := embedService.(domain.MetadataExtractor); ok {
		metadataExtractor = ext
	}

	ragClient, err := rag.NewClient(Cfg, embedService, llmService, metadataExtractor)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init RAG client: %w", err)
	}

	// Initialize MCP service (stdio servers are started on-demand)
	mcpConfig := &Cfg.MCP
	mcpService, err := mcp.NewService(mcpConfig, llmService)
	if err != nil {
		ragClient.Close()
		return nil, nil, fmt.Errorf("failed to init MCP service: %w", err)
	}

	// Start MCP servers BEFORE creating agent service (so tools are available)
	if err := mcpService.StartServers(ctx, nil); err != nil {
		// Log warning but continue - some tools might still work
		fmt.Printf("⚠️  Warning: Some MCP servers failed to start: %v\n", err)
	}

	agentDBPath := getAgentDBPath()
	adapter := &mcpToolAdapter{service: mcpService}
	agentService, err := agent.NewService(llmService, adapter, ragClient.GetProcessor(), agentDBPath, nil)
	if err != nil {
		mcpService.Close()
		ragClient.Close()
		return nil, nil, fmt.Errorf("failed to init agent service: %w", err)
	}

	// Initialize Semantic Router for improved intent recognition
	if embedService != nil {
		routerCfg := router.DefaultConfig()
		routerCfg.Threshold = 0.75 // Slightly lower for better coverage
		routerService, err := router.NewService(embedService, routerCfg)
		if err == nil {
			// Register default intents
			if err := routerService.RegisterDefaultIntents(); err == nil {
				agentService.SetRouter(routerService)
				fmt.Printf("✓ Semantic Router initialized with %d intents\n", len(routerService.ListIntents()))
			} else {
				fmt.Printf("⚠️  Warning: Failed to register default intents: %v\n", err)
			}
		} else {
			fmt.Printf("⚠️  Warning: Failed to initialize semantic router: %v\n", err)
		}
	}

	// Initialize and set skills service
	if err := initializeSkills(ctx, ragClient); err != nil {
		fmt.Printf("⚠️  Warning: Failed to initialize skills service: %v\n", err)
	} else if skillsService != nil {
		agentService.SetSkillsService(skillsService)
		allSkills, _ := skillsService.ListSkills(ctx, skills.SkillFilter{})
		fmt.Printf("✓ Skills service initialized with %d skills\n", len(allSkills))
	}

	// Display available tools summary
	mcpTools := mcpService.GetAvailableTools(ctx)
	skillToolsCount := 0
	if skillsService != nil {
		allSkills, _ := skillsService.ListSkills(ctx, skills.SkillFilter{})
		skillToolsCount = len(allSkills)
	}
	// RAG tools (rag_query, rag_ingest) = 2
	ragToolCount := 2
	totalTools := len(mcpTools) + skillToolsCount + ragToolCount
	fmt.Printf("✓ Available tools: %d (MCP: %d, Skills: %d, RAG: %d)\n",
		totalTools, len(mcpTools), skillToolsCount, ragToolCount)

	return ragClient, agentService, nil
}

// getAgentDBPath returns the agent database path
func getAgentDBPath() string {
	dbPath := Cfg.Sqvect.DBPath
	// Default path
	agentDBPath := "./.rago/data/agent.db"
	if len(dbPath) > 3 {
		// Use same directory as main DB but with agent.db filename
		dir := filepath.Dir(dbPath)
		if dir == "." || dir == "" {
			agentDBPath = "agent.db"
		} else {
			agentDBPath = filepath.Join(dir, "agent.db")
		}
	}
	return agentDBPath
}

// runSimple runs the agent with simple text output
func runSimple(ctx context.Context, goal string) error {
	fmt.Printf("🎯 Agent Goal: %s\n\n", goal)

	ragClient, agentService, err := initAgentServices(ctx)
	if err != nil {
		return err
	}
	defer ragClient.Close()
	defer agentService.Close()

	// Set up progress callback
	var lastRound int
	agentService.SetProgressCallback(func(event agent.ProgressEvent) {
		switch event.Type {
		case "thinking":
			if event.Round != lastRound {
				fmt.Printf("🔄 Round %d: Thinking...\n", event.Round)
				lastRound = event.Round
			}
		case "tool_call":
			if event.Tool != "" {
				fmt.Printf("   → %s\n", event.Message)
			}
		case "tool_result":
			fmt.Printf("   %s\n", event.Message)
		}
	})

	// Execute
	result, err := agentService.Run(ctx, goal)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Try to fetch plan details for display (if plan was created)
	plan, planErr := agentService.GetPlan(result.PlanID)

	// Print Results
	fmt.Println("\n✅ Results:")
	if planErr == nil && plan != nil {
		// Display plan steps
		for _, step := range plan.Steps {
			if step.Status == agent.StepCompleted && step.Result != nil {
				fmt.Printf("\n--- %s ---\n", step.Description)
				fmt.Println(formatResult(step.Result))
			} else if step.Status == agent.StepFailed {
				fmt.Printf("\n❌ --- %s (FAILED) ---\n", step.Description)
				fmt.Println(step.Error)
			}
		}
	} else {
		// No plan, just show final result
		if result.FinalResult != nil {
			fmt.Printf("\n--- Final Result ---\n")
			fmt.Println(formatResult(result.FinalResult))
		}
	}

	fmt.Println("\nDone!")
	return nil
}

// formatResult formats the result for display
func formatResult(v interface{}) string {
	if v == nil {
		return "(empty)"
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
