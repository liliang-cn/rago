package rago

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/rago/v2/pkg/agents/generation"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

var agentGenerateCmd = &cobra.Command{
	Use:   "generate [description]",
	Short: "Generate an agent from a natural language description",
	Long: `Generate a complete agent definition from a natural language description.
Uses LLM with structured output generation for reliable JSON creation.

Examples:
  rago agent generate "monitor CPU usage and alert when above 80%"
  rago agent generate "research latest AI papers on arxiv" --type research
  rago agent generate "daily backup workflow for database" --type workflow --save`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentGenerate,
}

var (
	generateType   string
	generateSave   bool
	generateOutput string
)

func init() {
	agentCmd.AddCommand(agentGenerateCmd)
	
	agentGenerateCmd.Flags().StringVarP(&generateType, "type", "t", "workflow", "Agent type: workflow, research, or monitoring")
	agentGenerateCmd.Flags().BoolVarP(&generateSave, "save", "s", false, "Save the generated agent to file")
	agentGenerateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "Output file path (implies --save)")
}

func runAgentGenerate(cmd *cobra.Command, args []string) error {
	description := strings.Join(args, " ")
	
	// Map string to agent type
	var agentType types.AgentType
	switch strings.ToLower(generateType) {
	case "research":
		agentType = types.AgentTypeResearch
	case "monitoring":
		agentType = types.AgentTypeMonitoring
	case "workflow":
		agentType = types.AgentTypeWorkflow
	default:
		return fmt.Errorf("invalid agent type: %s (use workflow, research, or monitoring)", generateType)
	}
	
	fmt.Printf("🤖 Generating %s Agent\n", agentType)
	fmt.Printf("📝 Description: %s\n", description)
	fmt.Println("=" + strings.Repeat("=", 50))
	
	// Load config if needed
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	
	// Initialize LLM provider
	ctx := context.Background()
	_, llmService, _, err := initializeProviders(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}
	
	// Create generator
	generator := generation.NewAgentGenerator(llmService)
	if verbose {
		generator.SetVerbose(true)
	}
	
	// Generate the agent
	fmt.Println("\n🧠 Using LLM to generate agent definition...")
	agent, err := generator.GenerateAgent(ctx, description, agentType)
	if err != nil {
		return fmt.Errorf("failed to generate agent: %w", err)
	}
	
	fmt.Println("✅ Agent generated successfully!")
	
	// Display agent summary
	fmt.Printf("\n📋 Generated Agent:\n")
	fmt.Printf("   Name: %s\n", agent.Name)
	fmt.Printf("   Type: %s\n", agent.Type)
	fmt.Printf("   Description: %s\n", agent.Description)
	fmt.Printf("   Autonomy Level: %s\n", agent.Config.AutonomyLevel)
	
	if len(agent.Workflow.Steps) > 0 {
		fmt.Printf("\n   Workflow Steps (%d):\n", len(agent.Workflow.Steps))
		for i, step := range agent.Workflow.Steps {
			emoji := "🔧"
			switch step.Tool {
			case "fetch":
				emoji = "🌐"
			case "sequential-thinking":
				emoji = "🧠"
			case "filesystem":
				emoji = "📁"
			case "memory":
				emoji = "💾"
			}
			fmt.Printf("   %d. %s %s (%s)\n", i+1, emoji, step.Name, step.Tool)
		}
	}
	
	// Save if requested
	if generateSave || generateOutput != "" {
		outputPath := generateOutput
		if outputPath == "" {
			safeName := strings.ReplaceAll(strings.ToLower(agent.Name), " ", "_")
			outputPath = fmt.Sprintf("%s_agent.json", safeName)
		}
		
		agentJSON, err := json.MarshalIndent(agent, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal agent: %w", err)
		}
		
		if err := ioutil.WriteFile(outputPath, agentJSON, 0644); err != nil {
			return fmt.Errorf("failed to save agent: %w", err)
		}
		
		fmt.Printf("\n💾 Agent saved to: %s\n", outputPath)
		fmt.Printf("   You can now run it with: rago agent execute -f %s\n", outputPath)
	} else {
		// Display full JSON if not saving
		agentJSON, err := json.MarshalIndent(agent, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal agent: %w", err)
		}
		
		fmt.Printf("\n📄 Full Agent Definition:\n")
		fmt.Println(string(agentJSON))
	}
	
	return nil
}