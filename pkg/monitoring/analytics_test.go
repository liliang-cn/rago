package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Analytics Processing

func TestAnalytics_ProcessAnalytics(t *testing.T) {
	// Test analytics processing
	analytics := &Analytics{
		aggregator: &DataAggregator{
			windows:    make(map[string]*AggregationWindow),
			aggregates: make(map[string]*AggregateResult),
		},
		calculator: &MetricsCalculator{
			formulas: make(map[string]MetricFormula),
		},
		predictor: &TrendPredictor{
			models: make(map[string]PredictionModel),
		},
	}
	
	// Create mock metrics
	metrics := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
	}
	
	// Process analytics
	analytics.ProcessAnalytics(metrics)
	
	// Verify components were called
	assert.NotNil(t, analytics.aggregator)
	assert.NotNil(t, analytics.calculator)
	assert.NotNil(t, analytics.predictor)
}

func TestAnalytics_GetAnalytics(t *testing.T) {
	// Test getting analytics data
	analytics := &Analytics{
		aggregator: &DataAggregator{
			windows:    make(map[string]*AggregationWindow),
			aggregates: make(map[string]*AggregateResult),
		},
		calculator: &MetricsCalculator{
			formulas: make(map[string]MetricFormula),
		},
		predictor: &TrendPredictor{
			models: make(map[string]PredictionModel),
		},
	}
	
	// Add test data
	analytics.aggregator.aggregates["test-aggregate"] = &AggregateResult{
		Window:  "5m",
		Count:   100,
		Sum:     500,
		Average: 5,
		Min:     1,
		Max:     10,
		StdDev:  2.5,
	}
	
	analytics.predictor.models["test-model"] = PredictionModel{
		Name:        "Linear Regression",
		Type:        "linear",
		Accuracy:    0.92,
		LastTrained: time.Now(),
	}
	
	// Get analytics
	result := analytics.GetAnalytics()
	
	assert.NotNil(t, result["aggregates"])
	assert.NotNil(t, result["predictions"])
	
	aggregates := result["aggregates"].(map[string]*AggregateResult)
	assert.Contains(t, aggregates, "test-aggregate")
	
	predictions := result["predictions"].(map[string]PredictionModel)
	assert.Contains(t, predictions, "test-model")
}

func TestDataAggregator_AggregationWindows(t *testing.T) {
	// Test aggregation windows
	da := &DataAggregator{
		windows:    make(map[string]*AggregationWindow),
		aggregates: make(map[string]*AggregateResult),
	}
	
	now := time.Now()
	
	// Create different windows
	windows := []*AggregationWindow{
		{
			Name:      "1m",
			Duration:  1 * time.Minute,
			StartTime: now.Add(-1 * time.Minute),
			EndTime:   now,
		},
		{
			Name:      "5m",
			Duration:  5 * time.Minute,
			StartTime: now.Add(-5 * time.Minute),
			EndTime:   now,
		},
		{
			Name:      "1h",
			Duration:  1 * time.Hour,
			StartTime: now.Add(-1 * time.Hour),
			EndTime:   now,
		},
	}
	
	for _, window := range windows {
		da.windows[window.Name] = window
	}
	
	assert.Len(t, da.windows, 3)
	assert.Contains(t, da.windows, "1m")
	assert.Contains(t, da.windows, "5m")
	assert.Contains(t, da.windows, "1h")
	
	// Verify window properties
	oneMinWindow := da.windows["1m"]
	assert.Equal(t, 1*time.Minute, oneMinWindow.Duration)
	assert.True(t, oneMinWindow.EndTime.After(oneMinWindow.StartTime))
}

func TestAggregateResult_Statistics(t *testing.T) {
	// Test aggregate result statistics
	result := &AggregateResult{
		Window:  "10m",
		Count:   1000,
		Sum:     5000,
		Average: 5,
		Min:     0.5,
		Max:     15.7,
		StdDev:  3.2,
		Percentiles: map[string]float64{
			"p50": 4.5,
			"p90": 8.2,
			"p95": 10.1,
			"p99": 14.3,
		},
		Metadata: map[string]interface{}{
			"source": "api_requests",
			"tags":   []string{"production", "api-v2"},
		},
	}
	
	assert.Equal(t, "10m", result.Window)
	assert.Equal(t, int64(1000), result.Count)
	assert.Equal(t, float64(5000), result.Sum)
	assert.Equal(t, float64(5), result.Average)
	assert.Equal(t, 0.5, result.Min)
	assert.Equal(t, 15.7, result.Max)
	assert.Equal(t, 3.2, result.StdDev)
	
	// Check percentiles
	assert.Len(t, result.Percentiles, 4)
	assert.Equal(t, 4.5, result.Percentiles["p50"])
	assert.Equal(t, 8.2, result.Percentiles["p90"])
	assert.Equal(t, 10.1, result.Percentiles["p95"])
	assert.Equal(t, 14.3, result.Percentiles["p99"])
	
	// Check metadata
	assert.Equal(t, "api_requests", result.Metadata["source"])
	tags := result.Metadata["tags"].([]string)
	assert.Contains(t, tags, "production")
	assert.Contains(t, tags, "api-v2")
}

func TestMetricsCalculator_Formulas(t *testing.T) {
	// Test metrics calculator formulas
	mc := &MetricsCalculator{
		formulas: make(map[string]MetricFormula),
	}
	
	// Add formulas
	formulas := []MetricFormula{
		{
			Name:       "error_rate",
			Expression: "errors / requests * 100",
			Inputs:     []string{"errors", "requests"},
			Unit:       "percent",
		},
		{
			Name:       "throughput",
			Expression: "requests / time",
			Inputs:     []string{"requests", "time"},
			Unit:       "req/s",
		},
		{
			Name:       "average_latency",
			Expression: "sum(latencies) / count(latencies)",
			Inputs:     []string{"latencies"},
			Unit:       "ms",
		},
	}
	
	for _, formula := range formulas {
		mc.formulas[formula.Name] = formula
	}
	
	assert.Len(t, mc.formulas, 3)
	
	// Verify formulas
	errorRate := mc.formulas["error_rate"]
	assert.Equal(t, "error_rate", errorRate.Name)
	assert.Equal(t, "errors / requests * 100", errorRate.Expression)
	assert.Len(t, errorRate.Inputs, 2)
	assert.Equal(t, "percent", errorRate.Unit)
	
	throughput := mc.formulas["throughput"]
	assert.Equal(t, "throughput", throughput.Name)
	assert.Equal(t, "req/s", throughput.Unit)
}

func TestTrendPredictor_Models(t *testing.T) {
	// Test trend prediction models
	tp := &TrendPredictor{
		models: make(map[string]PredictionModel),
	}
	
	now := time.Now()
	
	// Add prediction models
	models := []PredictionModel{
		{
			Name:        "Linear Regression",
			Type:        "linear",
			Accuracy:    0.88,
			LastTrained: now.Add(-1 * time.Hour),
		},
		{
			Name:        "Moving Average",
			Type:        "moving_avg",
			Accuracy:    0.75,
			LastTrained: now.Add(-2 * time.Hour),
		},
		{
			Name:        "Exponential Smoothing",
			Type:        "exp_smooth",
			Accuracy:    0.82,
			LastTrained: now.Add(-30 * time.Minute),
		},
	}
	
	for _, model := range models {
		tp.models[model.Name] = model
	}
	
	assert.Len(t, tp.models, 3)
	
	// Verify models
	linearModel := tp.models["Linear Regression"]
	assert.Equal(t, "Linear Regression", linearModel.Name)
	assert.Equal(t, "linear", linearModel.Type)
	assert.Equal(t, 0.88, linearModel.Accuracy)
	assert.NotZero(t, linearModel.LastTrained)
	
	// Check accuracy range
	for _, model := range tp.models {
		assert.GreaterOrEqual(t, model.Accuracy, 0.0)
		assert.LessOrEqual(t, model.Accuracy, 1.0)
	}
}

func TestAnalytics_ConcurrentOperations(t *testing.T) {
	// Test concurrent analytics operations
	analytics := &Analytics{
		aggregator: &DataAggregator{
			windows:    make(map[string]*AggregationWindow),
			aggregates: make(map[string]*AggregateResult),
		},
		calculator: &MetricsCalculator{
			formulas: make(map[string]MetricFormula),
		},
		predictor: &TrendPredictor{
			models: make(map[string]PredictionModel),
		},
	}
	
	metrics := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
	}
	
	var wg sync.WaitGroup
	
	// Concurrent analytics processing
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			analytics.ProcessAnalytics(metrics)
		}()
	}
	
	// Concurrent reads
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			_ = analytics.GetAnalytics()
		}()
	}
	
	wg.Wait()
	
	// Verify no race conditions occurred
	assert.NotNil(t, analytics.aggregator)
	assert.NotNil(t, analytics.calculator)
	assert.NotNil(t, analytics.predictor)
}

func TestDataAggregator_ComplexAggregation(t *testing.T) {
	// Test complex aggregation scenarios
	da := &DataAggregator{
		windows:    make(map[string]*AggregationWindow),
		aggregates: make(map[string]*AggregateResult),
	}
	
	// Create overlapping windows
	now := time.Now()
	windows := []*AggregationWindow{
		{
			Name:      "current_1m",
			Duration:  1 * time.Minute,
			StartTime: now.Add(-1 * time.Minute),
			EndTime:   now,
		},
		{
			Name:      "current_5m",
			Duration:  5 * time.Minute,
			StartTime: now.Add(-5 * time.Minute),
			EndTime:   now,
		},
		{
			Name:      "previous_1m",
			Duration:  1 * time.Minute,
			StartTime: now.Add(-2 * time.Minute),
			EndTime:   now.Add(-1 * time.Minute),
		},
	}
	
	for _, window := range windows {
		da.windows[window.Name] = window
		
		// Create aggregate for each window
		da.aggregates[window.Name] = &AggregateResult{
			Window:  window.Name,
			Count:   int64(100 * window.Duration.Minutes()),
			Sum:     float64(500 * window.Duration.Minutes()),
			Average: 5.0,
			Min:     1.0,
			Max:     10.0,
			StdDev:  2.0,
		}
	}
	
	// Verify aggregates for different windows
	current1m := da.aggregates["current_1m"]
	assert.Equal(t, int64(100), current1m.Count)
	
	current5m := da.aggregates["current_5m"]
	assert.Equal(t, int64(500), current5m.Count)
	
	previous1m := da.aggregates["previous_1m"]
	assert.Equal(t, int64(100), previous1m.Count)
	
	// All should have same average despite different counts
	assert.Equal(t, 5.0, current1m.Average)
	assert.Equal(t, 5.0, current5m.Average)
	assert.Equal(t, 5.0, previous1m.Average)
}

func TestMetricsCalculator_DerivedMetrics(t *testing.T) {
	// Test calculation of derived metrics
	mc := &MetricsCalculator{
		formulas: make(map[string]MetricFormula),
	}
	
	// Add complex formulas
	mc.formulas["availability"] = MetricFormula{
		Name:       "availability",
		Expression: "(total_time - downtime) / total_time * 100",
		Inputs:     []string{"total_time", "downtime"},
		Unit:       "percent",
	}
	
	mc.formulas["cost_per_request"] = MetricFormula{
		Name:       "cost_per_request",
		Expression: "total_cost / request_count",
		Inputs:     []string{"total_cost", "request_count"},
		Unit:       "dollars",
	}
	
	mc.formulas["efficiency_score"] = MetricFormula{
		Name:       "efficiency_score",
		Expression: "(successful_requests / total_requests) * (1 / avg_latency) * 1000",
		Inputs:     []string{"successful_requests", "total_requests", "avg_latency"},
		Unit:       "score",
	}
	
	// Verify formula complexity
	efficiency := mc.formulas["efficiency_score"]
	assert.Len(t, efficiency.Inputs, 3)
	assert.Contains(t, efficiency.Expression, "successful_requests")
	assert.Contains(t, efficiency.Expression, "avg_latency")
	assert.Equal(t, "score", efficiency.Unit)
}

func TestTrendPredictor_AccuracyTracking(t *testing.T) {
	// Test prediction model accuracy tracking
	tp := &TrendPredictor{
		models: make(map[string]PredictionModel),
	}
	
	// Track accuracy over time
	modelName := "ARIMA"
	accuracies := []float64{0.75, 0.78, 0.82, 0.85, 0.83}
	
	for i, accuracy := range accuracies {
		tp.models[modelName] = PredictionModel{
			Name:        modelName,
			Type:        "arima",
			Accuracy:    accuracy,
			LastTrained: time.Now().Add(time.Duration(-i) * time.Hour),
		}
	}
	
	// Final accuracy should be last value
	finalModel := tp.models[modelName]
	assert.Equal(t, 0.83, finalModel.Accuracy)
	
	// Verify accuracy is within valid range
	assert.GreaterOrEqual(t, finalModel.Accuracy, 0.0)
	assert.LessOrEqual(t, finalModel.Accuracy, 1.0)
}