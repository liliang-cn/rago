package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/liliang-cn/rago/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProcessor is a mock implementation of the processor interface
type MockProcessor struct {
	mock.Mock
}

func (m *MockProcessor) Ingest(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(domain.IngestResponse), args.Error(1)
}

func (m *MockProcessor) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return domain.QueryResponse{}, args.Error(1)
	}
	return args.Get(0).(domain.QueryResponse), args.Error(1)
}

func (m *MockProcessor) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Document), args.Error(1)
}

func (m *MockProcessor) DeleteDocument(ctx context.Context, documentID string) error {
	args := m.Called(ctx, documentID)
	return args.Error(0)
}

func (m *MockProcessor) Reset(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestRAGSearchTool_Name(t *testing.T) {
	tool := NewRAGSearchTool(nil)
	assert.Equal(t, "rag_search", tool.Name())
}

func TestRAGSearchTool_Description(t *testing.T) {
	tool := NewRAGSearchTool(nil)
	assert.NotEmpty(t, tool.Description())
}

func TestRAGSearchTool_Parameters(t *testing.T) {
	tool := NewRAGSearchTool(nil)
	params := tool.Parameters()

	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Required, "query")
	assert.Contains(t, params.Properties, "query")
	assert.Contains(t, params.Properties, "top_k")
	assert.Contains(t, params.Properties, "filters")
}

func TestRAGSearchTool_Validate(t *testing.T) {
	tool := NewRAGSearchTool(nil)

	// Valid case
	err := tool.Validate(map[string]interface{}{
		"query": "test query",
	})
	assert.NoError(t, err)

	// Missing query
	err = tool.Validate(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query parameter is required")

	// Empty query
	err = tool.Validate(map[string]interface{}{
		"query": "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-empty string")

	// Invalid top_k
	err = tool.Validate(map[string]interface{}{
		"query": "test",
		"top_k": 25, // exceeds max
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 20")

	// Valid with filters
	err = tool.Validate(map[string]interface{}{
		"query": "test",
		"filters": map[string]interface{}{
			"source": "book",
		},
	})
	assert.NoError(t, err)
}

func TestRAGSearchTool_Execute_Success(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewRAGSearchTool(mockProcessor)

	// Mock the processor response
	mockChunks := []domain.Chunk{
		{
			ID:         "chunk1",
			DocumentID: "doc1",
			Content:    "This is test content",
			Score:      0.95,
			Metadata: map[string]interface{}{
				"source": "test.txt",
			},
		},
	}

	mockResponse := domain.QueryResponse{
		Sources: mockChunks,
		Elapsed: "10ms",
	}

	mockProcessor.On("Query", mock.Anything, mock.MatchedBy(func(req domain.QueryRequest) bool {
		return req.Query == "test query" && req.TopK == 5
	})).Return(mockResponse, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test query",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "test query", data["query"])
	assert.Equal(t, 1, data["total_found"])
	assert.Equal(t, "10ms", data["search_time"])

	results, ok := data["results"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, results, 1)

	assert.Equal(t, 1, results[0]["rank"])
	assert.Equal(t, 0.95, results[0]["score"])
	assert.Equal(t, "This is test content", results[0]["content"])

	mockProcessor.AssertExpectations(t)
}

func TestRAGSearchTool_Execute_WithFilters(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewRAGSearchTool(mockProcessor)

	mockResponse := domain.QueryResponse{
		Sources: []domain.Chunk{},
		Elapsed: "5ms",
	}

	mockProcessor.On("Query", mock.Anything, mock.MatchedBy(func(req domain.QueryRequest) bool {
		return req.Query == "test" && req.Filters != nil && req.Filters["source"] == "book"
	})).Return(mockResponse, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test",
		"filters": map[string]interface{}{
			"source": "book",
		},
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	mockProcessor.AssertExpectations(t)
}

func TestRAGSearchTool_Execute_MissingQuery(t *testing.T) {
	tool := NewRAGSearchTool(nil)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "query parameter is required")
}

func TestDocumentInfoTool_Name(t *testing.T) {
	tool := NewDocumentInfoTool(nil)
	assert.Equal(t, "document_info", tool.Name())
}

func TestDocumentInfoTool_Execute_Count(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewDocumentInfoTool(mockProcessor)

	mockDocs := []domain.Document{
		{ID: "1", Path: "doc1.txt", Created: time.Now()},
		{ID: "2", Path: "doc2.txt", Created: time.Now()},
	}

	mockProcessor.On("ListDocuments", mock.Anything).Return(mockDocs, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "count",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, data["count"])

	mockProcessor.AssertExpectations(t)
}

func TestDocumentInfoTool_Execute_List(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewDocumentInfoTool(mockProcessor)

	now := time.Now()
	mockDocs := []domain.Document{
		{
			ID:      "1",
			Path:    "doc1.txt",
			Created: now,
			Metadata: map[string]interface{}{
				"source": "upload",
			},
		},
	}

	mockProcessor.On("ListDocuments", mock.Anything).Return(mockDocs, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action": "list",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, 1, data["total"])

	docs, ok := data["documents"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, docs, 1)

	assert.Equal(t, "1", docs[0]["id"])
	assert.Equal(t, "doc1.txt", docs[0]["path"])
	assert.Contains(t, docs[0], "metadata")

	mockProcessor.AssertExpectations(t)
}

func TestDocumentInfoTool_Execute_Get(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewDocumentInfoTool(mockProcessor)

	now := time.Now()
	mockDocs := []domain.Document{
		{
			ID:      "test-doc",
			Path:    "test.txt",
			Created: now,
			Metadata: map[string]interface{}{
				"author": "test user",
			},
		},
	}

	mockProcessor.On("ListDocuments", mock.Anything).Return(mockDocs, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":      "get",
		"document_id": "test-doc",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "test-doc", data["id"])
	assert.Equal(t, "test.txt", data["path"])
	assert.Contains(t, data, "metadata")

	mockProcessor.AssertExpectations(t)
}

func TestDocumentInfoTool_Execute_GetNotFound(t *testing.T) {
	mockProcessor := &MockProcessor{}
	tool := NewDocumentInfoTool(mockProcessor)

	mockDocs := []domain.Document{
		{ID: "other-doc", Path: "other.txt", Created: time.Now()},
	}

	mockProcessor.On("ListDocuments", mock.Anything).Return(mockDocs, nil)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"action":      "get",
		"document_id": "nonexistent",
	})

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")

	mockProcessor.AssertExpectations(t)
}

func TestDocumentInfoTool_Validate(t *testing.T) {
	tool := NewDocumentInfoTool(nil)

	// Valid count action
	err := tool.Validate(map[string]interface{}{
		"action": "count",
	})
	assert.NoError(t, err)

	// Valid get action with document_id
	err = tool.Validate(map[string]interface{}{
		"action":      "get",
		"document_id": "test",
	})
	assert.NoError(t, err)

	// Missing action
	err = tool.Validate(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action parameter is required")

	// Invalid action
	err = tool.Validate(map[string]interface{}{
		"action": "invalid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")

	// Get action without document_id
	err = tool.Validate(map[string]interface{}{
		"action": "get",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document_id parameter is required")
}
