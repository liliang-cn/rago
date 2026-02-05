// Package main shows how to use the rago gateway library in your Go program
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/gateway"
	"github.com/liliang-cn/rago/v2/pkg/services"
)

func main() {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Initialize global pool (required for LLM/Embedding services)
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(ctx, cfg); err != nil {
		log.Fatal(err)
	}

	// 3. Create gateway with response callback
	gw, err := gateway.New(ctx, cfg,
		gateway.WithResponseCallback(func(resp *gateway.Response) {
			// This is called whenever any agent responds
			if resp.Error != nil {
				log.Printf("[%s] Error: %v", resp.AgentName, resp.Error)
			} else {
				log.Printf("[%s] %s (took %v)", resp.AgentName, resp.Content, resp.Duration)
			}
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer gw.Close()

	// 4. Create agents (each runs in its own goroutine)
	assistant, _ := gw.CreateAgent("assistant", "You are a helpful assistant.")
	coder, _ := gw.CreateAgent("coder", "You are a code expert. Be concise.")
	writer, _ := gw.CreateAgent("writer", "You are a creative writer.")

	fmt.Println("Created 3 agents:", assistant.Name(), coder.Name(), writer.Name())

	// 5. Set current agent
	gw.SetCurrent("assistant")

	// 6. Example 1: Simple query (async)
	fmt.Println("\n--- Example 1: Async Query ---")
	respCh, err := gw.Query(ctx, "What is 2+2?")
	if err != nil {
		log.Fatal(err)
	}
	resp := <-respCh
	fmt.Printf("Response: %s\n", resp.Content)

	// 7. Example 2: Direct agent query (async)
	fmt.Println("\n--- Example 2: Query Specific Agent ---")
	respCh2, _ := coder.Query(ctx, "Write a Go hello world in one line")
	// Do other work while agent processes...
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Doing other work while coder thinks...")
	resp2 := <-respCh2
	fmt.Printf("Coder says: %s\n", resp2.Content)

	// 8. Example 3: Blocking query with Ask()
	fmt.Println("\n--- Example 3: Blocking Query ---")
	resp3, err := writer.Ask(ctx, "Write a haiku about code")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Writer says:\n%s\n", resp3.Content)

	// 9. Example 4: Concurrent queries to multiple agents
	fmt.Println("\n--- Example 4: Concurrent Queries ---")

	// Query all agents at once
	assistantCh, _ := assistant.Query(ctx, "Say 'A' and nothing else")
	coderCh, _ := coder.Query(ctx, "Say 'B' and nothing else")
	writerCh, _ := writer.Query(ctx, "Say 'C' and nothing else")

	// Collect responses
	responses := make([]*gateway.Response, 0)
	responses = append(responses, <-assistantCh)
	responses = append(responses, <-coderCh)
	responses = append(responses, <-writerCh)

	for _, r := range responses {
		fmt.Printf("[%s]: %s\n", r.AgentName, r.Content)
	}

	// 10. Example 5: Check agent status
	fmt.Println("\n--- Example 5: Agent Status ---")
	for _, agent := range gw.ListAgents() {
		status := "idle"
		if agent.IsBusy() {
			status = "busy"
		}
		fmt.Printf("%s: %s\n", agent.Name(), status)
	}
}
