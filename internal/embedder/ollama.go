package embedder

import (
	"context"
	"fmt"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/internal/domain"
)

type OllamaService struct {
	client  *ollama.Client
	model   string
	baseURL string
}

func NewOllamaService(baseURL, model string) (*OllamaService, error) {
	client, err := ollama.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &OllamaService{
		client:  client,
		model:   model,
		baseURL: baseURL,
	}, nil
}

func (s *OllamaService) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: empty text", domain.ErrInvalidInput)
	}

	req := &ollama.EmbedRequest{
		Model: s.model,
		Input: text,
	}

	resp, err := s.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrEmbeddingFailed, err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("%w: empty embedding response", domain.ErrEmbeddingFailed)
	}

	return resp.Embeddings[0], nil
}

func (s *OllamaService) Health(ctx context.Context) error {
	_, err := s.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}