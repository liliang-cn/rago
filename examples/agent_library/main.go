// Package main shows how to use the rago agent library
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// Create agent - that's all you need!
	svc, err := agent.New(&agent.AgentConfig{
		Name: "my-agent",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	// Ask a question
	result, err := svc.Run(ctx, "What is 2+2?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Answer: %v\n", result.FinalResult)
}
