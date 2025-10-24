package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ProgressStorage handles detailed progress tracking for agent executions
type ProgressStorage struct {
	db    *sql.DB
	mutex sync.RWMutex
}

// NewProgressStorage creates a new progress storage instance
func NewProgressStorage(dbPath string) (*ProgressStorage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open progress database: %w", err)
	}

	storage := &ProgressStorage{db: db}
	
	// Create tables if they don't exist
	if err := storage.createProgressTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create progress tables: %w", err)
	}

	return storage, nil
}

// createProgressTables creates enhanced tables for detailed progress tracking
func (ps *ProgressStorage) createProgressTables() error {
	schema := `
	-- Mission tracking table
	CREATE TABLE IF NOT EXISTS mission_progress (
		mission_id TEXT PRIMARY KEY,
		goal TEXT NOT NULL,
		strategy_type TEXT NOT NULL,
		strategy_json TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'planning',
		total_tasks INTEGER DEFAULT 0,
		completed_tasks INTEGER DEFAULT 0,
		failed_tasks INTEGER DEFAULT 0,
		progress_percentage REAL DEFAULT 0.0,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		metadata JSON
	);

	-- Detailed step execution tracking
	CREATE TABLE IF NOT EXISTS step_progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mission_id TEXT NOT NULL,
		task_id TEXT NOT NULL,
		agent_id TEXT,
		step_number INTEGER NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		progress_percentage REAL DEFAULT 0.0,
		started_at TIMESTAMP,
		updated_at TIMESTAMP,
		completed_at TIMESTAMP,
		duration_ms INTEGER,
		input_data JSON,
		output_data JSON,
		error_message TEXT,
		retry_count INTEGER DEFAULT 0,
		metrics JSON,
		FOREIGN KEY (mission_id) REFERENCES mission_progress (mission_id) ON DELETE CASCADE
	);

	-- Agent activity tracking
	CREATE TABLE IF NOT EXISTS agent_activity (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		mission_id TEXT,
		task_id TEXT,
		activity_type TEXT NOT NULL,
		description TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		data JSON
	);

	-- Progress checkpoints for resumability
	CREATE TABLE IF NOT EXISTS progress_checkpoints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mission_id TEXT NOT NULL,
		checkpoint_name TEXT NOT NULL,
		checkpoint_data JSON NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (mission_id) REFERENCES mission_progress (mission_id) ON DELETE CASCADE
	);

	-- Real-time progress events
	CREATE TABLE IF NOT EXISTS progress_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mission_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		event_data JSON,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (mission_id) REFERENCES mission_progress (mission_id) ON DELETE CASCADE
	);

	-- Performance metrics
	CREATE TABLE IF NOT EXISTS performance_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mission_id TEXT NOT NULL,
		metric_type TEXT NOT NULL,
		metric_value REAL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (mission_id) REFERENCES mission_progress (mission_id) ON DELETE CASCADE
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_mission_status ON mission_progress(status);
	CREATE INDEX IF NOT EXISTS idx_mission_updated ON mission_progress(updated_at);
	CREATE INDEX IF NOT EXISTS idx_step_mission ON step_progress(mission_id);
	CREATE INDEX IF NOT EXISTS idx_step_status ON step_progress(status);
	CREATE INDEX IF NOT EXISTS idx_agent_activity ON agent_activity(agent_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_checkpoints ON progress_checkpoints(mission_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_events ON progress_events(mission_id, timestamp);
	`

	_, err := ps.db.Exec(schema)
	return err
}

// SaveMissionProgress saves initial mission progress
func (ps *ProgressStorage) SaveMissionProgress(mission *Mission) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	strategyJSON, err := json.Marshal(mission.Strategy)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy: %w", err)
	}

	metadataJSON, err := json.Marshal(map[string]interface{}{
		"errors": mission.Errors,
	})
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = ps.db.Exec(`
		INSERT OR REPLACE INTO mission_progress (
			mission_id, goal, strategy_type, strategy_json, status,
			total_tasks, completed_tasks, failed_tasks, progress_percentage,
			started_at, updated_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, mission.ID, mission.Goal, mission.Strategy.Type, string(strategyJSON),
		mission.Status, len(mission.Strategy.Decomposition), 0, 0, 0.0,
		mission.StartTime, time.Now(), string(metadataJSON))

	if err != nil {
		return fmt.Errorf("failed to save mission progress: %w", err)
	}

	// Log event
	ps.logProgressEvent(mission.ID, "mission_started", map[string]interface{}{
		"goal":     mission.Goal,
		"strategy": mission.Strategy.Type,
	})

	return nil
}

// UpdateStepProgress updates progress for a specific step
func (ps *ProgressStorage) UpdateStepProgress(missionID, taskID, agentID string, step int, status string, progress float64, data interface{}) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Check if step exists
	var exists bool
	err := ps.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM step_progress WHERE mission_id = ? AND task_id = ? AND step_number = ?)
	`, missionID, taskID, step).Scan(&exists)
	if err != nil {
		return err
	}

	dataJSON, _ := json.Marshal(data)
	now := time.Now()

	if exists {
		// Update existing step
		_, err = ps.db.Exec(`
			UPDATE step_progress 
			SET status = ?, progress_percentage = ?, updated_at = ?, output_data = ?
			WHERE mission_id = ? AND task_id = ? AND step_number = ?
		`, status, progress, now, string(dataJSON), missionID, taskID, step)
	} else {
		// Insert new step
		_, err = ps.db.Exec(`
			INSERT INTO step_progress (
				mission_id, task_id, agent_id, step_number, status,
				progress_percentage, started_at, updated_at, output_data
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, missionID, taskID, agentID, step, status, progress, now, now, string(dataJSON))
	}

	if err != nil {
		return fmt.Errorf("failed to update step progress: %w", err)
	}

	// Update mission progress
	ps.updateMissionProgress(missionID)

	// Log event
	ps.logProgressEvent(missionID, "step_updated", map[string]interface{}{
		"task_id": taskID,
		"step":    step,
		"status":  status,
		"progress": progress,
	})

	return nil
}

// SaveStepResult saves the complete result of a step
func (ps *ProgressStorage) SaveStepResult(missionID, taskID string, step int, result interface{}, metrics *TaskMetrics, err error) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	var status string
	var errorMsg sql.NullString
	var outputJSON []byte

	if err != nil {
		status = "failed"
		errorMsg = sql.NullString{String: err.Error(), Valid: true}
	} else {
		status = "completed"
		outputJSON, _ = json.Marshal(result)
	}

	metricsJSON, _ := json.Marshal(metrics)
	completedAt := time.Now()

	// Get started time to calculate duration
	var startedAt time.Time
	ps.db.QueryRow(`
		SELECT started_at FROM step_progress 
		WHERE mission_id = ? AND task_id = ? AND step_number = ?
	`, missionID, taskID, step).Scan(&startedAt)

	duration := completedAt.Sub(startedAt).Milliseconds()

	_, dbErr := ps.db.Exec(`
		UPDATE step_progress 
		SET status = ?, completed_at = ?, duration_ms = ?, 
		    output_data = ?, error_message = ?, metrics = ?,
		    progress_percentage = 100.0
		WHERE mission_id = ? AND task_id = ? AND step_number = ?
	`, status, completedAt, duration, string(outputJSON), errorMsg, 
		string(metricsJSON), missionID, taskID, step)

	if dbErr != nil {
		return fmt.Errorf("failed to save step result: %w", dbErr)
	}

	// Update mission progress
	ps.updateMissionProgress(missionID)

	return nil
}

// updateMissionProgress recalculates and updates mission-level progress
func (ps *ProgressStorage) updateMissionProgress(missionID string) error {
	// Calculate progress from steps
	var totalSteps, completedSteps, failedSteps int
	var avgProgress float64

	err := ps.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			AVG(progress_percentage) as avg_progress
		FROM step_progress
		WHERE mission_id = ?
	`, missionID).Scan(&totalSteps, &completedSteps, &failedSteps, &avgProgress)

	if err != nil {
		return err
	}

	// Determine mission status
	var missionStatus string
	if completedSteps == totalSteps && totalSteps > 0 {
		missionStatus = "completed"
	} else if failedSteps > 0 {
		missionStatus = "partial_failure"
	} else if completedSteps > 0 {
		missionStatus = "executing"
	} else {
		missionStatus = "planning"
	}

	// Update mission progress
	_, err = ps.db.Exec(`
		UPDATE mission_progress 
		SET completed_tasks = ?, failed_tasks = ?, progress_percentage = ?,
		    status = ?, updated_at = ?
		WHERE mission_id = ?
	`, completedSteps, failedSteps, avgProgress, missionStatus, time.Now(), missionID)

	return err
}

// SaveCheckpoint saves a checkpoint for mission resumability
func (ps *ProgressStorage) SaveCheckpoint(missionID, name string, data interface{}) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint data: %w", err)
	}

	_, err = ps.db.Exec(`
		INSERT INTO progress_checkpoints (mission_id, checkpoint_name, checkpoint_data)
		VALUES (?, ?, ?)
	`, missionID, name, string(dataJSON))

	return err
}

// GetLatestCheckpoint retrieves the most recent checkpoint for a mission
func (ps *ProgressStorage) GetLatestCheckpoint(missionID string) (map[string]interface{}, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var dataJSON string
	err := ps.db.QueryRow(`
		SELECT checkpoint_data FROM progress_checkpoints
		WHERE mission_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, missionID).Scan(&dataJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(dataJSON), &data)
	return data, err
}

// LogAgentActivity logs agent activity
func (ps *ProgressStorage) LogAgentActivity(agentID, missionID, taskID, activityType, description string, data interface{}) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	dataJSON, _ := json.Marshal(data)

	_, err := ps.db.Exec(`
		INSERT INTO agent_activity (agent_id, mission_id, task_id, activity_type, description, data)
		VALUES (?, ?, ?, ?, ?, ?)
	`, agentID, missionID, taskID, activityType, description, string(dataJSON))

	return err
}

// logProgressEvent logs a progress event
func (ps *ProgressStorage) logProgressEvent(missionID, eventType string, data interface{}) error {
	dataJSON, _ := json.Marshal(data)

	_, err := ps.db.Exec(`
		INSERT INTO progress_events (mission_id, event_type, event_data)
		VALUES (?, ?, ?)
	`, missionID, eventType, string(dataJSON))

	return err
}

// SaveMetric saves a performance metric
func (ps *ProgressStorage) SaveMetric(missionID, metricType string, value float64) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	_, err := ps.db.Exec(`
		INSERT INTO performance_metrics (mission_id, metric_type, metric_value)
		VALUES (?, ?, ?)
	`, missionID, metricType, value)

	return err
}

// GetMissionProgress retrieves detailed progress for a mission
func (ps *ProgressStorage) GetMissionProgress(missionID string) (*MissionProgress, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var mp MissionProgress
	var strategyJSON, metadataJSON string
	var completedAt sql.NullTime

	err := ps.db.QueryRow(`
		SELECT mission_id, goal, strategy_type, strategy_json, status,
		       total_tasks, completed_tasks, failed_tasks, progress_percentage,
		       started_at, updated_at, completed_at, metadata
		FROM mission_progress
		WHERE mission_id = ?
	`, missionID).Scan(
		&mp.MissionID, &mp.Goal, &mp.StrategyType, &strategyJSON, &mp.Status,
		&mp.TotalTasks, &mp.CompletedTasks, &mp.FailedTasks, &mp.ProgressPercentage,
		&mp.StartedAt, &mp.UpdatedAt, &completedAt, &metadataJSON)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		mp.CompletedAt = &completedAt.Time
	}

	// Get step progress
	rows, err := ps.db.Query(`
		SELECT task_id, step_number, description, status, progress_percentage,
		       started_at, completed_at, duration_ms, error_message
		FROM step_progress
		WHERE mission_id = ?
		ORDER BY step_number
	`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sp StepProgress
		var startedAt, completedAt sql.NullTime
		var durationMs sql.NullInt64
		var errorMsg sql.NullString

		err := rows.Scan(&sp.TaskID, &sp.StepNumber, &sp.Description, &sp.Status,
			&sp.ProgressPercentage, &startedAt, &completedAt, &durationMs, &errorMsg)
		if err != nil {
			continue
		}

		if startedAt.Valid {
			sp.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			sp.CompletedAt = &completedAt.Time
		}
		if durationMs.Valid {
			sp.DurationMs = durationMs.Int64
		}
		if errorMsg.Valid {
			sp.ErrorMessage = errorMsg.String
		}

		mp.Steps = append(mp.Steps, sp)
	}

	return &mp, nil
}

// GetRecentEvents retrieves recent progress events for a mission
func (ps *ProgressStorage) GetRecentEvents(missionID string, limit int) ([]ProgressEvent, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	rows, err := ps.db.Query(`
		SELECT event_type, event_data, timestamp
		FROM progress_events
		WHERE mission_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, missionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ProgressEvent
	for rows.Next() {
		var pe ProgressEvent
		var dataJSON string

		err := rows.Scan(&pe.EventType, &dataJSON, &pe.Timestamp)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(dataJSON), &pe.EventData)
		events = append(events, pe)
	}

	return events, nil
}

// GetMetrics retrieves performance metrics for a mission
func (ps *ProgressStorage) GetMetrics(missionID string) (map[string][]MetricPoint, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	rows, err := ps.db.Query(`
		SELECT metric_type, metric_value, timestamp
		FROM performance_metrics
		WHERE mission_id = ?
		ORDER BY timestamp
	`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make(map[string][]MetricPoint)
	for rows.Next() {
		var metricType string
		var mp MetricPoint

		err := rows.Scan(&metricType, &mp.Value, &mp.Timestamp)
		if err != nil {
			continue
		}

		metrics[metricType] = append(metrics[metricType], mp)
	}

	return metrics, nil
}

// Close closes the database connection
func (ps *ProgressStorage) Close() error {
	return ps.db.Close()
}

// Data structures for progress tracking

// MissionProgress represents detailed mission progress
type MissionProgress struct {
	MissionID          string          `json:"mission_id"`
	Goal               string          `json:"goal"`
	StrategyType       string          `json:"strategy_type"`
	Status             string          `json:"status"`
	TotalTasks         int             `json:"total_tasks"`
	CompletedTasks     int             `json:"completed_tasks"`
	FailedTasks        int             `json:"failed_tasks"`
	ProgressPercentage float64         `json:"progress_percentage"`
	StartedAt          time.Time       `json:"started_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	Steps              []StepProgress  `json:"steps"`
}

// StepProgress represents progress for a single step
type StepProgress struct {
	TaskID             string     `json:"task_id"`
	StepNumber         int        `json:"step_number"`
	Description        string     `json:"description"`
	Status             string     `json:"status"`
	ProgressPercentage float64    `json:"progress_percentage"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	DurationMs         int64      `json:"duration_ms"`
	ErrorMessage       string     `json:"error_message,omitempty"`
}

// ProgressEvent represents a progress event
type ProgressEvent struct {
	EventType string                 `json:"event_type"`
	EventData map[string]interface{} `json:"event_data"`
	Timestamp time.Time              `json:"timestamp"`
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}