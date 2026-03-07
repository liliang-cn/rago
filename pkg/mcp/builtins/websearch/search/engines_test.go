package search

import (
	"testing"
)

func TestBingSearchEngine_Name(t *testing.T) {
	engine := NewBingSearchEngine()
	if engine.Name() != "bing" {
		t.Errorf("expected name 'bing', got %s", engine.Name())
	}
}

func TestBraveSearchEngine_Name(t *testing.T) {
	engine := NewBraveSearchEngine()
	if engine.Name() != "brave" {
		t.Errorf("expected name 'brave', got %s", engine.Name())
	}
}

func TestDuckDuckGoSearchEngine_Name(t *testing.T) {
	engine := NewDuckDuckGoSearchEngine()
	if engine.Name() != "duckduckgo" {
		t.Errorf("expected name 'duckduckgo', got %s", engine.Name())
	}
}

func TestNewMultiEngineSearcher(t *testing.T) {
	searcher := NewMultiEngineSearcher()
	if searcher == nil {
		t.Fatal("expected searcher to be non-nil")
	}

	ms, ok := searcher.(*HybridMultiEngineSearcher)
	if !ok {
		t.Fatal("expected HybridMultiEngineSearcher type")
	}

	if len(ms.engines) != 3 {
		t.Errorf("expected 3 engines, got %d", len(ms.engines))
	}

	if ms.engines["bing"] == nil {
		t.Error("expected bing engine to be present")
	}

	if ms.engines["brave"] == nil {
		t.Error("expected brave engine to be present")
	}

	if ms.engines["duckduckgo"] == nil {
		t.Error("expected duckduckgo engine to be present")
	}

	if ms.extractor == nil {
		t.Error("expected extractor to be non-nil")
	}
}
