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
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".rago", "data", "example_chat.db")

	// 1. 使用链式 Builder API 创建 Agent
	svc, err := agent.New("Alice").
		WithDBPath(dbPath).
		WithMemory().
		Build()
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer svc.Close()

	fmt.Println("--- 第一轮对话 ---")
	res1, err := svc.Chat(ctx, "你好，我叫李华，我是一名 Gopher。")
	if err != nil {
		log.Fatalf("对话失败: %v", err)
	}
	fmt.Printf("Alice: %s\n\n", res1.Text())

	fmt.Println("--- 第二轮对话 (展示自动会话记忆) ---")
	// Agent 会自动维护 Session，不需要手动传递 ID
	res2, err := svc.Chat(ctx, "你还记得我叫什么名字吗？我的职业是什么？")
	if err != nil {
		log.Fatalf("对话失败: %v", err)
	}
	fmt.Printf("Alice: %s\n\n", res2.Text())

	fmt.Printf("当前会话 ID: %s\n", svc.CurrentSessionID())
	fmt.Println("所有对话和计划已自动持久化。")
}
