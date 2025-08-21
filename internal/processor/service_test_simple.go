package processor

import (
	"context"
	"errors"
	"testing"
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

func TestMockEmbedder_Embed(t *testing.T) {
	tests := []struct {
		name      string
		embedder  *mockEmbedder
		text      string
		wantError bool
	}{
		{
			name: "successful embedding",
			embedder: &mockEmbedder{
				embedding: []float64{0.1, 0.2, 0.3},
			},
			text:      "test text",
			wantError: false,
		},
		{
			name: "embedding error",
			embedder: &mockEmbedder{
				err: errors.New("embedding failed"),
			},
			text:      "test text",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.embedder.Embed(context.Background(), tt.text)
			
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if len(result) != len(tt.embedder.embedding) {
				t.Errorf("Expected embedding length %d, got %d", len(tt.embedder.embedding), len(result))
			}
		})
	}
}