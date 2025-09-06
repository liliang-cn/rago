package router

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostTrackerCheckBudget(t *testing.T) {
	t.Run("no budgets", func(t *testing.T) {
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
		}

		// Should allow any cost when no budgets are set
		assert.True(t, ct.CheckBudget("provider1", 100.0))
		assert.True(t, ct.CheckBudget("provider2", 1000.0))
	})

	t.Run("within budget limit", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:      "daily",
					Provider:  "provider1",
					Limit:     100.0,
					Spent:     50.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		// Cost would bring total to 70, still under 100
		assert.True(t, ct.CheckBudget("provider1", 20.0))
	})

	t.Run("exceeding budget limit", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:      "daily",
					Provider:  "provider1",
					Limit:     100.0,
					Spent:     90.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		// Cost would bring total to 110, over 100
		assert.False(t, ct.CheckBudget("provider1", 20.0))
	})

	t.Run("global budget (no provider specified)", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"global": {
					Name:      "global",
					Provider:  "", // No specific provider
					Limit:     100.0,
					Spent:     80.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		// Global budget applies to all providers
		assert.False(t, ct.CheckBudget("provider1", 25.0))
		assert.False(t, ct.CheckBudget("provider2", 25.0))
		assert.True(t, ct.CheckBudget("provider3", 15.0))
	})

	t.Run("multiple budgets", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"global": {
					Name:      "global",
					Provider:  "",
					Limit:     1000.0,
					Spent:     500.0,
					Period:    BudgetPeriodMonthly,
					LastReset: time.Now(),
				},
				"provider-specific": {
					Name:      "provider-specific",
					Provider:  "provider1",
					Limit:     100.0,
					Spent:     95.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		// Should fail if ANY budget would be exceeded
		assert.False(t, ct.CheckBudget("provider1", 10.0)) // Exceeds provider-specific
		assert.True(t, ct.CheckBudget("provider2", 10.0))  // Within global budget
	})
}

func TestCostTrackerRecordCost(t *testing.T) {
	t.Run("new provider stats", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
			storage: storage,
		}

		ct.RecordCost("provider1", 10.0, 1000)

		stats := ct.usage["provider1"]
		require.NotNil(t, stats)
		assert.Equal(t, "provider1", stats.Provider)
		assert.Equal(t, 10.0, stats.TotalCost)
		assert.Equal(t, int64(1000), stats.TotalTokens)
		assert.Equal(t, int64(1), stats.TotalCalls)

		// Check storage was called
		savedStats := storage.usageStats["provider1"]
		assert.NotNil(t, savedStats)
		assert.Equal(t, 10.0, savedStats.TotalCost)
	})

	t.Run("update existing stats", func(t *testing.T) {
		ct := &CostTracker{
			usage: map[string]*UsageStats{
				"provider1": {
					Provider:    "provider1",
					TotalCost:   50.0,
					TotalTokens: 5000,
					TotalCalls:  5,
					LastReset:   time.Now().Add(-time.Hour),
				},
			},
			budgets: make(map[string]*Budget),
		}

		ct.RecordCost("provider1", 10.0, 1000)

		stats := ct.usage["provider1"]
		assert.Equal(t, 60.0, stats.TotalCost)
		assert.Equal(t, int64(6000), stats.TotalTokens)
		assert.Equal(t, int64(6), stats.TotalCalls)
	})

	t.Run("update budget spending", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:         "daily",
					Provider:     "provider1",
					Limit:        100.0,
					Spent:        50.0,
					Period:       BudgetPeriodDaily,
					LastReset:    time.Now(),
					AlertPercent: 80.0,
				},
			},
			alerts: []CostAlert{},
		}

		ct.RecordCost("provider1", 10.0, 1000)

		budget := ct.budgets["daily"]
		assert.Equal(t, 60.0, budget.Spent)
		assert.Len(t, ct.alerts, 0) // No alert yet (60% < 80%)
	})

	t.Run("concurrent cost recording", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
			storage: storage,
		}

		var wg sync.WaitGroup
		numGoroutines := 100
		costPerCall := 1.0
		tokensPerCall := int64(100)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ct.RecordCost("provider1", costPerCall, tokensPerCall)
			}()
		}

		wg.Wait()

		stats := ct.usage["provider1"]
		assert.Equal(t, float64(numGoroutines)*costPerCall, stats.TotalCost)
		assert.Equal(t, int64(numGoroutines)*tokensPerCall, stats.TotalTokens)
		assert.Equal(t, int64(numGoroutines), stats.TotalCalls)
	})
}

func TestCostTrackerBudgetAlerts(t *testing.T) {
	t.Run("warning alert at threshold", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:         "daily",
					Provider:     "provider1",
					Limit:        100.0,
					Spent:        75.0,
					Period:       BudgetPeriodDaily,
					LastReset:    time.Now(),
					AlertPercent: 80.0,
				},
			},
			alerts:  []CostAlert{},
			storage: storage,
		}

		// This should trigger a warning (85% > 80%)
		ct.RecordCost("provider1", 10.0, 1000)

		assert.Len(t, ct.alerts, 1)
		alert := ct.alerts[0]
		assert.Equal(t, AlertTypeBudgetWarning, alert.Type)
		assert.Contains(t, alert.Message, "85%")
		assert.NotNil(t, alert.Budget)

		// Check storage
		assert.Len(t, storage.alerts, 1)
	})

	t.Run("exceeded alert", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:         "daily",
					Provider:     "provider1",
					Limit:        100.0,
					Spent:        95.0,
					Period:       BudgetPeriodDaily,
					LastReset:    time.Now(),
					AlertPercent: 80.0,
				},
			},
			alerts:  []CostAlert{},
			storage: storage,
		}

		// This should trigger exceeded alert (105% > 100%)
		ct.RecordCost("provider1", 10.0, 1000)

		assert.Len(t, ct.alerts, 1)
		alert := ct.alerts[0]
		assert.Equal(t, AlertTypeBudgetExceeded, alert.Type)
		assert.Contains(t, alert.Message, "exceeded")
		assert.Equal(t, "daily", alert.Budget.Name)
	})

	t.Run("multiple alerts", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"global": {
					Name:         "global",
					Provider:     "",
					Limit:        200.0,
					Spent:        190.0,
					Period:       BudgetPeriodDaily,
					AlertPercent: 90.0,
				},
				"provider-specific": {
					Name:         "provider-specific",
					Provider:     "provider1",
					Limit:        100.0,
					Spent:        95.0,
					Period:       BudgetPeriodDaily,
					AlertPercent: 80.0,
				},
			},
			alerts: []CostAlert{},
		}

		// Both budgets should trigger alerts
		ct.RecordCost("provider1", 10.0, 1000)

		assert.Len(t, ct.alerts, 2)
		
		alertTypes := make(map[AlertType]bool)
		for _, alert := range ct.alerts {
			alertTypes[alert.Type] = true
		}
		assert.True(t, alertTypes[AlertTypeBudgetExceeded])
	})
}

func TestCostTrackerBudgetReset(t *testing.T) {
	t.Run("hourly budget reset", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"hourly": {
					Name:      "hourly",
					Provider:  "provider1",
					Limit:     10.0,
					Spent:     9.0,
					Period:    BudgetPeriodHourly,
					LastReset: time.Now().Add(-61 * time.Minute),
				},
			},
		}

		ct.UpdateCosts()

		budget := ct.budgets["hourly"]
		assert.Equal(t, 0.0, budget.Spent)
		assert.WithinDuration(t, time.Now(), budget.LastReset, time.Second)
	})

	t.Run("daily budget reset", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:      "daily",
					Provider:  "provider1",
					Limit:     100.0,
					Spent:     90.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now().Add(-25 * time.Hour),
				},
			},
		}

		ct.UpdateCosts()

		budget := ct.budgets["daily"]
		assert.Equal(t, 0.0, budget.Spent)
		assert.WithinDuration(t, time.Now(), budget.LastReset, time.Second)
	})

	t.Run("weekly budget reset", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"weekly": {
					Name:      "weekly",
					Provider:  "provider1",
					Limit:     1000.0,
					Spent:     900.0,
					Period:    BudgetPeriodWeekly,
					LastReset: time.Now().Add(-8 * 24 * time.Hour),
				},
			},
		}

		ct.UpdateCosts()

		budget := ct.budgets["weekly"]
		assert.Equal(t, 0.0, budget.Spent)
		assert.WithinDuration(t, time.Now(), budget.LastReset, time.Second)
	})

	t.Run("monthly budget reset", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"monthly": {
					Name:      "monthly",
					Provider:  "provider1",
					Limit:     10000.0,
					Spent:     9000.0,
					Period:    BudgetPeriodMonthly,
					LastReset: time.Now().Add(-31 * 24 * time.Hour),
				},
			},
		}

		ct.UpdateCosts()

		budget := ct.budgets["monthly"]
		assert.Equal(t, 0.0, budget.Spent)
		assert.WithinDuration(t, time.Now(), budget.LastReset, time.Second)
	})

	t.Run("no reset when not needed", func(t *testing.T) {
		originalSpent := 50.0
		originalReset := time.Now().Add(-10 * time.Minute)
		
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:      "daily",
					Provider:  "provider1",
					Limit:     100.0,
					Spent:     originalSpent,
					Period:    BudgetPeriodDaily,
					LastReset: originalReset,
				},
			},
		}

		ct.UpdateCosts()

		budget := ct.budgets["daily"]
		assert.Equal(t, originalSpent, budget.Spent)
		assert.Equal(t, originalReset, budget.LastReset)
	})
}

func TestCostTrackerStorageIntegration(t *testing.T) {
	t.Run("save usage stats on record", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
			storage: storage,
		}

		ct.RecordCost("provider1", 10.0, 1000)

		savedStats, err := storage.LoadUsageStats("provider1")
		require.NoError(t, err)
		assert.Equal(t, 10.0, savedStats.TotalCost)
		assert.Equal(t, int64(1000), savedStats.TotalTokens)
	})

	t.Run("handle storage errors", func(t *testing.T) {
		storage := newMockCostStorage()
		storage.saveError = fmt.Errorf("storage error")
		
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
			storage: storage,
		}

		// Should not panic even with storage errors
		ct.RecordCost("provider1", 10.0, 1000)
		
		// Local stats should still be updated
		assert.Equal(t, 10.0, ct.usage["provider1"].TotalCost)
	})

	t.Run("save alerts on trigger", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:         "daily",
					Provider:     "provider1",
					Limit:        100.0,
					Spent:        95.0,
					Period:       BudgetPeriodDaily,
					LastReset:    time.Now(),
					AlertPercent: 80.0,
				},
			},
			alerts:  []CostAlert{},
			storage: storage,
		}

		ct.RecordCost("provider1", 10.0, 1000)

		assert.Len(t, storage.alerts, 1)
		assert.Equal(t, AlertTypeBudgetExceeded, storage.alerts[0].Type)
	})
}

func TestCostTrackerEdgeCases(t *testing.T) {
	t.Run("zero budget limit", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"zero": {
					Name:      "zero",
					Provider:  "provider1",
					Limit:     0.0,
					Spent:     0.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		// Any positive cost exceeds zero budget
		assert.False(t, ct.CheckBudget("provider1", 0.01))
		assert.False(t, ct.CheckBudget("provider1", 1.0))
		
		// Zero cost is allowed (no change to spending)
		assert.True(t, ct.CheckBudget("provider1", 0.0))
	})

	t.Run("negative cost", func(t *testing.T) {
		ct := &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
		}

		// Negative cost (refund?) should still be recorded
		ct.RecordCost("provider1", -10.0, 0)
		
		assert.Equal(t, -10.0, ct.usage["provider1"].TotalCost)
	})

	t.Run("very large numbers", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"large": {
					Name:      "large",
					Provider:  "provider1",
					Limit:     1e10,
					Spent:     0.0,
					Period:    BudgetPeriodMonthly,
					LastReset: time.Now(),
				},
			},
		}

		assert.True(t, ct.CheckBudget("provider1", 1e9))
		ct.RecordCost("provider1", 1e9, 1e12)
		
		assert.Equal(t, 1e9, ct.usage["provider1"].TotalCost)
		assert.Equal(t, int64(1e12), ct.usage["provider1"].TotalTokens)
	})
}

func TestCostTrackerConcurrency(t *testing.T) {
	t.Run("concurrent budget checks", func(t *testing.T) {
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"daily": {
					Name:      "daily",
					Provider:  "",
					Limit:     1000.0,
					Spent:     500.0,
					Period:    BudgetPeriodDaily,
					LastReset: time.Now(),
				},
			},
		}

		var wg sync.WaitGroup
		results := make([]bool, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				provider := fmt.Sprintf("provider%d", idx%5)
				results[idx] = ct.CheckBudget(provider, 10.0)
			}(i)
		}

		wg.Wait()

		// All checks should succeed (total would be 1000, at limit)
		for _, result := range results {
			assert.True(t, result)
		}
	})

	t.Run("concurrent cost recording with budget updates", func(t *testing.T) {
		storage := newMockCostStorage()
		ct := &CostTracker{
			usage: make(map[string]*UsageStats),
			budgets: map[string]*Budget{
				"concurrent": {
					Name:         "concurrent",
					Provider:     "",
					Limit:        100.0,
					Spent:        0.0,
					Period:       BudgetPeriodDaily,
					LastReset:    time.Now(),
					AlertPercent: 50.0,
				},
			},
			alerts:  []CostAlert{},
			storage: storage,
		}

		var wg sync.WaitGroup
		numGoroutines := 20

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				provider := fmt.Sprintf("provider%d", id%3)
				ct.RecordCost(provider, 5.0, 500)
			}(i)
		}

		wg.Wait()

		// Total cost should be 100 (20 * 5.0)
		totalCost := 0.0
		for _, stats := range ct.usage {
			totalCost += stats.TotalCost
		}
		assert.Equal(t, 100.0, totalCost)

		// Budget should be at limit
		assert.Equal(t, 100.0, ct.budgets["concurrent"].Spent)

		// Should have triggered alert(s)
		assert.Greater(t, len(ct.alerts), 0)
	})
}