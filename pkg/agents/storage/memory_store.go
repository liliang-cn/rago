package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// MemoryAgentStore provides an in-memory implementation of AgentStorage
type MemoryAgentStore struct {
	agents     map[string]*types.Agent
	executions map[string]*types.ExecutionResult
	mutex      sync.RWMutex
}

// NewMemoryAgentStore creates a new in-memory agent store
func NewMemoryAgentStore() *MemoryAgentStore {
	return &MemoryAgentStore{
		agents:     make(map[string]*types.Agent),
		executions: make(map[string]*types.ExecutionResult),
	}
}

// SaveAgent saves an agent to the store
func (s *MemoryAgentStore) SaveAgent(agent *types.Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	if agent.ID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Set timestamps
	now := time.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	// Store a copy to prevent external modifications
	agentCopy := *agent
	s.agents[agent.ID] = &agentCopy

	return nil
}

// GetAgent retrieves an agent by ID
func (s *MemoryAgentStore) GetAgent(id string) (*types.Agent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	agent, exists := s.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	agentCopy := *agent
	return &agentCopy, nil
}

// ListAgents returns all agents in the store
func (s *MemoryAgentStore) ListAgents() ([]*types.Agent, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	agents := make([]*types.Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		// Return copies to prevent external modifications
		agentCopy := *agent
		agents = append(agents, &agentCopy)
	}

	return agents, nil
}

// DeleteAgent removes an agent from the store
func (s *MemoryAgentStore) DeleteAgent(id string) error {
	if id == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.agents[id]; !exists {
		return fmt.Errorf("agent with ID %s not found", id)
	}

	delete(s.agents, id)
	return nil
}

// SaveExecution saves an execution result to the store
func (s *MemoryAgentStore) SaveExecution(execution *types.ExecutionResult) error {
	if execution == nil {
		return fmt.Errorf("execution cannot be nil")
	}
	if execution.ExecutionID == "" {
		return fmt.Errorf("execution ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Store a copy to prevent external modifications
	executionCopy := *execution
	s.executions[execution.ExecutionID] = &executionCopy

	return nil
}

// GetExecution retrieves an execution result by ID
func (s *MemoryAgentStore) GetExecution(id string) (*types.ExecutionResult, error) {
	if id == "" {
		return nil, fmt.Errorf("execution ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	execution, exists := s.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	executionCopy := *execution
	return &executionCopy, nil
}

// ListExecutions returns all executions for a specific agent
func (s *MemoryAgentStore) ListExecutions(agentID string) ([]*types.ExecutionResult, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	executions := make([]*types.ExecutionResult, 0)
	for _, execution := range s.executions {
		if execution.AgentID == agentID {
			// Return copies to prevent external modifications
			executionCopy := *execution
			executions = append(executions, &executionCopy)
		}
	}

	return executions, nil
}

// ListAllExecutions returns all executions in the store
func (s *MemoryAgentStore) ListAllExecutions() ([]*types.ExecutionResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	executions := make([]*types.ExecutionResult, 0, len(s.executions))
	for _, execution := range s.executions {
		// Return copies to prevent external modifications
		executionCopy := *execution
		executions = append(executions, &executionCopy)
	}

	return executions, nil
}

// GetStats returns storage statistics
func (s *MemoryAgentStore) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_agents"] = len(s.agents)
	stats["total_executions"] = len(s.executions)

	// Count agents by type
	agentTypeCount := make(map[string]int)
	for _, agent := range s.agents {
		agentTypeCount[string(agent.Type)]++
	}
	stats["agents_by_type"] = agentTypeCount

	// Count executions by status
	executionStatusCount := make(map[string]int)
	for _, execution := range s.executions {
		executionStatusCount[string(execution.Status)]++
	}
	stats["executions_by_status"] = executionStatusCount

	return stats
}

// Clear removes all data from the store
func (s *MemoryAgentStore) Clear() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.agents = make(map[string]*types.Agent)
	s.executions = make(map[string]*types.ExecutionResult)

	return nil
}
