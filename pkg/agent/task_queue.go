package agent

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// Task status constants
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// Task represents a queued task for LongRun
type Task struct {
	ID          string     `json:"id"`
	Goal        string     `json:"goal"`
	Type        string     `json:"type,omitempty"` // "oneoff", "recurring", "scheduled"
	Status      string     `json:"status"`
	Priority    int        `json:"priority,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Metadata    string     `json:"metadata,omitempty"` // JSON for additional data
}

// TaskQueue manages persistent task storage
type TaskQueue struct {
	db *sql.DB
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(dbPath string) (*TaskQueue, error) {
	// Ensure directory exists
	dir := dbPath[:len(dbPath)-len("/tasks.db")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create table
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		goal TEXT NOT NULL,
		type TEXT DEFAULT 'oneoff',
		status TEXT DEFAULT 'pending',
		priority INTEGER DEFAULT 0,
		scheduled_at DATETIME,
		result TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		metadata TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_scheduled ON tasks(scheduled_at);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &TaskQueue{db: db}, nil
}

// AddTask adds a new task to the queue
func (q *TaskQueue) AddTask(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	query := `
		INSERT INTO tasks (id, goal, type, status, priority, scheduled_at, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := q.db.ExecContext(ctx, query,
		task.ID, task.Goal, task.Type, task.Status, task.Priority,
		task.ScheduledAt, task.CreatedAt, task.Metadata,
	)

	return err
}

// GetPendingTasks retrieves pending tasks
func (q *TaskQueue) GetPendingTasks(ctx context.Context, limit int) ([]*Task, error) {
	query := `
		SELECT id, goal, type, status, priority, scheduled_at, result, error,
		       created_at, started_at, completed_at, metadata
		FROM tasks
		WHERE status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)
		ORDER BY priority DESC, created_at ASC
		LIMIT ?
	`

	rows, err := q.db.QueryContext(ctx, query, TaskStatusPending, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var scheduledAt, startedAt, completedAt sql.NullTime
		var typeVal, result, errorMsg, metadata sql.NullString

		err := rows.Scan(
			&task.ID, &task.Goal, &typeVal, &task.Status, &task.Priority,
			&scheduledAt, &result, &errorMsg,
			&task.CreatedAt, &startedAt, &completedAt, &metadata,
		)
		if err != nil {
			return nil, err
		}

		if scheduledAt.Valid {
			task.ScheduledAt = &scheduledAt.Time
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		if typeVal.Valid {
			task.Type = typeVal.String
		}
		if result.Valid {
			task.Result = result.String
		}
		if errorMsg.Valid {
			task.Error = errorMsg.String
		}
		if metadata.Valid {
			task.Metadata = metadata.String
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateTask updates a task
func (q *TaskQueue) UpdateTask(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET status = ?, result = ?, error = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`

	var startedAt, completedAt interface{}
	if task.StartedAt != nil {
		startedAt = task.StartedAt
	}
	if task.CompletedAt != nil {
		completedAt = task.CompletedAt
	}

	_, err := q.db.ExecContext(ctx, query,
		task.Status, task.Result, task.Error, startedAt, completedAt, task.ID,
	)

	return err
}

// CountByStatus counts tasks by status
func (q *TaskQueue) CountByStatus(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE status = ?`

	var count int
	err := q.db.QueryRowContext(ctx, query, status).Scan(&count)
	return count, err
}

// DeleteTask deletes a task
func (q *TaskQueue) DeleteTask(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// ClearCompleted removes all completed tasks
func (q *TaskQueue) ClearCompleted(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM tasks WHERE status IN (?, ?)`, TaskStatusCompleted, TaskStatusFailed)
	return err
}

// Close closes the database connection
func (q *TaskQueue) Close() error {
	return q.db.Close()
}

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
