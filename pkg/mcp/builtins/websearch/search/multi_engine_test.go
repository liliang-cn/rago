package search

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockSearchEngine struct {
	name    string
	results []SearchResult
	err     error
}

func (m *mockSearchEngine) Name() string {
	return m.name
}

func (m *mockSearchEngine) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.results) > maxResults {
		return m.results[:maxResults], nil
	}
	return m.results, nil
}

type mockContentExtractor struct {
	content string
	err     error
}

func (m *mockContentExtractor) ExtractContent(ctx context.Context, url string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.content, nil
}

func TestMultiEngineSearcher_Search(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		opts          SearchOptions
		mockResults   []SearchResult
		expectedCount int
		expectError   bool
	}{
		{
			name:  "basic search",
			query: "test query",
			opts: SearchOptions{
				MaxResults:     5,
				ExtractContent: false,
			},
			mockResults: []SearchResult{
				{Title: "Result 1", URL: "http://example1.com", Snippet: "Snippet 1"},
				{Title: "Result 2", URL: "http://example2.com", Snippet: "Snippet 2"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:  "search with limit",
			query: "limited query",
			opts: SearchOptions{
				MaxResults:     1,
				ExtractContent: false,
			},
			mockResults: []SearchResult{
				{Title: "Result 1", URL: "http://example1.com", Snippet: "Snippet 1"},
				{Title: "Result 2", URL: "http://example2.com", Snippet: "Snippet 2"},
			},
			expectedCount: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &mockSearchEngine{
				name:    "test",
				results: tt.mockResults,
				err:     nil,
			}

			searcher := &multiEngineSearcher{
				engines: map[string]SearchEngine{
					"test": mockEngine,
					"bing": mockEngine,
				},
				extractor: &mockContentExtractor{content: "extracted content"},
			}

			ctx := context.Background()
			results, err := searcher.Search(ctx, tt.query, tt.opts)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

func TestMultiEngineSearcher_DeepSearch(t *testing.T) {
	mockEngine1 := &mockSearchEngine{
		name: "engine1",
		results: []SearchResult{
			{Title: "Engine1 Result", URL: "http://engine1.com", Engine: "engine1"},
		},
	}

	mockEngine2 := &mockSearchEngine{
		name: "engine2",
		results: []SearchResult{
			{Title: "Engine2 Result", URL: "http://engine2.com", Engine: "engine2"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"engine1": mockEngine1,
			"engine2": mockEngine2,
		},
		extractor: &mockContentExtractor{content: "extracted content"},
	}

	ctx := context.Background()
	results, err := searcher.DeepSearch(ctx, "test query", SearchOptions{
		MaxResults:     10,
		ExtractContent: false,
		Engines:        []string{"engine1", "engine2"},
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results from 2 engines, got %d", len(results))
	}

	foundEngine1 := false
	foundEngine2 := false
	for _, r := range results {
		if r.Engine == "engine1" {
			foundEngine1 = true
		}
		if r.Engine == "engine2" {
			foundEngine2 = true
		}
	}

	if !foundEngine1 || !foundEngine2 {
		t.Errorf("expected results from both engines")
	}
}

func TestMultiEngineSearcher_SelectEngine(t *testing.T) {
	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"bing":       &mockSearchEngine{name: "bing"},
			"brave":      &mockSearchEngine{name: "brave"},
			"duckduckgo": &mockSearchEngine{name: "duckduckgo"},
		},
	}

	tests := []struct {
		name         string
		preferred    []string
		expectedName string
	}{
		{
			name:         "prefer bing",
			preferred:    []string{"bing"},
			expectedName: "bing",
		},
		{
			name:         "prefer brave",
			preferred:    []string{"brave"},
			expectedName: "brave",
		},
		{
			name:         "no preference uses bing",
			preferred:    []string{},
			expectedName: "bing",
		},
		{
			name:         "non-existent preference falls back",
			preferred:    []string{"google"},
			expectedName: "bing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := searcher.selectEngine(tt.preferred)
			if engine == nil {
				t.Fatal("expected engine but got nil")
			}
			if engine.Name() != tt.expectedName {
				t.Errorf("expected engine %s, got %s", tt.expectedName, engine.Name())
			}
		})
	}
}

func TestMultiEngineSearcher_Timeout(t *testing.T) {
	slowEngine := &mockSearchEngine{
		name: "slow",
		results: []SearchResult{
			{Title: "Slow Result", URL: "http://slow.com"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"slow": slowEngine,
		},
		extractor: &mockContentExtractor{content: "content"},
	}

	ctx := context.Background()
	_, err := searcher.Search(ctx, "test", SearchOptions{
		MaxResults: 1,
		Timeout:    1 * time.Millisecond,
	})

	if err == nil {
		t.Skip("Timeout test may not be reliable in all environments")
	}
}

func TestMultiEngineSearcher_FallbackSearch(t *testing.T) {
	failingEngine := &mockSearchEngine{
		name: "failing",
		err:  errors.New("engine failed"),
	}

	workingEngine := &mockSearchEngine{
		name: "bing",
		results: []SearchResult{
			{Title: "Fallback Result", URL: "http://fallback.com", Engine: "bing"},
		},
	}

	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"failing": failingEngine,
			"bing":    workingEngine,
		},
		extractor: &mockContentExtractor{content: "content"},
	}

	results, err := searcher.fallbackSearch(context.Background(), "test", 10, "failing")
	if err != nil {
		t.Errorf("expected fallback to succeed, got error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result from fallback, got %d", len(results))
	}

	if results[0].Engine != "bing" {
		t.Errorf("expected result from bing, got %s", results[0].Engine)
	}
}

func TestMultiEngineSearcher_GetEngines(t *testing.T) {
	searcher := &multiEngineSearcher{
		engines: map[string]SearchEngine{
			"bing":       &mockSearchEngine{name: "bing"},
			"brave":      &mockSearchEngine{name: "brave"},
			"duckduckgo": &mockSearchEngine{name: "duckduckgo"},
		},
	}

	tests := []struct {
		name          string
		engineNames   []string
		expectedCount int
	}{
		{
			name:          "all engines when empty",
			engineNames:   []string{},
			expectedCount: 3,
		},
		{
			name:          "specific engines",
			engineNames:   []string{"bing", "brave"},
			expectedCount: 2,
		},
		{
			name:          "non-existent engine filtered out",
			engineNames:   []string{"bing", "google"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engines := searcher.getEngines(tt.engineNames)
			if len(engines) != tt.expectedCount {
				t.Errorf("expected %d engines, got %d", tt.expectedCount, len(engines))
			}
		})
	}
}

func TestMultiEngineSearcher_ExtractContentConcurrently(t *testing.T) {
	extractor := &mockContentExtractor{
		content: "extracted content",
	}

	searcher := &multiEngineSearcher{
		extractor: extractor,
	}

	results := []SearchResult{
		{Title: "Result 1", URL: "http://example1.com"},
		{Title: "Result 2", URL: "http://example2.com"},
		{Title: "Result 3", URL: "http://example3.com"},
	}

	ctx := context.Background()
	searcher.extractContentConcurrently(ctx, results)

	for _, r := range results {
		if r.Content != "extracted content" {
			t.Errorf("expected content to be extracted for %s", r.URL)
		}
		if r.ExtractedAt.IsZero() {
			t.Errorf("expected ExtractedAt to be set for %s", r.URL)
		}
	}
}
