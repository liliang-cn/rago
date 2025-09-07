package llm

import (
	"context"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// HealthChecker manages health checking for providers
type HealthChecker struct {
	config       core.HealthCheckConfig
	pool         *ProviderPool
	ticker       *time.Ticker
	stopChan     chan struct{}
	running      bool
	mu           sync.RWMutex
}

// HealthMetrics contains detailed health metrics for a provider
type HealthMetrics struct {
	Status           core.HealthStatus `json:"status"`
	LastCheck        time.Time         `json:"last_check"`
	ResponseTime     time.Duration     `json:"response_time"`
	SuccessRate      float64           `json:"success_rate"`
	ConsecutiveFails int               `json:"consecutive_fails"`
	TotalChecks      int               `json:"total_checks"`
	TotalFailures    int               `json:"total_failures"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config core.HealthCheckConfig, pool *ProviderPool) *HealthChecker {
	return &HealthChecker{
		config:   config,
		pool:     pool,
		stopChan: make(chan struct{}),
	}
}

// Start begins health checking
func (h *HealthChecker) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.running {
		return nil
	}
	
	if !h.config.Enabled || h.config.Interval <= 0 {
		return nil
	}
	
	h.ticker = time.NewTicker(h.config.Interval)
	h.running = true
	
	// Start health check loop
	go h.healthCheckLoop()
	
	return nil
}

// CheckNow performs an immediate health check on all providers
func (h *HealthChecker) CheckNow(ctx context.Context) {
	h.CheckAllProviders(ctx)
}

// Stop stops health checking
func (h *HealthChecker) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if !h.running {
		return nil
	}
	
	close(h.stopChan)
	if h.ticker != nil {
		h.ticker.Stop()
	}
	h.running = false
	
	return nil
}

// CheckProvider performs a health check on a specific provider
func (h *HealthChecker) CheckProvider(ctx context.Context, providerName string) (*HealthMetrics, error) {
	h.pool.mu.RLock()
	entry, exists := h.pool.providers[providerName]
	h.pool.mu.RUnlock()
	
	if !exists {
		return nil, core.ErrProviderNotFound
	}
	
	startTime := time.Now()
	
	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()
	
	// Perform health check
	err := entry.Provider.Health(checkCtx)
	responseTime := time.Since(startTime)
	
	entry.mu.Lock()
	defer entry.mu.Unlock()
	
	// Update metrics
	entry.LastCheck = time.Now()
	
	metrics := &HealthMetrics{
		LastCheck:    entry.LastCheck,
		ResponseTime: responseTime,
	}
	
	if err != nil {
		entry.Health = core.HealthStatusUnhealthy
		entry.Failures++
		metrics.Status = core.HealthStatusUnhealthy
		metrics.ConsecutiveFails = entry.Failures
	} else {
		// Check response time to determine if degraded
		if responseTime > h.config.Timeout/2 {
			entry.Health = core.HealthStatusDegraded
			metrics.Status = core.HealthStatusDegraded
		} else {
			entry.Health = core.HealthStatusHealthy
			metrics.Status = core.HealthStatusHealthy
			entry.Failures = 0 // Reset failures on success
		}
		metrics.ConsecutiveFails = 0
	}
	
	return metrics, err
}

// CheckAllProviders performs health checks on all providers
func (h *HealthChecker) CheckAllProviders(ctx context.Context) map[string]*HealthMetrics {
	h.pool.mu.RLock()
	providerNames := make([]string, 0, len(h.pool.providers))
	for name := range h.pool.providers {
		providerNames = append(providerNames, name)
	}
	h.pool.mu.RUnlock()
	
	results := make(map[string]*HealthMetrics)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, name := range providerNames {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()
			
			metrics, err := h.CheckProvider(ctx, providerName)
			if err != nil {
				// Create error metrics
				metrics = &HealthMetrics{
					Status:           core.HealthStatusUnhealthy,
					LastCheck:        time.Now(),
					ConsecutiveFails: 1,
				}
			}
			
			mu.Lock()
			results[providerName] = metrics
			mu.Unlock()
		}(name)
	}
	
	wg.Wait()
	return results
}

// GetHealthStatus returns the current health status of all providers
func (h *HealthChecker) GetHealthStatus() map[string]core.HealthStatus {
	return h.pool.GetProviderHealth()
}

// IsHealthy returns true if the provider is healthy
func (h *HealthChecker) IsHealthy(providerName string) bool {
	health := h.GetHealthStatus()
	status, exists := health[providerName]
	return exists && status == core.HealthStatusHealthy
}

// GetOverallHealth returns the overall health status of the LLM service
func (h *HealthChecker) GetOverallHealth() core.HealthStatus {
	health := h.GetHealthStatus()
	
	if len(health) == 0 {
		return core.HealthStatusUnknown
	}
	
	healthyCount := 0
	degradedCount := 0
	totalCount := len(health)
	
	for _, status := range health {
		switch status {
		case core.HealthStatusHealthy:
			healthyCount++
		case core.HealthStatusDegraded:
			degradedCount++
		}
	}
	
	// At least one healthy provider means service is operational
	if healthyCount > 0 {
		// If more than half are healthy, service is healthy
		if float64(healthyCount)/float64(totalCount) > 0.5 {
			return core.HealthStatusHealthy
		}
		// Some healthy, some issues - degraded
		return core.HealthStatusDegraded
	}
	
	// No healthy providers
	return core.HealthStatusUnhealthy
}

// === PRIVATE METHODS ===

// healthCheckLoop runs the periodic health checking
func (h *HealthChecker) healthCheckLoop() {
	for {
		select {
		case <-h.ticker.C:
			h.performScheduledChecks()
		case <-h.stopChan:
			return
		}
	}
}

// performScheduledChecks performs health checks on all providers
func (h *HealthChecker) performScheduledChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout*2)
	defer cancel()
	
	h.CheckAllProviders(ctx)
}

// CircuitBreakerManager manages circuit breaker logic for providers
type CircuitBreakerManager struct {
	maxFailures     int
	recoveryTimeout time.Duration
	pool            *ProviderPool
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(maxFailures int, recoveryTimeout time.Duration, pool *ProviderPool) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		maxFailures:     maxFailures,
		recoveryTimeout: recoveryTimeout,
		pool:            pool,
	}
}

// ShouldAllowRequest determines if a request should be allowed for a provider
func (c *CircuitBreakerManager) ShouldAllowRequest(providerName string) bool {
	c.pool.mu.RLock()
	entry, exists := c.pool.providers[providerName]
	c.pool.mu.RUnlock()
	
	if !exists {
		return false
	}
	
	entry.mu.RLock()
	defer entry.mu.RUnlock()
	
	switch entry.CircuitState {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if enough time has passed for recovery attempt
		if time.Since(entry.LastCheck) > c.recoveryTimeout {
			// Allow one request to test recovery
			entry.CircuitState = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true // Allow request in half-open state
	default:
		return false
	}
}

// RecordResult records the result of a provider operation
func (c *CircuitBreakerManager) RecordResult(providerName string, success bool) {
	if success {
		c.pool.RecordSuccess(providerName)
	} else {
		c.pool.RecordFailure(providerName)
	}
}