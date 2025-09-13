package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/client"
)

// Example struct for structured generation
type CodeReview struct {
	Language     string   `json:"language"`
	Issues       []Issue  `json:"issues"`
	Suggestions  []string `json:"suggestions"`
	OverallScore int      `json:"overall_score"`
}

type Issue struct {
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

func main() {
	// Create a new client
	ragClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragClient.Close()

	ctx := context.Background()

	// Example 1: Simple LLM generation
	fmt.Println("💬 Simple LLM Generation")
	fmt.Println("========================")
	
	simpleReq := client.LLMGenerateRequest{
		Prompt:      "Write a haiku about programming",
		Temperature: 0.9,
		MaxTokens:   100,
	}
	
	simpleResp, err := ragClient.LLMGenerate(ctx, simpleReq)
	if err != nil {
		log.Printf("Simple generation failed: %v", err)
	} else {
		fmt.Printf("🎋 Haiku:\n%s\n", simpleResp.Content)
	}

	// Example 2: Multi-turn chat
	fmt.Println("\n💬 Multi-turn Chat")
	fmt.Println("==================")
	
	chatReq := client.LLMChatRequest{
		Messages: []client.ChatMessage{
			{Role: "user", Content: "What is recursion in programming?"},
			{Role: "assistant", Content: "Recursion is a programming technique where a function calls itself to solve a problem by breaking it down into smaller subproblems."},
			{Role: "user", Content: "Can you give me a simple example?"},
		},
		Temperature: 0.7,
		MaxTokens:   300,
	}
	
	chatResp, err := ragClient.LLMChat(ctx, chatReq)
	if err != nil {
		log.Printf("Chat failed: %v", err)
	} else {
		fmt.Printf("🤖 Assistant:\n%s\n", chatResp.Content)
	}

	// Example 3: Streaming generation
	fmt.Println("\n🌊 Streaming Generation")
	fmt.Println("=======================")
	
	streamReq := client.LLMGenerateRequest{
		Prompt:      "Explain the benefits of Go programming in 3 points",
		Temperature: 0.5,
		MaxTokens:   200,
	}
	
	fmt.Print("🤖 Response: ")
	err = ragClient.LLMGenerateStream(ctx, streamReq, func(chunk string) {
		fmt.Print(chunk)
	})
	fmt.Println()
	if err != nil {
		log.Printf("Streaming failed: %v", err)
	}

	// Example 4: Structured generation (JSON output)
	fmt.Println("\n📊 Structured Generation")
	fmt.Println("========================")
	
	codeSnippet := `
func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`
	
	structuredReq := client.LLMStructuredRequest{
		Prompt: fmt.Sprintf("Review this Go code and provide a structured analysis:\n%s", codeSnippet),
		Schema: CodeReview{}, // Schema for the expected structure
		Temperature: 0.3,
		MaxTokens:   500,
	}
	
	structuredResp, err := ragClient.LLMGenerateStructured(ctx, structuredReq)
	if err != nil {
		log.Printf("Structured generation failed: %v", err)
	} else {
		fmt.Printf("📝 Structured Response (Valid: %v):\n", structuredResp.Valid)
		
		// Pretty print the JSON
		if structuredResp.Valid {
			var review CodeReview
			if err := json.Unmarshal([]byte(structuredResp.Raw), &review); err == nil {
				fmt.Printf("Language: %s\n", review.Language)
				fmt.Printf("Overall Score: %d/10\n", review.OverallScore)
				fmt.Printf("Issues Found: %d\n", len(review.Issues))
				for i, issue := range review.Issues {
					fmt.Printf("  [%d] Line %d (%s): %s\n", 
						i+1, issue.Line, issue.Severity, issue.Description)
				}
				fmt.Println("Suggestions:")
				for i, suggestion := range review.Suggestions {
					fmt.Printf("  %d. %s\n", i+1, suggestion)
				}
			}
		} else {
			fmt.Printf("Raw JSON: %s\n", structuredResp.Raw)
		}
	}

	// Example 5: Streaming chat with multiple turns
	fmt.Println("\n🌊 Streaming Chat")
	fmt.Println("=================")
	
	streamChatReq := client.LLMChatRequest{
		Messages: []client.ChatMessage{
			{Role: "user", Content: "Write a function to reverse a string in Go"},
		},
		Temperature: 0.5,
		MaxTokens:   300,
	}
	
	fmt.Print("🤖 Assistant (streaming): ")
	err = ragClient.LLMChatStream(ctx, streamChatReq, func(chunk string) {
		fmt.Print(chunk)
	})
	fmt.Println()
	if err != nil {
		log.Printf("Streaming chat failed: %v", err)
	}

	// Example 6: Different temperature settings
	fmt.Println("\n🌡️  Temperature Comparison")
	fmt.Println("==========================")
	
	prompt := "Write a one-line description of clouds"
	temperatures := []float64{0.1, 0.5, 0.9}
	
	for _, temp := range temperatures {
		req := client.LLMGenerateRequest{
			Prompt:      prompt,
			Temperature: temp,
			MaxTokens:   50,
		}
		
		resp, err := ragClient.LLMGenerate(ctx, req)
		if err != nil {
			log.Printf("Generation at temp %.1f failed: %v", temp, err)
			continue
		}
		fmt.Printf("Temperature %.1f: %s\n", temp, resp.Content)
	}

	// Example 7: Code generation
	fmt.Println("\n💻 Code Generation")
	fmt.Println("==================")
	
	codeGenReq := client.LLMGenerateRequest{
		Prompt: `Write a Go function that:
1. Takes a slice of integers
2. Returns the sum of even numbers
3. Includes error handling for nil slices
4. Has proper documentation`,
		Temperature: 0.2, // Low temperature for consistent code
		MaxTokens:   400,
	}
	
	codeResp, err := ragClient.LLMGenerate(ctx, codeGenReq)
	if err != nil {
		log.Printf("Code generation failed: %v", err)
	} else {
		fmt.Println("Generated Code:")
		fmt.Println(codeResp.Content)
	}

	// Example 8: Creative vs Factual generation
	fmt.Println("\n🎨 Creative vs Factual")
	fmt.Println("======================")
	
	// Factual (low temperature)
	factualReq := client.LLMGenerateRequest{
		Prompt:      "What is the capital of France?",
		Temperature: 0.1,
		MaxTokens:   50,
	}
	
	factualResp, err := ragClient.LLMGenerate(ctx, factualReq)
	if err == nil {
		fmt.Printf("Factual (temp=0.1): %s\n", factualResp.Content)
	}
	
	// Creative (high temperature)
	creativeReq := client.LLMGenerateRequest{
		Prompt:      "Invent a new programming language name and describe it",
		Temperature: 0.95,
		MaxTokens:   100,
	}
	
	creativeResp, err := ragClient.LLMGenerate(ctx, creativeReq)
	if err == nil {
		fmt.Printf("Creative (temp=0.95): %s\n", creativeResp.Content)
	}
}