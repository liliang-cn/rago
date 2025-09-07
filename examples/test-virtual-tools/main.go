package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	fmt.Println("🤖 RAGO Virtual Tools Test")
	fmt.Println("============================")
	fmt.Println("Testing tool definitions from mcpServers.json WITHOUT running actual servers\n")

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Create virtual tool definitions based on mcpServers.json
	// These are just SPECIFICATIONS - no servers need to run!
	virtualTools := []core.ToolInfo{
		{
			Name:        "filesystem_read",
			ServerName:  "filesystem",
			Description: "Read contents of a file",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to read",
					},
				},
			},
		},
		{
			Name:        "filesystem_list",
			ServerName:  "filesystem",
			Description: "List files in a directory",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"directory"},
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to list",
					},
				},
			},
		},
		{
			Name:        "fetch_url",
			ServerName:  "fetch",
			Description: "Fetch content from a URL",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"url"},
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to fetch",
					},
				},
			},
		},
		{
			Name:        "memory_set",
			ServerName:  "memory",
			Description: "Store a value in memory",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"key", "value"},
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key to store value under",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Value to store",
					},
				},
			},
		},
		{
			Name:        "get_current_time",
			ServerName:  "time",
			Description: "Get the current time",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "Timezone (optional)",
						"default":     "UTC",
					},
				},
			},
		},
	}

	// Test cases that would use these virtual tools
	testCases := []struct {
		prompt string
		desc   string
	}{
		{
			prompt: "List the files in the current directory",
			desc:   "Filesystem List",
		},
		{
			prompt: "What's the current time in New York?",
			desc:   "Time Tool",
		},
		{
			prompt: "Store the value 'hello world' with key 'greeting' in memory",
			desc:   "Memory Tool",
		},
		{
			prompt: "Fetch the content from https://example.com",
			desc:   "Fetch Tool",
		},
		{
			prompt: "Read the file README.md and tell me what time it is",
			desc:   "Multi-Tool",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📝 Test: %s\n", tc.desc)
		fmt.Printf("Prompt: %s\n", tc.prompt)
		fmt.Println(strings.Repeat("-", 60))

		// Create request with virtual tools
		req := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt:      tc.prompt,
				Temperature: 0.2,
				MaxTokens:   500,
			},
			Tools:      virtualTools,
			ToolChoice: "auto",
		}

		// Call LLM with virtual tools
		response, err := ragoClient.LLM().GenerateWithTools(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Check if LLM recognized and called the tools
		if len(response.ToolCalls) > 0 {
			fmt.Printf("✅ LLM recognized tools needed!\n")
			fmt.Printf("🔧 Tool Calls:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("  - Tool: %s\n", call.Name)
				params, _ := json.MarshalIndent(call.Parameters, "    ", "  ")
				fmt.Printf("    Parameters: %s\n", params)
				
				// Simulate what the tool would return
				result := simulateToolExecution(call.Name, call.Parameters)
				fmt.Printf("    Simulated Result: %s\n", result)
			}
		} else {
			fmt.Printf("ℹ️  No tools called\n")
			fmt.Printf("Response: %s\n", response.Content)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("💡 Key Points:")
	fmt.Println("- Tools are just DEFINITIONS passed to the LLM")
	fmt.Println("- No actual MCP servers need to be running")
	fmt.Println("- LLM recognizes when to use tools based on the prompt")
	fmt.Println("- Tool execution can be simulated or delegated to LLM")
	fmt.Println("\n✨ Virtual tools test completed!")
}

// Simulate tool execution for demonstration
func simulateToolExecution(toolName string, params map[string]interface{}) string {
	switch toolName {
	case "filesystem_list":
		dir, _ := params["directory"].(string)
		return fmt.Sprintf("[file1.txt, file2.md, folder/] in %s", dir)
	
	case "filesystem_read":
		path, _ := params["path"].(string)
		return fmt.Sprintf("Contents of %s: '# Example content'", path)
	
	case "get_current_time":
		tz, _ := params["timezone"].(string)
		if tz == "" {
			tz = "UTC"
		}
		return fmt.Sprintf("2024-01-15 14:30:00 %s", tz)
	
	case "memory_set":
		key, _ := params["key"].(string)
		value, _ := params["value"].(string)
		return fmt.Sprintf("Stored '%s' = '%s'", key, value)
	
	case "fetch_url":
		url, _ := params["url"].(string)
		return fmt.Sprintf("Fetched content from %s: '<html>...</html>'", url)
	
	default:
		return "Tool execution simulated"
	}
}