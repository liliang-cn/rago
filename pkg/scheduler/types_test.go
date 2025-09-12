package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskType(t *testing.T) {
	tests := []struct {
		name     string
		taskType TaskType
		expected string
	}{
		{"Query task type", TaskTypeQuery, "query"},
		{"Ingest task type", TaskTypeIngest, "ingest"},
		{"MCP task type", TaskTypeMCP, "mcp"},
		{"Script task type", TaskTypeScript, "script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.taskType))
		})
	}
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		expected string
	}{
		{"Pending status", TaskStatusPending, "pending"},
		{"Running status", TaskStatusRunning, "running"},
		{"Completed status", TaskStatusCompleted, "completed"},
		{"Failed status", TaskStatusFailed, "failed"},
		{"Cancelled status", TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestTask(t *testing.T) {
	now := time.Now()
	nextRun := now.Add(time.Hour)
	lastRun := now.Add(-time.Hour)

	task := &Task{
		ID:          "test-task-1",
		Type:        string(TaskTypeQuery),
		Schedule:    "0 * * * *",
		Parameters:  map[string]string{"query": "test query"},
		Description: "Test task",
		Priority:    1,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		NextRun:     &nextRun,
		LastRun:     &lastRun,
	}

	assert.Equal(t, "test-task-1", task.ID)
	assert.Equal(t, string(TaskTypeQuery), task.Type)
	assert.Equal(t, "0 * * * *", task.Schedule)
	assert.Equal(t, "test query", task.Parameters["query"])
	assert.Equal(t, "Test task", task.Description)
	assert.Equal(t, 1, task.Priority)
	assert.True(t, task.Enabled)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, now, task.UpdatedAt)
	assert.NotNil(t, task.NextRun)
	assert.Equal(t, nextRun, *task.NextRun)
	assert.NotNil(t, task.LastRun)
	assert.Equal(t, lastRun, *task.LastRun)
}

func TestTaskExecution(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)
	duration := 5 * time.Second

	execution := &TaskExecution{
		ID:        1,
		TaskID:    "test-task-1",
		StartTime: startTime,
		EndTime:   &endTime,
		Duration:  duration,
		Status:    TaskStatusCompleted,
		Output:    "Task completed successfully",
		Error:     "",
	}

	assert.Equal(t, int64(1), execution.ID)
	assert.Equal(t, "test-task-1", execution.TaskID)
	assert.Equal(t, startTime, execution.StartTime)
	assert.NotNil(t, execution.EndTime)
	assert.Equal(t, endTime, *execution.EndTime)
	assert.Equal(t, duration, execution.Duration)
	assert.Equal(t, TaskStatusCompleted, execution.Status)
	assert.Equal(t, "Task completed successfully", execution.Output)
	assert.Empty(t, execution.Error)
}

func TestTaskResult(t *testing.T) {
	tests := []struct {
		name     string
		result   TaskResult
		expected bool
	}{
		{
			name: "Successful result",
			result: TaskResult{
				Success:  true,
				Output:   "Query executed successfully",
				Error:    "",
				Duration: 100 * time.Millisecond,
			},
			expected: true,
		},
		{
			name: "Failed result",
			result: TaskResult{
				Success:  false,
				Output:   "",
				Error:    "Query failed: connection timeout",
				Duration: 5 * time.Second,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Success)
			if tt.result.Success {
				assert.NotEmpty(t, tt.result.Output)
				assert.Empty(t, tt.result.Error)
			} else {
				assert.NotEmpty(t, tt.result.Error)
			}
			assert.Greater(t, tt.result.Duration, time.Duration(0))
		})
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	require.NotNil(t, config)
	assert.Equal(t, "./data/scheduler.db", config.DatabasePath)
	assert.Equal(t, 5, config.MaxConcurrentTasks)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 5*time.Minute, config.RetryDelay)
	assert.Equal(t, 24*time.Hour, config.CleanupInterval)
	assert.Equal(t, 100, config.MaxExecutionHistory)
}

func TestSchedulerConfigCustomization(t *testing.T) {
	config := &SchedulerConfig{
		DatabasePath:        "/custom/path/scheduler.db",
		MaxConcurrentTasks:  10,
		RetryAttempts:       5,
		RetryDelay:          10 * time.Minute,
		CleanupInterval:     12 * time.Hour,
		MaxExecutionHistory: 200,
	}

	assert.Equal(t, "/custom/path/scheduler.db", config.DatabasePath)
	assert.Equal(t, 10, config.MaxConcurrentTasks)
	assert.Equal(t, 5, config.RetryAttempts)
	assert.Equal(t, 10*time.Minute, config.RetryDelay)
	assert.Equal(t, 12*time.Hour, config.CleanupInterval)
	assert.Equal(t, 200, config.MaxExecutionHistory)
}