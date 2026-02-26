package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx := context.Background()
	cwd, _ := os.Getwd()
	
	// 设置数据存储路径
	dataDir := filepath.Join(cwd, "examples", "things", "data")
	os.MkdirAll(dataDir, 0755)

	// 1. 初始化 Things Agent
	fmt.Println("🚀 正在启动 Things (Ollama qwen3:8b)...")
	svc, err := agent.New(&agent.AgentConfig{
		Name:            "ThingsAssistant",
		MemoryStoreType: "file", // 使用透明文件记忆
		MemoryDBPath:    filepath.Join(dataDir, "memories"),
		EnableMemory:    true,
		SystemPrompt: `你是一个专业的个人任务管理助理。
你的核心能力：
1. **时间提取**：从用户输入中提取任务内容和时间。
2. **绝对化**：参考 System Context 中的当前时间，将“明天”、“下午”等相对时间转化为 YYYY-MM-DD HH:MM 格式。
3. **记忆存储**：使用 memory_save 将整理后的任务存入记忆。
4. **计划查询**：当用户问及计划时，先查看记忆地图，然后回答。

请确保存储的内容条理清晰，例如：{"todo": "去医院", "time": "2026-02-27 15:00"}`,
	})
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	defer svc.Close()

	fmt.Println("✅ Things 已就绪。")

	// --- 模拟操作流程 ---

	// 步骤 1: 录入相对时间的任务
	fmt.Println("\n>>> 用户: 明天下午三点去医院复查")
	res1, err := svc.Run(ctx, "明天下午三点去医院复查")
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
	} else {
		fmt.Printf("Things: %v\n", res1.FinalResult)
	}

	// 步骤 2: 再次录入
	fmt.Println("\n>>> 用户: 下周一早上九点开周会")
	res2, err := svc.Run(ctx, "下周一早上九点开周会")
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
	} else {
		fmt.Printf("Things: %v\n", res2.FinalResult)
	}

	// 步骤 3: 综合查询
	fmt.Println("\n>>> 用户: 我明天有什么计划？")
	res3, err := svc.Run(ctx, "我明天有什么计划？")
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
	} else {
		fmt.Printf("Things: %v\n", res3.FinalResult)
	}

	fmt.Printf("\n[提示] 任务已以 Markdown 格式保存在: %s\n", filepath.Join(dataDir, "memories", "entities"))
}
