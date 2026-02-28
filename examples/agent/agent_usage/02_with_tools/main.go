package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	// 1. 使用链式 Builder API 创建具备完整能力的 Agent
	svc, err := agent.New("WorkerAgent").
		WithMCP(agent.WithMCPConfigPaths("examples/mcpServers.json")).
		WithSkills().
		WithRAG().
		WithMemory().
		WithProgressCallback(func(e agent.ProgressEvent) {
			if e.Type == "thinking" {
				fmt.Printf("🤔 %s\n", e.Message)
			} else if e.Type == "tool_call" {
				fmt.Printf("🛠️  正在使用工具: %s\n", e.Tool)
			}
		}).
		Build()
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer svc.Close()

	fmt.Println("=== 复杂任务执行 ===")
	// Agent 会根据问题自动决定是直接回答、查知识库还是调工具
	goal := "请帮我搜索关于 Go 1.24 版本的新特性，并将其总结存入我的长期记忆中。"

	result, err := svc.Run(ctx, goal)
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}

	fmt.Println("\n=== 最终结果 ===")
	fmt.Printf("%v\n", result.FinalResult)
}
