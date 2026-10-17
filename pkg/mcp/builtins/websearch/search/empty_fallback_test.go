package search

import (
	"context"
	"testing"
)

// An engine that answers "no results, no error" is one the searcher used to
// take at its word: Bing's HTML changed, its scraper returned nil, nil, and
// because the fallback only ran on err != nil the basic tool answered every
// query with zero results while duckduckgo would have answered it fine.
func TestSearch_EmptyResultFallsBackToNextEngine(t *testing.T) {
	want := []SearchResult{{Title: "hit", URL: "https://example.com", Engine: "duckduckgo"}}
	engines := map[string]SearchEngine{
		"bing":       &mockSearchEngine{name: "bing"},  // empty, no error
		"brave":      &mockSearchEngine{name: "brave"}, // empty, no error
		"duckduckgo": &mockSearchEngine{name: "duckduckgo", results: want},
	}

	for name, searcher := range map[string]MultiEngineSearcher{
		"hybrid": &HybridMultiEngineSearcher{engines: engines},
		"basic":  &multiEngineSearcher{engines: engines, extractor: &mockContentExtractor{}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := searcher.Search(context.Background(), "q", SearchOptions{MaxResults: 5})
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if len(got) != 1 || got[0].Engine != "duckduckgo" {
				t.Fatalf("expected duckduckgo's result after empty engines, got %+v", got)
			}
		})
	}
}

func TestSearch_AllEnginesEmptyIsAnError(t *testing.T) {
	engines := map[string]SearchEngine{
		"bing":       &mockSearchEngine{name: "bing"},
		"brave":      &mockSearchEngine{name: "brave"},
		"duckduckgo": &mockSearchEngine{name: "duckduckgo"},
	}
	h := &HybridMultiEngineSearcher{engines: engines}
	if _, err := h.Search(context.Background(), "q", SearchOptions{MaxResults: 5}); err == nil {
		t.Fatal("expected an error when every engine returns nothing; an empty success hides a broken scraper")
	}
}

func TestSearch_PreferredEngineEmptyFallsBack(t *testing.T) {
	want := []SearchResult{{Title: "hit", URL: "https://example.com", Engine: "bing"}}
	engines := map[string]SearchEngine{
		"bing":       &mockSearchEngine{name: "bing", results: want},
		"duckduckgo": &mockSearchEngine{name: "duckduckgo"},
	}
	h := &HybridMultiEngineSearcher{engines: engines}
	got, err := h.Search(context.Background(), "q", SearchOptions{MaxResults: 5, Engines: []string{"duckduckgo"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || got[0].Engine != "bing" {
		t.Fatalf("expected fallback to bing, got %+v", got)
	}
}
