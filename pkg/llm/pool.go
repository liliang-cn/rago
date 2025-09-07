package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm/providers"
)

// ProviderEntry represents a provider instance in the pool with metadata
type ProviderEntry struct {
	Name         string
	Provider     providers.Provider
	Config       core.ProviderConfig
	Health       core.HealthStatus
	Weight       int
	LastUsed     time.Time
	Failures     int
	CircuitState CircuitState
	LastCheck    time.Time
	mu           sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitClosed    CircuitState = "closed"     // Normal operation
	CircuitOpen      CircuitState = "open"       // Failing, requests rejected
	CircuitHalfOpen  CircuitState = "half_open"  // Testing if recovered
)

// ProviderPool manages a pool of LLM providers with health checking and load balancing
type ProviderPool struct {
	providers    map[string]*ProviderEntry
	factory      providers.ProviderFactory
	config       core.LoadBalancingConfig
	healthConfig core.HealthCheckConfig
	mu           sync.RWMutex
	
	// Health checking
	healthTicker  *time.Ticker
	healthStop    chan struct{}
	healthRunning bool
	
	// Circuit breaker settings
	maxFailures     int
	recoveryTimeout time.Duration
	
	// Load balancing
	roundRobinIndex int
}

// NewProviderPool creates a new provider pool
func NewProviderPool(config core.LoadBalancingConfig, healthConfig core.HealthCheckConfig) *ProviderPool {
	pool := &ProviderPool{
		providers:       make(map[string]*ProviderEntry),
		factory:         providers.NewProviderFactory(),
		config:          config,
		healthConfig:    healthConfig,
		healthStop:      make(chan struct{}),
		maxFailures:     3, // Default circuit breaker threshold
		recoveryTimeout: 30 * time.Second,
	}
	
	// Start health checking if enabled
	if healthConfig.Enabled && healthConfig.Interval > 0 {
		pool.startHealthChecking()
	}
	
	return pool
}

// AddProvider adds a provider to the pool
func (p *ProviderPool) AddProvider(name string, config core.ProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create provider instance using LLM pillar factory
	provider, err := p.factory.CreateProvider(config.Type, name, config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	// Create provider entry
	entry := &ProviderEntry{
		Name:         name,
		Provider:     provider,
		Config:       config,
		Health:       core.HealthStatusHealthy, // Start with healthy status
		Weight:       config.Weight,
		LastUsed:     time.Now(),
		Failures:     0,
		CircuitState: CircuitClosed,
		LastCheck:    time.Time{},
	}
	
	p.providers[name] = entry
	return nil
}

// RemoveProvider removes a provider from the pool
func (p *ProviderPool) RemoveProvider(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.providers[name]; !exists {
		return core.ErrProviderNotFound
	}
	
	delete(p.providers, name)
	return nil
}

// GetProvider selects a provider based on the load balancing strategy
func (p *ProviderPool) GetProvider() (*ProviderEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Filter healthy providers with closed circuits
	healthyProviders := make([]*ProviderEntry, 0)
	for _, entry := range p.providers {
		entry.mu.RLock()
		if entry.Health == core.HealthStatusHealthy && entry.CircuitState != CircuitOpen {
			healthyProviders = append(healthyProviders, entry)
		}
		entry.mu.RUnlock()
	}
	
	if len(healthyProviders) == 0 {
		return nil, core.ErrNoProvidersAvailable
	}
	
	// Apply load balancing strategy
	var selectedProvider *ProviderEntry
	switch p.config.Strategy {
	case "weighted":
		selectedProvider = p.selectWeightedProvider(healthyProviders)
	case "least_connections":
		selectedProvider = p.selectLeastUsedProvider(healthyProviders)
	case "round_robin":
		fallthrough
	default:
		selectedProvider = p.selectRoundRobinProvider(healthyProviders)
	}
	
	if selectedProvider != nil {
		selectedProvider.mu.Lock()
		selectedProvider.LastUsed = time.Now()
		selectedProvider.mu.Unlock()
	}
	
	return selectedProvider, nil
}

// ListProviders returns information about all providers
func (p *ProviderPool) ListProviders() []core.ProviderInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	providers := make([]core.ProviderInfo, 0, len(p.providers))
	for name, entry := range p.providers {
		entry.mu.RLock()
		providers = append(providers, core.ProviderInfo{
			Name:   name,
			Type:   entry.Config.Type,
			Model:  entry.Config.Model,
			Health: entry.Health,
			Weight: entry.Weight,
			Metadata: map[string]interface{}{
				"last_used":      entry.LastUsed,
				"failures":       entry.Failures,
				"circuit_state":  entry.CircuitState,
				"last_check":     entry.LastCheck,
			},
		})
		entry.mu.RUnlock()
	}
	
	return providers
}

// GetProviderHealth returns health status of all providers
func (p *ProviderPool) GetProviderHealth() map[string]core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	health := make(map[string]core.HealthStatus)
	for name, entry := range p.providers {
		entry.mu.RLock()
		health[name] = entry.Health
		entry.mu.RUnlock()
	}
	
	return health
}

// GetActiveProviders returns a list of all active provider names
func (p *ProviderPool) GetActiveProviders() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	providers := make([]string, 0, len(p.providers))
	for name, entry := range p.providers {
		entry.mu.RLock()
		if entry.Health == core.HealthStatusHealthy || entry.Health == core.HealthStatusUnknown {
			providers = append(providers, name)
		}
		entry.mu.RUnlock()
	}
	
	return providers
}

// GetProviderByName returns a specific provider by name
func (p *ProviderPool) GetProviderByName(name string) providers.Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if entry, exists := p.providers[name]; exists {
		return entry.Provider
	}
	
	return nil
}

// RecordFailure records a failure for a provider and updates circuit breaker state
func (p *ProviderPool) RecordFailure(providerName string) {
	p.mu.RLock()
	entry, exists := p.providers[providerName]
	p.mu.RUnlock()
	
	if !exists {
		return
	}
	
	entry.mu.Lock()
	defer entry.mu.Unlock()
	
	entry.Failures++
	
	// Open circuit if failure threshold exceeded
	if entry.Failures >= p.maxFailures && entry.CircuitState == CircuitClosed {
		entry.CircuitState = CircuitOpen
		entry.Health = core.HealthStatusUnhealthy
		
		// Schedule recovery attempt
		go p.scheduleCircuitRecovery(providerName)
	}
}

// RecordSuccess records a successful operation for a provider
func (p *ProviderPool) RecordSuccess(providerName string) {
	p.mu.RLock()
	entry, exists := p.providers[providerName]
	p.mu.RUnlock()
	
	if !exists {
		return
	}
	
	entry.mu.Lock()
	defer entry.mu.Unlock()
	
	entry.Failures = 0 // Reset failure count on success
	
	// Close circuit if it was half-open
	if entry.CircuitState == CircuitHalfOpen {
		entry.CircuitState = CircuitClosed
		entry.Health = core.HealthStatusHealthy
	}
}

// Close shuts down the provider pool
func (p *ProviderPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Stop health checking
	if p.healthRunning {
		close(p.healthStop)
		if p.healthTicker != nil {
			p.healthTicker.Stop()
		}
		p.healthRunning = false
	}
	
	// Clear providers
	p.providers = make(map[string]*ProviderEntry)
	
	return nil
}

// === PRIVATE METHODS ===

// startHealthChecking begins periodic health checking
func (p *ProviderPool) startHealthChecking() {
	if p.healthRunning {
		return
	}
	
	p.healthTicker = time.NewTicker(p.healthConfig.Interval)
	p.healthRunning = true
	
	go func() {
		for {
			select {
			case <-p.healthTicker.C:
				p.performHealthChecks()
			case <-p.healthStop:
				return
			}
		}
	}()
}

// performHealthChecks checks health of all providers
func (p *ProviderPool) performHealthChecks() {
	p.mu.RLock()
	entries := make([]*ProviderEntry, 0, len(p.providers))
	for _, entry := range p.providers {
		entries = append(entries, entry)
	}
	p.mu.RUnlock()
	
	// Check each provider's health
	for _, entry := range entries {
		go func(e *ProviderEntry) {
			err := p.checkProviderHealth(e)
			
			e.mu.Lock()
			e.LastCheck = time.Now()
			if err != nil {
				e.Health = core.HealthStatusUnhealthy
				e.Failures++
			} else {
				e.Health = core.HealthStatusHealthy
				e.Failures = 0
			}
			e.mu.Unlock()
		}(entry)
	}
}

// checkProviderHealth performs a health check on a provider
func (p *ProviderPool) checkProviderHealth(entry *ProviderEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.healthConfig.Timeout)
	defer cancel()
	
	return entry.Provider.Health(ctx)
}

// selectRoundRobinProvider selects the next provider in round-robin fashion
func (p *ProviderPool) selectRoundRobinProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	provider := providers[p.roundRobinIndex%len(providers)]
	p.roundRobinIndex++
	
	return provider
}

// selectWeightedProvider selects a provider based on weights
func (p *ProviderPool) selectWeightedProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	totalWeight := 0
	for _, provider := range providers {
		totalWeight += provider.Weight
	}
	
	if totalWeight == 0 {
		return providers[0] // Fallback to first provider if no weights
	}
	
	// Simple weighted selection - this could be improved with more sophisticated algorithms
	target := time.Now().UnixNano() % int64(totalWeight)
	current := int64(0)
	
	for _, provider := range providers {
		current += int64(provider.Weight)
		if current > target {
			return provider
		}
	}
	
	return providers[0] // Fallback
}

// selectLeastUsedProvider selects the provider that was used least recently
func (p *ProviderPool) selectLeastUsedProvider(providers []*ProviderEntry) *ProviderEntry {
	if len(providers) == 0 {
		return nil
	}
	
	leastUsed := providers[0]
	for _, provider := range providers[1:] {
		provider.mu.RLock()
		leastUsed.mu.RLock()
		if provider.LastUsed.Before(leastUsed.LastUsed) {
			leastUsed.mu.RUnlock()
			leastUsed = provider
			provider.mu.RUnlock()
		} else {
			provider.mu.RUnlock()
			leastUsed.mu.RUnlock()
		}
	}
	
	return leastUsed
}

// scheduleCircuitRecovery schedules a recovery attempt for an open circuit
func (p *ProviderPool) scheduleCircuitRecovery(providerName string) {
	time.Sleep(p.recoveryTimeout)
	
	p.mu.RLock()
	entry, exists := p.providers[providerName]
	p.mu.RUnlock()
	
	if !exists {
		return
	}
	
	entry.mu.Lock()
	if entry.CircuitState == CircuitOpen {
		entry.CircuitState = CircuitHalfOpen
	}
	entry.mu.Unlock()
}

