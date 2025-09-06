package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "Valid in-memory database",
			dbPath:  ":memory:",
			wantErr: false,
		},
		{
			name:    "Valid file database in temp directory",
			dbPath:  filepath.Join(os.TempDir(), "test_scheduler.db"),
			wantErr: false,
		},
		{
			name:    "Invalid directory path",
			dbPath:  "/nonexistent/directory/test.db",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(tt.dbPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewStorage(%s) expected error, got nil", tt.dbPath)
					if storage != nil {
						storage.Close()
					}
				}
				return
			}

			if err != nil {
				t.Errorf("NewStorage(%s) unexpected error: %v", tt.dbPath, err)
				return
			}

			if storage == nil {
				t.Errorf("NewStorage(%s) returned nil storage", tt.dbPath)
				return
			}

			// Clean up
			if err := storage.Close(); err != nil {
				t.Errorf("Failed to close storage: %v", err)
			}

			// Clean up file if it was created
			if tt.dbPath != ":memory:" {
				os.Remove(tt.dbPath)
			}
		})
	}
}

func TestStorageClose(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// First close should succeed
	err = storage.Close()
	if err != nil {
		t.Errorf("First Close() unexpected error: %v", err)
	}

	// Second close should not panic (idempotent)
	err = storage.Close()
	if err == nil {
		t.Log("Second Close() succeeded (database already closed)")
	} else {
		t.Logf("Second Close() error (expected): %v", err)
	}
}

func TestTaskCRUDOperations(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Test data
	now := time.Now()
	nextRun := now.Add(time.Hour)
	
	task := &Task{
		ID:          "test-task-123",
		Type:        "query",
		Schedule:    "@daily",
		Parameters:  map[string]string{"query": "test query", "limit": "10"},
		Description: "Test task for CRUD operations",
		Priority:    5,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		NextRun:     &nextRun,
		LastRun:     nil,
	}

	t.Run("CreateTask", func(t *testing.T) {
		err := storage.CreateTask(task)
		if err != nil {
			t.Errorf("CreateTask() unexpected error: %v", err)
		}
	})

	t.Run("GetTask", func(t *testing.T) {
		retrieved, err := storage.GetTask(task.ID)
		if err != nil {
			t.Errorf("GetTask() unexpected error: %v", err)
			return
		}

		if retrieved == nil {
			t.Error("GetTask() returned nil task")
			return
		}

		// Verify all fields
		if retrieved.ID != task.ID {
			t.Errorf("GetTask() ID = %s, want %s", retrieved.ID, task.ID)
		}

		if retrieved.Type != task.Type {
			t.Errorf("GetTask() Type = %s, want %s", retrieved.Type, task.Type)
		}

		if retrieved.Schedule != task.Schedule {
			t.Errorf("GetTask() Schedule = %s, want %s", retrieved.Schedule, task.Schedule)
		}

		if len(retrieved.Parameters) != len(task.Parameters) {
			t.Errorf("GetTask() Parameters length = %d, want %d", len(retrieved.Parameters), len(task.Parameters))
		} else {
			for key, value := range task.Parameters {
				if retrieved.Parameters[key] != value {
					t.Errorf("GetTask() Parameters[%s] = %s, want %s", key, retrieved.Parameters[key], value)
				}
			}
		}

		if retrieved.Description != task.Description {
			t.Errorf("GetTask() Description = %s, want %s", retrieved.Description, task.Description)
		}

		if retrieved.Priority != task.Priority {
			t.Errorf("GetTask() Priority = %d, want %d", retrieved.Priority, task.Priority)
		}

		if retrieved.Enabled != task.Enabled {
			t.Errorf("GetTask() Enabled = %v, want %v", retrieved.Enabled, task.Enabled)
		}

		// Time comparisons (allow small differences due to precision)
		if !retrieved.CreatedAt.Round(time.Second).Equal(task.CreatedAt.Round(time.Second)) {
			t.Errorf("GetTask() CreatedAt = %v, want %v", retrieved.CreatedAt, task.CreatedAt)
		}

		if retrieved.NextRun == nil || !retrieved.NextRun.Round(time.Second).Equal(task.NextRun.Round(time.Second)) {
			t.Errorf("GetTask() NextRun = %v, want %v", retrieved.NextRun, task.NextRun)
		}

		if retrieved.LastRun != nil {
			t.Errorf("GetTask() LastRun = %v, want nil", retrieved.LastRun)
		}
	})

	t.Run("UpdateTask", func(t *testing.T) {
		// Modify task
		task.Description = "Updated description"
		task.Priority = 10
		task.Enabled = false
		lastRun := now.Add(-time.Hour)
		task.LastRun = &lastRun

		err := storage.UpdateTask(task)
		if err != nil {
			t.Errorf("UpdateTask() unexpected error: %v", err)
			return
		}

		// Retrieve and verify
		updated, err := storage.GetTask(task.ID)
		if err != nil {
			t.Errorf("GetTask() after update unexpected error: %v", err)
			return
		}

		if updated.Description != "Updated description" {
			t.Errorf("UpdateTask() Description = %s, want 'Updated description'", updated.Description)
		}

		if updated.Priority != 10 {
			t.Errorf("UpdateTask() Priority = %d, want 10", updated.Priority)
		}

		if updated.Enabled != false {
			t.Errorf("UpdateTask() Enabled = %v, want false", updated.Enabled)
		}

		if updated.LastRun == nil || !updated.LastRun.Round(time.Second).Equal(lastRun.Round(time.Second)) {
			t.Errorf("UpdateTask() LastRun = %v, want %v", updated.LastRun, lastRun)
		}
	})

	t.Run("EnableTask", func(t *testing.T) {
		// Enable the task
		err := storage.EnableTask(task.ID, true)
		if err != nil {
			t.Errorf("EnableTask(true) unexpected error: %v", err)
			return
		}

		// Verify
		enabled, err := storage.GetTask(task.ID)
		if err != nil {
			t.Errorf("GetTask() after enable unexpected error: %v", err)
			return
		}

		if !enabled.Enabled {
			t.Error("EnableTask(true) task should be enabled")
		}

		// Disable the task
		err = storage.EnableTask(task.ID, false)
		if err != nil {
			t.Errorf("EnableTask(false) unexpected error: %v", err)
			return
		}

		// Verify
		disabled, err := storage.GetTask(task.ID)
		if err != nil {
			t.Errorf("GetTask() after disable unexpected error: %v", err)
			return
		}

		if disabled.Enabled {
			t.Error("EnableTask(false) task should be disabled")
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		err := storage.DeleteTask(task.ID)
		if err != nil {
			t.Errorf("DeleteTask() unexpected error: %v", err)
			return
		}

		// Verify task is deleted
		_, err = storage.GetTask(task.ID)
		if err == nil {
			t.Error("GetTask() after delete should return error")
		}
	})
}

func TestTaskCRUDErrors(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("GetNonexistentTask", func(t *testing.T) {
		_, err := storage.GetTask("nonexistent-id")
		if err == nil {
			t.Error("GetTask(nonexistent) should return error")
		}
	})

	t.Run("UpdateNonexistentTask", func(t *testing.T) {
		task := &Task{
			ID:          "nonexistent-id",
			Type:        "query",
			Schedule:    "@daily",
			Parameters:  map[string]string{},
			Description: "Nonexistent task",
			Priority:    1,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := storage.UpdateTask(task)
		if err == nil {
			t.Error("UpdateTask(nonexistent) should return error")
		}
	})

	t.Run("EnableNonexistentTask", func(t *testing.T) {
		err := storage.EnableTask("nonexistent-id", true)
		if err == nil {
			t.Error("EnableTask(nonexistent) should return error")
		}
	})

	t.Run("DeleteNonexistentTask", func(t *testing.T) {
		err := storage.DeleteTask("nonexistent-id")
		if err == nil {
			t.Error("DeleteTask(nonexistent) should return error")
		}
	})
}

func TestListTasks(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	now := time.Now()

	// Create test tasks
	tasks := []*Task{
		{
			ID:          "task-1",
			Type:        "query",
			Schedule:    "@daily",
			Parameters:  map[string]string{"query": "test1"},
			Description: "Task 1",
			Priority:    10,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "task-2",
			Type:        "ingest",
			Schedule:    "@hourly",
			Parameters:  map[string]string{"file": "test2.txt"},
			Description: "Task 2",
			Priority:    5,
			Enabled:     false,
			CreatedAt:   now.Add(-time.Hour),
			UpdatedAt:   now.Add(-time.Hour),
		},
		{
			ID:          "task-3",
			Type:        "mcp",
			Schedule:    "*/5 * * * *",
			Parameters:  map[string]string{"tool": "test"},
			Description: "Task 3",
			Priority:    15,
			Enabled:     true,
			CreatedAt:   now.Add(-time.Hour*2),
			UpdatedAt:   now.Add(-time.Hour*2),
		},
	}

	// Insert tasks
	for _, task := range tasks {
		if err := storage.CreateTask(task); err != nil {
			t.Fatalf("CreateTask(%s) failed: %v", task.ID, err)
		}
	}

	t.Run("ListTasksEnabledOnly", func(t *testing.T) {
		retrieved, err := storage.ListTasks(false)
		if err != nil {
			t.Errorf("ListTasks(false) unexpected error: %v", err)
			return
		}

		// Should return only enabled tasks (task-1 and task-3)
		if len(retrieved) != 2 {
			t.Errorf("ListTasks(false) returned %d tasks, want 2", len(retrieved))
			return
		}

		// Verify order (highest priority first, then by creation time)
		if retrieved[0].ID != "task-3" || retrieved[1].ID != "task-1" {
			t.Errorf("ListTasks(false) wrong order: [%s, %s], want [task-3, task-1]", 
				retrieved[0].ID, retrieved[1].ID)
		}

		// Verify all are enabled
		for _, task := range retrieved {
			if !task.Enabled {
				t.Errorf("ListTasks(false) returned disabled task: %s", task.ID)
			}
		}
	})

	t.Run("ListTasksIncludeDisabled", func(t *testing.T) {
		retrieved, err := storage.ListTasks(true)
		if err != nil {
			t.Errorf("ListTasks(true) unexpected error: %v", err)
			return
		}

		// Should return all tasks
		if len(retrieved) != 3 {
			t.Errorf("ListTasks(true) returned %d tasks, want 3", len(retrieved))
			return
		}

		// Verify order (highest priority first, then by creation time)
		expectedOrder := []string{"task-3", "task-1", "task-2"}
		for i, task := range retrieved {
			if task.ID != expectedOrder[i] {
				t.Errorf("ListTasks(true) position %d: got %s, want %s", i, task.ID, expectedOrder[i])
			}
		}
	})

	t.Run("ListTasksEmpty", func(t *testing.T) {
		// Delete all tasks
		for _, task := range tasks {
			if err := storage.DeleteTask(task.ID); err != nil {
				t.Fatalf("DeleteTask(%s) failed: %v", task.ID, err)
			}
		}

		retrieved, err := storage.ListTasks(true)
		if err != nil {
			t.Errorf("ListTasks(true) on empty database unexpected error: %v", err)
			return
		}

		if len(retrieved) != 0 {
			t.Errorf("ListTasks(true) on empty database returned %d tasks, want 0", len(retrieved))
		}
	})
}

func TestExecutionCRUD(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create a task first
	task := &Task{
		ID:          "exec-test-task",
		Type:        "query",
		Schedule:    "@daily",
		Parameters:  map[string]string{"query": "test"},
		Description: "Execution test task",
		Priority:    1,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := storage.CreateTask(task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	start := time.Now()
	execution := &TaskExecution{
		TaskID:    task.ID,
		StartTime: start,
		Status:    TaskStatusRunning,
	}

	t.Run("CreateExecution", func(t *testing.T) {
		err := storage.CreateExecution(execution)
		if err != nil {
			t.Errorf("CreateExecution() unexpected error: %v", err)
			return
		}

		// ID should be set after creation
		if execution.ID == 0 {
			t.Error("CreateExecution() should set execution ID")
		}
	})

	t.Run("UpdateExecution", func(t *testing.T) {
		// Complete the execution
		end := start.Add(time.Second * 30)
		execution.EndTime = &end
		execution.Duration = time.Second * 30
		execution.Status = TaskStatusCompleted
		execution.Output = "Execution completed successfully"

		err := storage.UpdateExecution(execution)
		if err != nil {
			t.Errorf("UpdateExecution() unexpected error: %v", err)
		}
	})

	t.Run("GetTaskExecutions", func(t *testing.T) {
		executions, err := storage.GetTaskExecutions(task.ID, 10)
		if err != nil {
			t.Errorf("GetTaskExecutions() unexpected error: %v", err)
			return
		}

		if len(executions) != 1 {
			t.Errorf("GetTaskExecutions() returned %d executions, want 1", len(executions))
			return
		}

		retrieved := executions[0]

		// Verify fields
		if retrieved.TaskID != execution.TaskID {
			t.Errorf("GetTaskExecutions() TaskID = %s, want %s", retrieved.TaskID, execution.TaskID)
		}

		if retrieved.Status != TaskStatusCompleted {
			t.Errorf("GetTaskExecutions() Status = %s, want %s", retrieved.Status, TaskStatusCompleted)
		}

		if retrieved.Output != "Execution completed successfully" {
			t.Errorf("GetTaskExecutions() Output = %s, want 'Execution completed successfully'", retrieved.Output)
		}

		if retrieved.EndTime == nil {
			t.Error("GetTaskExecutions() EndTime should not be nil")
		} else if !retrieved.EndTime.Round(time.Second).Equal(execution.EndTime.Round(time.Second)) {
			t.Errorf("GetTaskExecutions() EndTime = %v, want %v", retrieved.EndTime, execution.EndTime)
		}

		if retrieved.Duration != time.Second*30 {
			t.Errorf("GetTaskExecutions() Duration = %v, want %v", retrieved.Duration, time.Second*30)
		}
	})

	t.Run("GetTaskExecutionsWithLimit", func(t *testing.T) {
		// Create more executions
		for i := 0; i < 5; i++ {
			exec := &TaskExecution{
				TaskID:    task.ID,
				StartTime: start.Add(time.Minute * time.Duration(i+1)),
				Status:    TaskStatusCompleted,
				Output:    "Test execution",
			}
			
			if err := storage.CreateExecution(exec); err != nil {
				t.Fatalf("CreateExecution %d failed: %v", i, err)
			}
		}

		// Test limit
		executions, err := storage.GetTaskExecutions(task.ID, 3)
		if err != nil {
			t.Errorf("GetTaskExecutions(limit=3) unexpected error: %v", err)
			return
		}

		if len(executions) != 3 {
			t.Errorf("GetTaskExecutions(limit=3) returned %d executions, want 3", len(executions))
			return
		}

		// Verify order (most recent first)
		for i := 1; i < len(executions); i++ {
			if executions[i].StartTime.After(executions[i-1].StartTime) {
				t.Error("GetTaskExecutions() should return executions in reverse chronological order")
			}
		}
	})
}

func TestSchedulingQueries(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	now := time.Now()

	// Create test tasks
	tasks := []*Task{
		{
			ID:          "due-task-1",
			Type:        "query",
			Schedule:    "@daily",
			Parameters:  map[string]string{},
			Description: "Due task 1",
			Priority:    10,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
			NextRun:     &[]time.Time{now.Add(-time.Hour)}[0], // Past due
		},
		{
			ID:          "due-task-2",
			Type:        "ingest",
			Schedule:    "@hourly",
			Parameters:  map[string]string{},
			Description: "Due task 2",
			Priority:    5,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
			NextRun:     &[]time.Time{now.Add(-time.Minute)}[0], // Past due
		},
		{
			ID:          "future-task",
			Type:        "mcp",
			Schedule:    "@daily",
			Parameters:  map[string]string{},
			Description: "Future task",
			Priority:    15,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
			NextRun:     &[]time.Time{now.Add(time.Hour)}[0], // Future
		},
		{
			ID:          "disabled-task",
			Type:        "script",
			Schedule:    "@hourly",
			Parameters:  map[string]string{},
			Description: "Disabled task",
			Priority:    20,
			Enabled:     false,
			CreatedAt:   now,
			UpdatedAt:   now,
			NextRun:     &[]time.Time{now.Add(-time.Hour)}[0], // Past due but disabled
		},
		{
			ID:          "immediate-task",
			Type:        "query",
			Schedule:    "",
			Parameters:  map[string]string{},
			Description: "Immediate task",
			Priority:    1,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
			NextRun:     nil, // No schedule - immediate
		},
	}

	// Insert tasks
	for _, task := range tasks {
		if err := storage.CreateTask(task); err != nil {
			t.Fatalf("CreateTask(%s) failed: %v", task.ID, err)
		}
	}

	t.Run("GetTasksDueForExecution", func(t *testing.T) {
		dueTasks, err := storage.GetTasksDueForExecution()
		if err != nil {
			t.Errorf("GetTasksDueForExecution() unexpected error: %v", err)
			return
		}

		// Should return due-task-1, due-task-2, and immediate-task
		// Ordered by priority DESC, then next_run ASC
		expectedCount := 3
		if len(dueTasks) != expectedCount {
			t.Errorf("GetTasksDueForExecution() returned %d tasks, want %d", len(dueTasks), expectedCount)
			
			// Debug: show what was returned
			for i, task := range dueTasks {
				t.Logf("Task %d: ID=%s, Priority=%d, NextRun=%v, Enabled=%v", 
					i, task.ID, task.Priority, task.NextRun, task.Enabled)
			}
			return
		}

		// Verify order: highest priority first (due-task-1: 10, due-task-2: 5, immediate-task: 1)
		expectedOrder := []string{"due-task-1", "due-task-2", "immediate-task"}
		for i, task := range dueTasks {
			if task.ID != expectedOrder[i] {
				t.Errorf("GetTasksDueForExecution() position %d: got %s, want %s", 
					i, task.ID, expectedOrder[i])
			}
		}

		// Verify all returned tasks are enabled
		for _, task := range dueTasks {
			if !task.Enabled {
				t.Errorf("GetTasksDueForExecution() returned disabled task: %s", task.ID)
			}
		}
	})

	t.Run("UpdateTaskNextRun", func(t *testing.T) {
		nextRun := now.Add(time.Hour * 2)
		err := storage.UpdateTaskNextRun("due-task-1", &nextRun)
		if err != nil {
			t.Errorf("UpdateTaskNextRun() unexpected error: %v", err)
			return
		}

		// Verify update
		task, err := storage.GetTask("due-task-1")
		if err != nil {
			t.Errorf("GetTask() after UpdateTaskNextRun unexpected error: %v", err)
			return
		}

		if task.NextRun == nil || !task.NextRun.Round(time.Second).Equal(nextRun.Round(time.Second)) {
			t.Errorf("UpdateTaskNextRun() NextRun = %v, want %v", task.NextRun, nextRun)
		}
	})

	t.Run("UpdateTaskLastRun", func(t *testing.T) {
		lastRun := now.Add(-time.Minute * 30)
		err := storage.UpdateTaskLastRun("due-task-2", lastRun)
		if err != nil {
			t.Errorf("UpdateTaskLastRun() unexpected error: %v", err)
			return
		}

		// Verify update
		task, err := storage.GetTask("due-task-2")
		if err != nil {
			t.Errorf("GetTask() after UpdateTaskLastRun unexpected error: %v", err)
			return
		}

		if task.LastRun == nil || !task.LastRun.Round(time.Second).Equal(lastRun.Round(time.Second)) {
			t.Errorf("UpdateTaskLastRun() LastRun = %v, want %v", task.LastRun, lastRun)
		}
	})
}

func TestCleanupOldExecutions(t *testing.T) {
	storage, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	defer storage.Close()

	// Create test task
	task := &Task{
		ID:          "cleanup-test-task",
		Type:        "query",
		Schedule:    "@daily",
		Parameters:  map[string]string{},
		Description: "Cleanup test task",
		Priority:    1,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := storage.CreateTask(task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	now := time.Now()

	// Create old and new executions
	executions := []*TaskExecution{
		{
			TaskID:    task.ID,
			StartTime: now.Add(-time.Hour * 48), // 2 days old
			Status:    TaskStatusCompleted,
		},
		{
			TaskID:    task.ID,
			StartTime: now.Add(-time.Hour * 24), // 1 day old
			Status:    TaskStatusCompleted,
		},
		{
			TaskID:    task.ID,
			StartTime: now.Add(-time.Hour * 12), // 12 hours old
			Status:    TaskStatusCompleted,
		},
		{
			TaskID:    task.ID,
			StartTime: now.Add(-time.Hour * 1), // 1 hour old
			Status:    TaskStatusCompleted,
		},
		{
			TaskID:    task.ID,
			StartTime: now, // Current
			Status:    TaskStatusCompleted,
		},
	}

	// Insert executions
	for _, exec := range executions {
		if err := storage.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution failed: %v", err)
		}
	}

	t.Run("CleanupByAge", func(t *testing.T) {
		// Clean up executions older than 36 hours
		maxAge := time.Hour * 36
		
		err := storage.CleanupOldExecutions(maxAge, 0)
		if err != nil {
			t.Errorf("CleanupOldExecutions() unexpected error: %v", err)
			return
		}

		// Should keep 3 executions (1 day old, 12 hours old, 1 hour old, current)
		remaining, err := storage.GetTaskExecutions(task.ID, 0)
		if err != nil {
			t.Errorf("GetTaskExecutions() after cleanup unexpected error: %v", err)
			return
		}

		if len(remaining) != 4 {
			t.Errorf("CleanupOldExecutions() left %d executions, want 4", len(remaining))
		}
	})

	t.Run("CleanupByCount", func(t *testing.T) {
		// Keep only 2 most recent executions per task
		maxPerTask := 2
		
		err := storage.CleanupOldExecutions(0, maxPerTask)
		if err != nil {
			t.Errorf("CleanupOldExecutions() unexpected error: %v", err)
			return
		}

		// Should keep only 2 executions
		remaining, err := storage.GetTaskExecutions(task.ID, 0)
		if err != nil {
			t.Errorf("GetTaskExecutions() after cleanup unexpected error: %v", err)
			return
		}

		if len(remaining) != 2 {
			t.Errorf("CleanupOldExecutions() left %d executions, want 2", len(remaining))
		}

		// Verify we kept the most recent ones
		if len(remaining) >= 2 {
			// Should be in descending order by start time
			if remaining[0].StartTime.Before(remaining[1].StartTime) {
				t.Error("CleanupOldExecutions() didn't keep the most recent executions")
			}
		}
	})
}