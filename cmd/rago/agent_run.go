package rago

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/execution"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/spf13/cobra"
)

var agentRunCmd = &cobra.Command{
	Use:   "run [natural language request]",
	Short: "Generate and execute a workflow from natural language",
	Long: `Generate a workflow from natural language and execute it immediately.
This command uses LLM to understand your request, create an appropriate workflow,
and execute it - all in one step.

Examples:
  rago agent run "check the latest iPhone price and analyze if it's worth buying"
  rago agent run "monitor github.com/golang/go for new releases"
  rago agent run "fetch weather data for San Francisco and create a summary"
  rago agent run "analyze all JSON files in current directory and generate report"`,
	Aliases: []string{"do", "nl"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    runNaturalLanguageAgent,
}

func init() {
	agentCmd.AddCommand(agentRunCmd)

	agentRunCmd.Flags().BoolP("save", "s", false, "Save the generated workflow to file")
	agentRunCmd.Flags().StringP("output", "o", "", "Output file for workflow (implies --save)")
	agentRunCmd.Flags().BoolP("dry-run", "d", false, "Generate workflow but don't execute")
	agentRunCmd.Flags().BoolP("interactive", "i", false, "Review workflow before execution")
}

func runNaturalLanguageAgent(cmd *cobra.Command, args []string) error {
	request := strings.Join(args, " ")
	save, _ := cmd.Flags().GetBool("save")
	outputPath, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	interactive, _ := cmd.Flags().GetBool("interactive")

	if outputPath != "" {
		save = true
	}

	fmt.Printf("🤖 Natural Language Request: %s\n", request)
	fmt.Println("=" + strings.Repeat("=", 50))

	// Load config if not already loaded
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Initialize providers
	ctx := context.Background()
	_, llmService, _, err := initializeProviders(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	// Step 1: Generate workflow from natural language
	fmt.Println("\n📝 Step 1: Generating workflow from your request...")
	fmt.Println("   🧠 Calling LLM to understand and create workflow...")

	workflow, err := generateSmartWorkflow(ctx, llmService, request)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	fmt.Printf("   ✅ Generated workflow with %d steps\n", len(workflow.Steps))

	// Display workflow summary
	fmt.Println("\n📋 Generated Workflow:")
	for i, step := range workflow.Steps {
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
		case "time":
			emoji = "⏰"
		}
		fmt.Printf("   %d. %s %s (%s)\n", i+1, emoji, step.Name, step.Tool)
	}

	// Save workflow if requested
	if save || outputPath != "" {
		if outputPath == "" {
			safeName := strings.ReplaceAll(strings.ToLower(request), " ", "_")
			if len(safeName) > 30 {
				safeName = safeName[:30]
			}
			outputPath = fmt.Sprintf("%s_workflow_%d.json", safeName, time.Now().Unix())
		}

		workflowJSON, _ := json.MarshalIndent(workflow, "", "  ")
		if err := os.WriteFile(outputPath, workflowJSON, 0644); err != nil {
			fmt.Printf("⚠️  Failed to save workflow: %v\n", err)
		} else {
			fmt.Printf("\n💾 Workflow saved to: %s\n", outputPath)
		}
	}

	// Interactive review
	if interactive && !dryRun {
		fmt.Print("\n❓ Execute this workflow? (y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("❌ Execution cancelled")
			return nil
		}
	}

	if dryRun {
		fmt.Println("\n🔍 Dry run mode - workflow generated but not executed")
		return nil
	}

	// Step 2: Execute the workflow
	fmt.Println("\n⚡ Step 2: Executing the generated workflow...")
	fmt.Printf("   Workflow has %d steps\n", len(workflow.Steps))

	// Create workflow executor with LLM
	executor := execution.NewWorkflowExecutor(cfg, llmService)
	executor.SetVerbose(verbose)

	// Add input variables from the request
	workflow.Variables = extractWorkflowInputs(request)

	// Execute the workflow
	startTime := time.Now()
	result, err := executor.Execute(ctx, workflow)
	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	duration := time.Since(startTime)

	// Step 3: Display results
	fmt.Println("\n✅ Workflow completed successfully!")
	fmt.Printf("⏱️  Execution time: %v\n", duration)

	// Display key outputs
	fmt.Println("\n📊 Results:")
	if len(result.Outputs) == 0 {
		fmt.Println("   (No outputs generated)")
	} else {
		for key, value := range result.Outputs {
			// Show all outputs, not just specific ones
			fmt.Printf("\n--- %s ---\n", key)
			if str, ok := value.(string); ok {
				if len(str) > 500 {
					fmt.Println(str[:500] + "...\n[truncated]")
				} else {
					fmt.Println(str)
				}
			} else {
				// Format non-string values nicely
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > 500 {
					fmt.Println(valueStr[:500] + "...\n[truncated]")
				} else {
					fmt.Println(valueStr)
				}
			}
		}
	}

	// Check for saved files
	if savedFiles := findSavedFiles(result); len(savedFiles) > 0 {
		fmt.Println("\n📁 Files created:")
		for _, file := range savedFiles {
			fmt.Printf("   - %s\n", file)
		}
	}

	fmt.Println("\n🎉 Task completed! Your request has been processed.")

	return nil
}

func generateSmartWorkflow(ctx context.Context, llm domain.Generator, request string) (*types.WorkflowSpec, error) {
	systemPrompt := `You are a workflow generator for RAGO. Generate valid workflow JSON based on user requests.

Available MCP tools:
1. filesystem - File operations (read, write, list, execute, move, copy, delete, mkdir)
2. fetch - HTTP/HTTPS requests for APIs and websites
3. memory - Temporary storage (store, retrieve, delete, append)
4. time - Date/time operations (now, format, parse)
5. sequential-thinking - LLM analysis and reasoning (this calls the AI for complex tasks)

IMPORTANT: Data flows between steps using variables:
- When a step produces output, store it in a variable using "outputs"
- When a step needs that data, reference it using {{variableName}} in "inputs"
- For sequential-thinking, always pass data using "data": "{{variableName}}"

IMPORTANT API URL Guidelines:
- For GitHub repositories, use the GitHub API: https://api.github.com/repos/{owner}/{repo}
  Example: github.com/golang/go → https://api.github.com/repos/golang/go
- For GitHub commits: https://api.github.com/repos/{owner}/{repo}/commits
- For GitHub releases: https://api.github.com/repos/{owner}/{repo}/releases
- Always include headers for GitHub API: {"Accept": "application/vnd.github.v3+json"}

FREE NEWS SOURCES (No API key required):
- BBC Tech RSS: https://feeds.bbci.co.uk/news/technology/rss.xml
- BBC World RSS: https://feeds.bbci.co.uk/news/world/rss.xml
- Hacker News Top Stories: https://hacker-news.firebaseio.com/v0/topstories.json
- Hacker News Story Detail: https://hacker-news.firebaseio.com/v0/item/{id}.json
- Reddit JSON (add .json to any subreddit URL): https://www.reddit.com/r/technology/.json
- The Verge RSS: https://www.theverge.com/rss/index.xml
- TechCrunch RSS: https://techcrunch.com/feed/

Common patterns:
- Fetch data → Analyze with sequential-thinking → Store/Present results
- Use RSS feeds for news without API keys
- Use proper API endpoints, not HTML pages
- Pass data between steps using {{variableName}}`

	userPrompt := fmt.Sprintf(`User request: "%s"

Generate a complete workflow that accomplishes this request.
Return a valid JSON workflow structure.`, request)

	fullPrompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)

	opts := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	// Define the expected schema
	var workflowSchema types.WorkflowSpec

	// Use GenerateStructured for type-safe JSON generation
	result, err := llm.GenerateStructured(ctx, fullPrompt, &workflowSchema, opts)
	if err != nil {
		// Fallback to regular generation if structured generation fails
		if verbose {
			fmt.Printf("⚠️  Structured generation failed, falling back to regular generation: %v\n", err)
		}
		return generateWorkflowFallback(ctx, llm, request, opts)
	}

	// Debug output
	if verbose {
		fmt.Printf("\n🔍 Structured Response Valid: %v\n", result.Valid)
		fmt.Printf("📄 Generated Workflow JSON:\n%s\n", result.Raw)
	}

	// Extract the workflow from the result
	workflow, ok := result.Data.(*types.WorkflowSpec)
	if !ok {
		// Try to unmarshal from Raw if Data type assertion fails
		var fallbackWorkflow types.WorkflowSpec
		if err := json.Unmarshal([]byte(result.Raw), &fallbackWorkflow); err != nil {
			return nil, fmt.Errorf("failed to extract workflow from structured result: %w", err)
		}
		workflow = &fallbackWorkflow
	}

	// Add default variables if needed
	if workflow.Variables == nil {
		workflow.Variables = make(map[string]interface{})
	}

	// Ensure workflow is valid
	if len(workflow.Steps) == 0 {
		return nil, fmt.Errorf("generated workflow has no steps")
	}

	return workflow, nil
}

// generateWorkflowFallback is the original implementation as fallback
func generateWorkflowFallback(ctx context.Context, llm domain.Generator, request string, opts *domain.GenerationOptions) (*types.WorkflowSpec, error) {
	systemPrompt := `You are a workflow generator for RAGO. Generate ONLY valid workflow JSON.

Available MCP tools: filesystem, fetch, memory, time, sequential-thinking

EXACT JSON FORMAT REQUIRED:
{
  "steps": [
    {
      "id": "step1",
      "name": "Step Name",
      "type": "tool",
      "tool": "tool_name",
      "inputs": {"key": "value"},
      "outputs": {"outputKey": "variableName"}
    }
  ]
}

Return ONLY this JSON structure.`

	userPrompt := fmt.Sprintf(`User request: "%s"
Generate a complete workflow. Return ONLY the JSON workflow.`, request)

	fullPrompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)

	response, err := llm.Generate(ctx, fullPrompt, opts)
	if err != nil {
		return nil, err
	}

	// Extract and parse JSON
	jsonStr := extractJSON(response)

	var workflow types.WorkflowSpec
	if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
		// Try wrapped format
		var wrapped struct {
			Workflow types.WorkflowSpec `json:"workflow"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &wrapped); err != nil {
			jsonStr = fixCommonJSONErrors(jsonStr)
			if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
				return nil, fmt.Errorf("failed to parse generated workflow: %w", err)
			}
		} else {
			workflow = wrapped.Workflow
		}
	}

	if workflow.Variables == nil {
		workflow.Variables = make(map[string]interface{})
	}

	if len(workflow.Steps) == 0 {
		return nil, fmt.Errorf("generated workflow has no steps")
	}

	return &workflow, nil
}

func extractWorkflowInputs(request string) map[string]interface{} {
	inputs := make(map[string]interface{})

	// Extract common parameters from the request
	requestLower := strings.ToLower(request)

	// Extract location
	cities := []string{"san francisco", "new york", "beijing", "shanghai", "tokyo"}
	for _, city := range cities {
		if strings.Contains(requestLower, city) {
			inputs["city"] = city
			break
		}
	}

	// Extract product names
	products := []string{"iphone", "macbook", "airpods", "ipad", "pixel", "galaxy"}
	for _, product := range products {
		if strings.Contains(requestLower, product) {
			inputs["product"] = product
			break
		}
	}

	// Extract URLs if present
	words := strings.Fields(request)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			inputs["url"] = word
			break
		}
	}

	return inputs
}

func findSavedFiles(result *types.ExecutionResult) []string {
	var files []string

	// Look for file paths in outputs
	for _, value := range result.Outputs {
		if str, ok := value.(string); ok {
			if strings.HasPrefix(str, "./") || strings.HasPrefix(str, "/") {
				if strings.Contains(str, ".") { // likely a file
					files = append(files, str)
				}
			}
		}
	}

	// Check step results for filesystem operations
	for _, stepResult := range result.StepResults {
		if stepResult.Status == "completed" {
			if path, ok := stepResult.Outputs["path"].(string); ok {
				files = append(files, path)
			}
		}
	}

	return files
}
