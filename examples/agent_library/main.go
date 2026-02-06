// Package main shows how to use the rago agent library
package main

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// Create agent with minimal configuration
	svc, _ := agent.New(&agent.AgentConfig{
		Name: "assistant",
	})
	defer svc.Close()

	// === Plan ===
	plan, _ := svc.Plan(ctx, "写一个 Go 语言的 Hello World 程序")
	fmt.Printf("Plan ID: %s\n", plan.ID)

	// === Execute ===
	result, _ := svc.Execute(ctx, plan.ID)
	fmt.Printf("Result:\n%v\n", result.FinalResult)

	// === Save to file ===
	svc.SaveToFile(fmt.Sprintf("%v", result.FinalResult), "./hello.go")
	fmt.Println("✅ Saved to ./hello.go")
}
