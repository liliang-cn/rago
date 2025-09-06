package router

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock CostStorage implementation
type mockCostStorage struct {
	mu          sync.RWMutex
	usageStats  map[string]*UsageStats
	budgets     []*Budget
	alerts      []*CostAlert
	saveError   error
	loadError   error
}

func newMockCostStorage() *mockCostStorage {
	return &mockCostStorage{
		usageStats: make(map[string]*UsageStats),
		budgets:    make([]*Budget, 0),
		alerts:     make([]*CostAlert, 0),
	}
}

func (m *mockCostStorage) SaveUsageStats(stats *UsageStats) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usageStats[stats.Provider] = stats
	return nil
}

func (m *mockCostStorage) LoadUsageStats(provider string) (*UsageStats, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats, ok := m.usageStats[provider]
	if !ok {
		return nil, fmt.Errorf("no stats for provider %s", provider)
	}
	return stats, nil
}

func (m *mockCostStorage) SaveBudget(budget *Budget) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.budgets = append(m.budgets, budget)
	return nil
}

func (m *mockCostStorage) LoadBudgets() ([]*Budget, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.budgets, nil
}

func (m *mockCostStorage) SaveAlert(alert *CostAlert) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, alert)
	return nil
}

// Helper function to create test provider
func createTestProvider(name string, cost float64, latency time.Duration, status ProviderStatus) ProviderInfo {
	return ProviderInfo{
		Name:         name,
		Type:         domain.ProviderOpenAI,
		Status:       status,
		CostPerToken: cost,
		CostPerCall:  cost * 100,
		Latency:      latency,
		SuccessRate:  0.95,
		Priority:     1,
		Capabilities: []string{"chat", "completion"},
		Models: []ModelInfo{
			{
				Name:          "gpt-3.5-turbo",
				Type:          ModelTypeLLM,
				ContextLength: 4096,
				CostPerToken:  cost,
				Speed:         100.0,
				Quality:       0.8,
			},
		},
	}
}

func TestNewProviderRouter(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		storage := newMockCostStorage()
		router := NewProviderRouter(nil, storage)
		
		assert.NotNil(t, router)
		assert.NotNil(t, router.config)
		assert.Equal(t, true, router.config.EnableCostOptimization)
		assert.Equal(t, true, router.config.EnableLoadBalancing)
		assert.Equal(t, true, router.config.EnableFailover)
		assert.Equal(t, 3, router.config.MaxRetries)
		assert.NotNil(t, router.providers)
		assert.NotNil(t, router.breakers)
		assert.NotNil(t, router.costTracker)
		assert.NotNil(t, router.metrics)
		assert.Len(t, router.policies, 5)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &RouterConfig{
			EnableCostOptimization: false,
			EnableLoadBalancing:    false,
			EnableFailover:        false,
			MaxRetries:            5,
			RetryDelay:            2 * time.Second,
			HealthCheckInterval:   0, // Disable background tasks
			CostUpdateInterval:    0,
		}
		storage := newMockCostStorage()
		router := NewProviderRouter(config, storage)
		
		assert.NotNil(t, router)
		assert.Equal(t, config, router.config)
		assert.Equal(t, false, router.config.EnableCostOptimization)
		assert.Equal(t, 5, router.config.MaxRetries)
	})
}

func TestRegisterProvider(t *testing.T) {
	t.Run("valid provider", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		
		err := router.RegisterProvider(provider)
		require.NoError(t, err)
		
		router.mu.RLock()
		registered, exists := router.providers["provider1"]
		router.mu.RUnlock()
		
		assert.True(t, exists)
		assert.Equal(t, provider.Name, registered.Name)
		
		// Check circuit breaker was created
		assert.NotNil(t, router.breakers["provider1"])
		assert.Equal(t, BreakerStateClosed, router.breakers["provider1"].state)
	})

	t.Run("invalid provider - missing name", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		provider := createTestProvider("", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		
		err := router.RegisterProvider(provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider name is required")
	})

	t.Run("duplicate registration", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		
		err := router.RegisterProvider(provider)
		require.NoError(t, err)
		
		// Register again with different values
		provider2 := createTestProvider("provider1", 0.002, 200*time.Millisecond, ProviderStatusHealthy)
		err = router.RegisterProvider(provider2)
		require.NoError(t, err)
		
		// Should have updated values
		router.mu.RLock()
		registered := router.providers["provider1"]
		router.mu.RUnlock()
		
		assert.Equal(t, 0.002, registered.CostPerToken)
	})
}

func TestRoute(t *testing.T) {
	t.Run("single available provider", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type:     RequestTypeGeneration,
			Priority: 1,
		}
		
		selected, err := router.Route(context.Background(), request)
		require.NoError(t, err)
		assert.NotNil(t, selected)
		assert.Equal(t, "provider1", selected.Name)
	})

	t.Run("multiple providers - cost optimized", func(t *testing.T) {
		config := &RouterConfig{
			EnableCostOptimization: true,
			EnableLoadBalancing:    false,
			EnableFailover:        true,
			MaxRetries:            3,
			RetryDelay:            time.Second,
			HealthCheckInterval:   0,
			CostUpdateInterval:    0,
		}
		router := NewProviderRouter(config, newMockCostStorage())
		
		// Register providers with different costs
		provider1 := createTestProvider("expensive", 0.01, 100*time.Millisecond, ProviderStatusHealthy)
		provider2 := createTestProvider("cheap", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider3 := createTestProvider("medium", 0.005, 100*time.Millisecond, ProviderStatusHealthy)
		
		router.RegisterProvider(provider1)
		router.RegisterProvider(provider2)
		router.RegisterProvider(provider3)
		
		request := &RoutingRequest{
			Type:    RequestTypeGeneration,
			MaxCost: 1.0,
		}
		
		selected, err := router.Route(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, "cheap", selected.Name)
	})

	t.Run("no available providers", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		
		// Register offline provider
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusOffline)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		selected, err := router.Route(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, selected)
		assert.Contains(t, err.Error(), "no available providers")
	})

	t.Run("with capability requirements", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		
		provider1 := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider1.Capabilities = []string{"chat"}
		
		provider2 := createTestProvider("provider2", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider2.Capabilities = []string{"chat", "vision"}
		
		router.RegisterProvider(provider1)
		router.RegisterProvider(provider2)
		
		request := &RoutingRequest{
			Type:         RequestTypeGeneration,
			Requirements: []string{"vision"},
		}
		
		selected, err := router.Route(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, "provider2", selected.Name)
	})

	t.Run("with model requirements", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		
		provider1 := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider1.Models = []ModelInfo{
			{Name: "gpt-3.5-turbo", Type: ModelTypeLLM},
		}
		
		provider2 := createTestProvider("provider2", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider2.Models = []ModelInfo{
			{Name: "gpt-4", Type: ModelTypeLLM},
		}
		
		router.RegisterProvider(provider1)
		router.RegisterProvider(provider2)
		
		request := &RoutingRequest{
			Type:  RequestTypeGeneration,
			Model: "gpt-4",
		}
		
		selected, err := router.Route(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, "provider2", selected.Name)
	})

	t.Run("with budget constraints", func(t *testing.T) {
		config := &RouterConfig{
			EnableCostOptimization: true,
			EnableLoadBalancing:    false,
			EnableFailover:        true,
			MaxRetries:            3,
			RetryDelay:            time.Second,
			HealthCheckInterval:   0,
			CostUpdateInterval:    0,
		}
		router := NewProviderRouter(config, newMockCostStorage())
		
		// Add a budget
		router.costTracker.budgets["global"] = &Budget{
			Name:      "global",
			Limit:     10.0,
			Spent:     9.99,
			Period:    BudgetPeriodDaily,
			LastReset: time.Now(),
		}
		
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type:          RequestTypeGeneration,
			TokenEstimate: 100, // Would cost 0.1, exceeding budget
		}
		
		selected, err := router.Route(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, selected)
	})
}

func TestExecuteWithFallback(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		executed := false
		err := router.ExecuteWithFallback(context.Background(), request, func(p *ProviderInfo) error {
			executed = true
			assert.Equal(t, "provider1", p.Name)
			return nil
		})
		
		require.NoError(t, err)
		assert.True(t, executed)
		
		// Check metrics were recorded
		metrics := router.metrics.metrics["provider1"]
		assert.Equal(t, int64(1), metrics.SuccessCount)
		assert.Equal(t, int64(0), metrics.FailureCount)
	})

	t.Run("success with retry", func(t *testing.T) {
		config := &RouterConfig{
			EnableFailover:      true,
			MaxRetries:          3,
			RetryDelay:          10 * time.Millisecond,
			HealthCheckInterval: 0,
			CostUpdateInterval:  0,
		}
		router := NewProviderRouter(config, newMockCostStorage())
		
		provider1 := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider2 := createTestProvider("provider2", 0.002, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider1)
		router.RegisterProvider(provider2)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		attempts := 0
		err := router.ExecuteWithFallback(context.Background(), request, func(p *ProviderInfo) error {
			attempts++
			if attempts == 1 {
				return errors.New("first attempt failed")
			}
			return nil
		})
		
		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("all providers fail", func(t *testing.T) {
		config := &RouterConfig{
			EnableFailover:      true,
			MaxRetries:          2,
			RetryDelay:          10 * time.Millisecond,
			HealthCheckInterval: 0,
			CostUpdateInterval:  0,
		}
		router := NewProviderRouter(config, newMockCostStorage())
		
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		attempts := 0
		err := router.ExecuteWithFallback(context.Background(), request, func(p *ProviderInfo) error {
			attempts++
			return errors.New("always fails")
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all providers failed")
		assert.Equal(t, 2, attempts)
		
		// Check failure was recorded
		metrics := router.metrics.metrics["provider1"]
		assert.Equal(t, int64(0), metrics.SuccessCount)
		assert.Equal(t, int64(2), metrics.FailureCount)
	})

	t.Run("no failover when disabled", func(t *testing.T) {
		config := &RouterConfig{
			EnableFailover:      false,
			MaxRetries:          5, // Should be ignored
			RetryDelay:          10 * time.Millisecond,
			HealthCheckInterval: 0,
			CostUpdateInterval:  0,
		}
		router := NewProviderRouter(config, newMockCostStorage())
		
		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		router.RegisterProvider(provider)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		attempts := 0
		err := router.ExecuteWithFallback(context.Background(), request, func(p *ProviderInfo) error {
			attempts++
			return errors.New("fails")
		})
		
		assert.Error(t, err)
		assert.Equal(t, 1, attempts) // Only one attempt when failover is disabled
	})
}

func TestConcurrentRouting(t *testing.T) {
	router := NewProviderRouter(nil, newMockCostStorage())
	
	// Register multiple providers
	for i := 0; i < 5; i++ {
		provider := createTestProvider(
			fmt.Sprintf("provider%d", i),
			float64(i+1)*0.001,
			time.Duration(i+1)*100*time.Millisecond,
			ProviderStatusHealthy,
		)
		router.RegisterProvider(provider)
	}
	
	// Concurrent routing requests
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			request := &RoutingRequest{
				Type:     RequestTypeGeneration,
				Priority: id % 3,
			}
			
			provider, err := router.Route(context.Background(), request)
			if err != nil {
				errors <- err
				return
			}
			
			if provider == nil {
				errors <- fmt.Errorf("nil provider returned")
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent routing error: %v", err)
	}
}

func TestProviderFiltering(t *testing.T) {
	t.Run("filter by status", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		
		healthy := createTestProvider("healthy", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		degraded := createTestProvider("degraded", 0.001, 100*time.Millisecond, ProviderStatusDegraded)
		unhealthy := createTestProvider("unhealthy", 0.001, 100*time.Millisecond, ProviderStatusUnhealthy)
		offline := createTestProvider("offline", 0.001, 100*time.Millisecond, ProviderStatusOffline)
		
		router.RegisterProvider(healthy)
		router.RegisterProvider(degraded)
		router.RegisterProvider(unhealthy)
		router.RegisterProvider(offline)
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		available := router.getAvailableProviders(request)
		
		// Should only include healthy and degraded
		assert.Len(t, available, 2)
		names := make(map[string]bool)
		for _, p := range available {
			names[p.Name] = true
		}
		assert.True(t, names["healthy"])
		assert.True(t, names["degraded"])
		assert.False(t, names["unhealthy"])
		assert.False(t, names["offline"])
	})

	t.Run("filter by circuit breaker", func(t *testing.T) {
		router := NewProviderRouter(nil, newMockCostStorage())
		
		provider1 := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		provider2 := createTestProvider("provider2", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		
		router.RegisterProvider(provider1)
		router.RegisterProvider(provider2)
		
		// Open circuit breaker for provider1
		breaker1 := router.breakers["provider1"]
		breaker1.mu.Lock()
		breaker1.state = BreakerStateOpen
		breaker1.lastFailure = time.Now()
		breaker1.mu.Unlock()
		
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}
		
		available := router.getAvailableProviders(request)
		
		// Should only include provider2
		assert.Len(t, available, 1)
		assert.Equal(t, "provider2", available[0].Name)
	})
}

func TestProviderHealthChecking(t *testing.T) {
	router := NewProviderRouter(nil, newMockCostStorage())
	
	provider1 := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
	provider2 := createTestProvider("provider2", 0.001, 100*time.Millisecond, ProviderStatusUnhealthy)
	
	router.RegisterProvider(provider1)
	router.RegisterProvider(provider2)
	
	// Run health check
	router.checkProviderHealth()
	
	// After health check, all providers should be healthy (simplified implementation)
	router.mu.RLock()
	p1 := router.providers["provider1"]
	p2 := router.providers["provider2"]
	router.mu.RUnlock()
	
	assert.Equal(t, ProviderStatusHealthy, p1.Status)
	assert.Equal(t, ProviderStatusHealthy, p2.Status)
}