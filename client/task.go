package client

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/scheduler"
	"github.com/liliang-cn/rago/v2/pkg/scheduler/executors"
)

// TaskClient provides task scheduling functionality
type TaskClient struct {
	scheduler scheduler.Scheduler
	enabled   bool
}

// TaskOptions contains options for task creation
type TaskOptions struct {
	Type        string            `json:"type"`
	Schedule    string            `json:"schedule,omitempty"`
	Parameters  map[string]string `json:"parameters"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"`
	Enabled     bool              `json:"enabled"`
}

// TaskInfo represents task information for API responses
type TaskInfo struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Schedule    string            `json:"schedule,omitempty"`
	Parameters  map[string]string `json:"parameters"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	NextRun     *time.Time        `json:"next_run,omitempty"`
	LastRun     *time.Time        `json:"last_run,omitempty"`
}

// TaskExecution represents a task execution result
type TaskExecution struct {
	ID        int64         `json:"id"`
	TaskID    string        `json:"task_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"`
	Output    string        `json:"output,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// TaskResult represents the result of running a task
type TaskResult struct {
	Success  bool          `json:"success"`
	Output   string        `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// EnableTasks initializes the task scheduler
func (c *BaseClient) EnableTasks(ctx context.Context) error {
	if c.taskClient != nil && c.taskClient.enabled {
		return nil // Already enabled
	}

	// Create scheduler
	sched := scheduler.NewScheduler(c.config)
	if err := sched.Start(); err != nil {
		return fmt.Errorf("failed to start task scheduler: %w", err)
	}

	// Register default executors
	sched.RegisterExecutor(executors.NewQueryExecutor(c.config))
	sched.RegisterExecutor(executors.NewScriptExecutor(c.config))
	sched.RegisterExecutor(executors.NewIngestExecutor(c.config))
	sched.RegisterExecutor(executors.NewMCPExecutor(c.config))

	c.taskClient = &TaskClient{
		scheduler: sched,
		enabled:   true,
	}

	return nil
}

// DisableTasks stops the task scheduler
func (c *BaseClient) DisableTasks() error {
	if c.taskClient == nil || !c.taskClient.enabled {
		return nil
	}

	if err := c.taskClient.scheduler.Stop(); err != nil {
		return fmt.Errorf("failed to stop task scheduler: %w", err)
	}

	c.taskClient.enabled = false
	return nil
}

// IsTasksEnabled returns whether task scheduling is enabled
func (c *BaseClient) IsTasksEnabled() bool {
	return c.taskClient != nil && c.taskClient.enabled
}

// CreateTask creates a new scheduled task
func (c *BaseClient) CreateTask(options TaskOptions) (string, error) {
	if !c.IsTasksEnabled() {
		return "", fmt.Errorf("task scheduler is not enabled")
	}

	task := &scheduler.Task{
		Type:        options.Type,
		Schedule:    options.Schedule,
		Parameters:  options.Parameters,
		Description: options.Description,
		Priority:    options.Priority,
		Enabled:     options.Enabled,
	}

	return c.taskClient.scheduler.CreateTask(task)
}

// GetTask retrieves a task by ID
func (c *BaseClient) GetTask(taskID string) (*TaskInfo, error) {
	if !c.IsTasksEnabled() {
		return nil, fmt.Errorf("task scheduler is not enabled")
	}

	task, err := c.taskClient.scheduler.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	return &TaskInfo{
		ID:          task.ID,
		Type:        task.Type,
		Schedule:    task.Schedule,
		Parameters:  task.Parameters,
		Description: task.Description,
		Priority:    task.Priority,
		Enabled:     task.Enabled,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		NextRun:     task.NextRun,
		LastRun:     task.LastRun,
	}, nil
}

// ListTasks lists all tasks
func (c *BaseClient) ListTasks(includeDisabled bool) ([]*TaskInfo, error) {
	if !c.IsTasksEnabled() {
		return nil, fmt.Errorf("task scheduler is not enabled")
	}

	tasks, err := c.taskClient.scheduler.ListTasks(includeDisabled)
	if err != nil {
		return nil, err
	}

	var taskInfos []*TaskInfo
	for _, task := range tasks {
		taskInfos = append(taskInfos, &TaskInfo{
			ID:          task.ID,
			Type:        task.Type,
			Schedule:    task.Schedule,
			Parameters:  task.Parameters,
			Description: task.Description,
			Priority:    task.Priority,
			Enabled:     task.Enabled,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
			NextRun:     task.NextRun,
			LastRun:     task.LastRun,
		})
	}

	return taskInfos, nil
}

// RunScheduledTask executes a scheduled task immediately
func (c *BaseClient) RunScheduledTask(taskID string) (*TaskResult, error) {
	if !c.IsTasksEnabled() {
		return nil, fmt.Errorf("task scheduler is not enabled")
	}

	result, err := c.taskClient.scheduler.RunTask(taskID)
	if err != nil {
		return nil, err
	}

	return &TaskResult{
		Success:  result.Success,
		Output:   result.Output,
		Error:    result.Error,
		Duration: result.Duration,
	}, nil
}

// UpdateTask updates an existing task
func (c *BaseClient) UpdateTask(taskID string, options TaskOptions) error {
	if !c.IsTasksEnabled() {
		return fmt.Errorf("task scheduler is not enabled")
	}

	// Get existing task first
	existingTask, err := c.taskClient.scheduler.GetTask(taskID)
	if err != nil {
		return err
	}

	// Update task properties
	existingTask.Type = options.Type
	existingTask.Schedule = options.Schedule
	existingTask.Parameters = options.Parameters
	existingTask.Description = options.Description
	existingTask.Enabled = options.Enabled

	return c.taskClient.scheduler.UpdateTask(existingTask)
}

// DeleteTask deletes a task
func (c *BaseClient) DeleteTask(taskID string) error {
	if !c.IsTasksEnabled() {
		return fmt.Errorf("task scheduler is not enabled")
	}

	return c.taskClient.scheduler.DeleteTask(taskID)
}

// EnableTask enables a task
func (c *BaseClient) EnableTask(taskID string) error {
	if !c.IsTasksEnabled() {
		return fmt.Errorf("task scheduler is not enabled")
	}

	return c.taskClient.scheduler.EnableTask(taskID, true)
}

// DisableTask disables a task
func (c *BaseClient) DisableTask(taskID string) error {
	if !c.IsTasksEnabled() {
		return fmt.Errorf("task scheduler is not enabled")
	}

	return c.taskClient.scheduler.EnableTask(taskID, false)
}

// GetTaskExecutions retrieves execution history for a task
func (c *BaseClient) GetTaskExecutions(taskID string, limit int) ([]*TaskExecution, error) {
	if !c.IsTasksEnabled() {
		return nil, fmt.Errorf("task scheduler is not enabled")
	}

	executions, err := c.taskClient.scheduler.GetTaskExecutions(taskID, limit)
	if err != nil {
		return nil, err
	}

	var result []*TaskExecution
	for _, exec := range executions {
		result = append(result, &TaskExecution{
			ID:        exec.ID,
			TaskID:    exec.TaskID,
			StartTime: exec.StartTime,
			EndTime:   exec.EndTime,
			Duration:  exec.Duration,
			Status:    string(exec.Status),
			Output:    exec.Output,
			Error:     exec.Error,
		})
	}

	return result, nil
}

// CreateQueryTask creates a RAG query task
func (c *BaseClient) CreateQueryTask(query string, schedule string, options map[string]string) (string, error) {
	params := map[string]string{
		"query": query,
	}

	// Merge additional options
	for k, v := range options {
		params[k] = v
	}

	return c.CreateTask(TaskOptions{
		Type:        "query",
		Schedule:    schedule,
		Parameters:  params,
		Description: fmt.Sprintf("RAG Query: %s", query),
		Enabled:     true,
	})
}

// CreateScriptTask creates a script execution task
func (c *BaseClient) CreateScriptTask(script string, schedule string, options map[string]string) (string, error) {
	params := map[string]string{
		"script": script,
	}

	// Merge additional options
	for k, v := range options {
		params[k] = v
	}

	return c.CreateTask(TaskOptions{
		Type:        "script",
		Schedule:    schedule,
		Parameters:  params,
		Description: fmt.Sprintf("Script Task: %s", script[:min(50, len(script))]),
		Enabled:     true,
	})
}

// CreateIngestTask creates a document ingestion task
func (c *BaseClient) CreateIngestTask(path string, schedule string, options map[string]string) (string, error) {
	params := map[string]string{
		"path": path,
	}

	// Merge additional options
	for k, v := range options {
		params[k] = v
	}

	return c.CreateTask(TaskOptions{
		Type:        "ingest",
		Schedule:    schedule,
		Parameters:  params,
		Description: fmt.Sprintf("Ingest Task: %s", path),
		Enabled:     true,
	})
}

// CreateMCPTask creates an MCP tool execution task
func (c *BaseClient) CreateMCPTask(toolName string, arguments map[string]interface{}, schedule string) (string, error) {
	params := map[string]string{
		"tool": toolName,
	}

	// Convert arguments to string parameters
	for k, v := range arguments {
		params[fmt.Sprintf("arg_%s", k)] = fmt.Sprintf("%v", v)
	}

	return c.CreateTask(TaskOptions{
		Type:        "mcp",
		Schedule:    schedule,
		Parameters:  params,
		Description: fmt.Sprintf("MCP Task: %s", toolName),
		Enabled:     true,
	})
}

// Helper function for min (Go doesn't have it built-in for integers)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
