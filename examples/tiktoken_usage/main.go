package main

import (
	"fmt"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

func main() {
	// Create a new token counter with tiktoken support
	counter := usage.NewTokenCounter()
	
	// Example texts to count tokens for
	texts := []struct {
		model   string
		content string
	}{
		{
			model:   "gpt-4",
			content: "Hello, world! This is a test of tiktoken integration.",
		},
		{
			model:   "gpt-3.5-turbo",
			content: "The quick brown fox jumps over the lazy dog.",
		},
		{
			model:   "gpt-4o",
			content: `func main() {
	fmt.Println("Hello, World!")
}`,
		},
		{
			model:   "claude-3-opus",
			content: "This will use fallback estimation since tiktoken doesn't support Claude models directly.",
		},
	}
	
	fmt.Println("Token Counting with Tiktoken Integration")
	fmt.Println("=========================================")
	
	for _, example := range texts {
		tokens := counter.EstimateTokens(example.content, example.model)
		fmt.Printf("\nModel: %s\n", example.model)
		fmt.Printf("Text: %q\n", example.content)
		fmt.Printf("Tokens: %d\n", tokens)
	}
	
	// Example with messages (conversation)
	fmt.Println("\n\nConversation Token Counting")
	fmt.Println("============================")
	
	messages := []usage.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris. It's known for the Eiffel Tower, the Louvre Museum, and its rich history."},
	}
	
	conversationTokens := counter.EstimateMessagesTokens(messages, "gpt-4")
	fmt.Printf("\nTotal tokens in conversation (GPT-4): %d\n", conversationTokens)
	
	// Calculate cost estimation
	inputTokens := 50
	outputTokens := 100
	cost := usage.CalculateCost("gpt-4", inputTokens, outputTokens)
	fmt.Printf("\nCost Calculation Example:\n")
	fmt.Printf("Model: GPT-4\n")
	fmt.Printf("Input tokens: %d\n", inputTokens)
	fmt.Printf("Output tokens: %d\n", outputTokens)
	fmt.Printf("Estimated cost: $%.4f\n", cost)
}