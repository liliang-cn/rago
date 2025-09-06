package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
)

func main() {
	fmt.Println("RAGO LLM Pillar Example")
	fmt.Println("======================")

	// Create LLM configuration
	config := core.LLMConfig{
		DefaultProvider: "ollama-llama2",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   true,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: map[string]core.ProviderConfig{
			"ollama-llama2": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
				Weight:  10,
				Timeout: 30 * time.Second,
			},
			"openai-gpt3": {
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "your-api-key-here", // Replace with actual API key
				Model:   "gpt-3.5-turbo",
				Weight:  5,
				Timeout: 30 * time.Second,
			},
		},
	}

	// Create LLM service
	fmt.Println("Creating LLM service...")
	llmService, err := llm.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create LLM service: %v", err)
	}
	defer llmService.Close()

	// List configured providers
	fmt.Println("\nConfigured providers:")
	providers := llmService.ListProviders()
	for _, provider := range providers {
		fmt.Printf("- %s (%s) - Model: %s, Weight: %d, Health: %s\n",
			provider.Name, provider.Type, provider.Model, provider.Weight, provider.Health)
	}

	// Check provider health
	fmt.Println("\nProvider health status:")
	healthStatus := llmService.GetProviderHealth()
	for name, status := range healthStatus {
		fmt.Printf("- %s: %s\n", name, status)
	}

	// Get service metrics
	fmt.Println("\nService metrics:")
	metrics := llmService.GetMetrics()
	fmt.Printf("- Total requests: %d\n", metrics.TotalRequests)
	fmt.Printf("- Success rate: %.2f%%\n", metrics.SuccessRate*100)
	fmt.Printf("- Average latency: %v\n", metrics.AverageLatency)
	fmt.Printf("- Service uptime: %v\n", metrics.ServiceUptime)
	fmt.Printf("- Requests per second: %.2f\n", metrics.RequestsPerSecond)

	// Example 1: Basic text generation
	fmt.Println("\n=== Example 1: Basic Text Generation ===")
	ctx := context.Background()
	_ = ctx // Used in commented out examples

	request := core.GenerationRequest{
		Prompt:      "Explain the concept of load balancing in simple terms.",
		MaxTokens:   200,
		Temperature: 0.7,
	}

	fmt.Printf("Prompt: %s\n", request.Prompt)
	fmt.Println("Generating response...")

	// NOTE: This will fail if no providers are actually running
	// Uncomment the following lines to test with real providers
	/*
	response, err := llmService.Generate(ctx, request)
	if err != nil {
		log.Printf("Generation failed: %v", err)
	} else {
		fmt.Printf("Response: %s\n", response.Content)
		fmt.Printf("Provider: %s\n", response.Provider)
		fmt.Printf("Duration: %v\n", response.Duration)
		fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
	}
	*/

	// Example 2: Streaming generation
	fmt.Println("\n=== Example 2: Streaming Text Generation ===")
	
	streamRequest := core.GenerationRequest{
		Prompt:      "Write a short story about artificial intelligence.",
		MaxTokens:   300,
		Temperature: 0.8,
	}

	fmt.Printf("Prompt: %s\n", streamRequest.Prompt)
	fmt.Println("Streaming response...")

	// NOTE: This will fail if no providers are actually running
	// Uncomment the following lines to test with real providers
	/*
	err = llmService.Stream(ctx, streamRequest, func(chunk core.StreamChunk) error {
		fmt.Print(chunk.Delta)
		if chunk.Finished {
			fmt.Println("\n[Stream completed]")
		}
		return nil
	})
	
	if err != nil {
		log.Printf("Streaming failed: %v", err)
	}
	*/

	// Example 3: Batch generation
	fmt.Println("\n=== Example 3: Batch Text Generation ===")
	
	batchRequests := []core.GenerationRequest{
		{
			Prompt:      "What is machine learning?",
			MaxTokens:   100,
			Temperature: 0.5,
		},
		{
			Prompt:      "What is deep learning?",
			MaxTokens:   100,
			Temperature: 0.5,
		},
		{
			Prompt:      "What is neural networks?",
			MaxTokens:   100,
			Temperature: 0.5,
		},
	}

	fmt.Printf("Processing %d requests in batch...\n", len(batchRequests))

	// NOTE: This will fail if no providers are actually running
	// Uncomment the following lines to test with real providers
	/*
	batchResponses, err := llmService.GenerateBatch(ctx, batchRequests)
	if err != nil {
		log.Printf("Batch generation failed: %v", err)
	} else {
		fmt.Printf("Successfully generated %d responses\n", len(batchResponses))
		for i, response := range batchResponses {
			fmt.Printf("\nResponse %d:\n", i+1)
			fmt.Printf("Provider: %s\n", response.Provider)
			fmt.Printf("Content: %s\n", response.Content[:min(100, len(response.Content))])
			if len(response.Content) > 100 {
				fmt.Println("...")
			}
		}
	}
	*/

	// Example 4: Dynamic provider management
	fmt.Println("\n=== Example 4: Dynamic Provider Management ===")
	
	// Add a new provider dynamically
	newProviderConfig := core.ProviderConfig{
		Type:    "lmstudio",
		BaseURL: "http://localhost:1234/v1",
		Model:   "local-model",
		Weight:  7,
		Timeout: 45 * time.Second,
	}

	fmt.Println("Adding new LMStudio provider...")
	err = llmService.AddProvider("lmstudio-local", newProviderConfig)
	if err != nil {
		log.Printf("Failed to add provider: %v", err)
	} else {
		fmt.Println("Provider added successfully!")
	}

	// List providers again
	fmt.Println("\nUpdated provider list:")
	providers = llmService.ListProviders()
	for _, provider := range providers {
		fmt.Printf("- %s (%s) - Model: %s, Weight: %d\n",
			provider.Name, provider.Type, provider.Model, provider.Weight)
	}

	// Remove the provider
	fmt.Println("\nRemoving LMStudio provider...")
	err = llmService.RemoveProvider("lmstudio-local")
	if err != nil {
		log.Printf("Failed to remove provider: %v", err)
	} else {
		fmt.Println("Provider removed successfully!")
	}

	// Final metrics
	fmt.Println("\n=== Final Service Metrics ===")
	finalMetrics := llmService.GetMetrics()
	fmt.Printf("Total requests processed: %d\n", finalMetrics.TotalRequests)
	fmt.Printf("Overall success rate: %.2f%%\n", finalMetrics.SuccessRate*100)
	fmt.Printf("Service uptime: %v\n", finalMetrics.ServiceUptime)

	fmt.Println("\nExample completed successfully!")
	fmt.Println("\nNOTE: To test actual generation, uncomment the generation code")
	fmt.Println("and ensure you have Ollama or OpenAI configured and running.")
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}