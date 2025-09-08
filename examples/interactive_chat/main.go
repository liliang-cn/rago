package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Create a new rago client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Enable MCP if configured
	if err := ragoClient.EnableMCP(ctx); err != nil {
		fmt.Printf("Note: MCP not enabled: %v\n", err)
	}

	// Choose chat mode
	if len(os.Args) > 1 && os.Args[1] == "rag" {
		// Interactive RAG chat
		fmt.Println("Starting RAG-enhanced interactive chat...")
		
		opts := client.DefaultInteractiveChatOptions()
		opts.SystemPrompt = "You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately."
		opts.ShowToolCalls = true
		
		if err := ragoClient.InteractiveChatWithRAG(ctx, opts); err != nil {
			log.Fatalf("Interactive RAG chat failed: %v", err)
		}
	} else {
		// Regular interactive chat (with MCP tools if available)
		fmt.Println("Starting interactive chat...")
		
		opts := client.DefaultInteractiveChatOptions()
		opts.ShowToolCalls = true
		opts.ShowThinking = false
		
		if err := ragoClient.InteractiveChat(ctx, opts); err != nil {
			log.Fatalf("Interactive chat failed: %v", err)
		}
	}
}