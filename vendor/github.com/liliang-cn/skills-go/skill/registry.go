package skill

import (
	"context"
	"strings"
	"sync"
)

// Registry manages skill discovery and resolution
type Registry struct {
	loader  *Loader
	skills  map[string]*Skill // by path
	byName  map[string]*Skill // by name
	mu      sync.RWMutex
}

// NewRegistry creates a new skill registry
func NewRegistry(loader *Loader) *Registry {
	return &Registry{
		loader: loader,
		skills: make(map[string]*Skill),
		byName: make(map[string]*Skill),
	}
}

// Load loads all skills from configured paths
func (r *Registry) Load(ctx context.Context) error {
	skills, err := r.loader.LoadAll(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range skills {
		r.skills[s.Path] = s
		r.byName[s.Name] = s
	}

	return nil
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (*Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byName[name]
	if !ok {
		return nil, ErrSkillNotFound
	}
	return s, nil
}

// GetByPath retrieves a skill by path
func (r *Registry) GetByPath(path string) (*Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.skills[path]
	if !ok {
		return nil, ErrSkillNotFound
	}
	return s, nil
}

// Resolve finds relevant skills based on user input
func (r *Registry) Resolve(ctx context.Context, input string) ([]*Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	input = strings.ToLower(input)
	inputWords := extractWords(input)

	var relevant []*Skill

	for _, s := range r.byName {
		// Skip if model cannot invoke
		if !s.IsModelInvocable() {
			continue
		}

		// Check if description matches
		if r.matches(input, inputWords, s) {
			relevant = append(relevant, s)
		}
	}

	return relevant, nil
}

// matches checks if a skill matches the input
func (r *Registry) matches(input string, inputWords []string, s *Skill) bool {
	desc := strings.ToLower(s.Meta.Description)

	// Check for exact phrase matches
	for _, word := range inputWords {
		if len(word) < 4 {
			continue // Skip short words
		}
		if strings.Contains(desc, word) {
			return true
		}
	}

	// Check for skill name match
	if strings.Contains(input, s.Name) {
		return true
	}

	return false
}

// extractWords extracts words from input for matching
func extractWords(input string) []string {
	// Simple word extraction
	words := strings.Fields(input)
	var result []string

	for _, w := range words {
		// Remove punctuation
		w = strings.Trim(w, ".,!?;:\"'")
		if len(w) >= 4 {
			result = append(result, strings.ToLower(w))
		}
	}

	return result
}

// List returns all skill metadata
func (r *Registry) List() []Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var metas []Meta
	for _, s := range r.byName {
		metas = append(metas, s.Meta)
	}
	return metas
}

// ListSkills returns all skills
func (r *Registry) ListSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.byName))
	for _, s := range r.byName {
		skills = append(skills, s)
	}
	return skills
}

// Add adds a skill to the registry
func (r *Registry) Add(skill *Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills[skill.Path] = skill
	r.byName[skill.Name] = skill
}

// Remove removes a skill from the registry
func (r *Registry) Remove(skill *Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.skills, skill.Path)
	delete(r.byName, skill.Name)
}

// Clear clears all skills from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills = make(map[string]*Skill)
	r.byName = make(map[string]*Skill)
}

// Count returns the number of registered skills
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.byName)
}

// Names returns all skill names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	return names
}

// FindByPrefix finds skills whose name starts with the given prefix
func (r *Registry) FindByPrefix(prefix string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*Skill
	for _, s := range r.byName {
		if strings.HasPrefix(s.Name, prefix) {
			results = append(results, s)
		}
	}
	return results
}
