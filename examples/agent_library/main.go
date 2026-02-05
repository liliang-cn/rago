// Package main shows how to use the rago agent library with memory
package main

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// Create agent
	svc, _ := agent.New(&agent.AgentConfig{
		Name: "assistant",
	})
	defer svc.Close()

	// Chat with auto-generated session ID (UUID)
	sessionID := "chat-001"

	svc.RunWithSession(ctx, "My name is Alice", sessionID)
	result, _ := svc.RunWithSession(ctx, "What's my name?", sessionID)

	fmt.Printf("Answer: %v\n", result.FinalResult)
}
