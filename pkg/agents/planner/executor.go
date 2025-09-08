package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// PlanExecutor executes agent plans with tracking
type PlanExecutor struct {
	planner   *AgentPlanner
	mcpClient *mcp.Client
	verbose   bool
}

// NewPlanExecutor creates a new plan executor
func NewPlanExecutor(planner *AgentPlanner, mcpClient *mcp.Client) *PlanExecutor {
	return &PlanExecutor{
		planner:   planner,
		mcpClient: mcpClient,
		verbose:   false,
	}
}

// SetVerbose enables verbose output
func (e *PlanExecutor) SetVerbose(v bool) {
	e.verbose = v
}

// ExecutePlan executes a plan with progress tracking
func (e *PlanExecutor) ExecutePlan(ctx context.Context, planID string) error {
	// Load the plan
	plan, err := e.planner.LoadPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}
	
	if e.verbose {
		fmt.Printf("\n🚀 Executing plan: %s\n", plan.Goal)
		fmt.Printf("📋 Total tasks: %d, Total steps: %d\n\n", len(plan.Tasks), plan.TotalSteps)
	}
	
	// Update plan status to executing
	plan.Status = PlanStatusExecuting
	if err := e.planner.SavePlan(plan); err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}
	
	// Execute tasks in order (respecting dependencies)
	completedTasks := make(map[string]bool)
	
	for len(completedTasks) < len(plan.Tasks) {
		// Find next executable task
		nextTask := e.findNextTask(plan.Tasks, completedTasks)
		if nextTask == nil {
			return fmt.Errorf("no executable tasks found, possible circular dependency")
		}
		
		// Execute the task
		if err := e.executeTask(ctx, plan.ID, nextTask); err != nil {
			// Mark task as failed
			e.planner.UpdateTaskStatus(plan.ID, nextTask.ID, TaskStatusFailed)
			
			if e.verbose {
				fmt.Printf("❌ Task %s failed: %v\n", nextTask.Name, err)
			}
			
			// Continue with other tasks or fail the plan based on configuration
			// For now, we'll continue
		} else {
			completedTasks[nextTask.ID] = true
		}
	}
	
	// Update final plan status
	plan, _ = e.planner.LoadPlan(planID)
	if plan.CompletedSteps == plan.TotalSteps {
		plan.Status = PlanStatusCompleted
		if e.verbose {
			fmt.Printf("\n✅ Plan completed successfully! (%d/%d steps)\n", plan.CompletedSteps, plan.TotalSteps)
		}
	} else {
		plan.Status = PlanStatusFailed
		if e.verbose {
			fmt.Printf("\n⚠️  Plan completed with issues (%d/%d steps completed)\n", plan.CompletedSteps, plan.TotalSteps)
		}
	}
	
	return e.planner.SavePlan(plan)
}

// findNextTask finds the next task that can be executed
func (e *PlanExecutor) findNextTask(tasks []TaskPlan, completed map[string]bool) *TaskPlan {
	for i := range tasks {
		task := &tasks[i]
		
		// Skip completed tasks
		if completed[task.ID] {
			continue
		}
		
		// Check if all dependencies are satisfied
		canExecute := true
		for _, dep := range task.Dependencies {
			if !completed[dep] {
				canExecute = false
				break
			}
		}
		
		if canExecute {
			return task
		}
	}
	
	return nil
}

// executeTask executes a single task with all its steps
func (e *PlanExecutor) executeTask(ctx context.Context, planID string, task *TaskPlan) error {
	if e.verbose {
		fmt.Printf("\n📌 Starting task: %s\n", task.Name)
		fmt.Printf("   %s\n", task.Description)
	}
	
	// Update task status to in progress
	if err := e.planner.UpdateTaskStatus(planID, task.ID, TaskStatusInProgress); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	// Execute each step
	for _, step := range task.Steps {
		if err := e.executeStep(ctx, planID, task.ID, &step); err != nil {
			// Mark step as failed
			e.planner.UpdateStepStatus(planID, task.ID, step.ID, TaskStatusFailed, nil, err.Error())
			
			if e.verbose {
				fmt.Printf("   ❌ Step failed: %s - %v\n", step.Description, err)
			}
			
			// Mark task as failed and return
			e.planner.UpdateTaskStatus(planID, task.ID, TaskStatusFailed)
			return fmt.Errorf("step %s failed: %w", step.ID, err)
		}
		
		if e.verbose {
			fmt.Printf("   ✓ %s\n", step.Description)
		}
	}
	
	// Mark task as completed
	if err := e.planner.UpdateTaskStatus(planID, task.ID, TaskStatusCompleted); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	if e.verbose {
		fmt.Printf("✅ Task completed: %s\n", task.Name)
	}
	
	return nil
}

// executeStep executes a single step
func (e *PlanExecutor) executeStep(ctx context.Context, planID, taskID string, step *TaskStep) error {
	// Update step status to in progress
	if err := e.planner.UpdateStepStatus(planID, taskID, step.ID, TaskStatusInProgress, nil, ""); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}
	
	// Execute based on tool
	var output interface{}
	var err error
	
	if step.Tool != "" && e.mcpClient != nil {
		// Execute using MCP tool
		output, err = e.executeMCPTool(ctx, step.Tool, step.Parameters)
	} else {
		// Execute as a generic action (could be expanded)
		output, err = e.executeGenericAction(ctx, step.Action, step.Parameters)
	}
	
	if err != nil {
		return err
	}
	
	// Update step status to completed
	if err := e.planner.UpdateStepStatus(planID, taskID, step.ID, TaskStatusCompleted, output, ""); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}
	
	return nil
}

// executeMCPTool executes an MCP tool
func (e *PlanExecutor) executeMCPTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	if e.mcpClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}
	
	// Call the tool with parameters directly
	result, err := e.mcpClient.CallTool(ctx, toolName, params)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}
	
	if !result.Success {
		return nil, fmt.Errorf("MCP tool returned error: %s", result.Error)
	}
	
	return result.Data, nil
}

// executeGenericAction executes a generic action (placeholder for now)
func (e *PlanExecutor) executeGenericAction(ctx context.Context, action string, params map[string]interface{}) (interface{}, error) {
	// This could be expanded to handle various generic actions
	// For now, just simulate execution
	time.Sleep(100 * time.Millisecond)
	
	return map[string]interface{}{
		"action":     action,
		"parameters": params,
		"status":     "completed",
		"timestamp":  time.Now().Format(time.RFC3339),
	}, nil
}

// GetPlanProgress returns the current progress of a plan
func (e *PlanExecutor) GetPlanProgress(planID string) (*PlanProgress, error) {
	plan, err := e.planner.LoadPlan(planID)
	if err != nil {
		return nil, err
	}
	
	progress := &PlanProgress{
		PlanID:         plan.ID,
		Goal:           plan.Goal,
		Status:         plan.Status,
		TotalTasks:     len(plan.Tasks),
		CompletedTasks: 0,
		TotalSteps:     plan.TotalSteps,
		CompletedSteps: plan.CompletedSteps,
		TaskProgress:   make([]TaskProgress, 0),
	}
	
	for _, task := range plan.Tasks {
		if task.Status == TaskStatusCompleted {
			progress.CompletedTasks++
		}
		
		taskProg := TaskProgress{
			TaskID:         task.ID,
			Name:           task.Name,
			Status:         task.Status,
			TotalSteps:     len(task.Steps),
			CompletedSteps: 0,
		}
		
		for _, step := range task.Steps {
			if step.Status == TaskStatusCompleted {
				taskProg.CompletedSteps++
			}
		}
		
		progress.TaskProgress = append(progress.TaskProgress, taskProg)
	}
	
	// Calculate percentage
	if progress.TotalSteps > 0 {
		progress.PercentComplete = float64(progress.CompletedSteps) / float64(progress.TotalSteps) * 100
	}
	
	return progress, nil
}

// PlanProgress represents the progress of a plan execution
type PlanProgress struct {
	PlanID          string         `json:"plan_id"`
	Goal            string         `json:"goal"`
	Status          PlanStatus     `json:"status"`
	TotalTasks      int            `json:"total_tasks"`
	CompletedTasks  int            `json:"completed_tasks"`
	TotalSteps      int            `json:"total_steps"`
	CompletedSteps  int            `json:"completed_steps"`
	PercentComplete float64        `json:"percent_complete"`
	TaskProgress    []TaskProgress `json:"task_progress"`
}

// TaskProgress represents the progress of a single task
type TaskProgress struct {
	TaskID         string     `json:"task_id"`
	Name           string     `json:"name"`
	Status         TaskStatus `json:"status"`
	TotalSteps     int        `json:"total_steps"`
	CompletedSteps int        `json:"completed_steps"`
}

// ResumePlan resumes execution of a paused or failed plan
func (e *PlanExecutor) ResumePlan(ctx context.Context, planID string) error {
	plan, err := e.planner.LoadPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}
	
	if e.verbose {
		fmt.Printf("\n🔄 Resuming plan: %s\n", plan.Goal)
		fmt.Printf("📊 Progress: %d/%d steps completed\n\n", plan.CompletedSteps, plan.TotalSteps)
	}
	
	// Continue from where we left off
	return e.ExecutePlan(ctx, planID)
}