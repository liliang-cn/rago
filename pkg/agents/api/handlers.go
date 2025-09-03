package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/liliang-cn/rago/v2/pkg/agents/core"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// AgentHandler provides HTTP handlers for agent management
type AgentHandler struct {
	executor *core.AgentExecutor
	storage  core.AgentStorage
	factory  *core.AgentFactory
	workflow *core.WorkflowEngine
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(executor *core.AgentExecutor, storage core.AgentStorage) *AgentHandler {
	return &AgentHandler{
		executor: executor,
		storage:  storage,
		factory:  core.NewAgentFactory(),
		workflow: core.NewWorkflowEngine(executor),
	}
}

// RegisterRoutes registers all agent API routes
func (h *AgentHandler) RegisterRoutes(router *mux.Router) {
	// Agent management
	router.HandleFunc("/agents", h.CreateAgent).Methods("POST")
	router.HandleFunc("/agents", h.ListAgents).Methods("GET")
	router.HandleFunc("/agents/{id}", h.GetAgent).Methods("GET")
	router.HandleFunc("/agents/{id}", h.UpdateAgent).Methods("PUT")
	router.HandleFunc("/agents/{id}", h.DeleteAgent).Methods("DELETE")
	
	// Execution control
	router.HandleFunc("/agents/{id}/execute", h.ExecuteAgent).Methods("POST")
	router.HandleFunc("/agents/{id}/executions", h.GetExecutions).Methods("GET")
	router.HandleFunc("/executions/{exec_id}", h.GetExecution).Methods("GET")
	router.HandleFunc("/executions/{exec_id}/stop", h.StopExecution).Methods("POST")
	
	// Workflow management
	router.HandleFunc("/workflows/templates", h.GetWorkflowTemplates).Methods("GET")
	router.HandleFunc("/workflows/validate", h.ValidateWorkflow).Methods("POST")
	
	// Status and monitoring
	router.HandleFunc("/agents/status", h.GetAgentsStatus).Methods("GET")
	router.HandleFunc("/executions/active", h.GetActiveExecutions).Methods("GET")
}

// CreateAgent creates a new agent
func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	// Generate ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	
	// Create agent structure
	agent := &types.Agent{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Workflow:    req.Workflow,
		Status:      types.AgentStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Validate agent
	agentImpl, err := h.factory.CreateAgent(agent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusBadRequest)
		return
	}
	
	if err := agentImpl.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Agent validation failed: %v", err), http.StatusBadRequest)
		return
	}
	
	// Save agent
	if err := h.storage.SaveAgent(agent); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save agent: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateAgentResponse{
		Agent:   agent,
		Message: "Agent created successfully",
	})
}

// ListAgents lists all agents
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.storage.ListAgents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list agents: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListAgentsResponse{
		Agents: agents,
		Count:  len(agents),
	})
}

// GetAgent retrieves a specific agent
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	agent, err := h.storage.GetAgent(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// UpdateAgent updates an existing agent
func (h *AgentHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var req UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	// Get existing agent
	agent, err := h.storage.GetAgent(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}
	
	// Update fields
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.Config != nil {
		agent.Config = *req.Config
	}
	if req.Workflow != nil {
		agent.Workflow = *req.Workflow
	}
	
	agent.UpdatedAt = time.Now()
	
	// Save updated agent
	if err := h.storage.SaveAgent(agent); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update agent: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

// DeleteAgent deletes an agent
func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := h.storage.DeleteAgent(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete agent: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Agent deleted successfully",
	})
}

// ExecuteAgent executes an agent
func (h *AgentHandler) ExecuteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var req ExecuteAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	// Get agent
	agent, err := h.storage.GetAgent(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}
	
	// Create agent implementation
	agentImpl, err := h.factory.CreateAgent(agent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Execute agent
	ctx := context.Background()
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}
	
	result, err := h.workflow.ExecuteWorkflow(ctx, agentImpl, req.Variables)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetExecutions returns execution history for an agent
func (h *AgentHandler) GetExecutions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	
	executions, err := h.storage.ListExecutions(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list executions: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListExecutionsResponse{
		Executions: executions,
		Count:      len(executions),
	})
}

// GetExecution returns details of a specific execution
func (h *AgentHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	execID := vars["exec_id"]
	
	execution, err := h.storage.GetExecution(execID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution not found: %v", err), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// StopExecution stops a running execution
func (h *AgentHandler) StopExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	execID := vars["exec_id"]
	
	if err := h.executor.CancelExecution(execID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop execution: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Execution stopped successfully",
	})
}

// GetWorkflowTemplates returns available workflow templates
func (h *AgentHandler) GetWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.getBuiltinTemplates()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkflowTemplatesResponse{
		Templates: templates,
		Count:     len(templates),
	})
}

// ValidateWorkflow validates a workflow definition
func (h *AgentHandler) ValidateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflow types.WorkflowSpec
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		http.Error(w, fmt.Sprintf("Invalid workflow definition: %v", err), http.StatusBadRequest)
		return
	}
	
	validator := core.NewWorkflowValidator()
	if err := validator.Validate(workflow); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateWorkflowResponse{
			Valid:        false,
			ErrorMessage: err.Error(),
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ValidateWorkflowResponse{
		Valid:   true,
		Message: "Workflow is valid",
	})
}

// GetAgentsStatus returns overall status of all agents
func (h *AgentHandler) GetAgentsStatus(w http.ResponseWriter, r *http.Request) {
	agents, err := h.storage.ListAgents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent status: %v", err), http.StatusInternalServerError)
		return
	}
	
	status := AgentsStatusResponse{
		TotalAgents: len(agents),
		StatusCount: make(map[string]int),
		TypeCount:   make(map[string]int),
		Timestamp:   time.Now(),
	}
	
	for _, agent := range agents {
		status.StatusCount[string(agent.Status)]++
		status.TypeCount[string(agent.Type)]++
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetActiveExecutions returns currently running executions
func (h *AgentHandler) GetActiveExecutions(w http.ResponseWriter, r *http.Request) {
	activeExecutions := h.executor.GetActiveExecutions()
	
	executions := make([]ActiveExecutionInfo, 0, len(activeExecutions))
	for id, instance := range activeExecutions {
		executions = append(executions, ActiveExecutionInfo{
			ExecutionID:  id,
			AgentID:      instance.Agent.GetID(),
			AgentName:    instance.Agent.GetName(),
			Status:       string(instance.Status),
			StartTime:    instance.StartTime,
			CurrentStep:  instance.CurrentStep,
			Duration:     time.Since(instance.StartTime),
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ActiveExecutionsResponse{
		Executions: executions,
		Count:      len(executions),
	})
}

// getBuiltinTemplates returns built-in workflow templates
func (h *AgentHandler) getBuiltinTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		{
			ID:          "document_analyzer",
			Name:        "Document Analyzer",
			Description: "Analyze documents and extract insights",
			Category:    "research",
			Workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:   "extract_text",
						Name: "Extract Text",
						Type: types.StepTypeTool,
						Tool: "mcp_pdf_extract",
						Inputs: map[string]interface{}{
							"file_path": "{{trigger.file_path}}",
						},
						Outputs: map[string]string{
							"text": "text_content",
						},
					},
					{
						ID:   "analyze_content",
						Name: "Analyze Content",
						Type: types.StepTypeTool,
						Tool: "llm_analyze",
						Inputs: map[string]interface{}{
							"content": "{{text_content}}",
							"prompt":  "Analyze this document and extract key insights",
						},
						Outputs: map[string]string{
							"analysis": "insights",
						},
					},
				},
			},
		},
		{
			ID:          "data_pipeline",
			Name:        "Data Processing Pipeline",
			Description: "Process and transform data through multiple stages",
			Category:    "workflow",
			Workflow: types.WorkflowSpec{
				Steps: []types.WorkflowStep{
					{
						ID:   "fetch_data",
						Name: "Fetch Data",
						Type: types.StepTypeTool,
						Tool: "mcp_sqlite_query",
						Inputs: map[string]interface{}{
							"query": "SELECT * FROM raw_data",
						},
						Outputs: map[string]string{
							"result": "raw_data",
						},
					},
					{
						ID:   "transform_data",
						Name: "Transform Data",
						Type: types.StepTypeTool,
						Tool: "data_transform",
						Inputs: map[string]interface{}{
							"data":   "{{raw_data}}",
							"format": "json",
						},
						Outputs: map[string]string{
							"result": "transformed_data",
						},
					},
				},
			},
		},
	}
}