package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockEmbedder) Name() string {
	return "mock-embedder"
}

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	args := m.Called(ctx, messages, tools, opts)
	return args.Get(0).(*domain.GenerationResult), args.Error(1)
}

func (m *MockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	args := m.Called(ctx, prompt, opts, callback)
	return args.Error(0)
}

func (m *MockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	args := m.Called(ctx, messages, tools, opts, callback)
	return args.Error(0)
}

func (m *MockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	args := m.Called(ctx, prompt, schema, opts)
	return args.Get(0).(*domain.StructuredResult), args.Error(1)
}

type MockChunker struct {
	mock.Mock
}

func (m *MockChunker) Split(text string, options domain.ChunkOptions) ([]string, error) {
	args := m.Called(text, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type MockVectorStore struct {
	mock.Mock
}

func (m *MockVectorStore) Store(ctx context.Context, chunks []domain.Chunk) error {
	args := m.Called(ctx, chunks)
	return args.Error(0)
}

func (m *MockVectorStore) Search(ctx context.Context, vector []float64, k int) ([]domain.Chunk, error) {
	args := m.Called(ctx, vector, k)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Chunk), args.Error(1)
}

func (m *MockVectorStore) SearchWithFilters(ctx context.Context, vector []float64, k int, filters map[string]interface{}) ([]domain.Chunk, error) {
	args := m.Called(ctx, vector, k, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Chunk), args.Error(1)
}

func (m *MockVectorStore) Reset(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVectorStore) Delete(ctx context.Context, documentID string) error {
	args := m.Called(ctx, documentID)
	return args.Error(0)
}

func (m *MockVectorStore) List(ctx context.Context) ([]domain.Document, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Document), args.Error(1)
}

type MockKeywordStore struct {
	mock.Mock
}

func (m *MockKeywordStore) Index(ctx context.Context, chunk domain.Chunk) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockKeywordStore) Search(ctx context.Context, query string, k int) ([]domain.Chunk, error) {
	args := m.Called(ctx, query, k)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Chunk), args.Error(1)
}

func (m *MockKeywordStore) Reset(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockKeywordStore) Delete(ctx context.Context, documentID string) error {
	args := m.Called(ctx, documentID)
	return args.Error(0)
}

func (m *MockKeywordStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockDocumentStore struct {
	mock.Mock
}

func (m *MockDocumentStore) Store(ctx context.Context, doc domain.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockDocumentStore) Get(ctx context.Context, id string) (domain.Document, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Document), args.Error(1)
}

func (m *MockDocumentStore) List(ctx context.Context) ([]domain.Document, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Document), args.Error(1)
}

func (m *MockDocumentStore) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockMetadataExtractor struct {
	mock.Mock
}

func (m *MockMetadataExtractor) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	args := m.Called(ctx, content, model)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExtractedMetadata), args.Error(1)
}

// Helper functions

func createMockService() (*Service, *MockEmbedder, *MockGenerator, *MockChunker, *MockVectorStore, *MockKeywordStore, *MockDocumentStore, *MockMetadataExtractor) {
	embedder := &MockEmbedder{}
	generator := &MockGenerator{}
	chunker := &MockChunker{}
	vectorStore := &MockVectorStore{}
	keywordStore := &MockKeywordStore{}
	documentStore := &MockDocumentStore{}
	metadataExtractor := &MockMetadataExtractor{}

	cfg := &config.Config{
		Chunker: config.ChunkerConfig{
			ChunkSize: 1000,
			Overlap:   200,
		},
		Ingest: config.IngestConfig{
			MetadataExtraction: config.MetadataExtractionConfig{
				Enable:   false,
				LLMModel: "test-model",
			},
		},
		Tools: tools.ToolConfig{
			Enabled: false, // Disable tools for most tests
		},
	}

	service := New(
		embedder,
		generator,
		chunker,
		vectorStore,
		keywordStore,
		documentStore,
		cfg,
		metadataExtractor,
	)

	return service, embedder, generator, chunker, vectorStore, keywordStore, documentStore, metadataExtractor
}

func createTestFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// Tests

func TestNewService(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.embedder)
	assert.NotNil(t, service.generator)
	assert.NotNil(t, service.chunker)
	assert.NotNil(t, service.vectorStore)
	assert.NotNil(t, service.keywordStore)
	assert.NotNil(t, service.documentStore)
	assert.NotNil(t, service.config)
	assert.NotNil(t, service.llmService)
}

func TestIngest_FileContent(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	// Create test file
	content := "This is a test document for ingestion."
	filePath := createTestFile(t, content)
	
	// Setup mocks
	chunks := []string{"This is a test", "document for ingestion."}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, "This is a test").Return(embeddings, nil)
	embedder.On("Embed", mock.Anything, "document for ingestion.").Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(nil)
	
	// Test ingest
	req := domain.IngestRequest{
		FilePath:  filePath,
		ChunkSize: 500,
		Overlap:   50,
		Metadata:  map[string]interface{}{"source": "test"},
	}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.ChunkCount)
	assert.NotEmpty(t, resp.DocumentID)
	
	// Verify mocks were called
	embedder.AssertExpectations(t)
	chunker.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	documentStore.AssertExpectations(t)
}

func TestIngest_DirectContent(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	// Setup mocks
	content := "Direct content for testing"
	chunks := []string{content}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, content).Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(nil)
	
	// Test ingest with direct content
	req := domain.IngestRequest{
		Content: content,
	}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.ChunkCount)
	
	// Verify mocks
	embedder.AssertExpectations(t)
	chunker.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	documentStore.AssertExpectations(t)
}

func TestIngest_InvalidRequest(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	tests := []struct {
		name string
		req  domain.IngestRequest
		err  string
	}{
		{
			name: "empty request",
			req:  domain.IngestRequest{},
			err:  "no content source provided",
		},
		{
			name: "invalid file path",
			req:  domain.IngestRequest{FilePath: "/non/existent/file.txt"},
			err:  "failed to read file",
		},
		{
			name: "empty content",
			req:  domain.IngestRequest{Content: ""},
			err:  "no content source provided",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Ingest(context.Background(), tt.req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
			// Response will have Success=false on error
			assert.False(t, resp.Success)
		})
	}
}

func TestIngest_ChunkerError(t *testing.T) {
	service, _, _, chunker, _, _, _, _ := createMockService()
	
	content := "Test content"
	filePath := createTestFile(t, content)
	
	// Setup chunker to return error
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(nil, errors.New("chunking failed"))
	
	req := domain.IngestRequest{FilePath: filePath}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to chunk text")
	assert.Equal(t, domain.IngestResponse{}, resp)
}

func TestIngest_EmbeddingError(t *testing.T) {
	service, embedder, _, chunker, _, _, _, _ := createMockService()
	
	content := "Test content"
	filePath := createTestFile(t, content)
	chunks := []string{"Test content"}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, "Test content").Return(nil, errors.New("embedding failed"))
	
	req := domain.IngestRequest{FilePath: filePath}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate embedding")
	assert.Equal(t, domain.IngestResponse{}, resp)
}

func TestIngest_WithMetadataExtraction(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, metadataExtractor := createMockService()
	
	// Enable metadata extraction
	service.config.Ingest.MetadataExtraction.Enable = true
	
	content := "This is a test document about artificial intelligence."
	filePath := createTestFile(t, content)
	
	// Setup mocks
	chunks := []string{content}
	embeddings := []float64{0.1, 0.2, 0.3}
	extractedMetadata := &domain.ExtractedMetadata{
		Summary:      "Test document about AI",
		Keywords:     []string{"AI", "test"},
		DocumentType: "research",
		CreationDate: "2024-01-01",
	}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, content).Return(embeddings, nil)
	metadataExtractor.On("ExtractMetadata", mock.Anything, content, "test-model").Return(extractedMetadata, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(nil)
	
	req := domain.IngestRequest{
		FilePath: filePath,
		Metadata: map[string]interface{}{"source": "test"},
	}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	
	// Verify metadata extractor was called
	metadataExtractor.AssertExpectations(t)
}

func TestQuery_BasicSearch(t *testing.T) {
	service, embedder, generator, _, vectorStore, keywordStore, _, _ := createMockService()
	
	query := "test query"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	
	vectorResults := []domain.Chunk{
		{
			ID:    "chunk-1",
			DocumentID: "doc-1",
			Content:    "This is a test result",
			Score:      0.9,
			Metadata:   map[string]interface{}{"source": "test"},
		},
	}
	
	keywordResults := []domain.Chunk{
		{
			ID:    "chunk-2",
			DocumentID: "doc-2",
			Content:    "Another test result",
			Score:      0.8,
			Metadata:   map[string]interface{}{"source": "test2"},
		},
	}
	
	generatedResponse := "Based on the search results, here's the answer to your query."
	
	// Setup mocks
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("Search", mock.Anything, queryEmbedding, 10).Return(vectorResults, nil)
	keywordStore.On("Search", mock.Anything, query, 10).Return(keywordResults, nil)
	generator.On("Generate", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(generatedResponse, nil)
	
	req := domain.QueryRequest{
		Query:       query,
		TopK:        10,
		ShowSources: true, // Need to request sources to get them
	}
	
	resp, err := service.Query(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Answer != "")
	assert.Equal(t, generatedResponse, resp.Answer)
	// Sources may be combined/deduplicated, so check for at least 1
	assert.GreaterOrEqual(t, len(resp.Sources), 1)
	
	// Verify mocks
	embedder.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestQuery_EmbeddingError(t *testing.T) {
	service, embedder, generator, _, _, keywordStore, _, _ := createMockService()
	
	embedder.On("Embed", mock.Anything, "test query").Return(nil, errors.New("embedding failed"))
	// With embedding error, system falls back to keyword search only
	keywordResults := []domain.Chunk{
		{
			ID:      "chunk-1",
			Content: "Fallback result from keyword search",
			Score:   0.7,
		},
	}
	keywordStore.On("Search", mock.Anything, "test query", mock.Anything).Return(keywordResults, nil)
	generator.On("Generate", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("Answer based on keyword search", nil)
	
	req := domain.QueryRequest{
		Query:       "test query",
		ShowSources: true,
	}
	
	resp, err := service.Query(context.Background(), req)
	// The service should NOT return error - it falls back gracefully
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Answer)
	// Should have results from keyword search
	assert.GreaterOrEqual(t, len(resp.Sources), 1)
}

func TestQuery_WithTools(t *testing.T) {
	service, embedder, generator, _, vectorStore, keywordStore, _, _ := createMockService()
	
	// Enable tools for this test
	service.toolsEnabled = true
	
	query := "what's the weather like?"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	
	// Even with tools, we need some context chunks
	vectorResults := []domain.Chunk{
		{
			ID:      "chunk-1",
			Content: "Weather information context",
			Score:   0.5,
		},
	}
	keywordResults := []domain.Chunk{}
	
	// Tools are handled through MCP now, not directly
	
	responseWithTools := "The weather is sunny with a temperature of 22°C."
	
	// Setup mocks
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("Search", mock.Anything, queryEmbedding, 10).Return(vectorResults, nil)
	keywordStore.On("Search", mock.Anything, query, 10).Return(keywordResults, nil)
	// QueryWithTools will fall back to regular query since tools aren't configured
	generator.On("Generate", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(responseWithTools, nil)
	
	req := domain.QueryRequest{
		Query:        query,
		ToolsEnabled: true,
		TopK:         10,
	}
	
	resp, err := service.QueryWithTools(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Answer != "")
	assert.Equal(t, responseWithTools, resp.Answer)
}

func TestStreamQuery(t *testing.T) {
	service, embedder, generator, _, vectorStore, keywordStore, _, _ := createMockService()
	
	query := "streaming test query"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	// Need chunks to avoid "no information found" response
	vectorResults := []domain.Chunk{
		{
			ID:      "chunk-1",
			Content: "Test content for streaming",
			Score:   0.8,
		},
	}
	keywordResults := []domain.Chunk{}
	
	// Setup mocks
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("Search", mock.Anything, queryEmbedding, 10).Return(vectorResults, nil)
	keywordStore.On("Search", mock.Anything, query, 10).Return(keywordResults, nil)
	generator.On("Stream", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*domain.GenerationOptions"), mock.AnythingOfType("func(string)")).Return(nil).Run(func(args mock.Arguments) {
		callback := args.Get(3).(func(string))
		callback("Streaming")
		callback(" response")
		callback(" chunk")
	})
	
	req := domain.QueryRequest{
		Query:     query,
		TopK:      10,
	}
	
	var streamedContent []string
	callback := func(chunk string) {
		streamedContent = append(streamedContent, chunk)
	}
	
	err := service.StreamQuery(context.Background(), req, callback)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Streaming", " response", " chunk"}, streamedContent)
	
	// Verify mocks
	embedder.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestListDocuments(t *testing.T) {
	service, _, _, _, _, _, documentStore, _ := createMockService()
	
	expectedDocs := []domain.Document{
		{
			ID:      "doc-1",
			Path:    "/test/doc1.txt",
			Content: "Document 1 content",
			Created: time.Now(),
		},
		{
			ID:      "doc-2",
			Path:    "/test/doc2.txt",
			Content: "Document 2 content",
			Created: time.Now(),
		},
	}
	
	documentStore.On("List", mock.Anything).Return(expectedDocs, nil)
	
	docs, err := service.ListDocuments(context.Background())
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, expectedDocs[0].ID, docs[0].ID)
	assert.Equal(t, expectedDocs[1].ID, docs[1].ID)
	
	documentStore.AssertExpectations(t)
}

func TestDeleteDocument(t *testing.T) {
	service, _, _, _, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	documentID := "doc-to-delete"
	
	// Setup mocks
	vectorStore.On("Delete", mock.Anything, documentID).Return(nil)
	keywordStore.On("Delete", mock.Anything, documentID).Return(nil)
	documentStore.On("Delete", mock.Anything, documentID).Return(nil)
	
	err := service.DeleteDocument(context.Background(), documentID)
	assert.NoError(t, err)
	
	// Verify all stores were called
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	documentStore.AssertExpectations(t)
}

func TestReset(t *testing.T) {
	service, _, _, _, vectorStore, keywordStore, _, _ := createMockService()
	
	// Setup mocks to expect Reset calls
	vectorStore.On("Reset", mock.Anything).Return(nil)
	keywordStore.On("Reset", mock.Anything).Return(nil)
	// documentStore.Reset is not called by Service.Reset
	
	err := service.Reset(context.Background())
	assert.NoError(t, err)
	
	// Verify stores were reset
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
}

func TestValidateIngestRequest(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	tests := []struct {
		name string
		req  domain.IngestRequest
		want error
	}{
		{
			name: "valid file request",
			req:  domain.IngestRequest{FilePath: "/test/file.txt"},
			want: nil,
		},
		{
			name: "valid URL request",
			req:  domain.IngestRequest{URL: "https://example.com"},
			want: nil,
		},
		{
			name: "valid content request",
			req:  domain.IngestRequest{Content: "direct content"},
			want: nil,
		},
		{
			name: "empty request",
			req:  domain.IngestRequest{},
			want: errors.New("no content source provided"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateIngestRequest(tt.req)
			if tt.want == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.want.Error())
			}
		})
	}
}

func TestExtractContent_PDFFile(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	// Create a simple text file (we'll pretend it's PDF for testing)
	content := "PDF content for testing"
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.pdf")
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	
	req := domain.IngestRequest{FilePath: filePath}
	
	// This will actually try to parse as PDF and likely fail, but we can test the error handling
	_, err = service.extractContent(req)
	// We expect an error since it's not a real PDF
	assert.Error(t, err)
}

func TestConcurrentIngestion(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	// Setup mocks for concurrent operations
	chunks := []string{"concurrent test chunk"}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", mock.AnythingOfType("string"), mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, mock.AnythingOfType("string")).Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(nil)
	
	// Run concurrent ingestions
	concurrency := 5
	done := make(chan error, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			content := "Concurrent test content " + string(rune(index+'0'))
			req := domain.IngestRequest{Content: content}
			
			_, err := service.Ingest(context.Background(), req)
			done <- err
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-done
		assert.NoError(t, err)
	}
	
	// Verify mocks were called the expected number of times
	embedder.AssertExpectations(t)
	chunker.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	documentStore.AssertExpectations(t)
}

// TestHybridSearch tests the hybrid search functionality with RRF fusion
func TestHybridSearch(t *testing.T) {
	service, embedder, _, _, vectorStore, keywordStore, _, _ := createMockService()
	service.config.RRF = config.RRFConfig{K: 60} // Set RRF constant
	
	query := "test hybrid search"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	
	// Vector search results
	vectorChunks := []domain.Chunk{
		{ID: "chunk-1", Content: "Vector result 1", Score: 0.9},
		{ID: "chunk-2", Content: "Vector result 2", Score: 0.8},
		{ID: "chunk-3", Content: "Common result", Score: 0.7},
	}
	
	// Keyword search results
	keywordChunks := []domain.Chunk{
		{ID: "chunk-3", Content: "Common result", Score: 0.85}, // Overlapping with vector
		{ID: "chunk-4", Content: "Keyword result 1", Score: 0.75},
		{ID: "chunk-5", Content: "Keyword result 2", Score: 0.65},
	}
	
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("Search", mock.Anything, queryEmbedding, 5).Return(vectorChunks, nil)
	keywordStore.On("Search", mock.Anything, query, 5).Return(keywordChunks, nil)
	
	req := domain.QueryRequest{
		Query: query,
		TopK:  5,
	}
	
	chunks, err := service.hybridSearch(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, chunks)
	
	// Check that results are fused and deduplicated
	seenIDs := make(map[string]bool)
	for _, chunk := range chunks {
		assert.False(t, seenIDs[chunk.ID], "Duplicate chunk ID found: %s", chunk.ID)
		seenIDs[chunk.ID] = true
	}
	
	// Check that common result has higher score due to fusion
	for _, chunk := range chunks {
		if chunk.ID == "chunk-3" {
			assert.Greater(t, chunk.Score, 0.0, "Fused chunk should have positive score")
			break
		}
	}
	
	embedder.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
}

// TestHybridSearch_WithFilters tests search with metadata filters
func TestHybridSearch_WithFilters(t *testing.T) {
	service, embedder, _, _, vectorStore, keywordStore, _, _ := createMockService()
	
	query := "filtered search"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	filters := map[string]interface{}{
		"department": "engineering",
		"year":       2024,
	}
	
	filteredResults := []domain.Chunk{
		{ID: "chunk-1", Content: "Filtered result", Score: 0.95},
	}
	
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("SearchWithFilters", mock.Anything, queryEmbedding, 5, filters).Return(filteredResults, nil)
	keywordStore.On("Search", mock.Anything, query, 5).Return([]domain.Chunk{}, nil)
	
	req := domain.QueryRequest{
		Query:   query,
		TopK:    5,
		Filters: filters,
	}
	
	chunks, err := service.hybridSearch(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)
	assert.Equal(t, "chunk-1", chunks[0].ID)
	
	embedder.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
}

// TestFuseResults tests the RRF fusion algorithm
func TestFuseResults(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	service.config.RRF = config.RRFConfig{K: 60}
	
	listA := []domain.Chunk{
		{ID: "1", Content: "A1", Score: 0.9},
		{ID: "2", Content: "A2", Score: 0.8},
		{ID: "3", Content: "Common", Score: 0.7},
		{ID: "4", Content: "A4", Score: 0.6},
	}
	
	listB := []domain.Chunk{
		{ID: "3", Content: "Common", Score: 0.85},
		{ID: "5", Content: "B1", Score: 0.75},
		{ID: "2", Content: "A2", Score: 0.65},
		{ID: "6", Content: "B2", Score: 0.55},
	}
	
	fused := service.fuseResults(listA, listB)
	
	// Check that all unique chunks are present
	assert.Len(t, fused, 6)
	
	// Check that chunks appearing in both lists have higher scores
	idScores := make(map[string]float64)
	for _, chunk := range fused {
		idScores[chunk.ID] = chunk.Score
	}
	
	// Chunks in both lists should have higher RRF scores
	assert.Greater(t, idScores["3"], idScores["1"])
	assert.Greater(t, idScores["2"], idScores["4"])
}

// TestDeduplicateChunks tests chunk deduplication by content
func TestDeduplicateChunks(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	chunks := []domain.Chunk{
		{ID: "1", Content: "Unique content 1"},
		{ID: "2", Content: "Duplicate content"},
		{ID: "3", Content: "Unique content 2"},
		{ID: "4", Content: "Duplicate content"}, // Same content as ID 2
		{ID: "5", Content: "Unique content 3"},
	}
	
	deduplicated := service.deduplicateChunks(chunks)
	
	assert.Len(t, deduplicated, 4) // Should have 4 unique contents
	
	// Check that no duplicate contents exist
	contents := make(map[string]bool)
	for _, chunk := range deduplicated {
		assert.False(t, contents[chunk.Content], "Duplicate content found")
		contents[chunk.Content] = true
	}
}

// TestCleanThinkingTags tests removal of thinking tags from responses
func TestCleanThinkingTags(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple thinking tags",
			input:    "Here is <think>internal reasoning</think> the answer",
			expected: "Here is the answer",
		},
		{
			name:     "multiline thinking tags",
			input:    "Start\n<think>\nLine 1\nLine 2\n</think>\nEnd",
			expected: "Start End",
		},
		{
			name:     "no thinking tags",
			input:    "Plain response without tags",
			expected: "Plain response without tags",
		},
		{
			name:     "multiple thinking blocks",
			input:    "First <think>thought1</think> middle <think>thought2</think> end",
			expected: "First middle end",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.cleanThinkingTags(tt.input)
			assert.Equal(t, strings.TrimSpace(tt.expected), result)
		})
	}
}

// TestWrapCallbackForThinking tests the thinking tag filter in streaming
func TestWrapCallbackForThinking(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	t.Run("with thinking shown", func(t *testing.T) {
		var received []string
		callback := func(s string) {
			received = append(received, s)
		}
		
		wrapped := service.wrapCallbackForThinking(callback, true)
		wrapped("Here is ")
		wrapped("<think>")
		wrapped("internal thought")
		wrapped("</think>")
		wrapped(" the answer")
		
		assert.Equal(t, []string{"Here is ", "<think>", "internal thought", "</think>", " the answer"}, received)
	})
	
	t.Run("with thinking hidden", func(t *testing.T) {
		var received []string
		callback := func(s string) {
			received = append(received, s)
		}
		
		wrapped := service.wrapCallbackForThinking(callback, false)
		wrapped("Here is ")
		wrapped("<think>")
		wrapped("internal thought")
		wrapped("</think>")
		wrapped(" the answer")
		
		assert.Equal(t, []string{"Here is ", " the answer"}, received)
	})
}

// TestIngest_LargeDocument tests handling of large documents
func TestIngest_LargeDocument(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	// Create a large content string (1MB)
	largeContent := strings.Repeat("Large document content. ", 50000)
	
	// Simulate chunking into many pieces
	numChunks := 100
	chunks := make([]string, numChunks)
	for i := 0; i < numChunks; i++ {
		chunks[i] = fmt.Sprintf("Chunk %d content", i)
	}
	
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", mock.AnythingOfType("string"), mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	
	// Setup embedder expectations for all chunks
	for _, chunk := range chunks {
		embedder.On("Embed", mock.Anything, chunk).Return(embeddings, nil)
		keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil).Once()
	}
	
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(nil)
	
	req := domain.IngestRequest{
		Content: largeContent,
	}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, numChunks, resp.ChunkCount)
	
	embedder.AssertExpectations(t)
	chunker.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
	documentStore.AssertExpectations(t)
}

// TestQuery_NoResults tests query with no matching results
func TestQuery_NoResults(t *testing.T) {
	service, embedder, _, _, vectorStore, keywordStore, _, _ := createMockService()
	
	query := "no matching results"
	queryEmbedding := []float64{0.1, 0.2, 0.3}
	
	embedder.On("Embed", mock.Anything, query).Return(queryEmbedding, nil)
	vectorStore.On("Search", mock.Anything, queryEmbedding, 5).Return([]domain.Chunk{}, nil)
	keywordStore.On("Search", mock.Anything, query, 5).Return([]domain.Chunk{}, nil)
	
	req := domain.QueryRequest{
		Query: query,
		TopK:  5,
	}
	
	resp, err := service.Query(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, resp.Answer, "找不到相关信息")
	assert.Empty(t, resp.Sources)
	
	embedder.AssertExpectations(t)
	vectorStore.AssertExpectations(t)
	keywordStore.AssertExpectations(t)
}

// TestIngest_MultipleContentSources tests validation of multiple content sources
func TestIngest_MultipleContentSources(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	req := domain.IngestRequest{
		FilePath: "/path/to/file.txt",
		URL:      "https://example.com",
		Content:  "Direct content",
	}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple content sources")
	assert.Equal(t, domain.IngestResponse{}, resp)
}

// TestReadFile_UnsupportedFormat tests handling of unsupported file formats
func TestReadFile_UnsupportedFormat(t *testing.T) {
	service, _, _, _, _, _, _, _ := createMockService()
	
	tempDir := t.TempDir()
	unsupportedFile := filepath.Join(tempDir, "test.xyz")
	err := os.WriteFile(unsupportedFile, []byte("content"), 0644)
	require.NoError(t, err)
	
	content, err := service.readFile(unsupportedFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
	assert.Empty(t, content)
}

// TestIngest_KeywordStoreError tests handling of keyword store failures
func TestIngest_KeywordStoreError(t *testing.T) {
	service, embedder, _, chunker, _, keywordStore, _, _ := createMockService()
	
	content := "Test content"
	chunks := []string{"Test chunk"}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, "Test chunk").Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(errors.New("keyword store error"))
	
	req := domain.IngestRequest{Content: content}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to index chunk")
	assert.Equal(t, domain.IngestResponse{}, resp)
}

// TestIngest_VectorStoreError tests handling of vector store failures
func TestIngest_VectorStoreError(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, _, _ := createMockService()
	
	content := "Test content"
	chunks := []string{"Test chunk"}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, "Test chunk").Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(errors.New("vector store error"))
	
	req := domain.IngestRequest{Content: content}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store vectors")
	assert.Equal(t, domain.IngestResponse{}, resp)
}

// TestIngest_DocumentStoreError tests handling of document store failures
func TestIngest_DocumentStoreError(t *testing.T) {
	service, embedder, _, chunker, vectorStore, keywordStore, documentStore, _ := createMockService()
	
	content := "Test content"
	chunks := []string{"Test chunk"}
	embeddings := []float64{0.1, 0.2, 0.3}
	
	chunker.On("Split", content, mock.AnythingOfType("domain.ChunkOptions")).Return(chunks, nil)
	embedder.On("Embed", mock.Anything, "Test chunk").Return(embeddings, nil)
	keywordStore.On("Index", mock.Anything, mock.AnythingOfType("domain.Chunk")).Return(nil)
	vectorStore.On("Store", mock.Anything, mock.AnythingOfType("[]domain.Chunk")).Return(nil)
	documentStore.On("Store", mock.Anything, mock.AnythingOfType("domain.Document")).Return(errors.New("document store error"))
	
	req := domain.IngestRequest{Content: content}
	
	resp, err := service.Ingest(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store document")
	assert.Equal(t, domain.IngestResponse{}, resp)
}