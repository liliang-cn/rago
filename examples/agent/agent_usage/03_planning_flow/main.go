package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()

	svc, err := agent.New("PlannerAgent").
		WithRAG().
		WithMCP(agent.WithMCPConfigPaths("examples/mcpServers.json")).
		Build()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
	defer svc.Close()

	// 1. 制定计划 (不立即执行)
	fmt.Println("步骤 1: 正在制定计划...")
	goal := "分析 pkg/agent 目录的代码结构，并写一份简单的架构文档存为 architecture.md"
	plan, err := svc.Plan(ctx, goal)
	if err != nil {
		log.Fatalf("规划失败: %v", err)
	}

	fmt.Printf("\n目标: %s\n", plan.Goal)
	fmt.Println("推理过程:", plan.Reasoning)
	fmt.Println("\n待执行步骤:")
	for i, step := range plan.Steps {
		fmt.Printf("%d. [%s] %s\n", i+1, step.Tool, step.Description)
	}

	// 2. 模拟用户反馈修改计划 (可选)
	fmt.Println("\n步骤 2: 修改计划 (增加代码注释检查)...")
	revisedPlan, err := svc.RevisePlan(ctx, plan, "在架构文档中额外增加关于错误处理模式的分析。")
	if err != nil {
		log.Fatalf("修改计划失败: %v", err)
	}

	fmt.Println("\n修改后的步骤:")
	for i, step := range revisedPlan.Steps {
		fmt.Printf("%d. [%s] %s\n", i+1, step.Tool, step.Description)
	}

	// 3. 执行最终计划
	fmt.Println("\n步骤 3: 开始执行计划...")
	result, err := svc.ExecutePlan(ctx, revisedPlan)
	if err != nil {
		log.Fatalf("执行计划失败: %v", err)
	}

	fmt.Printf("\n执行状态: %s\n", result.Duration)
	fmt.Println("结果预览:", result.Text())
}
