package server

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates metrics for MCP servers
type MetricsCollector struct {
	mu                sync.RWMutex
	serverMetrics     map[string]*ServerMetrics
	globalMetrics     *GlobalMetrics
	metricsWindow     time.Duration
	aggregateInterval time.Duration
	stopCh            chan struct{}
	wg                sync.WaitGroup
}

// ServerMetrics contains metrics for a specific server
type ServerMetrics struct {
	ServerName          string
	TotalCalls          int64
	SuccessfulCalls     int64
	FailedCalls         int64
	TotalResponseTime   time.Duration
	AverageResponseTime time.Duration
	LastCallTime        time.Time
	ErrorRate           float64
	Uptime              time.Duration
	RestartCount        int
	HealthChecksPassed  int64
	HealthChecksFailed  int64
	RecoveryAttempts    int
	RecoverySuccesses   int
	ToolMetrics         map[string]*ToolMetrics
	mu                  sync.RWMutex
}

// ToolMetrics contains metrics for a specific tool
type ToolMetrics struct {
	ToolName            string
	TotalCalls          int64
	SuccessfulCalls     int64
	FailedCalls         int64
	TotalResponseTime   time.Duration
	AverageResponseTime time.Duration
	LastCallTime        time.Time
	ErrorRate           float64
	CacheHits           int64
	CacheMisses         int64
}

// GlobalMetrics contains system-wide metrics
type GlobalMetrics struct {
	TotalServers        int
	HealthyServers      int
	UnhealthyServers    int
	TotalTools          int
	AvailableTools      int
	TotalToolCalls      int64
	TotalCacheHits      int64
	TotalCacheMisses    int64
	SystemUptime        time.Duration
	LastUpdateTime      time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		serverMetrics:     make(map[string]*ServerMetrics),
		globalMetrics:     &GlobalMetrics{},
		metricsWindow:     24 * time.Hour,
		aggregateInterval: 1 * time.Minute,
		stopCh:            make(chan struct{}),
	}
}

// Start begins metrics collection
func (m *MetricsCollector) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.aggregateLoop(ctx)
}

// Stop stops metrics collection
func (m *MetricsCollector) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// aggregateLoop periodically aggregates metrics
func (m *MetricsCollector) aggregateLoop(ctx context.Context) {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.aggregateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.aggregateMetrics()
		}
	}
}

// aggregateMetrics aggregates current metrics
func (m *MetricsCollector) aggregateMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Update global metrics
	m.globalMetrics.TotalServers = len(m.serverMetrics)
	m.globalMetrics.LastUpdateTime = time.Now()
	
	healthyCount := 0
	totalTools := 0
	availableTools := 0
	
	for _, serverMetrics := range m.serverMetrics {
		serverMetrics.mu.RLock()
		
		// Calculate error rate
		if serverMetrics.TotalCalls > 0 {
			serverMetrics.ErrorRate = float64(serverMetrics.FailedCalls) / float64(serverMetrics.TotalCalls)
		}
		
		// Calculate average response time
		if serverMetrics.SuccessfulCalls > 0 {
			serverMetrics.AverageResponseTime = serverMetrics.TotalResponseTime / time.Duration(serverMetrics.SuccessfulCalls)
		}
		
		// Count healthy servers (simplified - based on error rate)
		if serverMetrics.ErrorRate < 0.1 { // Less than 10% error rate
			healthyCount++
		}
		
		// Count tools
		totalTools += len(serverMetrics.ToolMetrics)
		for _, toolMetrics := range serverMetrics.ToolMetrics {
			if toolMetrics.ErrorRate < 0.5 { // Tool is considered available if error rate < 50%
				availableTools++
			}
			
			// Update tool metrics
			if toolMetrics.TotalCalls > 0 {
				toolMetrics.ErrorRate = float64(toolMetrics.FailedCalls) / float64(toolMetrics.TotalCalls)
			}
			if toolMetrics.SuccessfulCalls > 0 {
				toolMetrics.AverageResponseTime = toolMetrics.TotalResponseTime / time.Duration(toolMetrics.SuccessfulCalls)
			}
		}
		
		serverMetrics.mu.RUnlock()
	}
	
	m.globalMetrics.HealthyServers = healthyCount
	m.globalMetrics.UnhealthyServers = m.globalMetrics.TotalServers - healthyCount
	m.globalMetrics.TotalTools = totalTools
	m.globalMetrics.AvailableTools = availableTools
}

// RecordServerRegistration records a server registration
func (m *MetricsCollector) RecordServerRegistration(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.serverMetrics[serverName]; !exists {
		m.serverMetrics[serverName] = &ServerMetrics{
			ServerName:  serverName,
			ToolMetrics: make(map[string]*ToolMetrics),
		}
	}
}

// RecordServerUnregistration records a server unregistration
func (m *MetricsCollector) RecordServerUnregistration(serverName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.serverMetrics, serverName)
}

// RecordServerRestart records a server restart
func (m *MetricsCollector) RecordServerRestart(serverName string) {
	m.mu.RLock()
	metrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if exists {
		metrics.mu.Lock()
		metrics.RestartCount++
		metrics.mu.Unlock()
	}
}

// RecordHealthCheckSuccess records a successful health check
func (m *MetricsCollector) RecordHealthCheckSuccess(serverName string, responseTime time.Duration) {
	m.mu.RLock()
	metrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if exists {
		metrics.mu.Lock()
		metrics.HealthChecksPassed++
		metrics.mu.Unlock()
	}
}

// RecordHealthCheckFailure records a failed health check
func (m *MetricsCollector) RecordHealthCheckFailure(serverName string, err error) {
	m.mu.RLock()
	metrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if exists {
		metrics.mu.Lock()
		metrics.HealthChecksFailed++
		metrics.mu.Unlock()
	}
}

// RecordRecoverySuccess records a successful recovery
func (m *MetricsCollector) RecordRecoverySuccess(serverName string) {
	m.mu.RLock()
	metrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if exists {
		metrics.mu.Lock()
		metrics.RecoveryAttempts++
		metrics.RecoverySuccesses++
		metrics.mu.Unlock()
	}
}

// RecordRecoveryFailure records a failed recovery
func (m *MetricsCollector) RecordRecoveryFailure(serverName string, err error) {
	m.mu.RLock()
	metrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if exists {
		metrics.mu.Lock()
		metrics.RecoveryAttempts++
		metrics.mu.Unlock()
	}
}

// RecordToolCall records a tool call
func (m *MetricsCollector) RecordToolCall(serverName, toolName string, success bool, duration time.Duration) {
	m.mu.RLock()
	serverMetrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return
	}
	
	serverMetrics.mu.Lock()
	defer serverMetrics.mu.Unlock()
	
	// Update server metrics
	serverMetrics.TotalCalls++
	if success {
		serverMetrics.SuccessfulCalls++
		serverMetrics.TotalResponseTime += duration
	} else {
		serverMetrics.FailedCalls++
	}
	serverMetrics.LastCallTime = time.Now()
	
	// Update tool metrics
	toolMetrics, exists := serverMetrics.ToolMetrics[toolName]
	if !exists {
		toolMetrics = &ToolMetrics{
			ToolName: toolName,
		}
		serverMetrics.ToolMetrics[toolName] = toolMetrics
	}
	
	toolMetrics.TotalCalls++
	if success {
		toolMetrics.SuccessfulCalls++
		toolMetrics.TotalResponseTime += duration
	} else {
		toolMetrics.FailedCalls++
	}
	toolMetrics.LastCallTime = time.Now()
	
	// Update global metrics
	m.mu.Lock()
	m.globalMetrics.TotalToolCalls++
	m.mu.Unlock()
}

// RecordCacheHit records a cache hit
func (m *MetricsCollector) RecordCacheHit(serverName, toolName string) {
	m.mu.RLock()
	serverMetrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return
	}
	
	serverMetrics.mu.Lock()
	toolMetrics, exists := serverMetrics.ToolMetrics[toolName]
	if exists {
		toolMetrics.CacheHits++
	}
	serverMetrics.mu.Unlock()
	
	m.mu.Lock()
	m.globalMetrics.TotalCacheHits++
	m.mu.Unlock()
}

// RecordCacheMiss records a cache miss
func (m *MetricsCollector) RecordCacheMiss(serverName, toolName string) {
	m.mu.RLock()
	serverMetrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return
	}
	
	serverMetrics.mu.Lock()
	toolMetrics, exists := serverMetrics.ToolMetrics[toolName]
	if exists {
		toolMetrics.CacheMisses++
	}
	serverMetrics.mu.Unlock()
	
	m.mu.Lock()
	m.globalMetrics.TotalCacheMisses++
	m.mu.Unlock()
}

// RecordDiscovery records a tool discovery event
func (m *MetricsCollector) RecordDiscovery(toolCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.globalMetrics.TotalTools = toolCount
	m.globalMetrics.AvailableTools = toolCount // Will be refined by aggregation
}

// GetMetrics returns current metrics
func (m *MetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Build metrics map
	metrics := make(map[string]interface{})
	
	// Add global metrics
	metrics["global"] = m.globalMetrics
	
	// Add server metrics
	serverMetricsMap := make(map[string]interface{})
	for name, serverMetrics := range m.serverMetrics {
		serverMetrics.mu.RLock()
		serverMetricsMap[name] = map[string]interface{}{
			"total_calls":           serverMetrics.TotalCalls,
			"successful_calls":      serverMetrics.SuccessfulCalls,
			"failed_calls":          serverMetrics.FailedCalls,
			"average_response_time": serverMetrics.AverageResponseTime,
			"error_rate":            serverMetrics.ErrorRate,
			"restart_count":         serverMetrics.RestartCount,
			"health_checks_passed":  serverMetrics.HealthChecksPassed,
			"health_checks_failed":  serverMetrics.HealthChecksFailed,
			"tool_count":            len(serverMetrics.ToolMetrics),
		}
		serverMetrics.mu.RUnlock()
	}
	metrics["servers"] = serverMetricsMap
	
	return metrics
}

// GetServerMetrics returns metrics for a specific server
func (m *MetricsCollector) GetServerMetrics(serverName string) (*ServerMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics, exists := m.serverMetrics[serverName]
	if !exists {
		return nil, fmt.Errorf("metrics not found for server: %s", serverName)
	}
	
	return metrics, nil
}

// GetToolMetrics returns metrics for a specific tool
func (m *MetricsCollector) GetToolMetrics(serverName, toolName string) (*ToolMetrics, error) {
	m.mu.RLock()
	serverMetrics, exists := m.serverMetrics[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}
	
	serverMetrics.mu.RLock()
	defer serverMetrics.mu.RUnlock()
	
	toolMetrics, exists := serverMetrics.ToolMetrics[toolName]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}
	
	return toolMetrics, nil
}