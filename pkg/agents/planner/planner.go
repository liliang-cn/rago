package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// TaskPlan represents a planned task with tracking
type TaskPlan struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       TaskStatus             `json:"status"`
	Priority     int                    `json:"priority"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Tools        []string               `json:"tools,omitempty"`
	Steps        []TaskStep             `json:"steps"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
}

// TaskStep represents a single step in a task
type TaskStep struct {
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Tool        string                 `json:"tool,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Status      TaskStatus             `json:"status"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// TaskStatus represents the status of a task or step
type TaskStatus string

const (
	TaskStatusPlanned    TaskStatus = "planned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusSkipped    TaskStatus = "skipped"
)

// AgentPlan represents a complete agent execution plan
type AgentPlan struct {
	ID             string                 `json:"id"`
	AgentID        string                 `json:"agent_id"`
	Goal           string                 `json:"goal"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Status         PlanStatus             `json:"status"`
	Tasks          []TaskPlan             `json:"tasks"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Summary        string                 `json:"summary,omitempty"`
	TotalSteps     int                    `json:"total_steps"`
	CompletedSteps int                    `json:"completed_steps"`
}

// PlanStatus represents the overall plan status
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusReady     PlanStatus = "ready"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
	PlanStatusCancelled PlanStatus = "cancelled"
)

// AgentPlanner handles plan creation and management
type AgentPlanner struct {
	llm        domain.Generator
	storageDir string
	mcpTools   []domain.ToolDefinition
	verbose    bool
}

// NewAgentPlanner creates a new agent planner
func NewAgentPlanner(llm domain.Generator, storageDir string) *AgentPlanner {
	return &AgentPlanner{
		llm:        llm,
		storageDir: storageDir,
		mcpTools:   []domain.ToolDefinition{},
		verbose:    false,
	}
}

// SetMCPTools sets available MCP tools for planning
func (p *AgentPlanner) SetMCPTools(tools []domain.ToolDefinition) {
	p.mcpTools = tools
}

// SetVerbose enables verbose output
func (p *AgentPlanner) SetVerbose(v bool) {
	p.verbose = v
}

// CreatePlan generates a plan for achieving a goal
func (p *AgentPlanner) CreatePlan(ctx context.Context, goal string) (*AgentPlan, error) {
	planID := uuid.New().String()

	// Create plan directory
	planDir := filepath.Join(p.storageDir, "plans", planID)
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plan directory: %w", err)
	}

	// Generate plan using LLM
	plan, err := p.generatePlan(ctx, goal)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	plan.ID = planID
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()
	plan.Status = PlanStatusReady

	// Count total steps
	totalSteps := 0
	for _, task := range plan.Tasks {
		totalSteps += len(task.Steps)
	}
	plan.TotalSteps = totalSteps

	// Save plan to filesystem
	if err := p.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	if p.verbose {
		fmt.Printf("✅ Created plan %s with %d tasks and %d total steps\n", planID, len(plan.Tasks), totalSteps)
	}

	return plan, nil
}

// generatePlan uses LLM to create a structured plan
func (p *AgentPlanner) generatePlan(ctx context.Context, goal string) (*AgentPlan, error) {
	// Build tool descriptions for the prompt
	toolDescriptions := ""
	if len(p.mcpTools) > 0 {
		toolDescriptions = "\n\nAvailable tools:\n"
		for _, tool := range p.mcpTools {
			toolDescriptions += fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
		}
	}

	prompt := fmt.Sprintf(`Create a detailed execution plan for the following goal:
%s
%s
Generate a structured plan with tasks and steps. Each task should have:
- A clear name and description
- A list of specific steps to complete
- Tools required (if any)
- Dependencies on other tasks (if any)

Return the plan as a JSON object with this structure:
{
  "goal": "the goal",
  "summary": "brief summary of the plan",
  "tasks": [
    {
      "name": "task name",
      "description": "task description",
      "priority": 1,
      "dependencies": [],
      "tools": ["tool1", "tool2"],
      "steps": [
        {
          "action": "step action",
          "description": "step description",
          "tool": "tool_name",
          "parameters": {}
        }
      ]
    }
  ]
}

Be specific and actionable in your plan. Break down complex tasks into manageable steps.`, goal, toolDescriptions)

	// Generate plan using LLM
	response, err := p.llm.Generate(ctx, prompt, &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Extract JSON from response
	jsonStr := p.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in LLM response")
	}

	// Parse the plan
	var planData struct {
		Goal    string     `json:"goal"`
		Summary string     `json:"summary"`
		Tasks   []TaskPlan `json:"tasks"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Initialize task IDs and statuses
	for i := range planData.Tasks {
		planData.Tasks[i].ID = fmt.Sprintf("task_%d", i+1)
		planData.Tasks[i].Status = TaskStatusPlanned
		for j := range planData.Tasks[i].Steps {
			planData.Tasks[i].Steps[j].ID = fmt.Sprintf("step_%d_%d", i+1, j+1)
			planData.Tasks[i].Steps[j].Status = TaskStatusPlanned
		}
	}

	return &AgentPlan{
		Goal:    planData.Goal,
		Summary: planData.Summary,
		Tasks:   planData.Tasks,
		Context: make(map[string]interface{}),
	}, nil
}

// extractJSON extracts JSON content from LLM response
func (p *AgentPlanner) extractJSON(content string) string {
	// Look for JSON block markers
	startIdx := -1
	endIdx := -1

	// Try to find ```json blocks first
	if idx := strings.Index(content, "```json"); idx != -1 {
		startIdx = idx + 7
		if idx := strings.Index(content[startIdx:], "```"); idx != -1 {
			endIdx = startIdx + idx
			return strings.TrimSpace(content[startIdx:endIdx])
		}
	}

	// Try to find raw JSON object
	if idx := strings.Index(content, "{"); idx != -1 {
		startIdx = idx
		// Find matching closing brace
		braceCount := 0
		for i := startIdx; i < len(content); i++ {
			if content[i] == '{' {
				braceCount++
			} else if content[i] == '}' {
				braceCount--
				if braceCount == 0 {
					endIdx = i + 1
					break
				}
			}
		}
		if endIdx > startIdx {
			return strings.TrimSpace(content[startIdx:endIdx])
		}
	}

	return ""
}

// SavePlan saves a plan to the filesystem
func (p *AgentPlanner) SavePlan(plan *AgentPlan) error {
	planFile := filepath.Join(p.storageDir, "plans", plan.ID, "plan.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(planFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal plan to JSON
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	// Write to file
	if err := os.WriteFile(planFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	// Also save a tracking file for quick status checks
	trackingFile := filepath.Join(p.storageDir, "plans", plan.ID, "tracking.json")
	tracking := map[string]interface{}{
		"id":              plan.ID,
		"goal":            plan.Goal,
		"status":          plan.Status,
		"total_steps":     plan.TotalSteps,
		"completed_steps": plan.CompletedSteps,
		"updated_at":      plan.UpdatedAt,
	}

	trackingData, err := json.MarshalIndent(tracking, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	if err := os.WriteFile(trackingFile, trackingData, 0644); err != nil {
		return fmt.Errorf("failed to write tracking file: %w", err)
	}

	return nil
}

// LoadPlan loads a plan from the filesystem
func (p *AgentPlanner) LoadPlan(planID string) (*AgentPlan, error) {
	planFile := filepath.Join(p.storageDir, "plans", planID, "plan.json")

	data, err := os.ReadFile(planFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan AgentPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// UpdateTaskStatus updates the status of a task in a plan
func (p *AgentPlanner) UpdateTaskStatus(planID, taskID string, status TaskStatus) error {
	plan, err := p.LoadPlan(planID)
	if err != nil {
		return err
	}

	// Find and update task
	for i := range plan.Tasks {
		if plan.Tasks[i].ID == taskID {
			plan.Tasks[i].Status = status
			now := time.Now()

			switch status {
			case TaskStatusInProgress:
				plan.Tasks[i].StartedAt = &now
			case TaskStatusCompleted, TaskStatusFailed:
				plan.Tasks[i].CompletedAt = &now
			}

			break
		}
	}

	// Update plan metadata
	plan.UpdatedAt = time.Now()
	p.updatePlanStatus(plan)

	return p.SavePlan(plan)
}

// UpdateStepStatus updates the status of a step in a task
func (p *AgentPlanner) UpdateStepStatus(planID, taskID, stepID string, status TaskStatus, output interface{}, errorMsg string) error {
	plan, err := p.LoadPlan(planID)
	if err != nil {
		return err
	}

	// Find and update step
	for i := range plan.Tasks {
		if plan.Tasks[i].ID == taskID {
			for j := range plan.Tasks[i].Steps {
				if plan.Tasks[i].Steps[j].ID == stepID {
					step := &plan.Tasks[i].Steps[j]
					step.Status = status
					now := time.Now()

					switch status {
					case TaskStatusInProgress:
						step.StartedAt = &now
					case TaskStatusCompleted, TaskStatusFailed:
						step.CompletedAt = &now
						if output != nil {
							step.Output = output
						}
						if errorMsg != "" {
							step.Error = errorMsg
						}
					}

					break
				}
			}
			break
		}
	}

	// Update completed steps count
	completedSteps := 0
	for _, task := range plan.Tasks {
		for _, step := range task.Steps {
			if step.Status == TaskStatusCompleted {
				completedSteps++
			}
		}
	}
	plan.CompletedSteps = completedSteps

	// Update plan metadata
	plan.UpdatedAt = time.Now()
	p.updatePlanStatus(plan)

	return p.SavePlan(plan)
}

// updatePlanStatus updates the overall plan status based on task statuses
func (p *AgentPlanner) updatePlanStatus(plan *AgentPlan) {
	allCompleted := true
	anyFailed := false
	anyInProgress := false

	for _, task := range plan.Tasks {
		switch task.Status {
		case TaskStatusFailed:
			anyFailed = true
		case TaskStatusInProgress:
			anyInProgress = true
			allCompleted = false
		case TaskStatusPlanned, TaskStatusSkipped:
			allCompleted = false
		}
	}

	if anyFailed {
		plan.Status = PlanStatusFailed
	} else if allCompleted {
		plan.Status = PlanStatusCompleted
	} else if anyInProgress {
		plan.Status = PlanStatusExecuting
	}
}

// ListPlans lists all available plans
func (p *AgentPlanner) ListPlans() ([]*AgentPlan, error) {
	plansDir := filepath.Join(p.storageDir, "plans")

	// Ensure directory exists
	if _, err := os.Stat(plansDir); os.IsNotExist(err) {
		return []*AgentPlan{}, nil
	}

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []*AgentPlan
	for _, entry := range entries {
		if entry.IsDir() {
			plan, err := p.LoadPlan(entry.Name())
			if err != nil {
				// Skip invalid plans
				continue
			}
			plans = append(plans, plan)
		}
	}

	return plans, nil
}
