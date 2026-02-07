package agent

import (
	"sync"
)

// Registry manages a collection of agents
type Registry struct {
	agents map[string]*Agent
	mu     sync.RWMutex
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*Agent),
	}
}

// Register registers an agent
func (r *Registry) Register(agent *Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.Name()] = agent
	// Also register by ID if needed, but Name is usually more user-friendly for lookup
	r.agents[agent.ID()] = agent
}

// GetAgent retrieves an agent by name or ID
func (r *Registry) GetAgent(nameOrID string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[nameOrID]
	return agent, ok
}

// ListAgents returns all registered agents
func (r *Registry) ListAgents() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]*Agent, 0, len(r.agents))
	seen := make(map[string]bool)
	
	for _, agent := range r.agents {
		if !seen[agent.ID()] {
			agents = append(agents, agent)
			seen[agent.ID()] = true
		}
	}
	return agents
}
