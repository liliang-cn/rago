package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Example 1: Ask a question that requires tool use
	fmt.Println("Example 1: Time Query")
	fmt.Println("--------------------")
	
	response, err := ragoClient.LLM().GenerateWithTools(ctx, core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "What is the current time and date?",
		},
		ToolChoice: "auto",
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response.Content)
		if len(response.ToolCalls) > 0 {
			fmt.Printf("Tools used: ")
			for _, tc := range response.ToolCalls {
				fmt.Printf("%s ", tc.Name)
			}
			fmt.Println()
		}
	}

	// Example 2: File operations
	fmt.Println("\nExample 2: File Operations")
	fmt.Println("--------------------------")
	
	response, err = ragoClient.LLM().GenerateWithTools(ctx, core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "List the files in the current directory",
		},
		ToolChoice: "auto",
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response.Content)
		if len(response.ToolCalls) > 0 {
			fmt.Printf("Tools used: ")
			for _, tc := range response.ToolCalls {
				fmt.Printf("%s ", tc.Name)
			}
			fmt.Println()
		}
	}

	// Example 3: Web fetch
	fmt.Println("\nExample 3: Web Content")
	fmt.Println("----------------------")
	
	response, err = ragoClient.LLM().GenerateWithTools(ctx, core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "What is the weather like in San Francisco? (Note: use available tools to get current information)",
		},
		ToolChoice: "auto",
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response.Content)
		if len(response.ToolCalls) > 0 {
			fmt.Printf("Tools used: ")
			for _, tc := range response.ToolCalls {
				fmt.Printf("%s ", tc.Name)
			}
			fmt.Println()
		}
	}

	fmt.Println("\nDemo completed!")
}