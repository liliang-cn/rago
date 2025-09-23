// Example: Agent and Task Automation
// This example demonstrates task scheduling and agent capabilities

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Initialize client
	configPath := filepath.Join(os.Getenv("HOME"), ".rago", "rago.toml")
	c, err := client.New(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Example 1: Simple agent task execution
	fmt.Println("=== Example 1: Simple Agent Task ===")
	if c.Agent != nil {
		task := "Summarize the benefits of using Go for backend development"

		result, err := c.Agent.Run(task)
		if err != nil {
			log.Printf("Task execution error: %v\n", err)
		} else {
			fmt.Printf("Task: %s\n", task)
			fmt.Printf("Success: %v\n", result.Success)
			if result.Output != nil {
				fmt.Printf("Result: %v\n", result.Output)
			}
		}
	} else {
		fmt.Println("Agent not initialized")
	}

	// Example 2: Agent task with options
	fmt.Println("\n=== Example 2: Agent with Options ===")
	if c.Agent != nil {
		opts := &client.AgentOptions{
			Verbose: true,
			Timeout: 30,
		}

		task := "Create a simple data structure for a todo list in Go"
		result, err := c.Agent.RunWithOptions(ctx, task, opts)
		if err != nil {
			log.Printf("Task execution error: %v\n", err)
		} else {
			fmt.Printf("Task completed successfully: %v\n", result.Success)
			if result.Output != nil {
				fmt.Printf("Output: %v\n", result.Output)
			}
		}
	}

	// Example 3: Task planning
	fmt.Println("\n=== Example 3: Task Planning ===")
	if c.Agent != nil {
		task := "Build a REST API for a blog system"

		plan, err := c.Agent.PlanWithOptions(ctx, task, nil)
		if err != nil {
			log.Printf("Planning error: %v\n", err)
		} else {
			fmt.Printf("Plan for: %s\n", plan.Task)
			fmt.Println("Steps:")
			for i, step := range plan.Steps {
				fmt.Printf("  %d. %s: %s\n", i+1, step.Name, step.Description)
			}
		}
	}

	// Example 4: Direct task execution using BaseClient
	fmt.Println("\n=== Example 4: Direct Task Execution ===")
	req := client.TaskRequest{
		Task:    "Write a haiku about programming",
		Verbose: false,
	}

	resp, err := c.RunTask(ctx, req)
	if err != nil {
		log.Printf("Task error: %v\n", err)
	} else {
		fmt.Printf("Task success: %v\n", resp.Success)
		if resp.Output != nil {
			fmt.Printf("Output: %v\n", resp.Output)
		}
	}

	// Example 5: Task scheduling
	fmt.Println("\n=== Example 5: Task Scheduling ===")

	// Enable task scheduler
	err = c.EnableTasks(ctx)
	if err != nil {
		log.Printf("Failed to enable task scheduler: %v\n", err)
	} else {
		fmt.Println("✓ Task scheduler enabled")

		// Create a scheduled RAG query task
		schedule := "*/5 * * * *" // Every 5 minutes
		taskID, err := c.CreateQueryTask(
			"What are the latest updates?",
			schedule,
			map[string]string{
				"top_k": "5",
			},
		)

		if err != nil {
			log.Printf("Failed to create scheduled task: %v\n", err)
		} else {
			fmt.Printf("✓ Created scheduled query task: %s\n", taskID)

			// Get task info
			taskInfo, err := c.GetTask(taskID)
			if err == nil {
				fmt.Printf("  Type: %s\n", taskInfo.Type)
				fmt.Printf("  Schedule: %s\n", taskInfo.Schedule)
				fmt.Printf("  Enabled: %v\n", taskInfo.Enabled)
			}

			// Run task immediately (for demo)
			fmt.Println("\nRunning task immediately...")
			result, err := c.RunScheduledTask(taskID)
			if err != nil {
				log.Printf("Task execution error: %v\n", err)
			} else {
				fmt.Printf("  Success: %v\n", result.Success)
				if result.Output != "" {
					fmt.Printf("  Output: %s\n", result.Output)
				}
			}

			// Clean up - delete the task
			if err := c.DeleteTask(taskID); err == nil {
				fmt.Println("✓ Cleaned up scheduled task")
			}
		}
	}

	// Example 6: Script task
	fmt.Println("\n=== Example 6: Script Task ===")
	if c.IsTasksEnabled() {
		script := `echo "Hello from RAGO script task"`

		// Create a one-time script task (empty schedule)
		taskID, err := c.CreateScriptTask(script, "", nil)
		if err != nil {
			log.Printf("Failed to create script task: %v\n", err)
		} else {
			fmt.Printf("✓ Created script task: %s\n", taskID)

			// Run it
			result, err := c.RunScheduledTask(taskID)
			if err != nil {
				log.Printf("Script execution error: %v\n", err)
			} else {
				fmt.Printf("  Success: %v\n", result.Success)
				fmt.Printf("  Duration: %v\n", result.Duration)
			}

			// Clean up
			c.DeleteTask(taskID)
		}
	}

	// Example 7: List all tasks
	fmt.Println("\n=== Example 7: Task Management ===")
	if c.IsTasksEnabled() {
		// Create a few sample tasks
		task1, _ := c.CreateQueryTask("Sample query 1", "0 0 * * *", nil)
		task2, _ := c.CreateIngestTask("/path/to/doc.pdf", "0 */6 * * *", nil)

		// List all tasks
		tasks, err := c.ListTasks(false)
		if err != nil {
			log.Printf("Failed to list tasks: %v\n", err)
		} else {
			fmt.Printf("Found %d active tasks:\n", len(tasks))
			for _, task := range tasks {
				fmt.Printf("  - %s: %s (%s)\n", task.ID, task.Description, task.Type)
				if task.NextRun != nil {
					fmt.Printf("    Next run: %v\n", task.NextRun.Format(time.RFC3339))
				}
			}
		}

		// Clean up
		c.DeleteTask(task1)
		c.DeleteTask(task2)

		// Disable task scheduler
		if err := c.DisableTasks(); err == nil {
			fmt.Println("✓ Task scheduler disabled")
		}
	}

	// Example 8: MCP tool task
	fmt.Println("\n=== Example 8: MCP Tool Task ===")

	// Re-enable tasks for this example
	c.EnableTasks(ctx)

	if c.IsTasksEnabled() && c.Tools != nil {
		// Check if any tools are available
		tools, _ := c.Tools.List()
		if len(tools) > 0 {
			// Create an MCP tool task
			toolName := tools[0].Name
			args := map[string]interface{}{
				"test": "value",
			}

			taskID, err := c.CreateMCPTask(toolName, args, "")
			if err != nil {
				log.Printf("Failed to create MCP task: %v\n", err)
			} else {
				fmt.Printf("✓ Created MCP tool task for '%s'\n", toolName)

				// Clean up
				c.DeleteTask(taskID)
			}
		} else {
			fmt.Println("No MCP tools available")
		}
	}

	// Final cleanup
	c.DisableTasks()

	fmt.Println("\n=== Agent and Task Automation Complete ===")
}
