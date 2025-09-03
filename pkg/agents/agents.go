// Package agents provides a comprehensive workflow automation and agent execution system
// that can be used both as part of RAGO or as a standalone Go library.
package agents

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/liliang-cn/rago/v2/pkg/agents/api"
	"github.com/liliang-cn/rago/v2/pkg/agents/core"
	"github.com/liliang-cn/rago/v2/pkg/agents/storage"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// Manager provides the main interface for the agents module
type Manager struct {
	executor *core.AgentExecutor
	storage  core.AgentStorage
	handler  *api.AgentHandler
	config   *Config
}

// Config holds configuration for the agents module
type Config struct {
	StorageBackend          string        `yaml:"storage_backend" json:"storage_backend"`                     // memory, sqlite, postgres
	MaxConcurrentExecutions int           `yaml:"max_concurrent_executions" json:"max_concurrent_executions"`
	DefaultTimeout          time.Duration `yaml:"default_timeout" json:"default_timeout"`
	EnableMetrics           bool          `yaml:"enable_metrics" json:"enable_metrics"`
	MCPIntegration          bool          `yaml:"mcp_integration" json:"mcp_integration"`
	API                     api.APIConfig `yaml:"api" json:"api"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		StorageBackend:          "memory",
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          30 * time.Minute,
		EnableMetrics:           true,
		MCPIntegration:          true,
		API: api.APIConfig{
			Host:            "localhost",
			Port:            8080,
			EnableCORS:      true,
			EnableMetrics:   true,
			EnableWebSocket: true,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	}
}

// NewManager creates a new agents manager
func NewManager(mcpService interface{}, config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize storage
	var agentStorage core.AgentStorage
	switch config.StorageBackend {
	case "memory":
		agentStorage = storage.NewMemoryAgentStore()
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", config.StorageBackend)
	}

	// Create executor
	executor := core.NewAgentExecutor(mcpService, agentStorage)

	// Create API handler
	handler := api.NewAgentHandler(executor, agentStorage)

	return &Manager{
		executor: executor,
		storage:  agentStorage,
		handler:  handler,
		config:   config,
	}, nil
}

// CreateAgent creates a new agent
func (m *Manager) CreateAgent(agent *types.Agent) (types.AgentInterface, error) {
	// Save to storage
	if err := m.storage.SaveAgent(agent); err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}

	// Create implementation
	factory := core.NewAgentFactory()
	return factory.CreateAgent(agent)
}

// GetAgent retrieves an agent by ID
func (m *Manager) GetAgent(id string) (types.AgentInterface, error) {
	agent, err := m.storage.GetAgent(id)
	if err != nil {
		return nil, err
	}

	factory := core.NewAgentFactory()
	return factory.CreateAgent(agent)
}

// ListAgents returns all agents
func (m *Manager) ListAgents() ([]*types.Agent, error) {
	return m.storage.ListAgents()
}

// ExecuteAgent executes an agent workflow
func (m *Manager) ExecuteAgent(ctx context.Context, agentID string, variables map[string]interface{}) (*types.ExecutionResult, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		return nil, err
	}

	return m.executor.Execute(ctx, agent)
}

// ExecuteWorkflow executes a workflow directly
func (m *Manager) ExecuteWorkflow(ctx context.Context, agent types.AgentInterface, variables map[string]interface{}) (*types.ExecutionResult, error) {
	workflow := core.NewWorkflowEngine(m.executor)
	return workflow.ExecuteWorkflow(ctx, agent, variables)
}

// HTTPHandler returns an HTTP handler for the agents API
func (m *Manager) HTTPHandler() http.Handler {
	router := mux.NewRouter()
	
	// Create a subrouter for agents API
	agentsRouter := router.PathPrefix("/api/agents").Subrouter()
	m.handler.RegisterRoutes(agentsRouter)

	// Add middleware
	router.Use(m.loggingMiddleware)
	if m.config.API.EnableCORS {
		router.Use(m.corsMiddleware)
	}

	return router
}

// RegisterHTTPRoutes registers routes on an existing router
func (m *Manager) RegisterHTTPRoutes(router *mux.Router) {
	agentsRouter := router.PathPrefix("/api/agents").Subrouter()
	m.handler.RegisterRoutes(agentsRouter)
}

// GetActiveExecutions returns currently running executions
func (m *Manager) GetActiveExecutions() map[string]*core.ExecutionInstance {
	return m.executor.GetActiveExecutions()
}

// CancelExecution cancels a running execution
func (m *Manager) CancelExecution(executionID string) error {
	return m.executor.CancelExecution(executionID)
}

// GetExecutionHistory returns execution history for an agent
func (m *Manager) GetExecutionHistory(agentID string) ([]*types.ExecutionResult, error) {
	return m.storage.ListExecutions(agentID)
}

// Middleware functions

func (m *Manager) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request
		log.Printf("[AGENTS] %s %s", r.Method, r.URL.Path)
		
		next.ServeHTTP(w, r)
		
		// Log response time
		duration := time.Since(start)
		log.Printf("[AGENTS] %s %s completed in %v", r.Method, r.URL.Path, duration)
	})
}

func (m *Manager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Utility functions for common agent creation patterns

// CreateResearchAgent creates a research agent with a document analysis workflow
func (m *Manager) CreateResearchAgent(name, description string) (types.AgentInterface, error) {
	agent := &types.Agent{
		ID:          fmt.Sprintf("research_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Type:        types.AgentTypeResearch,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 5,
			DefaultTimeout:          10 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:          types.AutonomyManual,
		},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "analyze_document",
					Name: "Analyze Document",
					Type: types.StepTypeTool,
					Tool: "document_analyzer",
					Inputs: map[string]interface{}{
						"document": "{{input.document}}",
						"analysis_type": "comprehensive",
					},
					Outputs: map[string]string{
						"analysis": "document_analysis",
						"summary":  "document_summary",
					},
				},
			},
			Variables: map[string]interface{}{
				"analysis_depth": "detailed",
				"output_format":  "json",
			},
		},
		Status: types.AgentStatusActive,
	}

	return m.CreateAgent(agent)
}

// CreateWorkflowAgent creates a workflow automation agent
func (m *Manager) CreateWorkflowAgent(name, description string, steps []types.WorkflowStep) (types.AgentInterface, error) {
	agent := &types.Agent{
		ID:          fmt.Sprintf("workflow_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Type:        types.AgentTypeWorkflow,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 3,
			DefaultTimeout:          20 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:          types.AutonomyScheduled,
		},
		Workflow: types.WorkflowSpec{
			Steps:     steps,
			Variables: make(map[string]interface{}),
		},
		Status: types.AgentStatusActive,
	}

	return m.CreateAgent(agent)
}

// CreateMonitoringAgent creates a monitoring agent
func (m *Manager) CreateMonitoringAgent(name, description string) (types.AgentInterface, error) {
	agent := &types.Agent{
		ID:          fmt.Sprintf("monitor_%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		Type:        types.AgentTypeMonitoring,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 1,
			DefaultTimeout:          5 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:          types.AutonomyReactive,
		},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "health_check",
					Name: "System Health Check",
					Type: types.StepTypeTool,
					Tool: "system_monitor",
					Inputs: map[string]interface{}{
						"check_type": "comprehensive",
						"timeout":    "30s",
					},
					Outputs: map[string]string{
						"status":  "system_status",
						"metrics": "system_metrics",
					},
				},
			},
			Variables: map[string]interface{}{
				"alert_threshold": 0.8,
				"notification_enabled": true,
			},
		},
		Status: types.AgentStatusActive,
	}

	return m.CreateAgent(agent)
}