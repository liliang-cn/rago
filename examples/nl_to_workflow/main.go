package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/agents/execution"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// NaturalLanguageToWorkflowExample demonstrates converting natural language to workflows
func main() {
	fmt.Println("🤖 RAGO Library Usage - Natural Language to Workflow")
	fmt.Println("====================================================")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize providers
	ctx := context.Background()
	_, llmService, _, err := utils.InitializeProviders(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}

	// Example natural language requests
	requests := []string{
		"Check the current time and tell me if it's morning, afternoon, or evening",
		"Get information about the golang/go GitHub repository",
		"Fetch the latest news headlines and summarize them",
	}

	fmt.Println("📝 Natural Language Requests:")
	for i, req := range requests {
		fmt.Printf("   %d. %s\n", i+1, req)
	}
	fmt.Println()

	// Process the first request as an example
	request := requests[0]
	fmt.Printf("⚡ Processing: \"%s\"\n\n", request)

	// Generate workflow from natural language
	workflow, err := generateWorkflowFromNL(ctx, llmService, request)
	if err != nil {
		log.Fatalf("Failed to generate workflow: %v", err)
	}

	// Display generated workflow
	fmt.Printf("📋 Generated Workflow:\n")
	fmt.Printf("   Steps: %d\n", len(workflow.Steps))
	for i, step := range workflow.Steps {
		fmt.Printf("   %d. %s (%s)\n", i+1, step.Name, step.Tool)
	}
	fmt.Println()

	// Execute the generated workflow
	executor := execution.NewWorkflowExecutor(cfg, llmService)
	executor.SetVerbose(true)

	fmt.Println("⚙️  Executing generated workflow...")
	result, err := executor.Execute(ctx, workflow)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Display results
	fmt.Println("\n✅ Execution completed!")
	fmt.Printf("   Duration: %v\n", result.Duration)
	
	// Show key outputs
	fmt.Println("\n📊 Results:")
	for key, value := range result.Outputs {
		if str, ok := value.(string); ok && len(str) > 100 {
			fmt.Printf("   %s: %s...\n", key, str[:100])
		} else {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}
}

// generateWorkflowFromNL converts natural language to a workflow
func generateWorkflowFromNL(ctx context.Context, llm domain.Generator, request string) (*types.WorkflowSpec, error) {
	systemPrompt := `You are a workflow generator. Generate valid workflow JSON based on user requests.

Available tools:
1. filesystem - File operations (read, write, list, execute, move, copy, delete, mkdir)
2. fetch - HTTP/HTTPS requests for APIs and websites
3. memory - Temporary storage (store, retrieve, delete, append)
4. time - Date/time operations (now, format, parse)
5. sequential-thinking - LLM analysis and reasoning

IMPORTANT: Data flows between steps using variables:
- When a step produces output, store it in a variable using "outputs"
- When a step needs that data, reference it using {{variableName}} in "inputs"

Return ONLY valid JSON in this format:
{
  "steps": [
    {
      "id": "step1",
      "name": "Step Name",
      "type": "tool",
      "tool": "tool_name",
      "inputs": {
        "key": "value"
      },
      "outputs": {
        "outputKey": "variableName"
      }
    }
  ]
}`

	userPrompt := fmt.Sprintf(`User request: "%s"

Generate a workflow to accomplish this request. Return ONLY the JSON workflow.`, request)

	fullPrompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)
	
	opts := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	}
	
	response, err := llm.Generate(ctx, fullPrompt, opts)
	if err != nil {
		return nil, err
	}
	
	// Extract JSON from response
	jsonStr := extractJSON(response)
	
	// Parse workflow
	var workflow types.WorkflowSpec
	if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}
	
	// Initialize variables if needed
	if workflow.Variables == nil {
		workflow.Variables = make(map[string]interface{})
	}
	
	return &workflow, nil
}

// extractJSON extracts JSON from LLM response
func extractJSON(text string) string {
	// Try to find JSON between ```json and ``` markers
	start := strings.Index(text, "```json")
	if start != -1 {
		start += 7 // Skip past ```json
		end := strings.Index(text[start:], "```")
		if end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	
	// Try to find JSON between ``` and ``` markers
	start = strings.Index(text, "```")
	if start != -1 {
		start += 3
		end := strings.Index(text[start:], "```")
		if end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	
	// Try to find JSON starting with { and ending with }
	start = strings.Index(text, "{")
	if start != -1 {
		end := strings.LastIndex(text, "}")
		if end != -1 && end > start {
			return strings.TrimSpace(text[start : end+1])
		}
	}
	
	// Return the whole text as last resort
	return strings.TrimSpace(text)
}