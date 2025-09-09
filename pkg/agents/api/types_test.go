package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAgentRequest_JSONSerialization(t *testing.T) {
	request := CreateAgentRequest{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent for API testing",
		Type:        types.AgentTypeResearch,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 5,
			DefaultTimeout:          10 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:           types.AutonomyScheduled,
		},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "step1",
					Name: "Test Step",
					Type: types.StepTypeTool,
					Tool: "test_tool",
				},
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(request)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled CreateAgentRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.ID, unmarshaled.ID)
	assert.Equal(t, request.Name, unmarshaled.Name)
	assert.Equal(t, request.Description, unmarshaled.Description)
	assert.Equal(t, request.Type, unmarshaled.Type)
	assert.Equal(t, request.Config, unmarshaled.Config)
	assert.Len(t, unmarshaled.Workflow.Steps, 1)
}

func TestUpdateAgentRequest_JSONSerialization(t *testing.T) {
	config := &types.AgentConfig{
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          15 * time.Minute,
	}
	
	workflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "updated_step",
				Name: "Updated Step",
				Type: types.StepTypeVariable,
			},
		},
	}

	request := UpdateAgentRequest{
		Name:        "Updated Agent Name",
		Description: "Updated description",
		Status:      types.AgentStatusActive,
		Config:      config,
		Workflow:    workflow,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(request)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled UpdateAgentRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, request.Name, unmarshaled.Name)
	assert.Equal(t, request.Description, unmarshaled.Description)
	assert.Equal(t, request.Status, unmarshaled.Status)
	assert.NotNil(t, unmarshaled.Config)
	assert.NotNil(t, unmarshaled.Workflow)
	assert.Equal(t, config.MaxConcurrentExecutions, unmarshaled.Config.MaxConcurrentExecutions)
}

func TestExecuteAgentRequest(t *testing.T) {
	request := ExecuteAgentRequest{
		Variables: map[string]interface{}{
			"input_file": "/path/to/file.txt",
			"max_results": 10,
			"enabled": true,
		},
		Timeout: 300, // 5 minutes in seconds
		UserID:  "user-123",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(request)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled ExecuteAgentRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Note: JSON unmarshaling converts numbers to float64, so we need to handle that
	assert.Equal(t, true, unmarshaled.Variables["enabled"])
	assert.Equal(t, "/path/to/file.txt", unmarshaled.Variables["input_file"])
	assert.Equal(t, float64(10), unmarshaled.Variables["max_results"]) // JSON numbers become float64
	assert.Equal(t, request.Timeout, unmarshaled.Timeout)
	assert.Equal(t, request.UserID, unmarshaled.UserID)
}

func TestCreateAgentResponse(t *testing.T) {
	agent := &types.Agent{
		ID:   "response-agent-1",
		Name: "Response Agent",
		Type: types.AgentTypeWorkflow,
		Status: types.AgentStatusActive,
	}

	response := CreateAgentResponse{
		Agent:   agent,
		Message: "Agent created successfully",
	}

	assert.Equal(t, agent, response.Agent)
	assert.Equal(t, "Agent created successfully", response.Message)

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "response-agent-1")
	assert.Contains(t, string(jsonData), "Agent created successfully")
}

func TestListAgentsResponse(t *testing.T) {
	agents := []*types.Agent{
		{ID: "agent-1", Name: "Agent 1", Type: types.AgentTypeResearch},
		{ID: "agent-2", Name: "Agent 2", Type: types.AgentTypeWorkflow},
		{ID: "agent-3", Name: "Agent 3", Type: types.AgentTypeMonitoring},
	}

	response := ListAgentsResponse{
		Agents: agents,
		Count:  len(agents),
	}

	assert.Len(t, response.Agents, 3)
	assert.Equal(t, 3, response.Count)

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	
	var unmarshaled ListAgentsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, 3, unmarshaled.Count)
	assert.Len(t, unmarshaled.Agents, 3)
}

func TestListExecutionsResponse(t *testing.T) {
	now := time.Now()
	executions := []*types.ExecutionResult{
		{
			ExecutionID: "exec-1",
			AgentID:     "agent-1",
			Status:      types.ExecutionStatusCompleted,
			StartTime:   now,
		},
		{
			ExecutionID: "exec-2",
			AgentID:     "agent-2",
			Status:      types.ExecutionStatusFailed,
			StartTime:   now,
		},
	}

	response := ListExecutionsResponse{
		Executions: executions,
		Count:      len(executions),
	}

	assert.Len(t, response.Executions, 2)
	assert.Equal(t, 2, response.Count)

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	
	var unmarshaled ListExecutionsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, 2, unmarshaled.Count)
	assert.Len(t, unmarshaled.Executions, 2)
}

func TestWorkflowTemplate(t *testing.T) {
	template := WorkflowTemplate{
		ID:          "template-1",
		Name:        "Document Processing Template",
		Description: "Template for processing documents",
		Category:    "document-processing",
		Author:      "System",
		Version:     "1.0.0",
		Tags:        []string{"document", "processing", "automation"},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "extract",
					Name: "Extract Content",
					Type: types.StepTypeTool,
					Tool: "document_extractor",
				},
				{
					ID:   "analyze",
					Name: "Analyze Content",
					Type: types.StepTypeTool,
					Tool: "content_analyzer",
				},
			},
		},
	}

	assert.Equal(t, "template-1", template.ID)
	assert.Equal(t, "Document Processing Template", template.Name)
	assert.Equal(t, "document-processing", template.Category)
	assert.Len(t, template.Tags, 3)
	assert.Len(t, template.Workflow.Steps, 2)

	// Test JSON serialization
	jsonData, err := json.Marshal(template)
	require.NoError(t, err)
	
	var unmarshaled WorkflowTemplate
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, template.ID, unmarshaled.ID)
	assert.Equal(t, template.Category, unmarshaled.Category)
	assert.Len(t, unmarshaled.Workflow.Steps, 2)
}

func TestWorkflowTemplatesResponse(t *testing.T) {
	templates := []WorkflowTemplate{
		{ID: "template-1", Name: "Template 1", Category: "category1"},
		{ID: "template-2", Name: "Template 2", Category: "category2"},
	}

	response := WorkflowTemplatesResponse{
		Templates: templates,
		Count:     len(templates),
	}

	assert.Len(t, response.Templates, 2)
	assert.Equal(t, 2, response.Count)
}

func TestValidateWorkflowResponse(t *testing.T) {
	tests := []struct {
		name     string
		response ValidateWorkflowResponse
	}{
		{
			name: "Valid workflow",
			response: ValidateWorkflowResponse{
				Valid:   true,
				Message: "Workflow is valid",
			},
		},
		{
			name: "Invalid workflow",
			response: ValidateWorkflowResponse{
				Valid:        false,
				Message:      "Workflow validation failed",
				ErrorMessage: "Step 'invalid_step' has missing required field 'tool'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON serialization
			jsonData, err := json.Marshal(tt.response)
			require.NoError(t, err)
			
			var unmarshaled ValidateWorkflowResponse
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)
			
			assert.Equal(t, tt.response.Valid, unmarshaled.Valid)
			assert.Equal(t, tt.response.Message, unmarshaled.Message)
			assert.Equal(t, tt.response.ErrorMessage, unmarshaled.ErrorMessage)
		})
	}
}

func TestAgentsStatusResponse(t *testing.T) {
	now := time.Now()
	response := AgentsStatusResponse{
		TotalAgents: 10,
		StatusCount: map[string]int{
			"active":   7,
			"inactive": 2,
			"error":    1,
		},
		TypeCount: map[string]int{
			"research":   4,
			"workflow":   3,
			"monitoring": 3,
		},
		Timestamp: now,
	}

	assert.Equal(t, 10, response.TotalAgents)
	assert.Equal(t, 7, response.StatusCount["active"])
	assert.Equal(t, 4, response.TypeCount["research"])
	assert.Equal(t, now, response.Timestamp)

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	
	var unmarshaled AgentsStatusResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, 10, unmarshaled.TotalAgents)
	assert.Len(t, unmarshaled.StatusCount, 3)
	assert.Len(t, unmarshaled.TypeCount, 3)
}

func TestActiveExecutionInfo(t *testing.T) {
	now := time.Now()
	info := ActiveExecutionInfo{
		ExecutionID: "exec-active-1",
		AgentID:     "agent-1",
		AgentName:   "Test Agent",
		Status:      "running",
		StartTime:   now,
		CurrentStep: "processing_step",
		Duration:    2 * time.Minute,
	}

	assert.Equal(t, "exec-active-1", info.ExecutionID)
	assert.Equal(t, "agent-1", info.AgentID)
	assert.Equal(t, "Test Agent", info.AgentName)
	assert.Equal(t, "running", info.Status)
	assert.Equal(t, "processing_step", info.CurrentStep)
	assert.Equal(t, 2*time.Minute, info.Duration)
}

func TestActiveExecutionsResponse(t *testing.T) {
	executions := []ActiveExecutionInfo{
		{
			ExecutionID: "exec-1",
			AgentID:     "agent-1",
			Status:      "running",
			StartTime:   time.Now(),
		},
		{
			ExecutionID: "exec-2",
			AgentID:     "agent-2",
			Status:      "running",
			StartTime:   time.Now(),
		},
	}

	response := ActiveExecutionsResponse{
		Executions: executions,
		Count:      len(executions),
	}

	assert.Len(t, response.Executions, 2)
	assert.Equal(t, 2, response.Count)
}

func TestErrorResponse(t *testing.T) {
	response := ErrorResponse{
		Error:   "Agent not found",
		Code:    "AGENT_NOT_FOUND",
		Details: "Agent with ID 'test-agent' does not exist",
	}

	assert.Equal(t, "Agent not found", response.Error)
	assert.Equal(t, "AGENT_NOT_FOUND", response.Code)
	assert.Equal(t, "Agent with ID 'test-agent' does not exist", response.Details)

	// Test JSON serialization
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	
	var unmarshaled ErrorResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.Error, unmarshaled.Error)
	assert.Equal(t, response.Code, unmarshaled.Code)
}

func TestPaginationRequest(t *testing.T) {
	request := PaginationRequest{
		Page:     2,
		PageSize: 50,
	}

	assert.Equal(t, 2, request.Page)
	assert.Equal(t, 50, request.PageSize)

	// Test JSON serialization
	jsonData, err := json.Marshal(request)
	require.NoError(t, err)
	
	var unmarshaled PaginationRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, 2, unmarshaled.Page)
	assert.Equal(t, 50, unmarshaled.PageSize)
}

func TestPaginationResponse(t *testing.T) {
	response := PaginationResponse{
		Page:       3,
		PageSize:   20,
		TotalItems: 150,
		TotalPages: 8,
	}

	assert.Equal(t, 3, response.Page)
	assert.Equal(t, 20, response.PageSize)
	assert.Equal(t, 150, response.TotalItems)
	assert.Equal(t, 8, response.TotalPages)
}

func TestAgentFilter(t *testing.T) {
	filter := AgentFilter{
		Type:   types.AgentTypeResearch,
		Status: types.AgentStatusActive,
		Name:   "Research",
		Tag:    "important",
	}

	assert.Equal(t, types.AgentTypeResearch, filter.Type)
	assert.Equal(t, types.AgentStatusActive, filter.Status)
	assert.Equal(t, "Research", filter.Name)
	assert.Equal(t, "important", filter.Tag)
}

func TestExecutionFilter(t *testing.T) {
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	filter := ExecutionFilter{
		AgentID:   "agent-1",
		Status:    types.ExecutionStatusCompleted,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	assert.Equal(t, "agent-1", filter.AgentID)
	assert.Equal(t, types.ExecutionStatusCompleted, filter.Status)
	assert.Equal(t, &startTime, filter.StartTime)
	assert.Equal(t, &endTime, filter.EndTime)
}

func TestSortOption(t *testing.T) {
	sort := SortOption{
		Field:     "created_at",
		Direction: "desc",
	}

	assert.Equal(t, "created_at", sort.Field)
	assert.Equal(t, "desc", sort.Direction)
}

func TestBatchAgentOperation(t *testing.T) {
	operation := BatchAgentOperation{
		Operation: "activate",
		AgentIDs:  []string{"agent-1", "agent-2", "agent-3"},
	}

	assert.Equal(t, "activate", operation.Operation)
	assert.Len(t, operation.AgentIDs, 3)
	assert.Contains(t, operation.AgentIDs, "agent-1")
}

func TestBatchAgentResponse(t *testing.T) {
	response := BatchAgentResponse{
		Success:      []string{"agent-1", "agent-2"},
		Failed:       []string{"agent-3"},
		Errors:       []string{"Agent agent-3 not found"},
		TotalCount:   3,
		SuccessCount: 2,
		FailedCount:  1,
	}

	assert.Len(t, response.Success, 2)
	assert.Len(t, response.Failed, 1)
	assert.Len(t, response.Errors, 1)
	assert.Equal(t, 3, response.TotalCount)
	assert.Equal(t, 2, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
}

func TestAgentMetrics(t *testing.T) {
	lastExecution := time.Now()
	metrics := AgentMetrics{
		AgentID:              "agent-metrics-1",
		TotalExecutions:      100,
		SuccessfulExecutions: 95,
		FailedExecutions:     5,
		AverageExecutionTime: 2 * time.Minute,
		LastExecutionTime:    &lastExecution,
		SuccessRate:          0.95,
	}

	assert.Equal(t, "agent-metrics-1", metrics.AgentID)
	assert.Equal(t, 100, metrics.TotalExecutions)
	assert.Equal(t, 95, metrics.SuccessfulExecutions)
	assert.Equal(t, 5, metrics.FailedExecutions)
	assert.Equal(t, 2*time.Minute, metrics.AverageExecutionTime)
	assert.Equal(t, &lastExecution, metrics.LastExecutionTime)
	assert.Equal(t, 0.95, metrics.SuccessRate)
}

func TestSystemMetrics(t *testing.T) {
	now := time.Now()
	metrics := SystemMetrics{
		TotalAgents:          25,
		ActiveAgents:         20,
		TotalExecutions:      1000,
		ActiveExecutions:     5,
		ExecutionsLastHour:   50,
		AverageExecutionTime: 90 * time.Second,
		SystemUptime:         24 * time.Hour,
		Timestamp:            now,
	}

	assert.Equal(t, 25, metrics.TotalAgents)
	assert.Equal(t, 20, metrics.ActiveAgents)
	assert.Equal(t, 1000, metrics.TotalExecutions)
	assert.Equal(t, 5, metrics.ActiveExecutions)
	assert.Equal(t, 50, metrics.ExecutionsLastHour)
	assert.Equal(t, 90*time.Second, metrics.AverageExecutionTime)
	assert.Equal(t, 24*time.Hour, metrics.SystemUptime)
	assert.Equal(t, now, metrics.Timestamp)
}

func TestAgentExport(t *testing.T) {
	export := AgentExport{
		Agents: []*types.Agent{
			{ID: "agent-1", Name: "Agent 1"},
			{ID: "agent-2", Name: "Agent 2"},
		},
		Executions: []*types.ExecutionResult{
			{ExecutionID: "exec-1", AgentID: "agent-1"},
		},
		Templates: []WorkflowTemplate{
			{ID: "template-1", Name: "Template 1"},
		},
		Metadata: ExportMetadata{
			ExportTime:      time.Now(),
			Version:         "1.0.0",
			TotalAgents:     2,
			TotalExecutions: 1,
		},
	}

	assert.Len(t, export.Agents, 2)
	assert.Len(t, export.Executions, 1)
	assert.Len(t, export.Templates, 1)
	assert.Equal(t, 2, export.Metadata.TotalAgents)
	assert.Equal(t, "1.0.0", export.Metadata.Version)
}

func TestWSMessage(t *testing.T) {
	now := time.Now()
	message := WSMessage{
		Type: "execution_update",
		Payload: map[string]interface{}{
			"execution_id": "exec-1",
			"status":       "running",
		},
		Timestamp: now,
	}

	assert.Equal(t, "execution_update", message.Type)
	assert.NotNil(t, message.Payload)
	assert.Equal(t, now, message.Timestamp)
}

func TestWSExecutionUpdate(t *testing.T) {
	update := WSExecutionUpdate{
		ExecutionID:  "exec-ws-1",
		AgentID:      "agent-ws-1",
		Status:       types.ExecutionStatusRunning,
		CurrentStep:  "processing",
		Progress:     0.65,
		Results:      map[string]interface{}{"partial": "data"},
		ErrorMessage: "",
	}

	assert.Equal(t, "exec-ws-1", update.ExecutionID)
	assert.Equal(t, "agent-ws-1", update.AgentID)
	assert.Equal(t, types.ExecutionStatusRunning, update.Status)
	assert.Equal(t, "processing", update.CurrentStep)
	assert.Equal(t, 0.65, update.Progress)
	assert.NotNil(t, update.Results)
}

func TestWSAgentStatus(t *testing.T) {
	now := time.Now()
	status := WSAgentStatus{
		AgentID:   "agent-ws-status-1",
		Status:    types.AgentStatusActive,
		Message:   "Agent started successfully",
		Timestamp: now,
	}

	assert.Equal(t, "agent-ws-status-1", status.AgentID)
	assert.Equal(t, types.AgentStatusActive, status.Status)
	assert.Equal(t, "Agent started successfully", status.Message)
	assert.Equal(t, now, status.Timestamp)
}

func TestAPIConfig(t *testing.T) {
	config := APIConfig{
		Host:             "0.0.0.0",
		Port:             8080,
		EnableCORS:       true,
		EnableMetrics:    true,
		EnableWebSocket:  true,
		RateLimitEnabled: true,
		RateLimitRPS:     100,
		MaxRequestSize:   10 * 1024 * 1024, // 10MB
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		ShutdownTimeout:  5 * time.Second,
	}

	assert.Equal(t, "0.0.0.0", config.Host)
	assert.Equal(t, 8080, config.Port)
	assert.True(t, config.EnableCORS)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableWebSocket)
	assert.True(t, config.RateLimitEnabled)
	assert.Equal(t, 100, config.RateLimitRPS)
	assert.Equal(t, int64(10*1024*1024), config.MaxRequestSize)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)

	// Test JSON serialization
	jsonData, err := json.Marshal(config)
	require.NoError(t, err)
	
	var unmarshaled APIConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, config.Host, unmarshaled.Host)
	assert.Equal(t, config.Port, unmarshaled.Port)
	assert.Equal(t, config.ReadTimeout, unmarshaled.ReadTimeout)
}