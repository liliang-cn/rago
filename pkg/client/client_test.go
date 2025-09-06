package client

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

func TestClient_New(t *testing.T) {
	config := core.TestConfig()
	
	client, err := New(config)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected client to be created, got nil")
	}
}

func TestClient_PillarAccess(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	// Test LLM pillar access
	llmService := client.LLM()
	if llmService == nil {
		t.Error("Expected LLM service, got nil")
	}
	
	// Test RAG pillar access
	ragService := client.RAG()
	if ragService == nil {
		t.Error("Expected RAG service, got nil")
	}
	
	// Test MCP pillar access
	mcpService := client.MCP()
	if mcpService == nil {
		t.Error("Expected MCP service, got nil")
	}
	
	// Test Agent pillar access
	agentService := client.Agents()
	if agentService == nil {
		t.Error("Expected Agent service, got nil")
	}
}

func TestClient_Chat(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.ChatRequest{
		Message: "Hello, world!",
		UseRAG:  true,
		UseTools: true,
	}
	
	// This will return an error since we haven't implemented high-level operations yet
	_, err = client.Chat(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestClient_ProcessDocument(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.DocumentRequest{
		Action:  "analyze",
		Content: "This is a test document for analysis.",
	}
	
	// This will return an error since we haven't implemented high-level operations yet
	_, err = client.ProcessDocument(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestClient_ExecuteTask(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	ctx := context.Background()
	req := core.TaskRequest{
		Task:    "Test task execution",
		Context: map[string]interface{}{"test": true},
	}
	
	// This will return an error since we haven't implemented high-level operations yet
	_, err = client.ExecuteTask(ctx, req)
	core.AssertError(t, err) // Expecting error until implementation
}

func TestClient_Health(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	health := client.Health()
	
	// Should return a health report, even if minimal
	if health.Overall == "" {
		t.Error("Expected health report with overall status")
	}
}

func TestClient_Close(t *testing.T) {
	config := core.TestConfig()
	client, err := New(config)
	core.AssertNoError(t, err)
	
	err = client.Close()
	core.AssertNoError(t, err)
}

func TestClient_ModeConfigurations(t *testing.T) {
	// Test RAG-only mode
	ragOnlyConfig := core.TestConfig()
	ragOnlyConfig.Mode.RAGOnly = true
	
	client, err := New(ragOnlyConfig)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected RAG-only client to be created")
	}
	
	// Test LLM-only mode
	llmOnlyConfig := core.TestConfig()
	llmOnlyConfig.Mode.LLMOnly = true
	
	client, err = New(llmOnlyConfig)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected LLM-only client to be created")
	}
	
	// Test with disabled pillars
	disabledConfig := core.TestConfig()
	disabledConfig.Mode.DisableMCP = true
	disabledConfig.Mode.DisableAgent = true
	
	client, err = New(disabledConfig)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected client with disabled pillars to be created")
	}
}

func TestNewLLMClient(t *testing.T) {
	config := core.TestLLMConfig()
	
	client, err := NewLLMClient(config)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected LLM client to be created, got nil")
	}
	
	err = client.Close()
	core.AssertNoError(t, err)
}

func TestNewRAGClient(t *testing.T) {
	config := core.TestRAGConfig()
	
	client, err := NewRAGClient(config)
	core.AssertNoError(t, err)
	
	if client == nil {
		t.Error("Expected RAG client to be created, got nil")
	}
	
	err = client.Close()
	core.AssertNoError(t, err)
}

func TestConfigAdapter_LoadCoreConfig(t *testing.T) {
	adapter := NewConfigAdapter("")
	
	config, err := adapter.LoadCoreConfig()
	core.AssertNoError(t, err)
	
	if config == nil {
		t.Error("Expected configuration to be loaded, got nil")
	}
	
	// Validate that configuration has expected structure
	if config.DataDir == "" {
		t.Error("Expected data directory to be set")
	}
	
	if config.LogLevel == "" {
		t.Error("Expected log level to be set")
	}
}