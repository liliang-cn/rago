package main

import (
	"fmt"
	"log"

	rago "github.com/liliang-cn/rago/client"
)

func main() {
	// Create a new rago client with config file
	// The config file can use the new provider system or legacy Ollama configuration
	// New provider system supports multiple providers: Ollama, OpenAI, and compatible services
	// See config examples in examples/ directory for different configurations
	client, err := rago.New("config.toml")
	if err != nil {
		log.Fatalf("Failed to create rago client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Warning: failed to close client: %v\n", err)
		}
	}()

	// Example 1: Basic query
	fmt.Println("=== 1. 基础查询 ===")
	response, err := client.Query("什么是机器学习？")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("答案: %s\n", response.Answer)
		fmt.Printf("来源数量: %d\n", len(response.Sources))
		fmt.Printf("查询耗时: %s\n\n", response.Elapsed)
	}

	// Example 2: Tool-enabled query
	fmt.Println("=== 2. 工具调用查询 ===")
	toolResponse, err := client.QueryWithTools("现在几点了？", []string{"datetime"}, 3)
	if err != nil {
		log.Printf("工具查询失败: %v", err)
	} else {
		fmt.Printf("答案: %s\n", toolResponse.Answer)
		if len(toolResponse.ToolCalls) > 0 {
			fmt.Printf("执行的工具调用: %d 次\n", len(toolResponse.ToolCalls))
			for i, call := range toolResponse.ToolCalls {
				status := "成功"
				if !call.Success {
					status = "失败"
				}
				fmt.Printf("  [%d] %s - %s (%s)\n", i+1, call.Function.Name, status, call.Elapsed)
			}
		}
		fmt.Printf("使用的工具: %v\n\n", toolResponse.ToolsUsed)
	}

	// Example 3: Direct tool execution
	fmt.Println("=== 3. 直接工具执行 ===")
	toolResult, err := client.ExecuteTool("datetime", map[string]interface{}{
		"action": "now",
	})
	if err != nil {
		log.Printf("工具执行失败: %v", err)
	} else {
		fmt.Printf("工具执行成功: %v\n", toolResult.Success)
		fmt.Printf("结果数据: %v\n\n", toolResult.Data)
	}

	// Example 4: List available tools
	fmt.Println("=== 4. 可用工具列表 ===")
	tools := client.ListEnabledTools()
	fmt.Printf("启用的工具数量: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// Example 5: Document management
	fmt.Println("=== 5. 文档管理 ===")
	
	// Add text document
	err = client.IngestText("这是一个关于 Go 语言的测试文档。Go 是一门开源的编程语言，由 Google 开发。", "test-go-doc")
	if err != nil {
		log.Printf("文档添加失败: %v", err)
	} else {
		fmt.Println("✓ 文档添加成功")
	}

	// List documents
	documents, err := client.ListDocuments()
	if err != nil {
		log.Printf("列出文档失败: %v", err)
	} else {
		fmt.Printf("共有 %d 个文档\n", len(documents))
		for i, doc := range documents {
			fmt.Printf("  [%d] ID: %s, 路径: %s\n", i+1, doc.ID, doc.Path)
		}
	}
	fmt.Println()

	// Example 6: System status check
	fmt.Println("=== 6. 系统状态检查 ===")
	status := client.CheckStatus()
	fmt.Printf("Provider 可用性: %v\n", status.ProvidersAvailable)
	fmt.Printf("LLM Provider: %s\n", status.LLMProvider)
	fmt.Printf("Embedder Provider: %s\n", status.EmbedderProvider)
	
	if status.Error != nil {
		fmt.Printf("❌ 错误: %v\n", status.Error)
	} else {
		fmt.Println("✅ 系统状态正常")
	}
	fmt.Println()

	// Example 7: Tool statistics
	fmt.Println("=== 7. 工具统计信息 ===")
	stats := client.GetToolStats()
	fmt.Printf("工具统计: %v\n\n", stats)

	// Example 8: Streaming query
	fmt.Println("=== 8. 流式查询 ===")
	fmt.Print("流式回答: ")
	err = client.StreamQuery("Go 语言有什么特点？", func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("流式查询失败: %v", err)
	}
	fmt.Println("\n✓ 流式查询完成")

	fmt.Println("\n🎉 RAGO 库使用示例完成！")
}
