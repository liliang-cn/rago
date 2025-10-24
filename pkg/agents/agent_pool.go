package agents

import (
	"fmt"
	"sync"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// AgentPool manages a pool of reusable agents
type AgentPool struct {
	config     *config.Config
	llm        domain.Generator
	mcpManager *mcp.Manager
	agents     chan *Agent
	maxSize    int
	created    int
	mutex      sync.Mutex
	verbose    bool
}

// NewAgentPool creates a new agent pool
func NewAgentPool(cfg *config.Config, llm domain.Generator, mcpManager *mcp.Manager) *AgentPool {
	maxSize := 5 // Default pool size
	if cfg.Agents != nil && cfg.Agents.MaxAgents > 0 {
		maxSize = cfg.Agents.MaxAgents
	}

	return &AgentPool{
		config:     cfg,
		llm:        llm,
		mcpManager: mcpManager,
		agents:     make(chan *Agent, maxSize),
		maxSize:    maxSize,
		created:    0,
		verbose:    false,
	}
}

// SetVerbose enables verbose output
func (p *AgentPool) SetVerbose(v bool) {
	p.verbose = v
}

// GetAgent gets an agent from the pool or creates a new one
func (p *AgentPool) GetAgent() *Agent {
	select {
	case agent := <-p.agents:
		if p.verbose {
			fmt.Printf("♻️  Reusing agent from pool (available: %d)\n", len(p.agents))
		}
		return agent
	default:
		// Create new agent if under limit
		p.mutex.Lock()
		defer p.mutex.Unlock()
		
		if p.created < p.maxSize {
			agent := NewAgent(p.config, p.llm, p.mcpManager)
			agent.SetVerbose(p.verbose)
			p.created++
			
			if p.verbose {
				fmt.Printf("🆕 Created new agent (total: %d/%d)\n", p.created, p.maxSize)
			}
			return agent
		}
		
		// If at limit, wait for an agent to be returned
		if p.verbose {
			fmt.Printf("⏳ Waiting for available agent (pool full: %d/%d)\n", p.created, p.maxSize)
		}
		return <-p.agents
	}
}

// ReturnAgent returns an agent to the pool
func (p *AgentPool) ReturnAgent(agent *Agent) {
	select {
	case p.agents <- agent:
		if p.verbose {
			fmt.Printf("✅ Agent returned to pool (available: %d)\n", len(p.agents))
		}
	default:
		// Pool is full, close the agent
		agent.Close()
		if p.verbose {
			fmt.Printf("🗑️  Agent closed (pool full)\n")
		}
	}
}

// Size returns the maximum pool size
func (p *AgentPool) Size() int {
	return p.maxSize
}

// Available returns the number of available agents
func (p *AgentPool) Available() int {
	return len(p.agents)
}

// Created returns the total number of agents created
func (p *AgentPool) Created() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.created
}

// Close closes all agents in the pool
func (p *AgentPool) Close() {
	close(p.agents)
	for agent := range p.agents {
		agent.Close()
	}
}

// AgentWorker represents a specialized agent for specific task types
type AgentWorker struct {
	ID           string
	Specialization TaskType
	Agent        *Agent
	Status       AgentStatus
	TaskCount    int
	SuccessCount int
	FailureCount int
}

// NewAgentWorker creates a specialized agent worker
func NewAgentWorker(id string, specialization TaskType, agent *Agent) *AgentWorker {
	return &AgentWorker{
		ID:             id,
		Specialization: specialization,
		Agent:          agent,
		Status:         AgentStatusIdle,
		TaskCount:      0,
		SuccessCount:   0,
		FailureCount:   0,
	}
}

// GetEfficiency returns the worker's success rate
func (w *AgentWorker) GetEfficiency() float64 {
	if w.TaskCount == 0 {
		return 1.0
	}
	return float64(w.SuccessCount) / float64(w.TaskCount)
}

// WorkerPool manages specialized agent workers
type WorkerPool struct {
	workers map[string]*AgentWorker
	mutex   sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make(map[string]*AgentWorker),
	}
}

// AddWorker adds a specialized worker to the pool
func (p *WorkerPool) AddWorker(worker *AgentWorker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.workers[worker.ID] = worker
}

// GetWorker gets a worker by ID
func (p *WorkerPool) GetWorker(id string) (*AgentWorker, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	worker, exists := p.workers[id]
	return worker, exists
}

// GetBestWorker gets the most efficient available worker for a task type
func (p *WorkerPool) GetBestWorker(taskType TaskType) *AgentWorker {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var bestWorker *AgentWorker
	bestEfficiency := 0.0

	for _, worker := range p.workers {
		if worker.Specialization == taskType && worker.Status == AgentStatusIdle {
			efficiency := worker.GetEfficiency()
			if efficiency > bestEfficiency {
				bestWorker = worker
				bestEfficiency = efficiency
			}
		}
	}

	if bestWorker != nil {
		bestWorker.Status = AgentStatusAssigned
	}

	return bestWorker
}

// ReleaseWorker marks a worker as available
func (p *WorkerPool) ReleaseWorker(id string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if worker, exists := p.workers[id]; exists {
		worker.Status = AgentStatusIdle
	}
}

// UpdateWorkerStats updates worker statistics after task completion
func (p *WorkerPool) UpdateWorkerStats(id string, success bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if worker, exists := p.workers[id]; exists {
		worker.TaskCount++
		if success {
			worker.SuccessCount++
		} else {
			worker.FailureCount++
		}
	}
}

// GetStatistics returns pool statistics
func (p *WorkerPool) GetStatistics() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	totalWorkers := len(p.workers)
	idleWorkers := 0
	workingWorkers := 0
	totalTasks := 0
	totalSuccess := 0
	totalFailure := 0

	specializationCount := make(map[TaskType]int)

	for _, worker := range p.workers {
		if worker.Status == AgentStatusIdle {
			idleWorkers++
		} else if worker.Status == AgentStatusWorking {
			workingWorkers++
		}

		totalTasks += worker.TaskCount
		totalSuccess += worker.SuccessCount
		totalFailure += worker.FailureCount

		specializationCount[worker.Specialization]++
	}

	avgEfficiency := 0.0
	if totalTasks > 0 {
		avgEfficiency = float64(totalSuccess) / float64(totalTasks)
	}

	return map[string]interface{}{
		"total_workers":       totalWorkers,
		"idle_workers":        idleWorkers,
		"working_workers":     workingWorkers,
		"total_tasks":         totalTasks,
		"successful_tasks":    totalSuccess,
		"failed_tasks":        totalFailure,
		"average_efficiency":  avgEfficiency,
		"specializations":     specializationCount,
	}
}