// Package main shows how to use the rago agent library in your Go programs
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// ========================================
	// Example 1: Simple Agent (Minimal Setup)
	// ========================================
	simpleAgent(ctx)

	// ========================================
	// Example 2: Agent with Session Memory
	// ========================================
	sessionAgent(ctx)

	// ========================================
	// Example 3: Advanced Agent with All Features
	// ========================================
	advancedAgent(ctx)

	// ========================================
	// Example 4: Planning Without Execution
	// ========================================
	planningExample(ctx)
}

// simpleAgent demonstrates the simplest way to create an agent
func simpleAgent(ctx context.Context) {
	fmt.Println("\n========== Example 1: Simple Agent ==========")

	svc, err := agent.New(&agent.AgentConfig{
		Name: "simple",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	result, err := svc.Run(ctx, "What is 2+2? Answer in one word.")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Question: What is 2+2?\n")
	fmt.Printf("Answer: %v\n", result.FinalResult)
}

// sessionAgent demonstrates conversation with memory
func sessionAgent(ctx context.Context) {
	fmt.Println("\n========== Example 2: Session Memory ==========")

	svc, err := agent.New(&agent.AgentConfig{
		Name: "assistant",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	sessionID := fmt.Sprintf("demo-%d", time.Now().Unix())

	// First message
	fmt.Println("\n[You:] My name is Alice and I love Go programming.")
	result1, _ := svc.RunWithSession(ctx, "My name is Alice and I love Go programming.", sessionID)
	fmt.Printf("[Agent]: %v\n", result1.FinalResult)

	// Second message - agent remembers
	fmt.Println("\n[You:] What's my name?")
	result2, _ := svc.RunWithSession(ctx, "What's my name?", sessionID)
	fmt.Printf("[Agent]: %v\n", result2.FinalResult)

	// Third message
	fmt.Println("\n[You:] What programming language do I like?")
	result3, _ := svc.RunWithSession(ctx, "What programming language do I like?", sessionID)
	fmt.Printf("[Agent]: %v\n", result3.FinalResult)
}

// advancedAgent demonstrates all available features
func advancedAgent(ctx context.Context) {
	fmt.Println("\n========== Example 3: Advanced Agent ==========")

	svc, err := agent.New(&agent.AgentConfig{
		Name:         "coder",
		SystemPrompt: "You are a code expert. Be concise and helpful.",
		EnableMCP:     true,
		EnableMemory:  true,
		EnableRouter:  true,
		ProgressCb:    progressCallback,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	fmt.Println("\n[You:] Write a Go function to calculate fibonacci(5)")
	result, _ := svc.Run(ctx, "Write a Go function to calculate fibonacci(5). Keep it short.")
	fmt.Printf("[Agent]: %v\n", result.FinalResult)

	// List sessions
	sessions, _ := svc.ListSessions(10)
	fmt.Printf("\nTotal sessions: %d\n", len(sessions))
}

// planningExample shows planning without execution
func planningExample(ctx context.Context) {
	fmt.Println("\n========== Example 4: Planning ==========")

	svc, err := agent.New(&agent.AgentConfig{
		Name: "planner",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	fmt.Println("\n[You:] Plan: Create a REST API in Go")
	plan, err := svc.Plan(ctx, "Plan the steps to create a REST API in Go")
	if err != nil {
		log.Printf("Planning error: %v\n", err)
		return
	}

	fmt.Printf("Plan ID: %s\n", plan.ID)
	fmt.Printf("Goal: %s\n\n", plan.Goal)
	fmt.Println("Steps:")
	for i, step := range plan.Steps {
		status := ""
		if step.Status == "completed" {
			status = " ✓"
		}
		fmt.Printf("  %d. %s%s\n", i+1, step.Description, status)
	}
}

// progressCallback prints agent execution events
func progressCallback(event agent.ProgressEvent) {
	switch event.Type {
	case "thinking":
		fmt.Printf("  🤔 %s\n", event.Message)
	case "tool_call":
		fmt.Printf("  🔧 Using tool: %s\n", event.Tool)
	case "tool_result":
		fmt.Printf("  ✅ Tool completed\n")
	case "done":
		fmt.Printf("  ✨ Done (%d steps)\n", event.Round)
	}
}
