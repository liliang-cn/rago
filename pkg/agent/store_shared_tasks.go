package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Store) initSharedTaskSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS shared_tasks (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			squad_id TEXT NOT NULL,
			squad_name TEXT,
			captain_name TEXT NOT NULL,
			agent_names TEXT NOT NULL,
			prompt TEXT NOT NULL,
			ack_message TEXT,
			status TEXT NOT NULL,
			queued_ahead INTEGER DEFAULT 0,
			result_text TEXT,
			results TEXT,
			created_at DATETIME NOT NULL,
			started_at DATETIME,
			finished_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create shared_tasks table: %w", err)
	}

	if _, err := s.db.Exec(`ALTER TABLE shared_tasks ADD COLUMN session_id TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate shared_tasks.session_id: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE shared_tasks ADD COLUMN squad_name TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate shared_tasks.squad_name: %w", err)
	}

	return nil
}

func (s *Store) SaveSharedTask(task *SharedTask) error {
	if task == nil {
		return fmt.Errorf("shared task is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agentNamesJSON, _ := json.Marshal(task.AgentNames)
	resultsJSON, _ := json.Marshal(task.Results)

	_, err := s.db.Exec(`
		INSERT INTO shared_tasks (
			id, session_id, squad_id, squad_name, captain_name, agent_names, prompt, ack_message,
			status, queued_ahead, result_text, results, created_at, started_at, finished_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			session_id = excluded.session_id,
			squad_id = excluded.squad_id,
			squad_name = excluded.squad_name,
			captain_name = excluded.captain_name,
			agent_names = excluded.agent_names,
			prompt = excluded.prompt,
			ack_message = excluded.ack_message,
			status = excluded.status,
			queued_ahead = excluded.queued_ahead,
			result_text = excluded.result_text,
			results = excluded.results,
			created_at = excluded.created_at,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at
	`,
		task.ID,
		strings.TrimSpace(task.SessionID),
		task.SquadID,
		strings.TrimSpace(task.SquadName),
		task.CaptainName,
		string(agentNamesJSON),
		task.Prompt,
		task.AckMessage,
		string(task.Status),
		task.QueuedAhead,
		task.ResultText,
		string(resultsJSON),
		task.CreatedAt,
		task.StartedAt,
		task.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save shared task: %w", err)
	}
	return nil
}

func (s *Store) ListSharedTasksPersisted() ([]*SharedTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, session_id, squad_id, squad_name, captain_name, agent_names, prompt, ack_message,
		       status, queued_ahead, result_text, results, created_at, started_at, finished_at
		FROM shared_tasks
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*SharedTask
	for rows.Next() {
		var task SharedTask
		var (
			agentNamesJSON string
			resultsJSON    string
		)
		if err := rows.Scan(
			&task.ID,
			&task.SessionID,
			&task.SquadID,
			&task.SquadName,
			&task.CaptainName,
			&agentNamesJSON,
			&task.Prompt,
			&task.AckMessage,
			&task.Status,
			&task.QueuedAhead,
			&task.ResultText,
			&resultsJSON,
			&task.CreatedAt,
			&task.StartedAt,
			&task.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared task: %w", err)
		}
		_ = json.Unmarshal([]byte(agentNamesJSON), &task.AgentNames)
		_ = json.Unmarshal([]byte(resultsJSON), &task.Results)
		tasks = append(tasks, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate shared tasks: %w", err)
	}
	return tasks, nil
}
