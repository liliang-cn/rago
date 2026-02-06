package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store provides persistent storage for agent plans and sessions
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	dbPath string
}

// NewStore creates a new agent store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *Store) initSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Plans table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY,
			goal TEXT NOT NULL,
			session_id TEXT NOT NULL,
			steps TEXT NOT NULL,
			status TEXT NOT NULL,
			reasoning TEXT,
			error TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create plans table: %w", err)
	}

	// Sessions table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			messages TEXT NOT NULL,
			summary TEXT,
			context TEXT,
			metadata TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create indexes
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_plans_session_id ON plans(session_id);
		CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);
		CREATE INDEX IF NOT EXISTS idx_plans_created_at ON plans(created_at);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// SavePlan saves a plan to the store
func (s *Store) SavePlan(plan *Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stepsJSON, err := json.Marshal(plan.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	now := time.Now()
	plan.UpdatedAt = now

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO plans
		(id, goal, session_id, steps, status, reasoning, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, plan.ID, plan.Goal, plan.SessionID, string(stepsJSON), plan.Status,
		plan.Reasoning, plan.Error, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	return nil
}

// GetPlan retrieves a plan by ID
func (s *Store) GetPlan(id string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var plan Plan
	var stepsJSON string

	err := s.db.QueryRow(`
		SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
		FROM plans WHERE id = ?
	`, id).Scan(&plan.ID, &plan.Goal, &plan.SessionID, &stepsJSON,
		&plan.Status, &plan.Reasoning, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if err := json.Unmarshal([]byte(stepsJSON), &plan.Steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
	}

	return &plan, nil
}

// ListPlans retrieves plans by session ID, or all plans if sessionID is empty
func (s *Store) ListPlans(sessionID string, limit int) ([]*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var rows *sql.Rows
	var err error

	if sessionID != "" {
		query = `
			SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
			FROM plans WHERE session_id = ?
			ORDER BY created_at DESC
		`
		if limit > 0 {
			query += " LIMIT ?"
			rows, err = s.db.Query(query, sessionID, limit)
		} else {
			rows, err = s.db.Query(query, sessionID)
		}
	} else {
		query = `
			SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
			FROM plans
			ORDER BY created_at DESC
		`
		if limit > 0 {
			query += " LIMIT ?"
			rows, err = s.db.Query(query, limit)
		} else {
			rows, err = s.db.Query(query)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		var plan Plan
		var stepsJSON string

		err := rows.Scan(&plan.ID, &plan.Goal, &plan.SessionID, &stepsJSON,
			&plan.Status, &plan.Reasoning, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}

		if err := json.Unmarshal([]byte(stepsJSON), &plan.Steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
		}

		plans = append(plans, &plan)
	}

	return plans, nil
}

// SaveSession saves a session to the store
func (s *Store) SaveSession(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	messagesJSON, err := json.Marshal(session.GetMessages())
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	contextJSON, err := json.Marshal(session.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	now := time.Now()
	session.UpdatedAt = now

	res, err := s.db.Exec(`
		INSERT OR REPLACE INTO sessions
		(id, agent_id, messages, summary, context, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.AgentID, string(messagesJSON), session.Summary, string(contextJSON),
		string(metadataJSON), session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	_, _ = res.RowsAffected()

	return nil
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var session Session
	var messagesJSON, contextJSON, metadataJSON string
	var summary sql.NullString

	err := s.db.QueryRow(`
		SELECT id, agent_id, messages, summary, context, metadata, created_at, updated_at
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.AgentID, &messagesJSON, &summary,
		&contextJSON, &metadataJSON, &session.CreatedAt, &session.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if summary.Valid {
		session.Summary = summary.String
	}

	if err := json.Unmarshal([]byte(messagesJSON), &session.Messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	if contextJSON != "" && contextJSON != "null" {
		if err := json.Unmarshal([]byte(contextJSON), &session.Context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	} else {
		session.Context = make(map[string]interface{})
	}

	if metadataJSON != "" && metadataJSON != "null" {
		if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		session.Metadata = make(map[string]interface{})
	}

	return &session, nil
}

// ListSessions retrieves all sessions
func (s *Store) ListSessions(limit int) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, agent_id, messages, summary, context, metadata, created_at, updated_at
		FROM sessions ORDER BY updated_at DESC
	`
	var rows *sql.Rows
	var err error

	if limit > 0 {
		query += " LIMIT ?"
		rows, err = s.db.Query(query, limit)
	} else {
		rows, err = s.db.Query(query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var messagesJSON, contextJSON, metadataJSON string
		var summary sql.NullString

		err := rows.Scan(&session.ID, &session.AgentID, &messagesJSON, &summary,
			&contextJSON, &metadataJSON, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if summary.Valid {
			session.Summary = summary.String
		}

		if err := json.Unmarshal([]byte(messagesJSON), &session.Messages); err != nil {
			return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
		}

		if contextJSON != "" && contextJSON != "null" {
			if err := json.Unmarshal([]byte(contextJSON), &session.Context); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		} else {
			session.Context = make(map[string]interface{})
		}

		if metadataJSON != "" && metadataJSON != "null" {
			if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		} else {
			session.Metadata = make(map[string]interface{})
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// DeleteSession removes a session
func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Close()
}
