// Package search provides advanced search algorithms for the RAG pillar.
// This package implements vector search, keyword search, and hybrid search
// with configurable fusion strategies and optimization capabilities.
package search

import (
	"context"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// ===== SEARCH INTERFACES =====

// Engine provides unified search capabilities across vector and keyword backends.
type Engine interface {
	// Vector search operations
	VectorSearch(ctx context.Context, req VectorSearchRequest) (*VectorSearchResponse, error)
	
	// Keyword search operations
	KeywordSearch(ctx context.Context, req KeywordSearchRequest) (*KeywordSearchResponse, error)
	
	// Hybrid search combining vector and keyword search
	HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResponse, error)
	
	// Search with query expansion
	ExpandedSearch(ctx context.Context, req ExpandedSearchRequest) (*SearchResponse, error)
	
	// Multi-modal search (future extension)
	// MultiModalSearch(ctx context.Context, req MultiModalSearchRequest) (*MultiModalSearchResponse, error)
	
	// Search performance analytics
	GetSearchStats(ctx context.Context) (*SearchStats, error)
	
	// Close the search engine
	Close() error
}

// VectorSearcher provides vector similarity search capabilities.
type VectorSearcher interface {
	Search(ctx context.Context, queryVector []float64, options storage.VectorSearchOptions) (*storage.VectorSearchResult, error)
}

// KeywordSearcher provides full-text search capabilities.
type KeywordSearcher interface {
	Search(ctx context.Context, query string, options storage.KeywordSearchOptions) (*storage.KeywordSearchResult, error)
}

// ResultFuser combines results from multiple search sources.
type ResultFuser interface {
	FuseResults(vectorResults []SearchResult, keywordResults []SearchResult, options FusionOptions) ([]SearchResult, error)
}

// QueryExpander expands queries for improved recall.
type QueryExpander interface {
	ExpandQuery(ctx context.Context, originalQuery string, options ExpansionOptions) (*ExpandedQuery, error)
}

// ===== REQUEST TYPES =====

// VectorSearchRequest represents a vector similarity search request.
type VectorSearchRequest struct {
	QueryVector []float64              `json:"query_vector"`
	Limit       int                    `json:"limit"`
	Offset      int                    `json:"offset"`
	Threshold   float64                `json:"threshold"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	Options     SearchOptions          `json:"options"`
}

// KeywordSearchRequest represents a full-text search request.
type KeywordSearchRequest struct {
	Query   string                 `json:"query"`
	Limit   int                    `json:"limit"`
	Offset  int                    `json:"offset"`
	Filter  map[string]interface{} `json:"filter,omitempty"`
	Options SearchOptions          `json:"options"`
}

// HybridSearchRequest represents a hybrid search combining vector and keyword search.
type HybridSearchRequest struct {
	Query         string                 `json:"query"`
	QueryVector   []float64              `json:"query_vector,omitempty"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	Filter        map[string]interface{} `json:"filter,omitempty"`
	FusionOptions FusionOptions          `json:"fusion_options"`
	Options       SearchOptions          `json:"options"`
}

// ExpandedSearchRequest represents a search with query expansion.
type ExpandedSearchRequest struct {
	Query             string            `json:"query"`
	Limit             int               `json:"limit"`
	Offset            int               `json:"offset"`
	Filter            map[string]interface{} `json:"filter,omitempty"`
	ExpansionOptions  ExpansionOptions  `json:"expansion_options"`
	SearchOptions     SearchOptions     `json:"search_options"`
}

// ===== RESPONSE TYPES =====

// VectorSearchResponse represents vector search results.
type VectorSearchResponse struct {
	Results     []SearchResult `json:"results"`
	Total       int            `json:"total"`
	MaxScore    float64        `json:"max_score"`
	Duration    time.Duration  `json:"duration"`
	QueryVector []float64      `json:"query_vector,omitempty"`
}

// KeywordSearchResponse represents keyword search results.
type KeywordSearchResponse struct {
	Results     []SearchResult `json:"results"`
	Total       int            `json:"total"`
	MaxScore    float64        `json:"max_score"`
	Duration    time.Duration  `json:"duration"`
	Query       string         `json:"query"`
	Highlights  []string       `json:"highlights,omitempty"`
}

// HybridSearchResponse represents hybrid search results.
type HybridSearchResponse struct {
	FusedResults   []SearchResult `json:"fused_results"`
	VectorResults  []SearchResult `json:"vector_results"`
	KeywordResults []SearchResult `json:"keyword_results"`
	Total          int            `json:"total"`
	MaxScore       float64        `json:"max_score"`
	Duration       time.Duration  `json:"duration"`
	FusionMethod   string         `json:"fusion_method"`
	Query          string         `json:"query"`
}

// SearchResponse represents a generic search response.
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Total      int            `json:"total"`
	MaxScore   float64        `json:"max_score"`
	Duration   time.Duration  `json:"duration"`
	Query      string         `json:"query"`
	Expanded   bool           `json:"expanded"`
	Suggestions []string      `json:"suggestions,omitempty"`
}

// ===== DATA TYPES =====

// SearchResult represents a single search result.
type SearchResult struct {
	ChunkID     string                 `json:"chunk_id"`
	DocumentID  string                 `json:"document_id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	Score       float64                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata"`
	Highlights  []string               `json:"highlights,omitempty"`
	Context     map[string]string      `json:"context,omitempty"`
	Source      string                 `json:"source"` // "vector", "keyword", "hybrid"
	Position    int                    `json:"position"`
}

// SearchOptions defines common search options.
type SearchOptions struct {
	IncludeContent   bool   `json:"include_content"`
	IncludeMetadata  bool   `json:"include_metadata"`
	IncludeHighlights bool  `json:"include_highlights"`
	ContextLength    int    `json:"context_length"`
	Timeout          int    `json:"timeout_seconds"`
}

// FusionOptions defines options for result fusion.
type FusionOptions struct {
	Method        string  `json:"method"`         // "rrf", "weighted", "linear"
	VectorWeight  float64 `json:"vector_weight"`
	KeywordWeight float64 `json:"keyword_weight"`
	RRFConstant   float64 `json:"rrf_constant"`   // k parameter for RRF
	Normalize     bool    `json:"normalize"`
}

// ExpansionOptions defines options for query expansion.
type ExpansionOptions struct {
	Method       string  `json:"method"`        // "synonyms", "embeddings", "llm"
	MaxTerms     int     `json:"max_terms"`
	MinScore     float64 `json:"min_score"`
	BoostOriginal float64 `json:"boost_original"`
}

// ExpandedQuery represents an expanded query with additional terms.
type ExpandedQuery struct {
	OriginalQuery string             `json:"original_query"`
	ExpandedTerms []ExpansionTerm    `json:"expanded_terms"`
	FinalQuery    string             `json:"final_query"`
	Method        string             `json:"method"`
}

// ExpansionTerm represents a single expanded query term.
type ExpansionTerm struct {
	Term   string  `json:"term"`
	Score  float64 `json:"score"`
	Source string  `json:"source"` // "synonym", "embedding", "llm"
}

// ===== STATISTICS =====

// SearchStats provides comprehensive search performance statistics.
type SearchStats struct {
	// Query statistics
	TotalQueries     int64                  `json:"total_queries"`
	VectorQueries    int64                  `json:"vector_queries"`
	KeywordQueries   int64                  `json:"keyword_queries"`
	HybridQueries    int64                  `json:"hybrid_queries"`
	ExpandedQueries  int64                  `json:"expanded_queries"`
	
	// Performance metrics
	AverageLatency   time.Duration          `json:"average_latency"`
	P95Latency       time.Duration          `json:"p95_latency"`
	P99Latency       time.Duration          `json:"p99_latency"`
	
	// Result metrics
	AverageResults   float64                `json:"average_results"`
	ZeroResultRate   float64                `json:"zero_result_rate"`
	
	// Fusion metrics (for hybrid search)
	FusionMethods    map[string]int64       `json:"fusion_methods"`
	
	// Cache metrics
	CacheHitRate     float64                `json:"cache_hit_rate"`
	
	// Error metrics
	ErrorRate        float64                `json:"error_rate"`
	
	// Time-based metrics
	LastUpdated      time.Time              `json:"last_updated"`
	Performance      map[string]interface{} `json:"performance"`
}

// ===== CONFIGURATION =====

// Config defines the search engine configuration.
type Config struct {
	// Vector search configuration
	Vector VectorSearchConfig `toml:"vector"`
	
	// Keyword search configuration
	Keyword KeywordSearchConfig `toml:"keyword"`
	
	// Hybrid search configuration
	Hybrid HybridSearchConfig `toml:"hybrid"`
	
	// Query expansion configuration
	Expansion ExpansionConfig `toml:"expansion"`
	
	// Performance tuning
	CacheSize     int           `toml:"cache_size"`
	CacheTTL      time.Duration `toml:"cache_ttl"`
	MaxConcurrent int           `toml:"max_concurrent"`
	Timeout       time.Duration `toml:"timeout"`
}

// VectorSearchConfig defines vector search specific configuration.
type VectorSearchConfig struct {
	DefaultLimit     int     `toml:"default_limit"`
	MaxLimit         int     `toml:"max_limit"`
	DefaultThreshold float64 `toml:"default_threshold"`
	EnableCache      bool    `toml:"enable_cache"`
}

// KeywordSearchConfig defines keyword search specific configuration.
type KeywordSearchConfig struct {
	DefaultLimit      int                    `toml:"default_limit"`
	MaxLimit          int                    `toml:"max_limit"`
	EnableFuzzy       bool                   `toml:"enable_fuzzy"`
	EnableHighlights  bool                   `toml:"enable_highlights"`
	HighlightLength   int                    `toml:"highlight_length"`
	BoostFields       map[string]float64     `toml:"boost_fields"`
}

// HybridSearchConfig defines hybrid search specific configuration.
type HybridSearchConfig struct {
	DefaultMethod     string  `toml:"default_method"`
	DefaultVectorWeight  float64 `toml:"default_vector_weight"`
	DefaultKeywordWeight float64 `toml:"default_keyword_weight"`
	RRFConstant       float64 `toml:"rrf_constant"`
	EnableNormalization bool  `toml:"enable_normalization"`
}

// ExpansionConfig defines query expansion configuration.
type ExpansionConfig struct {
	Enabled         bool    `toml:"enabled"`
	DefaultMethod   string  `toml:"default_method"`
	MaxTerms        int     `toml:"max_terms"`
	MinScore        float64 `toml:"min_score"`
	BoostOriginal   float64 `toml:"boost_original"`
}

// ===== HELPER FUNCTIONS =====

// ConvertFromStorageVector converts storage vector search results to search results.
func ConvertFromStorageVector(hits []storage.VectorSearchHit) []SearchResult {
	results := make([]SearchResult, len(hits))
	for i, hit := range hits {
		results[i] = SearchResult{
			ChunkID:    hit.ChunkID,
			DocumentID: hit.DocumentID,
			Content:    hit.Content,
			Score:      hit.Score,
			Metadata:   hit.Metadata,
			Source:     "vector",
			Position:   i,
		}
	}
	return results
}

// ConvertFromStorageKeyword converts storage keyword search results to search results.
func ConvertFromStorageKeyword(hits []storage.KeywordSearchHit) []SearchResult {
	results := make([]SearchResult, len(hits))
	for i, hit := range hits {
		results[i] = SearchResult{
			ChunkID:    hit.ChunkID,
			DocumentID: hit.DocumentID,
			Content:    hit.Content,
			Score:      hit.Score,
			Metadata:   hit.Metadata,
			Highlights: hit.Highlights,
			Context:    hit.Context,
			Source:     "keyword",
			Position:   i,
		}
	}
	return results
}

// ConvertToCoreSearchResults converts search results to core search results.
func ConvertToCoreSearchResults(results []SearchResult) []core.SearchResult {
	coreResults := make([]core.SearchResult, len(results))
	for i, result := range results {
		coreResults[i] = core.SearchResult{
			DocumentID: result.DocumentID,
			ChunkID:    result.ChunkID,
			Content:    result.Content,
			Score:      float32(result.Score),
			Metadata:   result.Metadata,
			Highlights: result.Highlights,
		}
	}
	return coreResults
}