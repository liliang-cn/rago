package agents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Handler handles Agent-related HTTP requests
type Handler struct {
	client *client.Client
}

// NewHandler creates a new Agent handler
func NewHandler(client *client.Client) *Handler {
	return &Handler{
		client: client,
	}
}

// ListWorkflows lists all workflows
// @Summary List workflows
// @Description Get a list of all defined workflows
// @Tags Agents
// @Produce json
// @Success 200 {array} core.WorkflowInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows [get]
func (h *Handler) ListWorkflows(c *gin.Context) {
	workflows := h.client.Agents().ListWorkflows()
	c.JSON(http.StatusOK, workflows)
}

// CreateWorkflow creates a new workflow
// @Summary Create workflow
// @Description Create a new workflow definition
// @Tags Agents
// @Accept json
// @Produce json
// @Param request body core.WorkflowDefinition true "Workflow definition"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows [post]
func (h *Handler) CreateWorkflow(c *gin.Context) {
	var definition core.WorkflowDefinition
	if err := c.ShouldBindJSON(&definition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.client.Agents().CreateWorkflow(definition)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "workflow created successfully"})
}

// DeleteWorkflow deletes a workflow
// @Summary Delete workflow
// @Description Delete a workflow by name
// @Tags Agents
// @Param name path string true "Workflow name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows/{name} [delete]
func (h *Handler) DeleteWorkflow(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow name required"})
		return
	}

	err := h.client.Agents().DeleteWorkflow(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted successfully"})
}

// ExecuteWorkflow executes a workflow
// @Summary Execute workflow
// @Description Execute a workflow with the provided parameters
// @Tags Agents
// @Accept json
// @Produce json
// @Param name path string true "Workflow name"
// @Param request body core.WorkflowRequest true "Workflow execution request"
// @Success 200 {object} core.WorkflowResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows/{name}/execute [post]
func (h *Handler) ExecuteWorkflow(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow name required"})
		return
	}

	var req core.WorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set workflow name from path
	req.WorkflowName = name

	resp, err := h.client.Agents().ExecuteWorkflow(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ScheduleWorkflow schedules a workflow
// @Summary Schedule workflow
// @Description Schedule a workflow to run at specified times
// @Tags Agents
// @Accept json
// @Produce json
// @Param name path string true "Workflow name"
// @Param request body core.ScheduleConfig true "Schedule configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows/{name}/schedule [post]
func (h *Handler) ScheduleWorkflow(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow name required"})
		return
	}

	var schedule core.ScheduleConfig
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.client.Agents().ScheduleWorkflow(name, schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow scheduled successfully"})
}

// ListAgents lists all agents
// @Summary List agents
// @Description Get a list of all defined agents
// @Tags Agents
// @Produce json
// @Success 200 {array} core.AgentInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/agents [get]
func (h *Handler) ListAgents(c *gin.Context) {
	agents := h.client.Agents().ListAgents()
	c.JSON(http.StatusOK, agents)
}

// CreateAgent creates a new agent
// @Summary Create agent
// @Description Create a new agent definition
// @Tags Agents
// @Accept json
// @Produce json
// @Param request body core.AgentDefinition true "Agent definition"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/agents [post]
func (h *Handler) CreateAgent(c *gin.Context) {
	var definition core.AgentDefinition
	if err := c.ShouldBindJSON(&definition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.client.Agents().CreateAgent(definition)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "agent created successfully"})
}

// DeleteAgent deletes an agent
// @Summary Delete agent
// @Description Delete an agent by name
// @Tags Agents
// @Param name path string true "Agent name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/agents/{name} [delete]
func (h *Handler) DeleteAgent(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent name required"})
		return
	}

	err := h.client.Agents().DeleteAgent(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent deleted successfully"})
}

// ExecuteAgent executes an agent
// @Summary Execute agent
// @Description Execute an agent with the provided parameters
// @Tags Agents
// @Accept json
// @Produce json
// @Param name path string true "Agent name"
// @Param request body core.AgentRequest true "Agent execution request"
// @Success 200 {object} core.AgentResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/agents/{name}/execute [post]
func (h *Handler) ExecuteAgent(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent name required"})
		return
	}

	var req core.AgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set agent name from path
	req.AgentName = name

	resp, err := h.client.Agents().ExecuteAgent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetScheduledTasks gets all scheduled tasks
// @Summary Get scheduled tasks
// @Description Get a list of all scheduled tasks
// @Tags Agents
// @Produce json
// @Success 200 {array} core.ScheduledTask
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/scheduled [get]
func (h *Handler) GetScheduledTasks(c *gin.Context) {
	tasks := h.client.Agents().GetScheduledTasks()
	c.JSON(http.StatusOK, tasks)
}

// StreamWorkflowExecution streams workflow execution progress
// @Summary Stream workflow execution
// @Description Stream real-time workflow execution progress using Server-Sent Events
// @Tags Agents
// @Accept json
// @Produce text/event-stream
// @Param name path string true "Workflow name"
// @Param request body core.WorkflowRequest true "Workflow execution request"
// @Success 200 {string} string "Event stream"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents/workflows/{name}/stream [post]
func (h *Handler) StreamWorkflowExecution(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow name required"})
		return
	}

	var req core.WorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Set workflow name from path
	req.WorkflowName = name

	// Execute workflow and stream progress
	// In production, this would stream actual progress events
	resp, err := h.client.Agents().ExecuteWorkflow(c.Request.Context(), req)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Send the result
	data, _ := json.Marshal(resp)
	fmt.Fprintf(w, "event: result\ndata: %s\n\n", data)
	flusher.Flush()

	// Send done event
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}