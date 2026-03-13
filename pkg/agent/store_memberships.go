package agent

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"
)

func (s *Store) initMembershipSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS squad_memberships (
			agent_id TEXT NOT NULL,
			squad_id TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (agent_id, squad_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create squad_memberships table: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT OR IGNORE INTO squad_memberships (agent_id, squad_id, role, created_at, updated_at)
		SELECT id, team_id, kind, created_at, updated_at
		FROM agents
		WHERE team_id IS NOT NULL AND trim(team_id) <> ''
	`); err != nil {
		return fmt.Errorf("failed to migrate legacy squad memberships: %w", err)
	}

	return nil
}

func (s *Store) SaveSquadMembership(membership *SquadMembership) error {
	if membership == nil {
		return fmt.Errorf("membership is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if membership.CreatedAt.IsZero() {
		membership.CreatedAt = time.Now()
	}
	membership.UpdatedAt = time.Now()

	_, err := s.db.Exec(`
		INSERT INTO squad_memberships (agent_id, squad_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(agent_id, squad_id) DO UPDATE SET
			role = excluded.role,
			updated_at = excluded.updated_at
	`, membership.AgentID, membership.SquadID, string(normalizeMembershipRole(membership.Role)), membership.CreatedAt, membership.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save squad membership: %w", err)
	}
	return nil
}

func (s *Store) DeleteSquadMembership(agentID, squadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM squad_memberships WHERE agent_id = ? AND squad_id = ?`, agentID, squadID); err != nil {
		return fmt.Errorf("failed to delete squad membership: %w", err)
	}
	return nil
}

func (s *Store) DeleteMembershipsBySquad(squadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM squad_memberships WHERE squad_id = ?`, squadID); err != nil {
		return fmt.Errorf("failed to delete squad memberships: %w", err)
	}
	return nil
}

func (s *Store) DeleteMembershipsByAgent(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM squad_memberships WHERE agent_id = ?`, agentID); err != nil {
		return fmt.Errorf("failed to delete agent memberships: %w", err)
	}
	return nil
}

func (s *Store) ListSquadMemberships() ([]SquadMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listSquadMembershipsNoLock("", "")
}

func (s *Store) ListSquadMembershipsByAgent(agentID string) ([]SquadMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listSquadMembershipsNoLock(agentID, "")
}

func (s *Store) ListSquadMembershipsBySquad(squadID string) ([]SquadMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listSquadMembershipsNoLock("", squadID)
}

func (s *Store) listSquadMembershipsNoLock(agentID, squadID string) ([]SquadMembership, error) {
	query := `
		SELECT m.agent_id, m.squad_id, t.name, m.role, m.created_at, m.updated_at
		FROM squad_memberships m
		LEFT JOIN teams t ON t.id = m.squad_id
		WHERE 1 = 1
	`
	args := make([]any, 0, 2)
	if strings.TrimSpace(agentID) != "" {
		query += ` AND m.agent_id = ?`
		args = append(args, agentID)
	}
	if strings.TrimSpace(squadID) != "" {
		query += ` AND m.squad_id = ?`
		args = append(args, squadID)
	}
	query += ` ORDER BY m.squad_id ASC, m.agent_id ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := make([]SquadMembership, 0)
	for rows.Next() {
		var membership SquadMembership
		var squadName sql.NullString
		var role string
		if err := rows.Scan(&membership.AgentID, &membership.SquadID, &squadName, &role, &membership.CreatedAt, &membership.UpdatedAt); err != nil {
			return nil, err
		}
		if squadName.Valid {
			membership.SquadName = squadName.String
		}
		membership.Role = normalizeMembershipRole(AgentKind(role))
		memberships = append(memberships, membership)
	}

	return memberships, nil
}

func (s *Store) hydrateAgentMemberships(agent *AgentModel) error {
	if agent == nil {
		return nil
	}
	memberships, err := s.listSquadMembershipsNoLock(agent.ID, "")
	if err != nil {
		return err
	}
	agent.Squads = append(agent.Squads[:0], memberships...)
	agent.TeamID = ""
	if len(memberships) > 0 {
		first := memberships[0]
		agent.TeamID = first.SquadID
		agent.Kind = AgentKindAgent
	}
	if strings.TrimSpace(agent.TeamID) == "" && agent.Kind != AgentKindAgent {
		agent.Kind = AgentKindAgent
	}
	return nil
}

func normalizeMembershipRole(role AgentKind) AgentKind {
	switch role {
	case AgentKindCaptain:
		return AgentKindCaptain
	case AgentKindSpecialist:
		return AgentKindSpecialist
	default:
		return AgentKindSpecialist
	}
}

func cloneAgentForMembership(model *AgentModel, membership SquadMembership) *AgentModel {
	if model == nil {
		return nil
	}
	cloned := *model
	cloned.TeamID = membership.SquadID
	cloned.Kind = membership.Role
	cloned.Squads = []SquadMembership{membership}
	return &cloned
}

func hasMembershipRole(memberships []SquadMembership, role AgentKind) bool {
	for _, membership := range memberships {
		if membership.Role == role {
			return true
		}
	}
	return false
}

func leadMemberships(memberships []SquadMembership) []SquadMembership {
	out := make([]SquadMembership, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Role == AgentKindCaptain {
			out = append(out, membership)
		}
	}
	slices.SortFunc(out, func(a, b SquadMembership) int {
		return strings.Compare(a.SquadID, b.SquadID)
	})
	return out
}
