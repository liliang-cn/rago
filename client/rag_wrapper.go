package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/rag"
)

// RAGWrapper wraps the RAG client to provide additional methods
type RAGWrapper struct {
	client *rag.Client
}

// NewRAGWrapper creates a new RAG wrapper
func NewRAGWrapper(client *rag.Client) *RAGWrapper {
	return &RAGWrapper{client: client}
}

// Ingest ingests text directly (simplified method)
func (r *RAGWrapper) Ingest(text string) error {
	if r.client == nil {
		return fmt.Errorf("RAG client not initialized")
	}
	
	ctx := context.Background()
	opts := &rag.IngestOptions{
		ChunkSize: 1000,
		Overlap:   200,
	}
	
	_, err := r.client.IngestText(ctx, text, "direct-input", opts)
	return err
}

// Query performs a simple query (simplified method)
func (r *RAGWrapper) Query(query string) (string, error) {
	ctx := context.Background()
	opts := &QueryOptions{
		TopK:        5,
		ShowSources: false,
	}
	resp, err := r.QueryWithOptions(ctx, query, opts)
	if err != nil {
		return "", err
	}
	return resp.Answer, nil
}

// QueryWithOptions performs a RAG query with specific options
func (r *RAGWrapper) QueryWithOptions(ctx context.Context, query string, opts *QueryOptions) (*QueryResponse, error) {
	if r.client == nil {
		return nil, fmt.Errorf("RAG client not initialized")
	}

	// Prepare options
	ragOpts := &rag.QueryOptions{
		TopK:        opts.TopK,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		ShowSources: opts.ShowSources,
	}

	// Perform query
	result, err := r.client.Query(ctx, query, ragOpts)
	if err != nil {
		return nil, err
	}

	// Convert to response
	resp := &QueryResponse{
		Answer: result.Answer,
	}

	if opts.ShowSources && result.Sources != nil {
		for _, src := range result.Sources {
			resp.Sources = append(resp.Sources, SearchResult{
				Score:    src.Score,
				Content:  src.Content,
				Metadata: src.Metadata,
			})
		}
	}

	return resp, nil
}

// IngestWithOptions ingests documents with specific options
func (r *RAGWrapper) IngestWithOptions(ctx context.Context, path string, opts *IngestOptions) error {
	if r.client == nil {
		return fmt.Errorf("RAG client not initialized")
	}

	// Delegate to RAG client
	ragOpts := &rag.IngestOptions{
		ChunkSize: opts.ChunkSize,
		Overlap:   opts.Overlap,
		Metadata:  make(map[string]interface{}),
	}

	// Convert metadata if provided
	if opts.Metadata != nil {
		for k, v := range opts.Metadata {
			ragOpts.Metadata[k] = v
		}
	}

	_, err := r.client.IngestFile(ctx, path, ragOpts)
	return err
}

// SearchWithOptions performs a search with specific options
func (r *RAGWrapper) SearchWithOptions(ctx context.Context, query string, opts *SearchOptions) ([]SearchResult, error) {
	if r.client == nil {
		return nil, fmt.Errorf("RAG client not initialized")
	}

	// Use QueryWithOptions internally
	queryOpts := &QueryOptions{
		TopK:        opts.TopK,
		ShowSources: true,
		Filters:     opts.Filters,
	}

	resp, err := r.QueryWithOptions(ctx, query, queryOpts)
	if err != nil {
		return nil, err
	}

	return resp.Sources, nil
}
