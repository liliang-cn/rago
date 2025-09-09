package storage

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryAgentStore(t *testing.T) {
	store := NewMemoryAgentStore()

	assert.NotNil(t, store)
	assert.NotNil(t, store.agents)
	assert.NotNil(t, store.executions)
	assert.Empty(t, store.agents)
	assert.Empty(t, store.executions)
}

func TestMemoryAgentStore_SaveAgent(t *testing.T) {
	store := NewMemoryAgentStore()

	agent := &types.Agent{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent",
		Type:        types.AgentTypeResearch,
		Status:      types.AgentStatusActive,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 5,
			DefaultTimeout:          10 * time.Minute,
		},
	}

	// Test successful save
	err := store.SaveAgent(agent)
	require.NoError(t, err)

	// Verify timestamps are set
	assert.False(t, agent.CreatedAt.IsZero())
	assert.False(t, agent.UpdatedAt.IsZero())

	// Test saving the same agent again (should update timestamp)
	originalUpdatedAt := agent.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference
	
	err = store.SaveAgent(agent)
	require.NoError(t, err)
	assert.True(t, agent.UpdatedAt.After(originalUpdatedAt))
}

func TestMemoryAgentStore_SaveAgent_ValidationErrors(t *testing.T) {
	store := NewMemoryAgentStore()

	tests := []struct {
		name        string
		agent       *types.Agent
		expectedErr string
	}{
		{
			name:        "Nil agent",
			agent:       nil,
			expectedErr: "agent cannot be nil",
		},
		{
			name: "Empty ID",
			agent: &types.Agent{
				Name: "Test Agent",
			},
			expectedErr: "agent ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveAgent(tt.agent)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMemoryAgentStore_GetAgent(t *testing.T) {
	store := NewMemoryAgentStore()

	originalAgent := &types.Agent{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent",
		Type:        types.AgentTypeWorkflow,
		Status:      types.AgentStatusActive,
	}

	// Save agent first
	err := store.SaveAgent(originalAgent)
	require.NoError(t, err)

	// Test successful retrieval
	retrievedAgent, err := store.GetAgent("test-agent-1")
	require.NoError(t, err)
	assert.Equal(t, originalAgent.ID, retrievedAgent.ID)
	assert.Equal(t, originalAgent.Name, retrievedAgent.Name)
	assert.Equal(t, originalAgent.Type, retrievedAgent.Type)

	// Verify it's a copy (modifying retrieved shouldn't affect stored)
	retrievedAgent.Name = "Modified Name"
	
	retrievedAgain, err := store.GetAgent("test-agent-1")
	require.NoError(t, err)
	assert.Equal(t, "Test Agent", retrievedAgain.Name) // Should be unchanged
}

func TestMemoryAgentStore_GetAgent_Errors(t *testing.T) {
	store := NewMemoryAgentStore()

	tests := []struct {
		name        string
		agentID     string
		expectedErr string
	}{
		{
			name:        "Empty ID",
			agentID:     "",
			expectedErr: "agent ID cannot be empty",
		},
		{
			name:        "Non-existent ID",
			agentID:     "non-existent",
			expectedErr: "agent with ID non-existent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := store.GetAgent(tt.agentID)
			require.Error(t, err)
			assert.Nil(t, agent)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMemoryAgentStore_ListAgents(t *testing.T) {
	store := NewMemoryAgentStore()

	// Test empty store
	agents, err := store.ListAgents()
	require.NoError(t, err)
	assert.Empty(t, agents)

	// Add some agents
	testAgents := []*types.Agent{
		{
			ID:   "agent-1",
			Name: "Agent 1",
			Type: types.AgentTypeResearch,
			Status: types.AgentStatusActive,
		},
		{
			ID:   "agent-2",
			Name: "Agent 2",
			Type: types.AgentTypeWorkflow,
			Status: types.AgentStatusInactive,
		},
		{
			ID:   "agent-3",
			Name: "Agent 3",
			Type: types.AgentTypeMonitoring,
			Status: types.AgentStatusActive,
		},
	}

	for _, agent := range testAgents {
		err := store.SaveAgent(agent)
		require.NoError(t, err)
	}

	// Test listing
	agents, err = store.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 3)

	// Verify all agents are returned and are copies
	agentIDs := make(map[string]bool)
	for _, agent := range agents {
		agentIDs[agent.ID] = true
	}

	assert.True(t, agentIDs["agent-1"])
	assert.True(t, agentIDs["agent-2"])
	assert.True(t, agentIDs["agent-3"])

	// Verify they're copies
	agents[0].Name = "Modified Name"
	
	retrievedAgent, err := store.GetAgent("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "Agent 1", retrievedAgent.Name) // Should be unchanged
}

func TestMemoryAgentStore_DeleteAgent(t *testing.T) {
	store := NewMemoryAgentStore()

	agent := &types.Agent{
		ID:   "test-agent-delete",
		Name: "Agent to Delete",
		Type: types.AgentTypeResearch,
		Status: types.AgentStatusActive,
	}

	// Save agent first
	err := store.SaveAgent(agent)
	require.NoError(t, err)

	// Verify it exists
	_, err = store.GetAgent("test-agent-delete")
	require.NoError(t, err)

	// Delete agent
	err = store.DeleteAgent("test-agent-delete")
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.GetAgent("test-agent-delete")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryAgentStore_DeleteAgent_Errors(t *testing.T) {
	store := NewMemoryAgentStore()

	tests := []struct {
		name        string
		agentID     string
		expectedErr string
	}{
		{
			name:        "Empty ID",
			agentID:     "",
			expectedErr: "agent ID cannot be empty",
		},
		{
			name:        "Non-existent ID",
			agentID:     "non-existent",
			expectedErr: "agent with ID non-existent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.DeleteAgent(tt.agentID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMemoryAgentStore_SaveExecution(t *testing.T) {
	store := NewMemoryAgentStore()

	now := time.Now()
	endTime := now.Add(5 * time.Second)

	execution := &types.ExecutionResult{
		ExecutionID:  "exec-1",
		AgentID:      "agent-1",
		Status:       types.ExecutionStatusCompleted,
		StartTime:    now,
		EndTime:      &endTime,
		Duration:     5 * time.Second,
		Results:      map[string]interface{}{"success": true},
		Outputs:      map[string]interface{}{"result": "completed"},
		ErrorMessage: "",
		Logs: []types.ExecutionLog{
			{
				Timestamp: now,
				Level:     types.LogLevelInfo,
				Message:   "Execution started",
			},
		},
		StepResults: []types.StepResult{
			{
				StepID:    "step-1",
				Name:      "Test Step",
				Status:    types.ExecutionStatusCompleted,
				StartTime: now,
				EndTime:   &endTime,
				Duration:  5 * time.Second,
			},
		},
	}

	// Test successful save
	err := store.SaveExecution(execution)
	require.NoError(t, err)
}

func TestMemoryAgentStore_SaveExecution_ValidationErrors(t *testing.T) {
	store := NewMemoryAgentStore()

	tests := []struct {
		name        string
		execution   *types.ExecutionResult
		expectedErr string
	}{
		{
			name:        "Nil execution",
			execution:   nil,
			expectedErr: "execution cannot be nil",
		},
		{
			name: "Empty execution ID",
			execution: &types.ExecutionResult{
				AgentID: "agent-1",
			},
			expectedErr: "execution ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveExecution(tt.execution)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMemoryAgentStore_GetExecution(t *testing.T) {
	store := NewMemoryAgentStore()

	execution := &types.ExecutionResult{
		ExecutionID: "exec-get-test",
		AgentID:     "agent-1",
		Status:      types.ExecutionStatusCompleted,
		StartTime:   time.Now(),
		Results:     map[string]interface{}{"test": "result"},
	}

	// Save execution first
	err := store.SaveExecution(execution)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := store.GetExecution("exec-get-test")
	require.NoError(t, err)
	assert.Equal(t, execution.ExecutionID, retrieved.ExecutionID)
	assert.Equal(t, execution.AgentID, retrieved.AgentID)
	assert.Equal(t, execution.Status, retrieved.Status)

	// Verify it's a copy
	retrieved.Status = types.ExecutionStatusFailed
	
	retrievedAgain, err := store.GetExecution("exec-get-test")
	require.NoError(t, err)
	assert.Equal(t, types.ExecutionStatusCompleted, retrievedAgain.Status)
}

func TestMemoryAgentStore_GetExecution_Errors(t *testing.T) {
	store := NewMemoryAgentStore()

	tests := []struct {
		name        string
		executionID string
		expectedErr string
	}{
		{
			name:        "Empty ID",
			executionID: "",
			expectedErr: "execution ID cannot be empty",
		},
		{
			name:        "Non-existent ID",
			executionID: "non-existent",
			expectedErr: "execution with ID non-existent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution, err := store.GetExecution(tt.executionID)
			require.Error(t, err)
			assert.Nil(t, execution)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMemoryAgentStore_ListExecutions(t *testing.T) {
	store := NewMemoryAgentStore()

	// Add executions for different agents
	executions := []*types.ExecutionResult{
		{
			ExecutionID: "exec-1",
			AgentID:     "agent-1",
			Status:      types.ExecutionStatusCompleted,
			StartTime:   time.Now(),
		},
		{
			ExecutionID: "exec-2",
			AgentID:     "agent-1",
			Status:      types.ExecutionStatusFailed,
			StartTime:   time.Now(),
		},
		{
			ExecutionID: "exec-3",
			AgentID:     "agent-2",
			Status:      types.ExecutionStatusCompleted,
			StartTime:   time.Now(),
		},
	}

	for _, exec := range executions {
		err := store.SaveExecution(exec)
		require.NoError(t, err)
	}

	// Test listing executions for agent-1
	agent1Executions, err := store.ListExecutions("agent-1")
	require.NoError(t, err)
	assert.Len(t, agent1Executions, 2)

	// Verify correct executions are returned
	executionIDs := make(map[string]bool)
	for _, exec := range agent1Executions {
		executionIDs[exec.ExecutionID] = true
		assert.Equal(t, "agent-1", exec.AgentID)
	}
	assert.True(t, executionIDs["exec-1"])
	assert.True(t, executionIDs["exec-2"])

	// Test listing executions for agent-2
	agent2Executions, err := store.ListExecutions("agent-2")
	require.NoError(t, err)
	assert.Len(t, agent2Executions, 1)
	assert.Equal(t, "exec-3", agent2Executions[0].ExecutionID)

	// Test listing executions for non-existent agent
	nonExistentExecutions, err := store.ListExecutions("non-existent-agent")
	require.NoError(t, err)
	assert.Empty(t, nonExistentExecutions)
}

func TestMemoryAgentStore_ListExecutions_EmptyAgentID(t *testing.T) {
	store := NewMemoryAgentStore()

	executions, err := store.ListExecutions("")
	require.Error(t, err)
	assert.Nil(t, executions)
	assert.Contains(t, err.Error(), "agent ID cannot be empty")
}

func TestMemoryAgentStore_ListAllExecutions(t *testing.T) {
	store := NewMemoryAgentStore()

	// Add executions for multiple agents
	executions := []*types.ExecutionResult{
		{ExecutionID: "exec-1", AgentID: "agent-1", StartTime: time.Now()},
		{ExecutionID: "exec-2", AgentID: "agent-1", StartTime: time.Now()},
		{ExecutionID: "exec-3", AgentID: "agent-2", StartTime: time.Now()},
		{ExecutionID: "exec-4", AgentID: "agent-3", StartTime: time.Now()},
	}

	for _, exec := range executions {
		err := store.SaveExecution(exec)
		require.NoError(t, err)
	}

	// Test listing all executions
	allExecutions, err := store.ListAllExecutions()
	require.NoError(t, err)
	assert.Len(t, allExecutions, 4)

	// Verify all executions are returned
	executionIDs := make(map[string]bool)
	for _, exec := range allExecutions {
		executionIDs[exec.ExecutionID] = true
	}
	assert.True(t, executionIDs["exec-1"])
	assert.True(t, executionIDs["exec-2"])
	assert.True(t, executionIDs["exec-3"])
	assert.True(t, executionIDs["exec-4"])

	// Verify they're copies
	allExecutions[0].Status = types.ExecutionStatusFailed
	
	retrieved, err := store.GetExecution("exec-1")
	require.NoError(t, err)
	assert.NotEqual(t, types.ExecutionStatusFailed, retrieved.Status)
}

func TestMemoryAgentStore_GetStats(t *testing.T) {
	store := NewMemoryAgentStore()

	// Add some agents
	agents := []*types.Agent{
		{ID: "agent-1", Type: types.AgentTypeResearch, Status: types.AgentStatusActive},
		{ID: "agent-2", Type: types.AgentTypeResearch, Status: types.AgentStatusActive},
		{ID: "agent-3", Type: types.AgentTypeWorkflow, Status: types.AgentStatusActive},
		{ID: "agent-4", Type: types.AgentTypeMonitoring, Status: types.AgentStatusInactive},
	}

	for _, agent := range agents {
		err := store.SaveAgent(agent)
		require.NoError(t, err)
	}

	// Add some executions
	executions := []*types.ExecutionResult{
		{ExecutionID: "exec-1", AgentID: "agent-1", Status: types.ExecutionStatusCompleted, StartTime: time.Now()},
		{ExecutionID: "exec-2", AgentID: "agent-1", Status: types.ExecutionStatusFailed, StartTime: time.Now()},
		{ExecutionID: "exec-3", AgentID: "agent-2", Status: types.ExecutionStatusCompleted, StartTime: time.Now()},
		{ExecutionID: "exec-4", AgentID: "agent-3", Status: types.ExecutionStatusRunning, StartTime: time.Now()},
	}

	for _, exec := range executions {
		err := store.SaveExecution(exec)
		require.NoError(t, err)
	}

	// Get stats
	stats := store.GetStats()

	assert.Equal(t, 4, stats["total_agents"])
	assert.Equal(t, 4, stats["total_executions"])

	// Check agent type counts
	agentTypeCount := stats["agents_by_type"].(map[string]int)
	assert.Equal(t, 2, agentTypeCount["research"])
	assert.Equal(t, 1, agentTypeCount["workflow"])
	assert.Equal(t, 1, agentTypeCount["monitoring"])

	// Check execution status counts
	executionStatusCount := stats["executions_by_status"].(map[string]int)
	assert.Equal(t, 2, executionStatusCount["completed"])
	assert.Equal(t, 1, executionStatusCount["failed"])
	assert.Equal(t, 1, executionStatusCount["running"])
}

func TestMemoryAgentStore_Clear(t *testing.T) {
	store := NewMemoryAgentStore()

	// Add some data
	agent := &types.Agent{ID: "agent-1", Name: "Test Agent", Status: types.AgentStatusActive}
	execution := &types.ExecutionResult{ExecutionID: "exec-1", AgentID: "agent-1", StartTime: time.Now()}

	err := store.SaveAgent(agent)
	require.NoError(t, err)

	err = store.SaveExecution(execution)
	require.NoError(t, err)

	// Verify data exists
	agents, err := store.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)

	executions, err := store.ListAllExecutions()
	require.NoError(t, err)
	assert.Len(t, executions, 1)

	// Clear the store
	err = store.Clear()
	require.NoError(t, err)

	// Verify data is gone
	agents, err = store.ListAgents()
	require.NoError(t, err)
	assert.Empty(t, agents)

	executions, err = store.ListAllExecutions()
	require.NoError(t, err)
	assert.Empty(t, executions)

	// Verify stats are reset
	stats := store.GetStats()
	assert.Equal(t, 0, stats["total_agents"])
	assert.Equal(t, 0, stats["total_executions"])
}

func TestMemoryAgentStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryAgentStore()
	
	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	numAgentsPerGoroutine := 10

	// Concurrent agent saves
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numAgentsPerGoroutine; j++ {
				agent := &types.Agent{
					ID:     fmt.Sprintf("agent-%d-%d", routineID, j),
					Name:   fmt.Sprintf("Agent %d-%d", routineID, j),
					Type:   types.AgentTypeResearch,
					Status: types.AgentStatusActive,
				}
				err := store.SaveAgent(agent)
				assert.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all agents were saved
	agents, err := store.ListAgents()
	require.NoError(t, err)
	assert.Len(t, agents, numGoroutines*numAgentsPerGoroutine)

	// Test concurrent reads while writing
	wg.Add(numGoroutines * 2)
	
	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numAgentsPerGoroutine; j++ {
				agentID := fmt.Sprintf("agent-%d-%d", routineID, j)
				agent, err := store.GetAgent(agentID)
				assert.NoError(t, err)
				assert.Equal(t, agentID, agent.ID)
			}
		}(i)
	}

	// Writers (executions)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numAgentsPerGoroutine; j++ {
				execution := &types.ExecutionResult{
					ExecutionID: fmt.Sprintf("exec-%d-%d", routineID, j),
					AgentID:     fmt.Sprintf("agent-%d-%d", routineID, j),
					Status:      types.ExecutionStatusCompleted,
					StartTime:   time.Now(),
				}
				err := store.SaveExecution(execution)
				assert.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify final state
	executions, err := store.ListAllExecutions()
	require.NoError(t, err)
	assert.Len(t, executions, numGoroutines*numAgentsPerGoroutine)
}

func TestMemoryAgentStore_DataIsolation(t *testing.T) {
	store := NewMemoryAgentStore()

	originalAgent := &types.Agent{
		ID:   "isolation-test",
		Name: "Original Agent",
		Type: types.AgentTypeResearch,
		Status: types.AgentStatusActive,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 5,
		},
	}

	// Save original
	err := store.SaveAgent(originalAgent)
	require.NoError(t, err)

	// Retrieve and modify
	retrieved1, err := store.GetAgent("isolation-test")
	require.NoError(t, err)
	retrieved1.Name = "Modified Agent 1"
	retrieved1.Config.MaxConcurrentExecutions = 10

	// Retrieve again
	retrieved2, err := store.GetAgent("isolation-test")
	require.NoError(t, err)

	// Verify isolation - changes to retrieved1 don't affect retrieved2
	assert.Equal(t, "Original Agent", retrieved2.Name)
	assert.Equal(t, 5, retrieved2.Config.MaxConcurrentExecutions)

	// Modify original after save and verify it doesn't affect stored copy
	originalAgent.Name = "Modified Original"
	
	retrieved3, err := store.GetAgent("isolation-test")
	require.NoError(t, err)
	assert.Equal(t, "Original Agent", retrieved3.Name)
}