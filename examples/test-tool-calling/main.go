package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	fmt.Println("🤖 RAGO Tool Calling Test")
	fmt.Println("==========================\n")

	// Use RAGO client, not direct Ollama client!
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Define some test tools in the proper format
	tools := []core.ToolInfo{
		{
			Name:        "get_temperature",
			Description: "Get the temperature for a city in Celsius",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"city"},
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "The name of the city",
					},
				},
			},
		},
		{
			Name:        "get_current_time",
			Description: "Get the current time in a specific timezone",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"timezone"},
				"properties": map[string]interface{}{
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "The timezone (e.g., 'UTC', 'America/New_York')",
					},
				},
			},
		},
		{
			Name:        "calculate",
			Description: "Perform a mathematical calculation",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"expression"},
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "The mathematical expression to evaluate",
					},
				},
			},
		},
	}

	// Test prompts that should trigger tool use
	testCases := []struct {
		prompt string
		desc   string
	}{
		{
			prompt: "What's the temperature in Paris?",
			desc:   "Weather Query",
		},
		{
			prompt: "What time is it in Tokyo?",
			desc:   "Time Query",
		},
		{
			prompt: "Calculate 25 * 4 + 10",
			desc:   "Math Calculation",
		},
		{
			prompt: "What's the weather like in London and what time is it there?",
			desc:   "Multi-Tool Query",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📝 Test: %s\n", tc.desc)
		fmt.Printf("Prompt: %s\n", tc.prompt)
		fmt.Println(strings.Repeat("-", 50))

		// Create tool generation request
		req := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt:      tc.prompt,
				Temperature: 0.3,
				MaxTokens:   500,
			},
			Tools:      tools,
			ToolChoice: "auto", // Let the model decide
		}

		// Call LLM with tools
		response, err := ragoClient.LLM().GenerateWithTools(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Display response
		fmt.Printf("Response: %s\n", response.Content)

		// Check for tool calls
		if len(response.ToolCalls) > 0 {
			fmt.Printf("\n🔧 Tool Calls:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("  - Tool: %s\n", call.Name)
				fmt.Printf("    Args: %v\n", call.Parameters)
				
				// Simulate tool execution
				result := executeToolSimulation(call.Name, call.Parameters)
				fmt.Printf("    Result: %s\n", result)
			}
		} else {
			fmt.Printf("ℹ️  No tools were called\n")
		}
	}

	// Test with streaming
	fmt.Printf("\n\n🌊 Streaming Test\n")
	fmt.Println(strings.Repeat("=", 50))

	streamReq := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:      "What's the temperature in New York and what time is it there?",
			Temperature: 0.3,
			MaxTokens:   500,
		},
		Tools:      tools,
		ToolChoice: "auto",
	}

	fmt.Print("Response: ")
	var toolCalls []core.ToolCall
	
	err = ragoClient.LLM().StreamWithTools(ctx, streamReq, func(chunk core.ToolStreamChunk) error {
		if chunk.Content != "" {
			fmt.Print(chunk.Delta)
		}
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
		if chunk.Finished {
			fmt.Println()
		}
		return nil
	})

	if err != nil {
		fmt.Printf("\n❌ Streaming error: %v\n", err)
	}

	if len(toolCalls) > 0 {
		fmt.Printf("\n🔧 Streamed Tool Calls:\n")
		for _, call := range toolCalls {
			fmt.Printf("  - Tool: %s (Args: %v)\n", call.Name, call.Parameters)
		}
	}

	fmt.Println("\n✨ Test completed!")
}

// Simulate tool execution for demonstration
func executeToolSimulation(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "get_temperature":
		city, _ := args["city"].(string)
		return fmt.Sprintf("22°C in %s", city)
	case "get_current_time":
		tz, _ := args["timezone"].(string)
		return fmt.Sprintf("14:30 in %s", tz)
	case "calculate":
		expr, _ := args["expression"].(string)
		return fmt.Sprintf("Result of %s = 110", expr)
	default:
		return "Unknown tool"
	}
}