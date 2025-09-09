package agents

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, "memory", config.StorageBackend)
	assert.Equal(t, 10, config.MaxConcurrentExecutions)
	assert.Equal(t, 30*time.Minute, config.DefaultTimeout)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.MCPIntegration)
	
	// Check API config
	assert.Equal(t, "localhost", config.API.Host)
	assert.Equal(t, 8080, config.API.Port)
	assert.True(t, config.API.EnableCORS)
	assert.True(t, config.API.EnableMetrics)
	assert.True(t, config.API.EnableWebSocket)
	assert.Equal(t, 30*time.Second, config.API.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.API.WriteTimeout)
	assert.Equal(t, 5*time.Second, config.API.ShutdownTimeout)
}

func TestNewManager_WithDefaultConfig(t *testing.T) {
	manager, err := NewManager(nil, nil)
	
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.executor)
	assert.NotNil(t, manager.storage)
	assert.NotNil(t, manager.handler)
	assert.NotNil(t, manager.config)
	assert.Equal(t, "memory", manager.config.StorageBackend)
}

func TestNewManager_WithCustomConfig(t *testing.T) {
	customConfig := &Config{
		StorageBackend:          "memory",
		MaxConcurrentExecutions: 20,
		DefaultTimeout:          60 * time.Minute,
		EnableMetrics:           false,
		MCPIntegration:          false,
	}
	
	manager, err := NewManager("mock-mcp-service", customConfig)
	
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, 20, manager.config.MaxConcurrentExecutions)
	assert.Equal(t, 60*time.Minute, manager.config.DefaultTimeout)
	assert.False(t, manager.config.EnableMetrics)
	assert.False(t, manager.config.MCPIntegration)
}

func TestNewManager_UnsupportedStorageBackend(t *testing.T) {
	customConfig := &Config{
		StorageBackend: "unsupported",
	}
	
	manager, err := NewManager(nil, customConfig)
	
	require.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "unsupported storage backend: unsupported")
}

func TestManager_CreateAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	agent := &types.Agent{
		ID:          "test-create-agent-1",
		Name:        "Test Create Agent",
		Description: "Agent for create testing",
		Type:        types.AgentTypeResearch,
		Status:      types.AgentStatusActive,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 5,
			DefaultTimeout:          10 * time.Minute,
		},
	}
	
	createdAgent, err := manager.CreateAgent(agent)
	
	require.NoError(t, err)
	assert.NotNil(t, createdAgent)
	assert.Equal(t, agent.ID, createdAgent.GetID())
	assert.Equal(t, agent.Name, createdAgent.GetName())
	assert.Equal(t, agent.Type, createdAgent.GetType())
	
	// Verify it was stored
	retrievedAgent, err := manager.GetAgent(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, agent.ID, retrievedAgent.GetID())
}

func TestManager_GetAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	agent := &types.Agent{
		ID:   "test-get-agent-1",
		Name: "Test Get Agent",
		Type: types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
	}
	
	// Create agent first
	_, err = manager.CreateAgent(agent)
	require.NoError(t, err)
	
	// Retrieve agent
	retrievedAgent, err := manager.GetAgent(agent.ID)
	
	require.NoError(t, err)
	assert.NotNil(t, retrievedAgent)
	assert.Equal(t, agent.ID, retrievedAgent.GetID())
	assert.Equal(t, agent.Name, retrievedAgent.GetName())
	assert.Equal(t, agent.Type, retrievedAgent.GetType())
}

func TestManager_GetAgent_NotFound(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	agent, err := manager.GetAgent("non-existent-agent")
	
	require.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_ListAgents(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Initially empty
	agents, err := manager.ListAgents()
	require.NoError(t, err)
	assert.Empty(t, agents)
	
	// Create some agents
	testAgents := []*types.Agent{
		{ID: "agent-list-1", Name: "Agent 1", Type: types.AgentTypeResearch, Status: types.AgentStatusActive},
		{ID: "agent-list-2", Name: "Agent 2", Type: types.AgentTypeWorkflow, Status: types.AgentStatusActive},
		{ID: "agent-list-3", Name: "Agent 3", Type: types.AgentTypeMonitoring, Status: types.AgentStatusActive},
	}
	
	for _, agent := range testAgents {
		_, err := manager.CreateAgent(agent)
		require.NoError(t, err)
	}
	
	// List agents
	agents, err = manager.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 3)
	
	// Verify agent IDs
	agentIDs := make(map[string]bool)
	for _, agent := range agents {
		agentIDs[agent.ID] = true
	}
	
	for _, testAgent := range testAgents {
		assert.True(t, agentIDs[testAgent.ID], "Agent %s should be in list", testAgent.ID)
	}
}

func TestManager_ExecuteAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	agent := &types.Agent{
		ID:   "test-execute-agent-1",
		Name: "Test Execute Agent",
		Type: types.AgentTypeResearch,
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "step1",
					Name: "Test Step",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"test_var": "test_value",
					},
				},
			},
		},
	}
	
	// Create agent first
	_, err = manager.CreateAgent(agent)
	require.NoError(t, err)
	
	// Execute agent
	ctx := context.Background()
	variables := map[string]interface{}{
		"input": "test_input",
	}
	
	result, err := manager.ExecuteAgent(ctx, agent.ID, variables)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ID, result.AgentID)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
}

func TestManager_ExecuteWorkflow(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	agent := &types.Agent{
		ID:   "test-workflow-agent-1",
		Name: "Test Workflow Agent",
		Type: types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "workflow_step1",
					Name: "Workflow Step 1",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"workflow_var": "workflow_value",
					},
				},
			},
		},
	}
	
	// Create agent instance
	agentInterface, err := manager.CreateAgent(agent)
	require.NoError(t, err)
	
	// Execute workflow directly
	ctx := context.Background()
	variables := map[string]interface{}{
		"workflow_input": "workflow_test",
	}
	
	result, err := manager.ExecuteWorkflow(ctx, agentInterface, variables)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ID, result.AgentID)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
}

func TestManager_HTTPHandler(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	handler := manager.HTTPHandler()
	
	assert.NotNil(t, handler)
	
	// Should be a mux router
	router, ok := handler.(*mux.Router)
	assert.True(t, ok, "Handler should be a mux.Router")
	assert.NotNil(t, router)
}

func TestManager_RegisterHTTPRoutes(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	router := mux.NewRouter()
	
	// Should not panic
	assert.NotPanics(t, func() {
		manager.RegisterHTTPRoutes(router)
	})
}

func TestManager_GetActiveExecutions(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Should return empty map initially
	active := manager.GetActiveExecutions()
	assert.NotNil(t, active)
	assert.Empty(t, active)
}

func TestManager_CancelExecution(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Try to cancel non-existent execution
	err = manager.CancelExecution("non-existent-execution")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_GetExecutionHistory(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Should return empty history for non-existent agent
	history, err := manager.GetExecutionHistory("non-existent-agent")
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestManager_CreateResearchAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	name := "Test Research Agent"
	description := "A test research agent"
	
	agent, err := manager.CreateResearchAgent(name, description)
	
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, name, agent.GetName())
	assert.Equal(t, types.AgentTypeResearch, agent.GetType())
	assert.Contains(t, agent.GetID(), "research_")
	
	// Verify workflow structure
	agentDef := agent.GetAgent()
	assert.Len(t, agentDef.Workflow.Steps, 1)
	assert.Equal(t, "analyze_document", agentDef.Workflow.Steps[0].ID)
	assert.Equal(t, types.StepTypeTool, agentDef.Workflow.Steps[0].Type)
}

func TestManager_CreateWorkflowAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	name := "Test Workflow Agent"
	description := "A test workflow agent"
	steps := []types.WorkflowStep{
		{
			ID:   "custom_step1",
			Name: "Custom Step 1",
			Type: types.StepTypeTool,
			Tool: "custom_tool",
		},
		{
			ID:   "custom_step2",
			Name: "Custom Step 2",
			Type: types.StepTypeVariable,
		},
	}
	
	agent, err := manager.CreateWorkflowAgent(name, description, steps)
	
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, name, agent.GetName())
	assert.Equal(t, types.AgentTypeWorkflow, agent.GetType())
	assert.Contains(t, agent.GetID(), "workflow_")
	
	// Verify workflow steps
	agentDef := agent.GetAgent()
	assert.Len(t, agentDef.Workflow.Steps, 2)
	assert.Equal(t, "custom_step1", agentDef.Workflow.Steps[0].ID)
	assert.Equal(t, "custom_step2", agentDef.Workflow.Steps[1].ID)
}

func TestManager_CreateMonitoringAgent(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	name := "Test Monitoring Agent"
	description := "A test monitoring agent"
	
	agent, err := manager.CreateMonitoringAgent(name, description)
	
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, name, agent.GetName())
	assert.Equal(t, types.AgentTypeMonitoring, agent.GetType())
	assert.Contains(t, agent.GetID(), "monitor_")
	
	// Verify workflow structure
	agentDef := agent.GetAgent()
	assert.Len(t, agentDef.Workflow.Steps, 1)
	assert.Equal(t, "health_check", agentDef.Workflow.Steps[0].ID)
	assert.Equal(t, types.StepTypeTool, agentDef.Workflow.Steps[0].Type)
}

func TestManager_Middleware_LoggingMiddleware(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
	
	// Wrap with logging middleware
	wrappedHandler := manager.loggingMiddleware(testHandler)
	
	assert.NotNil(t, wrappedHandler)
	
	// The middleware should not panic when called
	assert.NotPanics(t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := &testResponseWriter{}
		wrappedHandler.ServeHTTP(w, req)
	})
}

func TestManager_Middleware_CORSMiddleware(t *testing.T) {
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	// Wrap with CORS middleware
	wrappedHandler := manager.corsMiddleware(testHandler)
	
	assert.NotNil(t, wrappedHandler)
	
	// Test OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := &testResponseWriter{headers: make(http.Header)}
	
	wrappedHandler.ServeHTTP(w, req)
	
	// Should have CORS headers
	assert.Equal(t, "*", w.headers.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.headers.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.headers.Get("Access-Control-Allow-Headers"), "Content-Type")
}

func TestConfig_Structure(t *testing.T) {
	config := Config{
		StorageBackend:          "memory",
		MaxConcurrentExecutions: 15,
		DefaultTimeout:          45 * time.Minute,
		EnableMetrics:           true,
		MCPIntegration:          true,
	}
	
	assert.Equal(t, "memory", config.StorageBackend)
	assert.Equal(t, 15, config.MaxConcurrentExecutions)
	assert.Equal(t, 45*time.Minute, config.DefaultTimeout)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.MCPIntegration)
}

// Mock response writer for testing middleware
type testResponseWriter struct {
	headers http.Header
	status  int
	body    []byte
}

func (w *testResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *testResponseWriter) WriteHeader(status int) {
	w.status = status
}

func TestManager_Integration(t *testing.T) {
	// Integration test that tests the full flow
	manager, err := NewManager(nil, nil)
	require.NoError(t, err)
	
	// 1. Create an agent
	agent := &types.Agent{
		ID:   "integration-test-agent",
		Name: "Integration Test Agent",
		Type: types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "integration_step",
					Name: "Integration Step",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"test": "integration",
					},
				},
			},
		},
	}
	
	_, err = manager.CreateAgent(agent)
	require.NoError(t, err)
	
	// 2. List agents and verify it's there
	agents, err := manager.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, agent.ID, agents[0].ID)
	
	// 3. Get the agent
	retrievedAgent, err := manager.GetAgent(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, agent.ID, retrievedAgent.GetID())
	
	// 4. Execute the agent
	ctx := context.Background()
	result, err := manager.ExecuteAgent(ctx, agent.ID, map[string]interface{}{
		"input": "test",
	})
	require.NoError(t, err)
	assert.Equal(t, types.ExecutionStatusCompleted, result.Status)
	
	// 5. Get execution history
	history, err := manager.GetExecutionHistory(agent.ID)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, result.ExecutionID, history[0].ExecutionID)
	
	// 6. Test HTTP handler creation
	handler := manager.HTTPHandler()
	assert.NotNil(t, handler)
}