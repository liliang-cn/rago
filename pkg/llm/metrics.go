package llm

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	// Atomic counters for thread-safe operations
	totalRequests      int64
	totalSuccesses     int64
	totalFailures      int64
	totalTokensUsed    int64
	
	// Provider-specific metrics
	providerMetrics map[string]*ProviderPerformanceMetrics
	
	// Time-based metrics
	startTime time.Time
	
	mu sync.RWMutex
}

// ProviderPerformanceMetrics tracks detailed performance metrics for a provider
type ProviderPerformanceMetrics struct {
	// Request metrics
	RequestCount      int64         `json:"request_count"`
	SuccessCount      int64         `json:"success_count"`
	FailureCount      int64         `json:"failure_count"`
	TotalLatency      time.Duration `json:"total_latency"`
	MinLatency        time.Duration `json:"min_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	
	// Token usage metrics
	TotalTokens       int64 `json:"total_tokens"`
	PromptTokens      int64 `json:"prompt_tokens"`
	CompletionTokens  int64 `json:"completion_tokens"`
	
	// Error tracking
	ErrorsByType      map[string]int64 `json:"errors_by_type"`
	
	// Time tracking
	FirstRequest      time.Time `json:"first_request"`
	LastRequest       time.Time `json:"last_request"`
	
	mu sync.RWMutex
}

// ServiceMetrics provides comprehensive service-level metrics
type ServiceMetrics struct {
	// Overall service metrics
	TotalRequests     int64         `json:"total_requests"`
	TotalSuccesses    int64         `json:"total_successes"`
	TotalFailures     int64         `json:"total_failures"`
	SuccessRate       float64       `json:"success_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
	
	// Token usage
	TotalTokensUsed   int64 `json:"total_tokens_used"`
	
	// Provider metrics
	ProviderMetrics   map[string]*ProviderSummaryMetrics `json:"provider_metrics"`
	
	// Time metrics
	ServiceUptime     time.Duration `json:"service_uptime"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	
	// Health status
	HealthStatus      core.HealthStatus `json:"health_status"`
	
	// Generated at
	GeneratedAt       time.Time `json:"generated_at"`
}

// ProviderSummaryMetrics provides a summary of provider performance
type ProviderSummaryMetrics struct {
	RequestCount      int64         `json:"request_count"`
	SuccessRate       float64       `json:"success_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
	TokensUsed        int64         `json:"tokens_used"`
	LastUsed          time.Time     `json:"last_used"`
	ErrorCount        int64         `json:"error_count"`
	Health            core.HealthStatus `json:"health"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerMetrics: make(map[string]*ProviderPerformanceMetrics),
		startTime:       time.Now(),
	}
}

// RecordRequest records metrics for a completed request
func (m *MetricsCollector) RecordRequest(providerName string, success bool, latency time.Duration, tokenUsage *core.TokenUsage, errorType string) {
	// Update atomic counters
	atomic.AddInt64(&m.totalRequests, 1)
	
	if success {
		atomic.AddInt64(&m.totalSuccesses, 1)
	} else {
		atomic.AddInt64(&m.totalFailures, 1)
	}
	
	if tokenUsage != nil {
		atomic.AddInt64(&m.totalTokensUsed, int64(tokenUsage.TotalTokens))
	}
	
	// Update provider-specific metrics
	m.mu.Lock()
	providerMetrics, exists := m.providerMetrics[providerName]
	if !exists {
		providerMetrics = &ProviderPerformanceMetrics{
			ErrorsByType: make(map[string]int64),
			FirstRequest: time.Now(),
			MinLatency:   latency,
			MaxLatency:   latency,
		}
		m.providerMetrics[providerName] = providerMetrics
	}
	m.mu.Unlock()
	
	// Update provider metrics
	providerMetrics.mu.Lock()
	defer providerMetrics.mu.Unlock()
	
	providerMetrics.RequestCount++
	providerMetrics.LastRequest = time.Now()
	
	if success {
		providerMetrics.SuccessCount++
	} else {
		providerMetrics.FailureCount++
		if errorType != "" {
			providerMetrics.ErrorsByType[errorType]++
		}
	}
	
	// Update latency metrics
	providerMetrics.TotalLatency += latency
	if latency < providerMetrics.MinLatency || providerMetrics.MinLatency == 0 {
		providerMetrics.MinLatency = latency
	}
	if latency > providerMetrics.MaxLatency {
		providerMetrics.MaxLatency = latency
	}
	
	// Update token metrics
	if tokenUsage != nil {
		providerMetrics.TotalTokens += int64(tokenUsage.TotalTokens)
		providerMetrics.PromptTokens += int64(tokenUsage.PromptTokens)
		providerMetrics.CompletionTokens += int64(tokenUsage.CompletionTokens)
	}
}

// RecordStreamingRequest records metrics for a streaming request
func (m *MetricsCollector) RecordStreamingRequest(providerName string, success bool, duration time.Duration, chunks int, totalTokens int, errorType string) {
	tokenUsage := &core.TokenUsage{
		TotalTokens: totalTokens,
	}
	
	m.RecordRequest(providerName, success, duration, tokenUsage, errorType)
}

// GetServiceMetrics returns comprehensive service metrics
func (m *MetricsCollector) GetServiceMetrics(healthChecker *HealthChecker) *ServiceMetrics {
	totalRequests := atomic.LoadInt64(&m.totalRequests)
	totalSuccesses := atomic.LoadInt64(&m.totalSuccesses)
	totalFailures := atomic.LoadInt64(&m.totalFailures)
	totalTokens := atomic.LoadInt64(&m.totalTokensUsed)
	
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(totalSuccesses) / float64(totalRequests)
	}
	
	uptime := time.Since(m.startTime)
	requestsPerSecond := 0.0
	if uptime.Seconds() > 0 {
		requestsPerSecond = float64(totalRequests) / uptime.Seconds()
	}
	
	// Calculate average latency across all providers
	averageLatency := m.calculateAverageLatency()
	
	// Build provider summary metrics
	providerSummaries := m.buildProviderSummaries(healthChecker)
	
	// Get overall health status
	overallHealth := core.HealthStatusUnknown
	if healthChecker != nil {
		overallHealth = healthChecker.GetOverallHealth()
	}
	
	return &ServiceMetrics{
		TotalRequests:     totalRequests,
		TotalSuccesses:    totalSuccesses,
		TotalFailures:     totalFailures,
		SuccessRate:       successRate,
		AverageLatency:    averageLatency,
		TotalTokensUsed:   totalTokens,
		ProviderMetrics:   providerSummaries,
		ServiceUptime:     uptime,
		RequestsPerSecond: requestsPerSecond,
		HealthStatus:      overallHealth,
		GeneratedAt:       time.Now(),
	}
}

// GetProviderMetrics returns metrics for a specific provider
func (m *MetricsCollector) GetProviderMetrics(providerName string) *ProviderPerformanceMetrics {
	m.mu.RLock()
	providerMetrics, exists := m.providerMetrics[providerName]
	m.mu.RUnlock()
	
	if !exists {
		return nil
	}
	
	// Return a copy to avoid race conditions
	providerMetrics.mu.RLock()
	defer providerMetrics.mu.RUnlock()
	
	errorsCopy := make(map[string]int64)
	for k, v := range providerMetrics.ErrorsByType {
		errorsCopy[k] = v
	}
	
	return &ProviderPerformanceMetrics{
		RequestCount:      providerMetrics.RequestCount,
		SuccessCount:      providerMetrics.SuccessCount,
		FailureCount:      providerMetrics.FailureCount,
		TotalLatency:      providerMetrics.TotalLatency,
		MinLatency:        providerMetrics.MinLatency,
		MaxLatency:        providerMetrics.MaxLatency,
		TotalTokens:       providerMetrics.TotalTokens,
		PromptTokens:      providerMetrics.PromptTokens,
		CompletionTokens:  providerMetrics.CompletionTokens,
		ErrorsByType:      errorsCopy,
		FirstRequest:      providerMetrics.FirstRequest,
		LastRequest:       providerMetrics.LastRequest,
	}
}

// Reset resets all metrics
func (m *MetricsCollector) Reset() {
	atomic.StoreInt64(&m.totalRequests, 0)
	atomic.StoreInt64(&m.totalSuccesses, 0)
	atomic.StoreInt64(&m.totalFailures, 0)
	atomic.StoreInt64(&m.totalTokensUsed, 0)
	
	m.mu.Lock()
	m.providerMetrics = make(map[string]*ProviderPerformanceMetrics)
	m.startTime = time.Now()
	m.mu.Unlock()
}

// === PRIVATE METHODS ===

// calculateAverageLatency calculates the weighted average latency across all providers
func (m *MetricsCollector) calculateAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	totalLatency := time.Duration(0)
	totalRequests := int64(0)
	
	for _, metrics := range m.providerMetrics {
		metrics.mu.RLock()
		totalLatency += metrics.TotalLatency
		totalRequests += metrics.RequestCount
		metrics.mu.RUnlock()
	}
	
	if totalRequests == 0 {
		return 0
	}
	
	return time.Duration(int64(totalLatency) / totalRequests)
}

// buildProviderSummaries builds summary metrics for all providers
func (m *MetricsCollector) buildProviderSummaries(healthChecker *HealthChecker) map[string]*ProviderSummaryMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	summaries := make(map[string]*ProviderSummaryMetrics)
	healthStatus := make(map[string]core.HealthStatus)
	
	if healthChecker != nil {
		healthStatus = healthChecker.GetHealthStatus()
	}
	
	for name, metrics := range m.providerMetrics {
		metrics.mu.RLock()
		
		successRate := 0.0
		if metrics.RequestCount > 0 {
			successRate = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
		}
		
		averageLatency := time.Duration(0)
		if metrics.RequestCount > 0 {
			averageLatency = time.Duration(int64(metrics.TotalLatency) / metrics.RequestCount)
		}
		
		health, exists := healthStatus[name]
		if !exists {
			health = core.HealthStatusUnknown
		}
		
		summaries[name] = &ProviderSummaryMetrics{
			RequestCount:   metrics.RequestCount,
			SuccessRate:    successRate,
			AverageLatency: averageLatency,
			TokensUsed:     metrics.TotalTokens,
			LastUsed:       metrics.LastRequest,
			ErrorCount:     metrics.FailureCount,
			Health:         health,
		}
		
		metrics.mu.RUnlock()
	}
	
	return summaries
}

// GetDetailedMetrics returns detailed metrics including error breakdowns
func (m *MetricsCollector) GetDetailedMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	// Overall metrics
	metrics["total_requests"] = atomic.LoadInt64(&m.totalRequests)
	metrics["total_successes"] = atomic.LoadInt64(&m.totalSuccesses)
	metrics["total_failures"] = atomic.LoadInt64(&m.totalFailures)
	metrics["total_tokens"] = atomic.LoadInt64(&m.totalTokensUsed)
	metrics["uptime_seconds"] = time.Since(m.startTime).Seconds()
	
	// Provider details
	m.mu.RLock()
	providers := make(map[string]interface{})
	for name, providerMetrics := range m.providerMetrics {
		providerMetrics.mu.RLock()
		
		providers[name] = map[string]interface{}{
			"request_count":      providerMetrics.RequestCount,
			"success_count":      providerMetrics.SuccessCount,
			"failure_count":      providerMetrics.FailureCount,
			"total_latency_ms":   providerMetrics.TotalLatency.Milliseconds(),
			"min_latency_ms":     providerMetrics.MinLatency.Milliseconds(),
			"max_latency_ms":     providerMetrics.MaxLatency.Milliseconds(),
			"total_tokens":       providerMetrics.TotalTokens,
			"prompt_tokens":      providerMetrics.PromptTokens,
			"completion_tokens":  providerMetrics.CompletionTokens,
			"errors_by_type":     providerMetrics.ErrorsByType,
			"first_request":      providerMetrics.FirstRequest,
			"last_request":       providerMetrics.LastRequest,
		}
		
		providerMetrics.mu.RUnlock()
	}
	metrics["providers"] = providers
	m.mu.RUnlock()
	
	return metrics
}