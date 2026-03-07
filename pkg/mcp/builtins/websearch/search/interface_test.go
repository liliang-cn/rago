package search

import (
	"testing"
	"time"
)

func TestSearchResult(t *testing.T) {
	result := SearchResult{
		Title:       "Test Title",
		URL:         "http://example.com",
		Snippet:     "Test snippet",
		Content:     "Full content",
		Engine:      "test",
		ExtractedAt: time.Now(),
	}

	if result.Title != "Test Title" {
		t.Errorf("expected Title='Test Title', got %s", result.Title)
	}

	if result.URL != "http://example.com" {
		t.Errorf("expected URL='http://example.com', got %s", result.URL)
	}

	if result.Engine != "test" {
		t.Errorf("expected Engine='test', got %s", result.Engine)
	}
}

func TestSearchOptions(t *testing.T) {
	opts := SearchOptions{
		MaxResults:     10,
		ExtractContent: true,
		Engines:        []string{"bing", "brave"},
		Timeout:        30 * time.Second,
	}

	if opts.MaxResults != 10 {
		t.Errorf("expected MaxResults=10, got %d", opts.MaxResults)
	}

	if !opts.ExtractContent {
		t.Error("expected ExtractContent=true")
	}

	if len(opts.Engines) != 2 {
		t.Errorf("expected 2 engines, got %d", len(opts.Engines))
	}

	if opts.Timeout != 30*time.Second {
		t.Errorf("expected Timeout=30s, got %v", opts.Timeout)
	}
}
