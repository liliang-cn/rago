package skills

import (
	"context"
	"sync"
)

// MemoryStore implements an in-memory skill store
type MemoryStore struct {
	skills     map[string]*Skill
	executions map[string][]*ExecutionResult
	mu         sync.RWMutex
}

// NewMemoryStore creates a new in-memory skill store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		skills:     make(map[string]*Skill),
		executions: make(map[string][]*ExecutionResult),
	}
}

// SaveSkill saves a skill to the store
func (s *MemoryStore) SaveSkill(ctx context.Context, skill *Skill) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	skillCopy := *skill
	s.skills[skill.ID] = &skillCopy
	return nil
}

// GetSkill retrieves a skill by ID
func (s *MemoryStore) GetSkill(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill := s.skills[id]
	if skill == nil {
		return nil, ErrSkillNotFound
	}

	skillCopy := *skill
	return &skillCopy, nil
}

// ListSkills lists skills with optional filtering
func (s *MemoryStore) ListSkills(ctx context.Context, filter SkillFilter) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Skill
	for _, skill := range s.skills {
		if matchesFilter(skill, filter) {
			skillCopy := *skill
			result = append(result, &skillCopy)
		}
	}

	return result, nil
}

// DeleteSkill deletes a skill from the store
func (s *MemoryStore) DeleteSkill(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skills[id]; !exists {
		return ErrSkillNotFound
	}

	delete(s.skills, id)
	return nil
}

// SaveExecution saves an execution result
func (s *MemoryStore) SaveExecution(ctx context.Context, result *ExecutionResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resultCopy := *result
	s.executions[result.SkillID] = append(s.executions[result.SkillID], &resultCopy)
	return nil
}

// GetExecutions retrieves execution history for a skill
func (s *MemoryStore) GetExecutions(ctx context.Context, skillID string, limit int) ([]*ExecutionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	executions := s.executions[skillID]
	if executions == nil {
		return []*ExecutionResult{}, nil
	}

	if limit > 0 && len(executions) > limit {
		// Return the most recent executions
		start := len(executions) - limit
		result := make([]*ExecutionResult, limit)
		for i := 0; i < limit; i++ {
			execCopy := *executions[start+i]
			result[i] = &execCopy
		}
		return result, nil
	}

	result := make([]*ExecutionResult, len(executions))
	for i, exec := range executions {
		execCopy := *exec
		result[i] = &execCopy
	}

	return result, nil
}

// Close closes the store
func (s *MemoryStore) Close() error {
	return nil
}

// matchesFilter checks if a skill matches the filter
func matchesFilter(skill *Skill, filter SkillFilter) bool {
	if filter.Category != "" && skill.Category != filter.Category {
		return false
	}
	if filter.Tag != "" {
		found := false
		for _, tag := range skill.Tags {
			if tag == filter.Tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Enabled != nil && *filter.Enabled != skill.Enabled {
		return false
	}
	return true
}
