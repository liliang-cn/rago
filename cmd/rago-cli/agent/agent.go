package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/skills"
	"github.com/liliang-cn/rago/v2/pkg/store"
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

		// Use the new Event-Driven Stream Runner
		return runStream(ctx, goal)
	},
}

// executeCmd executes an existing plan by ID
var executeCmd = &cobra.Command{
	Use:   "execute [plan-id]",
	Short: "Execute an existing plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		ctx := context.Background()

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		// Get the plan
		plan, err := agentService.GetPlan(planID)
		if err != nil {
			return fmt.Errorf("plan not found: %w", err)
		}

		fmt.Printf("🎯 Executing Plan: %s\n", plan.ID)
		fmt.Printf("📋 Goal: %s\n\n", plan.Goal)

		// Execute the plan
		result, err := agentService.ExecutePlan(ctx, plan)
		if err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}

		// Print result
		fmt.Println("\n--- Results ---")
		if result.FinalResult != nil {
			fmt.Printf("\n--- Final Result ---\n%v\n", result.FinalResult)
		}
		if result.Duration != "" {
			fmt.Printf("Duration: %s\n", result.Duration)
		}
		fmt.Printf("Steps: %d done, %d failed\n", result.StepsDone, result.StepsFailed)

		return nil
	},
}

// planCmd is the parent command for plan operations
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Plan operations",
}

var planCreateCmd = &cobra.Command{
	Use:   "create [goal]",
	Short: "Create an agent plan (without execution)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := args[0]
		ctx := context.Background()

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		// Use agent service's Plan method
		plan, err := agentService.Plan(ctx, goal)
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

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent plans",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		plans, err := agentService.ListPlans("", 20)
		if err != nil {
			return fmt.Errorf("failed to list plans: %w", err)
		}

		if len(plans) == 0 {
			fmt.Println("No plans found")
			return nil
		}

		fmt.Println("Agent Plans:")
		for _, p := range plans {
			fmt.Printf("  [%s] %s\n", p.ID, p.Goal)
			fmt.Printf("     Status: %s | Steps: %d | Created: %s\n",
				p.Status, len(p.Steps), p.CreatedAt.Format("2006-01-02 15:04"))
		}

		return nil
	},
}

var planGetCmd = &cobra.Command{
	Use:   "get [plan-id]",
	Short: "Get plan details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		ctx := context.Background()

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		plan, err := agentService.GetPlan(planID)
		if err != nil {
			return fmt.Errorf("plan not found: %w", err)
		}

		fmt.Printf("📋 Plan ID: %s\n", plan.ID)
		fmt.Printf("Goal: %s\n", plan.Goal)
		fmt.Printf("Status: %s\n", plan.Status)
		fmt.Printf("Created: %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("\nSteps:\n")
		for i, step := range plan.Steps {
			status := "✓"
			if step.Status != "completed" {
				status = "✗"
			}
			fmt.Printf("  %d. [%s] %s\n", i+1, status, step.Description)
			fmt.Printf("     Tool: %s\n", step.Tool)
			if step.Error != "" {
				fmt.Printf("     Error: %s\n", step.Error)
			}
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

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		sessions, err := agentService.ListSessions(20)
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found")
			return nil
		}

		fmt.Println("Agent Sessions:")
		for _, s := range sessions {
			fmt.Printf("  [%s] %s - %d messages\n", s.ID, s.CreatedAt.Format("2006-01-02 15:04"), len(s.GetMessages()))
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
		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		session, err := agentService.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}

		fmt.Printf("Session ID: %s\n", session.GetID())
		fmt.Printf("Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Messages: %d\n", len(session.GetMessages()))

		return nil
	},
}

// reviseCmd revises an existing plan
var reviseCmd = &cobra.Command{
	Use:   "revise [plan-id] [instruction]",
	Short: "Revise an existing plan with natural language",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		instruction := args[1]
		ctx := context.Background()

		_, agentService, err := initAgentServices(ctx)
		if err != nil {
			return err
		}
		defer agentService.Close()

		// Get the original plan
		plan, err := agentService.GetPlan(planID)
		if err != nil {
			return fmt.Errorf("plan not found: %w", err)
		}

		fmt.Printf("📋 Original Plan: %s\n\n", plan.Goal)
		fmt.Println("Steps:")
		for i, step := range plan.Steps {
			fmt.Printf("  %d. [%s] %s\n", i+1, step.Tool, step.Description)
		}

		// Revise the plan
		fmt.Printf("\n✏️  Revising with: %s\n\n", instruction)
		revised, err := agentService.RevisePlan(ctx, plan, instruction)
		if err != nil {
			return fmt.Errorf("revision failed: %w", err)
		}

		fmt.Printf("📋 Revised Plan ID: %s\n", revised.ID)
		fmt.Printf("Goal: %s\n\n", revised.Goal)
		fmt.Println("Revised Steps:")
		for i, step := range revised.Steps {
			fmt.Printf("  %d. [%s] %s\n", i+1, step.Tool, step.Description)
		}

		return nil
	},
}

func init() {
	AgentCmd.AddCommand(runCmd)
	AgentCmd.AddCommand(planCmd)
	planCmd.AddCommand(planCreateCmd)
	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planGetCmd)
	AgentCmd.AddCommand(executeCmd)
	AgentCmd.AddCommand(reviseCmd)
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

func (a *mcpToolAdapter) AddServer(ctx context.Context, name string, command string, args []string) error {
	return a.service.AddDynamicServer(ctx, name, command, args)
}

// initializeSkills initializes the skills service
func initializeSkills(ctx context.Context, ragClient *rag.Client) error {
	skillsInitOnce.Do(func() {
		// Initialize skills service
		cfg := skills.DefaultConfig()
		cfg.AutoLoad = true
		cfg.Paths = []string{Cfg.SkillsDir()}
		cfg.DBPath = filepath.Join(Cfg.DataDir(), "skills.db")

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

// convertMCPToSkills converts MCP servers to skills (optional auto-conversion)
func convertMCPToSkills(ctx context.Context, mcpService *mcp.Service) error {
	// Check if auto-conversion is enabled via environment variable
	autoConvert := os.Getenv("RAGO_AUTO_CONVERT_MCP") == "true"
	if !autoConvert {
		return nil
	}

	fmt.Println("🔄 Auto-converting MCP servers to Skills...")

	// Create converter config
	convCfg := mcp.DefaultConverterConfig()
	convCfg.OutputDir = Cfg.SkillsDir()

	// Create converter
	converter, err := mcp.NewConverter(convCfg, mcpService)
	if err != nil {
		return fmt.Errorf("failed to create converter: %w", err)
	}

	// Convert all servers
	skills, err := converter.ConvertAllServers(ctx)
	if err != nil {
		fmt.Printf("⚠️  Some conversions failed: %v\n", err)
	}

	// Re-initialize skills service to load new skills
	if skillsService != nil && len(skills) > 0 {
		_ = skillsService.LoadAll(ctx)
		fmt.Printf("✓ Converted %d MCP servers to Skills\n", len(skills))
	}

	return nil
}

// initializeMemoryService initializes the memory service for long-term agent memory
func initializeMemoryService(ctx context.Context, llmService domain.Generator, embedService domain.Embedder) (domain.MemoryService, error) {
	if embedService == nil {
		return nil, nil // Embedder required
	}

	// Create memory store
	memDBPath := filepath.Join(Cfg.DataDir(), "memory.db")
	memStore, err := store.NewMemoryStore(memDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	// Initialize schema
	if err := memStore.InitSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to init memory schema: %w", err)
	}

	// Create memory service
	memSvc := memory.NewService(memStore, llmService, embedService, memory.DefaultConfig())

	return memSvc, nil
}

// initAgentServices initializes RAG client and agent service
func initAgentServices(ctx context.Context) (*rag.Client, *agent.Service, error) {
	globalLLM := services.GetGlobalLLMService()
	llmService, err := globalLLM.GetLLMService()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LLM service: %w", err)
	}

	// Embedding service is optional for agent (RAG features)
	var embedService domain.Embedder
	if e, err := globalLLM.GetEmbeddingService(ctx); err == nil {
		embedService = e
	} else {
		fmt.Printf("⚠️  Embedding service not available, RAG features disabled\n")
	}

	var metadataExtractor domain.MetadataExtractor
	if ext, ok := llmService.(domain.MetadataExtractor); ok {
		metadataExtractor = ext
	} else if embedService != nil {
		if ext, ok := embedService.(domain.MetadataExtractor); ok {
			metadataExtractor = ext
		}
	}

	// Only create RAG client if embedding service is available
	var ragClient *rag.Client
	if embedService != nil {
		ragClient, err = rag.NewClient(Cfg, embedService, llmService, metadataExtractor)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to init RAG client: %w", err)
		}
	} else {
		fmt.Printf("⚠️  RAG client not initialized (no embedding service)\n")
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

	// Auto-convert MCP servers to Skills if enabled
	if err := convertMCPToSkills(ctx, mcpService); err != nil {
		fmt.Printf("⚠️  Warning: MCP to Skills conversion failed: %v\n", err)
	}

	// Initialize Memory Service for long-term agent memory
	memoryService, err := initializeMemoryService(ctx, llmService, embedService)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to initialize memory service: %v\n", err)
	} else if memoryService != nil {
		fmt.Printf("✓ Memory service initialized\n")
	}

	agentDBPath := getAgentDBPath()
	adapter := &mcpToolAdapter{service: mcpService}

	// Get RAG processor (nil if RAG is not available)
	var processor domain.Processor
	if ragClient != nil {
		processor = ragClient.GetProcessor()
	}

	agentService, err := agent.NewService(llmService, adapter, processor, agentDBPath, memoryService)
	if err != nil {
		mcpService.Close()
		if ragClient != nil {
			ragClient.Close()
		}
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
	// RAG tools (rag_query, rag_ingest) = 2, only available if RAG client is initialized
	ragToolCount := 0
	if ragClient != nil {
		ragToolCount = 2
	}
	totalTools := len(mcpTools) + skillToolsCount + ragToolCount
	fmt.Printf("✓ Available tools: %d (MCP: %d, Skills: %d, RAG: %d)\n",
		totalTools, len(mcpTools), skillToolsCount, ragToolCount)

	return ragClient, agentService, nil
}

// getAgentDBPath returns the agent database path
func getAgentDBPath() string {
	// Use Cfg.DataDir() for agent database
	return filepath.Join(Cfg.DataDir(), "agent.db")
}

// runStream runs the agent with Event Loop streaming output
func runStream(ctx context.Context, goal string) error {
	fmt.Printf("🎯 Agent Goal: %s\n\n", goal)

	ragClient, agentService, err := initAgentServices(ctx)
	if err != nil {
		return err
	}
	if ragClient != nil {
		defer ragClient.Close()
	}
	defer agentService.Close()

	// Start streaming
	events, err := agentService.RunStream(ctx, goal)
	if err != nil {
		return err
	}

	// Consume events
	var currentRound int
	for evt := range events {
		switch evt.Type {
		case agent.EventTypeStart:
			fmt.Printf("🚀 %s\n", evt.Content)
		case agent.EventTypeThinking:
			currentRound++
			fmt.Printf("\n🔄 [Round %d] Thinking...\n", currentRound)
		case agent.EventTypeToolCall:
			fmt.Printf("🛠️  Using Tool: %s (args: %v)\n", evt.ToolName, evt.ToolArgs)
		case agent.EventTypeToolResult:
			fmt.Printf("✅ Tool Success: %s\n", evt.ToolName)
		case agent.EventTypeHandoff:
			fmt.Printf("🔀 Handoff: %s\n", evt.Content)
		case agent.EventTypePartial:
			// Print text as it comes (Typewriter effect)
			fmt.Print(evt.Content)
		case agent.EventTypeComplete:
			fmt.Printf("\n\n🏁 Task Completed!\n")
		case agent.EventTypeError:
			fmt.Printf("\n❌ Error: %s\n", evt.Content)
		}
	}

	return nil
}

// runSimple runs the agent with simple text output
func runSimple(ctx context.Context, goal string) error {
	fmt.Printf("🎯 Agent Goal: %s\n\n", goal)

	ragClient, agentService, err := initAgentServices(ctx)
	if err != nil {
		return err
	}
	if ragClient != nil {
		defer ragClient.Close()
	}
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
