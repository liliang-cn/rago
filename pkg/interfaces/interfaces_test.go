package interfaces

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRAGService implements RAGService for testing
type MockRAGService struct {
	ingestFunc             func(context.Context, domain.IngestRequest) (domain.IngestResponse, error)
	queryFunc              func(context.Context, domain.QueryRequest) (domain.QueryResponse, error)
	streamQueryFunc        func(context.Context, domain.QueryRequest, func(string)) error
	listDocumentsFunc      func(context.Context) ([]domain.Document, error)
	deleteDocumentFunc     func(context.Context, string) error
	resetFunc              func(context.Context) error
	queryWithToolsFunc     func(context.Context, domain.QueryRequest) (domain.QueryResponse, error)
	streamQueryWithToolsFunc func(context.Context, domain.QueryRequest, func(string)) error
}

func (m *MockRAGService) Ingest(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error) {
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, req)
	}
	return domain.IngestResponse{Success: true, DocumentID: "mock-doc-id", ChunkCount: 5}, nil
}

func (m *MockRAGService) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, req)
	}
	return domain.QueryResponse{Answer: "mock answer", Elapsed: "100ms"}, nil
}

func (m *MockRAGService) StreamQuery(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if m.streamQueryFunc != nil {
		return m.streamQueryFunc(ctx, req, callback)
	}
	callback("mock")
	callback(" streaming")
	callback(" response")
	return nil
}

func (m *MockRAGService) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	if m.listDocumentsFunc != nil {
		return m.listDocumentsFunc(ctx)
	}
	return []domain.Document{{ID: "doc1", Content: "content1", Created: time.Now()}}, nil
}

func (m *MockRAGService) DeleteDocument(ctx context.Context, documentID string) error {
	if m.deleteDocumentFunc != nil {
		return m.deleteDocumentFunc(ctx, documentID)
	}
	return nil
}

func (m *MockRAGService) Reset(ctx context.Context) error {
	if m.resetFunc != nil {
		return m.resetFunc(ctx)
	}
	return nil
}

func (m *MockRAGService) QueryWithTools(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	if m.queryWithToolsFunc != nil {
		return m.queryWithToolsFunc(ctx, req)
	}
	return domain.QueryResponse{Answer: "mock tools response", Elapsed: "200ms"}, nil
}

func (m *MockRAGService) StreamQueryWithTools(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if m.streamQueryWithToolsFunc != nil {
		return m.streamQueryWithToolsFunc(ctx, req, callback)
	}
	callback("mock tool streaming")
	return nil
}

// MockMCPService implements MCPService for testing
type MockMCPService struct {
	initializeFunc     func(context.Context) error
	closeFunc          func() error
	isHealthyFunc      func() bool
	getAvailableToolsFunc func() map[string]interface{}
	callToolFunc       func(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockMCPService) Initialize(ctx context.Context) error {
	if m.initializeFunc != nil {
		return m.initializeFunc(ctx)
	}
	return nil
}

func (m *MockMCPService) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockMCPService) IsHealthy() bool {
	if m.isHealthyFunc != nil {
		return m.isHealthyFunc()
	}
	return true
}

func (m *MockMCPService) GetAvailableTools() map[string]interface{} {
	if m.getAvailableToolsFunc != nil {
		return m.getAvailableToolsFunc()
	}
	return map[string]interface{}{
		"filesystem": map[string]string{"type": "file_operations"},
		"calculator": map[string]string{"type": "math_operations"},
	}
}

func (m *MockMCPService) CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, name, args)
	}
	return map[string]interface{}{"result": "mock tool result"}, nil
}

// MockLLMProvider implements LLMProvider for testing
type MockLLMProvider struct {
	name              string
	generateFunc      func(context.Context, string, *domain.GenerationOptions) (string, error)
	streamFunc        func(context.Context, string, *domain.GenerationOptions, func(string)) error
	generateWithToolsFunc func(context.Context, string, []domain.ToolDefinition, *domain.GenerationOptions) (map[string]interface{}, error)
	streamWithToolsFunc   func(context.Context, string, []domain.ToolDefinition, *domain.GenerationOptions, func(string)) error
	isAvailableFunc   func(context.Context) bool
}

func (m *MockLLMProvider) Generate(ctx context.Context, prompt string, options *domain.GenerationOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, options)
	}
	return "mock generated response", nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, prompt string, options *domain.GenerationOptions, callback func(string)) error {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, prompt, options, callback)
	}
	callback("mock")
	callback(" stream")
	return nil
}

func (m *MockLLMProvider) GenerateWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, options *domain.GenerationOptions) (map[string]interface{}, error) {
	if m.generateWithToolsFunc != nil {
		return m.generateWithToolsFunc(ctx, prompt, tools, options)
	}
	return map[string]interface{}{"content": "mock tools response"}, nil
}

func (m *MockLLMProvider) StreamWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, options *domain.GenerationOptions, callback func(string)) error {
	if m.streamWithToolsFunc != nil {
		return m.streamWithToolsFunc(ctx, prompt, tools, options, callback)
	}
	callback("mock tool stream")
	return nil
}

func (m *MockLLMProvider) Name() string {
	return m.name
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	if m.isAvailableFunc != nil {
		return m.isAvailableFunc(ctx)
	}
	return true
}

// MockEmbeddingProvider implements EmbeddingProvider for testing
type MockEmbeddingProvider struct {
	name            string
	dimension       int
	embedFunc       func(context.Context, string) ([]float32, error)
	batchEmbedFunc  func(context.Context, []string) ([][]float32, error)
	isAvailableFunc func(context.Context) bool
}

func (m *MockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	// Return a mock embedding vector
	embedding := make([]float32, m.dimension)
	for i := range embedding {
		embedding[i] = 0.1 * float32(i+1) // Simple pattern for testing
	}
	return embedding, nil
}

func (m *MockEmbeddingProvider) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.batchEmbedFunc != nil {
		return m.batchEmbedFunc(ctx, texts)
	}
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, m.dimension)
		for j := range embeddings[i] {
			embeddings[i][j] = 0.1 * float32(j+1+i*10) // Different pattern for each text
		}
	}
	return embeddings, nil
}

func (m *MockEmbeddingProvider) Dimension() int {
	return m.dimension
}

func (m *MockEmbeddingProvider) Name() string {
	return m.name
}

func (m *MockEmbeddingProvider) IsAvailable(ctx context.Context) bool {
	if m.isAvailableFunc != nil {
		return m.isAvailableFunc(ctx)
	}
	return true
}

// SimpleComponentRegistry implements ComponentRegistry for testing
type SimpleComponentRegistry struct {
	ragService        RAGService
	mcpService        MCPService
	llmProviders      map[string]LLMProvider
	embeddingProviders map[string]EmbeddingProvider
	defaultLLM        string
	defaultEmbedding  string
}

func NewSimpleComponentRegistry() *SimpleComponentRegistry {
	return &SimpleComponentRegistry{
		llmProviders:      make(map[string]LLMProvider),
		embeddingProviders: make(map[string]EmbeddingProvider),
	}
}

func (r *SimpleComponentRegistry) RegisterRAG(service RAGService) {
	r.ragService = service
}

func (r *SimpleComponentRegistry) GetRAG() RAGService {
	return r.ragService
}

func (r *SimpleComponentRegistry) RegisterLLMProvider(name string, provider LLMProvider) {
	r.llmProviders[name] = provider
	if r.defaultLLM == "" {
		r.defaultLLM = name
	}
}

func (r *SimpleComponentRegistry) GetLLMProvider(name string) LLMProvider {
	return r.llmProviders[name]
}

func (r *SimpleComponentRegistry) GetDefaultLLMProvider() LLMProvider {
	if r.defaultLLM == "" {
		return nil
	}
	return r.llmProviders[r.defaultLLM]
}

func (r *SimpleComponentRegistry) RegisterEmbeddingProvider(name string, provider EmbeddingProvider) {
	r.embeddingProviders[name] = provider
	if r.defaultEmbedding == "" {
		r.defaultEmbedding = name
	}
}

func (r *SimpleComponentRegistry) GetEmbeddingProvider(name string) EmbeddingProvider {
	return r.embeddingProviders[name]
}

func (r *SimpleComponentRegistry) GetDefaultEmbeddingProvider() EmbeddingProvider {
	if r.defaultEmbedding == "" {
		return nil
	}
	return r.embeddingProviders[r.defaultEmbedding]
}

func (r *SimpleComponentRegistry) RegisterMCP(service MCPService) {
	r.mcpService = service
}

func (r *SimpleComponentRegistry) GetMCP() MCPService {
	return r.mcpService
}

func (r *SimpleComponentRegistry) HasMCP() bool {
	return r.mcpService != nil
}

// Not implemented in this simple registry - would need additional fields
func (r *SimpleComponentRegistry) RegisterAgents(service AgentService)   {}
func (r *SimpleComponentRegistry) GetAgents() AgentService              { return nil }
func (r *SimpleComponentRegistry) HasAgents() bool                      { return false }
func (r *SimpleComponentRegistry) RegisterScheduler(service SchedulerService) {}
func (r *SimpleComponentRegistry) GetScheduler() SchedulerService       { return nil }
func (r *SimpleComponentRegistry) HasScheduler() bool                   { return false }

// Test ComponentRegistry Implementation
func TestComponentRegistry_RAGService(t *testing.T) {
	registry := NewSimpleComponentRegistry()
	
	// Test nil RAG service initially
	assert.Nil(t, registry.GetRAG())
	
	// Register RAG service
	mockRAG := &MockRAGService{}
	registry.RegisterRAG(mockRAG)
	
	// Test RAG service retrieval
	retrievedRAG := registry.GetRAG()
	assert.NotNil(t, retrievedRAG)
	assert.Equal(t, mockRAG, retrievedRAG)
	
	// Test RAG service functionality
	ctx := context.Background()
	
	// Test Ingest
	ingestReq := domain.IngestRequest{Content: "test content"}
	ingestResp, err := retrievedRAG.Ingest(ctx, ingestReq)
	require.NoError(t, err)
	assert.True(t, ingestResp.Success)
	assert.Equal(t, "mock-doc-id", ingestResp.DocumentID)
	assert.Equal(t, 5, ingestResp.ChunkCount)
	
	// Test Query
	queryReq := domain.QueryRequest{Query: "test query"}
	queryResp, err := retrievedRAG.Query(ctx, queryReq)
	require.NoError(t, err)
	assert.Equal(t, "mock answer", queryResp.Answer)
	assert.Equal(t, "100ms", queryResp.Elapsed)
	
	// Test ListDocuments
	docs, err := retrievedRAG.ListDocuments(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
}

func TestComponentRegistry_LLMProviders(t *testing.T) {
	registry := NewSimpleComponentRegistry()
	
	// Test no providers initially
	assert.Nil(t, registry.GetDefaultLLMProvider())
	assert.Nil(t, registry.GetLLMProvider("nonexistent"))
	
	// Register LLM providers
	mockOllama := &MockLLMProvider{name: "ollama"}
	mockOpenAI := &MockLLMProvider{name: "openai"}
	
	registry.RegisterLLMProvider("ollama", mockOllama)
	registry.RegisterLLMProvider("openai", mockOpenAI)
	
	// Test provider retrieval
	assert.Equal(t, mockOllama, registry.GetLLMProvider("ollama"))
	assert.Equal(t, mockOpenAI, registry.GetLLMProvider("openai"))
	
	// Test default provider (should be first registered)
	defaultLLM := registry.GetDefaultLLMProvider()
	assert.Equal(t, mockOllama, defaultLLM)
	
	// Test provider functionality
	ctx := context.Background()
	response, err := mockOllama.Generate(ctx, "test prompt", nil)
	require.NoError(t, err)
	assert.Equal(t, "mock generated response", response)
	
	assert.Equal(t, "ollama", mockOllama.Name())
	assert.True(t, mockOllama.IsAvailable(ctx))
}

func TestComponentRegistry_EmbeddingProviders(t *testing.T) {
	registry := NewSimpleComponentRegistry()
	
	// Test no providers initially
	assert.Nil(t, registry.GetDefaultEmbeddingProvider())
	assert.Nil(t, registry.GetEmbeddingProvider("nonexistent"))
	
	// Register embedding providers
	mockEmbedder1 := &MockEmbeddingProvider{name: "ollama-embed", dimension: 768}
	mockEmbedder2 := &MockEmbeddingProvider{name: "openai-embed", dimension: 1536}
	
	registry.RegisterEmbeddingProvider("ollama", mockEmbedder1)
	registry.RegisterEmbeddingProvider("openai", mockEmbedder2)
	
	// Test provider retrieval
	assert.Equal(t, mockEmbedder1, registry.GetEmbeddingProvider("ollama"))
	assert.Equal(t, mockEmbedder2, registry.GetEmbeddingProvider("openai"))
	
	// Test default provider
	defaultEmbedder := registry.GetDefaultEmbeddingProvider()
	assert.Equal(t, mockEmbedder1, defaultEmbedder)
	assert.Equal(t, 768, defaultEmbedder.Dimension())
	
	// Test embedding functionality
	ctx := context.Background()
	embedding, err := mockEmbedder1.Embed(ctx, "test text")
	require.NoError(t, err)
	assert.Len(t, embedding, 768)
	assert.Equal(t, float32(0.1), embedding[0])
	
	// Test batch embedding
	embeddings, err := mockEmbedder1.BatchEmbed(ctx, []string{"text1", "text2"})
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	assert.Len(t, embeddings[0], 768)
	assert.Len(t, embeddings[1], 768)
}

func TestComponentRegistry_MCPService(t *testing.T) {
	registry := NewSimpleComponentRegistry()
	
	// Test no MCP service initially
	assert.False(t, registry.HasMCP())
	assert.Nil(t, registry.GetMCP())
	
	// Register MCP service
	mockMCP := &MockMCPService{}
	registry.RegisterMCP(mockMCP)
	
	// Test MCP service retrieval
	assert.True(t, registry.HasMCP())
	retrievedMCP := registry.GetMCP()
	assert.NotNil(t, retrievedMCP)
	assert.Equal(t, mockMCP, retrievedMCP)
	
	// Test MCP service functionality
	ctx := context.Background()
	
	// Test Initialize
	err := retrievedMCP.Initialize(ctx)
	assert.NoError(t, err)
	
	// Test health check
	assert.True(t, retrievedMCP.IsHealthy())
	
	// Test available tools
	tools := retrievedMCP.GetAvailableTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "filesystem")
	assert.Contains(t, tools, "calculator")
	
	// Test tool calling
	result, err := retrievedMCP.CallTool(ctx, "filesystem", map[string]interface{}{"action": "read"})
	require.NoError(t, err)
	assert.Equal(t, "mock tool result", result["result"])
	
	// Test Close
	err = retrievedMCP.Close()
	assert.NoError(t, err)
}

// Test Error Scenarios and Edge Cases
func TestRAGService_ErrorHandling(t *testing.T) {
	mockRAG := &MockRAGService{
		ingestFunc: func(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error) {
			if req.Content == "" && req.FilePath == "" {
				return domain.IngestResponse{Success: false}, errors.New("no content provided")
			}
			return domain.IngestResponse{Success: true, DocumentID: "test-doc"}, nil
		},
		queryFunc: func(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
			if req.Query == "" {
				return domain.QueryResponse{}, errors.New("empty query")
			}
			return domain.QueryResponse{Answer: "response for: " + req.Query}, nil
		},
	}

	ctx := context.Background()

	// Test ingest error
	_, err := mockRAG.Ingest(ctx, domain.IngestRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no content provided")

	// Test successful ingest
	resp, err := mockRAG.Ingest(ctx, domain.IngestRequest{Content: "test content"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "test-doc", resp.DocumentID)

	// Test query error
	_, err = mockRAG.Query(ctx, domain.QueryRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty query")

	// Test successful query
	resp2, err := mockRAG.Query(ctx, domain.QueryRequest{Query: "test question"})
	require.NoError(t, err)
	assert.Equal(t, "response for: test question", resp2.Answer)
}

func TestLLMProvider_ErrorHandling(t *testing.T) {
	mockLLM := &MockLLMProvider{
		name: "test-llm",
		generateFunc: func(ctx context.Context, prompt string, options *domain.GenerationOptions) (string, error) {
			if prompt == "" {
				return "", errors.New("empty prompt")
			}
			return "Generated: " + prompt, nil
		},
		isAvailableFunc: func(ctx context.Context) bool {
			return false // Simulate unavailable provider
		},
	}

	ctx := context.Background()

	// Test generation error
	_, err := mockLLM.Generate(ctx, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty prompt")

	// Test successful generation
	resp, err := mockLLM.Generate(ctx, "test prompt", nil)
	require.NoError(t, err)
	assert.Equal(t, "Generated: test prompt", resp)

	// Test unavailable provider
	assert.False(t, mockLLM.IsAvailable(ctx))
}

func TestEmbeddingProvider_ErrorHandling(t *testing.T) {
	mockEmbedder := &MockEmbeddingProvider{
		name:      "test-embedder",
		dimension: 512,
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			if text == "" {
				return nil, errors.New("empty text")
			}
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	ctx := context.Background()

	// Test embedding error
	_, err := mockEmbedder.Embed(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty text")

	// Test successful embedding
	embedding, err := mockEmbedder.Embed(ctx, "test text")
	require.NoError(t, err)
	assert.Len(t, embedding, 3)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, embedding)
}

func TestMCPService_ErrorHandling(t *testing.T) {
	mockMCP := &MockMCPService{
		initializeFunc: func(ctx context.Context) error {
			return errors.New("initialization failed")
		},
		isHealthyFunc: func() bool {
			return false
		},
		callToolFunc: func(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
			if name == "nonexistent" {
				return nil, errors.New("tool not found")
			}
			return map[string]interface{}{"result": "success"}, nil
		},
	}

	ctx := context.Background()

	// Test initialization error
	err := mockMCP.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")

	// Test unhealthy service
	assert.False(t, mockMCP.IsHealthy())

	// Test tool call error
	_, err = mockMCP.CallTool(ctx, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool not found")

	// Test successful tool call
	result, err := mockMCP.CallTool(ctx, "existing-tool", nil)
	require.NoError(t, err)
	assert.Equal(t, "success", result["result"])
}

func TestStreamingOperations(t *testing.T) {
	mockRAG := &MockRAGService{
		streamQueryFunc: func(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
			if req.Query == "error" {
				return errors.New("streaming error")
			}
			callback("chunk1")
			callback("chunk2")
			callback("chunk3")
			return nil
		},
	}

	mockLLM := &MockLLMProvider{
		streamFunc: func(ctx context.Context, prompt string, options *domain.GenerationOptions, callback func(string)) error {
			if prompt == "error" {
				return errors.New("streaming error")
			}
			callback("token1")
			callback("token2")
			return nil
		},
	}

	ctx := context.Background()

	// Test RAG streaming success
	var chunks []string
	err := mockRAG.StreamQuery(ctx, domain.QueryRequest{Query: "test"}, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"chunk1", "chunk2", "chunk3"}, chunks)

	// Test RAG streaming error
	err = mockRAG.StreamQuery(ctx, domain.QueryRequest{Query: "error"}, func(chunk string) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming error")

	// Test LLM streaming success
	var tokens []string
	err = mockLLM.Stream(ctx, "test", nil, func(token string) {
		tokens = append(tokens, token)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"token1", "token2"}, tokens)

	// Test LLM streaming error
	err = mockLLM.Stream(ctx, "error", nil, func(token string) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming error")
}

func TestGenerationOptions(t *testing.T) {
	mockLLM := &MockLLMProvider{
		generateFunc: func(ctx context.Context, prompt string, options *domain.GenerationOptions) (string, error) {
			if options == nil {
				return "default response", nil
			}
			return "customized response", nil
		},
	}

	ctx := context.Background()

	// Test with nil options
	resp, err := mockLLM.Generate(ctx, "test", nil)
	require.NoError(t, err)
	assert.Equal(t, "default response", resp)

	// Test with custom options
	options := &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	resp, err = mockLLM.Generate(ctx, "test", options)
	require.NoError(t, err)
	assert.Equal(t, "customized response", resp)
}

func TestRegistryProviderOverwrite(t *testing.T) {
	registry := NewSimpleComponentRegistry()

	// Register first provider
	provider1 := &MockLLMProvider{name: "provider1"}
	registry.RegisterLLMProvider("test", provider1)
	assert.Equal(t, provider1, registry.GetLLMProvider("test"))

	// Overwrite with second provider
	provider2 := &MockLLMProvider{name: "provider2"}
	registry.RegisterLLMProvider("test", provider2)
	assert.Equal(t, provider2, registry.GetLLMProvider("test"))
	assert.NotEqual(t, provider1, registry.GetLLMProvider("test"))
}