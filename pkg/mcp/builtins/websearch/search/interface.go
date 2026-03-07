package search

import (
	"context"
	"time"
)

type SearchResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Snippet     string    `json:"snippet"`
	Content     string    `json:"content,omitempty"`
	Engine      string    `json:"engine"`
	ExtractedAt time.Time `json:"extracted_at,omitempty"`
}

type SearchOptions struct {
	MaxResults     int
	ExtractContent bool
	Engines        []string
	Timeout        time.Duration
}

type SearchEngine interface {
	Name() string
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
}

type ContentExtractor interface {
	ExtractContent(ctx context.Context, url string) (string, error)
}

type MultiEngineSearcher interface {
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	DeepSearch(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
}
