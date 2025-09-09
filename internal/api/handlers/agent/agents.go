package agent

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

// AgentsHandler handles agent-related HTTP requests
type AgentsHandler struct {
	manager *agents.Manager
}

// NewAgentsHandler creates a new agents handler
func NewAgentsHandler(cfg *config.Config, llmService interface{}, mcpService interface{}) (*AgentsHandler, error) {
	// Create agent config from main config
	agentConfig := &agents.Config{
		StorageBackend:          "memory",
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          30 * time.Minute,
		EnableMetrics:           true,
		MCPIntegration:          cfg.MCP.Enabled,
	}
	
	if cfg.Agents != nil {
		if cfg.Agents.StorageType != "" {
			agentConfig.StorageBackend = cfg.Agents.StorageType
		}
		if cfg.Agents.MaxConcurrent > 0 {
			agentConfig.MaxConcurrentExecutions = cfg.Agents.MaxConcurrent
		}
		if cfg.Agents.DefaultTimeout > 0 {
			agentConfig.DefaultTimeout = time.Duration(cfg.Agents.DefaultTimeout) * time.Second
		}
	}
	
	// Create agent manager
	manager, err := agents.NewManager(mcpService, agentConfig)
	if err != nil {
		return nil, err
	}

	return &AgentsHandler{
		manager: manager,
	}, nil
}

// ListAgents returns all available agents
func (h *AgentsHandler) ListAgents(c *gin.Context) {
	agents, err := h.manager.ListAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agents": agents,
			"count":  len(agents),
		},
	})
}

// GetAgent returns details of a specific agent
func (h *AgentsHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	agent, err := h.manager.GetAgent(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Agent not found",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    agent,
	})
}

// CreateAgentRequest represents a request to create an agent
type CreateAgentRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// CreateAgent creates a new agent
func (h *AgentsHandler) CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert type string to AgentType
	agentType := types.AgentType(req.Type)
	
	// Create agent config
	agentConfig := types.AgentConfig{
		MaxConcurrentExecutions: 5,
		DefaultTimeout:          30 * time.Minute,
		EnableMetrics:           true,
		AutonomyLevel:           types.AutonomyManual,
	}
	
	// Create agent
	agent := &types.Agent{
		ID:          "", // Will be generated
		Name:        req.Name,
		Type:        agentType,
		Description: req.Description,
		Config:      agentConfig,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the agent
	_, err := h.manager.CreateAgent(agent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id": agent.ID,
			"message":  "Agent created successfully",
		},
	})
}

// ExecuteAgentRequest represents a request to execute an agent
type ExecuteAgentRequest struct {
	AgentID string                 `json:"agent_id" binding:"required"`
	Input   map[string]interface{} `json:"input"`
	Timeout int                    `json:"timeout"` // timeout in seconds
}

// ExecuteAgent executes an agent
func (h *AgentsHandler) ExecuteAgent(c *gin.Context) {
	var req ExecuteAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set timeout
	timeout := 60 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	// Execute agent
	result, err := h.manager.ExecuteAgent(ctx, req.AgentID, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetActiveExecutions returns currently running executions
func (h *AgentsHandler) GetActiveExecutions(c *gin.Context) {
	executions := h.manager.GetActiveExecutions()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"executions": executions,
			"count":      len(executions),
		},
	})
}

// CancelExecutionRequest represents a request to cancel an execution
type CancelExecutionRequest struct {
	ExecutionID string `json:"execution_id" binding:"required"`
}

// CancelExecution cancels a running execution
func (h *AgentsHandler) CancelExecution(c *gin.Context) {
	var req CancelExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := h.manager.CancelExecution(req.ExecutionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Execution cancelled successfully",
	})
}

// GetExecutionHistory returns execution history for an agent
func (h *AgentsHandler) GetExecutionHistory(c *gin.Context) {
	agentID := c.Param("id")
	
	history, err := h.manager.GetExecutionHistory(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history": history,
			"count":   len(history),
		},
	})
}

// Close shuts down the agents handler
func (h *AgentsHandler) Close() error {
	// The manager doesn't have a Close method, so nothing to do
	return nil
}