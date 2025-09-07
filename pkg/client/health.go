// Package client - health.go  
// This file provides comprehensive health monitoring across all four RAGO pillars,
// enabling applications to monitor system health and make intelligent decisions.

package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// HealthMonitor provides comprehensive health monitoring for all pillars
type HealthMonitor struct {
	client      *Client
	lastReport  core.HealthReport
	mu          sync.RWMutex
	updateChan  chan struct{}
	stopChan    chan struct{}
	monitoring  bool
	checkInterval time.Duration
}

// NewHealthMonitor creates a new health monitor for the given client
func NewHealthMonitor(client *Client) *HealthMonitor {
	monitor := &HealthMonitor{
		client:        client,
		updateChan:    make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		monitoring:    true,
		checkInterval: 30 * time.Second, // Default check interval
		lastReport: core.HealthReport{
			Overall:   core.HealthStatusUnknown,
			Pillars:   make(map[string]core.HealthStatus),
			Providers: make(map[string]core.HealthStatus),
			Servers:   make(map[string]core.HealthStatus),
			LastCheck: time.Now(),
			Details:   make(map[string]interface{}),
		},
	}

	// Set initial pillar status to unknown for all enabled pillars
	if client.llmService != nil {
		monitor.lastReport.Pillars["LLM"] = core.HealthStatusUnknown
	}
	if client.ragService != nil {
		monitor.lastReport.Pillars["RAG"] = core.HealthStatusUnknown
	}
	if client.mcpService != nil {
		monitor.lastReport.Pillars["MCP"] = core.HealthStatusUnknown
	}
	if client.agentService != nil {
		monitor.lastReport.Pillars["Agent"] = core.HealthStatusUnknown
	}
	
	// Start background monitoring
	go monitor.startMonitoring()
	
	return monitor
}

// startMonitoring runs the background health checking goroutine
func (h *HealthMonitor) startMonitoring() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()
	
	h.monitoring = true
	
	// Initial check already done in NewHealthMonitor
	
	for {
		select {
		case <-ticker.C:
			h.performHealthCheck()
		case <-h.updateChan:
			h.performHealthCheck()
		case <-h.stopChan:
			h.monitoring = false
			return
		}
	}
}

// performHealthCheck conducts a comprehensive health check across all pillars
// NOTE: Optimized to only perform actual health checks on LLM providers for faster status command
func (h *HealthMonitor) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Handle nil client gracefully
	if h.client == nil {
		h.lastReport = core.HealthReport{
			Overall:   core.HealthStatusUnknown,
			Pillars:   make(map[string]core.HealthStatus),
			Providers: make(map[string]core.HealthStatus),
			Servers:   make(map[string]core.HealthStatus),
			LastCheck: time.Now(),
			Details:   map[string]interface{}{"error": "client is nil"},
		}
		return
	}
	
	report := core.HealthReport{
		Pillars:   make(map[string]core.HealthStatus),
		Providers: make(map[string]core.HealthStatus),
		Servers:   make(map[string]core.HealthStatus),
		LastCheck: time.Now(),
		Details:   make(map[string]interface{}),
	}
	
	var healthyPillars, totalPillars int
	
	// Check LLM pillar health - ALWAYS PERFORM FULL CHECK
	if h.client.llmService != nil {
		llmHealth := h.checkLLMHealth(ctx)
		report.Pillars["LLM"] = llmHealth  // Use uppercase for consistency
		totalPillars++
		if llmHealth == core.HealthStatusHealthy {
			healthyPillars++
		}
		
		// Get provider health from LLM service
		if providerHealths := h.client.llmService.GetProviderHealth(); providerHealths != nil {
			for name, status := range providerHealths {
				report.Providers["llm_"+name] = status
			}
		}
	}
	
	// RAG pillar - SKIP ACTUAL HEALTH CHECK, just show as configured/unconfigured
	if h.client.ragService != nil {
		// Don't perform actual health check, just mark as unknown or use last known state
		if existingStatus, exists := h.lastReport.Pillars["RAG"]; exists && existingStatus != core.HealthStatusUnknown {
			report.Pillars["RAG"] = existingStatus  
		} else {
			report.Pillars["RAG"] = core.HealthStatusUnknown  // Show as unknown, not checked
		}
		totalPillars++
		// Count as healthy for overall calculation if it exists
		if report.Pillars["RAG"] == core.HealthStatusHealthy {
			healthyPillars++
		}
	}
	
	// MCP pillar - SKIP ACTUAL HEALTH CHECK
	if h.client.mcpService != nil {
		// Don't perform actual health check, just mark as unknown or use last known state
		if existingStatus, exists := h.lastReport.Pillars["MCP"]; exists && existingStatus != core.HealthStatusUnknown {
			report.Pillars["MCP"] = existingStatus  
		} else {
			report.Pillars["MCP"] = core.HealthStatusUnknown  // Show as unknown, not checked
		}
		totalPillars++
		// Count as healthy for overall calculation if it exists
		if report.Pillars["MCP"] == core.HealthStatusHealthy {
			healthyPillars++
		}
	}
	
	// Agents pillar - SKIP ACTUAL HEALTH CHECK
	if h.client.agentService != nil {
		// Don't perform actual health check, just mark as unknown or use last known state
		if existingStatus, exists := h.lastReport.Pillars["Agent"]; exists && existingStatus != core.HealthStatusUnknown {
			report.Pillars["Agent"] = existingStatus  
		} else {
			report.Pillars["Agent"] = core.HealthStatusUnknown  // Show as unknown, not checked
		}
		totalPillars++
		// Count as healthy for overall calculation if it exists
		if report.Pillars["Agent"] == core.HealthStatusHealthy {
			healthyPillars++
		}
	}
	
	// Determine overall health based mainly on LLM status since that's what we actually check
	if totalPillars == 0 {
		report.Overall = core.HealthStatusUnknown
	} else if h.client.llmService != nil && report.Pillars["LLM"] == core.HealthStatusHealthy {
		// If LLM is healthy, consider overall as healthy (since that's what we care about)
		report.Overall = core.HealthStatusHealthy
	} else if h.client.llmService != nil && report.Pillars["LLM"] == core.HealthStatusDegraded {
		report.Overall = core.HealthStatusDegraded
	} else if h.client.llmService != nil && report.Pillars["LLM"] == core.HealthStatusUnhealthy {
		report.Overall = core.HealthStatusUnhealthy
	} else {
		// No LLM service or unknown status
		report.Overall = core.HealthStatusUnknown
	}
	
	// Add overall statistics
	report.Details["overall_stats"] = map[string]interface{}{
		"healthy_pillars": healthyPillars,
		"total_pillars":   totalPillars,
		"health_ratio":    float64(healthyPillars) / float64(totalPillars),
		"client_uptime":   time.Since(time.Now().Add(-h.checkInterval)), // Approximation
		"check_mode":      "llm_only",  // Indicate we're only checking LLM
	}
	
	h.lastReport = report
}

// checkLLMHealth performs health checks specific to the LLM pillar
func (h *HealthMonitor) checkLLMHealth(ctx context.Context) core.HealthStatus {
	if h.client.llmService == nil {
		return core.HealthStatusUnknown
	}
	
	// Check if we can list providers
	providers := h.client.llmService.ListProviders()
	if len(providers) == 0 {
		return core.HealthStatusUnhealthy
	}
	
	// Check provider health - this might trigger health checks
	providerHealths := h.client.llmService.GetProviderHealth()
	healthyProviders := 0
	unknownProviders := 0
	totalProviderHealths := len(providerHealths)
	
	for _, health := range providerHealths {
		if health == core.HealthStatusHealthy {
			healthyProviders++
		} else if health == core.HealthStatusUnknown {
			unknownProviders++
		}
	}
	
	// If we have no health data yet, providers haven't been checked
	if totalProviderHealths == 0 && len(providers) > 0 {
		return core.HealthStatusUnknown
	}
	
	// If all providers are unknown, we're still checking
	if unknownProviders == totalProviderHealths && totalProviderHealths > 0 {
		return core.HealthStatusUnknown
	}
	
	// Determine health based on provider availability
	if healthyProviders == 0 {
		return core.HealthStatusUnhealthy
	} else if healthyProviders < len(providers) {
		return core.HealthStatusDegraded
	} else {
		return core.HealthStatusHealthy
	}
}

// checkRAGHealth performs health checks specific to the RAG pillar
func (h *HealthMonitor) checkRAGHealth(ctx context.Context) core.HealthStatus {
	if h.client.ragService == nil {
		return core.HealthStatusUnknown
	}
	
	// Try to get stats to ensure storage is accessible
	stats, err := h.client.ragService.GetStats(ctx)
	if err != nil {
		return core.HealthStatusUnhealthy
	}
	
	// Check if we can perform a basic search (with empty query)
	// This tests the search infrastructure
	searchReq := core.SearchRequest{
		Query: "test",
		Limit: 1,
	}
	
	_, err = h.client.ragService.Search(ctx, searchReq)
	if err != nil {
		return core.HealthStatusDegraded
	}
	
	// Check storage health indicators
	if stats.TotalDocuments >= 0 && stats.TotalChunks >= 0 {
		return core.HealthStatusHealthy
	}
	
	return core.HealthStatusDegraded
}

// checkMCPHealth performs health checks specific to the MCP pillar
func (h *HealthMonitor) checkMCPHealth(ctx context.Context) core.HealthStatus {
	if h.client.mcpService == nil {
		return core.HealthStatusUnknown
	}
	
	// Check if tools are loaded
	tools := h.client.mcpService.GetTools()
	if len(tools) > 0 {
		return core.HealthStatusHealthy
	}
	
	// No tools loaded yet
	return core.HealthStatusDegraded
}

// checkAgentsHealth performs health checks specific to the Agents pillar
func (h *HealthMonitor) checkAgentsHealth(ctx context.Context) core.HealthStatus {
	if h.client.agentService == nil {
		return core.HealthStatusUnknown
	}
	
	// Check if we can list workflows and agents
	workflows := h.client.agentService.ListWorkflows()
	agents := h.client.agentService.ListAgents()
	
	// For now, if we can access these lists, the service is healthy
	// Future enhancements could include checking workflow execution capabilities
	if workflows != nil && agents != nil {
		return core.HealthStatusHealthy
	}
	
	return core.HealthStatusDegraded
}

// GetHealthReport returns the current health report
func (h *HealthMonitor) GetHealthReport() core.HealthReport {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastReport
}

// TriggerHealthCheck triggers an immediate health check (optimized for LLM only)
func (h *HealthMonitor) TriggerHealthCheck() {
	// First trigger LLM service's own health check if available
	if h.client.llmService != nil {
		if triggerable, ok := h.client.llmService.(interface{ TriggerHealthCheck() }); ok {
			triggerable.TriggerHealthCheck()
		}
	}
	
	// Then perform the overall health check (which now only checks LLM)
	h.performHealthCheck()
}

// TriggerLLMOnlyHealthCheck triggers a health check that ONLY checks LLM providers
// This is the optimized version for the status command
func (h *HealthMonitor) TriggerLLMOnlyHealthCheck() {
	// This is now the default behavior of TriggerHealthCheck
	h.TriggerHealthCheck()
}

// TriggerFullHealthCheck triggers a comprehensive health check on ALL pillars
// This is the slower but more thorough version
func (h *HealthMonitor) TriggerFullHealthCheck() {
	// First trigger LLM service's own health check if available
	if h.client.llmService != nil {
		if triggerable, ok := h.client.llmService.(interface{ TriggerHealthCheck() }); ok {
			triggerable.TriggerHealthCheck()
		}
	}
	
	// Then perform a full health check on all pillars
	h.performFullHealthCheck()
}

// performFullHealthCheck conducts a comprehensive health check across ALL pillars
// This is the original, thorough implementation that checks everything
func (h *HealthMonitor) performFullHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Handle nil client gracefully
	if h.client == nil {
		h.lastReport = core.HealthReport{
			Overall:   core.HealthStatusUnknown,
			Pillars:   make(map[string]core.HealthStatus),
			Providers: make(map[string]core.HealthStatus),
			Servers:   make(map[string]core.HealthStatus),
			LastCheck: time.Now(),
			Details:   map[string]interface{}{"error": "client is nil"},
		}
		return
	}
	
	report := core.HealthReport{
		Pillars:   make(map[string]core.HealthStatus),
		Providers: make(map[string]core.HealthStatus),
		Servers:   make(map[string]core.HealthStatus),
		LastCheck: time.Now(),
		Details:   make(map[string]interface{}),
	}
	
	var healthyPillars, totalPillars int
	
	// Check LLM pillar health
	if h.client.llmService != nil {
		llmHealth := h.checkLLMHealth(ctx)
		report.Pillars["LLM"] = llmHealth
		totalPillars++
		if llmHealth == core.HealthStatusHealthy {
			healthyPillars++
		}
		
		// Get provider health from LLM service
		if providerHealths := h.client.llmService.GetProviderHealth(); providerHealths != nil {
			for name, status := range providerHealths {
				report.Providers["llm_"+name] = status
			}
		}
	}
	
	// Check RAG pillar health - FULL CHECK
	if h.client.ragService != nil {
		ragHealth := h.checkRAGHealth(ctx)
		report.Pillars["RAG"] = ragHealth
		totalPillars++
		if ragHealth == core.HealthStatusHealthy {
			healthyPillars++
		}
		
		// Get RAG statistics for health details
		if stats, err := h.client.ragService.GetStats(ctx); err == nil {
			report.Details["rag_stats"] = map[string]interface{}{
				"total_documents": stats.TotalDocuments,
				"total_chunks":    stats.TotalChunks,
				"storage_size":    stats.StorageSize,
				"last_optimized":  stats.LastOptimized,
			}
		}
	}
	
	// Check MCP pillar health - FULL CHECK
	if h.client.mcpService != nil {
		mcpHealth := h.checkMCPHealth(ctx)
		report.Pillars["MCP"] = mcpHealth
		totalPillars++
		if mcpHealth == core.HealthStatusHealthy {
			healthyPillars++
		}
		
		// Get server health from MCP service
		servers := []core.ServerInfo{}
		for _, server := range servers {
			serverHealth := core.HealthStatusHealthy
			report.Servers[server.Name] = serverHealth
		}
		
		// Add MCP details
		tools := h.client.mcpService.GetTools()
		report.Details["mcp_stats"] = map[string]interface{}{
			"total_servers": len(servers),
			"total_tools":   len(tools),
		}
	}
	
	// Check Agents pillar health - FULL CHECK
	if h.client.agentService != nil {
		agentsHealth := h.checkAgentsHealth(ctx)
		report.Pillars["Agent"] = agentsHealth
		totalPillars++
		if agentsHealth == core.HealthStatusHealthy {
			healthyPillars++
		}
		
		// Add agent details
		workflows := h.client.agentService.ListWorkflows()
		agents := h.client.agentService.ListAgents()
		scheduledTasks := h.client.agentService.GetScheduledTasks()
		
		report.Details["agents_stats"] = map[string]interface{}{
			"total_workflows":    len(workflows),
			"total_agents":       len(agents),
			"scheduled_tasks":    len(scheduledTasks),
		}
	}
	
	// Determine overall health
	if totalPillars == 0 {
		report.Overall = core.HealthStatusUnknown
	} else if healthyPillars == totalPillars {
		report.Overall = core.HealthStatusHealthy
	} else if healthyPillars > 0 {
		report.Overall = core.HealthStatusDegraded
	} else {
		report.Overall = core.HealthStatusUnhealthy
	}
	
	// Add overall statistics
	report.Details["overall_stats"] = map[string]interface{}{
		"healthy_pillars": healthyPillars,
		"total_pillars":   totalPillars,
		"health_ratio":    float64(healthyPillars) / float64(totalPillars),
		"client_uptime":   time.Since(time.Now().Add(-h.checkInterval)),
		"check_mode":      "full",  // Indicate this was a full check
	}
	
	h.lastReport = report
}

// GetPillarHealth returns the health status of a specific pillar
func (h *HealthMonitor) GetPillarHealth(pillarName string) core.HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if status, exists := h.lastReport.Pillars[pillarName]; exists {
		return status
	}
	
	return core.HealthStatusUnknown
}

// IsHealthy returns true if the overall system is healthy
func (h *HealthMonitor) IsHealthy() bool {
	return h.GetHealthReport().Overall == core.HealthStatusHealthy
}

// IsDegraded returns true if the system is in degraded state
func (h *HealthMonitor) IsDegraded() bool {
	return h.GetHealthReport().Overall == core.HealthStatusDegraded
}

// RefreshHealth triggers an immediate health check
func (h *HealthMonitor) RefreshHealth() {
	select {
	case h.updateChan <- struct{}{}:
	default:
		// Channel is full, health check is already queued
	}
}

// SetCheckInterval updates the health check interval
func (h *HealthMonitor) SetCheckInterval(interval time.Duration) {
	h.mu.Lock()
	h.checkInterval = interval
	h.mu.Unlock()
}

// GetCheckInterval returns the current health check interval
func (h *HealthMonitor) GetCheckInterval() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.checkInterval
}

// Stop stops the health monitoring goroutine
func (h *HealthMonitor) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.monitoring {
		h.monitoring = false
		close(h.stopChan)
	}
}

// GetHealthSummary returns a human-readable health summary
func (h *HealthMonitor) GetHealthSummary() string {
	report := h.GetHealthReport()
	
	summary := fmt.Sprintf("Overall Health: %s\n", report.Overall)
	summary += fmt.Sprintf("Last Check: %s\n", report.LastCheck.Format(time.RFC3339))
	
	summary += "\nPillar Status:\n"
	for pillar, status := range report.Pillars {
		summary += fmt.Sprintf("  %s: %s\n", pillar, status)
	}
	
	if len(report.Providers) > 0 {
		summary += "\nProvider Status:\n"
		for provider, status := range report.Providers {
			summary += fmt.Sprintf("  %s: %s\n", provider, status)
		}
	}
	
	if len(report.Servers) > 0 {
		summary += "\nServer Status:\n"  
		for server, status := range report.Servers {
			summary += fmt.Sprintf("  %s: %s\n", server, status)
		}
	}
	
	return summary
}

// WaitForHealthy blocks until the system becomes healthy or the context is cancelled
func (h *HealthMonitor) WaitForHealthy(ctx context.Context) error {
	for {
		if h.IsHealthy() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			// Continue checking
		}
	}
}

// HealthCheckFunc defines a custom health check function
type HealthCheckFunc func(ctx context.Context) (core.HealthStatus, error)

// customHealthChecks stores additional health checks
type customHealthChecks struct {
	mu     sync.RWMutex
	checks map[string]HealthCheckFunc
}

var customChecks = &customHealthChecks{
	checks: make(map[string]HealthCheckFunc),
}

// RegisterHealthCheck registers a custom health check function
func RegisterHealthCheck(name string, checkFunc HealthCheckFunc) {
	customChecks.mu.Lock()
	defer customChecks.mu.Unlock()
	customChecks.checks[name] = checkFunc
}

// UnregisterHealthCheck removes a custom health check
func UnregisterHealthCheck(name string) {
	customChecks.mu.Lock()
	defer customChecks.mu.Unlock()
	delete(customChecks.checks, name)
}

// runCustomHealthChecks executes all registered custom health checks
func (h *HealthMonitor) runCustomHealthChecks(ctx context.Context) map[string]core.HealthStatus {
	customChecks.mu.RLock()
	defer customChecks.mu.RUnlock()
	
	results := make(map[string]core.HealthStatus)
	
	for name, checkFunc := range customChecks.checks {
		status, err := checkFunc(ctx)
		if err != nil {
			status = core.HealthStatusUnhealthy
		}
		results[name] = status
	}
	
	return results
}