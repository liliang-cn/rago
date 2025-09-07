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
	fmt.Println("🤖 RAGO LLM Tool Calling Integration Test")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("This test demonstrates proper LLM tool calling with MCP integration")
	fmt.Println()

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Test 1: Basic tool discovery
	fmt.Println("📋 Test 1: Tool Discovery")
	fmt.Println("-" + strings.Repeat("-", 40))
	testToolDiscovery(ctx, ragoClient)

	// Test 2: Simple tool calling
	fmt.Println("\n🔧 Test 2: Simple Tool Calling")
	fmt.Println("-" + strings.Repeat("-", 40))
	testSimpleToolCall(ctx, ragoClient)

	// Test 3: Tool calling with execution
	fmt.Println("\n⚡ Test 3: Tool Calling with Automatic Execution")
	fmt.Println("-" + strings.Repeat("-", 40))
	testToolCallWithExecution(ctx, ragoClient)

	// Test 4: Multiple tool calls
	fmt.Println("\n🔗 Test 4: Multiple Tool Calls")
	fmt.Println("-" + strings.Repeat("-", 40))
	testMultipleToolCalls(ctx, ragoClient)

	// Test 5: Streaming with tools
	fmt.Println("\n📡 Test 5: Streaming with Tool Support")
	fmt.Println("-" + strings.Repeat("-", 40))
	testStreamingWithTools(ctx, ragoClient)

	fmt.Println("\n✅ All tests completed!")
}

func testToolDiscovery(ctx context.Context, client *client.Client) {
	// Check available MCP servers
	servers := client.MCP().ListServers()
	fmt.Printf("Found %d MCP servers:\n", len(servers))
	for _, server := range servers {
		fmt.Printf("  - %s: %s (Status: %s)\n", server.Name, server.Description, server.Status)
	}

	// Check available tools
	tools := client.MCP().ListTools()
	fmt.Printf("\nFound %d tools:\n", len(tools))
	for i, tool := range tools {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(tools)-5)
			break
		}
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
}

func testSimpleToolCall(ctx context.Context, client *client.Client) {
	// Get available tools
	tools := client.MCP().ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available, skipping test")
		return
	}

	// Create a tool generation request
	req := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:      "What is the current time? Please use the time tool if available.",
			Temperature: 0.7,
			MaxTokens:   200,
		},
		Tools:      tools[:min(3, len(tools))], // Limit to 3 tools for testing
		ToolChoice: "auto",
	}

	fmt.Println("Sending prompt: \"What is the current time?\"")
	fmt.Println("Available tools provided to LLM:")
	for _, tool := range req.Tools {
		fmt.Printf("  - %s\n", tool.Name)
	}

	// Call LLM with tools
	response, err := client.LLM().GenerateWithTools(ctx, req)
	if err != nil {
		fmt.Printf("Error generating with tools: %v\n", err)
		return
	}

	fmt.Printf("\nLLM Response: %s\n", response.Content)

	if len(response.ToolCalls) > 0 {
		fmt.Printf("\nTool calls made by LLM:\n")
		for _, call := range response.ToolCalls {
			fmt.Printf("  - Tool: %s (ID: %s)\n", call.Name, call.ID)
			if len(call.Parameters) > 0 {
				params, _ := json.MarshalIndent(call.Parameters, "    ", "  ")
				fmt.Printf("    Parameters: %s\n", string(params))
			}
		}
	} else {
		fmt.Println("\nNo tool calls were made by the LLM")
	}
}

func testToolCallWithExecution(ctx context.Context, client *client.Client) {
	// Get LLM service and check if it supports tool execution
	llmService := client.LLM()
	
	// Check if the service has GenerateWithToolExecution method
	if executor, ok := llmService.(interface {
		GenerateWithToolExecution(context.Context, core.ToolGenerationRequest) (*core.ToolGenerationResponse, error)
	}); ok {
		// Get available tools
		tools := client.MCP().ListTools()
		if len(tools) == 0 {
			fmt.Println("No tools available, skipping test")
			return
		}

		req := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt:      "Create a file called test_tool_call.txt with the content 'Tool calling works!' and then read it back to confirm.",
				Temperature: 0.7,
				MaxTokens:   500,
			},
			Tools:      tools,
			ToolChoice: "auto",
		}

		fmt.Println("Testing automatic tool execution...")
		fmt.Println("Prompt: Create and read a test file")

		response, err := executor.GenerateWithToolExecution(ctx, req)
		if err != nil {
			fmt.Printf("Error with tool execution: %v\n", err)
			return
		}

		fmt.Printf("\nFinal Response: %s\n", response.Content)
		
		if len(response.ToolCalls) > 0 {
			fmt.Printf("\nTools that were executed:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("  - %s\n", call.Name)
			}
		}
	} else {
		fmt.Println("Tool execution not supported in current configuration")
	}
}

func testMultipleToolCalls(ctx context.Context, client *client.Client) {
	// Get available tools
	tools := client.MCP().ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available, skipping test")
		return
	}

	req := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "Please do the following: 1) Get the current time, 2) List files in the current directory, 3) Get system information. Use appropriate tools for each task.",
			Temperature: 0.7,
			MaxTokens:   500,
		},
		Tools:        tools,
		ToolChoice:   "auto",
		MaxToolCalls: 5, // Allow multiple tool calls
	}

	fmt.Println("Testing multiple tool calls in one request...")

	response, err := client.LLM().GenerateWithTools(ctx, req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nLLM Response: %s\n", response.Content)
	fmt.Printf("\nNumber of tool calls: %d\n", len(response.ToolCalls))

	for i, call := range response.ToolCalls {
		fmt.Printf("\nTool Call %d:\n", i+1)
		fmt.Printf("  Name: %s\n", call.Name)
		fmt.Printf("  ID: %s\n", call.ID)
		
		// Execute the tool call
		toolReq := core.ToolCallRequest{
			ToolName:  call.Name,
			Arguments: call.Parameters,
		}
		
		result, err := client.MCP().CallTool(ctx, toolReq)
		if err != nil {
			fmt.Printf("  Error executing: %v\n", err)
		} else {
			fmt.Printf("  Success: %v\n", result.Success)
			if result.Result != nil {
				resultStr, _ := json.MarshalIndent(result.Result, "  ", "  ")
				fmt.Printf("  Result: %s\n", string(resultStr))
			}
		}
	}
}

func testStreamingWithTools(ctx context.Context, client *client.Client) {
	// Get available tools
	tools := client.MCP().ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available, skipping test")
		return
	}

	req := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:      "Tell me about the current time and explain what time zones are.",
			Temperature: 0.7,
			MaxTokens:   300,
		},
		Tools:      tools[:min(3, len(tools))],
		ToolChoice: "auto",
	}

	fmt.Println("Testing streaming with tool support...")
	fmt.Print("\nStreaming response: ")

	var toolCalls []core.ToolCall
	err := client.LLM().StreamWithTools(ctx, req, func(chunk core.ToolStreamChunk) error {
		// Print streaming content
		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
		}
		
		// Collect tool calls
		if len(chunk.ToolCalls) > 0 {
			toolCalls = chunk.ToolCalls
		}
		
		return nil
	})

	fmt.Println() // New line after streaming

	if err != nil {
		fmt.Printf("\nError during streaming: %v\n", err)
		return
	}

	if len(toolCalls) > 0 {
		fmt.Printf("\nTool calls made during streaming:\n")
		for _, call := range toolCalls {
			fmt.Printf("  - %s (ID: %s)\n", call.Name, call.ID)
		}
	} else {
		fmt.Println("\nNo tool calls were made during streaming")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}