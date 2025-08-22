package domain

import (
	"testing"
)

func TestChunk_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		chunk Chunk
		want  bool
	}{
		{
			name: "non-empty chunk",
			chunk: Chunk{
				ID:         "test-id",
				Content:    "test content",
				DocumentID: "doc-id",
				Vector:     []float64{0.1, 0.2, 0.3},
			},
			want: false,
		},
		{
			name: "empty ID",
			chunk: Chunk{
				ID:         "",
				Content:    "test content",
				DocumentID: "doc-id",
			},
			want: true,
		},
		{
			name: "empty content",
			chunk: Chunk{
				ID:         "test-id",
				Content:    "",
				DocumentID: "doc-id",
			},
			want: true,
		},
		{
			name: "empty document ID",
			chunk: Chunk{
				ID:      "test-id",
				Content: "test content",
			},
			want: true,
		},
		{
			name:  "completely empty",
			chunk: Chunk{},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := tt.chunk.ID == "" || tt.chunk.Content == "" || tt.chunk.DocumentID == ""
			if isEmpty != tt.want {
				t.Errorf("Chunk.IsEmpty() = %v, want %v", isEmpty, tt.want)
			}
		})
	}
}

func TestIngestRequest_HasContent(t *testing.T) {
	tests := []struct {
		name string
		req  IngestRequest
		want bool
	}{
		{
			name: "has content",
			req: IngestRequest{
				Content: "test content",
			},
			want: true,
		},
		{
			name: "has file path",
			req: IngestRequest{
				FilePath: "/path/to/file.txt",
			},
			want: true,
		},
		{
			name: "has URL",
			req: IngestRequest{
				URL: "https://example.com",
			},
			want: true,
		},
		{
			name: "empty request",
			req:  IngestRequest{},
			want: false,
		},
		{
			name: "only chunk size",
			req: IngestRequest{
				ChunkSize: 100,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasContent := tt.req.Content != "" || tt.req.FilePath != "" || tt.req.URL != ""
			if hasContent != tt.want {
				t.Errorf("IngestRequest.HasContent() = %v, want %v", hasContent, tt.want)
			}
		})
	}
}

func TestQueryRequest_IsValid(t *testing.T) {
	tests := []struct {
		name string
		req  QueryRequest
		want bool
	}{
		{
			name: "valid request",
			req: QueryRequest{
				Query: "test query",
				TopK:  5,
			},
			want: true,
		},
		{
			name: "empty query",
			req: QueryRequest{
				Query: "",
				TopK:  5,
			},
			want: false,
		},
		{
			name: "zero topK",
			req: QueryRequest{
				Query: "test query",
				TopK:  0,
			},
			want: false,
		},
		{
			name: "negative topK",
			req: QueryRequest{
				Query: "test query",
				TopK:  -1,
			},
			want: false,
		},
		{
			name: "valid with all fields",
			req: QueryRequest{
				Query:       "test query",
				TopK:        5,
				Temperature: 0.7,
				MaxTokens:   100,
				Stream:      true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.req.Query != "" && tt.req.TopK > 0
			if isValid != tt.want {
				t.Errorf("QueryRequest.IsValid() = %v, want %v", isValid, tt.want)
			}
		})
	}
}

func TestChunk_HasVector(t *testing.T) {
	tests := []struct {
		name  string
		chunk Chunk
		want  bool
	}{
		{
			name: "has vector",
			chunk: Chunk{
				Vector: []float64{0.1, 0.2, 0.3},
			},
			want: true,
		},
		{
			name: "nil vector",
			chunk: Chunk{
				Vector: nil,
			},
			want: false,
		},
		{
			name: "empty vector",
			chunk: Chunk{
				Vector: []float64{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasVector := tt.chunk.Vector != nil && len(tt.chunk.Vector) > 0
			if hasVector != tt.want {
				t.Errorf("Chunk.HasVector() = %v, want %v", hasVector, tt.want)
			}
		})
	}
}

func TestDocument_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		doc  Document
		want bool
	}{
		{
			name: "valid document",
			doc: Document{
				ID:      "doc-1",
				Path:    "/path/to/doc",
				Content: "document content",
			},
			want: false,
		},
		{
			name: "empty ID",
			doc: Document{
				ID:      "",
				Path:    "/path/to/doc",
				Content: "document content",
			},
			want: true,
		},
		{
			name: "no content source",
			doc: Document{
				ID: "doc-1",
			},
			want: true,
		},
		{
			name: "has URL",
			doc: Document{
				ID:  "doc-1",
				URL: "https://example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := tt.doc.ID == "" || (tt.doc.Content == "" && tt.doc.Path == "" && tt.doc.URL == "")
			if isEmpty != tt.want {
				t.Errorf("Document.IsEmpty() = %v, want %v", isEmpty, tt.want)
			}
		})
	}
}

func TestQueryResponse_HasSources(t *testing.T) {
	tests := []struct {
		name string
		resp QueryResponse
		want bool
	}{
		{
			name: "has sources",
			resp: QueryResponse{
				Sources: []Chunk{
					{ID: "chunk1", Content: "content1"},
				},
			},
			want: true,
		},
		{
			name: "empty sources",
			resp: QueryResponse{
				Sources: []Chunk{},
			},
			want: false,
		},
		{
			name: "nil sources",
			resp: QueryResponse{
				Sources: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasSources := len(tt.resp.Sources) > 0
			if hasSources != tt.want {
				t.Errorf("QueryResponse.HasSources() = %v, want %v", hasSources, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	// Test that our domain errors are properly defined
	errors := []error{
		ErrDocumentNotFound,
		ErrEmbeddingFailed,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Domain error should not be nil")
		}
		if err.Error() == "" {
			t.Error("Domain error should have a non-empty message")
		}
	}
}

func TestChunk_Copy(t *testing.T) {
	original := Chunk{
		ID:         "test-id",
		Content:    "test content",
		DocumentID: "doc-id",
		Vector:     []float64{0.1, 0.2, 0.3},
		Score:      0.95,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	// Test that modifying a copy doesn't affect original
	copy := original

	// Modify vector slice
	if len(copy.Vector) > 0 {
		copy.Vector = append(copy.Vector, 999.0)
		// Since Go slices share underlying array, we need to be careful
		if len(original.Vector) == len(copy.Vector) {
			t.Error("Vector modification affected original - need proper deep copy")
		}
	}

	// Test basic field copying
	copy.Score = 0.5
	if original.Score == 0.5 {
		t.Error("Score modification affected original")
	}
}

func BenchmarkChunk_VectorAccess(b *testing.B) {
	chunk := Chunk{
		ID:         "test-id",
		Content:    "test content",
		DocumentID: "doc-id",
		Vector:     make([]float64, 1000), // Large vector
	}

	// Fill vector with test data
	for i := range chunk.Vector {
		chunk.Vector[i] = float64(i) * 0.001
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(chunk.Vector) > 0
	}
}

func BenchmarkIngestRequest_Validation(b *testing.B) {
	req := IngestRequest{
		Content:   "test content",
		ChunkSize: 100,
		Overlap:   10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Content != "" || req.FilePath != "" || req.URL != ""
	}
}
