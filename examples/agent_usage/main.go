// Package main shows how to use the rago agent library
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {

	// Create HTTP client that bypasses proxy
	http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 = true

	ctx := context.Background()

	fmt.Println("Creating agent...")
	// Create agent with minimal configuration
	svc, err := agent.New(&agent.AgentConfig{
		Name: "assistant",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()
	fmt.Println("Agent created successfully")

	// === Plan ===
	fmt.Println("Planning...")
	plan, err := svc.Plan(ctx, "写一个 Go 语言的 Hello World 程序")
	if err != nil {
		log.Fatalf("Plan failed: %v", err)
	}
	fmt.Printf("Plan ID: %s\n", plan.ID)

	// === Execute ===
	fmt.Println("Executing...")
	result, err := svc.Execute(ctx, plan.ID)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("Result:\n%v\n", result.FinalResult)

	// === Save to file ===
	svc.SaveToFile(fmt.Sprintf("%v", result.FinalResult), "./hello.go")
	fmt.Println("Saved to ./hello.go")
}
