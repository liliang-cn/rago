package skill

import (
	"sync"
	"time"
)

// Cache caches loaded skills
type Cache struct {
	mu     sync.RWMutex
	skills map[string]*Skill
	ttl    time.Duration
}

// NewCache creates a new skill cache
func NewCache() *Cache {
	return &Cache{
		skills: make(map[string]*Skill),
		ttl:    5 * time.Minute,
	}
}

// Get retrieves a skill from cache
func (c *Cache) Get(key string) (*Skill, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	skill, ok := c.skills[key]
	if !ok {
		return nil, false
	}

	// Check TTL
	if c.ttl > 0 && time.Since(skill.LoadedAt) > c.ttl {
		delete(c.skills, key)
		return nil, false
	}

	return skill, true
}

// Set stores a skill in cache
func (c *Cache) Set(key string, skill *Skill) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.skills[key] = skill
}

// Delete removes a skill from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.skills, key)
}

// Clear clears all cached skills
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.skills = make(map[string]*Skill)
}

// Size returns the number of cached skills
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.skills)
}
