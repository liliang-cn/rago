package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Metrics Collection

func TestMetricsCollector_RegisterCounter(t *testing.T) {
	// Test counter registration
	mc := &MetricsCollector{
		counters: make(map[string]*Counter),
	}
	
	mc.RegisterCounter("test_counter", []string{"tag1", "tag2"})
	
	assert.Contains(t, mc.counters, "test_counter")
	counter := mc.counters["test_counter"]
	assert.Equal(t, "test_counter", counter.Name)
	assert.Equal(t, int64(0), counter.Value)
	assert.Equal(t, []string{"tag1", "tag2"}, counter.Tags)
	assert.NotZero(t, counter.LastUpdate)
}

func TestMetricsCollector_RegisterGauge(t *testing.T) {
	// Test gauge registration
	mc := &MetricsCollector{
		gauges: make(map[string]*Gauge),
	}
	
	mc.RegisterGauge("test_gauge", []string{"env:prod"})
	
	assert.Contains(t, mc.gauges, "test_gauge")
	gauge := mc.gauges["test_gauge"]
	assert.Equal(t, "test_gauge", gauge.Name)
	assert.Equal(t, float64(0), gauge.Value)
	assert.Equal(t, []string{"env:prod"}, gauge.Tags)
	assert.NotZero(t, gauge.LastUpdate)
}

func TestMetricsCollector_RegisterHistogram(t *testing.T) {
	// Test histogram registration
	mc := &MetricsCollector{
		histograms: make(map[string]*Histogram),
	}
	
	buckets := []float64{0.1, 0.5, 1.0, 2.0, 5.0}
	mc.RegisterHistogram("response_time", buckets)
	
	assert.Contains(t, mc.histograms, "response_time")
	hist := mc.histograms["response_time"]
	assert.Equal(t, "response_time", hist.Name)
	assert.Equal(t, buckets, hist.buckets)
	assert.NotNil(t, hist.values)
	assert.NotNil(t, hist.Percentiles)
	assert.NotZero(t, hist.LastUpdate)
}

func TestMetricsCollector_IncrementCounter(t *testing.T) {
	// Test counter increment
	mc := &MetricsCollector{
		counters: make(map[string]*Counter),
	}
	
	mc.RegisterCounter("requests", []string{})
	
	// Increment counter
	mc.IncrementCounter("requests", 1, []string{})
	assert.Equal(t, int64(1), mc.counters["requests"].Value)
	
	// Increment again
	mc.IncrementCounter("requests", 5, []string{})
	assert.Equal(t, int64(6), mc.counters["requests"].Value)
	
	// Try to increment non-existent counter (should be no-op)
	mc.IncrementCounter("non_existent", 1, []string{})
	assert.NotContains(t, mc.counters, "non_existent")
}

func TestMetricsCollector_UpdateGauge(t *testing.T) {
	// Test gauge updates
	mc := &MetricsCollector{
		gauges: make(map[string]*Gauge),
	}
	
	mc.RegisterGauge("cpu_usage", []string{})
	
	// Update gauge
	mc.UpdateGauge("cpu_usage", 0.45, []string{})
	assert.Equal(t, 0.45, mc.gauges["cpu_usage"].Value)
	
	// Update to new value
	mc.UpdateGauge("cpu_usage", 0.67, []string{})
	assert.Equal(t, 0.67, mc.gauges["cpu_usage"].Value)
	
	// Try to update non-existent gauge (should be no-op)
	mc.UpdateGauge("non_existent", 0.5, []string{})
	assert.NotContains(t, mc.gauges, "non_existent")
}

func TestMetricsCollector_RecordHistogram(t *testing.T) {
	// Test histogram recording
	mc := &MetricsCollector{
		histograms: make(map[string]*Histogram),
	}
	
	mc.RegisterHistogram("latency", []float64{1, 5, 10, 50, 100})
	
	// Record values
	values := []float64{2.5, 7.3, 15.2, 3.1, 45.6}
	for _, v := range values {
		mc.RecordHistogram("latency", v)
	}
	
	hist := mc.histograms["latency"]
	assert.Equal(t, int64(5), hist.Count)
	assert.InDelta(t, 73.7, hist.Sum, 0.01)
	assert.Equal(t, 2.5, hist.Min)
	assert.Equal(t, 45.6, hist.Max)
	assert.InDelta(t, 14.74, hist.Mean, 0.01)
	
	// Try to record to non-existent histogram (should be no-op)
	mc.RecordHistogram("non_existent", 10.0)
	assert.NotContains(t, mc.histograms, "non_existent")
}

func TestMetricsCollector_GetCurrentMetrics(t *testing.T) {
	// Test getting current metrics snapshot
	mc := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timeSeries: make(map[string]*TimeSeries),
	}
	
	// Register various metrics
	mc.RegisterCounter("counter1", []string{})
	mc.RegisterGauge("gauge1", []string{})
	mc.RegisterHistogram("hist1", []float64{1, 5, 10})
	
	// Get snapshot
	snapshot := mc.GetCurrentMetrics()
	
	assert.NotNil(t, snapshot["counters"])
	assert.NotNil(t, snapshot["gauges"])
	assert.NotNil(t, snapshot["histograms"])
	assert.NotNil(t, snapshot["timeseries"])
	
	// Verify specific metrics are in snapshot
	counters := snapshot["counters"].(map[string]*Counter)
	assert.Contains(t, counters, "counter1")
	
	gauges := snapshot["gauges"].(map[string]*Gauge)
	assert.Contains(t, gauges, "gauge1")
	
	histograms := snapshot["histograms"].(map[string]*Histogram)
	assert.Contains(t, histograms, "hist1")
}

func TestMetricsCollector_GetMetricsCount(t *testing.T) {
	// Test metrics count
	mc := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timeSeries: make(map[string]*TimeSeries),
	}
	
	assert.Equal(t, 0, mc.GetMetricsCount())
	
	mc.RegisterCounter("c1", []string{})
	mc.RegisterCounter("c2", []string{})
	assert.Equal(t, 2, mc.GetMetricsCount())
	
	mc.RegisterGauge("g1", []string{})
	assert.Equal(t, 3, mc.GetMetricsCount())
	
	mc.RegisterHistogram("h1", []float64{1, 5})
	assert.Equal(t, 4, mc.GetMetricsCount())
}

func TestMetricsCollector_ConcurrentOperations(t *testing.T) {
	// Test concurrent metric operations for race conditions
	mc := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timeSeries: make(map[string]*TimeSeries),
	}
	
	// Register metrics
	mc.RegisterCounter("concurrent_counter", []string{})
	mc.RegisterGauge("concurrent_gauge", []string{})
	mc.RegisterHistogram("concurrent_hist", []float64{1, 5, 10, 50})
	
	var wg sync.WaitGroup
	
	// Concurrent counter increments
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mc.IncrementCounter("concurrent_counter", 1, []string{})
			}
		}()
	}
	
	// Concurrent gauge updates
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mc.UpdateGauge("concurrent_gauge", float64(id*j), []string{})
			}
		}(i)
	}
	
	// Concurrent histogram recordings
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mc.RecordHistogram("concurrent_hist", float64(id+j))
			}
		}(i)
	}
	
	// Concurrent reads
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = mc.GetCurrentMetrics()
				_ = mc.GetMetricsCount()
			}
		}()
	}
	
	wg.Wait()
	
	// Verify final state
	assert.Equal(t, int64(1000), mc.counters["concurrent_counter"].Value)
	assert.NotNil(t, mc.gauges["concurrent_gauge"].Value)
	assert.Equal(t, int64(1000), mc.histograms["concurrent_hist"].Count)
}

func TestMetricsCollector_TimeSeries(t *testing.T) {
	// Test time series functionality
	mc := &MetricsCollector{
		timeSeries: make(map[string]*TimeSeries),
	}
	
	// Create a time series
	ts := &TimeSeries{
		Name:       "request_rate",
		Resolution: time.Second,
		MaxPoints:  100,
		Points:     make([]DataPoint, 0),
	}
	
	mc.mu.Lock()
	mc.timeSeries["request_rate"] = ts
	mc.mu.Unlock()
	
	// Add data points
	now := time.Now()
	for i := 0; i < 5; i++ {
		point := DataPoint{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Value:     float64(i * 10),
		}
		mc.mu.Lock()
		mc.timeSeries["request_rate"].Points = append(mc.timeSeries["request_rate"].Points, point)
		mc.mu.Unlock()
	}
	
	// Verify points
	mc.mu.RLock()
	series := mc.timeSeries["request_rate"]
	mc.mu.RUnlock()
	
	assert.Equal(t, 5, len(series.Points))
	assert.Equal(t, float64(0), series.Points[0].Value)
	assert.Equal(t, float64(40), series.Points[4].Value)
}

func TestMetricsCollector_MetricsStorage(t *testing.T) {
	// Test metrics with storage interface
	storage := &mockMetricsStorage{}
	mc := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timeSeries: make(map[string]*TimeSeries),
		storage:    storage,
	}
	
	// Register and update metrics
	mc.RegisterCounter("stored_counter", []string{})
	mc.IncrementCounter("stored_counter", 5, []string{})
	
	// Save to storage
	if mc.storage != nil {
		err := mc.storage.SaveMetric(mc.counters["stored_counter"])
		assert.NoError(t, err)
	}
	
	// Query from storage
	query := &MetricsQuery{
		MetricNames: []string{"stored_counter"},
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
	}
	
	results, err := storage.QueryMetrics(query)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Test delete old metrics
	err = storage.DeleteOldMetrics(time.Now().Add(-24 * time.Hour))
	assert.NoError(t, err)
}

func TestMetricsCollector_ErrorHandling(t *testing.T) {
	// Test error handling with storage
	storage := &mockMetricsStorage{
		err: assert.AnError,
	}
	
	mc := &MetricsCollector{
		counters: make(map[string]*Counter),
		storage:  storage,
	}
	
	mc.RegisterCounter("error_counter", []string{})
	
	// Save should fail
	err := storage.SaveMetric(mc.counters["error_counter"])
	assert.Error(t, err)
	
	// Query should fail
	query := &MetricsQuery{
		MetricNames: []string{"error_counter"},
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
	}
	
	_, err = storage.QueryMetrics(query)
	assert.Error(t, err)
	
	// Delete should fail
	err = storage.DeleteOldMetrics(time.Now())
	assert.Error(t, err)
}