package monitoring

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing

type mockMetricsStorage struct {
	mu      sync.RWMutex
	metrics []interface{}
	err     error
}

func (m *mockMetricsStorage) SaveMetric(metric interface{}) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *mockMetricsStorage) QueryMetrics(query *MetricsQuery) ([]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics, nil
}

func (m *mockMetricsStorage) DeleteOldMetrics(before time.Time) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Filter out old metrics
	var newMetrics []interface{}
	for _, metric := range m.metrics {
		// In real implementation, check timestamp
		newMetrics = append(newMetrics, metric)
	}
	m.metrics = newMetrics
	return nil
}

type mockTraceStorage struct {
	mu     sync.RWMutex
	traces map[string]*Trace
	spans  map[string]*Span
	err    error
}

func newMockTraceStorage() *mockTraceStorage {
	return &mockTraceStorage{
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
}

func (m *mockTraceStorage) SaveTrace(trace *Trace) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traces[trace.TraceID] = trace
	return nil
}

func (m *mockTraceStorage) SaveSpan(span *Span) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans[span.SpanID] = span
	return nil
}

func (m *mockTraceStorage) GetTrace(traceID string) (*Trace, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	trace, exists := m.traces[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}
	return trace, nil
}

func (m *mockTraceStorage) QueryTraces(query *TraceQuery) ([]*Trace, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []*Trace
	for _, trace := range m.traces {
		result = append(result, trace)
		if query.Limit > 0 && len(result) >= query.Limit {
			break
		}
	}
	return result, nil
}

type mockTraceSampler struct {
	sampleRate float64
	counter    int
}

func (m *mockTraceSampler) ShouldSample(trace *Trace) bool {
	m.counter++
	// Simple deterministic sampling for testing
	return m.counter%int(1/m.sampleRate) == 0
}

type mockAlertChannel struct {
	mu     sync.RWMutex
	alerts []*Alert
	name   string
	err    error
}

func (m *mockAlertChannel) Send(alert *Alert) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockAlertChannel) Name() string {
	return m.name
}

// Test Dashboard Core

func TestNewDashboard_DefaultConfig(t *testing.T) {
	// Test creating dashboard with default configuration
	dashboard := NewDashboard(nil)
	
	assert.NotNil(t, dashboard)
	assert.NotNil(t, dashboard.config)
	assert.Equal(t, 8080, dashboard.config.Port)
	assert.Equal(t, 5*time.Second, dashboard.config.UpdateInterval)
	assert.Equal(t, 24*time.Hour, dashboard.config.MetricsRetention)
	assert.True(t, dashboard.config.EnableTracing)
	assert.True(t, dashboard.config.EnableAlerts)
	assert.True(t, dashboard.config.EnableAnalytics)
	assert.Equal(t, 100, dashboard.config.MaxClients)
	
	// Verify internal components are initialized
	assert.NotNil(t, dashboard.metrics)
	assert.NotNil(t, dashboard.tracer)
	assert.NotNil(t, dashboard.analytics)
	assert.NotNil(t, dashboard.alerts)
	assert.NotNil(t, dashboard.clients)
	assert.NotNil(t, dashboard.broadcast)
	
	// Verify default metrics are registered
	assert.NotEmpty(t, dashboard.metrics.counters)
	assert.NotEmpty(t, dashboard.metrics.gauges)
	assert.NotEmpty(t, dashboard.metrics.histograms)
	
	// Verify default alert rules are registered when alerts enabled
	assert.NotEmpty(t, dashboard.alerts.rules)
}

func TestNewDashboard_CustomConfig(t *testing.T) {
	// Test creating dashboard with custom configuration
	config := &DashboardConfig{
		Port:             9090,
		UpdateInterval:   10 * time.Second,
		MetricsRetention: 48 * time.Hour,
		EnableTracing:    false,
		EnableAlerts:     false,
		EnableAnalytics:  false,
		MaxClients:       50,
	}
	
	dashboard := NewDashboard(config)
	
	assert.NotNil(t, dashboard)
	assert.Equal(t, config, dashboard.config)
	assert.Equal(t, 9090, dashboard.config.Port)
	assert.Equal(t, 10*time.Second, dashboard.config.UpdateInterval)
	assert.Equal(t, 48*time.Hour, dashboard.config.MetricsRetention)
	assert.False(t, dashboard.config.EnableTracing)
	assert.False(t, dashboard.config.EnableAlerts)
	assert.False(t, dashboard.config.EnableAnalytics)
	assert.Equal(t, 50, dashboard.config.MaxClients)
	
	// Verify alert rules are not registered when alerts disabled
	assert.Empty(t, dashboard.alerts.rules)
}

func TestDashboard_InitializeDefaultMetrics(t *testing.T) {
	// Test that default metrics are properly initialized
	dashboard := NewDashboard(nil)
	
	// System metrics
	assert.Contains(t, dashboard.metrics.counters, "requests_total")
	assert.Contains(t, dashboard.metrics.counters, "errors_total")
	assert.Contains(t, dashboard.metrics.gauges, "memory_usage")
	assert.Contains(t, dashboard.metrics.gauges, "cpu_usage")
	assert.Contains(t, dashboard.metrics.gauges, "active_connections")
	assert.Contains(t, dashboard.metrics.histograms, "request_duration")
	assert.Contains(t, dashboard.metrics.histograms, "response_size")
	
	// RAG metrics
	assert.Contains(t, dashboard.metrics.counters, "rag_queries_total")
	assert.Contains(t, dashboard.metrics.counters, "rag_documents_ingested")
	assert.Contains(t, dashboard.metrics.gauges, "rag_index_size")
	
	// LLM metrics
	assert.Contains(t, dashboard.metrics.counters, "llm_requests_total")
	assert.Contains(t, dashboard.metrics.counters, "llm_tokens_total")
	assert.Contains(t, dashboard.metrics.gauges, "llm_cost_total")
	
	// Workflow metrics
	assert.Contains(t, dashboard.metrics.counters, "workflows_executed")
	assert.Contains(t, dashboard.metrics.histograms, "workflow_duration")
}

func TestDashboard_InitializeDefaultAlertRules(t *testing.T) {
	// Test that default alert rules are properly initialized
	dashboard := NewDashboard(&DashboardConfig{
		EnableAlerts: true,
	})
	
	// Check specific alert rules
	assert.Contains(t, dashboard.alerts.rules, "high_error_rate")
	assert.Contains(t, dashboard.alerts.rules, "high_latency")
	assert.Contains(t, dashboard.alerts.rules, "low_disk_space")
	assert.Contains(t, dashboard.alerts.rules, "cost_threshold")
	
	// Verify rule configuration
	errorRule := dashboard.alerts.rules["high_error_rate"]
	assert.Equal(t, "High Error Rate", errorRule.Name)
	assert.Equal(t, 0.05, errorRule.Threshold)
	assert.Equal(t, 5*time.Minute, errorRule.Duration)
	assert.Equal(t, AlertSeverityWarning, errorRule.Severity)
	assert.True(t, errorRule.Enabled)
	
	latencyRule := dashboard.alerts.rules["high_latency"]
	assert.Equal(t, "High Latency", latencyRule.Name)
	assert.Equal(t, float64(5000), latencyRule.Threshold)
	assert.Equal(t, 2*time.Minute, latencyRule.Duration)
	assert.Equal(t, AlertSeverityWarning, latencyRule.Severity)
	assert.True(t, latencyRule.Enabled)
}

// Test WebSocket Client Management

func TestDashboardClient_Registration(t *testing.T) {
	// Test client registration and cleanup
	dashboard := NewDashboard(nil)
	
	// Create a mock client
	client := &DashboardClient{
		ID:   "test-client-1",
		Send: make(chan DashboardUpdate, 10),
	}
	
	// Register client
	dashboard.mu.Lock()
	dashboard.clients[client.ID] = client
	dashboard.mu.Unlock()
	
	// Verify registration
	dashboard.mu.RLock()
	registeredClient, exists := dashboard.clients[client.ID]
	dashboard.mu.RUnlock()
	
	assert.True(t, exists)
	assert.Equal(t, client, registeredClient)
	
	// Cleanup client
	dashboard.mu.Lock()
	delete(dashboard.clients, client.ID)
	dashboard.mu.Unlock()
	
	// Verify cleanup
	dashboard.mu.RLock()
	_, exists = dashboard.clients[client.ID]
	dashboard.mu.RUnlock()
	
	assert.False(t, exists)
}

func TestDashboard_BroadcastToClients(t *testing.T) {
	// Test broadcasting updates to multiple clients
	dashboard := NewDashboard(nil)
	
	// Create multiple mock clients
	clients := make([]*DashboardClient, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &DashboardClient{
			ID:   fmt.Sprintf("client-%d", i),
			Send: make(chan DashboardUpdate, 10),
		}
		dashboard.mu.Lock()
		dashboard.clients[clients[i].ID] = clients[i]
		dashboard.mu.Unlock()
	}
	
	// Start broadcast handler in background
	go dashboard.handleBroadcast()
	
	// Send an update
	update := DashboardUpdate{
		Type:      UpdateTypeMetrics,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test": "data",
		},
	}
	
	dashboard.broadcast <- update
	
	// Wait for broadcast to complete
	time.Sleep(100 * time.Millisecond)
	
	// Verify all clients received the update
	for _, client := range clients {
		select {
		case received := <-client.Send:
			assert.Equal(t, update.Type, received.Type)
			assert.Equal(t, update.Data, received.Data)
		default:
			t.Errorf("Client %s did not receive update", client.ID)
		}
	}
}

func TestDashboard_ClientChannelOverflow(t *testing.T) {
	// Test handling of client send channel overflow
	dashboard := NewDashboard(nil)
	
	// Create a client with small buffer
	client := &DashboardClient{
		ID:   "overflow-client",
		Send: make(chan DashboardUpdate, 1), // Very small buffer
	}
	
	dashboard.mu.Lock()
	dashboard.clients[client.ID] = client
	dashboard.mu.Unlock()
	
	// Start broadcast handler
	go dashboard.handleBroadcast()
	
	// Fill the client's channel
	client.Send <- DashboardUpdate{Type: UpdateTypeMetrics}
	
	// Send another update that should trigger overflow handling
	update := DashboardUpdate{
		Type:      UpdateTypeAlert,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"alert": "test"},
	}
	
	dashboard.broadcast <- update
	
	// Wait for broadcast to process
	time.Sleep(100 * time.Millisecond)
	
	// Client should be removed due to overflow
	dashboard.mu.RLock()
	_, exists := dashboard.clients[client.ID]
	dashboard.mu.RUnlock()
	
	assert.False(t, exists, "Client should be removed after channel overflow")
}

func TestDashboard_MaxClientsLimit(t *testing.T) {
	// Test enforcement of maximum clients limit
	config := &DashboardConfig{
		MaxClients: 5,
	}
	dashboard := NewDashboard(config)
	
	// Add clients up to the limit
	for i := 0; i < 5; i++ {
		client := &DashboardClient{
			ID:   fmt.Sprintf("client-%d", i),
			Send: make(chan DashboardUpdate, 10),
		}
		dashboard.mu.Lock()
		dashboard.clients[client.ID] = client
		dashboard.mu.Unlock()
	}
	
	// Verify limit is reached
	dashboard.mu.RLock()
	clientCount := len(dashboard.clients)
	dashboard.mu.RUnlock()
	
	assert.Equal(t, 5, clientCount)
	
	// Attempting to add more should respect the limit
	// This would be enforced in the WebSocket handler
}

func TestDashboard_ConcurrentClientOperations(t *testing.T) {
	// Test concurrent client operations for race conditions
	dashboard := NewDashboard(nil)
	
	// Start broadcast handler
	go dashboard.handleBroadcast()
	
	var wg sync.WaitGroup
	clientCount := 10
	
	// Concurrently add clients
	wg.Add(clientCount)
	for i := 0; i < clientCount; i++ {
		go func(id int) {
			defer wg.Done()
			client := &DashboardClient{
				ID:   fmt.Sprintf("concurrent-client-%d", id),
				Send: make(chan DashboardUpdate, 10),
			}
			dashboard.mu.Lock()
			dashboard.clients[client.ID] = client
			dashboard.mu.Unlock()
		}(i)
	}
	
	// Concurrently broadcast updates
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer wg.Done()
			update := DashboardUpdate{
				Type:      UpdateTypeMetrics,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"update": id,
				},
			}
			dashboard.broadcast <- update
		}(i)
	}
	
	// Concurrently remove clients
	wg.Add(clientCount / 2)
	for i := 0; i < clientCount/2; i++ {
		go func(id int) {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Small delay
			clientID := fmt.Sprintf("concurrent-client-%d", id)
			dashboard.mu.Lock()
			delete(dashboard.clients, clientID)
			dashboard.mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	
	// Verify remaining clients
	dashboard.mu.RLock()
	remainingClients := len(dashboard.clients)
	dashboard.mu.RUnlock()
	
	assert.GreaterOrEqual(t, remainingClients, clientCount/2)
}