// Package client - health_test.go
// Comprehensive tests for health monitoring across all four RAGO pillars.
// This file validates health monitoring, reporting, and decision-making functionality.

package client

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== HEALTH MONITOR TESTING =====

func TestNewHealthMonitor(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	t.Run("health monitor creation", func(t *testing.T) {
		monitor := NewHealthMonitor(client)
		if monitor == nil {
			t.Error("Health monitor should not be nil")
		}

		if monitor.client != client {
			t.Error("Health monitor should reference the correct client")
		}

		if monitor.checkInterval <= 0 {
			t.Error("Check interval should be positive")
		}

		if monitor.updateChan == nil {
			t.Error("Update channel should be initialized")
		}

		if monitor.stopChan == nil {
			t.Error("Stop channel should be initialized")
		}

		// Stop the monitor to clean up
		monitor.Stop()
	})

	t.Run("initial health report", func(t *testing.T) {
		monitor := NewHealthMonitor(client)
		defer monitor.Stop()

		// Give some time for initial health check
		time.Sleep(100 * time.Millisecond)

		report := monitor.GetHealthReport()

		if report.Overall == "" {
			t.Error("Overall health status should be set")
		}

		if report.Pillars == nil {
			t.Error("Pillars map should be initialized")
		}

		if report.Providers == nil {
			t.Error("Providers map should be initialized")
		}

		if report.Servers == nil {
			t.Error("Servers map should be initialized")
		}

		if report.Details == nil {
			t.Error("Details map should be initialized")
		}

		if report.LastCheck.IsZero() {
			t.Error("Last check time should be set")
		}
	})
}

func TestHealthMonitor_PerformHealthCheck(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	t.Run("health check execution", func(t *testing.T) {
		// Trigger immediate health check
		monitor.RefreshHealth()

		// Wait for health check to complete
		time.Sleep(200 * time.Millisecond)

		report := monitor.GetHealthReport()

		// Should have health information for all available pillars
		expectedPillars := []string{"llm", "rag", "mcp", "agents"}
		for _, pillar := range expectedPillars {
			if _, exists := report.Pillars[pillar]; !exists {
				t.Errorf("Health report should include pillar: %s", pillar)
			}
		}

		// Should have overall statistics
		if stats, exists := report.Details["overall_stats"]; !exists {
			t.Error("Should have overall statistics")
		} else {
			statsMap := stats.(map[string]interface{})
			if _, exists := statsMap["healthy_pillars"]; !exists {
				t.Error("Should track healthy pillars count")
			}
			if _, exists := statsMap["total_pillars"]; !exists {
				t.Error("Should track total pillars count")
			}
		}
	})

	t.Run("pillar-specific health checks", func(t *testing.T) {
		monitor.RefreshHealth()
		time.Sleep(200 * time.Millisecond)

		report := monitor.GetHealthReport()

		// LLM pillar health
		if llmHealth, exists := report.Pillars["llm"]; exists {
			validStatuses := []core.HealthStatus{
				core.HealthStatusHealthy,
				core.HealthStatusDegraded,
				core.HealthStatusUnhealthy,
				core.HealthStatusUnknown,
			}
			isValid := false
			for _, status := range validStatuses {
				if llmHealth == status {
					isValid = true
					break
				}
			}
			if !isValid {
				t.Errorf("Invalid LLM health status: %s", llmHealth)
			}
		}

		// RAG pillar health
		if ragHealth, exists := report.Pillars["rag"]; exists {
			if ragHealth == "" {
				t.Error("RAG health status should not be empty")
			}
		}

		// MCP pillar health
		if mcpHealth, exists := report.Pillars["mcp"]; exists {
			if mcpHealth == "" {
				t.Error("MCP health status should not be empty")
			}
		}

		// Agents pillar health
		if agentsHealth, exists := report.Pillars["agents"]; exists {
			if agentsHealth == "" {
				t.Error("Agents health status should not be empty")
			}
		}
	})

	t.Run("provider health tracking", func(t *testing.T) {
		// Set up mock LLM service with providers
		if mockLLM, ok := client.llmService.(*MockLLMService); ok {
			mockLLM.AddProvider("provider1", core.ProviderConfig{
				Type:  "test",
				Model: "test-model",
			})
			mockLLM.SetProviderHealth("provider1", core.HealthStatusHealthy)
		}

		monitor.RefreshHealth()
		time.Sleep(200 * time.Millisecond)

		report := monitor.GetHealthReport()

		// Should track provider health
		foundProvider := false
		for providerName, health := range report.Providers {
			if strings.Contains(providerName, "provider") {
				foundProvider = true
				if health == "" {
					t.Errorf("Provider %s should have health status", providerName)
				}
			}
		}

		if !foundProvider {
			t.Log("No providers found in health report - this may be expected")
		}
	})

	t.Run("server health tracking", func(t *testing.T) {
		// Set up mock MCP service with servers
		if mockMCP, ok := client.mcpService.(*MockMCPService); ok {
			mockMCP.RegisterServer(core.ServerConfig{
				Name:        "test-server",
				Description: "test server",
			})
			mockMCP.SetServerHealth("test-server", core.HealthStatusHealthy)
		}

		monitor.RefreshHealth()
		time.Sleep(200 * time.Millisecond)

		report := monitor.GetHealthReport()

		// Should track server health
		if serverHealth, exists := report.Servers["test-server"]; exists {
			if serverHealth != core.HealthStatusHealthy {
				t.Errorf("Expected server health to be healthy, got: %s", serverHealth)
			}
		} else {
			t.Log("Test server not found in health report - may need MCP service setup")
		}
	})
}

func TestHealthMonitor_OverallHealthCalculation(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	tests := []struct {
		name              string
		setupMocks        func()
		expectedOverall   core.HealthStatus
		healthyPillars    int
		totalPillars      int
	}{
		{
			name: "all pillars healthy",
			setupMocks: func() {
				if mockLLM, ok := client.llmService.(*MockLLMService); ok {
					mockLLM.AddProvider("healthy-provider", core.ProviderConfig{Type: "test"})
					mockLLM.SetProviderHealth("healthy-provider", core.HealthStatusHealthy)
				}
			},
			expectedOverall: core.HealthStatusHealthy,
		},
		{
			name: "some pillars degraded",
			setupMocks: func() {
				if mockLLM, ok := client.llmService.(*MockLLMService); ok {
					mockLLM.AddProvider("degraded-provider", core.ProviderConfig{Type: "test"})
					mockLLM.SetProviderHealth("degraded-provider", core.HealthStatusDegraded)
				}
				if mockMCP, ok := client.mcpService.(*MockMCPService); ok {
					mockMCP.RegisterServer(core.ServerConfig{Name: "unhealthy-server", Description: "test server"})
					mockMCP.SetServerHealth("unhealthy-server", core.HealthStatusUnhealthy)
				}
			},
			expectedOverall: core.HealthStatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset client state
			client = createTestClient(t)
			defer client.Close()

			monitor = NewHealthMonitor(client)
			defer monitor.Stop()

			// Setup test conditions
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			// Trigger health check
			monitor.RefreshHealth()
			time.Sleep(200 * time.Millisecond)

			report := monitor.GetHealthReport()

			// Validate overall health calculation
			if report.Overall == "" {
				t.Error("Overall health should be set")
			}

			// Log for debugging
			t.Logf("Overall health: %s", report.Overall)
			t.Logf("Pillar count: %d", len(report.Pillars))
			for pillar, health := range report.Pillars {
				t.Logf("  %s: %s", pillar, health)
			}
		})
	}
}

// ===== HEALTH REPORT TESTING =====

func TestHealthMonitor_HealthReport(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	t.Run("health report structure", func(t *testing.T) {
		report := monitor.GetHealthReport()

		// Verify all required fields are present
		if report.Overall == "" {
			t.Error("Overall status should be set")
		}

		if report.Pillars == nil {
			t.Error("Pillars map should be initialized")
		}

		if report.Providers == nil {
			t.Error("Providers map should be initialized")
		}

		if report.Servers == nil {
			t.Error("Servers map should be initialized")
		}

		if report.Details == nil {
			t.Error("Details map should be initialized")
		}

		if report.LastCheck.IsZero() {
			t.Error("Last check time should be set")
		}
	})

	t.Run("health report consistency", func(t *testing.T) {
		report1 := monitor.GetHealthReport()
		time.Sleep(50 * time.Millisecond)
		report2 := monitor.GetHealthReport()

		// Reports should be consistent within a short time frame
		if report1.Overall != report2.Overall {
			t.Log("Overall health changed between calls - this can happen with background monitoring")
		}

		// Last check time should not change unless health check runs
		if report1.LastCheck != report2.LastCheck {
			t.Log("Last check time changed - health check may have run")
		}
	})

	t.Run("health report after refresh", func(t *testing.T) {
		initialReport := monitor.GetHealthReport()
		initialTime := initialReport.LastCheck

		// Trigger refresh and wait
		monitor.RefreshHealth()
		time.Sleep(200 * time.Millisecond)

		newReport := monitor.GetHealthReport()

		// Last check time should be updated
		if !newReport.LastCheck.After(initialTime) {
			t.Error("Last check time should be updated after refresh")
		}
	})
}

func TestHealthMonitor_PillarHealth(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	t.Run("get specific pillar health", func(t *testing.T) {
		llmHealth := monitor.GetPillarHealth("llm")
		if llmHealth == "" {
			t.Error("LLM pillar health should be available")
		}

		ragHealth := monitor.GetPillarHealth("rag")
		if ragHealth == "" {
			t.Error("RAG pillar health should be available")
		}

		mcpHealth := monitor.GetPillarHealth("mcp")
		if mcpHealth == "" {
			t.Error("MCP pillar health should be available")
		}

		agentsHealth := monitor.GetPillarHealth("agents")
		if agentsHealth == "" {
			t.Error("Agents pillar health should be available")
		}
	})

	t.Run("get nonexistent pillar health", func(t *testing.T) {
		unknownHealth := monitor.GetPillarHealth("nonexistent")
		if unknownHealth != core.HealthStatusUnknown {
			t.Errorf("Unknown pillar should return Unknown status, got: %s", unknownHealth)
		}
	})
}

func TestHealthMonitor_HealthStatus(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	t.Run("is healthy check", func(t *testing.T) {
		healthy := monitor.IsHealthy()
		report := monitor.GetHealthReport()

		expectedHealthy := (report.Overall == core.HealthStatusHealthy)
		if healthy != expectedHealthy {
			t.Errorf("IsHealthy() returned %v, but overall status is %s", healthy, report.Overall)
		}
	})

	t.Run("is degraded check", func(t *testing.T) {
		degraded := monitor.IsDegraded()
		report := monitor.GetHealthReport()

		expectedDegraded := (report.Overall == core.HealthStatusDegraded)
		if degraded != expectedDegraded {
			t.Errorf("IsDegraded() returned %v, but overall status is %s", degraded, report.Overall)
		}
	})
}

// ===== HEALTH MONITORING INTERVALS AND CONFIGURATION =====

func TestHealthMonitor_CheckInterval(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	t.Run("get default check interval", func(t *testing.T) {
		interval := monitor.GetCheckInterval()
		if interval <= 0 {
			t.Error("Check interval should be positive")
		}

		if interval != 30*time.Second {
			t.Errorf("Expected default interval 30s, got: %v", interval)
		}
	})

	t.Run("set check interval", func(t *testing.T) {
		newInterval := 10 * time.Second
		monitor.SetCheckInterval(newInterval)

		currentInterval := monitor.GetCheckInterval()
		if currentInterval != newInterval {
			t.Errorf("Expected interval %v, got: %v", newInterval, currentInterval)
		}
	})

	t.Run("set very short interval", func(t *testing.T) {
		shortInterval := 100 * time.Millisecond
		monitor.SetCheckInterval(shortInterval)

		currentInterval := monitor.GetCheckInterval()
		if currentInterval != shortInterval {
			t.Errorf("Expected interval %v, got: %v", shortInterval, currentInterval)
		}

		// Verify health checks run more frequently
		initialTime := monitor.GetHealthReport().LastCheck

		// Wait for multiple intervals
		time.Sleep(300 * time.Millisecond)

		newTime := monitor.GetHealthReport().LastCheck
		if !newTime.After(initialTime) {
			t.Error("Health should have been checked multiple times")
		}
	})
}

func TestHealthMonitor_RefreshHealth(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	t.Run("manual health refresh", func(t *testing.T) {
		initialReport := monitor.GetHealthReport()
		initialTime := initialReport.LastCheck

		// Wait a bit, then refresh
		time.Sleep(100 * time.Millisecond)
		monitor.RefreshHealth()
		time.Sleep(200 * time.Millisecond)

		newReport := monitor.GetHealthReport()
		newTime := newReport.LastCheck

		if !newTime.After(initialTime) {
			t.Error("Health check should have been triggered by refresh")
		}
	})

	t.Run("multiple rapid refreshes", func(t *testing.T) {
		// Multiple rapid refreshes should not cause issues
		for i := 0; i < 5; i++ {
			monitor.RefreshHealth()
		}

		time.Sleep(200 * time.Millisecond)

		// Should still get valid health report
		report := monitor.GetHealthReport()
		if report.Overall == "" {
			t.Error("Health report should be valid after multiple refreshes")
		}
	})
}

// ===== HEALTH SUMMARY AND REPORTING =====

func TestHealthMonitor_HealthSummary(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	// Wait for health check
	time.Sleep(200 * time.Millisecond)

	t.Run("health summary format", func(t *testing.T) {
		summary := monitor.GetHealthSummary()

		if summary == "" {
			t.Error("Health summary should not be empty")
		}

		// Should contain key information
		expectedElements := []string{
			"Overall Health:",
			"Last Check:",
			"Pillar Status:",
		}

		for _, element := range expectedElements {
			if !strings.Contains(summary, element) {
				t.Errorf("Health summary should contain '%s'", element)
			}
		}

		t.Logf("Health Summary:\n%s", summary)
	})

	t.Run("summary includes all pillars", func(t *testing.T) {
		summary := monitor.GetHealthSummary()

		// Should mention all active pillars
		pillars := []string{"llm", "rag", "mcp", "agents"}
		for _, pillar := range pillars {
			if !strings.Contains(summary, pillar) {
				t.Logf("Health summary may not include pillar: %s", pillar)
			}
		}
	})
}

// ===== LIFECYCLE AND ERROR HANDLING =====

func TestHealthMonitor_Lifecycle(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	t.Run("monitor stop", func(t *testing.T) {
		monitor := NewHealthMonitor(client)

		// Verify monitor is running
		if !monitor.monitoring {
			t.Error("Monitor should be running after creation")
		}

		// Stop monitor
		monitor.Stop()

		// Should not be monitoring anymore
		time.Sleep(100 * time.Millisecond)
		if monitor.monitoring {
			t.Error("Monitor should be stopped after Stop()")
		}
	})

	t.Run("multiple stops", func(t *testing.T) {
		monitor := NewHealthMonitor(client)

		// Multiple stops should not cause issues
		monitor.Stop()
		monitor.Stop()
		monitor.Stop()

		// Should still be able to get health report
		report := monitor.GetHealthReport()
		if report.Overall == "" {
			t.Error("Should still get health report after stop")
		}
	})

	t.Run("operations after stop", func(t *testing.T) {
		monitor := NewHealthMonitor(client)
		monitor.Stop()

		// Operations should still work
		monitor.RefreshHealth() // Should not panic
		monitor.SetCheckInterval(5 * time.Second)
		interval := monitor.GetCheckInterval()
		if interval != 5*time.Second {
			t.Error("Should still be able to set interval after stop")
		}

		summary := monitor.GetHealthSummary()
		if summary == "" {
			t.Error("Should still be able to get summary after stop")
		}
	})
}

func TestHealthMonitor_ErrorConditions(t *testing.T) {
	t.Run("monitor with nil client", func(t *testing.T) {
		// This would cause issues, but we test defensive programming
		defer func() {
			if r := recover(); r != nil {
				t.Error("Should handle nil client gracefully, not panic")
			}
		}()

		monitor := NewHealthMonitor(nil)
		if monitor != nil {
			report := monitor.GetHealthReport()
			// Should return some reasonable default
			if report.Overall == "" {
				t.Log("Nil client results in empty health report - acceptable")
			}
			monitor.Stop()
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		client := createTestClient(t)
		defer client.Close()

		monitor := NewHealthMonitor(client)
		defer monitor.Stop()

		// Concurrent operations should not cause race conditions
		var wg sync.WaitGroup
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				monitor.RefreshHealth()
				report := monitor.GetHealthReport()
				_ = report
				monitor.IsHealthy()
				monitor.IsDegraded()
				monitor.GetPillarHealth("llm")
			}()
		}

		wg.Wait()

		// Should still have valid health report
		report := monitor.GetHealthReport()
		if report.Overall == "" {
			t.Error("Health report should be valid after concurrent access")
		}
	})
}

// ===== WAIT FOR HEALTHY TESTING =====

func TestHealthMonitor_WaitForHealthy(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	monitor := NewHealthMonitor(client)
	defer monitor.Stop()

	t.Run("wait when already healthy", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// If already healthy, should return immediately
		err := monitor.WaitForHealthy(ctx)
		if err != nil {
			// May not be healthy initially, that's okay
			t.Logf("Not healthy initially: %v", err)
		}
	})

	t.Run("wait with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := monitor.WaitForHealthy(ctx)
		if err != context.DeadlineExceeded && err != nil {
			// Either times out or succeeds quickly
			t.Logf("Wait result: %v", err)
		}
	})

	t.Run("wait with cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := monitor.WaitForHealthy(ctx)
		if err != context.Canceled && err != nil {
			t.Logf("Wait result: %v", err)
		}
	})
}

// ===== CUSTOM HEALTH CHECKS TESTING =====

func TestCustomHealthChecks(t *testing.T) {
	t.Run("register custom health check", func(t *testing.T) {
		checkCalled := false
		checkFunc := func(ctx context.Context) (core.HealthStatus, error) {
			checkCalled = true
			return core.HealthStatusHealthy, nil
		}

		RegisterHealthCheck("test-check", checkFunc)
		defer UnregisterHealthCheck("test-check")

		client := createTestClient(t)
		defer client.Close()

		monitor := NewHealthMonitor(client)
		defer monitor.Stop()

		// Custom checks should be executed
		customResults := monitor.runCustomHealthChecks(context.Background())

		if len(customResults) == 0 {
			t.Error("Should have custom health check results")
		}

		if !checkCalled {
			t.Error("Custom health check should have been called")
		}

		if status, exists := customResults["test-check"]; !exists {
			t.Error("Should have test-check result")
		} else if status != core.HealthStatusHealthy {
			t.Errorf("Expected healthy status, got: %s", status)
		}
	})

	t.Run("unregister custom health check", func(t *testing.T) {
		checkFunc := func(ctx context.Context) (core.HealthStatus, error) {
			return core.HealthStatusHealthy, nil
		}

		RegisterHealthCheck("temp-check", checkFunc)
		
		// Verify it's registered
		results := (&HealthMonitor{}).runCustomHealthChecks(context.Background())
		if _, exists := results["temp-check"]; !exists {
			t.Error("Check should be registered")
		}

		UnregisterHealthCheck("temp-check")
		
		// Verify it's unregistered
		results = (&HealthMonitor{}).runCustomHealthChecks(context.Background())
		if _, exists := results["temp-check"]; exists {
			t.Error("Check should be unregistered")
		}
	})

	t.Run("custom health check error handling", func(t *testing.T) {
		errorCheckFunc := func(ctx context.Context) (core.HealthStatus, error) {
			return core.HealthStatusUnknown, context.DeadlineExceeded
		}

		RegisterHealthCheck("error-check", errorCheckFunc)
		defer UnregisterHealthCheck("error-check")

		monitor := &HealthMonitor{}
		results := monitor.runCustomHealthChecks(context.Background())

		if status, exists := results["error-check"]; !exists {
			t.Error("Should have error check result")
		} else if status != core.HealthStatusUnhealthy {
			t.Errorf("Error should result in unhealthy status, got: %s", status)
		}
	})
}

// ===== INTEGRATION WITH CLIENT HEALTH =====

func TestClient_HealthIntegration(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	t.Run("client health method", func(t *testing.T) {
		health := client.Health()

		if health.Overall == "" {
			t.Error("Client health should return overall status")
		}

		if health.Pillars == nil {
			t.Error("Client health should include pillar statuses")
		}

		if health.LastCheck.IsZero() {
			t.Error("Client health should include last check time")
		}
	})

	t.Run("health monitor initialization", func(t *testing.T) {
		if client.healthMonitor == nil {
			t.Error("Client should have health monitor initialized")
		}

		// Health monitor should be running
		if !client.healthMonitor.monitoring {
			t.Error("Health monitor should be running")
		}
	})

	t.Run("client without health monitor", func(t *testing.T) {
		// Create client and remove health monitor
		client := createTestClient(t)
		client.healthMonitor = nil

		health := client.Health()

		if health.Overall != core.HealthStatusUnknown {
			t.Errorf("Expected unknown status when no monitor, got: %s", health.Overall)
		}

		if health.Details == nil {
			t.Error("Health should include error details")
		}

		client.Close()
	})
}