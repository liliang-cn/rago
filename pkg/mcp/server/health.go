package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// HealthMonitor monitors the health of MCP servers and handles recovery
type HealthMonitor struct {
	manager         *Manager
	checkInterval   time.Duration
	unhealthyThreshold int
	mu              sync.RWMutex
	healthChecks    map[string]*HealthCheck
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// HealthCheck tracks health check state for a server
type HealthCheck struct {
	ServerName        string
	LastCheck         time.Time
	ConsecutiveFails  int
	LastError         error
	HealthMetrics     *HealthMetrics
	mu                sync.RWMutex
}

// HealthMetrics contains detailed health metrics
type HealthMetrics struct {
	ResponseTime      time.Duration
	MemoryUsage       int64
	CPUUsage          float64
	ActiveConnections int
	ToolCallsPerMin   float64
	ErrorRate         float64
	Uptime            time.Duration
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(manager *Manager) *HealthMonitor {
	interval := 30 * time.Second
	if manager.config.HealthCheckInterval > 0 {
		interval = manager.config.HealthCheckInterval
	}
	
	return &HealthMonitor{
		manager:            manager,
		checkInterval:      interval,
		unhealthyThreshold: 3,
		healthChecks:       make(map[string]*HealthCheck),
		stopCh:             make(chan struct{}),
	}
}

// Start begins health monitoring
func (h *HealthMonitor) Start(ctx context.Context) error {
	h.wg.Add(1)
	go h.monitorLoop(ctx)
	return nil
}

// Stop stops health monitoring
func (h *HealthMonitor) Stop() {
	close(h.stopCh)
	h.wg.Wait()
}

// monitorLoop runs the health check loop
func (h *HealthMonitor) monitorLoop(ctx context.Context) {
	defer h.wg.Done()
	
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()
	
	// Initial health check
	h.checkAllServers(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkAllServers(ctx)
		}
	}
}

// checkAllServers performs health checks on all registered servers
func (h *HealthMonitor) checkAllServers(ctx context.Context) {
	servers := h.manager.ListServers()
	
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(s *ServerInstance) {
			defer wg.Done()
			h.checkServer(ctx, s)
		}(server)
	}
	wg.Wait()
}

// checkServer performs a health check on a single server
func (h *HealthMonitor) checkServer(ctx context.Context, server *ServerInstance) {
	h.mu.Lock()
	check, exists := h.healthChecks[server.Config.Name]
	if !exists {
		check = &HealthCheck{
			ServerName: server.Config.Name,
		}
		h.healthChecks[server.Config.Name] = check
	}
	h.mu.Unlock()
	
	check.mu.Lock()
	defer check.mu.Unlock()
	
	// Perform health check
	start := time.Now()
	err := h.performHealthCheck(ctx, server)
	responseTime := time.Since(start)
	
	check.LastCheck = time.Now()
	
	if err != nil {
		// Health check failed
		check.ConsecutiveFails++
		check.LastError = err
		
		// Update server status
		server.mu.Lock()
		if check.ConsecutiveFails >= h.unhealthyThreshold {
			server.Status = StatusUnhealthy
			
			// Trigger recovery if configured
			if server.Config.RestartOnFailure && server.RestartCount < server.Config.MaxRestarts {
				server.mu.Unlock()
				go h.recoverServer(ctx, server)
			} else {
				server.mu.Unlock()
			}
		} else {
			server.mu.Unlock()
		}
		
		// Record metrics
		h.manager.metricsCollector.RecordHealthCheckFailure(server.Config.Name, err)
	} else {
		// Health check succeeded
		check.ConsecutiveFails = 0
		check.LastError = nil
		
		// Update metrics
		if check.HealthMetrics == nil {
			check.HealthMetrics = &HealthMetrics{}
		}
		check.HealthMetrics.ResponseTime = responseTime
		check.HealthMetrics.Uptime = time.Since(server.StartedAt)
		
		// Update server status
		server.mu.Lock()
		if server.Status != StatusHealthy {
			server.Status = StatusHealthy
		}
		server.LastHealthy = time.Now()
		server.mu.Unlock()
		
		// Record metrics
		h.manager.metricsCollector.RecordHealthCheckSuccess(server.Config.Name, responseTime)
	}
}

// performHealthCheck performs the actual health check
func (h *HealthMonitor) performHealthCheck(ctx context.Context, server *ServerInstance) error {
	// Check if client is connected
	if server.Client == nil || !server.Client.IsConnected() {
		return fmt.Errorf("client not connected")
	}
	
	// Create a context with timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Try to list tools as a health check
	// This verifies the server is responsive
	select {
	case <-checkCtx.Done():
		return fmt.Errorf("health check timeout")
	default:
		tools := server.Client.GetTools()
		if tools == nil {
			return fmt.Errorf("unable to retrieve tools")
		}
	}
	
	return nil
}

// recoverServer attempts to recover an unhealthy server
func (h *HealthMonitor) recoverServer(ctx context.Context, server *ServerInstance) {
	// Add delay before recovery attempt
	time.Sleep(server.Config.RestartDelay)
	
	// Attempt restart
	if err := h.manager.RestartServer(ctx, server.Config.Name); err != nil {
		// Recovery failed
		server.mu.Lock()
		server.Status = StatusFailed
		server.mu.Unlock()
		
		h.manager.metricsCollector.RecordRecoveryFailure(server.Config.Name, err)
	} else {
		// Recovery succeeded
		h.manager.metricsCollector.RecordRecoverySuccess(server.Config.Name)
	}
}

// GetHealthStatus returns the health status for a server
func (h *HealthMonitor) GetHealthStatus(serverName string) core.HealthStatus {
	h.mu.RLock()
	check, exists := h.healthChecks[serverName]
	h.mu.RUnlock()
	
	if !exists {
		return core.HealthStatusUnknown
	}
	
	check.mu.RLock()
	defer check.mu.RUnlock()
	
	if check.ConsecutiveFails == 0 {
		return core.HealthStatusHealthy
	} else if check.ConsecutiveFails < h.unhealthyThreshold {
		return core.HealthStatusDegraded
	}
	
	return core.HealthStatusUnhealthy
}

// GetHealthMetrics returns health metrics for a server
func (h *HealthMonitor) GetHealthMetrics(serverName string) (*HealthMetrics, error) {
	h.mu.RLock()
	check, exists := h.healthChecks[serverName]
	h.mu.RUnlock()
	
	if !exists {
		return nil, core.ErrServerNotFound
	}
	
	check.mu.RLock()
	defer check.mu.RUnlock()
	
	if check.HealthMetrics == nil {
		return nil, fmt.Errorf("no health metrics available")
	}
	
	// Return a copy to avoid race conditions
	metrics := *check.HealthMetrics
	return &metrics, nil
}

// SetHealthCheckInterval updates the health check interval
func (h *HealthMonitor) SetHealthCheckInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkInterval = interval
}

// SetUnhealthyThreshold updates the consecutive failure threshold
func (h *HealthMonitor) SetUnhealthyThreshold(threshold int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unhealthyThreshold = threshold
}