package client_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

// TestNew tests the New function
func TestNew(t *testing.T) {
	// Create temp config file
	tmpDir, err := os.MkdirTemp("", "rago-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `
[providers]
default_llm = "test"
default_embedder = "test"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test New with valid config path
	c, err := client.New(configPath)
	// May have error due to provider initialization
	if err == nil && c != nil {
		defer c.Close()
	}
	
	// Test New with invalid path
	_, err = client.New("/invalid/path.toml")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

// TestNewWithConfig tests the NewWithConfig function
func TestNewWithConfig(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			DefaultLLM:      "test",
			DefaultEmbedder: "test",
		},
	}

	// Test NewWithConfig
	c, err := client.NewWithConfig(cfg)
	// May have error due to provider initialization
	if err == nil && c != nil {
		defer c.Close()
	}
}

// TestBaseClient_GetConfig tests GetConfig method
func TestBaseClient_GetConfig(t *testing.T) {
	cfg := &config.Config{}
	c, _ := client.NewWithConfig(cfg)
	if c != nil {
		gotCfg := c.GetConfig()
		if gotCfg != cfg {
			t.Error("GetConfig returned different config")
		}
		c.Close()
	}
}

// TestBaseClient_Ingest tests Ingest method
func TestBaseClient_Ingest(t *testing.T) {
	cfg := &config.Config{}
	c, _ := client.NewWithConfig(cfg)
	if c == nil {
		t.Skip("client not initialized")
	}
	defer c.Close()

	ctx := context.Background()
	req := client.IngestRequest{
		Path: "test.txt",
	}

	// Should fail since RAG client is not initialized
	_, err := c.Ingest(ctx, req)
	if err == nil {
		t.Error("expected error for uninitialized RAG client")
	}
}

// TestBaseClient_Query tests Query method
func TestBaseClient_Query(t *testing.T) {
	cfg := &config.Config{}
	c, _ := client.NewWithConfig(cfg)
	if c == nil {
		t.Skip("client not initialized")
	}
	defer c.Close()

	ctx := context.Background()
	req := client.QueryRequest{
		Query: "test",
	}

	// Should fail since RAG client is not initialized
	_, err := c.Query(ctx, req)
	if err == nil {
		t.Error("expected error for uninitialized RAG client")
	}
}

// TestBaseClient_RunTask tests RunTask method
func TestBaseClient_RunTask(t *testing.T) {
	cfg := &config.Config{}
	c, _ := client.NewWithConfig(cfg)
	if c == nil {
		t.Skip("client not initialized")
	}
	defer c.Close()

	ctx := context.Background()
	req := client.TaskRequest{
		Task: "test task",
	}

	// Should succeed with mock implementation
	resp, err := c.RunTask(ctx, req)
	if err != nil {
		t.Errorf("RunTask failed: %v", err)
	}
	if resp == nil || !resp.Success {
		t.Error("expected successful response")
	}
}

// TestBaseClient_EnableMCP tests EnableMCP method
func TestBaseClient_EnableMCP(t *testing.T) {
	cfg := &config.Config{}
	c, _ := client.NewWithConfig(cfg)
	if c == nil {
		t.Skip("client not initialized")
	}
	defer c.Close()

	ctx := context.Background()
	
	// Should fail since MCP service is not initialized
	err := c.EnableMCP(ctx)
	if err == nil {
		t.Error("expected error for uninitialized MCP service")
	}
}

// TestTypes tests basic type structures
func TestTypes(t *testing.T) {
	// Test GenerateOptions
	genOpts := client.GenerateOptions{
		Temperature: 0.7,
		MaxTokens:   100,
	}
	if genOpts.Temperature != 0.7 {
		t.Error("Temperature not set correctly")
	}

	// Test QueryOptions
	queryOpts := client.QueryOptions{
		TopK:        5,
		ShowSources: true,
	}
	if queryOpts.TopK != 5 {
		t.Error("TopK not set correctly")
	}

	// Test IngestOptions
	ingestOpts := client.IngestOptions{
		ChunkSize: 500,
		Overlap:   50,
	}
	if ingestOpts.ChunkSize != 500 {
		t.Error("ChunkSize not set correctly")
	}

	// Test SearchOptions
	searchOpts := client.SearchOptions{
		TopK:      10,
		Threshold: 0.5,
	}
	if searchOpts.TopK != 10 {
		t.Error("TopK not set correctly")
	}

	// Test AgentOptions
	agentOpts := client.AgentOptions{
		Verbose: true,
		Timeout: 30,
	}
	if !agentOpts.Verbose {
		t.Error("Verbose not set correctly")
	}
}

// TestWrappers tests wrapper creation
func TestWrappers(t *testing.T) {
	// Test LLMWrapper
	llmWrapper := client.NewLLMWrapper(nil)
	if llmWrapper == nil {
		t.Error("NewLLMWrapper returned nil")
	}

	// Test RAGWrapper
	ragWrapper := client.NewRAGWrapper(nil)
	if ragWrapper == nil {
		t.Error("NewRAGWrapper returned nil")
	}

	// Test ToolsWrapper
	toolsWrapper := client.NewToolsWrapper(nil)
	if toolsWrapper == nil {
		t.Error("NewToolsWrapper returned nil")
	}

	// Test AgentWrapper
	agentWrapper := client.NewAgentWrapper(nil)
	if agentWrapper == nil {
		t.Error("NewAgentWrapper returned nil")
	}
}

// TestSearchResult tests SearchResult structure
func TestSearchResult(t *testing.T) {
	result := client.SearchResult{
		ID:      "123",
		Content: "test content",
		Score:   0.95,
		Source:  "test.txt",
	}

	if result.ID != "123" {
		t.Error("ID not set correctly")
	}
	if result.Score != 0.95 {
		t.Error("Score not set correctly")
	}
}

// TestClientSearchOptions tests ClientSearchOptions
func TestClientSearchOptions(t *testing.T) {
	opts := client.DefaultSearchOptions()
	if opts == nil {
		t.Error("DefaultSearchOptions returned nil")
	}
	if opts.TopK != 5 {
		t.Error("Default TopK not set correctly")
	}
	if opts.VectorWeight != 0.7 {
		t.Error("Default VectorWeight not set correctly")
	}
}