// This example demonstrates using the unified RAGO client with all four pillars.
package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Create a unified client with all four pillars
	ragoClient, err := client.NewWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()
	
	ctx := context.Background()
	
	// Example 1: Simple LLM generation
	fmt.Println("=== Example 1: LLM Generation ===")
	generateExample(ctx, ragoClient)
	
	// Example 2: RAG-enhanced chat
	fmt.Println("\n=== Example 2: RAG-Enhanced Chat ===")
	ragChatExample(ctx, ragoClient)
	
	// Example 3: Document processing with multiple pillars
	fmt.Println("\n=== Example 3: Document Processing ===")
	documentProcessingExample(ctx, ragoClient)
	
	// Example 4: Task execution with agents
	fmt.Println("\n=== Example 4: Task Execution ===")
	taskExecutionExample(ctx, ragoClient)
	
	// Example 5: Health monitoring
	fmt.Println("\n=== Example 5: Health Monitoring ===")
	healthMonitoringExample(ragoClient)
}

func generateExample(ctx context.Context, client core.Client) {
	// Direct LLM generation
	req := core.GenerationRequest{
		Prompt:      "Explain the concept of artificial intelligence in simple terms.",
		Temperature: 0.7,
		MaxTokens:   200,
	}
	
	resp, err := client.LLM().Generate(ctx, req)
	if err != nil {
		log.Printf("Generation failed: %v", err)
		return
	}
	
	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Model used: %s\n", resp.Model)
	fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)
}

func ragChatExample(ctx context.Context, client core.Client) {
	// First, ingest some knowledge
	ingestReq := core.IngestRequest{
		DocumentID: "ai-basics",
		Content: `Artificial Intelligence (AI) refers to the simulation of human intelligence 
		in machines. Machine Learning is a subset of AI that enables systems to learn from data.
		Deep Learning is a subset of Machine Learning that uses neural networks with multiple layers.`,
		Metadata: map[string]interface{}{
			"source": "ai_tutorial",
			"type":   "educational",
		},
	}
	
	_, err := client.RAG().IngestDocument(ctx, ingestReq)
	if err != nil {
		log.Printf("Ingestion failed: %v", err)
		return
	}
	
	// Now chat with RAG context
	chatReq := core.ChatRequest{
		Message: "What is the relationship between AI, Machine Learning, and Deep Learning?",
		UseRAG:  true,
		Parameters: map[string]interface{}{
			"rag_limit":     5,
			"rag_threshold": float32(0.5),
			"temperature":   float32(0.6),
			"max_tokens":    300,
		},
	}
	
	chatResp, err := client.Chat(ctx, chatReq)
	if err != nil {
		log.Printf("Chat failed: %v", err)
		return
	}
	
	fmt.Printf("Chat Response: %s\n", chatResp.Response)
	fmt.Printf("Context items used: %d\n", len(chatResp.Context))
	for i, ctx := range chatResp.Context {
		content := ctx.Content
		if len(content) > 50 {
			content = content[:50]
		}
		fmt.Printf("  [%d] %s: %s\n", i+1, ctx.Role, content)
	}
}

func documentProcessingExample(ctx context.Context, client core.Client) {
	// Process a document using multiple pillars
	docReq := core.DocumentRequest{
		Action:     "analyze",
		DocumentID: "sample-doc",
		Content: `# Introduction to Go Programming
		
		Go is a statically typed, compiled programming language designed at Google.
		It is known for its simplicity, efficiency, and excellent support for concurrent programming.
		
		## Key Features
		- Simple syntax
		- Built-in concurrency with goroutines
		- Fast compilation
		- Garbage collection
		- Strong standard library`,
		Parameters: map[string]interface{}{
			"chunk_size":    200,
			"prompt":        "Summarize the key points of this document about Go programming",
			"entity_types":  []string{"programming_language", "feature", "company"},
		},
	}
	
	docResp, err := client.ProcessDocument(ctx, docReq)
	if err != nil {
		log.Printf("Document processing failed: %v", err)
		return
	}
	
	fmt.Printf("Document Action: %s\n", docResp.Action)
	fmt.Printf("Result: %s\n", docResp.Result)
	if docResp.Usage.TotalTokens > 0 {
		fmt.Printf("Tokens used: %d\n", docResp.Usage.TotalTokens)
	}
}

func taskExecutionExample(ctx context.Context, client core.Client) {
	// Execute a complex task using agents
	taskReq := core.TaskRequest{
		Task: "Analyze the performance characteristics of Go's goroutines versus traditional threads",
		Context: map[string]interface{}{
			"focus_areas": []string{"memory usage", "context switching", "scalability"},
			"depth":       "technical",
		},
	}
	
	taskResp, err := client.ExecuteTask(ctx, taskReq)
	if err != nil {
		log.Printf("Task execution failed: %v", err)
		return
	}
	
	fmt.Printf("Task: %s\n", taskResp.Task)
	fmt.Printf("Steps executed: %d\n", len(taskResp.Steps))
	for i, step := range taskResp.Steps {
		fmt.Printf("  Step %d: %s - Status: %s\n", i+1, step.StepID, step.Status)
	}
	fmt.Printf("Final Result: %s\n", taskResp.Result)
}

func healthMonitoringExample(client core.Client) {
	// Check the health of all pillars
	health := client.Health()
	
	fmt.Printf("Overall System Health: %s\n", health.Overall)
	fmt.Println("Pillar Health Status:")
	for pillar, status := range health.Pillars {
		fmt.Printf("  %s: %s\n", pillar, status)
	}
	
	if len(health.Providers) > 0 {
		fmt.Println("Provider Health Status:")
		for provider, status := range health.Providers {
			fmt.Printf("  %s: %s\n", provider, status)
		}
	}
	
	if len(health.Servers) > 0 {
		fmt.Println("MCP Server Health Status:")
		for server, status := range health.Servers {
			fmt.Printf("  %s: %s\n", server, status)
		}
	}
	
	// Display summary from details
	if summary, ok := health.Details["summary"].(map[string]interface{}); ok {
		fmt.Printf("\nHealth Summary:\n")
		fmt.Printf("  Healthy pillars: %v\n", summary["healthy_pillars"])
		fmt.Printf("  Degraded pillars: %v\n", summary["degraded_pillars"])
		fmt.Printf("  Unhealthy pillars: %v\n", summary["unhealthy_pillars"])
		fmt.Printf("  Total pillars: %v\n", summary["total_pillars"])
	}
}