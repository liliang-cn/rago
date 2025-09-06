package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/processor"
)

// HealthHandler provides comprehensive health check for all components
type HealthHandler struct {
	config     *config.Config
	processor  *processor.Service
	mcpService *mcp.MCPService
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(cfg *config.Config, proc *processor.Service, mcpSvc *mcp.MCPService) *HealthHandler {
	return &HealthHandler{
		config:     cfg,
		processor:  proc,
		mcpService: mcpSvc,
	}
}

// ComponentStatus represents the health status of a component
type ComponentStatus struct {
	Status   string                 `json:"status"` // "healthy", "degraded", "unhealthy", "disabled"
	Message  string                 `json:"message,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthResponse represents the overall health status
type HealthResponse struct {
	Status     string                     `json:"status"` // "healthy", "degraded", "unhealthy"
	Components map[string]ComponentStatus `json:"components"`
	Timestamp  string                     `json:"timestamp"`
	Version    string                     `json:"version"`
}

func (h *HealthHandler) Handle(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Components: make(map[string]ComponentStatus),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Version:    "3.0.0",
	}

	// Check RAG (core component)
	response.Components["rag"] = h.checkRAG(ctx)

	// Check MCP (optional component)
	response.Components["mcp"] = h.checkMCP(ctx)

	// Check LLM providers
	response.Components["llm"] = h.checkLLM(ctx)

	// Check Scheduler (if enabled)
	response.Components["scheduler"] = h.checkScheduler(ctx)

	// Check Agents (if enabled)
	response.Components["agents"] = h.checkAgents(ctx)

	// Determine overall status
	response.Status = h.determineOverallStatus(response.Components)

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// checkRAG checks the health of the RAG system
func (h *HealthHandler) checkRAG(ctx context.Context) ComponentStatus {
	if h.processor == nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "RAG processor not initialized",
		}
	}

	// Try to list documents as a health check
	docs, err := h.processor.ListDocuments(ctx)
	if err != nil {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "Failed to access document store",
		}
	}

	return ComponentStatus{
		Status: "healthy",
		Metadata: map[string]interface{}{
			"document_count":  len(docs),
			"chunker_enabled": h.config.Chunker.ChunkSize > 0,
			"chunk_size":      h.config.Chunker.ChunkSize,
		},
	}
}

// checkMCP checks the health of MCP servers
func (h *HealthHandler) checkMCP(ctx context.Context) ComponentStatus {
	// MCP is optional
	if !h.config.Tools.Enabled {
		return ComponentStatus{
			Status:  "disabled",
			Message: "MCP tools not enabled in configuration",
		}
	}

	if h.mcpService == nil {
		return ComponentStatus{
			Status:  "degraded",
			Message: "MCP service not initialized but tools are enabled",
		}
	}

	// Get MCP server status (simplified since GetServers is not available)
	healthyCount := 0
	totalCount := 1 // Assume at least one server if MCP is initialized

	// Check if MCP service is healthy
	if h.mcpService != nil {
		// Use a simple health check instead of enumerating servers
		if tools := h.mcpService.GetAvailableTools(); len(tools) > 0 {
			healthyCount = 1
		}
	}

	if totalCount == 0 {
		return ComponentStatus{
			Status:  "degraded",
			Message: "No MCP servers configured",
		}
	}

	status := "healthy"
	message := ""
	if healthyCount < totalCount {
		if healthyCount == 0 {
			status = "unhealthy"
			message = "All MCP servers are unhealthy"
		} else {
			status = "degraded"
			message = "Some MCP servers are unhealthy"
		}
	}

	return ComponentStatus{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"total_servers":   totalCount,
			"healthy_servers": healthyCount,
		},
	}
}

// checkLLM checks the health of LLM providers
func (h *HealthHandler) checkLLM(ctx context.Context) ComponentStatus {
	healthyProviders := []string{}

	// Check which providers are configured
	// We'll do a simple configuration check rather than instantiating providers

	// Check Ollama
	if h.config.Providers.ProviderConfigs.Ollama != nil && h.config.Providers.ProviderConfigs.Ollama.LLMModel != "" {
		// For Ollama, we could check if the service is reachable, but for now just check config
		healthyProviders = append(healthyProviders, "ollama")
	}

	// Check OpenAI
	if h.config.Providers.ProviderConfigs.OpenAI != nil && h.config.Providers.ProviderConfigs.OpenAI.APIKey != "" {
		healthyProviders = append(healthyProviders, "openai")
	}

	// Check LMStudio
	if h.config.Providers.ProviderConfigs.LMStudio != nil && h.config.Providers.ProviderConfigs.LMStudio.BaseURL != "" {
		healthyProviders = append(healthyProviders, "lmstudio")
	}

	if len(healthyProviders) == 0 {
		return ComponentStatus{
			Status:  "unhealthy",
			Message: "No LLM providers configured or healthy",
		}
	}

	return ComponentStatus{
		Status: "healthy",
		Metadata: map[string]interface{}{
			"providers": healthyProviders,
		},
	}
}

// checkScheduler checks the health of the scheduler
func (h *HealthHandler) checkScheduler(ctx context.Context) ComponentStatus {
	// Scheduler will be implemented in Phase 2
	return ComponentStatus{
		Status:  "disabled",
		Message: "Scheduler not yet implemented",
	}
}

// checkAgents checks the health of the agent system
func (h *HealthHandler) checkAgents(ctx context.Context) ComponentStatus {
	if !h.config.Agents.Enabled {
		return ComponentStatus{
			Status:  "disabled",
			Message: "Agents not enabled in configuration",
		}
	}

	return ComponentStatus{
		Status: "healthy",
		Metadata: map[string]interface{}{
			"enabled": true,
		},
	}
}

// determineOverallStatus determines the overall health status
func (h *HealthHandler) determineOverallStatus(components map[string]ComponentStatus) string {
	// RAG is core - if unhealthy, whole system is unhealthy
	if rag, exists := components["rag"]; exists && rag.Status == "unhealthy" {
		return "unhealthy"
	}

	// LLM is critical - if unhealthy, system is unhealthy
	if llm, exists := components["llm"]; exists && llm.Status == "unhealthy" {
		return "unhealthy"
	}

	// Check for degraded components
	for _, status := range components {
		if status.Status == "disabled" {
			continue
		}
		if status.Status == "degraded" {
			return "degraded"
		}
	}

	return "healthy"
}
