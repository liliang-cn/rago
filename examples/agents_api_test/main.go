package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/api"
	"github.com/liliang-cn/rago/v2/pkg/agents/tools"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

func main() {
	fmt.Println("🧪 Testing RAGO Agents API Integration")
	fmt.Println("======================================")

	// Initialize mock MCP client
	mcpClient := tools.NewMockMCPClient()

	// Initialize agents manager
	manager, err := agents.NewManager(mcpClient, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create manager: %v", err))
	}

	// Create router and register routes
	router := mux.NewRouter()
	manager.RegisterHTTPRoutes(router)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	baseURL := server.URL + "/api/agents"
	
	// Test 1: Create agent via API
	fmt.Println("\n📝 Test 1: Create Agent via API")
	createReq := api.CreateAgentRequest{
		Name:        "API Test Agent",
		Description: "Agent created through API",
		Type:        types.AgentTypeWorkflow,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 1,
			DefaultTimeout:          60,
			EnableMetrics:           true,
			AutonomyLevel:          types.AutonomyManual,
		},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "test_step",
					Name: "Test Step",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"test": "value",
					},
					Outputs: map[string]string{
						"test": "result",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(createReq)
	resp, err := http.Post(baseURL+"/agents", "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(fmt.Sprintf("Failed to create agent: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var createResp api.CreateAgentResponse
	json.Unmarshal(respBody, &createResp)

	fmt.Printf("✅ Agent created: %s (ID: %s)\n", createResp.Agent.Name, createResp.Agent.ID)
	agentID := createResp.Agent.ID

	// Test 2: List agents
	fmt.Println("\n📋 Test 2: List Agents")
	resp, err = http.Get(baseURL + "/agents")
	if err != nil {
		panic(fmt.Sprintf("Failed to list agents: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	var listResp api.ListAgentsResponse
	json.Unmarshal(respBody, &listResp)

	fmt.Printf("✅ Found %d agents\n", listResp.Count)
	for _, agent := range listResp.Agents {
		fmt.Printf("   - %s (%s): %s\n", agent.Name, agent.ID, agent.Type)
	}

	// Test 3: Get specific agent
	fmt.Println("\n🔍 Test 3: Get Specific Agent")
	resp, err = http.Get(fmt.Sprintf("%s/agents/%s", baseURL, agentID))
	if err != nil {
		panic(fmt.Sprintf("Failed to get agent: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	var agent types.Agent
	json.Unmarshal(respBody, &agent)

	fmt.Printf("✅ Retrieved agent: %s\n", agent.Name)
	fmt.Printf("   Status: %s\n", agent.Status)
	fmt.Printf("   Type: %s\n", agent.Type)

	// Test 4: Execute agent
	fmt.Println("\n🚀 Test 4: Execute Agent")
	execReq := api.ExecuteAgentRequest{
		Variables: map[string]interface{}{
			"input": "test data",
		},
	}

	body, _ = json.Marshal(execReq)
	resp, err = http.Post(
		fmt.Sprintf("%s/agents/%s/execute", baseURL, agentID),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to execute agent: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	var execResult types.ExecutionResult
	json.Unmarshal(respBody, &execResult)

	fmt.Printf("✅ Execution completed: %s\n", execResult.Status)
	fmt.Printf("   Duration: %v\n", execResult.Duration)
	fmt.Printf("   Steps executed: %d\n", len(execResult.StepResults))

	// Test 5: Get executions
	fmt.Println("\n📊 Test 5: Get Agent Executions")
	resp, err = http.Get(fmt.Sprintf("%s/agents/%s/executions", baseURL, agentID))
	if err != nil {
		panic(fmt.Sprintf("Failed to get executions: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	var execsResp api.ListExecutionsResponse
	json.Unmarshal(respBody, &execsResp)

	fmt.Printf("✅ Found %d executions\n", execsResp.Count)
	for _, exec := range execsResp.Executions {
		fmt.Printf("   - %s: %s (Duration: %v)\n", exec.ExecutionID, exec.Status, exec.Duration)
	}

	// Test 6: Get workflow templates
	fmt.Println("\n📦 Test 6: Get Workflow Templates")
	resp, err = http.Get(baseURL + "/workflows/templates")
	if err != nil {
		panic(fmt.Sprintf("Failed to get templates: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	var templatesResp api.WorkflowTemplatesResponse
	json.Unmarshal(respBody, &templatesResp)

	fmt.Printf("✅ Found %d templates\n", templatesResp.Count)
	for _, template := range templatesResp.Templates {
		fmt.Printf("   - %s: %s\n", template.Name, template.Description)
	}

	// Test 7: Delete agent
	fmt.Println("\n🗑️  Test 7: Delete Agent")
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/agents/%s", baseURL, agentID), nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to delete agent: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ Agent %s deleted successfully\n", agentID)
	}

	fmt.Println("\n✨ All API tests completed successfully!")
}