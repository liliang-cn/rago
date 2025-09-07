package client

import (
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// TestClientCreation tests basic client creation with default configuration
func TestClientCreation(t *testing.T) {
	// Test creating client with defaults
	client, err := NewWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create client with defaults: %v", err)
	}

	// Verify basic structure
	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Test accessing pillar services (should not panic)
	llmService := client.LLM()
	ragService := client.RAG()
	mcpService := client.MCP()
	agentService := client.Agents()

	// For now, some services might be nil due to configuration
	// We're just testing the interface works
	_ = llmService
	_ = ragService
	_ = mcpService
	_ = agentService

	// Test health monitoring
	health := client.Health()
	if health.LastCheck.IsZero() {
		t.Error("Health report should have a last check time")
	}

	// Test closing
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}
}

// TestBuilderPattern tests the builder pattern for custom configurations
func TestBuilderPattern(t *testing.T) {
	// Test LLM-only client
	llmConfig := core.LLMConfig{
		DefaultProvider: "test-provider",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "test-model",
				Weight:  1,
				Timeout: 30 * time.Second,
				Parameters: map[string]interface{}{
					"temperature": 0.7,
				},
			},
		},
	}

	client, err := NewBuilder().
		WithLLM(llmConfig).
		WithoutRAG().
		WithoutMCP().
		WithoutAgents().
		Build()

	if err != nil {
		t.Fatalf("Failed to create LLM-only client: %v", err)
	}

	// Verify LLM service is available
	if client.LLM() == nil {
		t.Error("LLM service should be available")
	}

	// Verify other services are nil (since disabled)
	if client.RAG() != nil {
		t.Error("RAG service should be nil when disabled")
	}

	if client.MCP() != nil {
		t.Error("MCP service should be nil when disabled")
	}

	if client.Agents() != nil {
		t.Error("Agent service should be nil when disabled")
	}

	// Test client configuration
	config := client.GetConfig()
	if config == nil {
		t.Error("Client config should not be nil")
	}

	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close LLM-only client: %v", err)
	}
}

// TestIndividualPillarClients tests individual pillar client creation
func TestIndividualPillarClients(t *testing.T) {
	// Test LLM client creation
	llmConfig := core.LLMConfig{
		DefaultProvider: "test-provider",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "test-model",
				Weight:  1,
				Timeout: 30 * time.Second,
			},
		},
	}

	llmClient, err := NewLLMClient(llmConfig)
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}

	// Test basic interface compliance
	providers := llmClient.ListProviders()
	_ = providers // Just testing it doesn't panic

	err = llmClient.Close()
	if err != nil {
		t.Fatalf("Failed to close LLM client: %v", err)
	}
}

// TestHealthMonitoring tests the health monitoring functionality
func TestHealthMonitoring(t *testing.T) {
	client, err := NewWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test initial health report
	health := client.Health()

	// Basic health report structure validation
	if health.Overall == "" {
		t.Error("Health report should have overall status")
	}

	if health.Pillars == nil {
		t.Error("Health report should have pillars map")
	}

	if health.Providers == nil {
		t.Error("Health report should have providers map")
	}

	if health.LastCheck.IsZero() {
		t.Error("Health report should have last check time")
	}

	if health.Details == nil {
		t.Error("Health report should have details map")
	}
}

// TestConfigurationValidation tests configuration validation
func TestConfigurationValidation(t *testing.T) {
	config := getDefaultConfig()

	// Test basic validation
	err := config.Validate()
	if err != nil {
		t.Fatalf("Default config should be valid: %v", err)
	}

	// Test invalid config (no pillars enabled)
	invalidConfig := &Config{
		Config: core.Config{
			Mode: core.ModeConfig{
				RAGOnly:      false,
				LLMOnly:      false,
				DisableMCP:   true,
				DisableAgent: true,
			},
		},
	}

	err = invalidConfig.Validate()
	if err == nil {
		t.Error("Invalid config should fail validation")
	}
}