package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	fmt.Println("🚀 Progress Tracking Demo")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("This demo shows how agent progress is tracked and saved to database")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure agents with database storage
	if cfg.Agents == nil {
		cfg.Agents = &config.AgentsConfig{}
	}
	cfg.Agents.Enabled = true
	cfg.Agents.MaxAgents = 3
	cfg.Agents.DataPath = ".rago/data"

	// Ensure data directory exists
	os.MkdirAll(cfg.Agents.DataPath, 0755)

	// Initialize LLM provider
	factory := providers.NewFactory()
	provider, err := providers.InitializeLLM(context.Background(), cfg, factory)
	if err != nil {
		log.Fatalf("Failed to initialize LLM: %v", err)
	}

	// Create commander with progress tracking
	commander := agents.NewCommander(cfg, provider, nil)
	commander.SetVerbose(true)

	// Mission 1: Create a mission that we'll track
	fmt.Println("\n📋 Starting Mission with Progress Tracking")
	fmt.Println("-" * 40)

	goal := `Complete these tasks with full progress tracking:
1. Generate a list of 5 creative project ideas
2. Analyze the pros and cons of each idea
3. Select the best idea and create an implementation plan
4. Write a summary of the final decision`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Execute mission
	mission, err := commander.ExecuteMission(ctx, goal)
	if err != nil {
		fmt.Printf("❌ Mission failed: %v\n", err)
	} else {
		fmt.Printf("✅ Mission completed: %s\n", mission.ID[:8])

		// Display progress information
		fmt.Println("\n📊 Retrieving Progress from Database...")

		// Get detailed progress
		progress, err := commander.GetMissionProgress(mission.ID)
		if err != nil {
			fmt.Printf("Error getting progress: %v\n", err)
		} else {
			displayProgress(progress)
		}

		// Get recent events
		fmt.Println("\n📅 Recent Progress Events:")
		events, err := commander.GetRecentProgressEvents(mission.ID, 5)
		if err == nil {
			for _, event := range events {
				fmt.Printf("  [%s] %s\n",
					event.Timestamp.Format("15:04:05"),
					event.EventType)
			}
		}

		// Save a checkpoint
		fmt.Println("\n💾 Saving Checkpoint...")
		err = commander.SaveCheckpoint(mission.ID)
		if err != nil {
			fmt.Printf("Failed to save checkpoint: %v\n", err)
		} else {
			fmt.Println("Checkpoint saved successfully!")
		}
	}

	// Demonstrate database persistence
	fmt.Println("\n" + "="*60)
	fmt.Println("📁 Database Files Created:")
	fmt.Println("-" * 40)

	// List database files
	dbPath := filepath.Join(cfg.Agents.DataPath, "*.db")
	files, _ := filepath.Glob(dbPath)
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			fmt.Printf("  • %s (%.2f KB)\n",
				filepath.Base(file),
				float64(info.Size())/1024)
		}
	}

	// Show mission can be resumed
	if mission != nil {
		fmt.Println("\n🔄 Mission Resumability Test:")
		fmt.Println("-" * 40)
		fmt.Printf("Mission ID %s is saved and can be resumed later using:\n", mission.ID[:8])
		fmt.Printf("  ./rago commander resume %s\n", mission.ID)
		fmt.Printf("  ./rago commander progress %s\n", mission.ID)
	}

	// Create another mission to show multiple missions in database
	fmt.Println("\n📋 Starting Second Mission")
	fmt.Println("-" * 40)

	goal2 := `Quick task: Write a haiku about databases`

	mission2, err := commander.ExecuteMission(ctx, goal2)
	if err == nil {
		fmt.Printf("✅ Mission 2 completed: %s\n", mission2.ID[:8])
	}

	// Show all missions in database
	fmt.Println("\n📚 All Missions in Database:")
	missions := commander.ListMissions()
	for i, m := range missions {
		fmt.Printf("  %d. %s - %s (Status: %s)\n",
			i+1, m.ID[:8], m.Goal[:50], m.Status)
	}

	// Display final metrics
	fmt.Println("\n📈 System Metrics:")
	metrics := commander.GetMetrics()
	fmt.Printf("  Total missions: %v\n", metrics["total_missions"])
	fmt.Printf("  Completed: %v\n", metrics["completed"])
	fmt.Printf("  Success rate: %.2f%%\n", metrics["success_rate"].(float64)*100)

	fmt.Println("\n✨ Progress Tracking Demo Complete!")
	fmt.Println("All progress has been saved to database files in:", cfg.Agents.DataPath)
}

func displayProgress(progress *agents.MissionProgress) {
	fmt.Printf("  Mission: %s\n", progress.MissionID[:8])
	fmt.Printf("  Goal: %.60s...\n", progress.Goal)
	fmt.Printf("  Status: %s\n", progress.Status)
	fmt.Printf("  Progress: %.1f%%\n", progress.ProgressPercentage)
	fmt.Printf("  Tasks: %d total, %d completed, %d failed\n",
		progress.TotalTasks, progress.CompletedTasks, progress.FailedTasks)

	// Show step breakdown
	if len(progress.Steps) > 0 {
		fmt.Println("\n  Step Details:")
		for _, step := range progress.Steps {
			status := "⏳"
			if step.Status == "completed" {
				status = "✅"
			} else if step.Status == "failed" {
				status = "❌"
			} else if step.Status == "executing" || step.Status == "working" {
				status = "🔄"
			}

			fmt.Printf("    %s Step %d: %.40s... (%.0f%%)\n",
				status, step.StepNumber, step.Description, step.ProgressPercentage)

			if step.DurationMs > 0 {
				fmt.Printf("       Duration: %dms\n", step.DurationMs)
			}
		}
	}
}
