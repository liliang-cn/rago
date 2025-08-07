package processor

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/internal/domain"
)

type mockEmbedder struct {
	embedding []float64
	err       error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

type mockGenerator struct {
	response string
	err      error
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if m.err != nil {
		return m.err
	}
	callback(m.response)
	return nil
}

type mockChunker struct {
	chunks []string
	err    error
}

func (m *mockChunker) Split(text string, options domain.ChunkOptions) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.chunks, nil
}

type mockVectorStore struct {
	chunks []domain.Chunk
	err    error
}

func (m *mockVectorStore) Store(ctx context.Context, chunks []domain.Chunk) error {
	if m.err != nil {
		return m.err
	}
	m.chunks = append(m.chunks, chunks...)
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, vector []float64, topK int) ([]domain.Chunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.chunks) > topK {
		return m.chunks[:topK], nil
	}
	return m.chunks, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, documentID string) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockVectorStore) List(ctx context.Context) ([]domain.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []domain.Document{}, nil
}

func (m *mockVectorStore) Reset(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	m.chunks = nil
	return nil
}

type mockDocumentStore struct {
	documents map[string]domain.Document
	err       error
}

func newMockDocumentStore() *mockDocumentStore {
	return &mockDocumentStore{
		documents: make(map[string]domain.Document),
	}
}

func (m *mockDocumentStore) Store(ctx context.Context, doc domain.Document) error {
	if m.err != nil {
		return m.err
	}
	m.documents[doc.ID] = doc
	return nil
}

func (m *mockDocumentStore) Get(ctx context.Context, id string) (domain.Document, error) {
	if m.err != nil {
		return domain.Document{}, m.err
	}
	doc, exists := m.documents[id]
	if !exists {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, nil
}

func (m *mockDocumentStore) List(ctx context.Context) ([]domain.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	var docs []domain.Document
	for _, doc := range m.documents {
		docs = append(docs, doc)
	}
	return docs, nil
}

func (m *mockDocumentStore) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.documents, id)
	return nil
}

func TestService_Ingest(t *testing.T) {
	tests := []struct {
		name    string
		request domain.IngestRequest
		setup   func() *Service
		wantErr bool
	}{
		{
			name: "successful ingest with content",
			request: domain.IngestRequest{
				Content:   "This is test content. It has multiple sentences.",
				ChunkSize: 50,
				Overlap:   10,
			},
			setup: func() *Service {
				return New(
					&mockEmbedder{embedding: []float64{0.1, 0.2, 0.3}},
					&mockGenerator{response: "test response"},
					&mockChunker{chunks: []string{"This is test content.", "It has multiple sentences."}},
					&mockVectorStore{},
					newMockDocumentStore(),
				)
			},
			wantErr: false,
		},
		{
			name: "empty content",
			request: domain.IngestRequest{
				ChunkSize: 50,
				Overlap:   10,
			},
			setup: func() *Service {
				return New(
					&mockEmbedder{},
					&mockGenerator{},
					&mockChunker{},
					&mockVectorStore{},
					newMockDocumentStore(),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.setup()
			ctx := context.Background()
			
			resp, err := service.Ingest(ctx, tt.request)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Ingest() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Ingest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !resp.Success {
				t.Errorf("Ingest() success = %v, expected true", resp.Success)
			}
		})
	}
}

func TestService_Query(t *testing.T) {
	tests := []struct {
		name    string
		request domain.QueryRequest
		setup   func() *Service
		wantErr bool
	}{
		{
			name: "successful query",
			request: domain.QueryRequest{
				Query: "test query",
				TopK:  3,
			},
			setup: func() *Service {
				mockVS := &mockVectorStore{
					chunks: []domain.Chunk{
						{ID: "1", Content: "chunk 1", Score: 0.9},
						{ID: "2", Content: "chunk 2", Score: 0.8},
					},
				}
				return New(
					&mockEmbedder{embedding: []float64{0.1, 0.2, 0.3}},
					&mockGenerator{response: "test response"},
					&mockChunker{},
					mockVS,
					newMockDocumentStore(),
				)
			},
			wantErr: false,
		},
		{
			name: "empty query",
			request: domain.QueryRequest{
				Query: "",
				TopK:  3,
			},
			setup: func() *Service {
				return New(
					&mockEmbedder{},
					&mockGenerator{},
					&mockChunker{},
					&mockVectorStore{},
					newMockDocumentStore(),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.setup()
			ctx := context.Background()
			
			resp, err := service.Query(ctx, tt.request)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Query() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if resp.Answer == "" {
				t.Errorf("Query() answer is empty")
			}
			
			if resp.Elapsed == "" {
				t.Errorf("Query() elapsed time is empty")
			}
		})
	}
}

func TestService_validateIngestRequest(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name    string
		request domain.IngestRequest
		wantErr bool
	}{
		{
			name: "valid content request",
			request: domain.IngestRequest{
				Content: "test content",
			},
			wantErr: false,
		},
		{
			name: "valid file path request",
			request: domain.IngestRequest{
				FilePath: "/path/to/file.txt",
			},
			wantErr: false,
		},
		{
			name: "valid URL request",
			request: domain.IngestRequest{
				URL: "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "no content source",
			request: domain.IngestRequest{},
			wantErr: true,
		},
		{
			name: "multiple content sources",
			request: domain.IngestRequest{
				Content:  "test content",
				FilePath: "/path/to/file.txt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateIngestRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIngestRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}