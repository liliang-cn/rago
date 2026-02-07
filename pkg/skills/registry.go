package skills

import (
	"sync"
)

// Registry manages skill registration and lookup
type Registry struct {
	skills map[string]*Skill
	mu     sync.RWMutex
}

// NewRegistry creates a new skill registry
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*Skill),
	}
}

// Register registers a skill in the registry
func (r *Registry) Register(skill *Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Make a copy to avoid external mutations
	skillCopy := *skill
	r.skills[skill.ID] = &skillCopy
}

// Unregister removes a skill from the registry
func (r *Registry) Unregister(skillID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, skillID)
}

// Get retrieves a skill by ID
func (r *Registry) Get(skillID string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill := r.skills[skillID]
	if skill == nil {
		return nil
	}

	// Return a copy
	skillCopy := *skill
	skillCopy.Handler = skill.Handler
	return &skillCopy
}

// List returns all registered skills
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skillCopy := *skill
		skillCopy.Handler = skill.Handler
		result = append(result, &skillCopy)
	}

	return result
}

// Count returns the number of registered skills
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// Clear removes all skills from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*Skill)
}

// Update updates a skill in the registry
func (r *Registry) Update(skill *Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	skillCopy := *skill
	skillCopy.Handler = skill.Handler
	r.skills[skill.ID] = &skillCopy
}

// Exists checks if a skill is registered
func (r *Registry) Exists(skillID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.skills[skillID]
	return exists
}
