package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PlanStorage handles database storage for plans
type PlanStorage struct {
	db *sql.DB
}

// NewPlanStorage creates a new plan storage instance
func NewPlanStorage(dbPath string) (*PlanStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &PlanStorage{db: db}
	
	// Create tables if they don't exist
	if err := storage.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (ps *PlanStorage) Close() error {
	return ps.db.Close()
}

// createTables creates the necessary database tables
func (ps *PlanStorage) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		request TEXT NOT NULL,
		goal TEXT NOT NULL,
		output_format TEXT,
		plan_json TEXT NOT NULL,
		file_path TEXT,
		status TEXT DEFAULT 'created',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		executed_at TIMESTAMP,
		execution_count INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS plan_steps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plan_id TEXT NOT NULL,
		step_number INTEGER NOT NULL,
		tool TEXT NOT NULL,
		arguments TEXT NOT NULL,
		description TEXT,
		expected_output TEXT,
		depends_on TEXT,
		FOREIGN KEY (plan_id) REFERENCES plans (id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plan_id TEXT NOT NULL,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		status TEXT NOT NULL,
		results TEXT,
		error_message TEXT,
		FOREIGN KEY (plan_id) REFERENCES plans (id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_plans_created_at ON plans(created_at);
	CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);
	CREATE INDEX IF NOT EXISTS idx_executions_plan_id ON executions(plan_id);
	`

	_, err := ps.db.Exec(schema)
	return err
}

// SavePlan saves a plan to the database
func (ps *PlanStorage) SavePlan(id string, plan *Plan, filePath string) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Marshal plan to JSON for storage
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	// Insert plan
	_, err = tx.Exec(`
		INSERT INTO plans (id, request, goal, output_format, plan_json, file_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, plan.Request, plan.Goal, plan.OutputFormat, string(planJSON), filePath)
	if err != nil {
		return fmt.Errorf("failed to insert plan: %w", err)
	}

	// Insert steps
	for _, step := range plan.Steps {
		argsJSON, _ := json.Marshal(step.Arguments)
		dependsJSON, _ := json.Marshal(step.DependsOn)
		
		_, err = tx.Exec(`
			INSERT INTO plan_steps (plan_id, step_number, tool, arguments, description, expected_output, depends_on)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, id, step.StepNumber, step.Tool, string(argsJSON), step.Description, step.ExpectedOutput, string(dependsJSON))
		if err != nil {
			return fmt.Errorf("failed to insert step %d: %w", step.StepNumber, err)
		}
	}

	return tx.Commit()
}

// GetPlan retrieves a plan from the database
func (ps *PlanStorage) GetPlan(id string) (*Plan, error) {
	var planJSON string
	err := ps.db.QueryRow(`
		SELECT plan_json FROM plans WHERE id = ?
	`, id).Scan(&planJSON)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// ListPlans retrieves a list of all plans
func (ps *PlanStorage) ListPlans(limit int, offset int) ([]PlanRecord, error) {
	query := `
		SELECT id, request, goal, status, created_at, executed_at, execution_count, file_path
		FROM plans
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := ps.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query plans: %w", err)
	}
	defer rows.Close()

	var plans []PlanRecord
	for rows.Next() {
		var pr PlanRecord
		var executedAt sql.NullTime
		var filePath sql.NullString
		
		err := rows.Scan(&pr.ID, &pr.Request, &pr.Goal, &pr.Status, 
			&pr.CreatedAt, &executedAt, &pr.ExecutionCount, &filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		
		if executedAt.Valid {
			pr.ExecutedAt = &executedAt.Time
		}
		if filePath.Valid {
			pr.FilePath = filePath.String
		}
		
		plans = append(plans, pr)
	}

	return plans, nil
}

// RecordExecution records the start of a plan execution
func (ps *PlanStorage) RecordExecution(planID string) (int64, error) {
	result, err := ps.db.Exec(`
		INSERT INTO executions (plan_id, status)
		VALUES (?, 'running')
	`, planID)
	
	if err != nil {
		return 0, fmt.Errorf("failed to record execution: %w", err)
	}
	
	execID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get execution ID: %w", err)
	}
	
	// Update plan's execution count and last executed time
	_, err = ps.db.Exec(`
		UPDATE plans 
		SET execution_count = execution_count + 1,
		    executed_at = CURRENT_TIMESTAMP,
		    status = 'executing'
		WHERE id = ?
	`, planID)
	
	if err != nil {
		return 0, fmt.Errorf("failed to update plan: %w", err)
	}
	
	return execID, nil
}

// CompleteExecution marks an execution as completed
func (ps *PlanStorage) CompleteExecution(execID int64, results map[string]interface{}, err error) error {
	var status string
	var errorMsg sql.NullString
	var resultsJSON sql.NullString
	
	if err != nil {
		status = "failed"
		errorMsg = sql.NullString{String: err.Error(), Valid: true}
	} else {
		status = "completed"
		if results != nil {
			if data, err := json.Marshal(results); err == nil {
				resultsJSON = sql.NullString{String: string(data), Valid: true}
			}
		}
	}
	
	_, dbErr := ps.db.Exec(`
		UPDATE executions
		SET completed_at = CURRENT_TIMESTAMP,
		    status = ?,
		    results = ?,
		    error_message = ?
		WHERE id = ?
	`, status, resultsJSON, errorMsg, execID)
	
	if dbErr != nil {
		return fmt.Errorf("failed to update execution: %w", dbErr)
	}
	
	// Update plan status
	var planID string
	ps.db.QueryRow("SELECT plan_id FROM executions WHERE id = ?", execID).Scan(&planID)
	
	if planID != "" {
		_, dbErr = ps.db.Exec(`
			UPDATE plans 
			SET status = ?
			WHERE id = ?
		`, status, planID)
	}
	
	return dbErr
}

// GetExecutionHistory retrieves execution history for a plan
func (ps *PlanStorage) GetExecutionHistory(planID string) ([]ExecutionRecord, error) {
	query := `
		SELECT id, started_at, completed_at, status, error_message
		FROM executions
		WHERE plan_id = ?
		ORDER BY started_at DESC
	`
	
	rows, err := ps.db.Query(query, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	var executions []ExecutionRecord
	for rows.Next() {
		var er ExecutionRecord
		var completedAt sql.NullTime
		var errorMsg sql.NullString
		
		err := rows.Scan(&er.ID, &er.StartedAt, &completedAt, &er.Status, &errorMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		
		if completedAt.Valid {
			er.CompletedAt = &completedAt.Time
		}
		if errorMsg.Valid {
			er.ErrorMessage = errorMsg.String
		}
		
		executions = append(executions, er)
	}

	return executions, nil
}

// SearchPlans searches for plans matching criteria
func (ps *PlanStorage) SearchPlans(searchTerm string) ([]PlanRecord, error) {
	query := `
		SELECT id, request, goal, status, created_at, executed_at, execution_count, file_path
		FROM plans
		WHERE request LIKE ? OR goal LIKE ? OR id LIKE ?
		ORDER BY created_at DESC
		LIMIT 50
	`
	
	searchPattern := "%" + searchTerm + "%"
	rows, err := ps.db.Query(query, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search plans: %w", err)
	}
	defer rows.Close()

	var plans []PlanRecord
	for rows.Next() {
		var pr PlanRecord
		var executedAt sql.NullTime
		var filePath sql.NullString
		
		err := rows.Scan(&pr.ID, &pr.Request, &pr.Goal, &pr.Status, 
			&pr.CreatedAt, &executedAt, &pr.ExecutionCount, &filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		
		if executedAt.Valid {
			pr.ExecutedAt = &executedAt.Time
		}
		if filePath.Valid {
			pr.FilePath = filePath.String
		}
		
		plans = append(plans, pr)
	}

	return plans, nil
}

// DeletePlan removes a plan and all associated data
func (ps *PlanStorage) DeletePlan(id string) error {
	_, err := ps.db.Exec("DELETE FROM plans WHERE id = ?", id)
	return err
}

// PlanRecord represents a plan record in the database
type PlanRecord struct {
	ID             string     `json:"id"`
	Request        string     `json:"request"`
	Goal           string     `json:"goal"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty"`
	ExecutionCount int        `json:"execution_count"`
	FilePath       string     `json:"file_path,omitempty"`
}

// ExecutionRecord represents an execution record
type ExecutionRecord struct {
	ID           int64      `json:"id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
}