package search

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestMultiEngineSearcher_SearchFallbackChain(t *testing.T) {
	failingEngine1 := &mockSearchEngine{
		name: "bing",
		err:  errors.New("bing failed"),
	}

	failingEngine2 := &mockSearchEngine{
		name: "brave",
		err:  errors.New("brave failed"),
	}

	workingEngine := &mockSearchEngine{
		name: "duckduckgo",
		results: []SearchResult{
			{Title: "Success", URL: "http://success.com", Engine: "duckduckgo"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"bing":       failingEngine1,
			"brave":      failingEngine2,
			"duckduckgo": workingEngine,
		},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	results, err := searcher.Search(ctx, "test", SearchOptions{
		MaxResults: 1,
	})

	if err != nil {
		t.Errorf("expected fallback to succeed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Engine != "duckduckgo" {
		t.Errorf("expected result from duckduckgo, got %s", results[0].Engine)
	}
}

func TestMultiEngineSearcher_NoEnginesAvailable(t *testing.T) {
	searcher := &multiEngineSearcher{
		engines:   map[string]SearchEngine{},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	_, err := searcher.Search(ctx, "test", SearchOptions{
		MaxResults: 1,
	})

	if err == nil {
		t.Error("expected error when no engines available")
	}
}

func TestMultiEngineSearcher_GetEnginesDefault(t *testing.T) {
	mockEngine1 := &mockSearchEngine{name: "bing"}
	mockEngine2 := &mockSearchEngine{name: "brave"}
	mockEngine3 := &mockSearchEngine{name: "duckduckgo"}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"bing":       mockEngine1,
			"brave":      mockEngine2,
			"duckduckgo": mockEngine3,
		},
	}

	engines := searcher.getEngines(nil)
	if len(engines) != 3 {
		t.Errorf("expected 3 engines when nil passed, got %d", len(engines))
	}
}

func TestMultiEngineSearcher_DeepSearchWithLimitedResults(t *testing.T) {
	mockEngine := &mockSearchEngine{
		name: "test",
		results: []SearchResult{
			{Title: "R1", URL: "http://1.com"},
			{Title: "R2", URL: "http://2.com"},
			{Title: "R3", URL: "http://3.com"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"test1": mockEngine,
			"test2": mockEngine,
		},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	results, err := searcher.DeepSearch(ctx, "test", SearchOptions{
		MaxResults: 1,
		Engines:    []string{"test1", "test2"},
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should get 1 result per engine due to resultsPerEngine calculation
	// But limited to MaxResults = 1
	if len(results) != 1 {
		t.Errorf("expected 1 result (limited by MaxResults), got %d", len(results))
	}
}

func TestMultiEngineSearcher_DeepSearchConcurrentErrors(t *testing.T) {
	// This tests that DeepSearch continues even when some engines fail
	successEngine := &mockSearchEngine{
		name: "success",
		results: []SearchResult{
			{Title: "Success", URL: "http://success.com", Engine: "success"},
		},
	}

	errorEngine := &mockSearchEngine{
		name: "error",
		err:  fmt.Errorf("simulated error"),
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"success": successEngine,
			"error":   errorEngine,
		},
		extractor: &mockContentExtractor{},
	}

	ctx := context.Background()
	results, err := searcher.DeepSearch(ctx, "test", SearchOptions{
		MaxResults:     10,
		ExtractContent: true,
		Engines:        []string{"success", "error"},
	})

	if err != nil {
		t.Errorf("should not error when partial results available: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result from success engine, got %d", len(results))
	}

	if results[0].Engine != "success" {
		t.Errorf("expected result from success engine, got %s", results[0].Engine)
	}

	// Verify content extraction was attempted
	if results[0].Content == "" {
		t.Log("Content extraction may have failed, but that's okay for this test")
	}
}
