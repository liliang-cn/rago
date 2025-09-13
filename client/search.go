package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/rag"
)

// SearchOptions configures the search parameters
type SearchOptions struct {
	// Number of results to return
	TopK int

	// Minimum score threshold for results
	ScoreThreshold float64

	// Whether to include metadata in results
	IncludeMetadata bool

	// Whether to use hybrid search (vector + keyword)
	HybridSearch bool

	// Weight for vector search in hybrid mode (0-1)
	VectorWeight float64
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		TopK:            5,
		ScoreThreshold:  0.0,
		IncludeMetadata: true,
		HybridSearch:    true,
		VectorWeight:    0.7,
	}
}

// SearchResult represents a search result
type SearchResult struct {
	Content  string
	Score    float64
	Metadata map[string]interface{}
	Source   string
}

// Search performs a search on the knowledge base
func (c *Client) Search(ctx context.Context, query string, opts *SearchOptions) ([]SearchResult, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	if c.ragClient == nil {
		return nil, fmt.Errorf("RAG client not initialized")
	}

	// Perform the search using RAG client
	ragOpts := &rag.QueryOptions{
		TopK:        opts.TopK,
		ShowSources: true,
	}

	resp, err := c.ragClient.Query(ctx, query, ragOpts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to SearchResult
	results := make([]SearchResult, 0, len(resp.Sources))
	for _, chunk := range resp.Sources {
		result := SearchResult{
			Content: chunk.Content,
			Score:   chunk.Score,
			Source:  chunk.DocumentID,
		}

		if opts.IncludeMetadata && chunk.Metadata != nil {
			result.Metadata = chunk.Metadata
		}

		results = append(results, result)
	}

	return results, nil
}

// SearchWithContext performs a search and returns results with context
func (c *Client) SearchWithContext(ctx context.Context, query string, opts *SearchOptions) (string, []SearchResult, error) {
	results, err := c.Search(ctx, query, opts)
	if err != nil {
		return "", nil, err
	}

	if len(results) == 0 {
		return "", results, nil
	}

	// Build context from results
	context := "Relevant information from knowledge base:\n"
	for i, result := range results {
		context += fmt.Sprintf("\n[%d] (Score: %.2f) %s\n", i+1, result.Score, result.Content)
		if result.Source != "" {
			context += fmt.Sprintf("   Source: %s\n", result.Source)
		}
	}

	return context, results, nil
}
