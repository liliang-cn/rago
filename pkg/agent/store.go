package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Store provides persistent storage for agent plans and sessions
type Store struct {
	mu     sync.RWMutex
	db     *sql.DB
	dbPath string
}

// NewStore creates a new storage backend for agent data
func NewStore(dbPath string) (*Store, error) {
	// Use modernc.org/sqlite which doesn't require CGO
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := configureSQLiteDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure sqlite: %w", err)
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

	// Plans table (renamed to agent_plans to avoid collision)
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_plans (
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
		return fmt.Errorf("failed to create agent_plans table: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS teams (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create teams table: %w", err)
	}

	// Dynamic Agents table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			team_id TEXT,
			name TEXT UNIQUE NOT NULL,
			kind TEXT DEFAULT 'captain',
			description TEXT NOT NULL,
			instructions TEXT NOT NULL,
			model TEXT,
			preferred_provider TEXT,
			preferred_model TEXT,
			required_llm_capability INTEGER DEFAULT 0,
			mcp_tools TEXT,
			skills TEXT,
			enable_rag BOOLEAN DEFAULT 0,
			enable_memory BOOLEAN DEFAULT 0,
			enable_ptc BOOLEAN DEFAULT 0,
			enable_mcp BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create agents table: %w", err)
	}

	if _, err := s.db.Exec(`ALTER TABLE agents ADD COLUMN team_id TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate agents.team_id: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE agents ADD COLUMN required_llm_capability INTEGER DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate agents.required_llm_capability: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE agents ADD COLUMN kind TEXT DEFAULT 'captain'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate agents.kind: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE agents ADD COLUMN preferred_provider TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate agents.preferred_provider: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE agents ADD COLUMN preferred_model TEXT`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to migrate agents.preferred_model: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE agents SET kind = 'captain' WHERE kind = 'leader' OR kind = 'commander' OR kind = '' OR kind IS NULL`); err != nil {
		return fmt.Errorf("failed to migrate agents.kind values: %w", err)
	}
	if err := s.initMembershipSchema(); err != nil {
		return err
	}

	// Sessions table (renamed to agent_sessions to avoid collision with core library)
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_sessions (
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
		return fmt.Errorf("failed to create agent_sessions table: %w", err)
	}

	return nil
}

// SavePlan saves or updates an agent plan
func (s *Store) SavePlan(plan *Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stepsJSON, _ := json.Marshal(plan.Steps)
	_, err := s.db.Exec(`
		INSERT INTO agent_plans (id, goal, session_id, steps, status, reasoning, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			goal = excluded.goal,
			steps = excluded.steps,
			status = excluded.status,
			reasoning = excluded.reasoning,
			error = excluded.error,
			updated_at = excluded.updated_at
	`, plan.ID, plan.Goal, plan.SessionID, string(stepsJSON), plan.Status, plan.Reasoning, plan.Error, plan.CreatedAt, plan.UpdatedAt)
	return err
}

// GetPlan retrieves a plan by ID
func (s *Store) GetPlan(id string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var plan Plan
	var stepsJSON string
	err := s.db.QueryRow(`
		SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
		FROM agent_plans
		WHERE id = ?
	`, id).Scan(&plan.ID, &plan.Goal, &plan.SessionID, &stepsJSON,
		&plan.Status, &plan.Reasoning, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt)

	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(stepsJSON), &plan.Steps)
	return &plan, nil
}

// ListPlans retrieves plans with optional limit and session filtering
func (s *Store) ListPlans(sessionID string, limit int) ([]*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var rows *sql.Rows
	var err error

	if sessionID != "" {
		query = `
			SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
			FROM agent_plans WHERE session_id = ?
			ORDER BY created_at DESC
		`
		if limit > 0 {
			rows, err = s.db.Query(query, sessionID, limit)
		} else {
			rows, err = s.db.Query(query, sessionID)
		}
	} else {
		query = `
			SELECT id, goal, session_id, steps, status, reasoning, error, created_at, updated_at
			FROM agent_plans
			ORDER BY created_at DESC
		`
		if limit > 0 {
			rows, err = s.db.Query(query, limit)
		} else {
			rows, err = s.db.Query(query)
		}
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		var plan Plan
		var stepsJSON string
		err := rows.Scan(&plan.ID, &plan.Goal, &plan.SessionID, &stepsJSON,
			&plan.Status, &plan.Reasoning, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt)
		if err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(stepsJSON), &plan.Steps)
		plans = append(plans, &plan)
	}

	return plans, nil
}

// SaveSession saves or updates an agent session
func (s *Store) SaveSession(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	messagesJSON, _ := json.Marshal(session.Messages)
	contextJSON, _ := json.Marshal(session.Context)
	metadataJSON, _ := json.Marshal(session.Metadata)

	_, err := s.db.Exec(`
		INSERT INTO agent_sessions (id, agent_id, messages, summary, context, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			messages = excluded.messages,
			summary = excluded.summary,
			context = excluded.context,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`, session.ID, session.AgentID, string(messagesJSON), session.Summary, string(contextJSON), string(metadataJSON), session.CreatedAt, session.UpdatedAt)
	return err
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session := &Session{}
	var messagesJSON, contextJSON, metadataJSON string
	var summary sql.NullString

	err := s.db.QueryRow(`
		SELECT id, agent_id, messages, summary, context, metadata, created_at, updated_at
		FROM agent_sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.AgentID, &messagesJSON, &summary,
		&contextJSON, &metadataJSON, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if summary.Valid {
		session.Summary = summary.String
	}

	_ = json.Unmarshal([]byte(messagesJSON), &session.Messages)
	_ = json.Unmarshal([]byte(contextJSON), &session.Context)
	_ = json.Unmarshal([]byte(metadataJSON), &session.Metadata)

	return session, nil
}

// ListSessions retrieves all sessions
func (s *Store) ListSessions(limit int) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, agent_id, messages, summary, context, metadata, created_at, updated_at
		FROM agent_sessions ORDER BY updated_at DESC
	`
	var rows *sql.Rows
	var err error

	if limit > 0 {
		rows, err = s.db.Query(query+" LIMIT ?", limit)
	} else {
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		var messagesJSON, contextJSON, metadataJSON string
		var summary sql.NullString

		err := rows.Scan(&session.ID, &session.AgentID, &messagesJSON, &summary,
			&contextJSON, &metadataJSON, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			continue
		}

		if summary.Valid {
			session.Summary = summary.String
		}

		_ = json.Unmarshal([]byte(messagesJSON), &session.Messages)
		_ = json.Unmarshal([]byte(contextJSON), &session.Context)
		_ = json.Unmarshal([]byte(metadataJSON), &session.Metadata)

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// DeleteSession deletes a session and its associated plans
func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM agent_sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// SaveAgentModel saves or updates an agent model configuration
func (s *Store) SaveAgentModel(agent *AgentModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mcpToolsJSON, _ := json.Marshal(agent.MCPTools)
	skillsJSON, _ := json.Marshal(agent.Skills)
	kind := normalizeAgentKind(agent)

	_, err := s.db.Exec(`
		INSERT INTO agents (id, team_id, name, kind, description, instructions, model, preferred_provider, preferred_model, required_llm_capability, mcp_tools, skills, enable_rag, enable_memory, enable_ptc, enable_mcp, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			team_id = excluded.team_id,
			name = excluded.name,
			kind = excluded.kind,
			description = excluded.description,
			instructions = excluded.instructions,
			model = excluded.model,
			preferred_provider = excluded.preferred_provider,
			preferred_model = excluded.preferred_model,
			required_llm_capability = excluded.required_llm_capability,
			mcp_tools = excluded.mcp_tools,
			skills = excluded.skills,
			enable_rag = excluded.enable_rag,
			enable_memory = excluded.enable_memory,
			enable_ptc = excluded.enable_ptc,
			enable_mcp = excluded.enable_mcp,
			updated_at = CURRENT_TIMESTAMP
	`, agent.ID, agent.TeamID, agent.Name, string(kind), agent.Description, agent.Instructions, agent.Model, agent.PreferredProvider, agent.PreferredModel, agent.RequiredLLMCapability, string(mcpToolsJSON), string(skillsJSON), agent.EnableRAG, agent.EnableMemory, agent.EnablePTC, agent.EnableMCP, agent.CreatedAt, agent.UpdatedAt)
	return err
}

// GetAgentModel retrieves an agent model by ID
func (s *Store) GetAgentModel(id string) (*AgentModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent := &AgentModel{}
	var mcpToolsJSON, skillsJSON string
	var kindStr string

	err := s.db.QueryRow(`
		SELECT id, team_id, name, kind, description, instructions, model, preferred_provider, preferred_model, required_llm_capability, mcp_tools, skills, enable_rag, enable_memory, enable_ptc, enable_mcp, created_at, updated_at
		FROM agents WHERE id = ?
	`, id).Scan(&agent.ID, &agent.TeamID, &agent.Name, &kindStr, &agent.Description, &agent.Instructions, &agent.Model, &agent.PreferredProvider, &agent.PreferredModel, &agent.RequiredLLMCapability,
		&mcpToolsJSON, &skillsJSON, &agent.EnableRAG, &agent.EnableMemory, &agent.EnablePTC, &agent.EnableMCP, &agent.CreatedAt, &agent.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %s", id)
		}
		return nil, err
	}

	agent.Kind = normalizeLoadedAgentKind(kindStr, agent)
	_ = json.Unmarshal([]byte(mcpToolsJSON), &agent.MCPTools)
	_ = json.Unmarshal([]byte(skillsJSON), &agent.Skills)
	if err := s.hydrateAgentMemberships(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// GetAgentModelByName retrieves an agent model by Name
func (s *Store) GetAgentModelByName(name string) (*AgentModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent := &AgentModel{}
	var mcpToolsJSON, skillsJSON string
	var kindStr string

	err := s.db.QueryRow(`
		SELECT id, team_id, name, kind, description, instructions, model, preferred_provider, preferred_model, required_llm_capability, mcp_tools, skills, enable_rag, enable_memory, enable_ptc, enable_mcp, created_at, updated_at
		FROM agents WHERE name = ?
	`, name).Scan(&agent.ID, &agent.TeamID, &agent.Name, &kindStr, &agent.Description, &agent.Instructions, &agent.Model, &agent.PreferredProvider, &agent.PreferredModel, &agent.RequiredLLMCapability,
		&mcpToolsJSON, &skillsJSON, &agent.EnableRAG, &agent.EnableMemory, &agent.EnablePTC, &agent.EnableMCP, &agent.CreatedAt, &agent.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found by name: %s", name)
		}
		return nil, err
	}

	agent.Kind = normalizeLoadedAgentKind(kindStr, agent)
	_ = json.Unmarshal([]byte(mcpToolsJSON), &agent.MCPTools)
	_ = json.Unmarshal([]byte(skillsJSON), &agent.Skills)
	if err := s.hydrateAgentMemberships(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// ListAgentModels retrieves all agent models
func (s *Store) ListAgentModels() ([]*AgentModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, team_id, name, kind, description, instructions, model, preferred_provider, preferred_model, required_llm_capability, mcp_tools, skills, enable_rag, enable_memory, enable_ptc, enable_mcp, created_at, updated_at
		FROM agents ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*AgentModel
	for rows.Next() {
		agent := &AgentModel{}
		var mcpToolsJSON, skillsJSON string
		var kindStr string

		err := rows.Scan(&agent.ID, &agent.TeamID, &agent.Name, &kindStr, &agent.Description, &agent.Instructions, &agent.Model, &agent.PreferredProvider, &agent.PreferredModel, &agent.RequiredLLMCapability,
			&mcpToolsJSON, &skillsJSON, &agent.EnableRAG, &agent.EnableMemory, &agent.EnablePTC, &agent.EnableMCP, &agent.CreatedAt, &agent.UpdatedAt)
		if err != nil {
			continue
		}

		agent.Kind = normalizeLoadedAgentKind(kindStr, agent)
		_ = json.Unmarshal([]byte(mcpToolsJSON), &agent.MCPTools)
		_ = json.Unmarshal([]byte(skillsJSON), &agent.Skills)
		if err := s.hydrateAgentMemberships(agent); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func normalizeAgentKind(agent *AgentModel) AgentKind {
	if agent == nil {
		return AgentKindCaptain
	}
	if agent.Kind == AgentKindCaptain || agent.Kind == AgentKindSpecialist || agent.Kind == AgentKindAgent {
		return agent.Kind
	}
	if strings.TrimSpace(agent.TeamID) == "" {
		return AgentKindAgent
	}
	return AgentKindCaptain
}

func normalizeLoadedAgentKind(kind string, agent *AgentModel) AgentKind {
	switch AgentKind(kind) {
	case AgentKindCaptain, AgentKindSpecialist, AgentKindAgent:
		return AgentKind(kind)
	case "leader", "lead", "lead-agent", "commander":
		return AgentKindCaptain
	default:
		return normalizeAgentKind(agent)
	}
}

func (s *Store) SaveTeam(team *Squad) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO teams (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			updated_at = CURRENT_TIMESTAMP
	`, team.ID, team.Name, team.Description, team.CreatedAt, team.UpdatedAt)
	return err
}

func (s *Store) GetTeam(id string) (*Squad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team := &Squad{}
	err := s.db.QueryRow(`
		SELECT id, name, description, created_at, updated_at
		FROM teams WHERE id = ?
	`, id).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team not found: %s", id)
		}
		return nil, err
	}
	return team, nil
}

func (s *Store) GetTeamByName(name string) (*Squad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team := &Squad{}
	err := s.db.QueryRow(`
		SELECT id, name, description, created_at, updated_at
		FROM teams WHERE lower(name) = lower(?)
	`, name).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team not found: %s", name)
		}
		return nil, err
	}
	return team, nil
}

func (s *Store) ListTeams() ([]*Squad, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM teams ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*Squad
	for rows.Next() {
		team := &Squad{}
		if err := rows.Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt, &team.UpdatedAt); err != nil {
			continue
		}
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *Store) DeleteTeam(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM teams WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// DeleteAgentModel deletes an agent model
func (s *Store) DeleteAgentModel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM squad_memberships WHERE agent_id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete agent memberships: %w", err)
	}
	_, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Close()
}
