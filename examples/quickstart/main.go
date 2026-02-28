// Package main demonstrates the simplest way to get started with RAGO.
//
// RAGO is a Retrieval-Augmented Generation system with built-in:
// - Document ingestion and semantic search
// - Multi-provider LLM support (OpenAI, Ollama, etc.)
// - MCP tools integration
// - Agent automation with PTC (Programmatic Tool Calling)
//
// Usage:
//
//	go run examples/quickstart/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== RAGO Quickstart ===")
	fmt.Println()

	// Create an agent - that's it! Configuration is loaded from ~/.rago/config/rago.toml
	svc, err := agent.NewBuilder("quickstart").Build()
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	// Ask a question
	fmt.Println("Q: What can you help me with?")
	fmt.Print("A: ")

	result, err := svc.Run(ctx, "What can you help me with? Give a brief answer.")
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("%v\n", result.FinalResult)
	fmt.Println()
	fmt.Println("✅ Done! Explore more examples in the examples/ directory.")
}
