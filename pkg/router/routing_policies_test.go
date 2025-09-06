package router

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostOptimizedPolicy(t *testing.T) {
	policy := &CostOptimizedPolicy{}

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "cost_optimized", policy.Name())
	})

	t.Run("selects cheapest provider", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "expensive", CostPerToken: 0.01},
			{Name: "cheap", CostPerToken: 0.001},
			{Name: "medium", CostPerToken: 0.005},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "cheap", selected.Name)
		assert.Equal(t, 0.001, selected.CostPerToken)
	})

	t.Run("with equal costs", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "provider1", CostPerToken: 0.001},
			{Name: "provider2", CostPerToken: 0.001},
			{Name: "provider3", CostPerToken: 0.001},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.NotNil(t, selected)
		assert.Equal(t, 0.001, selected.CostPerToken)
	})

	t.Run("empty provider list", func(t *testing.T) {
		providers := []ProviderInfo{}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		assert.Error(t, err)
		assert.Nil(t, selected)
		assert.Contains(t, err.Error(), "no providers available")
	})

	t.Run("single provider", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "only", CostPerToken: 0.01},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "only", selected.Name)
	})
}

func TestLatencyOptimizedPolicy(t *testing.T) {
	policy := &LatencyOptimizedPolicy{}

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "latency_optimized", policy.Name())
	})

	t.Run("selects fastest provider", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "slow", Latency: 1000 * time.Millisecond},
			{Name: "fast", Latency: 100 * time.Millisecond},
			{Name: "medium", Latency: 500 * time.Millisecond},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "fast", selected.Name)
		assert.Equal(t, 100*time.Millisecond, selected.Latency)
	})

	t.Run("with equal latencies", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "provider1", Latency: 100 * time.Millisecond},
			{Name: "provider2", Latency: 100 * time.Millisecond},
			{Name: "provider3", Latency: 100 * time.Millisecond},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.NotNil(t, selected)
		assert.Equal(t, 100*time.Millisecond, selected.Latency)
	})

	t.Run("zero latency", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "instant", Latency: 0},
			{Name: "slow", Latency: 100 * time.Millisecond},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "instant", selected.Name)
		assert.Equal(t, time.Duration(0), selected.Latency)
	})

	t.Run("empty provider list", func(t *testing.T) {
		providers := []ProviderInfo{}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		assert.Error(t, err)
		assert.Nil(t, selected)
	})
}

func TestQualityOptimizedPolicy(t *testing.T) {
	policy := &QualityOptimizedPolicy{}

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "quality_optimized", policy.Name())
	})

	t.Run("selects by success rate", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "unreliable", SuccessRate: 0.70, Priority: 1},
			{Name: "reliable", SuccessRate: 0.99, Priority: 1},
			{Name: "medium", SuccessRate: 0.85, Priority: 1},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "reliable", selected.Name)
		assert.Equal(t, 0.99, selected.SuccessRate)
	})

	t.Run("selects by priority when success rate equal", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "low-priority", SuccessRate: 0.95, Priority: 1},
			{Name: "high-priority", SuccessRate: 0.95, Priority: 10},
			{Name: "medium-priority", SuccessRate: 0.95, Priority: 5},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "high-priority", selected.Name)
		assert.Equal(t, 10, selected.Priority)
	})

	t.Run("combined success rate and priority", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "high-success-low-priority", SuccessRate: 0.99, Priority: 1},
			{Name: "low-success-high-priority", SuccessRate: 0.80, Priority: 10},
			{Name: "medium-both", SuccessRate: 0.90, Priority: 5},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		// Should select by success rate first
		assert.Equal(t, "high-success-low-priority", selected.Name)
	})

	t.Run("empty provider list", func(t *testing.T) {
		providers := []ProviderInfo{}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		assert.Error(t, err)
		assert.Nil(t, selected)
	})
}

func TestLoadBalancedPolicy(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}
		assert.Equal(t, "load_balanced", policy.Name())
	})

	t.Run("round-robin selection", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}

		providers := []ProviderInfo{
			{Name: "provider1"},
			{Name: "provider2"},
			{Name: "provider3"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		// First round
		selected1, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)

		selected2, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)

		selected3, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)

		// Second round - should wrap around
		selected4, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)

		// Check round-robin behavior
		names := []string{selected1.Name, selected2.Name, selected3.Name, selected4.Name}
		
		// Should cycle through all providers
		assert.Contains(t, names[:3], "provider1")
		assert.Contains(t, names[:3], "provider2")
		assert.Contains(t, names[:3], "provider3")
		
		// Fourth selection should be from the beginning
		assert.Contains(t, []string{"provider1", "provider2", "provider3"}, selected4.Name)
	})

	t.Run("single provider", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}

		providers := []ProviderInfo{
			{Name: "only"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		// Should always return the same provider
		for i := 0; i < 5; i++ {
			selected, err := policy.SelectProvider(context.Background(), request, providers)
			require.NoError(t, err)
			assert.Equal(t, "only", selected.Name)
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}

		providers := []ProviderInfo{
			{Name: "provider1"},
			{Name: "provider2"},
			{Name: "provider3"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		var wg sync.WaitGroup
		selections := make([]string, 100)
		var mu sync.Mutex

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				selected, err := policy.SelectProvider(context.Background(), request, providers)
				if err == nil {
					mu.Lock()
					selections[idx] = selected.Name
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Count selections per provider
		counts := make(map[string]int)
		for _, name := range selections {
			if name != "" {
				counts[name]++
			}
		}

		// Should be roughly balanced (within reasonable variance)
		for _, provider := range providers {
			count := counts[provider.Name]
			assert.Greater(t, count, 20) // At least 20% of requests
			assert.Less(t, count, 50)    // At most 50% of requests
		}
	})

	t.Run("empty provider list", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}
		providers := []ProviderInfo{}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		assert.Error(t, err)
		assert.Nil(t, selected)
	})

	t.Run("changing provider list", func(t *testing.T) {
		policy := &LoadBalancedPolicy{}

		providers1 := []ProviderInfo{
			{Name: "provider1"},
			{Name: "provider2"},
		}

		providers2 := []ProviderInfo{
			{Name: "provider1"},
			{Name: "provider2"},
			{Name: "provider3"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		// Select from first list
		selected1, err := policy.SelectProvider(context.Background(), request, providers1)
		require.NoError(t, err)
		assert.Contains(t, []string{"provider1", "provider2"}, selected1.Name)

		// Select from expanded list
		selected2, err := policy.SelectProvider(context.Background(), request, providers2)
		require.NoError(t, err)
		assert.Contains(t, []string{"provider1", "provider2", "provider3"}, selected2.Name)
	})
}

func TestFallbackPolicy(t *testing.T) {
	policy := &FallbackPolicy{}

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "fallback", policy.Name())
	})

	t.Run("returns first provider", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "first"},
			{Name: "second"},
			{Name: "third"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "first", selected.Name)
	})

	t.Run("consistent selection", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "always-first"},
			{Name: "never-selected"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		// Should always return the first
		for i := 0; i < 10; i++ {
			selected, err := policy.SelectProvider(context.Background(), request, providers)
			require.NoError(t, err)
			assert.Equal(t, "always-first", selected.Name)
		}
	})

	t.Run("single provider", func(t *testing.T) {
		providers := []ProviderInfo{
			{Name: "only"},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "only", selected.Name)
	})

	t.Run("empty provider list", func(t *testing.T) {
		providers := []ProviderInfo{}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		selected, err := policy.SelectProvider(context.Background(), request, providers)
		assert.Error(t, err)
		assert.Nil(t, selected)
	})
}

func TestPolicyIntegration(t *testing.T) {
	t.Run("all policies handle empty list", func(t *testing.T) {
		policies := []RoutingPolicy{
			&CostOptimizedPolicy{},
			&LatencyOptimizedPolicy{},
			&QualityOptimizedPolicy{},
			&LoadBalancedPolicy{},
			&FallbackPolicy{},
		}

		providers := []ProviderInfo{}
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		for _, policy := range policies {
			selected, err := policy.SelectProvider(context.Background(), request, providers)
			assert.Error(t, err, "Policy %s should error on empty list", policy.Name())
			assert.Nil(t, selected, "Policy %s should return nil on empty list", policy.Name())
		}
	})

	t.Run("all policies handle single provider", func(t *testing.T) {
		policies := []RoutingPolicy{
			&CostOptimizedPolicy{},
			&LatencyOptimizedPolicy{},
			&QualityOptimizedPolicy{},
			&LoadBalancedPolicy{},
			&FallbackPolicy{},
		}

		providers := []ProviderInfo{
			{
				Name:         "single",
				CostPerToken: 0.001,
				Latency:      100 * time.Millisecond,
				SuccessRate:  0.95,
				Priority:     1,
			},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		for _, policy := range policies {
			selected, err := policy.SelectProvider(context.Background(), request, providers)
			require.NoError(t, err, "Policy %s should not error with single provider", policy.Name())
			assert.Equal(t, "single", selected.Name, "Policy %s should select the only provider", policy.Name())
		}
	})

	t.Run("policies make different choices", func(t *testing.T) {
		providers := []ProviderInfo{
			{
				Name:         "cheap-slow",
				CostPerToken: 0.001,
				Latency:      1000 * time.Millisecond,
				SuccessRate:  0.85,
				Priority:     1,
			},
			{
				Name:         "expensive-fast",
				CostPerToken: 0.01,
				Latency:      100 * time.Millisecond,
				SuccessRate:  0.90,
				Priority:     2,
			},
			{
				Name:         "medium-reliable",
				CostPerToken: 0.005,
				Latency:      500 * time.Millisecond,
				SuccessRate:  0.99,
				Priority:     3,
			},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		// Cost optimized should choose cheap-slow
		costPolicy := &CostOptimizedPolicy{}
		selected, err := costPolicy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "cheap-slow", selected.Name)

		// Latency optimized should choose expensive-fast
		latencyPolicy := &LatencyOptimizedPolicy{}
		selected, err = latencyPolicy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "expensive-fast", selected.Name)

		// Quality optimized should choose medium-reliable
		qualityPolicy := &QualityOptimizedPolicy{}
		selected, err = qualityPolicy.SelectProvider(context.Background(), request, providers)
		require.NoError(t, err)
		assert.Equal(t, "medium-reliable", selected.Name)
	})
}

func TestPolicyConcurrency(t *testing.T) {
	t.Run("concurrent policy selection", func(t *testing.T) {
		// Create single instances to share across goroutines
		policies := []RoutingPolicy{
			&CostOptimizedPolicy{},
			&LatencyOptimizedPolicy{},
			&QualityOptimizedPolicy{},
			&LoadBalancedPolicy{}, // Single instance shared
			&FallbackPolicy{},
		}

		providers := []ProviderInfo{
			{Name: "provider1", CostPerToken: 0.001, Latency: 100 * time.Millisecond, SuccessRate: 0.95, Priority: 1},
			{Name: "provider2", CostPerToken: 0.002, Latency: 200 * time.Millisecond, SuccessRate: 0.90, Priority: 2},
			{Name: "provider3", CostPerToken: 0.003, Latency: 50 * time.Millisecond, SuccessRate: 0.99, Priority: 3},
		}

		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		var wg sync.WaitGroup
		errors := make(chan error, 500)

		// Run many concurrent selections
		for i := 0; i < 100; i++ {
			for _, policy := range policies {
				wg.Add(1)
				go func(p RoutingPolicy) {
					defer wg.Done()
					selected, err := p.SelectProvider(context.Background(), request, providers)
					if err != nil {
						errors <- err
					} else if selected == nil {
						errors <- assert.AnError
					}
				}(policy)
			}
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent policy selection error: %v", err)
		}
	})
}

func BenchmarkPolicies(b *testing.B) {
	providers := []ProviderInfo{
		{Name: "provider1", CostPerToken: 0.001, Latency: 100 * time.Millisecond, SuccessRate: 0.95, Priority: 1},
		{Name: "provider2", CostPerToken: 0.002, Latency: 200 * time.Millisecond, SuccessRate: 0.90, Priority: 2},
		{Name: "provider3", CostPerToken: 0.003, Latency: 50 * time.Millisecond, SuccessRate: 0.99, Priority: 3},
		{Name: "provider4", CostPerToken: 0.0015, Latency: 150 * time.Millisecond, SuccessRate: 0.92, Priority: 4},
		{Name: "provider5", CostPerToken: 0.0025, Latency: 75 * time.Millisecond, SuccessRate: 0.97, Priority: 5},
	}

	request := &RoutingRequest{
		Type: RequestTypeGeneration,
	}

	b.Run("CostOptimized", func(b *testing.B) {
		policy := &CostOptimizedPolicy{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			policy.SelectProvider(context.Background(), request, providers)
		}
	})

	b.Run("LatencyOptimized", func(b *testing.B) {
		policy := &LatencyOptimizedPolicy{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			policy.SelectProvider(context.Background(), request, providers)
		}
	})

	b.Run("QualityOptimized", func(b *testing.B) {
		policy := &QualityOptimizedPolicy{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			policy.SelectProvider(context.Background(), request, providers)
		}
	})

	b.Run("LoadBalanced", func(b *testing.B) {
		policy := &LoadBalancedPolicy{}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				policy.SelectProvider(context.Background(), request, providers)
			}
		})
	})

	b.Run("Fallback", func(b *testing.B) {
		policy := &FallbackPolicy{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			policy.SelectProvider(context.Background(), request, providers)
		}
	})
}