package scheduler

import (
	"testing"
	"time"
)

func TestTaskTypeConstants(t *testing.T) {
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
			if string(tt.taskType) != tt.expected {
				t.Errorf("TaskType %s = %s, want %s", tt.name, string(tt.taskType), tt.expected)
			}
		})
	}
}

func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		name       string
		taskStatus TaskStatus
		expected   string
	}{
		{"Pending status", TaskStatusPending, "pending"},
		{"Running status", TaskStatusRunning, "running"},
		{"Completed status", TaskStatusCompleted, "completed"},
		{"Failed status", TaskStatusFailed, "failed"},
		{"Cancelled status", TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.taskStatus) != tt.expected {
				t.Errorf("TaskStatus %s = %s, want %s", tt.name, string(tt.taskStatus), tt.expected)
			}
		})
	}
}

func TestTaskStructFields(t *testing.T) {
	now := time.Now()
	nextRun := now.Add(time.Hour)
	lastRun := now.Add(-time.Hour)

	task := &Task{
		ID:          "test-id",
		Type:        "query",
		Schedule:    "@daily",
		Parameters:  map[string]string{"param1": "value1"},
		Description: "Test task",
		Priority:    5,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		NextRun:     &nextRun,
		LastRun:     &lastRun,
	}

	// Verify all fields are accessible and correctly typed
	if task.ID != "test-id" {
		t.Errorf("Task.ID = %s, want test-id", task.ID)
	}

	if task.Type != "query" {
		t.Errorf("Task.Type = %s, want query", task.Type)
	}

	if task.Schedule != "@daily" {
		t.Errorf("Task.Schedule = %s, want @daily", task.Schedule)
	}

	if task.Parameters["param1"] != "value1" {
		t.Errorf("Task.Parameters[param1] = %s, want value1", task.Parameters["param1"])
	}

	if task.Description != "Test task" {
		t.Errorf("Task.Description = %s, want Test task", task.Description)
	}

	if task.Priority != 5 {
		t.Errorf("Task.Priority = %d, want 5", task.Priority)
	}

	if !task.Enabled {
		t.Errorf("Task.Enabled = %v, want true", task.Enabled)
	}

	if task.CreatedAt != now {
		t.Errorf("Task.CreatedAt = %v, want %v", task.CreatedAt, now)
	}

	if task.UpdatedAt != now {
		t.Errorf("Task.UpdatedAt = %v, want %v", task.UpdatedAt, now)
	}

	if task.NextRun == nil || !task.NextRun.Equal(nextRun) {
		t.Errorf("Task.NextRun = %v, want %v", task.NextRun, nextRun)
	}

	if task.LastRun == nil || !task.LastRun.Equal(lastRun) {
		t.Errorf("Task.LastRun = %v, want %v", task.LastRun, lastRun)
	}
}

func TestTaskExecutionStructFields(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Second * 30)
	duration := time.Second * 30

	execution := &TaskExecution{
		ID:        123,
		TaskID:    "task-123",
		StartTime: start,
		EndTime:   &end,
		Duration:  duration,
		Status:    TaskStatusCompleted,
		Output:    "Task completed successfully",
		Error:     "",
	}

	// Verify all fields are accessible and correctly typed
	if execution.ID != 123 {
		t.Errorf("TaskExecution.ID = %d, want 123", execution.ID)
	}

	if execution.TaskID != "task-123" {
		t.Errorf("TaskExecution.TaskID = %s, want task-123", execution.TaskID)
	}

	if execution.StartTime != start {
		t.Errorf("TaskExecution.StartTime = %v, want %v", execution.StartTime, start)
	}

	if execution.EndTime == nil || !execution.EndTime.Equal(end) {
		t.Errorf("TaskExecution.EndTime = %v, want %v", execution.EndTime, end)
	}

	if execution.Duration != duration {
		t.Errorf("TaskExecution.Duration = %v, want %v", execution.Duration, duration)
	}

	if execution.Status != TaskStatusCompleted {
		t.Errorf("TaskExecution.Status = %s, want %s", execution.Status, TaskStatusCompleted)
	}

	if execution.Output != "Task completed successfully" {
		t.Errorf("TaskExecution.Output = %s, want 'Task completed successfully'", execution.Output)
	}

	if execution.Error != "" {
		t.Errorf("TaskExecution.Error = %s, want empty string", execution.Error)
	}
}

func TestTaskResultStructFields(t *testing.T) {
	duration := time.Second * 15

	result := &TaskResult{
		Success:  true,
		Output:   "Operation successful",
		Error:    "",
		Duration: duration,
	}

	// Verify all fields are accessible and correctly typed
	if !result.Success {
		t.Errorf("TaskResult.Success = %v, want true", result.Success)
	}

	if result.Output != "Operation successful" {
		t.Errorf("TaskResult.Output = %s, want 'Operation successful'", result.Output)
	}

	if result.Error != "" {
		t.Errorf("TaskResult.Error = %s, want empty string", result.Error)
	}

	if result.Duration != duration {
		t.Errorf("TaskResult.Duration = %v, want %v", result.Duration, duration)
	}

	// Test failed result
	failedResult := &TaskResult{
		Success:  false,
		Output:   "",
		Error:    "Task failed with error",
		Duration: duration,
	}

	if failedResult.Success {
		t.Errorf("Failed TaskResult.Success = %v, want false", failedResult.Success)
	}

	if failedResult.Error != "Task failed with error" {
		t.Errorf("Failed TaskResult.Error = %s, want 'Task failed with error'", failedResult.Error)
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	// Verify config is not nil
	if config == nil {
		t.Fatal("DefaultSchedulerConfig() returned nil")
	}

	// Verify default values are reasonable
	if config.DatabasePath == "" {
		t.Error("DefaultSchedulerConfig.DatabasePath is empty")
	}

	if config.DatabasePath != "./data/scheduler.db" {
		t.Errorf("DefaultSchedulerConfig.DatabasePath = %s, want ./data/scheduler.db", config.DatabasePath)
	}

	if config.MaxConcurrentTasks <= 0 {
		t.Errorf("DefaultSchedulerConfig.MaxConcurrentTasks = %d, want > 0", config.MaxConcurrentTasks)
	}

	if config.MaxConcurrentTasks != 5 {
		t.Errorf("DefaultSchedulerConfig.MaxConcurrentTasks = %d, want 5", config.MaxConcurrentTasks)
	}

	if config.RetryAttempts < 0 {
		t.Errorf("DefaultSchedulerConfig.RetryAttempts = %d, want >= 0", config.RetryAttempts)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("DefaultSchedulerConfig.RetryAttempts = %d, want 3", config.RetryAttempts)
	}

	if config.RetryDelay <= 0 {
		t.Errorf("DefaultSchedulerConfig.RetryDelay = %v, want > 0", config.RetryDelay)
	}

	expectedRetryDelay := time.Minute * 5
	if config.RetryDelay != expectedRetryDelay {
		t.Errorf("DefaultSchedulerConfig.RetryDelay = %v, want %v", config.RetryDelay, expectedRetryDelay)
	}

	if config.CleanupInterval <= 0 {
		t.Errorf("DefaultSchedulerConfig.CleanupInterval = %v, want > 0", config.CleanupInterval)
	}

	expectedCleanupInterval := time.Hour * 24
	if config.CleanupInterval != expectedCleanupInterval {
		t.Errorf("DefaultSchedulerConfig.CleanupInterval = %v, want %v", config.CleanupInterval, expectedCleanupInterval)
	}

	if config.MaxExecutionHistory <= 0 {
		t.Errorf("DefaultSchedulerConfig.MaxExecutionHistory = %d, want > 0", config.MaxExecutionHistory)
	}

	if config.MaxExecutionHistory != 100 {
		t.Errorf("DefaultSchedulerConfig.MaxExecutionHistory = %d, want 100", config.MaxExecutionHistory)
	}
}

func TestSchedulerConfigFields(t *testing.T) {
	config := &SchedulerConfig{
		DatabasePath:        "/custom/path/scheduler.db",
		MaxConcurrentTasks:  10,
		RetryAttempts:       5,
		RetryDelay:          time.Minute * 2,
		CleanupInterval:     time.Hour * 12,
		MaxExecutionHistory: 200,
	}

	// Verify all fields are accessible and correctly typed
	if config.DatabasePath != "/custom/path/scheduler.db" {
		t.Errorf("SchedulerConfig.DatabasePath = %s, want /custom/path/scheduler.db", config.DatabasePath)
	}

	if config.MaxConcurrentTasks != 10 {
		t.Errorf("SchedulerConfig.MaxConcurrentTasks = %d, want 10", config.MaxConcurrentTasks)
	}

	if config.RetryAttempts != 5 {
		t.Errorf("SchedulerConfig.RetryAttempts = %d, want 5", config.RetryAttempts)
	}

	expectedRetryDelay := time.Minute * 2
	if config.RetryDelay != expectedRetryDelay {
		t.Errorf("SchedulerConfig.RetryDelay = %v, want %v", config.RetryDelay, expectedRetryDelay)
	}

	expectedCleanupInterval := time.Hour * 12
	if config.CleanupInterval != expectedCleanupInterval {
		t.Errorf("SchedulerConfig.CleanupInterval = %v, want %v", config.CleanupInterval, expectedCleanupInterval)
	}

	if config.MaxExecutionHistory != 200 {
		t.Errorf("SchedulerConfig.MaxExecutionHistory = %d, want 200", config.MaxExecutionHistory)
	}
}

func TestNullTimeScan(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectValid bool
		expectError bool
		expectTime  time.Time
	}{
		{
			name:        "Nil value",
			value:       nil,
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Time value",
			value:       time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			expectValid: true,
			expectError: false,
			expectTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
		},
		{
			name:        "String RFC3339 value",
			value:       "2023-12-25T15:30:00Z",
			expectValid: true,
			expectError: false,
			expectTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
		},
		{
			name:        "Invalid string value",
			value:       "not-a-time",
			expectValid: true, // Valid flag is set before parsing
			expectError: true,
		},
		{
			name:        "Invalid type",
			value:       123,
			expectValid: true, // Valid flag is set before type checking
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nt NullTime
			err := nt.Scan(tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("NullTime.Scan(%v) expected error, got nil", tt.value)
				}
				return
			}

			if err != nil {
				t.Errorf("NullTime.Scan(%v) unexpected error: %v", tt.value, err)
				return
			}

			if nt.Valid != tt.expectValid {
				t.Errorf("NullTime.Valid = %v, want %v", nt.Valid, tt.expectValid)
			}

			if tt.expectValid && !nt.Time.Equal(tt.expectTime) {
				t.Errorf("NullTime.Time = %v, want %v", nt.Time, tt.expectTime)
			}
		})
	}
}

func TestNullTimeScanEdgeCases(t *testing.T) {
	// Test empty string
	var nt NullTime
	err := nt.Scan("")
	if err == nil {
		t.Error("NullTime.Scan(\"\") expected error for empty string, got nil")
	}

	// Test zero time
	zeroTime := time.Time{}
	nt = NullTime{}
	err = nt.Scan(zeroTime)
	if err != nil {
		t.Errorf("NullTime.Scan(zero time) unexpected error: %v", err)
	}
	if !nt.Valid {
		t.Error("NullTime.Valid should be true for zero time")
	}
	if !nt.Time.Equal(zeroTime) {
		t.Errorf("NullTime.Time = %v, want %v", nt.Time, zeroTime)
	}

	// Test with timezone string
	nt = NullTime{}
	err = nt.Scan("2023-12-25T15:30:00+05:00")
	if err != nil {
		t.Errorf("NullTime.Scan(timezone string) unexpected error: %v", err)
	}
	if !nt.Valid {
		t.Error("NullTime.Valid should be true for valid timezone string")
	}

	expectedTime := time.Date(2023, 12, 25, 15, 30, 0, 0, time.FixedZone("+05:00", 5*60*60))
	if !nt.Time.Equal(expectedTime) {
		t.Errorf("NullTime.Time = %v, want %v", nt.Time, expectedTime)
	}
}