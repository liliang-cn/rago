package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/pool"
)

func main() {
	ctx := context.Background()

	cfg := &config.Config{}
	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1",
		Key:            "ollama",
		ModelName:      "qwen3.5:latest",
		MaxConcurrency: 10,
	}
	cfg.LLMPool.Providers = []pool.Provider{provider}
	cfg.LLMPool.Enabled = true

	agentSvc, err := agent.New("Greeter").
		WithConfig(cfg).
		Build()

	if err != nil {
		log.Fatalf("Failed to build agent: %v", err)
	}

	resp, err := agentSvc.Chat(ctx, "Hi")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	
	fmt.Printf("User: Hi\nAgent: %s\n", resp.Text())
}
