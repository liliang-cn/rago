package search

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/mcp/builtins/websearch/extraction"
)

type multiEngineSearcher struct {
	engines   map[string]SearchEngine
	extractor ContentExtractor
}

func NewMultiEngineSearcher() MultiEngineSearcher {
	// Use the hybrid approach by default (goquery + chromedp)
	return NewHybridSearcher()
}

// NewBasicMultiEngineSearcher creates a basic searcher without chromedp
func NewBasicMultiEngineSearcher() MultiEngineSearcher {
	return &multiEngineSearcher{
		engines:   defaultEngines(),
		extractor: extraction.NewChromedpExtractor(),
	}
}

func (m *multiEngineSearcher) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	engine := m.selectEngine(opts.Engines)
	if engine == nil {
		return nil, fmt.Errorf("no search engine available")
	}

	results, err := searchWithFallback(ctx, m.engines, engine, query, opts.MaxResults)
	if err != nil {
		return nil, err
	}

	if opts.ExtractContent && len(results) > 0 {
		m.extractContentConcurrently(ctx, results)
	}

	return results, nil
}

func (m *multiEngineSearcher) DeepSearch(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var allResults []SearchResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	engines := m.getEngines(opts.Engines)
	if len(engines) == 0 {
		return nil, fmt.Errorf("no search engines available")
	}

	resultsPerEngine := opts.MaxResults / len(engines)
	if resultsPerEngine < 1 {
		resultsPerEngine = 1
	}

	for _, engine := range engines {
		wg.Add(1)
		go func(eng SearchEngine) {
			defer wg.Done()

			results, err := eng.Search(ctx, query, resultsPerEngine)
			if err != nil {
				fmt.Printf("Engine %s failed: %v\n", eng.Name(), err)
				return
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(engine)
	}

	wg.Wait()

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results from any search engine")
	}

	if opts.ExtractContent {
		m.extractContentConcurrently(ctx, allResults)
	}

	if len(allResults) > opts.MaxResults {
		allResults = allResults[:opts.MaxResults]
	}

	return allResults, nil
}

func (m *multiEngineSearcher) selectEngine(preferred []string) SearchEngine {
	if len(preferred) > 0 {
		for _, name := range preferred {
			if engine, ok := m.engines[name]; ok {
				return engine
			}
		}
	}

	for _, name := range enginePriority() {
		if engine, ok := m.engines[name]; ok {
			return engine
		}
	}

	return nil
}

// searchWithFallback asks the chosen engine, then every other engine in
// priority order, and stops at the first that returns at least one result.
//
// "No results, no error" counts as a failure. A scraper whose selectors no
// longer match the site's markup answers exactly that way, and a fallback that
// only ran on err != nil took it at its word: the basic tool answered every
// query with nothing while an engine further down the list would have
// answered it fine (measured: bing returned 0, nil; duckduckgo returned 5).
func searchWithFallback(ctx context.Context, engines map[string]SearchEngine, first SearchEngine, query string, maxResults int) ([]SearchResult, error) {
	var errs []error
	try := func(eng SearchEngine) ([]SearchResult, bool) {
		results, err := eng.Search(ctx, query, maxResults)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", eng.Name(), err))
			return nil, false
		}
		if len(results) == 0 {
			errs = append(errs, fmt.Errorf("%s: no results", eng.Name()))
			return nil, false
		}
		return results, true
	}
	if results, ok := try(first); ok {
		return results, nil
	}
	for _, name := range enginePriority() {
		if name == first.Name() {
			continue
		}
		eng, ok := engines[name]
		if !ok {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		if results, ok := try(eng); ok {
			return results, nil
		}
	}
	return nil, fmt.Errorf("all search engines failed: %w", errors.Join(errs...))
}

func (m *multiEngineSearcher) getEngines(names []string) []SearchEngine {
	if len(names) == 0 {
		names = enginePriority()
	}

	var engines []SearchEngine
	for _, name := range names {
		if engine, ok := m.engines[name]; ok {
			engines = append(engines, engine)
		}
	}

	return engines
}

func (m *multiEngineSearcher) extractContentConcurrently(ctx context.Context, results []SearchResult) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			content, err := m.extractor.ExtractContent(ctx, results[idx].URL)
			if err == nil {
				results[idx].Content = content
				results[idx].ExtractedAt = time.Now()
			}
		}(i)
	}

	wg.Wait()
}
