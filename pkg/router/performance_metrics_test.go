package router

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceMetricsInit(t *testing.T) {
	pm := &PerformanceMetrics{
		metrics: make(map[string]*ProviderMetrics),
		window:  time.Hour,
	}

	pm.initProvider("provider1")

	metrics, exists := pm.metrics["provider1"]
	require.True(t, exists)
	assert.Equal(t, "provider1", metrics.Provider)
	assert.Equal(t, int64(0), metrics.RequestCount)
	assert.Equal(t, int64(0), metrics.SuccessCount)
	assert.Equal(t, int64(0), metrics.FailureCount)
	assert.NotNil(t, metrics.LatencySamples)
	assert.Len(t, metrics.LatencySamples, 0)
	assert.Equal(t, 1000, cap(metrics.LatencySamples))
}

func TestPerformanceMetricsRecordRequest(t *testing.T) {
	pm := &PerformanceMetrics{
		metrics: make(map[string]*ProviderMetrics),
		window:  time.Hour,
	}

	pm.initProvider("provider1")

	// Record multiple requests
	for i := 0; i < 10; i++ {
		pm.RecordRequest("provider1")
	}

	metrics := pm.metrics["provider1"]
	assert.Equal(t, int64(10), metrics.RequestCount)
}

func TestPerformanceMetricsRecordSuccess(t *testing.T) {
	t.Run("basic success recording", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Record successes with different latencies
		latencies := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			150 * time.Millisecond,
		}

		for _, latency := range latencies {
			pm.RecordSuccess("provider1", latency)
		}

		metrics := pm.metrics["provider1"]
		assert.Equal(t, int64(3), metrics.SuccessCount)
		assert.Equal(t, 450*time.Millisecond, metrics.TotalLatency)
		assert.Equal(t, 150*time.Millisecond, metrics.AverageLatency)
		assert.Len(t, metrics.LatencySamples, 3)
	})

	t.Run("latency samples window", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Record more than 1000 samples
		for i := 0; i < 1100; i++ {
			pm.RecordSuccess("provider1", time.Duration(i)*time.Millisecond)
		}

		metrics := pm.metrics["provider1"]
		assert.Equal(t, int64(1100), metrics.SuccessCount)
		assert.Len(t, metrics.LatencySamples, 1000) // Should be limited to 1000

		// Verify we kept the most recent samples
		expectedFirstSample := 100 * time.Millisecond
		assert.Equal(t, expectedFirstSample, metrics.LatencySamples[0])
	})

	t.Run("non-existent provider", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		// Should not panic for non-existent provider
		pm.RecordSuccess("non-existent", 100*time.Millisecond)

		// Provider should not be created
		_, exists := pm.metrics["non-existent"]
		assert.False(t, exists)
	})
}

func TestPerformanceMetricsRecordFailure(t *testing.T) {
	pm := &PerformanceMetrics{
		metrics: make(map[string]*ProviderMetrics),
		window:  time.Hour,
	}

	pm.initProvider("provider1")

	// Record multiple failures
	for i := 0; i < 5; i++ {
		pm.RecordFailure("provider1")
	}

	metrics := pm.metrics["provider1"]
	assert.Equal(t, int64(5), metrics.FailureCount)
	assert.Equal(t, int64(0), metrics.SuccessCount)
}

func TestPerformanceMetricsLatencyCalculations(t *testing.T) {
	t.Run("average latency", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Record various latencies
		latencies := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
			400 * time.Millisecond,
		}

		for _, latency := range latencies {
			pm.RecordSuccess("provider1", latency)
		}

		metrics := pm.metrics["provider1"]
		expectedAvg := 250 * time.Millisecond
		assert.Equal(t, expectedAvg, metrics.AverageLatency)
	})

	t.Run("P95 latency", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Create 100 samples with known distribution
		for i := 1; i <= 100; i++ {
			pm.RecordSuccess("provider1", time.Duration(i)*time.Millisecond)
		}

		metrics := pm.metrics["provider1"]
		// P95 of 1-100 should be 95
		assert.Equal(t, 95*time.Millisecond, metrics.P95Latency)
	})

	t.Run("P99 latency", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Create 100 samples with known distribution
		for i := 1; i <= 100; i++ {
			pm.RecordSuccess("provider1", time.Duration(i)*time.Millisecond)
		}

		metrics := pm.metrics["provider1"]
		// P99 of 1-100 should be 99
		assert.Equal(t, 99*time.Millisecond, metrics.P99Latency)
	})

	t.Run("percentiles with few samples", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Only 5 samples
		samples := []time.Duration{10, 20, 30, 40, 50}
		for _, s := range samples {
			pm.RecordSuccess("provider1", s*time.Millisecond)
		}

		metrics := pm.metrics["provider1"]
		// P95 of 5 samples (index 4.75 -> 4)
		assert.Equal(t, 50*time.Millisecond, metrics.P95Latency)
		// P99 of 5 samples (index 4.95 -> 4)
		assert.Equal(t, 50*time.Millisecond, metrics.P99Latency)
	})

	t.Run("percentiles with single sample", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		pm.RecordSuccess("provider1", 100*time.Millisecond)

		metrics := pm.metrics["provider1"]
		assert.Equal(t, 100*time.Millisecond, metrics.P95Latency)
		assert.Equal(t, 100*time.Millisecond, metrics.P99Latency)
	})

	t.Run("empty samples", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		metrics := &ProviderMetrics{
			Provider:       "provider1",
			LatencySamples: []time.Duration{},
		}
		
		pm.updateLatencyStats(metrics)

		// Should not panic and values should remain zero
		assert.Equal(t, time.Duration(0), metrics.P95Latency)
		assert.Equal(t, time.Duration(0), metrics.P99Latency)
	})
}

func TestPerformanceMetricsConcurrency(t *testing.T) {
	t.Run("concurrent requests recording", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pm.RecordRequest("provider1")
			}()
		}

		wg.Wait()

		metrics := pm.metrics["provider1"]
		assert.Equal(t, int64(numGoroutines), metrics.RequestCount)
	})

	t.Run("concurrent success recording", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				latency := time.Duration(id) * time.Millisecond
				pm.RecordSuccess("provider1", latency)
			}(i)
		}

		wg.Wait()

		metrics := pm.metrics["provider1"]
		assert.Equal(t, int64(numGoroutines), metrics.SuccessCount)
		assert.Len(t, metrics.LatencySamples, numGoroutines)
	})

	t.Run("concurrent mixed operations", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		providers := []string{"provider1", "provider2", "provider3"}
		for _, p := range providers {
			pm.initProvider(p)
		}

		var wg sync.WaitGroup
		numOperations := 300

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				provider := providers[id%len(providers)]
				
				switch id % 3 {
				case 0:
					pm.RecordRequest(provider)
				case 1:
					pm.RecordSuccess(provider, time.Duration(id)*time.Millisecond)
				case 2:
					pm.RecordFailure(provider)
				}
			}(i)
		}

		wg.Wait()

		// Verify all providers have data
		for _, provider := range providers {
			metrics := pm.metrics[provider]
			assert.NotNil(t, metrics)
			totalOps := metrics.RequestCount + metrics.SuccessCount + metrics.FailureCount
			assert.Greater(t, totalOps, int64(0))
		}
	})

	t.Run("concurrent latency updates", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		var wg sync.WaitGroup
		
		// Many goroutines recording success simultaneously
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				pm.RecordSuccess("provider1", time.Duration(id%100)*time.Millisecond)
			}(i)
		}

		wg.Wait()

		metrics := pm.metrics["provider1"]
		assert.Equal(t, int64(500), metrics.SuccessCount)
		
		// Check percentiles are calculated correctly
		assert.Greater(t, metrics.P95Latency, time.Duration(0))
		assert.Greater(t, metrics.P99Latency, time.Duration(0))
		assert.GreaterOrEqual(t, metrics.P99Latency, metrics.P95Latency)
	})
}

func TestPerformanceMetricsEdgeCases(t *testing.T) {
	t.Run("zero latency", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")
		pm.RecordSuccess("provider1", 0)

		metrics := pm.metrics["provider1"]
		assert.Equal(t, time.Duration(0), metrics.TotalLatency)
		assert.Equal(t, time.Duration(0), metrics.AverageLatency)
	})

	t.Run("very large latency", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")
		
		largeLatency := 24 * time.Hour
		pm.RecordSuccess("provider1", largeLatency)

		metrics := pm.metrics["provider1"]
		assert.Equal(t, largeLatency, metrics.TotalLatency)
		assert.Equal(t, largeLatency, metrics.AverageLatency)
	})

	t.Run("negative latency index bounds", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		// Test with empty samples
		metrics := &ProviderMetrics{
			Provider:       "provider1",
			LatencySamples: []time.Duration{},
		}

		pm.updateLatencyStats(metrics)

		// Should not panic
		assert.Equal(t, time.Duration(0), metrics.P95Latency)
		assert.Equal(t, time.Duration(0), metrics.P99Latency)
	})

	t.Run("percentile calculation edge cases", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		testCases := []struct {
			name     string
			samples  []time.Duration
			expected struct {
				p95 time.Duration
				p99 time.Duration
			}
		}{
			{
				name:    "two samples",
				samples: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
				expected: struct {
					p95 time.Duration
					p99 time.Duration
				}{
					p95: 20 * time.Millisecond,
					p99: 20 * time.Millisecond,
				},
			},
			{
				name:    "three samples",
				samples: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
				expected: struct {
					p95 time.Duration
					p99 time.Duration
				}{
					p95: 30 * time.Millisecond,
					p99: 30 * time.Millisecond,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				metrics := &ProviderMetrics{
					Provider:       "test",
					LatencySamples: tc.samples,
					RequestCount:   int64(len(tc.samples)),
				}

				pm.updateLatencyStats(metrics)

				assert.Equal(t, tc.expected.p95, metrics.P95Latency)
				assert.Equal(t, tc.expected.p99, metrics.P99Latency)
			})
		}
	})

	t.Run("integer overflow protection", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		pm.initProvider("provider1")

		// Test with max int64 values
		pm.metrics["provider1"].RequestCount = math.MaxInt64 - 1
		pm.RecordRequest("provider1")

		// Should handle overflow gracefully
		assert.Equal(t, int64(math.MaxInt64), pm.metrics["provider1"].RequestCount)
	})
}

func TestPerformanceMetricsIntegration(t *testing.T) {
	t.Run("realistic usage pattern", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		providers := []string{"fast", "medium", "slow"}
		latencyRanges := map[string]struct{ min, max time.Duration }{
			"fast":   {10 * time.Millisecond, 50 * time.Millisecond},
			"medium": {50 * time.Millisecond, 200 * time.Millisecond},
			"slow":   {200 * time.Millisecond, 1000 * time.Millisecond},
		}

		// Initialize providers
		for _, p := range providers {
			pm.initProvider(p)
		}

		// Simulate realistic traffic
		for i := 0; i < 1000; i++ {
			provider := providers[i%len(providers)]
			pm.RecordRequest(provider)

			// 90% success rate
			if i%10 != 0 {
				latencyRange := latencyRanges[provider]
				latency := latencyRange.min + time.Duration(i%100)*time.Millisecond
				if latency > latencyRange.max {
					latency = latencyRange.max
				}
				pm.RecordSuccess(provider, latency)
			} else {
				pm.RecordFailure(provider)
			}
		}

		// Verify metrics make sense
		for _, provider := range providers {
			metrics := pm.metrics[provider]
			
			// Should have roughly equal distribution
			assert.Greater(t, metrics.RequestCount, int64(300))
			assert.Less(t, metrics.RequestCount, int64(400))
			
			// ~90% success rate
			successRate := float64(metrics.SuccessCount) / float64(metrics.RequestCount)
			assert.Greater(t, successRate, 0.85)
			assert.Less(t, successRate, 0.95)
			
			// Latency should be in expected range
			latencyRange := latencyRanges[provider]
			assert.GreaterOrEqual(t, metrics.AverageLatency, latencyRange.min)
			assert.LessOrEqual(t, metrics.AverageLatency, latencyRange.max)
			
			// P99 should be higher than P95
			assert.GreaterOrEqual(t, metrics.P99Latency, metrics.P95Latency)
		}
	})
}

func BenchmarkPerformanceMetrics(b *testing.B) {
	b.Run("RecordSuccess", func(b *testing.B) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}
		pm.initProvider("provider1")

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pm.RecordSuccess("provider1", 100*time.Millisecond)
			}
		})
	})

	b.Run("UpdateLatencyStats", func(b *testing.B) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		// Create metrics with many samples
		metrics := &ProviderMetrics{
			Provider:       "provider1",
			LatencySamples: make([]time.Duration, 1000),
		}
		for i := 0; i < 1000; i++ {
			metrics.LatencySamples[i] = time.Duration(i) * time.Millisecond
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.updateLatencyStats(metrics)
		}
	})
}

func TestPerformanceMetricsRaceConditions(t *testing.T) {
	t.Run("race between init and record", func(t *testing.T) {
		pm := &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		}

		var wg sync.WaitGroup

		// Concurrent initialization and recording
		for i := 0; i < 100; i++ {
			wg.Add(2)
			provider := fmt.Sprintf("provider%d", i%10)
			
			go func(p string) {
				defer wg.Done()
				pm.initProvider(p)
			}(provider)
			
			go func(p string) {
				defer wg.Done()
				pm.RecordSuccess(p, 100*time.Millisecond)
			}(provider)
		}

		wg.Wait()

		// Should not panic or have data corruption
		for i := 0; i < 10; i++ {
			provider := fmt.Sprintf("provider%d", i)
			metrics, exists := pm.metrics[provider]
			if exists {
				assert.NotNil(t, metrics)
				assert.NotNil(t, metrics.LatencySamples)
			}
		}
	})
}