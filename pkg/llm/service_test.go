package llm

import (
	"context"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

func TestNewService(t *testing.T) {
	config := core.LLMConfig{
		DefaultProvider: "test",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   true,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: make(map[string]core.ProviderConfig),
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	if service == nil {
		t.Fatal("Service is nil")
	}
	
	if !service.IsStarted() {
		t.Error("Service should be started")
	}
	
	// Test service properties
	providers := service.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
	
	health := service.GetProviderHealth()
	if len(health) != 0 {
		t.Errorf("Expected 0 health entries, got %d", len(health))
	}
	
	// Test cleanup
	if err := service.Close(); err != nil {
		t.Errorf("Failed to close service: %v", err)
	}
	
	if service.IsStarted() {
		t.Error("Service should be stopped after Close()")
	}
}

func TestServiceWithProviders(t *testing.T) {
	config := core.LLMConfig{
		DefaultProvider: "test-ollama",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false, // Disable health check for testing
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  false, // Disable health check for testing
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: map[string]core.ProviderConfig{
			"test-ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
				Weight:  10,
				Timeout: 30 * time.Second,
			},
		},
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test provider listing
	providers := service.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
	
	if providers[0].Name != "test-ollama" {
		t.Errorf("Expected provider name 'test-ollama', got %s", providers[0].Name)
	}
	
	if providers[0].Type != "ollama" {
		t.Errorf("Expected provider type 'ollama', got %s", providers[0].Type)
	}
}

func TestServiceGeneration(t *testing.T) {
	// Skip this test if we don't have real providers configured
	// This is more of an integration test
	t.Skip("Skipping generation test - requires real providers")
	
	config := core.LLMConfig{
		DefaultProvider: "test-ollama",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  false,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: map[string]core.ProviderConfig{
			"test-ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
				Weight:  10,
				Timeout: 30 * time.Second,
			},
		},
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test generation
	req := core.GenerationRequest{
		Prompt:      "Hello, world!",
		MaxTokens:   100,
		Temperature: 0.7,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	response, err := service.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}
	
	if response.Content == "" {
		t.Error("Response content is empty")
	}
	
	if response.Provider == "" {
		t.Error("Response provider is empty")
	}
}

func TestServiceMetrics(t *testing.T) {
	config := core.LLMConfig{
		DefaultProvider: "test",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  false,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: make(map[string]core.ProviderConfig),
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test metrics
	metrics := service.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics is nil")
	}
	
	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", metrics.TotalRequests)
	}
	
	if metrics.ServiceUptime <= 0 {
		t.Error("Service uptime should be positive")
	}
}

func TestServiceBatchGeneration(t *testing.T) {
	// Skip this test if we don't have real providers configured
	t.Skip("Skipping batch generation test - requires real providers")
	
	config := core.LLMConfig{
		DefaultProvider: "test-ollama",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  false,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: map[string]core.ProviderConfig{
			"test-ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
				Weight:  10,
				Timeout: 30 * time.Second,
			},
		},
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test batch generation
	requests := []core.GenerationRequest{
		{
			Prompt:      "Hello, world!",
			MaxTokens:   50,
			Temperature: 0.7,
		},
		{
			Prompt:      "Tell me a joke",
			MaxTokens:   100,
			Temperature: 0.8,
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	responses, err := service.GenerateBatch(ctx, requests)
	if err != nil {
		t.Fatalf("Batch generation failed: %v", err)
	}
	
	if len(responses) != len(requests) {
		t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
	}
	
	for i, response := range responses {
		if response.Content == "" {
			t.Errorf("Response %d content is empty", i)
		}
	}
}

func TestServiceProviderManagement(t *testing.T) {
	config := core.LLMConfig{
		DefaultProvider: "test",
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false,
			CheckInterval: 30 * time.Second,
		},
		HealthCheck: core.HealthCheckConfig{
			Enabled:  false,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Providers: make(map[string]core.ProviderConfig),
	}
	
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test adding provider
	providerConfig := core.ProviderConfig{
		Type:    "ollama",
		BaseURL: "http://localhost:11434",
		Model:   "llama2",
		Weight:  10,
		Timeout: 30 * time.Second,
	}
	
	err = service.AddProvider("dynamic-ollama", providerConfig)
	if err != nil {
		t.Errorf("Failed to add provider: %v", err)
	}
	
	// Verify provider was added
	providers := service.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}
	
	// Test removing provider
	err = service.RemoveProvider("dynamic-ollama")
	if err != nil {
		t.Errorf("Failed to remove provider: %v", err)
	}
	
	// Verify provider was removed
	providers = service.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
	
	// Test removing non-existent provider
	err = service.RemoveProvider("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent provider")
	}
}