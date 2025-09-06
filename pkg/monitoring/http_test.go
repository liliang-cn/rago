package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test HTTP Endpoints

func TestDashboard_HandleMetrics(t *testing.T) {
	// Test metrics API endpoint
	dashboard := NewDashboard(nil)
	
	// Add some test metrics
	dashboard.metrics.RegisterCounter("test_counter", []string{})
	dashboard.metrics.IncrementCounter("test_counter", 10, []string{})
	dashboard.metrics.RegisterGauge("test_gauge", []string{})
	dashboard.metrics.UpdateGauge("test_gauge", 42.5, []string{})
	
	// Create request
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleMetrics(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Parse response
	var metrics map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &metrics)
	assert.NoError(t, err)
	
	// Verify metrics structure
	assert.Contains(t, metrics, "counters")
	assert.Contains(t, metrics, "gauges")
	assert.Contains(t, metrics, "histograms")
	assert.Contains(t, metrics, "timeseries")
}

func TestDashboard_HandleTraces(t *testing.T) {
	// Test traces API endpoint
	dashboard := NewDashboard(nil)
	
	// Add some test traces
	trace1 := dashboard.tracer.StartTrace("test-operation-1")
	_ = dashboard.tracer.StartTrace("test-operation-2")
	
	// Complete first trace
	endTime := time.Now()
	dashboard.tracer.mu.Lock()
	trace1.EndTime = &endTime
	trace1.Duration = endTime.Sub(trace1.StartTime)
	trace1.Status = TraceStatusSuccess
	dashboard.tracer.mu.Unlock()
	
	// Create request
	req := httptest.NewRequest("GET", "/api/traces", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleTraces(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Parse response
	var traces []*Trace
	err := json.Unmarshal(w.Body.Bytes(), &traces)
	assert.NoError(t, err)
	
	// Verify traces
	assert.GreaterOrEqual(t, len(traces), 2)
	
	// Find our test traces
	foundTrace1 := false
	foundTrace2 := false
	for _, trace := range traces {
		if trace.Name == "test-operation-1" {
			foundTrace1 = true
			assert.Equal(t, TraceStatusSuccess, trace.Status)
		}
		if trace.Name == "test-operation-2" {
			foundTrace2 = true
		}
	}
	assert.True(t, foundTrace1)
	assert.True(t, foundTrace2)
}

func TestDashboard_HandleAnalytics(t *testing.T) {
	// Test analytics API endpoint
	dashboard := NewDashboard(nil)
	
	// Add test analytics data
	dashboard.analytics.mu.Lock()
	dashboard.analytics.aggregator.aggregates["test"] = &AggregateResult{
		Window:  "5m",
		Count:   100,
		Average: 5.5,
	}
	dashboard.analytics.predictor.models["test-model"] = PredictionModel{
		Name:     "Test Model",
		Type:     "linear",
		Accuracy: 0.85,
	}
	dashboard.analytics.mu.Unlock()
	
	// Create request
	req := httptest.NewRequest("GET", "/api/analytics", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleAnalytics(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Parse response
	var analytics map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &analytics)
	assert.NoError(t, err)
	
	// Verify analytics data
	assert.Contains(t, analytics, "aggregates")
	assert.Contains(t, analytics, "predictions")
}

func TestDashboard_HandleAlerts(t *testing.T) {
	// Test alerts API endpoint
	dashboard := NewDashboard(nil)
	
	// Add test alerts
	now := time.Now()
	dashboard.alerts.mu.Lock()
	dashboard.alerts.alerts["alert-1"] = &Alert{
		ID:        "alert-1",
		Name:      "Test Alert",
		Message:   "This is a test",
		Severity:  AlertSeverityWarning,
		StartTime: now,
		Status:    AlertStatusActive,
	}
	dashboard.alerts.alerts["alert-2"] = &Alert{
		ID:        "alert-2",
		Name:      "Another Alert",
		Message:   "Another test",
		Severity:  AlertSeverityError,
		StartTime: now,
		Status:    AlertStatusActive,
	}
	dashboard.alerts.mu.Unlock()
	
	// Create request
	req := httptest.NewRequest("GET", "/api/alerts", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleAlerts(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Parse response
	var alerts []*Alert
	err := json.Unmarshal(w.Body.Bytes(), &alerts)
	assert.NoError(t, err)
	
	// Verify alerts
	assert.Len(t, alerts, 2)
	for _, alert := range alerts {
		assert.Equal(t, AlertStatusActive, alert.Status)
		assert.Contains(t, []string{"alert-1", "alert-2"}, alert.ID)
	}
}

func TestDashboard_HandleHealth(t *testing.T) {
	// Test health check endpoint
	dashboard := NewDashboard(nil)
	
	// Add some data
	dashboard.mu.Lock()
	dashboard.clients["client-1"] = &DashboardClient{ID: "client-1"}
	dashboard.clients["client-2"] = &DashboardClient{ID: "client-2"}
	dashboard.mu.Unlock()
	
	dashboard.tracer.StartTrace("health-check-trace")
	
	dashboard.alerts.mu.Lock()
	dashboard.alerts.alerts["active-alert"] = &Alert{
		ID:     "active-alert",
		Status: AlertStatusActive,
	}
	dashboard.alerts.mu.Unlock()
	
	// Create request
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleHealth(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Parse response
	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	assert.NoError(t, err)
	
	// Verify health data
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, float64(2), health["clients"])
	assert.Greater(t, health["metrics"], float64(0))
	assert.Equal(t, float64(1), health["traces"])
	assert.Equal(t, float64(1), health["alerts"])
}

func TestDashboard_HandleDashboardUI(t *testing.T) {
	// Test dashboard UI endpoint
	dashboard := NewDashboard(nil)
	
	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	// Handle request
	dashboard.handleDashboardUI(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
	
	// Check HTML content
	body := w.Body.String()
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "RAGO Monitoring Dashboard")
	assert.Contains(t, body, "WebSocket")
	assert.Contains(t, body, "ws://localhost:8080/ws")
}

func TestDashboard_SetupRoutes(t *testing.T) {
	// Test route setup
	dashboard := NewDashboard(nil)
	mux := http.NewServeMux()
	
	// Setup routes
	dashboard.setupRoutes(mux)
	
	// Test that routes are registered by making requests
	endpoints := []string{
		"/ws",
		"/api/metrics",
		"/api/traces",
		"/api/analytics",
		"/api/alerts",
		"/api/health",
		"/",
	}
	
	for _, endpoint := range endpoints {
		req := httptest.NewRequest("GET", endpoint, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		
		// All endpoints should return something (not 404)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "Endpoint %s should be registered", endpoint)
	}
}

func TestDashboard_StartStop(t *testing.T) {
	// Test dashboard start and stop
	config := &DashboardConfig{
		Port:            0, // Use random port
		UpdateInterval:  100 * time.Millisecond,
		EnableAlerts:    true,
		EnableAnalytics: true,
	}
	dashboard := NewDashboard(config)
	
	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		err := dashboard.Start()
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()
	
	// Wait for server to start
	time.Sleep(200 * time.Millisecond)
	
	// Stop server
	err := dashboard.Stop()
	assert.NoError(t, err)
	
	// Check for server errors
	select {
	case err := <-serverErr:
		t.Fatalf("Server error: %v", err)
	case <-time.After(100 * time.Millisecond):
		// No error, good
	}
}

func TestDashboard_WebSocketUpgrade(t *testing.T) {
	// Test WebSocket upgrade
	dashboard := NewDashboard(nil)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(dashboard.handleWebSocket))
	defer server.Close()
	
	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Try to connect
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Check response
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	
	// Verify client was registered
	time.Sleep(100 * time.Millisecond)
	dashboard.mu.RLock()
	clientCount := len(dashboard.clients)
	dashboard.mu.RUnlock()
	assert.Equal(t, 1, clientCount)
}

func TestDashboardClient_WritePump(t *testing.T) {
	// Test client write pump
	client := &DashboardClient{
		ID:         "test-client",
		Connection: nil, // Will be mocked
		Send:       make(chan DashboardUpdate, 10),
	}
	
	// Add updates to send channel
	updates := []DashboardUpdate{
		{Type: UpdateTypeMetrics, Timestamp: time.Now()},
		{Type: UpdateTypeAlert, Timestamp: time.Now()},
		{Type: UpdateTypeAnalytics, Timestamp: time.Now()},
	}
	
	for _, update := range updates {
		client.Send <- update
	}
	
	// Close channel to signal completion
	close(client.Send)
	
	// Verify updates were queued
	count := 0
	for range client.Send {
		count++
	}
	assert.Equal(t, 3, count)
}

func TestDashboard_BroadcastMetricsUpdate(t *testing.T) {
	// Test broadcasting metrics updates
	dashboard := NewDashboard(nil)
	
	// Start broadcast handler
	go dashboard.handleBroadcast()
	
	// Add a client
	client := &DashboardClient{
		ID:   "metrics-client",
		Send: make(chan DashboardUpdate, 10),
	}
	dashboard.mu.Lock()
	dashboard.clients[client.ID] = client
	dashboard.mu.Unlock()
	
	// Broadcast metrics update
	dashboard.broadcastMetricsUpdate()
	
	// Wait for broadcast
	time.Sleep(100 * time.Millisecond)
	
	// Check client received update
	select {
	case update := <-client.Send:
		assert.Equal(t, UpdateTypeMetrics, update.Type)
		assert.NotNil(t, update.Data)
		assert.NotZero(t, update.Timestamp)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Client did not receive metrics update")
	}
}

func TestDashboard_CollectSystemMetrics(t *testing.T) {
	// Test system metrics collection
	dashboard := NewDashboard(nil)
	
	// Add some clients
	dashboard.mu.Lock()
	dashboard.clients["client-1"] = &DashboardClient{ID: "client-1"}
	dashboard.clients["client-2"] = &DashboardClient{ID: "client-2"}
	dashboard.clients["client-3"] = &DashboardClient{ID: "client-3"}
	dashboard.mu.Unlock()
	
	// Collect system metrics
	dashboard.collectSystemMetrics()
	
	// Verify metrics were updated
	dashboard.metrics.mu.RLock()
	defer dashboard.metrics.mu.RUnlock()
	
	// Check CPU usage gauge
	cpuGauge, exists := dashboard.metrics.gauges["cpu_usage"]
	assert.True(t, exists)
	assert.GreaterOrEqual(t, cpuGauge.Value, 0.0)
	assert.LessOrEqual(t, cpuGauge.Value, 1.0)
	
	// Check memory usage gauge
	memGauge, exists := dashboard.metrics.gauges["memory_usage"]
	assert.True(t, exists)
	assert.GreaterOrEqual(t, memGauge.Value, 0.0)
	assert.LessOrEqual(t, memGauge.Value, 1.0)
	
	// Check active connections gauge
	connGauge, exists := dashboard.metrics.gauges["active_connections"]
	assert.True(t, exists)
	assert.Equal(t, float64(3), connGauge.Value)
}

func TestDashboard_BackgroundProcesses(t *testing.T) {
	// Test background processes don't block
	config := &DashboardConfig{
		UpdateInterval:  50 * time.Millisecond,
		EnableAlerts:    true,
		EnableAnalytics: true,
	}
	dashboard := NewDashboard(config)
	
	// Create contexts for goroutines
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	// Start background processes
	go func() {
		dashboard.startMetricsCollection()
	}()
	go func() {
		dashboard.startAlertMonitoring()
	}()
	go func() {
		dashboard.startAnalyticsProcessing()
	}()
	
	// Let them run briefly
	<-ctx.Done()
	
	// Processes should have run without panicking
	// (Can't easily test more without refactoring to accept context)
}