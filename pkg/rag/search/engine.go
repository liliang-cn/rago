package search

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// DefaultEngine provides the default implementation of the search engine.
type DefaultEngine struct {
	vectorBackend  storage.VectorBackend
	keywordBackend storage.KeywordBackend
	embedder       storage.Embedder
	fuser          ResultFuser
	expander       QueryExpander
	config         Config
	stats          *SearchStats
	cache          map[string]*CacheEntry
	mu             sync.RWMutex
}

// CacheEntry represents a cached search result.
type CacheEntry struct {
	Results   interface{} `json:"results"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// NewEngine creates a new search engine with the given backends and configuration.
func NewEngine(
	vectorBackend storage.VectorBackend,
	keywordBackend storage.KeywordBackend,
	embedder storage.Embedder,
	config Config,
) (*DefaultEngine, error) {
	if vectorBackend == nil {
		return nil, core.NewValidationError("vector_backend", nil, "vector backend cannot be nil")
	}
	if keywordBackend == nil {
		return nil, core.NewValidationError("keyword_backend", nil, "keyword backend cannot be nil")
	}
	if embedder == nil {
		return nil, core.NewValidationError("embedder", nil, "embedder cannot be nil")
	}

	engine := &DefaultEngine{
		vectorBackend:  vectorBackend,
		keywordBackend: keywordBackend,
		embedder:       embedder,
		config:         config,
		stats: &SearchStats{
			FusionMethods: make(map[string]int64),
			Performance:   make(map[string]interface{}),
			LastUpdated:   time.Now(),
		},
		cache: make(map[string]*CacheEntry),
	}

	// Initialize result fuser
	engine.fuser = NewRRFFuser(config.Hybrid.RRFConstant)

	// Initialize query expander if enabled
	if config.Expansion.Enabled {
		// TODO: Initialize query expander based on method
		// For now, we'll use a no-op expander
		engine.expander = &NoOpExpander{}
	}

	log.Printf("[SEARCH] Engine initialized with vector and keyword backends")
	return engine, nil
}

// VectorSearch performs vector similarity search.
func (e *DefaultEngine) VectorSearch(ctx context.Context, req VectorSearchRequest) (*VectorSearchResponse, error) {
	start := time.Now()
	
	e.mu.Lock()
	e.stats.TotalQueries++
	e.stats.VectorQueries++
	e.mu.Unlock()

	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = e.config.Vector.DefaultLimit
	}
	if req.Limit > e.config.Vector.MaxLimit {
		req.Limit = e.config.Vector.MaxLimit
	}
	if req.Threshold == 0 {
		req.Threshold = e.config.Vector.DefaultThreshold
	}

	// Prepare storage options
	storageOptions := storage.VectorSearchOptions{
		Limit:         req.Limit,
		Offset:        req.Offset,
		Threshold:     req.Threshold,
		Filter:        req.Filter,
		IncludeVector: req.Options.IncludeContent,
	}

	// Perform search
	result, err := e.vectorBackend.SearchVectors(ctx, req.QueryVector, storageOptions)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "search", "VectorSearch", "vector search failed")
	}

	// Convert results
	searchResults := ConvertFromStorageVector(result.Chunks)

	// Apply post-processing
	if req.Options.IncludeHighlights {
		// Vector search doesn't typically have highlights, but we could add them
	}

	response := &VectorSearchResponse{
		Results:     searchResults,
		Total:       result.Total,
		MaxScore:    result.MaxScore,
		Duration:    time.Since(start),
		QueryVector: req.QueryVector,
	}

	// Update statistics
	e.updateSearchStats(len(searchResults), time.Since(start), nil)

	log.Printf("[SEARCH] Vector search completed: %d results in %v", len(searchResults), response.Duration)
	return response, nil
}

// KeywordSearch performs full-text search.
func (e *DefaultEngine) KeywordSearch(ctx context.Context, req KeywordSearchRequest) (*KeywordSearchResponse, error) {
	start := time.Now()
	
	e.mu.Lock()
	e.stats.TotalQueries++
	e.stats.KeywordQueries++
	e.mu.Unlock()

	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = e.config.Keyword.DefaultLimit
	}
	if req.Limit > e.config.Keyword.MaxLimit {
		req.Limit = e.config.Keyword.MaxLimit
	}

	// Prepare storage options
	storageOptions := storage.KeywordSearchOptions{
		Limit:     req.Limit,
		Offset:    req.Offset,
		Filter:    req.Filter,
		Fuzzy:     e.config.Keyword.EnableFuzzy,
		Highlight: req.Options.IncludeHighlights && e.config.Keyword.EnableHighlights,
	}

	// Perform search
	result, err := e.keywordBackend.SearchKeywords(ctx, req.Query, storageOptions)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "search", "KeywordSearch", "keyword search failed")
	}

	// Convert results
	searchResults := ConvertFromStorageKeyword(result.Chunks)

	response := &KeywordSearchResponse{
		Results:  searchResults,
		Total:    result.Total,
		MaxScore: result.MaxScore,
		Duration: time.Since(start),
		Query:    req.Query,
	}

	// Update statistics
	e.updateSearchStats(len(searchResults), time.Since(start), nil)

	log.Printf("[SEARCH] Keyword search completed: %d results in %v", len(searchResults), response.Duration)
	return response, nil
}

// HybridSearch performs hybrid search combining vector and keyword search.
func (e *DefaultEngine) HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResponse, error) {
	start := time.Now()
	
	e.mu.Lock()
	e.stats.TotalQueries++
	e.stats.HybridQueries++
	e.stats.FusionMethods[req.FusionOptions.Method]++
	e.mu.Unlock()

	// Generate query vector if not provided
	queryVector := req.QueryVector
	if len(queryVector) == 0 && req.Query != "" {
		var err error
		queryVector, err = e.embedder.Embed(ctx, req.Query)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "search", "HybridSearch", "failed to generate query vector")
		}
	}

	// Apply defaults
	if req.Limit <= 0 {
		req.Limit = e.config.Vector.DefaultLimit
	}

	// Set fusion defaults
	if req.FusionOptions.Method == "" {
		req.FusionOptions.Method = e.config.Hybrid.DefaultMethod
	}
	if req.FusionOptions.VectorWeight == 0 {
		req.FusionOptions.VectorWeight = e.config.Hybrid.DefaultVectorWeight
	}
	if req.FusionOptions.KeywordWeight == 0 {
		req.FusionOptions.KeywordWeight = e.config.Hybrid.DefaultKeywordWeight
	}
	if req.FusionOptions.RRFConstant == 0 {
		req.FusionOptions.RRFConstant = e.config.Hybrid.RRFConstant
	}

	// Perform searches concurrently
	var vectorResults []SearchResult
	var keywordResults []SearchResult
	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	// Vector search
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorReq := VectorSearchRequest{
			QueryVector: queryVector,
			Limit:       req.Limit,
			Offset:      req.Offset,
			Filter:      req.Filter,
			Options:     req.Options,
		}
		
		vectorResp, err := e.VectorSearch(ctx, vectorReq)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("vector search failed: %w", err))
		} else {
			vectorResults = vectorResp.Results
		}
		mu.Unlock()
	}()

	// Keyword search
	wg.Add(1)
	go func() {
		defer wg.Done()
		keywordReq := KeywordSearchRequest{
			Query:   req.Query,
			Limit:   req.Limit,
			Offset:  req.Offset,
			Filter:  req.Filter,
			Options: req.Options,
		}
		
		keywordResp, err := e.KeywordSearch(ctx, keywordReq)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("keyword search failed: %w", err))
		} else {
			keywordResults = keywordResp.Results
		}
		mu.Unlock()
	}()

	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("hybrid search partially failed: %v", errors)
	}

	// Fuse results
	fusedResults, err := e.fuser.FuseResults(vectorResults, keywordResults, req.FusionOptions)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "search", "HybridSearch", "result fusion failed")
	}

	// Calculate max score
	var maxScore float64
	for _, result := range fusedResults {
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}

	response := &HybridSearchResponse{
		FusedResults:   fusedResults,
		VectorResults:  vectorResults,
		KeywordResults: keywordResults,
		Total:          len(fusedResults),
		MaxScore:       maxScore,
		Duration:       time.Since(start),
		FusionMethod:   req.FusionOptions.Method,
		Query:          req.Query,
	}

	// Update statistics
	e.updateSearchStats(len(fusedResults), time.Since(start), nil)

	log.Printf("[SEARCH] Hybrid search completed: %d results in %v (method: %s)", 
		len(fusedResults), response.Duration, req.FusionOptions.Method)
	return response, nil
}

// ExpandedSearch performs search with query expansion.
func (e *DefaultEngine) ExpandedSearch(ctx context.Context, req ExpandedSearchRequest) (*SearchResponse, error) {
	start := time.Now()
	
	e.mu.Lock()
	e.stats.TotalQueries++
	e.stats.ExpandedQueries++
	e.mu.Unlock()

	if e.expander == nil {
		return nil, core.NewServiceError("search", "ExpandedSearch", "query expansion not enabled", nil)
	}

	// Expand the query
	expandedQuery, err := e.expander.ExpandQuery(ctx, req.Query, req.ExpansionOptions)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "search", "ExpandedSearch", "query expansion failed")
	}

	// Perform hybrid search with expanded query
	hybridReq := HybridSearchRequest{
		Query:   expandedQuery.FinalQuery,
		Limit:   req.Limit,
		Offset:  req.Offset,
		Filter:  req.Filter,
		Options: req.SearchOptions,
		FusionOptions: FusionOptions{
			Method: e.config.Hybrid.DefaultMethod,
			VectorWeight:  e.config.Hybrid.DefaultVectorWeight,
			KeywordWeight: e.config.Hybrid.DefaultKeywordWeight,
			RRFConstant:   e.config.Hybrid.RRFConstant,
		},
	}

	hybridResp, err := e.HybridSearch(ctx, hybridReq)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "search", "ExpandedSearch", "hybrid search failed")
	}

	// Generate suggestions based on expansion
	var suggestions []string
	for _, term := range expandedQuery.ExpandedTerms {
		suggestions = append(suggestions, term.Term)
	}

	response := &SearchResponse{
		Results:     hybridResp.FusedResults,
		Total:       hybridResp.Total,
		MaxScore:    hybridResp.MaxScore,
		Duration:    time.Since(start),
		Query:       expandedQuery.FinalQuery,
		Expanded:    true,
		Suggestions: suggestions,
	}

	log.Printf("[SEARCH] Expanded search completed: %d results in %v", len(response.Results), response.Duration)
	return response, nil
}

// GetSearchStats returns search performance statistics.
func (e *DefaultEngine) GetSearchStats(ctx context.Context) (*SearchStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy of stats to avoid race conditions
	stats := *e.stats
	stats.LastUpdated = time.Now()

	return &stats, nil
}

// Close closes the search engine and cleans up resources.
func (e *DefaultEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear cache
	e.cache = make(map[string]*CacheEntry)

	log.Printf("[SEARCH] Engine closed successfully")
	return nil
}

// updateSearchStats updates internal search statistics.
func (e *DefaultEngine) updateSearchStats(resultCount int, duration time.Duration, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update latency metrics (simplified - in production you'd use proper percentile calculation)
	if e.stats.AverageLatency == 0 {
		e.stats.AverageLatency = duration
	} else {
		// Simple moving average approximation
		e.stats.AverageLatency = (e.stats.AverageLatency + duration) / 2
	}

	// Update result metrics
	totalQueries := float64(e.stats.TotalQueries)
	currentAvg := e.stats.AverageResults
	e.stats.AverageResults = (currentAvg*(totalQueries-1) + float64(resultCount)) / totalQueries

	// Update zero result rate
	if resultCount == 0 {
		zeroResults := e.stats.ZeroResultRate * (totalQueries - 1) + 1
		e.stats.ZeroResultRate = zeroResults / totalQueries
	} else {
		e.stats.ZeroResultRate = (e.stats.ZeroResultRate * (totalQueries - 1)) / totalQueries
	}

	// Update error rate
	if err != nil {
		errors := e.stats.ErrorRate * (totalQueries - 1) + 1
		e.stats.ErrorRate = errors / totalQueries
	} else {
		e.stats.ErrorRate = (e.stats.ErrorRate * (totalQueries - 1)) / totalQueries
	}

	e.stats.LastUpdated = time.Now()
}

// ===== RESULT FUSION IMPLEMENTATIONS =====

// RRFFuser implements Reciprocal Rank Fusion for combining search results.
type RRFFuser struct {
	k float64 // RRF constant
}

// NewRRFFuser creates a new RRF fuser with the specified constant.
func NewRRFFuser(k float64) *RRFFuser {
	if k <= 0 {
		k = 60.0 // Default RRF constant
	}
	return &RRFFuser{k: k}
}

// FuseResults combines results using Reciprocal Rank Fusion.
func (f *RRFFuser) FuseResults(vectorResults, keywordResults []SearchResult, options FusionOptions) ([]SearchResult, error) {
	if options.Method == "" {
		options.Method = "rrf"
	}

	switch options.Method {
	case "rrf":
		return f.fuseWithRRF(vectorResults, keywordResults, options)
	case "weighted":
		return f.fuseWithWeights(vectorResults, keywordResults, options)
	case "linear":
		return f.fuseWithLinear(vectorResults, keywordResults, options)
	default:
		return nil, fmt.Errorf("unsupported fusion method: %s", options.Method)
	}
}

// fuseWithRRF implements Reciprocal Rank Fusion.
func (f *RRFFuser) fuseWithRRF(vectorResults, keywordResults []SearchResult, options FusionOptions) ([]SearchResult, error) {
	k := options.RRFConstant
	if k == 0 {
		k = f.k
	}

	scores := make(map[string]float64)
	resultsMap := make(map[string]SearchResult)

	// Process vector results
	for i, result := range vectorResults {
		rank := float64(i + 1)
		scores[result.ChunkID] += 1.0 / (k + rank)
		if _, exists := resultsMap[result.ChunkID]; !exists {
			result.Source = "vector"
			resultsMap[result.ChunkID] = result
		}
	}

	// Process keyword results
	for i, result := range keywordResults {
		rank := float64(i + 1)
		scores[result.ChunkID] += 1.0 / (k + rank)
		if existing, exists := resultsMap[result.ChunkID]; exists {
			// Combine sources if chunk appears in both
			existing.Source = "hybrid"
			resultsMap[result.ChunkID] = existing
		} else {
			result.Source = "keyword"
			resultsMap[result.ChunkID] = result
		}
	}

	// Create final results with RRF scores
	var fusedResults []SearchResult
	for chunkID, result := range resultsMap {
		result.Score = scores[chunkID]
		fusedResults = append(fusedResults, result)
	}

	// Sort by fused score
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	return fusedResults, nil
}

// fuseWithWeights combines results using weighted scores.
func (f *RRFFuser) fuseWithWeights(vectorResults, keywordResults []SearchResult, options FusionOptions) ([]SearchResult, error) {
	vWeight := options.VectorWeight
	kWeight := options.KeywordWeight
	
	if vWeight == 0 && kWeight == 0 {
		vWeight = 0.5
		kWeight = 0.5
	}

	scores := make(map[string]float64)
	resultsMap := make(map[string]SearchResult)

	// Process vector results
	for _, result := range vectorResults {
		scores[result.ChunkID] += result.Score * vWeight
		if _, exists := resultsMap[result.ChunkID]; !exists {
			result.Source = "vector"
			resultsMap[result.ChunkID] = result
		}
	}

	// Process keyword results
	for _, result := range keywordResults {
		scores[result.ChunkID] += result.Score * kWeight
		if existing, exists := resultsMap[result.ChunkID]; exists {
			existing.Source = "hybrid"
			resultsMap[result.ChunkID] = existing
		} else {
			result.Source = "keyword"
			resultsMap[result.ChunkID] = result
		}
	}

	// Create final results with weighted scores
	var fusedResults []SearchResult
	for chunkID, result := range resultsMap {
		result.Score = scores[chunkID]
		fusedResults = append(fusedResults, result)
	}

	// Sort by weighted score
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	return fusedResults, nil
}

// fuseWithLinear combines results using linear combination.
func (f *RRFFuser) fuseWithLinear(vectorResults, keywordResults []SearchResult, options FusionOptions) ([]SearchResult, error) {
	// Linear fusion is similar to weighted but normalizes scores first
	return f.fuseWithWeights(vectorResults, keywordResults, options)
}

// ===== QUERY EXPANSION IMPLEMENTATIONS =====

// NoOpExpander is a placeholder expander that doesn't modify queries.
type NoOpExpander struct{}

// ExpandQuery returns the original query unchanged.
func (e *NoOpExpander) ExpandQuery(ctx context.Context, originalQuery string, options ExpansionOptions) (*ExpandedQuery, error) {
	return &ExpandedQuery{
		OriginalQuery: originalQuery,
		ExpandedTerms: []ExpansionTerm{},
		FinalQuery:    originalQuery,
		Method:        "none",
	}, nil
}