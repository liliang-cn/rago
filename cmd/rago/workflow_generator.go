package rago

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/spf13/cobra"
)

var workflowGenerateCmd = &cobra.Command{
	Use:   "generate [description]",
	Short: "Generate a workflow from natural language description",
	Long: `Generate a workflow JSON file from a natural language description.
The AI will analyze your requirements and create an appropriate workflow using available MCP tools.

Examples:
  rago agent generate "Monitor a website every hour and save changes to a file"
  rago agent generate "Read all Python files and generate documentation"
  rago agent generate "Fetch news from RSS feed and create a daily summary"`,
	Aliases: []string{"gen", "create-from"},
	Args: cobra.MinimumNArgs(1),
	RunE: generateWorkflow,
}

func init() {
	agentCmd.AddCommand(workflowGenerateCmd)
	
	workflowGenerateCmd.Flags().StringP("output", "o", "", "Output file path (default: workflow.json)")
	workflowGenerateCmd.Flags().BoolP("execute", "e", false, "Execute the workflow immediately after generation")
	workflowGenerateCmd.Flags().BoolP("interactive", "i", false, "Interactive mode with refinement")
}

func generateWorkflow(cmd *cobra.Command, args []string) error {
	description := strings.Join(args, " ")
	outputPath, _ := cmd.Flags().GetString("output")
	execute, _ := cmd.Flags().GetBool("execute")
	interactive, _ := cmd.Flags().GetBool("interactive")
	
	if outputPath == "" {
		// Generate a filename based on description
		safeName := strings.ReplaceAll(strings.ToLower(description), " ", "_")
		if len(safeName) > 30 {
			safeName = safeName[:30]
		}
		outputPath = fmt.Sprintf("%s_workflow.json", safeName)
	}

	// Initialize LLM service
	ctx := context.Background()
	_, llmService, _, err := initializeProviders(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	fmt.Printf("🤖 Generating workflow for: %s\n", description)
	
	// Generate the workflow
	workflow, err := generateWorkflowFromDescription(ctx, llmService, description)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Interactive refinement
	if interactive {
		workflow, err = refineWorkflowInteractively(ctx, llmService, workflow, description)
		if err != nil {
			return err
		}
	}

	// Save the workflow
	workflowJSON, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	if err := os.WriteFile(outputPath, workflowJSON, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	fmt.Printf("✅ Workflow generated and saved to: %s\n", outputPath)
	
	// Display the workflow
	if verbose {
		fmt.Println("\n📋 Generated Workflow:")
		fmt.Println(string(workflowJSON))
	} else {
		fmt.Printf("\n📋 Workflow Summary:\n")
		fmt.Printf("  Steps: %d\n", len(workflow.Steps))
		for i, step := range workflow.Steps {
			fmt.Printf("    %d. %s (%s)\n", i+1, step.Name, step.Type)
		}
	}

	// Execute if requested
	if execute {
		fmt.Println("\n🚀 Executing generated workflow...")
		return executeGeneratedWorkflow(outputPath)
	}

	fmt.Println("\n💡 To execute this workflow:")
	fmt.Printf("   rago agent create --name \"My Agent\" --type workflow --workflow-file %s\n", outputPath)
	fmt.Println("   rago agent execute [agent-id]")

	return nil
}

func generateWorkflowFromDescription(ctx context.Context, llmService domain.Generator, description string) (*types.WorkflowSpec, error) {
	systemPrompt := `You are a workflow generation expert for RAGO. Generate valid workflow JSON based on user descriptions.

Available MCP tools:
1. filesystem - File operations (read, write, list, execute, move, copy, delete, mkdir)
2. fetch - HTTP/HTTPS requests (GET, POST, etc.)
3. memory - Temporary storage (store, retrieve, delete, append)
4. time - Date/time operations (now, format, parse)
5. sequential-thinking - Complex reasoning and analysis
6. sqlite - Database operations (if configured)

Workflow JSON structure:
{
  "steps": [
    {
      "id": "unique_id",
      "name": "Human readable name",
      "type": "tool|condition|loop|variable",
      "tool": "tool_name",
      "inputs": { "key": "value or {{variable}}" },
      "outputs": { "result_key": "variable_name" }
    }
  ],
  "triggers": [ /* optional */ ],
  "variables": { /* optional initial variables */ }
}

Rules:
1. Use {{variable}} syntax for referencing outputs from previous steps
2. Each step must have a unique id
3. Store intermediate results using outputs
4. Use meaningful step names
5. Add error handling where appropriate
6. Use loops for repetitive tasks
7. Use conditions for branching logic

Generate ONLY valid JSON, no explanations.`

	userPrompt := fmt.Sprintf(`Generate a workflow for: %s

Requirements:
- Break down the task into logical steps
- Use appropriate MCP tools
- Handle data flow between steps
- Add error handling if needed
- Make it production-ready

Return ONLY the JSON workflow specification.`, description)

	// Combine system and user prompts for the simple Generate interface
	fullPrompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)
	
	opts := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	}
	
	response, err := llmService.Generate(ctx, fullPrompt, opts)
	if err != nil {
		return nil, err
	}

	// Extract JSON from response
	jsonStr := extractJSON(response)
	
	// Parse the workflow
	var workflow types.WorkflowSpec
	if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
		// Try to fix common JSON errors
		jsonStr = fixCommonJSONErrors(jsonStr)
		if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
			return nil, fmt.Errorf("failed to parse generated workflow: %w", err)
		}
	}

	// Validate the workflow
	if err := validateGeneratedWorkflow(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	return &workflow, nil
}

func refineWorkflowInteractively(ctx context.Context, llmService domain.Generator, workflow *types.WorkflowSpec, originalDesc string) (*types.WorkflowSpec, error) {
	fmt.Println("\n🔄 Interactive Refinement Mode")
	fmt.Println("Type 'done' to finish, 'show' to display current workflow, or describe changes needed:")
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		if input == "done" {
			break
		}
		
		if input == "show" {
			workflowJSON, _ := json.MarshalIndent(workflow, "", "  ")
			fmt.Println(string(workflowJSON))
			continue
		}
		
		// Refine the workflow based on feedback
		refinedWorkflow, err := refineWorkflow(ctx, llmService, workflow, originalDesc, input)
		if err != nil {
			fmt.Printf("❌ Refinement failed: %v\n", err)
			continue
		}
		
		workflow = refinedWorkflow
		fmt.Println("✅ Workflow refined successfully")
	}
	
	return workflow, nil
}

func refineWorkflow(ctx context.Context, llmService domain.Generator, current *types.WorkflowSpec, originalDesc, feedback string) (*types.WorkflowSpec, error) {
	currentJSON, _ := json.MarshalIndent(current, "", "  ")
	
	prompt := fmt.Sprintf(`Original requirement: %s

Current workflow:
%s

User feedback: %s

Generate an improved workflow that addresses the feedback while maintaining the original requirements.
Return ONLY the complete JSON workflow specification.`, originalDesc, string(currentJSON), feedback)

	opts := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	}
	
	response, err := llmService.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(response)
	
	var workflow types.WorkflowSpec
	if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse refined workflow: %w", err)
	}

	return &workflow, nil
}

func extractJSON(content string) string {
	// Find JSON content between ```json and ``` or just extract the JSON object
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	
	// Try to find JSON object directly
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	
	return content
}

func fixCommonJSONErrors(jsonStr string) string {
	// Fix trailing commas
	jsonStr = strings.ReplaceAll(jsonStr, ",]", "]")
	jsonStr = strings.ReplaceAll(jsonStr, ",}", "}")
	
	// Fix single quotes
	jsonStr = strings.ReplaceAll(jsonStr, "'", "\"")
	
	// Fix missing quotes on keys (basic fix)
	// This is a simplified approach and may not catch all cases
	
	return jsonStr
}

func validateGeneratedWorkflow(workflow *types.WorkflowSpec) error {
	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}
	
	// Check for unique IDs
	ids := make(map[string]bool)
	for _, step := range workflow.Steps {
		if step.ID == "" {
			return fmt.Errorf("step missing ID")
		}
		if ids[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		ids[step.ID] = true
		
		if step.Name == "" {
			return fmt.Errorf("step %s missing name", step.ID)
		}
		
		if step.Type == "" {
			return fmt.Errorf("step %s missing type", step.ID)
		}
	}
	
	return nil
}

func executeGeneratedWorkflow(workflowPath string) error {
	// Create a temporary agent and execute it
	agentName := fmt.Sprintf("generated_%d", time.Now().Unix())
	
	// This would actually create and execute the agent
	// For now, we'll just show the command
	fmt.Printf("\nTo execute: rago agent create --name \"%s\" --workflow-file %s\n", agentName, workflowPath)
	
	return nil
}