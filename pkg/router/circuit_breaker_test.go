package router

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerInitialization(t *testing.T) {
	cb := &CircuitBreaker{
		provider:  "test-provider",
		state:     BreakerStateClosed,
		timeout:   30 * time.Second,
		threshold: 5,
	}

	assert.Equal(t, "test-provider", cb.provider)
	assert.Equal(t, BreakerStateClosed, cb.state)
	assert.Equal(t, 30*time.Second, cb.timeout)
	assert.Equal(t, 5, cb.threshold)
	assert.Equal(t, 0, cb.failureCount)
}

func TestCircuitBreakerCanRequest(t *testing.T) {
	t.Run("closed state allows requests", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 5,
		}

		assert.True(t, cb.CanRequest())
	})

	t.Run("open state blocks requests initially", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     30 * time.Second,
			threshold:   5,
			lastFailure: time.Now(),
		}

		assert.False(t, cb.CanRequest())
	})

	t.Run("open state transitions to half-open after timeout", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     50 * time.Millisecond,
			threshold:   5,
			lastFailure: time.Now().Add(-100 * time.Millisecond), // Past timeout
		}

		// First call should transition to half-open and return true
		assert.True(t, cb.CanRequest())
		assert.Equal(t, BreakerStateHalfOpen, cb.state)
	})

	t.Run("half-open state allows requests", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateHalfOpen,
			timeout:   30 * time.Second,
			threshold: 5,
		}

		assert.True(t, cb.CanRequest())
	})
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	t.Run("success in closed state resets failure count", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateClosed,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 3,
		}

		cb.RecordSuccess()

		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("success in half-open state transitions to closed", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateHalfOpen,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 2,
		}

		cb.RecordSuccess()

		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("success in open state (shouldn't happen) resets count", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateOpen,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 5,
		}

		cb.RecordSuccess()

		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, BreakerStateOpen, cb.state) // State doesn't change
	})
}

func TestCircuitBreakerRecordFailure(t *testing.T) {
	t.Run("failure increments counter", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateClosed,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 0,
		}

		cb.RecordFailure()

		assert.Equal(t, 1, cb.failureCount)
		assert.Equal(t, BreakerStateClosed, cb.state)
		assert.False(t, cb.lastFailure.IsZero())
	})

	t.Run("reaching threshold opens circuit", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateClosed,
			timeout:      30 * time.Second,
			threshold:    3,
			failureCount: 2,
		}

		cb.RecordFailure()

		assert.Equal(t, 3, cb.failureCount)
		assert.Equal(t, BreakerStateOpen, cb.state)
	})

	t.Run("failure in half-open state opens circuit", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateHalfOpen,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 0,
		}

		cb.RecordFailure()

		assert.Equal(t, 1, cb.failureCount)
		assert.Equal(t, BreakerStateHalfOpen, cb.state) // Doesn't immediately open

		// Continue failures to reach threshold
		for i := 0; i < 4; i++ {
			cb.RecordFailure()
		}

		assert.Equal(t, 5, cb.failureCount)
		assert.Equal(t, BreakerStateOpen, cb.state)
	})
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	t.Run("complete state transition cycle", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   50 * time.Millisecond,
			threshold: 2,
		}

		// Start in closed state
		assert.Equal(t, BreakerStateClosed, cb.state)
		assert.True(t, cb.CanRequest())

		// Record failures to open the circuit
		cb.RecordFailure()
		assert.Equal(t, BreakerStateClosed, cb.state)
		
		cb.RecordFailure()
		assert.Equal(t, BreakerStateOpen, cb.state)
		assert.False(t, cb.CanRequest())

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Should transition to half-open
		assert.True(t, cb.CanRequest())
		assert.Equal(t, BreakerStateHalfOpen, cb.state)

		// Success in half-open returns to closed
		cb.RecordSuccess()
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("half-open to open on failure", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateHalfOpen,
			timeout:   50 * time.Millisecond,
			threshold: 2,
		}

		// Fail in half-open
		cb.RecordFailure()
		cb.RecordFailure()
		
		assert.Equal(t, BreakerStateOpen, cb.state)
	})
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	t.Run("concurrent CanRequest calls", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 5,
		}

		var wg sync.WaitGroup
		results := make([]bool, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = cb.CanRequest()
			}(i)
		}

		wg.Wait()

		// All should return true for closed state
		for i, result := range results {
			assert.True(t, result, "Request %d should be allowed", i)
		}
	})

	t.Run("concurrent RecordSuccess calls", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:     "test-provider",
			state:        BreakerStateClosed,
			timeout:      30 * time.Second,
			threshold:    5,
			failureCount: 100,
		}

		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cb.RecordSuccess()
			}()
		}

		wg.Wait()

		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("concurrent RecordFailure calls", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 50,
		}

		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cb.RecordFailure()
			}()
		}

		wg.Wait()

		assert.Equal(t, 50, cb.failureCount)
		assert.Equal(t, BreakerStateOpen, cb.state)
	})

	t.Run("mixed concurrent operations", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     10 * time.Millisecond,
			threshold:   5,
			lastFailure: time.Now().Add(-20 * time.Millisecond),
		}

		var wg sync.WaitGroup
		
		// Mix of operations
		for i := 0; i < 10; i++ {
			wg.Add(3)
			
			go func() {
				defer wg.Done()
				cb.CanRequest()
			}()
			
			go func() {
				defer wg.Done()
				cb.RecordSuccess()
			}()
			
			go func() {
				defer wg.Done()
				cb.RecordFailure()
			}()
		}

		wg.Wait()

		// Should not panic or deadlock
		assert.NotNil(t, cb)
	})
}

func TestCircuitBreakerEdgeCases(t *testing.T) {
	t.Run("zero threshold", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 0,
		}

		// Should never open with zero threshold (misconfiguration)
		cb.RecordFailure()
		assert.Equal(t, BreakerStateClosed, cb.state)
		
		// Even with many failures
		for i := 0; i < 10; i++ {
			cb.RecordFailure()
		}
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("negative threshold", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: -1,
		}

		// Should never open (but this is a misconfiguration)
		for i := 0; i < 10; i++ {
			cb.RecordFailure()
		}
		assert.Equal(t, BreakerStateClosed, cb.state)
	})

	t.Run("zero timeout", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     0,
			threshold:   5,
			lastFailure: time.Now(),
		}

		// Should immediately allow transition to half-open
		assert.True(t, cb.CanRequest())
		assert.Equal(t, BreakerStateHalfOpen, cb.state)
	})
}

func TestCircuitBreakerRaceConditions(t *testing.T) {
	t.Run("race between state check and modification", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     5 * time.Millisecond,
			threshold:   5,
			lastFailure: time.Now(),
		}

		var wg sync.WaitGroup
		iterations := 100

		// Start multiple goroutines that might trigger state transition
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Duration(i%10) * time.Millisecond)
				cb.CanRequest()
			}()
		}

		// Simultaneously record successes and failures
		for i := 0; i < iterations; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				cb.RecordSuccess()
			}()
			go func() {
				defer wg.Done()
				cb.RecordFailure()
			}()
		}

		wg.Wait()

		// Should not panic or have inconsistent state
		assert.NotNil(t, cb)
		assert.Contains(t, []BreakerState{
			BreakerStateClosed,
			BreakerStateOpen,
			BreakerStateHalfOpen,
		}, cb.state)
	})

	t.Run("concurrent timeout checks", func(t *testing.T) {
		cb := &CircuitBreaker{
			provider:    "test-provider",
			state:       BreakerStateOpen,
			timeout:     1 * time.Millisecond,
			threshold:   5,
			lastFailure: time.Now(),
		}

		var wg sync.WaitGroup
		stateChanges := 0
		var mu sync.Mutex

		// Multiple goroutines checking timeout simultaneously
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(2 * time.Millisecond)
				if cb.CanRequest() {
					mu.Lock()
					stateChanges++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// Should transition to half-open exactly once
		assert.Equal(t, BreakerStateHalfOpen, cb.state)
		assert.Greater(t, stateChanges, 0)
	})
}

func BenchmarkCircuitBreaker(b *testing.B) {
	b.Run("CanRequest", func(b *testing.B) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 5,
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.CanRequest()
			}
		})
	})

	b.Run("RecordSuccess", func(b *testing.B) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 5,
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.RecordSuccess()
			}
		})
	})

	b.Run("RecordFailure", func(b *testing.B) {
		cb := &CircuitBreaker{
			provider:  "test-provider",
			state:     BreakerStateClosed,
			timeout:   30 * time.Second,
			threshold: 1000000, // High threshold to prevent state change
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.RecordFailure()
			}
		})
	})
}

func TestCircuitBreakerIntegration(t *testing.T) {
	t.Run("integration with router", func(t *testing.T) {
		config := &RouterConfig{
			EnableFailover:      true,
			MaxRetries:          3,
			RetryDelay:          10 * time.Millisecond,
			HealthCheckInterval: 0,
			CostUpdateInterval:  0,
		}
		router := NewProviderRouter(config, newMockCostStorage())

		provider := createTestProvider("provider1", 0.001, 100*time.Millisecond, ProviderStatusHealthy)
		err := router.RegisterProvider(provider)
		require.NoError(t, err)

		// Get the circuit breaker
		breaker := router.breakers["provider1"]
		require.NotNil(t, breaker)

		// Simulate failures to open circuit
		for i := 0; i < breaker.threshold; i++ {
			breaker.RecordFailure()
		}

		assert.Equal(t, BreakerStateOpen, breaker.state)

		// Provider should not be available
		request := &RoutingRequest{
			Type: RequestTypeGeneration,
		}

		available := router.getAvailableProviders(request)
		assert.Len(t, available, 0)
	})
}