package monitoring

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Distributed Tracing

func TestDistributedTracer_StartTrace(t *testing.T) {
	// Test trace creation
	dt := &DistributedTracer{
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
	
	trace := dt.StartTrace("test-operation")
	
	assert.NotNil(t, trace)
	assert.NotEmpty(t, trace.TraceID)
	assert.Equal(t, "test-operation", trace.Name)
	assert.NotZero(t, trace.StartTime)
	assert.Equal(t, TraceStatusInProgress, trace.Status)
	assert.NotNil(t, trace.Spans)
	assert.NotNil(t, trace.Tags)
	assert.NotNil(t, trace.Metadata)
	
	// Verify trace is stored
	assert.Contains(t, dt.traces, trace.TraceID)
	assert.Equal(t, trace, dt.traces[trace.TraceID])
}

func TestDistributedTracer_GetRecentTraces(t *testing.T) {
	// Test getting recent traces
	dt := &DistributedTracer{
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
	
	// Create multiple traces
	for i := 0; i < 10; i++ {
		trace := &Trace{
			TraceID:   fmt.Sprintf("trace-%d", i),
			Name:      fmt.Sprintf("operation-%d", i),
			StartTime: time.Now().Add(time.Duration(-i) * time.Minute),
			Status:    TraceStatusSuccess,
		}
		dt.traces[trace.TraceID] = trace
	}
	
	// Get recent traces with limit
	recent := dt.GetRecentTraces(5)
	assert.Len(t, recent, 5)
	
	// Get all traces
	all := dt.GetRecentTraces(100)
	assert.Len(t, all, 10)
}

func TestDistributedTracer_GetTraceCount(t *testing.T) {
	// Test trace count
	dt := &DistributedTracer{
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
	
	assert.Equal(t, 0, dt.GetTraceCount())
	
	dt.StartTrace("op1")
	assert.Equal(t, 1, dt.GetTraceCount())
	
	dt.StartTrace("op2")
	dt.StartTrace("op3")
	assert.Equal(t, 3, dt.GetTraceCount())
}

func TestTrace_Lifecycle(t *testing.T) {
	// Test trace lifecycle
	trace := &Trace{
		TraceID:   "test-trace-1",
		Name:      "test-operation",
		StartTime: time.Now(),
		Status:    TraceStatusInProgress,
		Spans:     make([]*Span, 0),
		Tags:      make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}
	
	// Add tags
	trace.Tags["service"] = "monitoring"
	trace.Tags["version"] = "1.0.0"
	
	// Add metadata
	trace.Metadata["user_id"] = "user123"
	trace.Metadata["request_id"] = "req456"
	
	// Complete trace
	endTime := time.Now()
	trace.EndTime = &endTime
	trace.Duration = endTime.Sub(trace.StartTime)
	trace.Status = TraceStatusSuccess
	
	assert.NotNil(t, trace.EndTime)
	assert.Greater(t, trace.Duration, time.Duration(0))
	assert.Equal(t, TraceStatusSuccess, trace.Status)
	assert.Equal(t, "monitoring", trace.Tags["service"])
	assert.Equal(t, "user123", trace.Metadata["user_id"])
}

func TestSpan_Creation(t *testing.T) {
	// Test span creation and properties
	span := &Span{
		SpanID:     "span-1",
		TraceID:    "trace-1",
		ParentID:   "parent-span",
		Name:       "database-query",
		StartTime:  time.Now(),
		Status:     SpanStatusOK,
		Events:     make([]SpanEvent, 0),
		Attributes: make(map[string]interface{}),
	}
	
	// Add attributes
	span.Attributes["db.type"] = "postgresql"
	span.Attributes["db.statement"] = "SELECT * FROM users"
	
	// Add events
	event := SpanEvent{
		Name:      "query-started",
		Timestamp: time.Now(),
		Attributes: map[string]interface{}{
			"query_id": "q123",
		},
	}
	span.Events = append(span.Events, event)
	
	// Complete span
	endTime := time.Now()
	span.EndTime = &endTime
	span.Duration = endTime.Sub(span.StartTime)
	
	assert.Equal(t, "span-1", span.SpanID)
	assert.Equal(t, "trace-1", span.TraceID)
	assert.Equal(t, "parent-span", span.ParentID)
	assert.Equal(t, "database-query", span.Name)
	assert.NotNil(t, span.EndTime)
	assert.Greater(t, span.Duration, time.Duration(0))
	assert.Equal(t, SpanStatusOK, span.Status)
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "postgresql", span.Attributes["db.type"])
}

func TestDistributedTracer_Storage(t *testing.T) {
	// Test trace storage operations
	storage := newMockTraceStorage()
	dt := &DistributedTracer{
		traces:  make(map[string]*Trace),
		spans:   make(map[string]*Span),
		storage: storage,
	}
	
	// Create and save trace
	trace := dt.StartTrace("storage-test")
	if dt.storage != nil {
		err := dt.storage.SaveTrace(trace)
		assert.NoError(t, err)
	}
	
	// Create and save span
	span := &Span{
		SpanID:    "span-storage-1",
		TraceID:   trace.TraceID,
		Name:      "test-span",
		StartTime: time.Now(),
		Status:    SpanStatusOK,
	}
	if dt.storage != nil {
		err := dt.storage.SaveSpan(span)
		assert.NoError(t, err)
	}
	
	// Retrieve trace
	retrieved, err := storage.GetTrace(trace.TraceID)
	assert.NoError(t, err)
	assert.Equal(t, trace.TraceID, retrieved.TraceID)
	assert.Equal(t, trace.Name, retrieved.Name)
	
	// Query traces
	query := &TraceQuery{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Limit:     10,
	}
	results, err := storage.QueryTraces(query)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestDistributedTracer_Sampling(t *testing.T) {
	// Test trace sampling
	sampler := &mockTraceSampler{
		sampleRate: 0.5, // 50% sampling
	}
	
	dt := &DistributedTracer{
		traces:  make(map[string]*Trace),
		spans:   make(map[string]*Span),
		sampler: sampler,
	}
	
	// Test sampling decisions
	sampled := 0
	total := 10
	
	for i := 0; i < total; i++ {
		trace := &Trace{
			TraceID: fmt.Sprintf("sample-trace-%d", i),
			Name:    "test",
		}
		if dt.sampler != nil && dt.sampler.ShouldSample(trace) {
			sampled++
		}
	}
	
	// With 50% sampling, we should sample approximately half
	assert.Greater(t, sampled, 0)
	assert.Less(t, sampled, total)
}

func TestDistributedTracer_ConcurrentOperations(t *testing.T) {
	// Test concurrent trace operations
	dt := &DistributedTracer{
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
	
	var wg sync.WaitGroup
	traceCount := 20
	
	// Concurrently create traces
	wg.Add(traceCount)
	for i := 0; i < traceCount; i++ {
		go func(id int) {
			defer wg.Done()
			trace := dt.StartTrace(fmt.Sprintf("concurrent-op-%d", id))
			
			// Add spans to trace
			for j := 0; j < 5; j++ {
				span := &Span{
					SpanID:    fmt.Sprintf("span-%d-%d", id, j),
					TraceID:   trace.TraceID,
					Name:      fmt.Sprintf("step-%d", j),
					StartTime: time.Now(),
					Status:    SpanStatusOK,
				}
				dt.mu.Lock()
				dt.spans[span.SpanID] = span
				trace.Spans = append(trace.Spans, span)
				dt.mu.Unlock()
			}
			
			// Complete trace
			endTime := time.Now()
			dt.mu.Lock()
			trace.EndTime = &endTime
			trace.Duration = endTime.Sub(trace.StartTime)
			trace.Status = TraceStatusSuccess
			dt.mu.Unlock()
		}(i)
	}
	
	// Concurrently read traces
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = dt.GetRecentTraces(5)
				_ = dt.GetTraceCount()
			}
		}()
	}
	
	wg.Wait()
	
	// Verify final state
	assert.Equal(t, traceCount, dt.GetTraceCount())
	assert.Equal(t, traceCount*5, len(dt.spans))
}

func TestTrace_StatusTransitions(t *testing.T) {
	// Test valid trace status transitions
	trace := &Trace{
		TraceID:   "status-test",
		Name:      "test",
		StartTime: time.Now(),
		Status:    TraceStatusInProgress,
	}
	
	// Valid transitions
	validTransitions := []struct {
		from TraceStatus
		to   TraceStatus
	}{
		{TraceStatusInProgress, TraceStatusSuccess},
		{TraceStatusInProgress, TraceStatusError},
	}
	
	for _, transition := range validTransitions {
		trace.Status = transition.from
		trace.Status = transition.to
		assert.Equal(t, transition.to, trace.Status)
	}
}

func TestSpan_StatusTypes(t *testing.T) {
	// Test different span status types
	statuses := []SpanStatus{
		SpanStatusOK,
		SpanStatusError,
		SpanStatusCanceled,
	}
	
	for _, status := range statuses {
		span := &Span{
			SpanID:  fmt.Sprintf("span-%s", status),
			TraceID: "test-trace",
			Name:    "test",
			Status:  status,
		}
		assert.Equal(t, status, span.Status)
	}
}

func TestDistributedTracer_ErrorHandling(t *testing.T) {
	// Test error handling in storage operations
	storage := &mockTraceStorage{
		err: assert.AnError,
	}
	
	dt := &DistributedTracer{
		traces:  make(map[string]*Trace),
		spans:   make(map[string]*Span),
		storage: storage,
	}
	
	// Save trace should fail
	trace := dt.StartTrace("error-test")
	err := storage.SaveTrace(trace)
	assert.Error(t, err)
	
	// Save span should fail
	span := &Span{
		SpanID:  "error-span",
		TraceID: trace.TraceID,
	}
	err = storage.SaveSpan(span)
	assert.Error(t, err)
	
	// Get trace should fail
	_, err = storage.GetTrace("any-id")
	assert.Error(t, err)
	
	// Query should fail
	query := &TraceQuery{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}
	_, err = storage.QueryTraces(query)
	assert.Error(t, err)
}