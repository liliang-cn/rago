// Package main shows how to use the rago agent library in your Go program
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// That's it! Just create an agent with minimal config
	svc, err := agent.New(&agent.AgentConfig{
		Name: "my-agent",
		// Everything else uses defaults from rago.toml
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	// Simple query
	fmt.Println("--- Example 1: Simple Query ---")
	result, err := svc.Run(ctx, "What is 2+2?")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response: %v\n", result.FinalResult)

	// Chat with session memory
	fmt.Println("\n--- Example 2: Chat with Memory ---")
	result2, _ := svc.RunWithSession(ctx, "My name is Alice", "session-123")
	fmt.Printf("Alice: %v\n", result2.FinalResult)

	result3, _ := svc.RunWithSession(ctx, "What's my name?", "session-123")
	fmt.Printf("Memory: %v\n", result3.FinalResult)

	// With options
	fmt.Println("\n--- Example 3: With Options ---")
	advanced, _ := agent.New(&agent.AgentConfig{
		Name:         "coder-agent",
		SystemPrompt: "You are a code expert. Be concise.",
		EnableMCP:     true,  // Enable MCP tools
		EnableMemory:  true,  // Enable long-term memory
		EnableRouter:  true,  // Enable semantic routing
		ProgressCb: func(event agent.ProgressEvent) {
			fmt.Printf("[%s] %s\n", event.Type, event.Message)
		},
	})
	defer advanced.Close()

	result4, _ := advanced.Run(ctx, "Write a Go hello world in one line")
	fmt.Printf("Coder: %v\n", result4.FinalResult)

	// List sessions
	fmt.Println("\n--- Example 4: List Sessions ---")
	sessions, _ := svc.ListSessions(10)
	fmt.Printf("Found %d sessions\n", len(sessions))
}
