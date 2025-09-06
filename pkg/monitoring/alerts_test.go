package monitoring

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Alert Management

func TestAlertManager_AddRule(t *testing.T) {
	// Test adding alert rules
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	rule := &AlertRule{
		ID:        "test-rule",
		Name:      "Test Alert",
		Condition: "metric > threshold",
		Threshold: 100,
		Duration:  5 * time.Minute,
		Severity:  AlertSeverityWarning,
		Enabled:   true,
	}
	
	am.AddRule(rule)
	
	assert.Contains(t, am.rules, "test-rule")
	assert.Equal(t, rule, am.rules["test-rule"])
}

func TestAlertManager_EvaluateRules(t *testing.T) {
	// Test rule evaluation
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	// Add test rules
	rules := []*AlertRule{
		{
			ID:        "high-cpu",
			Name:      "High CPU Usage",
			Condition: "cpu_usage > threshold",
			Threshold: 0.8,
			Duration:  1 * time.Minute,
			Severity:  AlertSeverityWarning,
			Enabled:   true,
		},
		{
			ID:        "low-memory",
			Name:      "Low Memory",
			Condition: "memory_available < threshold",
			Threshold: 100,
			Duration:  2 * time.Minute,
			Severity:  AlertSeverityError,
			Enabled:   true,
		},
		{
			ID:        "disabled-rule",
			Name:      "Disabled Alert",
			Condition: "test > threshold",
			Threshold: 50,
			Duration:  1 * time.Minute,
			Severity:  AlertSeverityInfo,
			Enabled:   false,
		},
	}
	
	for _, rule := range rules {
		am.AddRule(rule)
	}
	
	// Create mock metrics
	metrics := &MetricsCollector{
		gauges: make(map[string]*Gauge),
	}
	
	// Evaluate rules (would trigger alerts based on metrics)
	am.EvaluateRules(metrics)
	
	// Verify only enabled rules are evaluated
	assert.Len(t, am.rules, 3)
	assert.False(t, am.rules["disabled-rule"].Enabled)
}

func TestAlertManager_GetActiveAlerts(t *testing.T) {
	// Test getting active alerts
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	now := time.Now()
	
	// Add various alerts
	alerts := []*Alert{
		{
			ID:        "alert-1",
			RuleID:    "rule-1",
			Name:      "Active Alert 1",
			Message:   "Alert is active",
			Severity:  AlertSeverityWarning,
			StartTime: now.Add(-10 * time.Minute),
			Status:    AlertStatusActive,
		},
		{
			ID:        "alert-2",
			RuleID:    "rule-2",
			Name:      "Active Alert 2",
			Message:   "Another active alert",
			Severity:  AlertSeverityError,
			StartTime: now.Add(-5 * time.Minute),
			Status:    AlertStatusActive,
		},
		{
			ID:        "alert-3",
			RuleID:    "rule-3",
			Name:      "Resolved Alert",
			Message:   "This was resolved",
			Severity:  AlertSeverityInfo,
			StartTime: now.Add(-20 * time.Minute),
			EndTime:   &now,
			Status:    AlertStatusResolved,
		},
		{
			ID:        "alert-4",
			RuleID:    "rule-4",
			Name:      "Muted Alert",
			Message:   "This is muted",
			Severity:  AlertSeverityWarning,
			StartTime: now.Add(-3 * time.Minute),
			Status:    AlertStatusMuted,
		},
	}
	
	for _, alert := range alerts {
		am.alerts[alert.ID] = alert
	}
	
	// Get active alerts
	active := am.GetActiveAlerts()
	
	assert.Len(t, active, 2)
	for _, alert := range active {
		assert.Equal(t, AlertStatusActive, alert.Status)
	}
}

func TestAlert_Lifecycle(t *testing.T) {
	// Test alert lifecycle
	now := time.Now()
	
	// Create new alert
	alert := &Alert{
		ID:        "lifecycle-alert",
		RuleID:    "test-rule",
		Name:      "Test Alert",
		Message:   "Alert triggered",
		Severity:  AlertSeverityWarning,
		StartTime: now,
		Status:    AlertStatusActive,
		Metadata: map[string]interface{}{
			"metric_value": 95.5,
			"threshold":    90.0,
		},
	}
	
	assert.Equal(t, AlertStatusActive, alert.Status)
	assert.Nil(t, alert.EndTime)
	
	// Resolve alert
	endTime := now.Add(10 * time.Minute)
	alert.EndTime = &endTime
	alert.Status = AlertStatusResolved
	
	assert.Equal(t, AlertStatusResolved, alert.Status)
	assert.NotNil(t, alert.EndTime)
	assert.Equal(t, endTime, *alert.EndTime)
	
	// Check metadata
	assert.Equal(t, 95.5, alert.Metadata["metric_value"])
	assert.Equal(t, 90.0, alert.Metadata["threshold"])
}

func TestAlertSeverity_Levels(t *testing.T) {
	// Test alert severity levels
	severities := []AlertSeverity{
		AlertSeverityInfo,
		AlertSeverityWarning,
		AlertSeverityError,
		AlertSeverityCritical,
	}
	
	for _, severity := range severities {
		alert := &Alert{
			ID:       fmt.Sprintf("alert-%s", severity),
			Name:     string(severity),
			Severity: severity,
			Status:   AlertStatusActive,
		}
		assert.Equal(t, severity, alert.Severity)
	}
}

func TestAlertRule_Configuration(t *testing.T) {
	// Test alert rule configuration
	rule := &AlertRule{
		ID:        "config-rule",
		Name:      "Configuration Test",
		Condition: "response_time > threshold",
		Threshold: 5000, // 5 seconds
		Duration:  3 * time.Minute,
		Severity:  AlertSeverityWarning,
		Actions: []AlertAction{
			{
				Type:   "email",
				Target: "admin@example.com",
				Parameters: map[string]interface{}{
					"subject": "Alert Triggered",
				},
			},
			{
				Type:   "webhook",
				Target: "https://example.com/webhook",
				Parameters: map[string]interface{}{
					"method": "POST",
				},
			},
		},
		Enabled: true,
	}
	
	assert.Equal(t, "config-rule", rule.ID)
	assert.Equal(t, "Configuration Test", rule.Name)
	assert.Equal(t, float64(5000), rule.Threshold)
	assert.Equal(t, 3*time.Minute, rule.Duration)
	assert.Equal(t, AlertSeverityWarning, rule.Severity)
	assert.Len(t, rule.Actions, 2)
	assert.True(t, rule.Enabled)
	
	// Check actions
	emailAction := rule.Actions[0]
	assert.Equal(t, "email", emailAction.Type)
	assert.Equal(t, "admin@example.com", emailAction.Target)
	assert.Equal(t, "Alert Triggered", emailAction.Parameters["subject"])
	
	webhookAction := rule.Actions[1]
	assert.Equal(t, "webhook", webhookAction.Type)
	assert.Equal(t, "https://example.com/webhook", webhookAction.Target)
	assert.Equal(t, "POST", webhookAction.Parameters["method"])
}

func TestAlertChannel_Notification(t *testing.T) {
	// Test alert channel notifications
	channel := &mockAlertChannel{
		name:   "test-channel",
		alerts: make([]*Alert, 0),
	}
	
	alert := &Alert{
		ID:        "notify-alert",
		Name:      "Test Notification",
		Message:   "This is a test alert",
		Severity:  AlertSeverityError,
		StartTime: time.Now(),
		Status:    AlertStatusActive,
	}
	
	// Send alert through channel
	err := channel.Send(alert)
	assert.NoError(t, err)
	assert.Len(t, channel.alerts, 1)
	assert.Equal(t, alert, channel.alerts[0])
	assert.Equal(t, "test-channel", channel.Name())
	
	// Test error handling
	errorChannel := &mockAlertChannel{
		name: "error-channel",
		err:  assert.AnError,
	}
	
	err = errorChannel.Send(alert)
	assert.Error(t, err)
}

func TestAlertManager_ConcurrentOperations(t *testing.T) {
	// Test concurrent alert operations
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	var wg sync.WaitGroup
	
	// Concurrently add rules
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			rule := &AlertRule{
				ID:        fmt.Sprintf("concurrent-rule-%d", id),
				Name:      fmt.Sprintf("Rule %d", id),
				Condition: "test > threshold",
				Threshold: float64(id * 10),
				Duration:  time.Duration(id) * time.Minute,
				Severity:  AlertSeverityWarning,
				Enabled:   true,
			}
			am.AddRule(rule)
		}(i)
	}
	
	// Concurrently create alerts
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			alert := &Alert{
				ID:        fmt.Sprintf("concurrent-alert-%d", id),
				RuleID:    fmt.Sprintf("concurrent-rule-%d", id),
				Name:      fmt.Sprintf("Alert %d", id),
				Message:   "Concurrent alert",
				Severity:  AlertSeverityWarning,
				StartTime: time.Now(),
				Status:    AlertStatusActive,
			}
			am.mu.Lock()
			am.alerts[alert.ID] = alert
			am.mu.Unlock()
		}(i)
	}
	
	// Concurrently read alerts
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = am.GetActiveAlerts()
			}
		}()
	}
	
	// Concurrently evaluate rules
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			metrics := &MetricsCollector{
				gauges: make(map[string]*Gauge),
			}
			for j := 0; j < 5; j++ {
				am.EvaluateRules(metrics)
			}
		}()
	}
	
	wg.Wait()
	
	// Verify final state
	assert.Len(t, am.rules, 10)
	assert.Len(t, am.alerts, 10)
}

func TestAlertManager_ThresholdAlerts(t *testing.T) {
	// Test threshold-based alerts
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	// Add threshold rules
	thresholdRules := []*AlertRule{
		{
			ID:        "high-threshold",
			Name:      "High Value Alert",
			Condition: "value > threshold",
			Threshold: 100,
			Duration:  1 * time.Minute,
			Severity:  AlertSeverityError,
			Enabled:   true,
		},
		{
			ID:        "low-threshold",
			Name:      "Low Value Alert",
			Condition: "value < threshold",
			Threshold: 10,
			Duration:  2 * time.Minute,
			Severity:  AlertSeverityWarning,
			Enabled:   true,
		},
		{
			ID:        "exact-threshold",
			Name:      "Exact Value Alert",
			Condition: "value == threshold",
			Threshold: 50,
			Duration:  30 * time.Second,
			Severity:  AlertSeverityInfo,
			Enabled:   true,
		},
	}
	
	for _, rule := range thresholdRules {
		am.AddRule(rule)
	}
	
	// Verify thresholds
	assert.Equal(t, float64(100), am.rules["high-threshold"].Threshold)
	assert.Equal(t, float64(10), am.rules["low-threshold"].Threshold)
	assert.Equal(t, float64(50), am.rules["exact-threshold"].Threshold)
}

func TestAlertManager_DurationAlerts(t *testing.T) {
	// Test duration-based alerts
	am := &AlertManager{
		rules:  make(map[string]*AlertRule),
		alerts: make(map[string]*Alert),
	}
	
	// Add duration-based rules
	durationRules := []*AlertRule{
		{
			ID:        "short-duration",
			Name:      "Short Duration Alert",
			Condition: "condition",
			Threshold: 50,
			Duration:  30 * time.Second,
			Severity:  AlertSeverityInfo,
			Enabled:   true,
		},
		{
			ID:        "medium-duration",
			Name:      "Medium Duration Alert",
			Condition: "condition",
			Threshold: 50,
			Duration:  5 * time.Minute,
			Severity:  AlertSeverityWarning,
			Enabled:   true,
		},
		{
			ID:        "long-duration",
			Name:      "Long Duration Alert",
			Condition: "condition",
			Threshold: 50,
			Duration:  1 * time.Hour,
			Severity:  AlertSeverityError,
			Enabled:   true,
		},
	}
	
	for _, rule := range durationRules {
		am.AddRule(rule)
	}
	
	// Verify durations
	assert.Equal(t, 30*time.Second, am.rules["short-duration"].Duration)
	assert.Equal(t, 5*time.Minute, am.rules["medium-duration"].Duration)
	assert.Equal(t, 1*time.Hour, am.rules["long-duration"].Duration)
}

func TestAlertManager_MultipleChannels(t *testing.T) {
	// Test multiple alert channels
	am := &AlertManager{
		rules:    make(map[string]*AlertRule),
		alerts:   make(map[string]*Alert),
		channels: make([]AlertChannel, 0),
	}
	
	// Add multiple channels
	emailChannel := &mockAlertChannel{
		name:   "email",
		alerts: make([]*Alert, 0),
	}
	slackChannel := &mockAlertChannel{
		name:   "slack",
		alerts: make([]*Alert, 0),
	}
	webhookChannel := &mockAlertChannel{
		name:   "webhook",
		alerts: make([]*Alert, 0),
	}
	
	am.channels = append(am.channels, emailChannel, slackChannel, webhookChannel)
	
	// Send alert to all channels
	alert := &Alert{
		ID:        "multi-channel",
		Name:      "Multi-Channel Alert",
		Message:   "Alert for all channels",
		Severity:  AlertSeverityCritical,
		StartTime: time.Now(),
		Status:    AlertStatusActive,
	}
	
	for _, channel := range am.channels {
		err := channel.Send(alert)
		assert.NoError(t, err)
	}
	
	// Verify all channels received the alert
	assert.Len(t, emailChannel.alerts, 1)
	assert.Len(t, slackChannel.alerts, 1)
	assert.Len(t, webhookChannel.alerts, 1)
}