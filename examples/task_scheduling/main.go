package main

import (
	"context"
	"fmt"
	"log"

	rago "github.com/liliang-cn/rago/lib"
)

func main() {
	// Initialize rago client with configuration
	client, err := rago.New("../../config.toml")
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Enable task scheduling
	err = client.EnableTasks(ctx)
	if err != nil {
		log.Fatalf("Failed to enable tasks: %v", err)
	}

	fmt.Println("🚀 Rago Task Demo with Priorities")
	fmt.Println("=================================")

	// Example 1: Create a high-priority script task
	fmt.Println("\n📝 Creating high-priority script task...")
	scriptTaskID, err := client.CreateTask(rago.TaskOptions{
		Type:     "script",
		Schedule: "", // empty schedule = run once
		Parameters: map[string]string{
			"script":  "echo 'High priority task executed!'",
			"workdir": "/tmp",
		},
		Description: "High priority script task",
		Priority:    10, // high priority
		Enabled:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create script task: %v", err)
	}
	fmt.Printf("✅ High-priority script task created: %s\n", scriptTaskID[:8])

	// Example 2: Create a medium-priority query task
	fmt.Println("\n🔍 Creating medium-priority query task...")
	queryTaskID, err := client.CreateTask(rago.TaskOptions{
		Type:     "query",
		Schedule: "0 */6 * * *", // run every 6 hours
		Parameters: map[string]string{
			"query":        "What are the latest developments in AI?",
			"top-k":        "3",
			"show-sources": "true",
		},
		Description: "Medium priority AI news query",
		Priority:    5, // medium priority
		Enabled:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create query task: %v", err)
	}
	fmt.Printf("✅ Medium-priority query task created: %s\n", queryTaskID[:8])

	// Example 3: Create a low-priority ingest task
	fmt.Println("\n📄 Creating low-priority ingest task...")
	ingestTaskID, err := client.CreateTask(rago.TaskOptions{
		Type:     "ingest",
		Schedule: "0 0 * * 0", // run weekly on Sundays at midnight
		Parameters: map[string]string{
			"path":      "./knowledge",
			"recursive": "true",
			"pattern":   "*.md,*.pdf",
		},
		Description: "Low priority document ingest",
		Priority:    1, // low priority
		Enabled:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create ingest task: %v", err)
	}
	fmt.Printf("✅ Low-priority ingest task created: %s\n", ingestTaskID[:8])

	// Example 4: Create a very high priority MCP tool task
	fmt.Println("\n🛠️ Creating very high priority MCP task...")
	mcpTaskID, err := client.CreateTask(rago.TaskOptions{
		Type:     "mcp",
		Schedule: "0 9 * * *", // run daily at 9 AM
		Parameters: map[string]string{
			"tool": "fetch_latest_news",
			"url":  "https://api.example.com/news",
		},
		Description: "Very high priority news fetch",
		Priority:    20, // very high priority
		Enabled:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP task: %v", err)
	}
	fmt.Printf("✅ Very high-priority MCP task created: %s\n", mcpTaskID[:8])

	// List all tasks (should be sorted by priority)
	fmt.Println("\n📋 Listing all tasks (sorted by priority)...")
	tasks, err := client.ListTasks(false) // don't include disabled
	if err != nil {
		log.Fatalf("Failed to list tasks: %v", err)
	}

	fmt.Printf("Found %d tasks:\n", len(tasks))
	for _, task := range tasks {
		status := "⏸️"
		if task.Enabled {
			status = "▶️"
		}
		schedule := task.Schedule
		if schedule == "" {
			schedule = "run once"
		}
		fmt.Printf("  %s P%d: %s (%s) - %s\n", status, task.Priority, task.Description, task.Type, schedule)
	}

	// Run the high priority script task immediately
	fmt.Println("\n🎯 Running high-priority script task...")
	result, err := client.RunTask(scriptTaskID)
	if err != nil {
		log.Printf("Failed to run script task: %v", err)
	} else {
		fmt.Println("✅ Script task executed successfully")
		if result.Output != "" {
			fmt.Printf("📄 Output: %s\n", result.Output)
		}
	}

	fmt.Println("\n✨ Demo completed!")
	fmt.Println("💡 Use 'rago task list' to see tasks sorted by priority")
	fmt.Println("💡 Use 'rago task run <id>' to execute tasks")
	fmt.Println("💡 Higher priority numbers run first")
}
