package search

import (
	"context"
	"errors"
	"testing"
)

func TestMultiEngineSearcher_SearchWithContent(t *testing.T) {
	mockEngine := &mockSearchEngine{
		name: "test",
		results: []SearchResult{
			{Title: "Result 1", URL: "http://example1.com"},
			{Title: "Result 2", URL: "http://example2.com"},
		},
	}

	extractor := &mockContentExtractor{
		content: "extracted content here",
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"test": mockEngine,
			"bing": mockEngine,
		},
		extractor: extractor,
	}

	ctx := context.Background()
	results, err := searcher.Search(ctx, "test query", SearchOptions{
		MaxResults:     2,
		ExtractContent: true,
		Timeout:        0, // Test default timeout
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Content != "extracted content here" {
			t.Errorf("expected content to be extracted")
		}
	}
}

func TestMultiEngineSearcher_DeepSearchNoEngines(t *testing.T) {
	searcher := &multiEngineSearcher{
		engines:   map[string]SearchEngine{},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	_, err := searcher.DeepSearch(ctx, "test", SearchOptions{
		MaxResults: 1,
		Engines:    []string{"nonexistent"},
	})

	if err == nil {
		t.Error("expected error when no engines available")
	}
}

func TestMultiEngineSearcher_SearchWithPreferredEngine(t *testing.T) {
	preferredEngine := &mockSearchEngine{
		name: "preferred",
		results: []SearchResult{
			{Title: "Preferred Result", URL: "http://preferred.com", Engine: "preferred"},
		},
	}

	otherEngine := &mockSearchEngine{
		name: "other",
		results: []SearchResult{
			{Title: "Other Result", URL: "http://other.com", Engine: "other"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"preferred": preferredEngine,
			"other":     otherEngine,
			"bing":      otherEngine,
		},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	results, err := searcher.Search(ctx, "test", SearchOptions{
		MaxResults: 1,
		Engines:    []string{"preferred"},
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Engine != "preferred" {
		t.Errorf("expected result from preferred engine, got %s", results[0].Engine)
	}
}

func TestMultiEngineSearcher_FallbackAllFail(t *testing.T) {
	failingEngine := &mockSearchEngine{
		name: "failing",
		err:  errors.New("engine failed"),
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"bing":       failingEngine,
			"brave":      failingEngine,
			"duckduckgo": failingEngine,
		},
		extractor: &mockContentExtractor{},
	}

	_, err := searcher.fallbackSearch(context.Background(), "test", 10, "primary")
	if err == nil {
		t.Error("expected error when all engines fail")
	}
}

func TestMultiEngineSearcher_DeepSearchPartialFailure(t *testing.T) {
	workingEngine := &mockSearchEngine{
		name: "working",
		results: []SearchResult{
			{Title: "Working Result", URL: "http://working.com", Engine: "working"},
		},
	}

	failingEngine := &mockSearchEngine{
		name: "failing",
		err:  errors.New("engine failed"),
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"working": workingEngine,
			"failing": failingEngine,
		},
		extractor: &mockContentExtractor{content: "content"},
	}

	ctx := context.Background()
	results, err := searcher.DeepSearch(ctx, "test", SearchOptions{
		MaxResults:     10,
		ExtractContent: false,
		Engines:        []string{"working", "failing"},
		Timeout:        0, // Test default timeout
	})

	if err != nil {
		t.Errorf("should succeed with partial results: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result from working engine, got %d", len(results))
	}
}

func TestMultiEngineSearcher_ExtractorFailure(t *testing.T) {
	mockEngine := &mockSearchEngine{
		name: "test",
		results: []SearchResult{
			{Title: "Result", URL: "http://example.com"},
		},
	}

	failingExtractor := &mockContentExtractor{
		err: errors.New("extraction failed"),
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"test": mockEngine,
			"bing": mockEngine,
		},
		extractor: failingExtractor,
	}

	ctx := context.Background()
	results, err := searcher.Search(ctx, "test", SearchOptions{
		MaxResults:     1,
		ExtractContent: true,
	})

	if err != nil {
		t.Errorf("search should succeed even if extraction fails: %v", err)
	}

	if results[0].Content != "" {
		t.Error("content should be empty when extraction fails")
	}

	if !results[0].ExtractedAt.IsZero() {
		t.Error("ExtractedAt should not be set when extraction fails")
	}
}

func TestMultiEngineSearcher_LimitResults(t *testing.T) {
	mockEngine := &mockSearchEngine{
		name: "test",
		results: []SearchResult{
			{Title: "Result 1", URL: "http://1.com"},
			{Title: "Result 2", URL: "http://2.com"},
			{Title: "Result 3", URL: "http://3.com"},
			{Title: "Result 4", URL: "http://4.com"},
			{Title: "Result 5", URL: "http://5.com"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"test":  mockEngine,
			"bing":  mockEngine,
			"brave": mockEngine,
		},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	results, err := searcher.DeepSearch(ctx, "test", SearchOptions{
		MaxResults: 2,
		Engines:    []string{"test", "bing", "brave"},
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected results to be limited to 2, got %d", len(results))
	}
}
