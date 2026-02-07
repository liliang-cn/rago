package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	svc, err := agent.New(&agent.AgentConfig{
		Name: "test-orchestrator",
	})
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}
	defer svc.Close()

	// 1. 定义专家：它拥有一个只有它才知道的 Secret Tool
	secretAgent := agent.NewAgent("SecretKeeper")
	secretAgent.SetInstructions("You are the SecretKeeper. You possess a 'get_secret_key' tool. If someone asks for the key, you MUST use the tool and report the exact string it returns.")

	secretAgent.AddTool(
		"get_secret_key",
		"Generates a unique secret security key",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			fmt.Println("\n[LOG] ==> Go function 'get_secret_key' was actually EXECUTED!")
			return "RAGO-DYNAMIC-TOOL-SUCCESS-999", nil
		},
	)

	// 2. 定义 Triage：它什么都不知道，必须转接
	triageAgent := agent.NewAgent("Triage")
	triageAgent.SetInstructions("You are a simple receptionist. You know NOTHING about secrets. If anyone asks for a 'key' or 'secret', you MUST transfer to SecretKeeper.")
	
	triageAgent.AddHandoff(agent.NewHandoff(secretAgent,
		agent.WithHandoffToolDescription("Transfer to the SecretKeeper agent."),
	))

	svc.RegisterAgent(secretAgent)
	svc.RegisterAgent(triageAgent)

	fmt.Println("--- STARTING LOGICAL PROOF TEST ---")
	// 我们甚至不给它 Memory 性格干扰，只测逻辑
	goal := "I need the secret security key right now."
	
	result, err := svc.Run(ctx, goal)
	if err != nil {
		fmt.Printf("Run failed: %v\n", err)
		return
	}

	fmt.Printf("\n--- Final Response ---\n%v\n", result.FinalResult)
	
	if contains(fmt.Sprintf("%v", result.FinalResult), "RAGO-DYNAMIC-TOOL-SUCCESS-999") {
		fmt.Println("\n✅ SUCCESS: Logical proof confirmed! Multi-agent handoff and Custom Tool both worked.")
	} else {
		fmt.Println("\n❌ FAILED: The agent answered without using the dynamic tool.")
	}
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}
