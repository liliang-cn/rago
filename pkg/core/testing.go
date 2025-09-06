// Package core provides testing utilities for RAGO pillars.
// This file contains test helpers, mocks, and utilities for testing
// individual pillars and the unified client.
package core

import (
	"context"
	"time"
)

// ===== TEST UTILITIES =====

// TestConfig provides a default configuration for testing
func TestConfig() Config {
	return Config{
		DataDir:  "/tmp/rago-test",
		LogLevel: "debug",
		
		LLM: TestLLMConfig(),
		RAG: TestRAGConfig(),
		MCP: TestMCPConfig(),
		Agents: TestAgentConfig(),
		
		Mode: ModeConfig{
			RAGOnly:      false,
			LLMOnly:      true, // Disable other pillars for simpler tests
			DisableMCP:   true,
			DisableAgent: true,
		},
	}
}

// TestLLMConfig provides a default LLM configuration for testing
func TestLLMConfig() LLMConfig {
	return LLMConfig{
		DefaultProvider: "ollama",
		LoadBalancing: LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   false, // Disable health check for tests
			CheckInterval: 10 * time.Second,
		},
		Providers: map[string]ProviderConfig{
			"ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama3.2",
				Weight:  1,
				Timeout: 30 * time.Second,
				Parameters: map[string]interface{}{
					"test_mode": true,
				},
			},
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  true,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  2,
		},
	}
}

// TestRAGConfig provides a default RAG configuration for testing
func TestRAGConfig() RAGConfig {
	return RAGConfig{
		StorageBackend: "sqlite",
		ChunkingStrategy: ChunkingConfig{
			Strategy:     "sentence",
			ChunkSize:    500,
			ChunkOverlap: 50,
			MinChunkSize: 100,
		},
		VectorStore: VectorStoreConfig{
			Backend:    "sqlite",
			Dimensions: 384,
			Metric:     "cosine",
			IndexType:  "flat",
		},
		KeywordStore: KeywordStoreConfig{
			Backend:   "bleve",
			Analyzer:  "standard",
			Languages: []string{"en"},
			Stemming:  true,
		},
		Search: SearchConfig{
			DefaultLimit:     10,
			MaxLimit:         100,
			DefaultThreshold: 0.0,
			HybridWeights: struct {
				Vector  float32 `toml:"vector"`
				Keyword float32 `toml:"keyword"`
			}{
				Vector:  0.7,
				Keyword: 0.3,
			},
		},
		Embedding: EmbeddingConfig{
			Provider:   "ollama",
			Model:      "nomic-embed-text",
			Dimensions: 384,
			BatchSize:  10,
		},
	}
}

// TestMCPConfig provides a default MCP configuration for testing
func TestMCPConfig() MCPConfig {
	return MCPConfig{
		ServersPath: "/tmp/rago-test/mcp",
		Servers: []ServerConfig{
			{
				Name:    "test-server",
				Command: []string{"echo", "test"},
				Args:    []string{},
				Env: map[string]string{
					"TEST_ENV": "test_value",
				},
				WorkingDir: "/tmp",
				Timeout:    30 * time.Second,
				Retries:    3,
			},
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  true,
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  2,
		},
		ToolExecution: ToolExecutionConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 30 * time.Second,
			EnableCache:    true,
			CacheTTL:       5 * time.Minute,
		},
	}
}

// TestAgentConfig provides a default Agent configuration for testing
func TestAgentConfig() AgentsConfig {
	return AgentsConfig{
		WorkflowEngine: WorkflowEngineConfig{
			MaxSteps:       50,
			StepTimeout:    30 * time.Second,
			StateBackend:   "memory",
			EnableRecovery: true,
		},
		Scheduling: SchedulingConfig{
			Backend:       "memory",
			MaxConcurrent: 5,
			QueueSize:     100,
		},
		StateStorage: StateStorageConfig{
			Backend:    "memory",
			Persistent: false,
			TTL:        1 * time.Hour,
		},
		ReasoningChains: ReasoningChainsConfig{
			MaxSteps:      25,
			MaxMemorySize: 1000,
			StepTimeout:   10 * time.Second,
		},
	}
}

// ===== MOCK IMPLEMENTATIONS =====

// MockLLMService provides a mock implementation of LLMService for testing
type MockLLMService struct {
	providers map[string]ProviderInfo
}

// NewMockLLMService creates a new mock LLM service
func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		providers: make(map[string]ProviderInfo),
	}
}

// AddProvider adds a mock provider
func (m *MockLLMService) AddProvider(name string, config ProviderConfig) error {
	m.providers[name] = ProviderInfo{
		Name:   name,
		Type:   config.Type,
		Model:  config.Model,
		Health: HealthStatusHealthy,
		Weight: config.Weight,
	}
	return nil
}

// RemoveProvider removes a mock provider
func (m *MockLLMService) RemoveProvider(name string) error {
	delete(m.providers, name)
	return nil
}

// ListProviders returns all mock providers
func (m *MockLLMService) ListProviders() []ProviderInfo {
	var providers []ProviderInfo
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
}

// GetProviderHealth returns mock provider health
func (m *MockLLMService) GetProviderHealth() map[string]HealthStatus {
	health := make(map[string]HealthStatus)
	for name := range m.providers {
		health[name] = HealthStatusHealthy
	}
	return health
}

// Generate returns a mock generation response
func (m *MockLLMService) Generate(ctx context.Context, req GenerationRequest) (*GenerationResponse, error) {
	return &GenerationResponse{
		Content:  "Mock response to: " + req.Prompt,
		Model:    "mock-model",
		Provider: "mock",
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Duration: 100 * time.Millisecond,
	}, nil
}

// Stream returns a mock streaming response
func (m *MockLLMService) Stream(ctx context.Context, req GenerationRequest, callback StreamCallback) error {
	chunks := []string{"Mock ", "streaming ", "response"}
	for i, chunk := range chunks {
		if err := callback(StreamChunk{
			Content:  chunk,
			Delta:    chunk,
			Finished: i == len(chunks)-1,
			Duration: 50 * time.Millisecond,
		}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateWithTools returns a mock tool generation response
func (m *MockLLMService) GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error) {
	return &ToolGenerationResponse{
		GenerationResponse: GenerationResponse{
			Content:  "Mock tool response",
			Model:    "mock-model",
			Provider: "mock",
			Usage: TokenUsage{
				PromptTokens:     15,
				CompletionTokens: 25,
				TotalTokens:      40,
			},
			Duration: 150 * time.Millisecond,
		},
		ToolCalls: []ToolCall{
			{
				ID:   "mock-tool-call",
				Name: "mock_tool",
				Parameters: map[string]interface{}{
					"param": "value",
				},
			},
		},
	}, nil
}

// StreamWithTools returns a mock streaming tool response
func (m *MockLLMService) StreamWithTools(ctx context.Context, req ToolGenerationRequest, callback ToolStreamCallback) error {
	return callback(ToolStreamChunk{
		StreamChunk: StreamChunk{
			Content:  "Mock tool stream",
			Delta:    "Mock tool stream",
			Finished: true,
			Duration: 100 * time.Millisecond,
		},
	})
}

// GenerateBatch returns mock batch responses
func (m *MockLLMService) GenerateBatch(ctx context.Context, requests []GenerationRequest) ([]GenerationResponse, error) {
	var responses []GenerationResponse
	for _, req := range requests {
		response, err := m.Generate(ctx, req)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *response)
	}
	return responses, nil
}

// ===== MOCK RAG SERVICE =====

// MockRAGService provides a mock implementation of RAGService for testing
type MockRAGService struct {
	documents map[string]Document
}

// NewMockRAGService creates a new mock RAG service
func NewMockRAGService() *MockRAGService {
	return &MockRAGService{
		documents: make(map[string]Document),
	}
}

// IngestDocument adds a mock document
func (m *MockRAGService) IngestDocument(ctx context.Context, req IngestRequest) (*IngestResponse, error) {
	m.documents[req.DocumentID] = Document{
		ID:          req.DocumentID,
		Content:     req.Content,
		ContentType: req.ContentType,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		Size:        int64(len(req.Content)),
		ChunksCount: 1,
	}
	
	return &IngestResponse{
		DocumentID:   req.DocumentID,
		ChunksCount:  1,
		ProcessedAt:  time.Now(),
		Duration:     50 * time.Millisecond,
		StorageSize:  int64(len(req.Content)),
	}, nil
}

// IngestBatch adds mock documents in batch
func (m *MockRAGService) IngestBatch(ctx context.Context, requests []IngestRequest) (*BatchIngestResponse, error) {
	var responses []IngestResponse
	for _, req := range requests {
		response, err := m.IngestDocument(ctx, req)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *response)
	}
	
	return &BatchIngestResponse{
		Responses:       responses,
		TotalDocuments:  len(requests),
		SuccessfulCount: len(responses),
		FailedCount:     0,
		Duration:        100 * time.Millisecond,
	}, nil
}

// DeleteDocument removes a mock document
func (m *MockRAGService) DeleteDocument(ctx context.Context, docID string) error {
	delete(m.documents, docID)
	return nil
}

// ListDocuments returns mock documents
func (m *MockRAGService) ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error) {
	var documents []Document
	for _, doc := range m.documents {
		documents = append(documents, doc)
	}
	return documents, nil
}

// Search returns mock search results
func (m *MockRAGService) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	var results []SearchResult
	for _, doc := range m.documents {
		results = append(results, SearchResult{
			DocumentID: doc.ID,
			ChunkID:    doc.ID + "-chunk-1",
			Content:    doc.Content,
			Score:      0.8,
			Metadata:   doc.Metadata,
		})
	}
	
	return &SearchResponse{
		Results:  results,
		Total:    len(results),
		Duration: 25 * time.Millisecond,
		Query:    req.Query,
	}, nil
}

// HybridSearch returns mock hybrid search results
func (m *MockRAGService) HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResponse, error) {
	searchResp, err := m.Search(ctx, req.SearchRequest)
	if err != nil {
		return nil, err
	}
	
	return &HybridSearchResponse{
		SearchResponse:  *searchResp,
		VectorResults:   searchResp.Results,
		KeywordResults:  searchResp.Results,
		FusionMethod:    "rrf",
	}, nil
}

// GetStats returns mock stats
func (m *MockRAGService) GetStats(ctx context.Context) (*RAGStats, error) {
	return &RAGStats{
		TotalDocuments: len(m.documents),
		TotalChunks:    len(m.documents),
		StorageSize:    1024,
		IndexSize:      512,
		ByContentType:  map[string]int{"text": len(m.documents)},
		LastOptimized:  time.Now(),
	}, nil
}

// Optimize does nothing in mock
func (m *MockRAGService) Optimize(ctx context.Context) error {
	return nil
}

// Reset clears mock documents
func (m *MockRAGService) Reset(ctx context.Context) error {
	m.documents = make(map[string]Document)
	return nil
}

// ===== TEST HELPERS =====

// AssertNoError fails the test if err is not nil
func AssertNoError(t interface{ Errorf(format string, args ...interface{}) }, err error) {
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t interface{ Errorf(format string, args ...interface{}) }, err error) {
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t interface{ Errorf(format string, args ...interface{}) }, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}