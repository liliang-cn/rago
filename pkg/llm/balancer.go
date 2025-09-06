package llm

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// LoadBalancer handles provider selection based on different strategies
type LoadBalancer struct {
	strategy        string
	pool           *ProviderPool
	roundRobinIndex int
	mu             sync.Mutex
	
	// Metrics for intelligent load balancing
	providerMetrics map[string]*ProviderMetrics
	metricsMu      sync.RWMutex
}

// ProviderMetrics tracks performance metrics for a provider
type ProviderMetrics struct {
	RequestCount     int64         `json:"request_count"`
	SuccessCount     int64         `json:"success_count"`
	FailureCount     int64         `json:"failure_count"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastUsed         time.Time     `json:"last_used"`
	ActiveRequests   int32         `json:"active_requests"`
	
	// Rolling window metrics (last N requests)
	recentLatencies  []time.Duration
	recentResults    []bool // true for success, false for failure
	windowSize       int
	windowIndex      int
}

// LoadBalancingStrategy defines different load balancing strategies
type LoadBalancingStrategy string

const (
	StrategyRoundRobin      LoadBalancingStrategy = "round_robin"
	StrategyWeighted        LoadBalancingStrategy = "weighted"
	StrategyLeastConnections LoadBalancingStrategy = "least_connections"
	StrategyResponseTime    LoadBalancingStrategy = "response_time"
	StrategyAdaptive        LoadBalancingStrategy = "adaptive"
)

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(strategy string, pool *ProviderPool) *LoadBalancer {
	return &LoadBalancer{
		strategy:        strategy,
		pool:           pool,
		providerMetrics: make(map[string]*ProviderMetrics),
		roundRobinIndex: int(time.Now().UnixNano() % 1000), // Random start
	}
}

// SelectProvider selects the best provider based on the configured strategy
func (lb *LoadBalancer) SelectProvider() (*ProviderEntry, error) {
	// Get healthy providers
	healthyProviders := lb.getHealthyProviders()
	if len(healthyProviders) == 0 {
		return nil, core.ErrNoProvidersAvailable
	}
	
	var selected *ProviderEntry
	
	switch LoadBalancingStrategy(lb.strategy) {
	case StrategyWeighted:
		selected = lb.selectWeightedProvider(healthyProviders)
	case StrategyLeastConnections:
		selected = lb.selectLeastConnectionsProvider(healthyProviders)
	case StrategyResponseTime:
		selected = lb.selectResponseTimeProvider(healthyProviders)
	case StrategyAdaptive:
		selected = lb.selectAdaptiveProvider(healthyProviders)
	case StrategyRoundRobin:
		fallthrough
	default:
		selected = lb.selectRoundRobinProvider(healthyProviders)
	}
	
	if selected != nil {
		lb.updateProviderSelection(selected.Name)
	}
	
	return selected, nil
}

// RecordRequest records metrics for a provider request
func (lb *LoadBalancer) RecordRequest(providerName string, latency time.Duration, success bool) {
	lb.metricsMu.Lock()
	defer lb.metricsMu.Unlock()
	
	metrics, exists := lb.providerMetrics[providerName]
	if !exists {
		metrics = &ProviderMetrics{
			recentLatencies: make([]time.Duration, 0, 100),
			recentResults:   make([]bool, 0, 100),
			windowSize:      100,
		}
		lb.providerMetrics[providerName] = metrics
	}
	
	// Update basic metrics
	metrics.RequestCount++
	if success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}
	
	// Update average latency (exponential moving average)
	if metrics.AverageLatency == 0 {
		metrics.AverageLatency = latency
	} else {
		// α = 0.1 for exponential moving average
		alpha := 0.1
		metrics.AverageLatency = time.Duration(
			float64(metrics.AverageLatency)*(1-alpha) + float64(latency)*alpha,
		)
	}
	
	// Update rolling window
	if len(metrics.recentLatencies) < metrics.windowSize {
		metrics.recentLatencies = append(metrics.recentLatencies, latency)
		metrics.recentResults = append(metrics.recentResults, success)
	} else {
		metrics.recentLatencies[metrics.windowIndex] = latency
		metrics.recentResults[metrics.windowIndex] = success
		metrics.windowIndex = (metrics.windowIndex + 1) % metrics.windowSize
	}
	
	metrics.LastUsed = time.Now()
}

// GetProviderMetrics returns metrics for a specific provider
func (lb *LoadBalancer) GetProviderMetrics(providerName string) *ProviderMetrics {
	lb.metricsMu.RLock()
	defer lb.metricsMu.RUnlock()
	
	metrics, exists := lb.providerMetrics[providerName]
	if !exists {
		return nil
	}
	
	// Return a copy to avoid race conditions
	return &ProviderMetrics{
		RequestCount:     metrics.RequestCount,
		SuccessCount:     metrics.SuccessCount,
		FailureCount:     metrics.FailureCount,
		AverageLatency:   metrics.AverageLatency,
		LastUsed:         metrics.LastUsed,
		ActiveRequests:   metrics.ActiveRequests,
	}
}

// GetAllProviderMetrics returns metrics for all providers
func (lb *LoadBalancer) GetAllProviderMetrics() map[string]*ProviderMetrics {
	lb.metricsMu.RLock()
	defer lb.metricsMu.RUnlock()
	
	result := make(map[string]*ProviderMetrics)
	for name, metrics := range lb.providerMetrics {
		result[name] = &ProviderMetrics{
			RequestCount:     metrics.RequestCount,
			SuccessCount:     metrics.SuccessCount,
			FailureCount:     metrics.FailureCount,
			AverageLatency:   metrics.AverageLatency,
			LastUsed:         metrics.LastUsed,
			ActiveRequests:   metrics.ActiveRequests,
		}
	}
	
	return result
}

// === PRIVATE METHODS ===

// getHealthyProviders returns a list of healthy providers
func (lb *LoadBalancer) getHealthyProviders() []*ProviderEntry {
	lb.pool.mu.RLock()
	defer lb.pool.mu.RUnlock()
	
	healthy := make([]*ProviderEntry, 0)
	for _, entry := range lb.pool.providers {
		entry.mu.RLock()
		if entry.Health == core.HealthStatusHealthy && entry.CircuitState == CircuitClosed {
			healthy = append(healthy, entry)
		}
		entry.mu.RUnlock()
	}
	
	return healthy
}

// selectRoundRobinProvider selects providers in round-robin fashion
func (lb *LoadBalancer) selectRoundRobinProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	lb.mu.Lock()
	selected := providers[lb.roundRobinIndex%len(providers)]
	lb.roundRobinIndex++
	lb.mu.Unlock()
	
	return selected
}

// selectWeightedProvider selects providers based on weights
func (lb *LoadBalancer) selectWeightedProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, provider := range providers {
		totalWeight += provider.Weight
	}
	
	if totalWeight == 0 {
		// Fall back to round robin if no weights
		return lb.selectRoundRobinProvider(providers)
	}
	
	// Use weighted random selection
	random := rand.Intn(totalWeight)
	currentWeight := 0
	
	for _, provider := range providers {
		currentWeight += provider.Weight
		if random < currentWeight {
			return provider
		}
	}
	
	return providers[0] // Fallback
}

// selectLeastConnectionsProvider selects the provider with the fewest active connections
func (lb *LoadBalancer) selectLeastConnectionsProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	lb.metricsMu.RLock()
	defer lb.metricsMu.RUnlock()
	
	var selected *ProviderEntry
	minConnections := int32(^uint32(0) >> 1) // Max int32
	
	for _, provider := range providers {
		metrics, exists := lb.providerMetrics[provider.Name]
		connections := int32(0)
		if exists {
			connections = metrics.ActiveRequests
		}
		
		if connections < minConnections {
			minConnections = connections
			selected = provider
		}
	}
	
	return selected
}

// selectResponseTimeProvider selects the provider with the best response time
func (lb *LoadBalancer) selectResponseTimeProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	lb.metricsMu.RLock()
	defer lb.metricsMu.RUnlock()
	
	var selected *ProviderEntry
	bestLatency := time.Duration(^uint64(0) >> 1) // Max duration
	
	for _, provider := range providers {
		metrics, exists := lb.providerMetrics[provider.Name]
		if !exists {
			// New provider, give it a chance
			selected = provider
			break
		}
		
		if metrics.AverageLatency < bestLatency {
			bestLatency = metrics.AverageLatency
			selected = provider
		}
	}
	
	if selected == nil {
		// Fallback to round robin
		return lb.selectRoundRobinProvider(providers)
	}
	
	return selected
}

// selectAdaptiveProvider uses an adaptive algorithm considering multiple factors
func (lb *LoadBalancer) selectAdaptiveProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	lb.metricsMu.RLock()
	defer lb.metricsMu.RUnlock()
	
	type providerScore struct {
		provider *ProviderEntry
		score    float64
	}
	
	scores := make([]providerScore, 0, len(providers))
	
	for _, provider := range providers {
		metrics, exists := lb.providerMetrics[provider.Name]
		if !exists {
			// New provider gets a high score to encourage load distribution
			scores = append(scores, providerScore{
				provider: provider,
				score:    1000.0,
			})
			continue
		}
		
		// Calculate composite score based on multiple factors
		score := lb.calculateAdaptiveScore(provider, metrics)
		scores = append(scores, providerScore{
			provider: provider,
			score:    score,
		})
	}
	
	// Sort by score (higher is better)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	return scores[0].provider
}

// calculateAdaptiveScore calculates a composite score for adaptive load balancing
func (lb *LoadBalancer) calculateAdaptiveScore(provider *ProviderEntry, metrics *ProviderMetrics) float64 {
	score := 1000.0 // Base score
	
	// Factor 1: Success rate (0-1, higher is better)
	successRate := 0.0
	if metrics.RequestCount > 0 {
		successRate = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	}
	score *= successRate
	
	// Factor 2: Response time (lower is better)
	if metrics.AverageLatency > 0 {
		// Convert to seconds and invert (so lower latency = higher score)
		latencySeconds := metrics.AverageLatency.Seconds()
		latencyScore := 1.0 / (1.0 + latencySeconds)
		score *= latencyScore
	}
	
	// Factor 3: Load (active requests, lower is better)
	loadScore := 1.0 / (1.0 + float64(metrics.ActiveRequests))
	score *= loadScore
	
	// Factor 4: Provider weight
	score *= float64(provider.Weight) / 10.0
	
	// Factor 5: Recent performance (based on rolling window)
	if len(metrics.recentResults) > 0 {
		recentSuccesses := 0
		for _, success := range metrics.recentResults {
			if success {
				recentSuccesses++
			}
		}
		recentSuccessRate := float64(recentSuccesses) / float64(len(metrics.recentResults))
		score *= recentSuccessRate
	}
	
	return score
}

// updateProviderSelection updates metrics when a provider is selected
func (lb *LoadBalancer) updateProviderSelection(providerName string) {
	lb.metricsMu.Lock()
	defer lb.metricsMu.Unlock()
	
	metrics, exists := lb.providerMetrics[providerName]
	if !exists {
		metrics = &ProviderMetrics{
			recentLatencies: make([]time.Duration, 0, 100),
			recentResults:   make([]bool, 0, 100),
			windowSize:      100,
		}
		lb.providerMetrics[providerName] = metrics
	}
	
	metrics.ActiveRequests++
	metrics.LastUsed = time.Now()
}

// FinishRequest should be called when a request completes
func (lb *LoadBalancer) FinishRequest(providerName string) {
	lb.metricsMu.Lock()
	defer lb.metricsMu.Unlock()
	
	metrics, exists := lb.providerMetrics[providerName]
	if exists && metrics.ActiveRequests > 0 {
		metrics.ActiveRequests--
	}
}